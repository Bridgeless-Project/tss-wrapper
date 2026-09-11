package types

type Party struct {
	Connection         string `yaml:"connection"`
	CoreAddress        string `yaml:"core_address"`
	TLSCertificatePath string `yaml:"tls_certificate_path"`
}

type PartiesConfig struct {
	List []Party `yaml:"list"`
}

type Chains struct {
	List []Chain `yaml:"list"`
}

type Chain struct {
	BridgeAddress string `yaml:"bridge_address"`
	Id            string `yaml:"id"`
	Meta          Meta   `yaml:"meta"`
}

type Meta struct {
	Centralized bool   `yaml:"centralized"`
	SignerKey   string `yaml:"signer_key"`
	RPC         RPC    `yaml:"rpc"`
	Type        string `yaml:"type"`
}

type RPC struct {
	Wallet Node `yaml:"wallet"`
	Node   Node `yaml:"node"`
}

type Node struct {
	Host string `yaml:"host"`
	Pass string `yaml:"pass"`
	User string `yaml:"user"`
}

type TSSConfigFile struct {
	Parties PartiesConfig          `yaml:"parties"`
	Chains  Chains                 `yaml:"chains"`
	other   map[string]interface{} `yaml:"-"`
}
