package scheduler

import (
	"context"
	"sync"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
)

type Scheduler struct {
	taskChan       <-chan types.Task
	tasks          sync.Map
	tasksWaitGroup *sync.WaitGroup
	// internal chan that receives task when
	readyTasks       chan types.Task
	orchestratorChan chan<- types.Task
}

func New(taskChan <-chan types.Task, orchestratorChan chan<- types.Task) *Scheduler {
	readyChan := make(chan types.Task)
	taskWaitGroup := new(sync.WaitGroup)
	return &Scheduler{
		taskChan:         taskChan,
		orchestratorChan: orchestratorChan,
		readyTasks:       readyChan,
		tasksWaitGroup:   taskWaitGroup,
	}
}

func (u *Scheduler) Run(ctx context.Context) error {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		defer wg.Done()
		u.handleIncomingTasks(ctx)
	}()
	go func() {
		defer wg.Done()
		u.handleScheduledTime(ctx)
	}()
	wg.Wait()
	return nil
}

func (u *Scheduler) handleIncomingTasks(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():

			return
		case task := <-u.taskChan:
			u.tasksWaitGroup.Add(1)
			go task.StartScheduling(ctx, u.readyTasks)
		}
	}
}

func (u *Scheduler) handleScheduledTime(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			//TODO: gracefully shutdown

			return
		case task := <-u.readyTasks:
			u.orchestratorChan <- task
			u.tasksWaitGroup.Done()
		}
	}
}
