package migrate_up

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/config"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
)

const TaskType = "migrate_up"

// taskData represents the serializable part of the task for database storage
type taskData struct {
	StartTime int64 `json:"start_time"`
}

type Task struct {
	id int64 // database ID

	BinaryPath string
	ConfigPath string
	StartTime  time.Time
}

func NewTask(tssconfig *config.TSSConfig) *Task {
	return &Task{
		BinaryPath: tssconfig.BinaryPath,
		ConfigPath: tssconfig.ConfigPath,
	}
}

func (t Task) GetTime() time.Time {
	return t.StartTime
}

func (t Task) GetName() string {
	return "Migrate up"
}

func (t Task) GetID() int64 {
	return t.id
}

func (t Task) GetTaskType() string {
	return TaskType
}

func (t *Task) SetID(id int64) {
	t.id = id
}

func (t Task) Parse(attributes []types.Attribute) (types.Task, error) {
	task := &Task{
		BinaryPath: t.BinaryPath,
		ConfigPath: t.ConfigPath,
	}

	for _, attribute := range attributes {
		switch attribute.Key {
		default:
			continue
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

func (t Task) MarshalData() (string, error) {
	data := taskData{
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

	t.StartTime = time.Unix(td.StartTime, 0)
	return nil
}

func (t Task) Execute(ctx context.Context) (bool, error) {
	if t.BinaryPath == "" {
		return true, errors.New("binary path is not set")
	}

	args := []string{
		"service",
		"migrate",
		"up",
		"--config", t.ConfigPath,
	}

	cmd := exec.CommandContext(ctx, t.BinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return true, errors.Wrap(cmd.Run(), "failed to execute migrate_up task")
}
