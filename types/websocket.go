package types

type WebSocketMessage struct {
	Action string      `json:"action,omitempty"`
	Type   string      `json:"type,omitempty"`
	Value  interface{} `json:"value,omitempty"`
}
