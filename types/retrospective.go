package types

import (
	"github.com/google/uuid"
)

type Retrospective struct {
	ID          uuid.UUID  `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Questions   []Question `json:"questions,omitempty"`
}

type Question struct {
	ID      uuid.UUID `json:"id,omitempty"`
	Text    string    `json:"text,omitempty"`
	Answers []Answer  `json:"answers,omitempty"`
}

type Answer struct {
	ID       uuid.UUID `json:"id,omitempty"`
	Text     string    `json:"text,omitempty"`
	Position int       `json:"position,omitempty"`
}
