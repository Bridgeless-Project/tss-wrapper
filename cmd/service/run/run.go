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
	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	pg "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data/postgress"
	autoresharing "github.com/Bridgeless-Project/tss-wrapper-svc/internal/tss/autoreshering"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/tss/update"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	pbTypes "github.com/Bridgeless-Project/tss-wrapper-svc/resources/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.com/distributed_lab/logan/v3"
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
	tasksDb := pg.NewTasksQ(cfg.DB())

	orchestratorTaskChan := make(chan types.Task)
	schedulerTaskChan := make(chan types.Task)

	orchestrator := core.NewOrchestrator(
		tssConfig.BinaryPath,
		tssConfig.BinaryParams,
		tssConfig.APIParams,
		orchestratorTaskChan,
		logger,
		tasksDb,
	)
	taskScheduler := scheduler.New(schedulerTaskChan, orchestratorTaskChan, tasksDb, logger)
	eventObserver := observer.New(
		cfg.TendermintHttpClient(),
		schedulerTaskChan,
		logger,
		blocksDb,
		tasksDb,
	)

	for _, eventCfg := range eventsConfig {
		task, err := createTask(eventCfg.TaskType, cfg)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to create task for event %s", eventCfg.Event))
		}
		eventObserver.WithEvent(eventCfg.Event, task)
	}

	// Load and schedule incomplete tasks from the database
	if err := loadIncompleteTasks(
		ctx,
		tasksDb,
		taskScheduler,
		cfg,
		logger,
	); err != nil {
		return errors.Wrap(err, "failed to load incomplete tasks")
	}

	lastBlock, err := blocksDb.GetLatestBlock()
	if err != nil {
		return errors.Wrap(err, "failed to get latest block")
	}

	eg.Go(func() error {
		return errors.Wrap(eventObserver.Run(ctx, lastBlock), "error while running observer")
	})

	eg.Go(func() error {
		return errors.Wrap(orchestrator.Run(ctx), "error while running orchestrator")
	})

	eg.Go(func() error {
		return errors.Wrap(taskScheduler.Run(ctx), "error while running task scheduler")
	})

	return eg.Wait()
}

// loadIncompleteTasks loads tasks from database that are not completed and schedules them
func loadIncompleteTasks(
	ctx context.Context,
	tasksDb db.TasksQ,
	taskScheduler *scheduler.Scheduler,
	cfg config.Config,
	logger *logan.Entry,
) error {
	incompleteTasks, err := tasksDb.GetIncomplete()
	if err != nil {
		return errors.Wrap(err, "failed to get incomplete tasks from database")
	}

	logger.WithField("count", len(incompleteTasks)).Info("loading incomplete tasks from database")

	for _, record := range incompleteTasks {
		task, err := createTaskFromRecord(record, cfg)
		if err != nil {
			logger.WithError(err).
				WithField("task_id", record.ID).
				WithField("task_type", record.TaskType).
				Error("failed to restore task from database, skipping")
			continue
		}

		task.SetID(record.ID)

		// Update status to Planned and schedule
		if err := tasksDb.UpdateStatus(record.ID, pbTypes.ProcessStatus_PROCESS_STATUS_PLANNED); err != nil {
			logger.WithError(err).
				WithField("task_id", record.ID).
				Error("failed to update task status to planned")
		}

		taskScheduler.ScheduleTask(ctx, task)

		logger.WithField("task_id", record.ID).
			WithField("task_type", record.TaskType).
			Info("restored and scheduled task from database")
	}

	return nil
}

// createTask creates a task template based on the task type and TSS config
func createTask(taskType config.TaskType, cfg config.Config) (types.Task, error) {
	switch taskType {
	case config.TaskTypeAutoResharing:
		return autoresharing.NewTask(
			cfg.TSSConfig(),
			cfg.TendermintGrpcClient(),
			cfg.TendermintHttpClient(),
		), nil
	case config.TaskTypeUpdate:
		return update.NewTask(), nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

// createTaskFromRecord creates a task from a database record
func createTaskFromRecord(record db.TaskRecord, cfg config.Config) (types.Task, error) {
	var task types.Task

	switch record.TaskType {
	case autoresharing.TaskType:
		t := autoresharing.NewTask(
			cfg.TSSConfig(),
			cfg.TendermintGrpcClient(),
			cfg.TendermintHttpClient(),
		)
		if err := t.UnmarshalData(record.Data); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal autoresharing task data")
		}
		task = t
	case update.TaskType:
		t := update.NewTask()
		if err := t.UnmarshalData(record.Data); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal update task data")
		}
		task = t
	default:
		return nil, fmt.Errorf("unknown task type: %s", record.TaskType)
	}

	return task, nil
}
