package autoresharing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/config"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/helpers"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/avast/retry-go"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client/http"
)

const TaskType = "auto_resharing"

// taskData represents the serializable part of the task for database storage
type taskData struct {
	EpochId   uint32                `json:"epoch_id"`
	TssInfo   []bridgetypes.TSSInfo `json:"tss_info"`
	StartTime int64                 `json:"start_time"`
	Threshold uint32
}

type Task struct {
	id          int64 // database ID
	EpochId     uint32
	TssInfo     []bridgetypes.TSSInfo
	CoreAddress string
	Threshold   uint32
	StartTime   time.Time

	BinaryPath       string
	ConfigPath       string
	CertificatesPath string

	GRPCCore grpc.ClientConn
	HTTPCore *http.HTTP
}

func NewTask(tssconfig *config.TSSConfig, grpccon grpc.ClientConn, httpcon *http.HTTP) *Task {
	return &Task{
		BinaryPath:       tssconfig.BinaryPath,
		ConfigPath:       tssconfig.ConfigPath,
		CertificatesPath: tssconfig.CertificatesPath,
		CoreAddress:      tssconfig.CoreAddress,
		GRPCCore:         grpccon,
		HTTPCore:         httpcon,
	}
}

func (t Task) GetTime() time.Time {
	return t.StartTime
}

func (t Task) GetName() string {
	return "AutoResharingTask"
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
		BinaryPath:       t.BinaryPath,
		ConfigPath:       t.ConfigPath,
		CertificatesPath: t.CertificatesPath,

		CoreAddress: t.CoreAddress,

		HTTPCore: t.HTTPCore,
		GRPCCore: t.GRPCCore,
	}

	for _, attribute := range attributes {
		switch attribute.Key {
		case bridgetypes.AttributeTssInfo:
			if err := json.Unmarshal([]byte(attribute.Value), &task.TssInfo); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal tss info")
			}
		case bridgetypes.AttributeEpochId:
			epoch, err := strconv.ParseUint(attribute.Value, 10, 32)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse epoch id")
			}
			task.EpochId = uint32(epoch)
		case bridgetypes.AttributeEpochStartTime:
			startTime, err := strconv.ParseInt(attribute.Value, 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse start time")
			}
			task.StartTime = time.Unix(startTime, 0)
		case bridgetypes.AttributeTSSThreshold:
			threshold, err := strconv.ParseUint(attribute.Value, 10, 32)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse threshold")
			}
			task.Threshold = uint32(threshold)
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
		EpochId:   t.EpochId,
		TssInfo:   t.TssInfo,
		StartTime: t.StartTime.Unix(),
		Threshold: t.Threshold,
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
	t.EpochId = td.EpochId
	t.TssInfo = td.TssInfo
	t.StartTime = time.Unix(td.StartTime, 0)
	t.Threshold = td.Threshold
	return nil
}

func (t Task) Execute(ctx context.Context) (bool, error) {
	isRevoked := t.isRevokedParty()
	if t.BinaryPath == "" {
		return !isRevoked, errors.New("binary path is not set")
	}

	fmt.Println("Update resharing params")
	if err := t.updateConfigBeforeResharing(); err != nil {
		return !isRevoked, errors.Wrap(err, "failed to update parties config")
	}
	fmt.Println("Start resharing")

	args := []string{
		"service",
		"run",
		"reshare",
		"all",
		"--config", t.ConfigPath,
	}

	cmd := exec.CommandContext(ctx, t.BinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return !isRevoked, errors.Wrap(err, "failed to execute resharing task")
	}
	fmt.Println("Resharing completed, updating config with new epoch info")

	opt := make([]retry.Option, 0)
	if t.isNewParty() { // new parties take a part only first session
		fmt.Println("Waiting for new party to be active")
		opt = append(opt,
			retry.Delay(1*time.Minute),
			retry.Attempts(120),
		)
	}

	// do not change config if party is revoked, just return
	if isRevoked {
		fmt.Println("Party is revoked, skipping config update")
		return false, nil
	}

	var (
		utxoChains []bridgetypes.Chain
		epoch      *bridgetypes.Epoch
		blockTime  time.Time
		err        error
	)

	err = retry.Do(
		func() error {
			fmt.Println(fmt.Sprintf("Waiting for epoch %d to start", t.EpochId))
			epoch, err = helpers.GetEpochState(ctx, t.EpochId, t.GRPCCore)
			if epoch.Status != bridgetypes.EpochStatus_RUNNING {
				return errors.New(fmt.Sprintf("epoch %d is not running yet", t.EpochId))
			}
			return err
		},
		opt...,
	)

	if err != nil {
		return !isRevoked, errors.Wrap(err, fmt.Sprintf("failed to get epoch %d state", t.EpochId))
	}

	err = retry.Do(
		func() error {
			fmt.Println("Waiting for blocktime")
			height := int64(epoch.FinalizedBlock)
			block, err := t.HTTPCore.Block(ctx, &height)
			if err != nil {
				return err
			}
			blockTime = block.Block.Time
			return nil
		},
		opt...,
	)
	if err != nil {
		return !isRevoked, errors.Wrap(err, "failed to get blocktime ")
	}

	err = retry.Do(
		func() error {
			fmt.Println("Waiting for bitcoin bridge address")
			utxoChains, err = helpers.GetChains(ctx, bridgetypes.ChainType_BITCOIN, t.GRPCCore)
			return err
		},
		opt...,
	)
	if err != nil {
		return !isRevoked, errors.Wrap(err, "failed to get bitcoin bridge address")
	}

	fmt.Println("Waiting for 15 minutes for new epoch to start")
	return true, errors.Wrap(t.updateConfigAfterResharing(epoch, blockTime.Add(15*time.Minute), utxoChains), "failed to update config after start")
}

func (t Task) isNewParty() bool {
	for _, info := range t.TssInfo {
		if info.Address == t.CoreAddress {
			return info.Active
		}
	}

	return false
}

func (t Task) isRevokedParty() bool {
	for _, info := range t.TssInfo {
		if info.Address == t.CoreAddress {
			return !info.Active
		}
	}
	return false
}
