package repository

import (
	"api/types"
	"context"
	"net/http"

	"github.com/google/uuid"
)

type Repository interface {
	GetAllRetrospectives(ctx context.Context) ([]uuid.UUID, error)
	GetRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error)
	CreateRetrospective(ctx context.Context, retro *types.Retrospective) error
	UpdateRetrospective(ctx context.Context, retro *types.Retrospective) error
	DeleteRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error)
	CreateQuestion(ctx context.Context, question *types.Question) error
	UpdateQuestion(ctx context.Context, question *types.Question) error
	DeleteQuestion(ctx context.Context, id uuid.UUID) (*types.Question, error)
	CreateAnswer(ctx context.Context, answer *types.Answer) error
	UpdateAnswer(ctx context.Context, answer *types.Answer) error
	DeleteAnswer(ctx context.Context, answer *types.Answer) error
}

type WebSocketRepository interface {
	Repository
	AddConnection(ctx context.Context, w http.ResponseWriter, r *http.Request) error
}
