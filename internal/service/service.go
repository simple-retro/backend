package service

import (
	"api/internal/repository"
	"api/types"
)

type Service struct {
	repository repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{
		repository: repo,
	}
}

func (s *Service) CreateRetrospective(retro *types.Retrospective) error {
	return nil
}
