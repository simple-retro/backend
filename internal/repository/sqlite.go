package repository

import (
	"context"
	"database/sql"
	"os"

	"api/config"
	"api/types"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type SQLite struct {
	conn *sql.DB
}

func NewSQLite() (*SQLite, error) {
	conf := config.Get()
	db, err := sql.Open("sqlite3", conf.Database.Address)
	if err != nil {
		return nil, err
	}

	// Set the maximum number of open connections
	db.SetMaxOpenConns(conf.Database.MaxConn)

	// Ping to check if the database connection is established
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	repo := &SQLite{
		conn: db,
	}

	err = repo.migrate("database/schema.sql")
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (s *SQLite) migrate(filepath string) error {
	// Read the schema file
	schema, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	// Execute the SQL statements from the schema file
	_, err = s.conn.Exec(string(schema))
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLite) CreateRetrospective(ctx context.Context, retro *types.Retrospective) error {
	sql := `INSERT INTO retrospectives (id, name, description) VALUES ($1, $2, $3)`
	_, err := s.conn.Exec(sql,
		retro.ID,
		retro.Name,
		retro.Description,
	)
	return err
}

func (s *SQLite) UpdateRetrospective(ctx context.Context, retro *types.Retrospective) error {
	foundRetro := &types.Retrospective{
		ID: retro.ID,
	}

	sqlQuery := `SELECT name, description FROM retrospectives WHERE id = $1`
	err := s.conn.QueryRow(sqlQuery, foundRetro.ID).Scan(
		&foundRetro.Name,
		&foundRetro.Description,
	)
	if err != nil {
		return err
	}

	if len(retro.Name) == 0 {
		retro.Name = foundRetro.Name
	}

	if len(retro.Description) == 0 {
		retro.Description = foundRetro.Description
	}

	sqlQuery = `UPDATE retrospectives SET name = $1, description = $2 WHERE id = $3`
	_, err = s.conn.Exec(sqlQuery,
		retro.Name,
		retro.Description,
		retro.ID,
	)

	return err
}

func (s *SQLite) DeleteRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	retro := &types.Retrospective{
		ID: id,
	}

	sqlQuery := `SELECT name, description FROM retrospectives WHERE id = $1`
	err := s.conn.QueryRow(sqlQuery, id).Scan(
		&retro.Name,
		&retro.Description,
	)
	if err != nil {
		return nil, err
	}

	tx, err := s.conn.Begin()
	if err != nil {
		return retro, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	// Delete answers associated with questions of the retrospective
	sqlQuery = `DELETE FROM answers WHERE question_id IN (SELECT id FROM questions WHERE retrospective_id = $1)`
	_, err = tx.Exec(sqlQuery, id)
	if err != nil {
		return retro, err
	}

	// Delete questions associated with the retrospective
	sqlQuery = `DELETE FROM questions WHERE retrospective_id = $1`
	_, err = tx.Exec(sqlQuery, id)
	if err != nil {
		return retro, err
	}

	// Delete the retrospective
	sqlQuery = `DELETE FROM retrospectives WHERE id = $1`
	_, err = tx.Exec(sqlQuery, id)
	if err != nil {
		return retro, err
	}

	return retro, nil
}

func (s *SQLite) GetRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	retro := &types.Retrospective{
		ID: id,
	}

	sqlQuery := `SELECT name, description FROM retrospectives WHERE id = $1`
	err := s.conn.QueryRow(sqlQuery, id).Scan(
		&retro.Name,
		&retro.Description,
	)
	if err != nil {
		return nil, err
	}

	// Query questions for the retrospective
	sqlQuery = `SELECT id, text FROM questions WHERE retrospective_id = $1`
	rows, err := s.conn.Query(sqlQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var question types.Question
		err := rows.Scan(
			&question.ID,
			&question.Text,
		)
		if err != nil {
			return nil, err
		}

		// Query answers for the question
		sqlQuery = `SELECT id, text, position FROM answers WHERE question_id = $1`
		answerRows, err := s.conn.Query(sqlQuery, question.ID)
		if err != nil {
			return nil, err
		}
		defer answerRows.Close()

		// Loop through answers and append to the question
		for answerRows.Next() {
			var answer types.Answer
			err := answerRows.Scan(
				&answer.ID,
				&answer.Text,
				&answer.Position,
			)
			if err != nil {
				return nil, err
			}
			question.Answers = append(question.Answers, answer)
		}

		// Append the question to the retrospective
		retro.Questions = append(retro.Questions, question)
	}

	return retro, nil
}

func (s *SQLite) CreateQuestion(ctx context.Context, question *types.Question) error {
	return nil
}

func (s *SQLite) UpdateQuestion(ctx context.Context, question *types.Question) error {
	return nil
}

func (s *SQLite) DeleteQuestion(ctx context.Context, question *types.Question) error {
	return nil
}

func (s *SQLite) CreateAnswer(ctx context.Context, answer *types.Answer) error {
	return nil
}

func (s *SQLite) UpdateAnswer(ctx context.Context, answer *types.Answer) error {
	return nil
}

func (s *SQLite) DeleteAnswer(ctx context.Context, answer *types.Answer) error {
	return nil
}
