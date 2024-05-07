package types

import (
	"time"

	"github.com/google/uuid"
)

type Retrospective struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Questions   []Question `json:"questions"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpireAt    time.Time  `json:"expire_at"`
}

type Question struct {
	ID      uuid.UUID `json:"id"`
	Text    string    `json:"text"`
	Answers []Answer  `json:"answers"`
}

type Answer struct {
	ID         uuid.UUID `json:"id"`
	QuestionID uuid.UUID `json:"question_id"`
	Text       string    `json:"text"`
	Position   int       `json:"position"`
}

type RetrospectiveCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type QuestionCreateRequest struct {
	Text string `json:"text"`
}

type AnswerCreateRequest struct {
	QuestionID uuid.UUID `json:"question_id"`
	Text       string    `json:"text"`
}
