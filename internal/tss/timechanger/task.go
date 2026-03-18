package timechanger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/config"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/helpers"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
)

const TaskType = "timechanger"

// taskData represents the serializable part of the task for database storage
type taskData struct {
	StartTime int64 `json:"start_time"`
}

type Task struct {
	id         int64 // database ID
	ConfigPath string
	StartTime  time.Time
}

func NewTask(tssconfig *config.TSSConfig) *Task {
	return &Task{ConfigPath: tssconfig.ConfigPath}
}

func (t Task) Execute(ctx context.Context) (bool, error) {
	configer := helpers.NewConfigManager(t.ConfigPath)
	err := configer.Load()
	if err != nil {
		return false, err
	}
	err = configer.UpdateResharingTime(t.StartTime)
	if err != nil {
		return false, err
	}
	err = configer.Save()
	if err != nil {
		return false, err
	}
	return true, nil
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
	taskChan <- &t
	return
}

func (t Task) GetName() string {
	return "UpdateTask"
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
