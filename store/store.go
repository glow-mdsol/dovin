package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open() (*Store, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "dovin")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	dbPath := filepath.Join(dir, "dovin.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS recurrences (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			title             TEXT NOT NULL,
			priority          INTEGER NOT NULL DEFAULT 2,
			schedule          TEXT NOT NULL,
			next_due_at       DATETIME,
			last_completed_at DATETIME,
			active            BOOLEAN NOT NULL DEFAULT 1
		);

		CREATE TABLE IF NOT EXISTS tasks (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			title          TEXT NOT NULL,
			status         TEXT NOT NULL DEFAULT 'todo',
			priority       INTEGER,
			notes_path     TEXT,
			parent_id      INTEGER REFERENCES tasks(id),
			recurrence_id  INTEGER REFERENCES recurrences(id),
			created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at     DATETIME NOT NULL DEFAULT (datetime('now'))
		);

		CREATE INDEX IF NOT EXISTS idx_tasks_parent   ON tasks(parent_id);
		CREATE INDEX IF NOT EXISTS idx_tasks_status   ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_recur    ON tasks(recurrence_id);
	`)
	return err
}
