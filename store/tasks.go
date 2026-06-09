package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Task struct {
	ID           int64
	Title        string
	Status       string
	Priority     sql.NullInt64
	NotesPath    sql.NullString
	ParentID     sql.NullInt64
	RecurrenceID sql.NullInt64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Subtasks     []Task // populated by GetTask/ListTasks, not a DB column
}

var allowedTransitions = map[string][]string{
	"todo":        {"in_progress", "blocked", "archived"},
	"in_progress": {"blocked", "done"},
	"blocked":     {"in_progress", "todo"},
	"done":        {"archived"},
	"archived":    {},
}

func ValidTransition(from, to string) bool {
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func (s *Store) CreateTask(title string, priority int, parentID, recurrenceID sql.NullInt64) (*Task, error) {
	if parentID.Valid {
		// enforce one-level max
		var grandparent sql.NullInt64
		err := s.db.QueryRow(`SELECT parent_id FROM tasks WHERE id = ?`, parentID.Int64).Scan(&grandparent)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("parent task %d not found", parentID.Int64)
			}
			return nil, fmt.Errorf("check parent: %w", err)
		}
		if grandparent.Valid {
			return nil, errors.New("subtasks cannot have subtasks")
		}
	}
	var p interface{}
	if parentID.Valid {
		p = nil // subtasks have null priority
	} else {
		p = priority
	}
	res, err := s.db.Exec(
		`INSERT INTO tasks(title, status, priority, parent_id, recurrence_id) VALUES(?,?,?,?,?)`,
		title, "todo", p, nullableInt(parentID), nullableInt(recurrenceID),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetTask(id)
}

func (s *Store) GetTask(id int64) (*Task, error) {
	t, err := s.scanTask(s.db.QueryRow(
		`SELECT id,title,status,priority,notes_path,parent_id,recurrence_id,created_at,updated_at FROM tasks WHERE id=?`, id,
	))
	if err != nil {
		return nil, err
	}
	if !t.ParentID.Valid {
		t.Subtasks, err = s.listSubtasks(t.ID)
	}
	return t, err
}

func (s *Store) ListActiveTasks() ([]Task, error) {
	rows, err := s.db.Query(`
		SELECT id,title,status,priority,notes_path,parent_id,recurrence_id,created_at,updated_at
		FROM tasks
		WHERE parent_id IS NULL
		  AND status IN ('todo','in_progress','blocked')
		ORDER BY priority ASC, created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.collectTasksWithSubtasks(rows)
}

func (s *Store) ListArchivedTasks() ([]Task, error) {
	rows, err := s.db.Query(`
		SELECT id,title,status,priority,notes_path,parent_id,recurrence_id,created_at,updated_at
		FROM tasks
		WHERE parent_id IS NULL AND status = 'archived'
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.collectTasksWithSubtasks(rows)
}

func (s *Store) UpdateStatus(id int64, newStatus string) error {
	var current string
	var parentID sql.NullInt64
	if err := s.db.QueryRow(`SELECT status, parent_id FROM tasks WHERE id=?`, id).Scan(&current, &parentID); err != nil {
		return fmt.Errorf("get status: %w", err)
	}
	if parentID.Valid {
		return fmt.Errorf("use UpdateSubtaskStatus for subtasks")
	}
	if !ValidTransition(current, newStatus) {
		return fmt.Errorf("invalid transition %s→%s", current, newStatus)
	}
	_, err := s.db.Exec(
		`UPDATE tasks SET status=?, updated_at=datetime('now') WHERE id=?`,
		newStatus, id,
	)
	return err
}

func (s *Store) UpdateSubtaskStatus(id int64, done bool) error {
	status := "todo"
	if done {
		status = "done"
	}
	res, err := s.db.Exec(
		`UPDATE tasks SET status=?, updated_at=datetime('now') WHERE id=? AND parent_id IS NOT NULL`,
		status, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task %d not found or is not a subtask", id)
	}
	return nil
}

func (s *Store) UpdatePriority(id int64, priority int) error {
	res, err := s.db.Exec(
		`UPDATE tasks SET priority=?, updated_at=datetime('now') WHERE id=? AND parent_id IS NULL`,
		priority, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task %d not found or is a subtask", id)
	}
	return nil
}

func (s *Store) SetNotesPath(id int64, path string) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET notes_path=?, updated_at=datetime('now') WHERE id=?`,
		path, id,
	)
	return err
}

func (s *Store) PromoteSubtask(id int64) error {
	res, err := s.db.Exec(
		`UPDATE tasks SET parent_id=NULL, priority=2, updated_at=datetime('now') WHERE id=? AND parent_id IS NOT NULL`,
		id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task %d not found or is not a subtask", id)
	}
	return nil
}

func (s *Store) DeleteTask(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id=? OR parent_id=?`, id, id)
	return err
}

func (s *Store) PendingTaskCount() (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM tasks WHERE parent_id IS NULL AND status IN ('todo','in_progress')`,
	).Scan(&n)
	return n, err
}

// helpers

func parseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func (s *Store) scanTask(row *sql.Row) (*Task, error) {
	var t Task
	var createdStr, updatedStr string
	err := row.Scan(&t.ID, &t.Title, &t.Status, &t.Priority, &t.NotesPath,
		&t.ParentID, &t.RecurrenceID, &createdStr, &updatedStr)
	if err != nil {
		return nil, err
	}
	t.CreatedAt, err = parseTimestamp(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	t.UpdatedAt, err = parseTimestamp(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return &t, nil
}

func (s *Store) scanTaskRow(rows *sql.Rows) (*Task, error) {
	var t Task
	var createdStr, updatedStr string
	err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Priority, &t.NotesPath,
		&t.ParentID, &t.RecurrenceID, &createdStr, &updatedStr)
	if err != nil {
		return nil, err
	}
	t.CreatedAt, err = parseTimestamp(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	t.UpdatedAt, err = parseTimestamp(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return &t, nil
}

func (s *Store) listSubtasks(parentID int64) ([]Task, error) {
	rows, err := s.db.Query(
		`SELECT id,title,status,priority,notes_path,parent_id,recurrence_id,created_at,updated_at
		 FROM tasks WHERE parent_id=? ORDER BY created_at ASC`, parentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		t, err := s.scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Store) collectTasksWithSubtasks(rows *sql.Rows) ([]Task, error) {
	// Scan all parent rows first, then close, before issuing subtask queries.
	// This avoids a connection-pool deadlock when MaxOpenConns == 1.
	var tasks []*Task
	for rows.Next() {
		t, err := s.scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	out := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		var err error
		t.Subtasks, err = s.listSubtasks(t.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, nil
}

func nullableInt(n sql.NullInt64) interface{} {
	if n.Valid {
		return n.Int64
	}
	return nil
}
