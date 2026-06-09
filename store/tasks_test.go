package store_test

import (
	"database/sql"
	"testing"

	"github.com/glow-mdsol/dovin/store"
)

func TestValidTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"todo", "in_progress", true},
		{"todo", "done", false},
		{"in_progress", "done", true},
		{"in_progress", "todo", false},
		{"blocked", "in_progress", true},
		{"blocked", "archived", false},
		{"done", "archived", true},
		{"done", "todo", false},
		{"archived", "todo", false},
	}
	for _, c := range cases {
		got := store.ValidTransition(c.from, c.to)
		if got != c.want {
			t.Errorf("ValidTransition(%q→%q) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	s, err := store.Open()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetTask(t *testing.T) {
	s := openTestStore(t)
	task, err := s.CreateTask("write tests", 1, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if task.Title != "write tests" {
		t.Errorf("title = %q, want %q", task.Title, "write tests")
	}
	if task.Status != "todo" {
		t.Errorf("status = %q, want todo", task.Status)
	}

	got, err := s.GetTask(task.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != task.ID {
		t.Errorf("id mismatch")
	}
}

func TestUpdateStatus(t *testing.T) {
	s := openTestStore(t)
	task, _ := s.CreateTask("do thing", 2, sql.NullInt64{}, sql.NullInt64{})

	if err := s.UpdateStatus(task.ID, "in_progress"); err != nil {
		t.Fatalf("transition to in_progress: %v", err)
	}
	if err := s.UpdateStatus(task.ID, "todo"); err == nil {
		t.Fatal("expected error for in_progress→todo, got nil")
	}
	if err := s.UpdateStatus(task.ID, "done"); err != nil {
		t.Fatalf("transition to done: %v", err)
	}
}

func TestSubtaskOneLevel(t *testing.T) {
	s := openTestStore(t)
	parent, _ := s.CreateTask("parent", 2, sql.NullInt64{}, sql.NullInt64{})
	sub, err := s.CreateTask("sub", 0, sql.NullInt64{Valid: true, Int64: parent.ID}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create subtask: %v", err)
	}

	// should reject a sub-subtask
	_, err = s.CreateTask("subsub", 0, sql.NullInt64{Valid: true, Int64: sub.ID}, sql.NullInt64{})
	if err == nil {
		t.Fatal("expected error creating sub-subtask, got nil")
	}
}

func TestPromoteSubtask(t *testing.T) {
	s := openTestStore(t)
	parent, _ := s.CreateTask("parent", 2, sql.NullInt64{}, sql.NullInt64{})
	sub, _ := s.CreateTask("sub", 0, sql.NullInt64{Valid: true, Int64: parent.ID}, sql.NullInt64{})

	if err := s.PromoteSubtask(sub.ID); err != nil {
		t.Fatalf("promote: %v", err)
	}
	got, _ := s.GetTask(sub.ID)
	if got.ParentID.Valid {
		t.Error("parent_id should be null after promotion")
	}
	if !got.Priority.Valid || got.Priority.Int64 != 2 {
		t.Errorf("priority after promotion = %v, want 2", got.Priority)
	}
}
