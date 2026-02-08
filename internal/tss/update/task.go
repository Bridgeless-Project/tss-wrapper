package update

import (
	"context"
	"fmt"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
)

type Task struct {
	Link      string // link to the tss binary to download
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

func (t Task) Name() string {
	return "UpdateTask"
}

func (t Task) DownloadTSSBinary(ctx context.Context) error {
	// TODO: Implement logic to download the TSS binary from t.Link
	return nil
}
