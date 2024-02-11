package repository

import (
	"api/config"
	"api/types"
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestCreateRetrospective(t *testing.T) {
	_, err := config.Load("../../config/config_test.yaml")
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

func TestUpdateRetrospective(t *testing.T) {
	_, err := config.Load("../../config/config_test.yaml")
	assert.Nilf(t, err, "error loading config")

	db, err := NewSQLite()
	assert.Nilf(t, err, "error connecting to database")

	id, err := uuid.NewV7()
	assert.Nilf(t, err, "error generating UUID")
	retro := &types.Retrospective{
		ID:          id,
		Name:        "mtg",
		Description: "df/dx = 0",
	}

	sqlQuery := `INSERT INTO retrospectives (id, name, description) VALUES ($1, $2, $3)`
	_, err = db.conn.Exec(
		sqlQuery,
		&retro.ID,
		&retro.Name,
		&retro.Description,
	)

	assert.Nilf(t, err, "error creating retrospective")

	retro.Name = "Changed name"
	retro.Description = "Changed description"

	ctx := context.Background()
	err = db.UpdateRetrospective(ctx, retro)
	assert.Nilf(t, err, "error updating retrospective")

	res := &types.Retrospective{}
	sqlQuery = `SELECT id, name, description FROM retrospectives WHERE id = $1`
	err = db.conn.QueryRow(sqlQuery, retro.ID).Scan(
		&res.ID,
		&res.Name,
		&res.Description,
	)

	assert.Nilf(t, err, "error getting created retrospective")
	assert.Equal(t, retro, res)
}

func TestDeleteRetrospective(t *testing.T) {
	_, err := config.Load("../../config/config_test.yaml")
	assert.Nilf(t, err, "error loading config")

	db, err := NewSQLite()
	assert.Nilf(t, err, "error connecting to database")

	id, err := uuid.NewV7()
	assert.Nilf(t, err, "error generating UUID")
	retro := &types.Retrospective{
		ID:          id,
		Name:        "mtg",
		Description: "df/dx = 0",
		Questions:   []types.Question{},
	}

	sqlQuery := `INSERT INTO retrospectives (id, name, description) VALUES ($1, $2, $3)`
	_, err = db.conn.Exec(
		sqlQuery,
		&retro.ID,
		&retro.Name,
		&retro.Description,
	)

	assert.Nilf(t, err, "error creating retrospective")

	ctx := context.Background()
	res, err := db.DeleteRetrospective(ctx, retro.ID)
	assert.Nilf(t, err, "error deleting retrospective")
	assert.Equal(t, retro, res)

	res = &types.Retrospective{}
	sqlQuery = `SELECT id, name, description FROM retrospectives WHERE id = $1`
	err = db.conn.QueryRow(sqlQuery, retro.ID).Scan(
		&res.ID,
		&res.Name,
		&res.Description,
	)

	assert.Equal(t, sql.ErrNoRows, err)
}

func TestGetRetrospective(t *testing.T) {
	_, err := config.Load("../../config/config_test.yaml")
	assert.Nilf(t, err, "error loading config")

	db, err := NewSQLite()
	assert.Nilf(t, err, "error connecting to database")

	id, err := uuid.NewV7()
	assert.Nilf(t, err, "error generating UUID")

	questionID, err := uuid.NewV7()
	assert.Nilf(t, err, "error generating UUID")

	answerID, err := uuid.NewV7()
	assert.Nilf(t, err, "error generating UUID")

	retro := &types.Retrospective{
		ID:          id,
		Name:        "mtg",
		Description: "df/dx = 0",
		Questions: []types.Question{
			{
				ID:   questionID,
				Text: "what is the best mtg of the moment?",
				Answers: []types.Answer{
					{
						ID:         answerID,
						QuestionID: questionID,
						Text:       "Any of d(respect)/dx = 0 playlist 😎",
						Position:   1,
					},
				},
			},
		},
	}

	sqlQuery := `INSERT INTO retrospectives (id, name, description) VALUES ($1, $2, $3)`
	_, err = db.conn.Exec(
		sqlQuery,
		&retro.ID,
		&retro.Name,
		&retro.Description,
	)
	assert.Nilf(t, err, "error creating retrospective")

	sqlQuery = `INSERT INTO questions (id, text, retrospective_id) VALUES ($1, $2, $3)`
	_, err = db.conn.Exec(
		sqlQuery,
		&retro.Questions[0].ID,
		&retro.Questions[0].Text,
		&retro.ID,
	)
	assert.Nilf(t, err, "error creating question")

	sqlQuery = `INSERT INTO answers (id, text, question_id, position) VALUES ($1, $2, $3, $4)`
	_, err = db.conn.Exec(
		sqlQuery,
		&retro.Questions[0].Answers[0].ID,
		&retro.Questions[0].Answers[0].Text,
		&retro.Questions[0].Answers[0].QuestionID,
		&retro.Questions[0].Answers[0].Position,
	)
	assert.Nilf(t, err, "error creating answer")

	ctx := context.Background()
	res, err := db.GetRetrospective(ctx, retro.ID)
	assert.Nilf(t, err, "error getting retrospective")
	assert.Equal(t, retro, res)
}
