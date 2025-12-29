package main

import (
	"database/sql"
	"os"
)

type Database struct {
	// example: file:test.db?cache=shared&mode=memory | file:locked.sqlite?cache=shared
	DSN string
}

func NewDatabase() (*sql.DB, error) {
	os.Getenv("DATABASE_DSN")
	conn, err := sql.Open("sqlite3", "file:sus.db?")

	if err != nil {
		return nil, err
	}

	return conn, nil
}
