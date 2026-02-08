package utils

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Party represents a single party entry in the TSS config
type Party struct {
	Connection         string `yaml:"connection"`
	CoreAddress        string `yaml:"core_address"`
	TLSCertificatePath string `yaml:"tls_certificate_path"`
}

// PartiesConfig represents the parties section of the TSS config
type PartiesConfig struct {
	List []Party `yaml:"list"`
}

// TSSConfigFile represents the structure of the TSS config file
type TSSConfigFile struct {
	Parties PartiesConfig `yaml:"parties"`
	// Preserve other fields using yaml.Node for round-trip parsing
	other map[string]interface{} `yaml:"-"`
}

// ConfigManager handles reading and writing the TSS config file
type ConfigManager struct {
	configPath string
	rawConfig  map[string]interface{}
}

// NewConfigManager creates a new config manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// Load reads the config file
func (c *ConfigManager) Load() error {
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return errors.Wrap(err, "failed to read config file")
	}

	c.rawConfig = make(map[string]interface{})
	if err := yaml.Unmarshal(data, &c.rawConfig); err != nil {
		return errors.Wrap(err, "failed to unmarshal config")
	}

	return nil
}

// GetParties returns the current parties list
func (c *ConfigManager) GetParties() ([]Party, error) {
	partiesRaw, ok := c.rawConfig["parties"]
	if !ok {
		return []Party{}, nil
	}

	partiesMap, ok := partiesRaw.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid parties format")
	}

	listRaw, ok := partiesMap["list"]
	if !ok {
		return []Party{}, nil
	}

	listSlice, ok := listRaw.([]interface{})
	if !ok {
		return nil, errors.New("invalid parties list format")
	}

	var parties []Party
	for _, item := range listSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		party := Party{}
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

// SetParties updates the parties list in the config
func (c *ConfigManager) SetParties(parties []Party) {
	// Convert parties to interface slice for YAML
	var listRaw []interface{}
	for _, p := range parties {
		listRaw = append(listRaw, map[string]interface{}{
			"connection":           p.Connection,
			"core_address":         p.CoreAddress,
			"tls_certificate_path": p.TLSCertificatePath,
		})
	}

	// Ensure parties map exists
	if c.rawConfig["parties"] == nil {
		c.rawConfig["parties"] = make(map[string]interface{})
	}

	partiesMap, ok := c.rawConfig["parties"].(map[string]interface{})
	if !ok {
		partiesMap = make(map[string]interface{})
		c.rawConfig["parties"] = partiesMap
	}

	partiesMap["list"] = listRaw
}

// Save writes the config back to file
func (c *ConfigManager) Save() error {
	data, err := yaml.Marshal(c.rawConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal config")
	}

	if err := os.WriteFile(c.configPath, data, 0644); err != nil {
		return errors.Wrap(err, "failed to write config file")
	}

	return nil
}
