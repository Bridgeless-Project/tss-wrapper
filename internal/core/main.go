package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	grpcTypes "github.com/Bridgeless-Project/tss-wrapper-svc/resources/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Orchestrator struct {
	binaryPath  string
	defaultArgs []string
	apiParams   []string
	logger      *logan.Entry

	coreCmd  *exec.Cmd
	apiCmd   *exec.Cmd
	taskChan <-chan types.Task
	tasksDb  db.TasksQ

	prestartTasks []types.Task
}

func New(binaryPath, binaryParams, apiParams string, taskChan <-chan types.Task, logger *logan.Entry, tasksDb db.TasksQ) *Orchestrator {
	return &Orchestrator{
		binaryPath:  binaryPath,
		taskChan:    taskChan,
		logger:      logger.WithField("component", "orchestrator"),
		tasksDb:     tasksDb,
		defaultArgs: strings.Split(binaryParams, " "),
		apiParams:   strings.Split(apiParams, " "),
	}
}

func (o *Orchestrator) WithPreStartTask(task types.Task) {
	o.prestartTasks = append(o.prestartTasks, task)
}

func (o *Orchestrator) StartDefaultMode(ctx context.Context) error {
	o.coreCmd = exec.CommandContext(ctx, o.binaryPath, o.defaultArgs...)
	o.coreCmd.Stdout = os.Stdout
	o.coreCmd.Stderr = os.Stderr

	isApiNeeded := len(o.apiParams) > 0 && strings.TrimSpace(o.apiParams[0]) != ""

	if isApiNeeded {
		o.apiCmd = exec.CommandContext(ctx, o.binaryPath, o.apiParams...)
		o.apiCmd.Stdout = os.Stdout
		o.apiCmd.Stderr = os.Stderr

		if err := o.apiCmd.Start(); err != nil {
			return errors.Wrap(err, "failed to start api")
		}
		o.logger.WithField("binary", o.binaryPath).Info("started api mode")
	}

	if err := o.coreCmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start default mode")
	}

	o.logger.WithField("binary", o.binaryPath).Info("started default mode")
	return nil
}

func (o *Orchestrator) Stop() error {
	if o.coreCmd == nil || o.coreCmd.Process == nil {
		o.logger.Warn("no process to stop")
		return nil
	}

	if err := o.coreCmd.Process.Signal(syscall.SIGTERM); err != nil {
		return errors.Wrap(err, "failed to kill process")
	}

	if err := o.coreCmd.Wait(); err != nil {
		return errors.Wrap(err, "default mode killed process exited with error")
	}

	o.coreCmd = nil
	return nil
}

func (o *Orchestrator) launchPrestartTasks(ctx context.Context) error {
	for _, task := range o.prestartTasks {
		err := o.Stop()
		if err != nil {
			o.logger.WithError(err).Error("failed to stop prestart task")
		}

		_, err = task.Execute(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to execute prestart task")
		}
	}

	return nil
}

func (o *Orchestrator) Run(ctx context.Context) error {
	err := o.launchPrestartTasks(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to execute prestart tasks")
	}

	if err = o.StartDefaultMode(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("shutting down orchestrator")
			if err := o.Stop(); err != nil { // ignore this error during shutdown, just log it
				o.logger.WithError(err).Warn("failed to stop process during shutdown")
			}
			return nil

		case task := <-o.taskChan:
			taskID := task.GetID()
			o.logger.
				WithField("task_id", taskID).
				WithField("task_type", fmt.Sprintf("%T", task)).
				Info("executing task")

			// This error appears only if process was stopped before, so we can just log it and continue
			if err := o.Stop(); err != nil {
				o.logger.WithError(err).Warn("failed to stop process before executing task")
			}

			err := o.launchPrestartTasks(ctx)
			if err != nil {
				o.updateTaskFailed(taskID, err)
				if err = o.StartDefaultMode(ctx); err != nil {
					return errors.Wrap(err, "failed to restart default mode after task failure")
				}
				continue
			}

			o.logger.WithField("task_id", taskID).Info("stopped process before executing task")
			startDefaultMode, err := task.Execute(ctx)
			if err != nil {
				o.updateTaskFailed(taskID, err)
				o.logger.WithError(err).
					WithField("task_id", taskID).
					Error("task execution failed")

				// kill api before start the default mode
				if o.apiCmd != nil {
					if err = o.apiCmd.Process.Signal(syscall.SIGTERM); err != nil {
						return errors.Wrap(err, "failed to kill api process")
					}
				}

				// Continue running, start default mode again
				if startDefaultMode {
					if err = o.StartDefaultMode(ctx); err != nil {
						return errors.Wrap(err, "failed to restart default mode after task failure")
					}
					continue
				}

				continue
			}

			if o.apiCmd != nil {
				if err = o.apiCmd.Process.Signal(syscall.SIGTERM); err != nil {
					return errors.Wrap(err, "failed to kill api process")
				}
			}

			if startDefaultMode {
				if err = o.launchPrestartTasks(ctx); err != nil {
					o.updateTaskFailed(taskID, err)
					o.logger.WithError(err).
						WithField("task_id", taskID).
						Error("failed to execute prestart tasks after task")

					o.revertTask(ctx, task, taskID)

					if err = o.StartDefaultMode(ctx); err != nil {
						return errors.Wrap(err, "failed to restart default mode after prestart tasks failure")
					}
					continue
				}
			}

			// Update task status to Completed
			if err = o.tasksDb.UpdateStatus(taskID, grpcTypes.ProcessStatus_PROCESS_STATUS_COMPLETED); err != nil {
				o.logger.WithError(err).
					WithField("task_id", taskID).
					Error("failed to update task status to completed")
			}

			o.logger.WithField("task_id", taskID).Info("task completed successfully")

			if !startDefaultMode {
				continue
			}

			if err = o.StartDefaultMode(ctx); err != nil {
				return errors.Wrap(err, "failed to restart default mode after task")
			}
		}
	}
}

func (o *Orchestrator) revertTask(ctx context.Context, task types.Task, taskID int64) {
	reverter, ok := task.(types.Reverter)
	if !ok {
		return
	}

	if err := reverter.Revert(); err != nil {
		o.logger.WithError(err).
			WithField("task_id", taskID).
			Error("failed to revert task after prestart tasks failure")
		return
	}

	o.logger.WithField("task_id", taskID).Info("reverted task after prestart tasks failure")

	if err := o.launchPrestartTasks(ctx); err != nil {
		o.logger.WithError(err).
			WithField("task_id", taskID).
			Error("failed to execute prestart tasks after revert")
	}
}

// updateTaskFailed updates task status to Failed with error message
func (o *Orchestrator) updateTaskFailed(taskID int64, err error) {
	if updateErr := o.tasksDb.UpdateStatusWithError(taskID, grpcTypes.ProcessStatus_PROCESS_STATUS_FAILED, err.Error()); updateErr != nil {
		o.logger.WithError(updateErr).
			WithField("task_id", taskID).
			Error("failed to update task status to failed")
	}
}
