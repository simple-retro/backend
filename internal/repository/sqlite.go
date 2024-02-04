package repository

import (
	"api/config"
	"api/types"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SQLite struct {
	conn *sql.DB
}

func New() (*SQLite, error) {
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
	return &SQLite{
		conn: db,
	}, nil
}

func (s *SQLite) CreateRetrospective(retro *types.Retrospective) error {
	return nil
}
