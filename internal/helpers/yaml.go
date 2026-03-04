package helpers

import (
	"os"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	ResharingKey = "resharing"
	PartiesKey   = "parties"
)

const (
	keyChains           = "chains"
	keyPartiesList      = "list"
	keyChainType        = "type"
	keyIsNewParticipant = "new_participant"
	keyEpoch            = "epoch"
	keyTss              = "tss"
	keyThreshold        = "threshold"
	keyStartTime        = "start_time"
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

// -------------------CHAINS-------------------
func (c *ConfigManager) UpdateBitcoinWallet(address string, epoch uint32, chainType string) error {
	partiesMap, ok := c.rawConfig[keyChains].(map[string]interface{})
	if !ok {
		return errors.New("invalid chains format")
	}

	listSlice, ok := partiesMap[keyPartiesList].([]interface{})
	if !ok {
		return errors.New("invalid parties list format")
	}

	for _, chain := range listSlice {
		chainMap, ok := chain.(map[string]interface{})
		if !ok {
			continue
		}

		if chainMap[keyChainType] == chainType {
			chainMap["address"] = address
			chainMap["epoch"] = epoch
		}
	}

	return nil
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
