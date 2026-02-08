package update

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
)

const TaskType = "update"

// taskData represents the serializable part of the task for database storage
type taskData struct {
	Link      string `json:"link"`
	Version   string `json:"version"`
	StartTime int64  `json:"start_time"`
}

type Task struct {
	id        int64 // database ID
	Link      string
	Version   string
	StartTime time.Time
}

func NewTask() *Task {
	return &Task{}
}

func (t Task) Execute(ctx context.Context) error {
	if err := t.DownloadTSSBinary(ctx); err != nil {
		return errors.Wrap(err, "failed to download TSS binary")
	}
	return nil
}

func (t Task) GetTime() time.Time {
	return t.StartTime
}

func (t Task) Parse(attributes []types.Attribute) (types.Task, error) {
	task := new(Task)
	for _, attribute := range attributes {
		switch attribute.Key {
		default:
			return nil, errors.New(fmt.Sprintf("unknown attribute key: %s", attribute.Key))
		}
	}
	return task, nil
}

func (t Task) StartScheduling(ctx context.Context, taskChan chan<- types.Task) {
	delay := time.Until(t.StartTime)
	if delay <= 0 {
		taskChan <- &t
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		taskChan <- &t
	}
}

func (t Task) Name() string {
	return "UpdateTask"
}

func (t Task) DownloadTSSBinary(ctx context.Context) error {
	// TODO: Implement logic to download the TSS binary from t.Link
	return nil
}

func (t Task) GetID() int64 {
	return t.id
}

func (t *Task) SetID(id int64) {
	t.id = id
}

func (t Task) GetTaskType() string {
	return TaskType
}

func (t Task) MarshalData() (string, error) {
	data := taskData{
		Link:      t.Link,
		Version:   t.Version,
		StartTime: t.StartTime.Unix(),
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal task data")
	}
	return string(bytes), nil
}

func (t *Task) UnmarshalData(data string) error {
	var td taskData
	if err := json.Unmarshal([]byte(data), &td); err != nil {
		return errors.Wrap(err, "failed to unmarshal task data")
	}
	t.Link = td.Link
	t.Version = td.Version
	t.StartTime = time.Unix(td.StartTime, 0)
	return nil
}
