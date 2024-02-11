package repository

import (
	"api/config"
	"api/types"
	"context"
	"testing"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestCreateRepository(t *testing.T) {
	conf, err := config.Load("../../config/config-test.yaml")
	t.Log(conf)
	assert.Nilf(t, err, "error loading config")

	db, err := NewSQLite()
	assert.Nilf(t, err, "error connecting to database")

	id, err := uuid.NewV7()
	assert.Nilf(t, err, "error generating UUID")
	ctx := context.Background()
	retro := &types.Retrospective{
		ID:          id,
		Name:        "mtg",
		Description: "df/dx = 0",
	}

	err = db.CreateRetrospective(ctx, retro)
	assert.Nilf(t, err, "error creating retrospective")

	res := &types.Retrospective{}
	sqlQuery := `SELECT id, name, description FROM retrospectives WHERE id = $1`
	err = db.conn.QueryRow(sqlQuery, retro.ID).Scan(
		&res.ID,
		&res.Name,
		&res.Description,
	)

	assert.Nilf(t, err, "error getting created retrospective")
	assert.Equal(t, retro, res)
}
