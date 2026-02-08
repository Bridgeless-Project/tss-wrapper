package run

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/Bridgeless-Project/tss-wrapper-svc/cmd/utils"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/config"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/core"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/core/observer"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/core/scheduler"
	pg "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data/postgress"
	autoresharing "github.com/Bridgeless-Project/tss-wrapper-svc/internal/tss/autoreshering"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/tss/update"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	utils.RegisterConfigFlag(Cmd)
}

var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Starts the service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := utils.ConfigFromFlags(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to get config from flags")
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		err = runService(ctx, cfg)

		return errors.Wrap(err, "failed to run tss wrapper service")
	},
}

func runService(ctx context.Context, cfg config.Config) error {
	eg, ctx := errgroup.WithContext(ctx)
	logger := cfg.Log()
	tssConfig := cfg.TSSConfig()
	eventsConfig := cfg.EventsConfig()

	blocksDb := pg.NewBlocksQ(cfg.DB())

	task, err := createTask(eventsConfig.TaskType, tssConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create task")
	}

	orchestratorTaskChan := make(chan types.Task)
	schedulerTaskChan := make(chan types.Task)

	orchestrator := core.NewOrchestrator(tssConfig.BinaryPath, orchestratorTaskChan, logger)
	taskScheduler := scheduler.New(schedulerTaskChan, orchestratorTaskChan)
	eventObserver := observer.
		New(cfg.TendermintHttpClient(), schedulerTaskChan, logger, blocksDb).
		WithEvent(eventsConfig.Event, task)

	eg.Go(func() error {
		return errors.Wrap(eventObserver.Run(ctx, 0), "error while running observer")
	})

	eg.Go(func() error {
		return errors.Wrap(orchestrator.Run(ctx), "error while running orchestrator")
	})

	eg.Go(func() error {
		return errors.Wrap(taskScheduler.Run(ctx), "error while running task scheduler")
	})

	return eg.Wait()
}

// createTask creates a task template based on the task type and TSS config
func createTask(taskType config.TaskType, tssConfig *config.TSSConfig) (types.Task, error) {
	switch taskType {
	case config.TaskTypeAutoResharing:
		return autoresharing.NewTask(
			tssConfig.BinaryPath,
			tssConfig.ConfigPath,
			tssConfig.CertificatesPath,
		), nil
	case config.TaskTypeUpdate:
		return update.NewTask(), nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
