package migration

import (
	"context"
	"encoding/json"
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
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

const TaskType = "auto_resharing_migration"

type taskData struct {
	EpochId uint32 `json:"epoch_id"`
}
type Task struct {
	id          int64 // database ID
	EpochId     uint32
	CoreAddress string

	BinaryPath string
	ConfigPath string
	GRPCCore   grpc.ClientConn
	HTTPCore   *http.HTTP

	btcChainsData map[string]helpers.BitcoinChainData
	StartTime     time.Time
}

func NewTask(tssconfig *config.TSSConfig, grpccon grpc.ClientConn, httpcon *http.HTTP) *Task {
	return &Task{
		BinaryPath:    tssconfig.BinaryPath,
		ConfigPath:    tssconfig.ConfigPath,
		CoreAddress:   tssconfig.CoreAddress,
		GRPCCore:      grpccon,
		HTTPCore:      httpcon,
		btcChainsData: make(map[string]helpers.BitcoinChainData),
	}
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
	task := &Task{
		BinaryPath: t.BinaryPath,
		ConfigPath: t.ConfigPath,

		CoreAddress: t.CoreAddress,

		HTTPCore:  t.HTTPCore,
		GRPCCore:  t.GRPCCore,
		StartTime: time.Now(),
	}
	for _, attr := range attributes {
		switch attr.Key {
		case bridgetypes.AttributeEpochId:
			epoch, err := strconv.ParseUint(attr.Value, 10, 32)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse epoch id")
			}
			task.EpochId = uint32(epoch)
		default:
			continue
		}
	}

	return task, nil
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
	data := taskData{
		EpochId: t.EpochId,
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
	t.StartTime = time.Now()

	return nil
}

func (t *Task) Execute(ctx context.Context) (bool, error) {
	retryOpts := []retry.Option{
		retry.Attempts(3),
		retry.Delay(5 * time.Second),
	}

	var (
		oldEpoch            *bridgetypes.Epoch
		migrationEventBlock *coretypes.ResultBlock
		err                 error
	)

	if err = retry.Do(
		func() error {
			oldEpoch, err = helpers.GetEpochState(ctx, t.EpochId, t.GRPCCore)
			return errors.Wrap(err, "failed to get old epoch info")
		},
		retryOpts...,
	); err != nil {
		return false, err
	}

	if err = retry.Do(
		func() error {
			height := int64(oldEpoch.FinalizedBlock)
			migrationEventBlock, err = t.HTTPCore.Block(ctx, &height)
			return errors.Wrap(err, "failed to get finalized block info")
		},
		retryOpts...,
	); err != nil {
		return false, err
	}

	migrationSessionStartTime := migrationEventBlock.Block.Time.Add(5 * time.Minute)
	nextSessionStartTime := migrationSessionStartTime.Add(time.Hour)

	// check if party is a member of the old epoch, if not, we can skip the migration
	oldEpochParticipant := false
	for _, party := range oldEpoch.Parties {
		if party.Address == t.CoreAddress {
			oldEpochParticipant = true
			break
		}
	}
	if !oldEpochParticipant {
		// awaiting other parties to join the next signing sessions
		configer := helpers.NewConfigManager(t.ConfigPath)
		if err := configer.Load(); err != nil {
			return false, errors.Wrap(err, "failed to load config")
		}
		configer.SetStartTime(nextSessionStartTime)
		err = configer.Save()
		return err == nil, errors.Wrap(err, "failed to save config with new start time")
	}

	var utxoChains []bridgetypes.Chain
	if err = retry.Do(
		func() error {
			utxoChains, err = helpers.GetChains(ctx, bridgetypes.ChainType_BITCOIN, t.GRPCCore)
			return errors.Wrap(err, "failed to get utxo chains")
		},
		retryOpts...,
	); err != nil {
		return false, err
	}

	if err = t.updateConfigBeforeExecution(migrationSessionStartTime, utxoChains); err != nil {
		return false, errors.Wrap(err, "failed to update config before execution")
	}

	args := []string{
		"service",
		"run",
		"reshare",
		"migration",
		"--config", t.ConfigPath,
	}
	cmd := exec.CommandContext(ctx, t.BinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return false, errors.Wrap(err, "failed to execute resharing task")
	}

	var (
		params           *bridgetypes.Params
		currentEpochInfo *bridgetypes.Epoch
	)

	if err = retry.Do(
		func() error {
			params, err = helpers.GetParams(ctx, t.GRPCCore)
			return errors.Wrap(err, "failed to get bridge params")
		},
		retryOpts...,
	); err != nil {
		return false, err
	}

	if err = retry.Do(
		func() error {
			currentEpochInfo, err = helpers.GetEpochState(ctx, params.Epoch, t.GRPCCore)
			return errors.Wrap(err, "failed to get current epoch info")
		},
		retryOpts...,
	); err != nil {
		return false, err
	}

	// check party is a member of the new epoch, if not, we can skip the config update after execution
	newEpochParticipant := false
	for _, party := range currentEpochInfo.Parties {
		if party.Address == t.CoreAddress {
			newEpochParticipant = true
			break
		}
	}
	if !newEpochParticipant {
		// do not run tss service anymore
		return false, nil
	}

	return true, errors.Wrap(t.updateConfigAfterExecution(nextSessionStartTime, utxoChains), "failed to update config after execution")
}
