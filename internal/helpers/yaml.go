package helpers

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	ResharingKey     = "resharing"
	PartiesKey       = "parties"
	PrevEpochDataKey = "prev_epoch_data"
)

const (
	keyChains           = "chains"
	keyPartiesList      = "list"
	keyIsNewParticipant = "new_participant"
	keyEpoch            = "epoch"
	keyTss              = "tss"
	keyThreshold        = "threshold"
	keyStartTime        = "start_time"
	keyChainId          = "id"
	keyPRC              = "rpc"
	keyWallet           = "wallet"
	keyWallets          = "wallets"
	keyHost             = "host"
	keyBridgeAddress    = "bridge_addresses"
)

var (
	ErrOldEpochDataNotFound = errors.New("no data found for previous epoch")
)

type ConfigManager struct {
	configPath string
	rawConfig  map[string]interface{}
}

func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// -------------------BASE LOGIC----------------------

func (c *ConfigManager) Load() error {
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return errors.Wrap(err, "failed to read config file")
	}

	c.rawConfig = make(map[string]interface{})
	if err = yaml.Unmarshal(data, &c.rawConfig); err != nil {
		return errors.Wrap(err, "failed to unmarshal config")
	}

	return nil
}

func (c *ConfigManager) Save() error {
	data, err := yaml.Marshal(c.rawConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal config")
	}

	if err = os.WriteFile(c.configPath, data, 0644); err != nil {
		return errors.Wrap(err, "failed to write config file")
	}

	return nil
}

func (c *ConfigManager) GetThreshold() (uint32, error) {
	tssMap, ok := c.rawConfig[keyTss].(map[string]interface{})
	if !ok {
		return 0, errors.New("invalid tss format")
	}

	switch threshold := tssMap[keyThreshold].(type) {
	case int:
		return uint32(threshold), nil
	case uint32:
		return threshold, nil
	default:
		return 0, errors.New(fmt.Sprintf("invalid tss threshold type: %s", threshold))
	}
}

func (c *ConfigManager) SetThreshold(threshold uint32) {
	tssMap, ok := c.rawConfig[keyTss].(map[string]interface{})
	if !ok {
		tssMap = make(map[string]interface{})
		c.rawConfig[keyTss] = tssMap
	}

	tssMap[keyThreshold] = threshold
}

func (c *ConfigManager) SetStartTime(startTime time.Time) {
	tssMap, ok := c.rawConfig[keyTss].(map[string]interface{})
	if !ok {
		tssMap = make(map[string]interface{})
		c.rawConfig[keyTss] = tssMap
	}

	tssMap[keyStartTime] = startTime
}

// -------------------PARTIES---------------------

func (c *ConfigManager) GetParties(key string) ([]types.Party, error) {
	partiesMap, ok := c.rawConfig[key].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid parties format")
	}

	listSlice, ok := partiesMap[keyPartiesList].([]interface{})
	if !ok {
		return nil, errors.New("invalid parties list format")
	}

	var parties []types.Party
	for _, item := range listSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		party := types.Party{}
		if conn, ok := itemMap["connection"].(string); ok {
			party.Connection = conn
		}
		if addr, ok := itemMap["core_address"].(string); ok {
			party.CoreAddress = addr
		}
		if cert, ok := itemMap["tls_certificate_path"].(string); ok {
			party.TLSCertificatePath = cert
		}
		parties = append(parties, party)
	}

	return parties, nil
}

func (c *ConfigManager) SetParties(key string, parties []types.Party) {
	var listRaw []interface{}
	for _, p := range parties {
		listRaw = append(listRaw, map[string]interface{}{
			"connection":           p.Connection,
			"core_address":         p.CoreAddress,
			"tls_certificate_path": p.TLSCertificatePath,
		})
	}

	if c.rawConfig[key] == nil {
		c.rawConfig[key] = make(map[string]interface{})
	}

	partiesMap, ok := c.rawConfig[key].(map[string]interface{})
	if !ok {
		partiesMap = make(map[string]interface{})
		c.rawConfig[key] = partiesMap
	}

	partiesMap[keyPartiesList] = listRaw
}

// ----------------- RESHARING PARAMS ----------

func (c *ConfigManager) UpdateResharingParams(epoch uint32, startTime time.Time, isNew bool, threshold uint32, parties []types.Party) error {
	resharingParamsMap, ok := c.rawConfig[ResharingKey].(map[string]interface{})
	if !ok {
		resharingParamsMap = make(map[string]interface{})
		c.rawConfig[ResharingKey] = resharingParamsMap
	}

	var listRaw []interface{}
	for _, p := range parties {
		listRaw = append(listRaw, map[string]interface{}{
			"connection":           p.Connection,
			"core_address":         p.CoreAddress,
			"tls_certificate_path": p.TLSCertificatePath,
		})
	}

	resharingParamsMap[keyEpoch] = epoch
	resharingParamsMap[keyStartTime] = startTime
	resharingParamsMap[keyIsNewParticipant] = isNew
	resharingParamsMap[keyThreshold] = threshold
	resharingParamsMap[PartiesKey] = map[string]interface{}{
		keyPartiesList: listRaw,
	}

	return nil
}

func (c *ConfigManager) UpdateResharingTime(startTime time.Time) error {
	resharingParamsMap, ok := c.rawConfig[ResharingKey].(map[string]interface{})
	if !ok {
		resharingParamsMap = make(map[string]interface{})
		c.rawConfig[ResharingKey] = resharingParamsMap
	}
	resharingParamsMap[keyStartTime] = startTime
	return nil
}

func (c *ConfigManager) GetResharingParties() ([]types.Party, error) {
	resharingParams, ok := c.rawConfig[ResharingKey].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid resharing format")
	}

	partiesMap, ok := resharingParams[PartiesKey].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid parties format")
	}

	listSlice, ok := partiesMap[keyPartiesList].([]interface{})
	if !ok {
		return nil, errors.New("invalid parties list format")
	}

	var parties []types.Party
	for _, item := range listSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		party := types.Party{}
		if conn, ok := itemMap["connection"].(string); ok {
			party.Connection = conn
		}
		if addr, ok := itemMap["core_address"].(string); ok {
			party.CoreAddress = addr
		}
		if cert, ok := itemMap["tls_certificate_path"].(string); ok {
			party.TLSCertificatePath = cert
		}
		parties = append(parties, party)
	}

	return parties, nil
}

type PrevEpochParams struct {
	Threshold         uint32
	Parties           []types.Party
	BitcoinChainsData map[string]BitcoinChainData
}

type BitcoinChainData struct {
	NewBridgeAddress string
	OldWalletName    string
}

func (c *ConfigManager) SetPrevEpochParams(params PrevEpochParams) {
	prevEpochMap, ok := c.rawConfig[PrevEpochDataKey].(map[string]interface{})
	if !ok {
		prevEpochMap = make(map[string]interface{})
		c.rawConfig[PrevEpochDataKey] = prevEpochMap
	}

	prevEpochMap[keyThreshold] = params.Threshold
	prevEpochMap[PartiesKey] = map[string]interface{}{
		keyPartiesList: partiesToListRaw(params.Parties),
	}

	walletsMap := make(map[string]interface{}, len(params.BitcoinChainsData))
	for chainId, chainData := range params.BitcoinChainsData {
		walletsMap[chainId] = map[string]interface{}{
			"new_bridge_address": chainData.NewBridgeAddress,
			"old_wallet_name":    chainData.OldWalletName,
		}
	}
	prevEpochMap[keyWallets] = walletsMap
}

func (c *ConfigManager) GetPrevEpochParams() (PrevEpochParams, error) {
	params := PrevEpochParams{}
	prevEpochMap, ok := c.rawConfig[PrevEpochDataKey].(map[string]interface{})
	if !ok {
		return params, ErrOldEpochDataNotFound
	}
	params.Threshold, ok = prevEpochMap[keyThreshold].(uint32)
	if !ok {
		return params, errors.New("invalid threshold format")
	}

	partiesMap, ok := prevEpochMap[PartiesKey].(map[string]interface{})
	if !ok {
		return params, errors.New("invalid parties format")
	}
	listSlice, ok := partiesMap[keyPartiesList].([]interface{})
	if !ok {
		return params, errors.New("invalid parties list format")
	}
	params.Parties = partiesFromListRaw(listSlice)

	walletsMap, ok := prevEpochMap[keyWallets].(map[string]interface{})
	if !ok {
		return params, errors.New("invalid wallets format")
	}

	params.BitcoinChainsData = make(map[string]BitcoinChainData, len(walletsMap))
	for chainId, walletData := range walletsMap {
		walletDataMap, ok := walletData.(map[string]interface{})
		if !ok {
			return params, errors.New("invalid wallet data format")
		}
		newBridgeAddress, ok := walletDataMap["new_bridge_address"].(string)
		if !ok || newBridgeAddress == "" {
			return params, errors.New("invalid new bridge address format")
		}
		oldWalletName, ok := walletDataMap["old_wallet_name"].(string)
		if !ok || oldWalletName == "" {
			return params, errors.New("invalid old wallet name format")
		}
		params.BitcoinChainsData[chainId] = BitcoinChainData{
			NewBridgeAddress: newBridgeAddress,
			OldWalletName:    oldWalletName,
		}
	}

	return params, nil
}

type WalletBridgeAddressData struct {
	Address    string
	ReplaceAll bool
}

// -------------------CHAINS-------------------
func (c *ConfigManager) UpdateBitcoinWallet(walletName, chainId string, address ...WalletBridgeAddressData) (string, error) {
	partiesMap, ok := c.rawConfig[keyChains].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid chains format")
	}

	listSlice, ok := partiesMap[keyPartiesList].([]interface{})
	if !ok {
		return "", errors.New("invalid parties list format")
	}

	for _, chain := range listSlice {
		chainMap, ok := chain.(map[string]interface{})
		if !ok {
			continue
		}

		if chainMap[keyChainId] != chainId {
			continue
		}

		chainMap[keyChainId] = chainId
		rpc := chainMap[keyPRC].(map[string]interface{})
		wallet := rpc[keyWallet].(map[string]interface{})
		oldWalletHost := wallet[keyHost].(string)
		hostParts := strings.SplitAfter(oldWalletHost, "/wallet")
		wallet[keyHost] = fmt.Sprintf("%s/%s", hostParts[0], walletName)

		if len(address) == 0 {
			return oldWalletHost, nil
		}
		bridgeAddresses := chainMap[keyBridgeAddress].([]interface{})
		if address[0].ReplaceAll {
			bridgeAddresses = []interface{}{address}
		} else {
			bridgeAddresses = append(bridgeAddresses, address)
		}

		chainMap[keyBridgeAddress] = bridgeAddresses

		return oldWalletHost, nil
	}

	return "", errors.New("chain not found")
}

// ------------------ START TIME ------------------

func (c *ConfigManager) SetStartInfo(timestamp time.Time, threshold uint32) error {
	tssMap, ok := c.rawConfig[keyTss].(map[string]interface{})
	if !ok {
		return errors.New("invalid tss format")
	}

	tssMap[keyStartTime] = timestamp
	tssMap[keyThreshold] = threshold

	return nil
}

func partiesToListRaw(parties []types.Party) []interface{} {
	var listRaw []interface{}
	for _, p := range parties {
		listRaw = append(listRaw, map[string]interface{}{
			"connection":           p.Connection,
			"core_address":         p.CoreAddress,
			"tls_certificate_path": p.TLSCertificatePath,
		})
	}

	return listRaw
}

func partiesFromListRaw(listRaw []interface{}) []types.Party {
	var parties []types.Party
	for _, item := range listRaw {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		party := types.Party{}
		if conn, ok := itemMap["connection"].(string); ok {
			party.Connection = conn
		}
		if addr, ok := itemMap["core_address"].(string); ok {
			party.CoreAddress = addr
		}
		if cert, ok := itemMap["tls_certificate_path"].(string); ok {
			party.TLSCertificatePath = cert
		}
		parties = append(parties, party)
	}

	return parties
}
