package service

import (
	"context"
	"net/http"

	"api/internal/repository"
	"api/types"

	"github.com/google/uuid"
)

type Service struct {
	repository          repository.Repository
	webSocketRepository repository.WebSocketRepository
}

func New(repo repository.Repository, webSocketRepo repository.WebSocketRepository) *Service {
	return &Service{
		repository:          repo,
		webSocketRepository: webSocketRepo,
	}
}

func (s *Service) CreateRetrospective(ctx context.Context, retro *types.Retrospective) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}

	retro.ID = id
	err = s.repository.CreateRetrospective(ctx, retro)
	if err != nil {
		return err
	}
	return s.webSocketRepository.CreateRetrospective(ctx, retro)
}

func (s *Service) GetRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	return s.repository.GetRetrospective(ctx, id)
}

func (s *Service) DeleteRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	retro, err := s.repository.DeleteRetrospective(ctx, id)
	if err != nil {
		return nil, err
	}
	_, err = s.webSocketRepository.DeleteRetrospective(ctx, id)
	return retro, err
}

func (s *Service) UpdateRetrospective(ctx context.Context, retro *types.Retrospective) error {
	return s.repository.UpdateRetrospective(ctx, retro)
}

func (s *Service) CreateQuestion(ctx context.Context, question *types.Question) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}

	question.ID = id
	err = s.repository.CreateQuestion(ctx, question)
	if err != nil {
		return err
	}
	return s.webSocketRepository.CreateQuestion(ctx, question)
}

func (s *Service) UpdateQuestion(ctx context.Context, question *types.Question) error {
	err := s.repository.UpdateQuestion(ctx, question)
	if err != nil {
		return err
	}
	return s.webSocketRepository.UpdateQuestion(ctx, question)
}

func (s *Service) DeleteQuestion(ctx context.Context, id uuid.UUID) (*types.Question, error) {
	question, err := s.repository.DeleteQuestion(ctx, id)
	if err != nil {
		return question, err
	}
	_, err = s.webSocketRepository.DeleteQuestion(ctx, id)
	return question, err
}

func (s *Service) SubscribeChanges(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return s.webSocketRepository.AddConnection(ctx, w, r)
}
