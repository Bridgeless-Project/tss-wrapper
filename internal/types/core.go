package types

type BridgeParams struct {
	ModuleAdmin string `json:"module_admin"`
	Parties     []struct {
		Address string `json:"address"`
	} `json:"parties"`
	TssThreshold    int      `json:"tss_threshold"`
	RelayerAccounts []string `json:"relayer_accounts"`
	Epoch           int      `json:"epoch"`
	SupportingTime  string   `json:"supporting_time"`
}
