package autoreshering

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

type Task struct {
	EpochId   uint32
	TssInfo   []bridgeTypes.TSSInfo
	StartTime time.Time
}

func NewTask() *Task {
	return &Task{}
}

func (a Task) Execute(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "", "")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("CRITICAL: Failed to start default mode: %v", err)
		return errors.Wrap(err, "failed to start default mode")
	}

	return nil
}

func (a Task) GetTime() time.Time {
	return a.StartTime
}

func (a Task) Parse(attributes []types.Attribute) (types.Task, error) {

	task := new(Task)
	for _, attribute := range attributes {
		switch attribute.Key {
		case bridgeTypes.AttributeTssInfo:
			if err := json.Unmarshal([]byte(attribute.Value), &task.TssInfo); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal tss info")
			}
		case bridgeTypes.AttributeEpochId:
			epoch, err := strconv.ParseUint(attribute.Value, 10, 32)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse deposit nonce")
			}
			task.EpochId = uint32(epoch)
		default:
			return nil, errors.Wrap(errors.New(fmt.Sprintf("unknown attribute key: %s", attribute.Key)), "failed to parse attribute")
		}
	}
	return task, nil
}

func (a Task) StartScheduling(ctx context.Context, taskChan chan<- types.Task) {
	delay := time.Until(a.StartTime)
	if delay <= 0 {
		taskChan <- a
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		taskChan <- a
	}
}

func (a Task) Name() string {
	return "AutoResheringTask"
}
