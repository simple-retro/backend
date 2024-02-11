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

func createGenericRetrospective(db *SQLite) (*types.Retrospective, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
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
	return retro, err
}

func createGenericQuestion(db *SQLite, retro *types.Retrospective) (*types.Question, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	question := &types.Question{
		ID:   id,
		Text: "Do you like japanese peanout?",
	}

	sqlQuery := `INSERT INTO questions (id, text, retrospective_id) VALUES ($1, $2, $3)`
	_, err = db.conn.Exec(
		sqlQuery,
		&question.ID,
		&question.Text,
		&retro.ID,
	)
	return question, err
}

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

	retro, err := createGenericRetrospective(db)
	assert.Nilf(t, err, "error creating retrospective")

	retro.Name = "Changed name"
	retro.Description = "Changed description"

	ctx := context.Background()
	err = db.UpdateRetrospective(ctx, retro)
	assert.Nilf(t, err, "error updating retrospective")

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

func TestDeleteRetrospective(t *testing.T) {
	_, err := config.Load("../../config/config_test.yaml")
	assert.Nilf(t, err, "error loading config")

	db, err := NewSQLite()
	assert.Nilf(t, err, "error connecting to database")

	retro, err := createGenericRetrospective(db)
	retro.Questions = []types.Question{}
	assert.Nilf(t, err, "error creating retrospective")

	ctx := context.Background()
	res, err := db.DeleteRetrospective(ctx, retro.ID)
	assert.Nilf(t, err, "error deleting retrospective")
	assert.Equal(t, retro, res)

	res = &types.Retrospective{}
	sqlQuery := `SELECT id, name, description FROM retrospectives WHERE id = $1`
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

func TestGetAllRetrospective(t *testing.T) {
	_, err := config.Load("../../config/config_test.yaml")
	assert.Nilf(t, err, "error loading config")

	db, err := NewSQLite()
	assert.Nilf(t, err, "error connecting to database")

	id, err := uuid.NewV7()
	assert.Nilf(t, err, "error generating UUID")

	id2, err := uuid.NewV7()
	assert.Nilf(t, err, "error generating UUID")

	ids := []uuid.UUID{id, id2}
	retros := []types.Retrospective{
		{
			ID:          id,
			Name:        "mtg",
			Description: "df/dx = 0",
		},
		{
			ID:          id2,
			Name:        "Tututu",
			Description: "Boom boom boom",
		},
	}

	// Clear all database to avoid problems with previous tests
	sqlQuery := `DELETE FROM answers`
	_, err = db.conn.Exec(sqlQuery)
	assert.Nilf(t, err, "error deleting all answers")

	sqlQuery = `DELETE FROM questions`
	_, err = db.conn.Exec(sqlQuery)
	assert.Nilf(t, err, "error deleting all questions")

	sqlQuery = `DELETE FROM retrospectives`
	_, err = db.conn.Exec(sqlQuery)
	assert.Nilf(t, err, "error deleting all retrospectives")

	sqlQuery = `INSERT INTO retrospectives (id, name, description) VALUES ($1, $2, $3), ($4, $5, $6)`
	_, err = db.conn.Exec(
		sqlQuery,
		&retros[0].ID,
		&retros[0].Name,
		&retros[0].Description,
		&retros[1].ID,
		&retros[1].Name,
		&retros[1].Description,
	)
	assert.Nilf(t, err, "error creating retrospective")

	ctx := context.Background()
	res, err := db.GetAllRetrospectives(ctx)
	assert.Nilf(t, err, "error getting all retrospectives")

	assert.Equal(t, ids, res)
}

func TestCreateQuestion(t *testing.T) {
	_, err := config.Load("../../config/config_test.yaml")
	assert.Nilf(t, err, "error loading config")

	db, err := NewSQLite()
	assert.Nilf(t, err, "error connecting to database")

	retro, err := createGenericRetrospective(db)
	assert.Nilf(t, err, "error creating retrospective")

	id, err := uuid.NewV7()
	assert.Nilf(t, err, "error generating UUID")
	question := &types.Question{
		ID:   id,
		Text: "Do you like japanese peanout?",
	}
	ctx := context.WithValue(context.Background(), "retrospective_id", retro.ID)

	err = db.CreateQuestion(ctx, question)
	assert.Nilf(t, err, "error creating question")

	res := &types.Question{}
	sqlQuery := `SELECT id, text FROM questions WHERE id = $1`
	err = db.conn.QueryRow(sqlQuery, id).Scan(
		&res.ID,
		&res.Text,
	)

	assert.Nilf(t, err, "error getting created question")
	assert.Equal(t, question, res)
}

func TestUpdateQuestion(t *testing.T) {
	_, err := config.Load("../../config/config_test.yaml")
	assert.Nilf(t, err, "error loading config")

	db, err := NewSQLite()
	assert.Nilf(t, err, "error connecting to database")

	retro, err := createGenericRetrospective(db)
	assert.Nilf(t, err, "error creating retrospective")

	question, err := createGenericQuestion(db, retro)
	assert.Nilf(t, err, "error creating question")

	question.Text = "Do you drink coffe with cinnamon?"
	ctx := context.WithValue(context.Background(), "retrospective_id", retro.ID)
	err = db.UpdateQuestion(ctx, question)
	assert.Nilf(t, err, "error updating question")

	res := &types.Question{}
	var resRetroID uuid.UUID

	sqlQuery := `SELECT id, text, retrospective_id  FROM questions WHERE id = $1`
	err = db.conn.QueryRow(sqlQuery, question.ID).Scan(
		&res.ID,
		&res.Text,
		&resRetroID,
	)

	assert.Nilf(t, err, "error getting created question")
	assert.Equal(t, question, res)
	assert.Equal(t, retro.ID, resRetroID)
}

func TestDeleteQuestion(t *testing.T) {
	_, err := config.Load("../../config/config_test.yaml")
	assert.Nilf(t, err, "error loading config")

	db, err := NewSQLite()
	assert.Nilf(t, err, "error connecting to database")

	retro, err := createGenericRetrospective(db)
	assert.Nilf(t, err, "error creating retrospective")

	question, err := createGenericQuestion(db, retro)
	assert.Nilf(t, err, "error creating question")
	question.Answers = []types.Answer{}

	ctx := context.WithValue(context.Background(), "retrospective_id", retro.ID)

	res, err := db.DeleteQuestion(ctx, question.ID)
	assert.Nilf(t, err, "error deleting question")
	assert.Equal(t, question, res)

	res = &types.Question{}
	sqlQuery := `SELECT id, text FROM questions WHERE id = $1`
	err = db.conn.QueryRow(sqlQuery, question.ID).Scan(
		&res.ID,
		&res.Text,
	)

	assert.Equal(t, sql.ErrNoRows, err)
}
