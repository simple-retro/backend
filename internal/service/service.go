package service

import (
	"api/config"
	"api/internal/repository"
	"api/types"
	"context"
	"log"
	"net/http"
	"time"

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
	retro.CreatedAt = time.Now().UTC()
	err = s.repository.CreateRetrospective(ctx, retro)
	if err != nil {
		return err
	}
	return s.webSocketRepository.CreateRetrospective(ctx, retro)
}

func (s *Service) GetRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	config := config.Get()
	cleanUpDays := time.Duration(config.Schedule.CleanUpDays)

	retro, err := s.repository.GetRetrospective(ctx, id)
	retro.ExpireAt = retro.CreatedAt.Add(cleanUpDays * 24 * time.Hour)
	return retro, err
}

func (s *Service) DeleteRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	config := config.Get()
	cleanUpDays := time.Duration(config.Schedule.CleanUpDays)
	retro, err := s.repository.DeleteRetrospective(ctx, id)
	if err != nil {
		return nil, err
	}
	retro.ExpireAt = retro.CreatedAt.Add(cleanUpDays * 24 * time.Hour)
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

func (s *Service) CreateAnswer(ctx context.Context, answer *types.Answer) error {
	id, err := uuid.NewV7()
	if err != nil {
		return nil
	}

	answer.ID = id
	err = s.repository.CreateAnswer(ctx, answer)
	if err != nil {
		return err
	}
	return s.webSocketRepository.CreateAnswer(ctx, answer)
}

func (s *Service) UpdateAnswer(ctx context.Context, answer *types.Answer) error {
	err := s.repository.UpdateAnswer(ctx, answer)
	if err != nil {
		return err
	}
	return s.webSocketRepository.UpdateAnswer(ctx, answer)
}

func (s *Service) DeleteAnswer(ctx context.Context, answer *types.Answer) error {
	err := s.repository.DeleteAnswer(ctx, answer)
	if err != nil {
		return err
	}
	err = s.webSocketRepository.DeleteAnswer(ctx, answer)
	return err
}

func (s *Service) SubscribeChanges(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return s.webSocketRepository.AddConnection(ctx, w, r)
}

func (s *Service) LoadAllRetrospectives(ctx context.Context) error {
	ids, err := s.repository.GetAllRetrospectives(ctx)
	if err != nil {
		return err
	}

	for _, id := range ids {
		err := s.webSocketRepository.CreateRetrospective(ctx, &types.Retrospective{ID: id})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) GetLimits(ctx context.Context) *types.ApiLimits {
	return types.GetApiLimits()
}

func (s *Service) CleanUpRetros(ctx context.Context) error {
	config := config.Get()
	cleanUpDays := time.Duration(config.Schedule.CleanUpDays)
	date := time.Now().Add(-(cleanUpDays * 24 * time.Hour))
	ids, err := s.repository.GetOldRetrospectives(ctx, date)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if _, err := s.repository.DeleteRetrospective(ctx, id); err != nil {
			return err
		}
	}

	log.Printf("deleted %d retrospectives older than %s", len(ids), date.String())
	return nil
}
