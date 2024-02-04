package repository

import "api/types"

type Repository interface {
	CreateRetrospective(retro *types.Retrospective) error
}
