package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	pbTypes "github.com/Bridgeless-Project/tss-wrapper-svc/resources/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Orchestrator struct {
	binaryPath  string
	defaultArgs []string
	logger      *logan.Entry

	cmd      *exec.Cmd
	taskChan <-chan types.Task
	tasksDb  db.TasksQ
}

func NewOrchestrator(binaryPath string, taskChan <-chan types.Task, logger *logan.Entry, tasksDb db.TasksQ) *Orchestrator {
	return &Orchestrator{
		binaryPath: binaryPath,
		taskChan:   taskChan,
		logger:     logger.WithField("component", "orchestrator"),
		tasksDb:    tasksDb,
	}
}

func (o *Orchestrator) StartDefaultMode(ctx context.Context) error {
	o.cmd = exec.CommandContext(ctx, o.binaryPath, "-f", "/dev/null")
	o.cmd.Stdout = os.Stdout
	o.cmd.Stderr = os.Stderr

	if err := o.cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start default mode")
	}

	o.logger.WithField("binary", o.binaryPath).Info("started default mode")
	return nil
}

func (o *Orchestrator) Stop() error {
	if o.cmd == nil || o.cmd.Process == nil {
		o.logger.Warn("no process to stop")
		return nil
	}

	if err := o.cmd.Process.Kill(); err != nil {
		return errors.Wrap(err, "failed to kill process")
	}

	o.cmd = nil
	return nil
}

func (o *Orchestrator) Run(ctx context.Context) error {
	if err := o.StartDefaultMode(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("shutting down orchestrator")
			if err := o.Stop(); err != nil {
				o.logger.WithError(err).Warn("failed to stop process during shutdown")
			}
			return nil

		case task := <-o.taskChan:
			taskID := task.GetID()
			o.logger.
				WithField("task_id", taskID).
				WithField("task_type", fmt.Sprintf("%T", task)).
				Info("executing task")

			if err := o.Stop(); err != nil {
				o.updateTaskFailed(taskID, err)
				return errors.Wrap(err, "failed to stop process")
			}

			if err := task.Execute(ctx); err != nil {
				o.updateTaskFailed(taskID, err)
				o.logger.WithError(err).
					WithField("task_id", taskID).
					Error("task execution failed")
				// Continue running, start default mode again
				if startErr := o.StartDefaultMode(ctx); startErr != nil {
					return errors.Wrap(startErr, "failed to restart default mode after task failure")
				}
				continue
			}

			// Update task status to Completed
			if err := o.tasksDb.UpdateStatus(taskID, pbTypes.ProcessStatus_PROCESS_STATUS_COMPLETED); err != nil {
				o.logger.WithError(err).
					WithField("task_id", taskID).
					Error("failed to update task status to completed")
			}

			o.logger.WithField("task_id", taskID).Info("task completed successfully")

			if err := o.StartDefaultMode(ctx); err != nil {
				return errors.Wrap(err, "failed to restart default mode after task")
			}
		}
	}
}

// updateTaskFailed updates task status to Failed with error message
func (o *Orchestrator) updateTaskFailed(taskID int64, err error) {
	if updateErr := o.tasksDb.UpdateStatusWithError(taskID, pbTypes.ProcessStatus_PROCESS_STATUS_FAILED, err.Error()); updateErr != nil {
		o.logger.WithError(updateErr).
			WithField("task_id", taskID).
			Error("failed to update task status to failed")
	}
}
