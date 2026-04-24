package models

import (
	"database/sql"
	"log"
	_ "modernc.org/sqlite"
)

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	log.Printf("DB initialized: %s", path)
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS quiz_sets (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			title       TEXT NOT NULL,
			source_text TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS questions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			quiz_set_id INTEGER NOT NULL REFERENCES quiz_sets(id) ON DELETE CASCADE,
			type        TEXT NOT NULL CHECK(type IN ('multiple', 'fillblank')),
			question    TEXT NOT NULL,
			options     TEXT NOT NULL,
			correct     TEXT NOT NULL,
			explanation TEXT,
			position    INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS answers (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			quiz_set_id INTEGER NOT NULL,
			question_id INTEGER NOT NULL,
			chosen      TEXT NOT NULL,
			is_correct  INTEGER NOT NULL,
			answered_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}
