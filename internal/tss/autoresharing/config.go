package autoresharing

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/helpers"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"

	"github.com/pkg/errors"
)

func (t Task) updateConfigBeforeResharing() error {
	configer := helpers.NewConfigManager(t.ConfigPath)
	if err := configer.Load(); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	parties, err := configer.GetParties(helpers.PartiesKey)
	if err != nil {
		return errors.Wrap(err, "failed to get parties")
	}

	newParties, err := t.determinePartiesConfig(parties)
	if err != nil {
		return errors.Wrap(err, "failed to determine parties config")
	}

	err = configer.UpdateResharingParams(t.EpochId, t.StartTime.Add(10*time.Second), t.isNewParty(), t.Threshold, newParties)
	if err != nil {
		return errors.Wrap(err, "failed to update parties config")
	}

	return errors.Wrap(configer.Save(), "failed to save config")
}

func (t Task) updateConfigAfterResharing(epoch *bridgetypes.Epoch, startTime time.Time, utxoChains []bridgetypes.Chain) error {
	configer := helpers.NewConfigManager(t.ConfigPath)
	if err := configer.Load(); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	for _, chain := range utxoChains {
		if err := configer.UpdateBitcoinWallet(chain.BridgeAddress, epoch.Id, chain.Id); err != nil {
			return errors.Wrap(err, "failed to update bitcoin wallet")
		}
	}

	newParties, err := configer.GetResharingParties()
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
	isNewPartiesMember := true

	for _, tssInfo := range t.TssInfo {
		if tssInfo.Active {
			if tssInfo.Address == t.CoreAddress {
				continue
			}
			certPath, err := t.storeCertificate(tssInfo.Domain, tssInfo.Certificate)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to store certificate for %s", tssInfo.Domain))
			}

			partyMap[tssInfo.Address] = types.Party{
				Connection:         tssInfo.Domain,
				CoreAddress:        tssInfo.Address,
				TLSCertificatePath: certPath,
			}
			continue
		} else {
			if tssInfo.Address == t.CoreAddress {
				isNewPartiesMember = false
			}

			if _, exists := partyMap[tssInfo.Address]; exists {
				delete(partyMap, tssInfo.Address)
			}
		}

	}

	var updatedParties []types.Party

	for _, p := range partyMap {
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
