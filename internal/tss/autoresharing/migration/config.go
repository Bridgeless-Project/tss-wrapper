package migration

import (
	"time"

	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/helpers"
	"github.com/pkg/errors"
)

func (t *Task) updateConfigBeforeExecution(startTime time.Time, utxoChains []bridgetypes.Chain) error {
	configer := helpers.NewConfigManager(t.ConfigPath)
	if err := configer.Load(); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	params, err := configer.GetPrevEpochParams()
	if err != nil {
		return errors.Wrap(err, "failed to get previous epoch params")
	}

	if err = configer.UpdateResharingParams(
		t.EpochId,
		startTime,
		false,
		params.Threshold,
		params.Parties,
	); err != nil {
		return errors.Wrap(err, "failed to update resharing params")
	}

	for _, chain := range utxoChains {
		oldData, ok := params.BitcoinChainsData[chain.Id]
		if !ok {
			return errors.Errorf("no data found for chain %s in previous epoch params", chain.Id)
		}

		newEpochWalletName, err := configer.UpdateBitcoinWallet(
			oldData.OldWalletName,
			chain.Id,
			// no address to update here
		)
		if err != nil {
			return errors.Wrapf(err, "failed to update btc wallet for chain %s", chain.Id)
		}

		params.BitcoinChainsData[chain.Id] = helpers.BitcoinChainData{
			OldWalletName:    newEpochWalletName,
			NewBridgeAddress: oldData.NewBridgeAddress,
		}
	}
	t.btcChainsData = params.BitcoinChainsData

	return errors.Wrap(configer.Save(), "failed to save config")
}

func (t *Task) updateConfigAfterExecution(startTime time.Time, utxoChains []bridgetypes.Chain) error {
	configer := helpers.NewConfigManager(t.ConfigPath)
	if err := configer.Load(); err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	configer.SetStartTime(startTime)

	for _, chain := range utxoChains {
		oldData, ok := t.btcChainsData[chain.Id]
		if !ok {
			return errors.Errorf("no data found for chain %s in previous epoch params", chain.Id)
		}

		_, err := configer.UpdateBitcoinWallet(
			oldData.OldWalletName,
			chain.Id,
			helpers.WalletBridgeAddressData{
				Address:    chain.BridgeAddress,
				ReplaceAll: true, // remove old addresses since final migration is done
			},
		)
		if err != nil {
			return errors.Wrapf(err, "failed to update btc wallet for chain %s", chain.Id)
		}
	}

	return errors.Wrap(configer.Save(), "failed to save config")
}
