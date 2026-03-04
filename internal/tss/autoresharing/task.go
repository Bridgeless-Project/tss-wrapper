package autoresharing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	return nil
}

func (t Task) Execute(ctx context.Context) error {
	if t.BinaryPath == "" {
		return errors.New("binary path is not set")
	}

	if err := t.updateConfigBeforeResharing(); err != nil {
		return errors.Wrap(err, "failed to update parties config")
	}

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
		return errors.Wrap(err, "failed to execute resharing task")
	}

	var (
		epochId       uint32
		bridgeAddress string
		epoch         *bridgetypes.Epoch
		blockTime     time.Time
		err           error
	)

	err = retry.Do(
		func() error {
			epochId, err = helpers.GetEpoch(ctx, t.GRPCCore)
			if err != nil {
				return err
			}

			if epochId != t.EpochId {
				return errors.New("invalid epoch id")
			}
			return nil
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed to wait updated epoch")
	}

	err = retry.Do(
		func() error {
			epoch, err = helpers.GetEpochState(ctx, epochId, t.GRPCCore)
			return err
		})

	if err != nil {
		return errors.Wrap(err, "failed to get epoch state")
	}

	err = retry.Do(
		func() error {
			bridgeAddress, err = helpers.GetChainAddress(ctx, bridgetypes.ChainType_BITCOIN, t.GRPCCore)
			return err
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed to get bitcoin bridge address")
	}

	err = retry.Do(
		func() error {
			height := int64(epoch.FinalizedBlock)
			block, err := t.HTTPCore.Block(ctx, &height)
			if err != nil {
				return err
			}
			blockTime = block.Block.Time
			return nil
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed to get blocktime ")
	}

	return errors.Wrap(t.updateConfigAfterResharing(epoch, blockTime.Add(time.Hour), bridgeAddress), "failed to update config after start")
}

func (t Task) updateConfigBeforeResharing() error {
	configer := helpers.NewConfigManager(t.ConfigPath)
	if err := configer.Load(); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	parties, err := configer.GetParties(helpers.ResharingKey)
	if err != nil {
		return errors.Wrap(err, "failed to get parties")
	}

	newParties, err := t.determinePartiesConfig(parties)
	if err != nil {
		return errors.Wrap(err, "failed to determine parties config")
	}

	err = configer.UpdateResharingParams(t.EpochId, t.StartTime, t.isNewParty(), t.Threshold, newParties)
	if err != nil {
		return errors.Wrap(err, "failed to update parties config")
	}

	return errors.Wrap(configer.Save(), "failed to save config")
}

func (t Task) updateConfigAfterResharing(epoch *bridgetypes.Epoch, startTime time.Time, bridgeAddress string) error {
	configer := helpers.NewConfigManager(t.ConfigPath)
	if err := configer.Load(); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	if err := configer.UpdateBitcoinWallet(bridgeAddress, epoch.Id, "bitcoin"); err != nil {
		return errors.Wrap(err, "failed to update bitcoin wallet")
	}

	newParties, err := configer.GetParties(helpers.PartiesKey)
	if err != nil {
		return errors.Wrap(err, "failed to get new parties")
	}

	configer.SetParties(helpers.PartiesKey, newParties)
	err = configer.SetStartInfo(startTime, epoch.TssThreshold)
	if err != nil {
		return errors.Wrap(err, "failed to set start time")
	}

	return errors.Wrap(configer.Save(), "failed to save new parties")
}

// - Active TSS: add to parties list and store certificate
// - Inactive TSS: remove from parties list
func (t Task) determinePartiesConfig(currentParties []types.Party) ([]types.Party, error) {
	partyMap := make(map[string]types.Party)
	for _, p := range currentParties {
		partyMap[p.CoreAddress] = p
	}

	for _, tssInfo := range t.TssInfo {
		if tssInfo.Active {
			certPath, err := t.storeCertificate(tssInfo.Domen, tssInfo.Certificate)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to store certificate for %s", tssInfo.Domen))
			}

			partyMap[tssInfo.Address] = types.Party{
				Connection:         tssInfo.Domen,
				CoreAddress:        tssInfo.Address,
				TLSCertificatePath: certPath,
			}
			continue
		}

		if _, exists := partyMap[tssInfo.Address]; exists {
			delete(partyMap, tssInfo.Address)
		}

	}

	var updatedParties []types.Party
	isNewPartiesMember := false

	for _, p := range partyMap {
		if p.CoreAddress == t.CoreAddress {
			isNewPartiesMember = true
			// DO NOT store its own address
			continue
		}
		updatedParties = append(updatedParties, p)
	}

	if !isNewPartiesMember {
		updatedParties = []types.Party{}
	}

	return updatedParties, nil
}

func (t Task) storeCertificate(domain, certificate string) (string, error) {
	if t.CertificatesPath == "" {
		return "", errors.New("certificates path is not set")
	}

	if err := os.MkdirAll(t.CertificatesPath, 0755); err != nil {
		return "", errors.Wrap(err, "failed to create certificates directory")
	}

	certPath := filepath.Join(t.CertificatesPath, fmt.Sprintf("%s.crt", domain))
	if err := os.WriteFile(certPath, []byte(certificate), 0644); err != nil {
		return "", errors.Wrap(err, "failed to write certificate file")
	}

	return certPath, nil
}

func (t Task) isNewParty() bool {
	for _, info := range t.TssInfo {
		if info.Address == t.CoreAddress {
			return info.Active
		}
	}

	return false
}
