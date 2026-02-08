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
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/Bridgeless-Project/tss-wrapper-svc/utils"
	"github.com/pkg/errors"
)

type Task struct {
	EpochId          uint32
	TssInfo          []bridgeTypes.TSSInfo
	StartTime        time.Time
	BinaryPath       string
	ConfigPath       string
	CertificatesPath string
}

func NewTask(binaryPath, configPath, certificatesPath string) *Task {
	return &Task{
		BinaryPath:       binaryPath,
		ConfigPath:       configPath,
		CertificatesPath: certificatesPath,
	}
}

func (t Task) Execute(ctx context.Context) error {
	if t.BinaryPath == "" {
		return errors.New("binary path is not set")
	}

	// Update config with new parties based on TSSInfo
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

	return nil
}

// updatePartiesConfig updates the TSS config file based on TSSInfo:
// - Active TSS: add to parties list and store certificate
// - Inactive TSS: remove from parties list
func (t Task) updatePartiesConfig() error {
	configMgr := utils.NewConfigManager(t.ConfigPath)
	if err := configMgr.Load(); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	currentParties, err := configMgr.GetParties()
	if err != nil {
		return errors.Wrap(err, "failed to get current parties")
	}

	partyMap := make(map[string]utils.Party)
	for _, p := range currentParties {
		partyMap[p.CoreAddress] = p
	}

	for _, tssInfo := range t.TssInfo {
		if tssInfo.Active {
			certPath, err := t.storeCertificate(tssInfo.Domen, tssInfo.Certificate)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to store certificate for %s", tssInfo.Domen))
			}

			partyMap[tssInfo.Address] = utils.Party{
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

	var updatedParties []utils.Party
	for _, p := range partyMap {
		updatedParties = append(updatedParties, p)
	}

	configMgr.SetParties(updatedParties)
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

	certFileName := fmt.Sprintf("%s.crt", domain)
	certPath := filepath.Join(t.CertificatesPath, certFileName)

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
	return "AutoResharingTask"
}
