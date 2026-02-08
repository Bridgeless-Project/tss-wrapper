package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Orchestrator struct {
	binaryPath  string
	defaultArgs []string
	logger      *logan.Entry

	cmd      *exec.Cmd
	taskChan <-chan types.Task
}

func NewOrchestrator(binaryPath string, taskChan <-chan types.Task, logger *logan.Entry) *Orchestrator {
	return &Orchestrator{
		binaryPath: binaryPath,
		taskChan:   taskChan,
		logger:     logger.WithField("component", "orchestrator"),
	}
}

func (o *Orchestrator) StartDefaultMode(ctx context.Context) error {
	o.cmd = exec.CommandContext(ctx, o.binaryPath, o.defaultArgs...)
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

	// Wait for process to fully exit to avoid zombie processes
	// and ensure clean state before starting new process
	if err := o.cmd.Wait(); err != nil {
		o.logger.WithError(err).Debug("process wait completed")
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
			o.logger.WithField("task", fmt.Sprintf("%T", task)).Info("received task")

			if err := o.Stop(); err != nil {
				return errors.Wrap(err, "failed to stop process")
			}
			if err := task.Execute(ctx); err != nil {
				return errors.Wrap(err, "failed to execute task")
			}
			if err := o.StartDefaultMode(ctx); err != nil {
				return errors.Wrap(err, "failed to restart default mode after task")
			}
		}
	}
}
