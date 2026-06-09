package store_test

import (
	"database/sql"
	"testing"
	"time"
)

func TestDueRecurrences(t *testing.T) {
	s := openTestStore(t)

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	r1, err := s.CreateRecurrence("GitHub admin", 1, "0 9 * * 1", past)
	if err != nil {
		t.Fatalf("create recurrence: %v", err)
	}
	_, err = s.CreateRecurrence("Weekly review", 2, "0 17 * * 5", future)
	if err != nil {
		t.Fatalf("create recurrence 2: %v", err)
	}

	due, err := s.DueRecurrences()
	if err != nil {
		t.Fatalf("due recurrences: %v", err)
	}
	if len(due) != 1 || due[0].ID != r1.ID {
		t.Errorf("expected 1 due recurrence (id=%d), got %v", r1.ID, due)
	}

	// create a task for it — should no longer appear as due
	_, err = s.CreateTask("GitHub admin", 1,
		sql.NullInt64{},
		sql.NullInt64{Valid: true, Int64: r1.ID},
	)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	due2, _ := s.DueRecurrences()
	if len(due2) != 0 {
		t.Error("recurrence should not be due when an active task exists")
	}
}
