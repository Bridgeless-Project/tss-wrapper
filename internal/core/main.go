package core

import (
	"context"
	"log"
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
		log.Printf("CRITICAL: Failed to start default mode: %v", err)
		return errors.Wrap(err, "failed to start default mode")
	}
	return nil
}

func (o *Orchestrator) Stop() error {
	if o.cmd == nil || o.cmd.Process == nil {
		log.Printf("WARNING: No process to stop")
		return nil
	}

	if err := o.cmd.Process.Kill(); err != nil {
		log.Printf("CRITICAL: Failed to stop process: %v", err)
		return errors.Wrap(err, "failed to stop process")
	}

	return nil
}

func (o *Orchestrator) Run(ctx context.Context) error {
	err := o.StartDefaultMode(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():

			//TODO implement graceful shutdown
			return nil

		case task := <-o.taskChan:
			log.Printf("Received task: %v", task)

			if err = o.Stop(); err != nil {
				log.Printf("CRITICAL: Failed to stop process: %v", err)
				return errors.Wrap(err, "failed to stop process")
			}
			if err = task.Execute(ctx); err != nil {
				log.Printf("CRITICAL: Failed to execute task: %v", err)
				return errors.Wrap(err, "failed to execute task")
			}
			if err = o.StartDefaultMode(ctx); err != nil {
				log.Printf("CRITICAL: Failed to start default mode: %v", err)
				return errors.Wrap(err, "failed to start default mode")
			}
		}

	}
}
