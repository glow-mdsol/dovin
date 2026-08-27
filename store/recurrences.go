package store

import (
	"database/sql"
	"time"
)

type Recurrence struct {
	ID              int64
	Title           string
	Priority        int
	Schedule        string
	NextDueAt       sql.NullTime
	LastCompletedAt sql.NullTime
	Active          bool
}

func (s *Store) CreateRecurrence(title string, priority int, schedule string, firstDue time.Time) (*Recurrence, error) {
	res, err := s.db.Exec(
		`INSERT INTO recurrences(title,priority,schedule,next_due_at) VALUES(?,?,?,?)`,
		title, priority, schedule, firstDue.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetRecurrence(id)
}

func (s *Store) GetRecurrence(id int64) (*Recurrence, error) {
	row := s.db.QueryRow(
		`SELECT id,title,priority,schedule,next_due_at,last_completed_at,active FROM recurrences WHERE id=?`, id,
	)
	return s.scanRecurrence(row)
}

func (s *Store) ListRecurrences() ([]Recurrence, error) {
	rows, err := s.db.Query(
		`SELECT id,title,priority,schedule,next_due_at,last_completed_at,active FROM recurrences ORDER BY title`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Recurrence
	for rows.Next() {
		r, err := s.scanRecurrenceRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// DueRecurrences returns active recurrences with next_due_at <= now
// that have no current todo/in_progress task instance.
func (s *Store) DueRecurrences() ([]Recurrence, error) {
	rows, err := s.db.Query(`
		SELECT r.id,r.title,r.priority,r.schedule,r.next_due_at,r.last_completed_at,r.active
		FROM recurrences r
		WHERE r.active = 1
		  AND r.next_due_at <= datetime('now')
		  AND NOT EXISTS (
		      SELECT 1 FROM tasks t
		      WHERE t.recurrence_id = r.id
		        AND t.status IN ('todo','in_progress')
		  )
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Recurrence
	for rows.Next() {
		r, err := s.scanRecurrenceRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (s *Store) MarkRecurrenceCompleted(id int64, nextDue time.Time) error {
	_, err := s.db.Exec(
		`UPDATE recurrences SET last_completed_at=datetime('now'), next_due_at=? WHERE id=?`,
		nextDue.UTC().Format("2006-01-02 15:04:05"), id,
	)
	return err
}

func (s *Store) SetRecurrenceActive(id int64, active bool) error {
	_, err := s.db.Exec(`UPDATE recurrences SET active=? WHERE id=?`, active, id)
	return err
}

func (s *Store) DeleteRecurrence(id int64) error {
	_, err := s.db.Exec(`UPDATE tasks SET recurrence_id=NULL WHERE recurrence_id=?`, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM recurrences WHERE id=?`, id)
	return err
}

func (s *Store) scanRecurrence(row *sql.Row) (*Recurrence, error) {
	var r Recurrence
	var nextDue, lastComp sql.NullString
	err := row.Scan(&r.ID, &r.Title, &r.Priority, &r.Schedule, &nextDue, &lastComp, &r.Active)
	if err != nil {
		return nil, err
	}
	if nextDue.Valid {
		t, err := parseTimestamp(nextDue.String)
		if err != nil {
			return nil, err
		}
		r.NextDueAt = sql.NullTime{Time: t, Valid: true}
	}
	if lastComp.Valid {
		t, err := parseTimestamp(lastComp.String)
		if err != nil {
			return nil, err
		}
		r.LastCompletedAt = sql.NullTime{Time: t, Valid: true}
	}
	return &r, nil
}

func (s *Store) scanRecurrenceRow(rows *sql.Rows) (*Recurrence, error) {
	var r Recurrence
	var nextDue, lastComp sql.NullString
	err := rows.Scan(&r.ID, &r.Title, &r.Priority, &r.Schedule, &nextDue, &lastComp, &r.Active)
	if err != nil {
		return nil, err
	}
	if nextDue.Valid {
		t, err := parseTimestamp(nextDue.String)
		if err != nil {
			return nil, err
		}
		r.NextDueAt = sql.NullTime{Time: t, Valid: true}
	}
	if lastComp.Valid {
		t, err := parseTimestamp(lastComp.String)
		if err != nil {
			return nil, err
		}
		r.LastCompletedAt = sql.NullTime{Time: t, Valid: true}
	}
	return &r, nil
}
