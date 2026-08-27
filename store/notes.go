package store

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Note represents a stored note, optionally linked to a task.
type Note struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	TaskID     *int64    `json:"task_id"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

// ListNotes returns all notes ordered by creation date descending.
func (s *Store) ListNotes() ([]*Note, error) {
	rows, err := s.db.Query(
		`SELECT id, task_id, title, content, created_at, modified_at
		 FROM notes ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*Note
	for rows.Next() {
		n := &Note{}
		var createdStr, modifiedStr string
		if err := rows.Scan(&n.ID, &n.TaskID, &n.Title, &n.Content, &createdStr, &modifiedStr); err != nil {
			return nil, err
		}
		n.CreatedAt, _ = parseTimestamp(createdStr)
		n.ModifiedAt, _ = parseTimestamp(modifiedStr)
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// GetNote returns a single note by ID.
func (s *Store) GetNote(id int64) (*Note, error) {
	n := &Note{}
	var createdStr, modifiedStr string
	err := s.db.QueryRow(
		`SELECT id, task_id, title, content, created_at, modified_at FROM notes WHERE id = ?`, id,
	).Scan(&n.ID, &n.TaskID, &n.Title, &n.Content, &createdStr, &modifiedStr)
	if err != nil {
		return nil, fmt.Errorf("note not found: %d", id)
	}
	n.CreatedAt, _ = parseTimestamp(createdStr)
	n.ModifiedAt, _ = parseTimestamp(modifiedStr)
	return n, nil
}

// CreateNote creates a standalone note (not linked to a task).
func (s *Store) CreateNote(title string) (*Note, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO notes (title, content, task_id, created_at, modified_at)
		 VALUES (?, ?, NULL, ?, ?)`,
		title, "# "+title+"\n\n", now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetNote(id)
}

// AddNoteToTask creates a note linked to the given task ID.
// Pass taskID=0 for a standalone note.
func (s *Store) AddNoteToTask(taskID int64, title string) (*Note, error) {
	now := time.Now().UTC()
	var tid interface{}
	if taskID > 0 {
		tid = taskID
	}
	res, err := s.db.Exec(
		`INSERT INTO notes (title, content, task_id, created_at, modified_at)
		 VALUES (?, ?, ?, ?, ?)`,
		title, "# "+title+"\n\n", tid, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetNote(id)
}

// SaveNote updates the content of an existing note.
func (s *Store) SaveNote(id int64, content string) error {
	res, err := s.db.Exec(
		`UPDATE notes SET content = ?, modified_at = ? WHERE id = ?`,
		content, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("note not found: %d", id)
	}
	return nil
}

// DeleteNote removes a note from the database.
func (s *Store) DeleteNote(id int64) error {
	_, err := s.db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}

// GetNotesForTask returns all notes associated with the given task.
func (s *Store) GetNotesForTask(taskID int64) ([]*Note, error) {
	rows, err := s.db.Query(
		`SELECT id, task_id, title, content, created_at, modified_at
		 FROM notes WHERE task_id = ? ORDER BY created_at DESC`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*Note
	for rows.Next() {
		n := &Note{}
		var createdStr, modifiedStr string
		if err := rows.Scan(&n.ID, &n.TaskID, &n.Title, &n.Content, &createdStr, &modifiedStr); err != nil {
			return nil, err
		}
		n.CreatedAt, _ = parseTimestamp(createdStr)
		n.ModifiedAt, _ = parseTimestamp(modifiedStr)
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// GetOrCreateNoteForTask returns the most recent note linked to taskID, or
// creates a new one (titled after the task) if none exists.
func (s *Store) GetOrCreateNoteForTask(taskID int64, taskTitle string) (*Note, error) {
	notes, err := s.GetNotesForTask(taskID)
	if err != nil {
		return nil, err
	}
	if len(notes) > 0 {
		return notes[0], nil
	}
	return s.AddNoteToTask(taskID, taskTitle)
}

// --- aliases used by api/notes.go ---

// ListNotesAPI is an alias for ListNotes for API use.
func (s *Store) ListNotesAPI() ([]*Note, error) { return s.ListNotes() }

// GetNoteById is an alias for GetNote for API use.
func (s *Store) GetNoteById(id int64) (*Note, error) { return s.GetNote(id) }
