package types

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type Event struct {
	Type       string      `json:"type"`
	Attributes []Attribute `json:"attributes"`
}

type MsgEvent struct {
	MsgIndex int     `json:"msg_index"`
	Events   []Event `json:"events"`
}
