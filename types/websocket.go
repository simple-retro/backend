package types

import "github.com/google/uuid"

type Object struct {
	ID uuid.UUID `json:"id,omitempty"`
}

type WebSocketMessage struct {
	Action string      `json:"action,omitempty"`
	Type   string      `json:"type,omitempty"`
	Value  interface{} `json:"value,omitempty"`
}
