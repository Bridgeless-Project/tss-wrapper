package scheduler

import (
	"context"
	"sync"

	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	grpcTypes "github.com/Bridgeless-Project/tss-wrapper-svc/resources/types"
	"gitlab.com/distributed_lab/logan/v3"
)

type Scheduler struct {
	taskChan       <-chan types.Task
	tasksWaitGroup *sync.WaitGroup
	// internal chan that receives task when ready
	readyTasks       chan types.Task
	orchestratorChan chan<- types.Task

	tasksDb db.TasksQ
	logger  *logan.Entry
}

func New(taskChan <-chan types.Task, orchestratorChan chan<- types.Task, tasksDb db.TasksQ, logger *logan.Entry) *Scheduler {
	readyChan := make(chan types.Task)
	taskWaitGroup := new(sync.WaitGroup)
	return &Scheduler{
		taskChan:         taskChan,
		orchestratorChan: orchestratorChan,
		readyTasks:       readyChan,
		tasksWaitGroup:   taskWaitGroup,
		tasksDb:          tasksDb,
		logger:           logger.WithField("component", "scheduler"),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.handleIncomingTasks(ctx)
	}()
	go func() {
		defer wg.Done()
		s.handleScheduledTime(ctx)
	}()
	wg.Wait()
	return nil
}

func (s *Scheduler) handleIncomingTasks(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-s.taskChan:
			// Update status to Planned when scheduling starts
			if err := s.tasksDb.UpdateStatus(task.GetID(), grpcTypes.ProcessStatus_PROCESS_STATUS_PLANNED); err != nil {
				s.logger.WithError(err).
					WithField("task_id", task.GetID()).
					Error("failed to update task status to planned")
			}

			s.ScheduleTask(ctx, task)
		}
	}
}

func (s *Scheduler) handleScheduledTime(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-s.readyTasks:
			// Update status to Ongoing when task is ready for execution
			if err := s.tasksDb.UpdateStatus(task.GetID(), grpcTypes.ProcessStatus_PROCESS_STATUS_ONGOING); err != nil {
				s.logger.WithError(err).
					WithField("task_id", task.GetID()).
					Error("failed to update task status to ongoing")
			}

			s.orchestratorChan <- task
			s.tasksWaitGroup.Done()
		}
	}
}

// ScheduleTask adds a task directly to the scheduler (for loading from DB)
func (s *Scheduler) ScheduleTask(ctx context.Context, task types.Task) {
	s.tasksWaitGroup.Add(1)
	go task.StartScheduling(ctx, s.readyTasks)
}
