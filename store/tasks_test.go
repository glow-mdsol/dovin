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
	task, err := s.CreateTask("do thing", 2, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

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
	parent, err := s.CreateTask("parent", 2, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
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
	parent, err := s.CreateTask("parent", 2, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	sub, err := s.CreateTask("sub", 0, sql.NullInt64{Valid: true, Int64: parent.ID}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create sub: %v", err)
	}

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

func TestUpdateSubtaskStatus(t *testing.T) {
	s := openTestStore(t)
	parent, err := s.CreateTask("parent", 2, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	sub, err := s.CreateTask("sub", 0, sql.NullInt64{Valid: true, Int64: parent.ID}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create sub: %v", err)
	}

	if err := s.UpdateSubtaskStatus(sub.ID, true); err != nil {
		t.Fatalf("mark done: %v", err)
	}
	got, _ := s.GetTask(sub.ID)
	if got.Status != "done" {
		t.Errorf("status = %q, want done", got.Status)
	}

	// should fail on top-level task
	if err := s.UpdateSubtaskStatus(parent.ID, true); err == nil {
		t.Error("expected error calling UpdateSubtaskStatus on top-level task")
	}
}

func TestDeleteTask(t *testing.T) {
	s := openTestStore(t)
	parent, err := s.CreateTask("parent", 2, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	_, err = s.CreateTask("sub", 0, sql.NullInt64{Valid: true, Int64: parent.ID}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create sub: %v", err)
	}

	if err := s.DeleteTask(parent.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// parent should be gone
	if _, err := s.GetTask(parent.ID); err == nil {
		t.Error("expected error getting deleted task")
	}
}

func TestListActiveTasks(t *testing.T) {
	s := openTestStore(t)
	// high priority
	_, err := s.CreateTask("high prio", 1, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create high prio: %v", err)
	}
	// low priority
	_, err = s.CreateTask("low prio", 3, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create low prio: %v", err)
	}

	tasks, err := s.ListActiveTasks()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("want 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Priority.Int64 != 1 {
		t.Errorf("first task priority = %d, want 1 (high)", tasks[0].Priority.Int64)
	}
}

func TestPendingTaskCount(t *testing.T) {
	s := openTestStore(t)
	t1, err := s.CreateTask("one", 2, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create one: %v", err)
	}
	_, err = s.CreateTask("two", 2, sql.NullInt64{}, sql.NullInt64{})
	if err != nil {
		t.Fatalf("create two: %v", err)
	}
	if err := s.UpdateStatus(t1.ID, "in_progress"); err != nil {
		t.Fatalf("update status: %v", err)
	}

	n, err := s.PendingTaskCount()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
}
