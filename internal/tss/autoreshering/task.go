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

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/helpers"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/avast/retry-go"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/pkg/errors"
)

const TaskType = "auto_resharing"

// taskData represents the serializable part of the task for database storage
type taskData struct {
	EpochId   uint32                `json:"epoch_id"`
	TssInfo   []bridgeTypes.TSSInfo `json:"tss_info"`
	StartTime int64                 `json:"start_time"`
}

type Task struct {
	id               int64 // database ID
	EpochId          uint32
	TssInfo          []bridgeTypes.TSSInfo
	StartTime        time.Time
	BinaryPath       string
	ConfigPath       string
	CertificatesPath string
	Core             grpc.ClientConn
}

func NewTask(binaryPath, configPath, certificatesPath string, con grpc.ClientConn) *Task {
	return &Task{
		BinaryPath:       binaryPath,
		ConfigPath:       configPath,
		CertificatesPath: certificatesPath,
		Core:             con,
	}
}

func (t Task) Execute(ctx context.Context) error {
	if t.BinaryPath == "" {
		return errors.New("binary path is not set")
	}

	if err := t.updatePartiesConfig(); err != nil {
		return errors.Wrap(err, "failed to update parties config")
	}

	args := []string{
		"reshare",
		"--config", t.ConfigPath,
		"--epoch", fmt.Sprintf("%d", t.EpochId),
	}

	cmd := exec.CommandContext(ctx, t.BinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to execute resharing task")
	}

	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "resharing process did not complete successfully")
	}

	var epoch uint32
	var err error

	retry.Do(
		func() error {
			epoch, err = helpers.GetEpoch(ctx, t.Core)
			if err != nil {
				return err
			}

			if epoch != t.EpochId {
				return errors.New("invalid epoch id")
			}
			return nil
		},
	)

	// Update config
	// move new parties to parties
	//

	return nil
}

// updatePartiesConfig updates the TSS config file based on TSSInfo:
// - Active TSS: add to parties list and store certificate
// - Inactive TSS: remove from parties list
func (t Task) updatePartiesConfig() error {
	configMgr := helpers.NewConfigManager(t.ConfigPath)
	if err := configMgr.Load(); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	currentParties, err := configMgr.GetParties(helpers.PartiesKey)
	if err != nil {
		return errors.Wrap(err, "failed to get current parties")
	}

	partyMap := make(map[string]types.Party)
	for _, p := range currentParties {
		partyMap[p.CoreAddress] = p
	}

	for _, tssInfo := range t.TssInfo {
		if tssInfo.Active {
			certPath, err := t.storeCertificate(tssInfo.Domen, tssInfo.Certificate)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to store certificate for %s", tssInfo.Domen))
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
	for _, p := range partyMap {
		updatedParties = append(updatedParties, p)
	}

	configMgr.SetParties(helpers.PartiesKey, updatedParties)
	if err = configMgr.Save(); err != nil {
		return errors.Wrap(err, "failed to save config")
	}

	return nil
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

func (t Task) GetTime() time.Time {
	return t.StartTime
}

func (t Task) Parse(attributes []types.Attribute) (types.Task, error) {
	task := &Task{
		BinaryPath:       t.BinaryPath,
		ConfigPath:       t.ConfigPath,
		CertificatesPath: t.CertificatesPath,
	}

	for _, attribute := range attributes {
		switch attribute.Key {
		case bridgeTypes.AttributeTssInfo:
			if err := json.Unmarshal([]byte(attribute.Value), &task.TssInfo); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal tss info")
			}
		case bridgeTypes.AttributeEpochId:
			epoch, err := strconv.ParseUint(attribute.Value, 10, 32)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse epoch id")
			}
			task.EpochId = uint32(epoch)
		case bridgeTypes.AttributeEpochStartTime:
			startTime, err := strconv.ParseInt(attribute.Value, 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse start time")
			}
			task.StartTime = time.Unix(startTime, 0)
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

func (t Task) Name() string {
	return "AutoResharingTask"
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

// UnmarshalData deserializes the task data from JSON
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
