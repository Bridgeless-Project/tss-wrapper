package migration

import (
	"context"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
)

const TaskType = "auto_resharing_migration"

type Task struct {
	id int64 // database ID

	ConfigPath string

	StartTime time.Time
}

func (t *Task) GetTime() time.Time {
	return t.StartTime
}

func (t *Task) GetName() string {
	return "AutoResharingMigrationTask"
}

func (t *Task) GetID() int64 {
	return t.id
}

func (t *Task) GetTaskType() string {
	return TaskType
}

func (t *Task) SetID(id int64) {
	t.id = id
}

func (t *Task) Parse(attributes []types.Attribute) (types.Task, error) {
	panic("implement me")
}

func (t *Task) StartScheduling(ctx context.Context, taskChan chan<- types.Task) {
	delay := time.Until(t.StartTime)
	if delay <= 0 {
		taskChan <- t
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		taskChan <- t
	}
}

func (t *Task) MarshalData() (string, error) {
	panic("implement me")
}

func (t *Task) UnmarshalData(data string) error {
	panic("implement me")
}

func (t *Task) Execute(ctx context.Context) (bool, error) {
	panic("implement me")
}
