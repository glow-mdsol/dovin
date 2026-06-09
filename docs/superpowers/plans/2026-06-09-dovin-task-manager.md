# Dovin Task Manager Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a macOS menu bar task manager in Go with a webview popup UI, SQLite storage, recurring tasks, subtasks, priority sorting, and launchd startup.

**Architecture:** A Go binary registers a systray icon and starts a loopback HTTP server. Clicking the icon opens a webview popup that loads the local server's single-page UI. A background goroutine ticks every 60s to promote due recurring tasks to `todo`. All state lives in SQLite at `~/.config/dovin/dovin.db`.

**Tech Stack:** Go 1.22+, `github.com/fyne-io/systray`, `github.com/webview/webview_go`, `modernc.org/sqlite`, `github.com/robfig/cron/v3`, stdlib `net/http`, `embed`

---

### Task 1: Project scaffold

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `main.go` (stub)
- Create: `assets/icon.png` (placeholder)

- [ ] **Step 1: Initialise the module**

```bash
cd /Users/GLW1/Documents/Devel/glow-mdsol/dovin
go mod init github.com/glow-mdsol/dovin
```

Expected: `go.mod` created with `module github.com/glow-mdsol/dovin` and `go 1.22` (or current version).

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/fyne-io/systray@latest
go get github.com/webview/webview_go@latest
go get modernc.org/sqlite@latest
go get github.com/robfig/cron/v3@latest
```

- [ ] **Step 3: Create directory structure**

```bash
mkdir -p store api scheduler notes ui/static assets
```

- [ ] **Step 4: Create stub main.go**

```go
package main

func main() {
}
```

- [ ] **Step 5: Create a 16×16 placeholder icon**

Download or create a minimal PNG at `assets/icon.png`. Any 16×16 PNG works for now — replace it later.

```bash
# Quick placeholder using Go's image package — run this once:
cat > /tmp/mkicon.go << 'EOF'
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{80, 120, 255, 255})
		}
	}
	f, _ := os.Create("assets/icon.png")
	defer f.Close()
	png.Encode(f, img)
}
EOF
go run /tmp/mkicon.go
```

- [ ] **Step 6: Create Makefile**

```makefile
BIN := bin/dovin
BINARY_NAME := dovin
INSTALL_DIR := $(HOME)/.local/bin
PLIST_NAME := com.glow.dovin
PLIST_PATH := $(HOME)/Library/LaunchAgents/$(PLIST_NAME).plist
LOG_PATH := $(HOME)/Library/Logs/dovin.log

.PHONY: build test install uninstall

build:
	mkdir -p bin
	go build -o $(BIN) .

test:
	go test ./...

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BIN) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Writing launchd plist to $(PLIST_PATH)"
	@cat > $(PLIST_PATH) << 'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.glow.dovin</string>
    <key>ProgramArguments</key>
    <array>
        <string>$(INSTALL_DIR)/$(BINARY_NAME)</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$(LOG_PATH)</string>
    <key>StandardErrorPath</key>
    <string>$(LOG_PATH)</string>
</dict>
</plist>
PLIST
	launchctl load $(PLIST_PATH)
	@echo "Dovin installed and running."

uninstall:
	-launchctl unload $(PLIST_PATH) 2>/dev/null
	-rm -f $(PLIST_PATH)
	-rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Dovin uninstalled."
```

- [ ] **Step 7: Verify it compiles**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum Makefile main.go assets/icon.png
git commit -m "feat: project scaffold, go module, Makefile"
```

---

### Task 2: Store — schema & migrations

**Files:**
- Create: `store/store.go`
- Create: `store/config.go`

- [ ] **Step 1: Write store.go with Open() and schema**

```go
// store/store.go
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
```

- [ ] **Step 2: Write config.go**

```go
// store/config.go
package store

func (s *Store) ConfigGet(key string) (string, error) {
	var val string
	err := s.db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&val)
	return val, err
}

func (s *Store) ConfigSet(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO config(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add store/
git commit -m "feat: store open, schema migrations, config table"
```

---

### Task 3: Store — task CRUD

**Files:**
- Create: `store/tasks.go`
- Create: `store/tasks_test.go`

- [ ] **Step 1: Write the Task type and allowed transitions map**

```go
// store/tasks.go
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
```

- [ ] **Step 2: Write failing tests for transitions**

```go
// store/tasks_test.go
package store_test

import (
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
```

- [ ] **Step 3: Run failing tests**

```bash
go test ./store/... -run TestValidTransition -v
```

Expected: FAIL (ValidTransition not yet defined).

- [ ] **Step 4: Add CRUD methods to tasks.go**

Append to `store/tasks.go`:

```go
func (s *Store) CreateTask(title string, priority int, parentID, recurrenceID sql.NullInt64) (*Task, error) {
	if parentID.Valid {
		// enforce one-level max
		var grandparent sql.NullInt64
		err := s.db.QueryRow(`SELECT parent_id FROM tasks WHERE id = ?`, parentID.Int64).Scan(&grandparent)
		if err != nil {
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
	if err := s.db.QueryRow(`SELECT status FROM tasks WHERE id=?`, id).Scan(&current); err != nil {
		return fmt.Errorf("get status: %w", err)
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
	_, err := s.db.Exec(
		`UPDATE tasks SET status=?, updated_at=datetime('now') WHERE id=? AND parent_id IS NOT NULL`,
		status, id,
	)
	return err
}

func (s *Store) UpdatePriority(id int64, priority int) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET priority=?, updated_at=datetime('now') WHERE id=? AND parent_id IS NULL`,
		priority, id,
	)
	return err
}

func (s *Store) SetNotesPath(id int64, path string) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET notes_path=?, updated_at=datetime('now') WHERE id=?`,
		path, id,
	)
	return err
}

func (s *Store) PromoteSubtask(id int64) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET parent_id=NULL, priority=2, updated_at=datetime('now') WHERE id=? AND parent_id IS NOT NULL`,
		id,
	)
	return err
}

func (s *Store) DeleteTask(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id=? OR parent_id=?`, id, id)
	return err
}

func (s *Store) ActiveTaskCount() (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM tasks WHERE parent_id IS NULL AND status IN ('todo','in_progress')`,
	).Scan(&n)
	return n, err
}

// helpers

func (s *Store) scanTask(row *sql.Row) (*Task, error) {
	var t Task
	var createdStr, updatedStr string
	err := row.Scan(&t.ID, &t.Title, &t.Status, &t.Priority, &t.NotesPath,
		&t.ParentID, &t.RecurrenceID, &createdStr, &updatedStr)
	if err != nil {
		return nil, err
	}
	t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
	t.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)
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
	t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
	t.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)
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
	var out []Task
	for rows.Next() {
		t, err := s.scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		t.Subtasks, err = s.listSubtasks(t.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func nullableInt(n sql.NullInt64) interface{} {
	if n.Valid {
		return n.Int64
	}
	return nil
}
```

- [ ] **Step 5: Add integration test for task CRUD using a temp DB**

Append to `store/tasks_test.go`:

```go
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
```

- [ ] **Step 6: Run tests**

```bash
go test ./store/... -v
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add store/
git commit -m "feat: store task CRUD, status transitions, subtasks, promote"
```

---

### Task 4: Store — recurrences

**Files:**
- Create: `store/recurrences.go`
- Create: `store/recurrences_test.go`

- [ ] **Step 1: Write recurrences.go**

```go
// store/recurrences.go
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

func (s *Store) scanRecurrence(row *sql.Row) (*Recurrence, error) {
	var r Recurrence
	var nextDue, lastComp sql.NullString
	err := row.Scan(&r.ID, &r.Title, &r.Priority, &r.Schedule, &nextDue, &lastComp, &r.Active)
	if err != nil {
		return nil, err
	}
	if nextDue.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", nextDue.String)
		r.NextDueAt = sql.NullTime{Time: t, Valid: true}
	}
	if lastComp.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastComp.String)
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
		t, _ := time.Parse("2006-01-02 15:04:05", nextDue.String)
		r.NextDueAt = sql.NullTime{Time: t, Valid: true}
	}
	if lastComp.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", lastComp.String)
		r.LastCompletedAt = sql.NullTime{Time: t, Valid: true}
	}
	return &r, nil
}
```

- [ ] **Step 2: Write recurrences test**

```go
// store/recurrences_test.go
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
```

- [ ] **Step 3: Run tests**

```bash
go test ./store/... -v
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add store/recurrences.go store/recurrences_test.go
git commit -m "feat: store recurrences CRUD and due-recurrence query"
```

---

### Task 5: Scheduler

**Files:**
- Create: `scheduler/scheduler.go`
- Create: `scheduler/scheduler_test.go`

- [ ] **Step 1: Write scheduler.go**

```go
// scheduler/scheduler.go
package scheduler

import (
	"database/sql"
	"log"
	"time"

	"github.com/glow-mdsol/dovin/store"
	"github.com/robfig/cron/v3"
)

// NextAfter returns the next time a cron schedule fires after t.
func NextAfter(schedule string, after time.Time) (time.Time, error) {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	s, err := p.Parse(schedule)
	if err != nil {
		return time.Time{}, err
	}
	return s.Next(after), nil
}

// Run starts the scheduler loop. It ticks every 60 seconds and promotes
// due recurrences to todo tasks. Call with go Run(store, done).
func Run(s *store.Store, done <-chan struct{}) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	tick(s) // run immediately on start
	for {
		select {
		case <-ticker.C:
			tick(s)
		case <-done:
			return
		}
	}
}

func tick(s *store.Store) {
	due, err := s.DueRecurrences()
	if err != nil {
		log.Printf("scheduler: list due recurrences: %v", err)
		return
	}
	for _, r := range due {
		_, err := s.CreateTask(r.Title, r.Priority,
			sql.NullInt64{},
			sql.NullInt64{Valid: true, Int64: r.ID},
		)
		if err != nil {
			log.Printf("scheduler: create task for recurrence %d: %v", r.ID, err)
			continue
		}
		next, err := NextAfter(r.Schedule, time.Now())
		if err != nil {
			log.Printf("scheduler: parse schedule %q: %v", r.Schedule, err)
			continue
		}
		if err := s.MarkRecurrenceCompleted(r.ID, next); err != nil {
			log.Printf("scheduler: mark recurrence %d: %v", r.ID, err)
		}
	}
}
```

- [ ] **Step 2: Write scheduler tests**

```go
// scheduler/scheduler_test.go
package scheduler_test

import (
	"testing"
	"time"

	"github.com/glow-mdsol/dovin/scheduler"
)

func TestNextAfter(t *testing.T) {
	// "0 9 * * 1" = every Monday at 09:00
	// Use a known Monday + advance to find next Monday
	monday := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC) // a Monday
	next, err := scheduler.NextAfter("0 9 * * 1", monday)
	if err != nil {
		t.Fatalf("NextAfter: %v", err)
	}
	// next Monday 09:00 after midnight Monday 8 June = same day at 09:00
	want := time.Date(2026, 6, 8, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}

	// after 09:00 on Monday, next should be following Monday
	next2, _ := scheduler.NextAfter("0 9 * * 1", time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC))
	want2 := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	if !next2.Equal(want2) {
		t.Errorf("next2 = %v, want %v", next2, want2)
	}
}

func TestInvalidSchedule(t *testing.T) {
	_, err := scheduler.NextAfter("not-a-cron", time.Now())
	if err == nil {
		t.Fatal("expected error for invalid cron, got nil")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./scheduler/... -v
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add scheduler/
git commit -m "feat: scheduler tick loop and cron NextAfter helper"
```

---

### Task 6: Notes package

**Files:**
- Create: `notes/notes.go`

- [ ] **Step 1: Write notes.go**

```go
// notes/notes.go
package notes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Dir returns the notes directory, creating it if needed.
func Dir() (string, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "dovin", "notes")
	return dir, os.MkdirAll(dir, 0755)
}

// Filename returns a deterministic filename for a task.
func Filename(id int64, title string) string {
	slug := slugify(title)
	return fmt.Sprintf("%d-%s.md", id, slug)
}

// EnsureFile creates the notes file if it does not exist, then opens it.
// Returns the relative filename (for storage in notes_path).
func EnsureFile(id int64, title string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	name := Filename(id, title)
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte("# "+title+"\n\n"), 0644); err != nil {
			return "", fmt.Errorf("create notes file: %w", err)
		}
	}
	if err := open(path); err != nil {
		return "", fmt.Errorf("open notes file: %w", err)
	}
	return name, nil
}

func open(path string) error {
	editor := os.Getenv("EDITOR")
	if editor != "" {
		return exec.Command(editor, path).Start()
	}
	return exec.Command("open", path).Start()
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
```

- [ ] **Step 2: Commit**

```bash
git add notes/
git commit -m "feat: notes file creation and open via \$EDITOR"
```

---

### Task 7: API — server setup and task handlers

**Files:**
- Create: `api/server.go`
- Create: `api/tasks.go`

- [ ] **Step 1: Write api/response.go — flat JSON types**

`store.Task` uses `sql.NullInt64` / `sql.NullString` which serialize as `{"Int64":2,"Valid":true}`. The frontend needs plain values. Define flat response structs here and use them everywhere in the API.

```go
// api/response.go
package api

import "github.com/glow-mdsol/dovin/store"

type taskResp struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	Priority     *int64     `json:"priority"`
	NotesPath    *string    `json:"notes_path"`
	ParentID     *int64     `json:"parent_id"`
	RecurrenceID *int64     `json:"recurrence_id"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
	Subtasks     []taskResp `json:"subtasks"`
}

func toResp(t *store.Task) taskResp {
	r := taskResp{
		ID:        t.ID,
		Title:     t.Title,
		Status:    t.Status,
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if t.Priority.Valid     { r.Priority = &t.Priority.Int64 }
	if t.NotesPath.Valid    { r.NotesPath = &t.NotesPath.String }
	if t.ParentID.Valid     { r.ParentID = &t.ParentID.Int64 }
	if t.RecurrenceID.Valid { r.RecurrenceID = &t.RecurrenceID.Int64 }
	for _, s := range t.Subtasks {
		sc := s
		r.Subtasks = append(r.Subtasks, toResp(&sc))
	}
	return r
}

func toRespSlice(tasks []store.Task) []taskResp {
	out := make([]taskResp, len(tasks))
	for i := range tasks {
		out[i] = toResp(&tasks[i])
	}
	return out
}
```

Everywhere `writeJSON(w, ..., task)` or `writeJSON(w, ..., tasks)` is used in `api/tasks.go`, replace with `writeJSON(w, ..., toResp(task))` or `writeJSON(w, ..., toRespSlice(tasks))`.

- [ ] **Step 2: Write server.go**

```go
// api/server.go
package api

import (
	"net"
	"net/http"

	"github.com/glow-mdsol/dovin/store"
)

type Server struct {
	store    *store.Store
	listener net.Listener
	port     int
}

func New(s *store.Store) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	srv := &Server{
		store:    s,
		listener: ln,
		port:     ln.Addr().(*net.TCPAddr).Port,
	}
	return srv, nil
}

func (s *Server) Port() int { return s.port }

func (s *Server) Handler(uiFS http.FileSystem) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(uiFS))
	mux.HandleFunc("/tasks", s.handleTasks)
	mux.HandleFunc("/tasks/", s.handleTask)
	mux.HandleFunc("/recurrences", s.handleRecurrences)
	mux.HandleFunc("/recurrences/", s.handleRecurrence)
	return mux
}

func (s *Server) Serve(h http.Handler) error {
	return http.Serve(s.listener, h)
}
```

- [ ] **Step 2: Write api/tasks.go**

```go
// api/tasks.go
package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/glow-mdsol/dovin/notes"
	"github.com/glow-mdsol/dovin/store"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	switch r.Method {
	case http.MethodGet:
		archive := r.URL.Query().Get("archive") == "1"
		var tasks []store.Task
		var err error
		if archive {
			tasks, err = s.store.ListArchivedTasks()
		} else {
			tasks, err = s.store.ListActiveTasks()
		}
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, toRespSlice(tasks))
	case http.MethodPost:
		var body struct {
			Title        string `json:"title"`
			Priority     int    `json:"priority"`
			ParentID     *int64 `json:"parent_id"`
			RecurrenceID *int64 `json:"recurrence_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, 400, "invalid JSON")
			return
		}
		if body.Title == "" {
			writeError(w, 400, "title required")
			return
		}
		if body.Priority == 0 {
			body.Priority = 2
		}
		parentID := sql.NullInt64{}
		if body.ParentID != nil {
			parentID = sql.NullInt64{Valid: true, Int64: *body.ParentID}
		}
		recurID := sql.NullInt64{}
		if body.RecurrenceID != nil {
			recurID = sql.NullInt64{Valid: true, Int64: *body.RecurrenceID}
		}
		task, err := s.store.CreateTask(body.Title, body.Priority, parentID, recurID)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 201, toResp(task))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleTask routes /tasks/{id}[/status|/promote|/notes]
func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// parts[0]="tasks", parts[1]=id, parts[2]=action (optional)
	if len(parts) < 2 {
		writeError(w, 400, "missing task id")
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		writeError(w, 400, "invalid task id")
		return
	}
	action := ""
	if len(parts) >= 3 {
		action = parts[2]
	}

	switch action {
	case "status":
		s.handleTaskStatus(w, r, id)
	case "promote":
		s.handleTaskPromote(w, r, id)
	case "notes":
		s.handleTaskNotes(w, r, id)
	case "priority":
		s.handleTaskPriority(w, r, id)
	default:
		s.handleTaskDefault(w, r, id)
	}
}

func (s *Server) handleTaskDefault(w http.ResponseWriter, r *http.Request, id int64) {
	switch r.Method {
	case http.MethodGet:
		task, err := s.store.GetTask(id)
		if err != nil {
			writeError(w, 404, "not found")
			return
		}
		writeJSON(w, 200, toResp(task))
	case http.MethodDelete:
		if err := s.store.DeleteTask(id); err != nil {
			writeError(w, 500, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTaskStatus(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Status string `json:"status"`
		Done   *bool  `json:"done"` // for subtasks
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	task, err := s.store.GetTask(id)
	if err != nil {
		writeError(w, 404, "not found")
		return
	}

	if task.ParentID.Valid {
		// subtask — use done bool
		if body.Done == nil {
			writeError(w, 400, "done required for subtasks")
			return
		}
		if err := s.store.UpdateSubtaskStatus(id, *body.Done); err != nil {
			writeError(w, 500, err.Error())
			return
		}
	} else {
		if err := s.store.UpdateStatus(id, body.Status); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		// auto-archive on done
		if body.Status == "done" {
			_ = s.store.UpdateStatus(id, "archived")
			// update recurrence if linked
			if task.RecurrenceID.Valid {
				r, _ := s.store.GetRecurrence(task.RecurrenceID.Int64)
				if r != nil {
					// recalc next due — handled by scheduler on next tick; just mark completed
					// Import scheduler to get NextAfter
				}
			}
		}
	}
	updated, _ := s.store.GetTask(id)
	writeJSON(w, 200, toResp(updated))
}

func (s *Server) handleTaskPromote(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := s.store.PromoteSubtask(id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	task, _ := s.store.GetTask(id)
	writeJSON(w, 200, toResp(task))
}

func (s *Server) handleTaskNotes(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	task, err := s.store.GetTask(id)
	if err != nil {
		writeError(w, 404, "not found")
		return
	}
	name, err := notes.EnsureFile(id, task.Title)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if err := s.store.SetNotesPath(id, name); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"notes_path": name})
}

func (s *Server) handleTaskPriority(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Priority int `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if body.Priority < 1 || body.Priority > 3 {
		writeError(w, 400, "priority must be 1, 2, or 3")
		return
	}
	if err := s.store.UpdatePriority(id, body.Priority); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	task, _ := s.store.GetTask(id)
	writeJSON(w, 200, toResp(task))
}
```

- [ ] **Step 3: Build to catch errors**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add api/
git commit -m "feat: HTTP API server, task handlers (CRUD/status/promote/notes/priority)"
```

---

### Task 8: API — recurrence handlers

**Files:**
- Create: `api/recurrences.go`

- [ ] **Step 1: Write api/recurrences.go**

```go
// api/recurrences.go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/glow-mdsol/dovin/scheduler"
)

func (s *Server) handleRecurrences(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	switch r.Method {
	case http.MethodGet:
		recs, err := s.store.ListRecurrences()
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, recs)
	case http.MethodPost:
		var body struct {
			Title    string `json:"title"`
			Priority int    `json:"priority"`
			Schedule string `json:"schedule"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, 400, "invalid JSON")
			return
		}
		if body.Title == "" || body.Schedule == "" {
			writeError(w, 400, "title and schedule required")
			return
		}
		if body.Priority == 0 {
			body.Priority = 2
		}
		firstDue, err := scheduler.NextAfter(body.Schedule, time.Now())
		if err != nil {
			writeError(w, 400, "invalid cron schedule: "+err.Error())
			return
		}
		rec, err := s.store.CreateRecurrence(body.Title, body.Priority, body.Schedule, firstDue)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 201, rec)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRecurrence(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		writeError(w, 400, "missing id")
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// toggle active
	var body struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if err := s.store.SetRecurrenceActive(id, body.Active); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	rec, _ := s.store.GetRecurrence(id)
	writeJSON(w, 200, rec)
}
```

- [ ] **Step 2: Build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add api/recurrences.go
git commit -m "feat: recurrence API handlers (list, create, toggle active)"
```

---

### Task 9: UI — embed setup and HTML shell

**Files:**
- Create: `ui/embed.go`
- Create: `ui/static/index.html`
- Create: `ui/static/style.css`
- Create: `ui/static/app.js` (stub)

- [ ] **Step 1: Write ui/embed.go**

```go
// ui/embed.go
package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func FileSystem() http.FileSystem {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}
```

- [ ] **Step 2: Write ui/static/style.css**

```css
* { box-sizing: border-box; margin: 0; padding: 0; }

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  font-size: 14px;
  background: #1c1c1e;
  color: #f2f2f7;
  height: 100vh;
  display: flex;
  flex-direction: column;
}

#header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid #2c2c2e;
  flex-shrink: 0;
}

#header h1 { font-size: 15px; font-weight: 600; }

#archive-toggle {
  font-size: 12px;
  color: #8e8e93;
  cursor: pointer;
  background: none;
  border: none;
  color: #636366;
}

#archive-toggle.active { color: #636366; text-decoration: underline; }

#task-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.group-label {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #636366;
  padding: 10px 14px 4px;
}

.task-row {
  padding: 8px 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  border-left: 3px solid transparent;
  transition: background 0.1s;
}

.task-row:hover { background: #2c2c2e; }
.task-row.expanded { background: #2c2c2e; border-left-color: #3a3a3c; }

.priority-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.p1 { background: #ff453a; }
.p2 { background: #ff9f0a; }
.p3 { background: #48484a; }

.task-title { flex: 1; }

.subtask-chip {
  font-size: 11px;
  color: #636366;
  background: #2c2c2e;
  border-radius: 10px;
  padding: 1px 7px;
}

.notes-icon { font-size: 13px; color: #636366; }

.detail-panel {
  background: #2c2c2e;
  padding: 8px 14px 10px 25px;
  border-left: 3px solid #3a3a3c;
}

.detail-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  align-items: center;
  font-size: 13px;
}

.detail-row label { color: #8e8e93; width: 60px; flex-shrink: 0; }

select.inline {
  background: #3a3a3c;
  color: #f2f2f7;
  border: none;
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 13px;
  cursor: pointer;
}

.subtask-list { margin-top: 8px; }

.subtask-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  font-size: 13px;
}

.subtask-item input[type=checkbox] { cursor: pointer; }

.subtask-promote {
  font-size: 11px;
  color: #636366;
  cursor: pointer;
  background: none;
  border: none;
  margin-left: auto;
  padding: 0;
}

.subtask-promote:hover { color: #aeaeb2; }

.add-subtask-row {
  display: flex;
  gap: 6px;
  margin-top: 6px;
}

.add-subtask-row input {
  flex: 1;
  background: #3a3a3c;
  border: none;
  border-radius: 4px;
  padding: 4px 8px;
  color: #f2f2f7;
  font-size: 13px;
}

.add-subtask-row button, .btn-notes {
  background: #3a3a3c;
  color: #aeaeb2;
  border: none;
  border-radius: 4px;
  padding: 4px 10px;
  font-size: 12px;
  cursor: pointer;
}

.btn-notes { margin-top: 6px; }

#footer {
  border-top: 1px solid #2c2c2e;
  padding: 8px 14px;
  flex-shrink: 0;
}

#add-form { display: flex; flex-direction: column; gap: 6px; }

#add-form.hidden { display: none; }

#add-title {
  background: #2c2c2e;
  border: none;
  border-radius: 6px;
  padding: 8px 10px;
  color: #f2f2f7;
  font-size: 14px;
  width: 100%;
}

.form-row { display: flex; gap: 6px; align-items: center; }

.form-row select {
  background: #2c2c2e;
  color: #f2f2f7;
  border: none;
  border-radius: 4px;
  padding: 4px 8px;
  font-size: 13px;
}

.form-row input[type=text] {
  flex: 1;
  background: #2c2c2e;
  border: none;
  border-radius: 4px;
  padding: 4px 8px;
  color: #f2f2f7;
  font-size: 13px;
}

#add-trigger {
  width: 100%;
  background: #2c2c2e;
  color: #aeaeb2;
  border: none;
  border-radius: 6px;
  padding: 7px;
  font-size: 13px;
  cursor: pointer;
  text-align: left;
}

#add-trigger:hover { background: #3a3a3c; }

.btn-primary {
  background: #0a84ff;
  color: white;
  border: none;
  border-radius: 4px;
  padding: 5px 14px;
  font-size: 13px;
  cursor: pointer;
}

.btn-primary:hover { background: #409cff; }
```

- [ ] **Step 3: Write ui/static/index.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Dovin</title>
  <link rel="stylesheet" href="/style.css">
</head>
<body>
  <div id="header">
    <h1>Dovin</h1>
    <button id="archive-toggle" onclick="toggleArchive()">Archive</button>
  </div>
  <div id="task-list"></div>
  <div id="footer">
    <button id="add-trigger" onclick="showAddForm()">+ Add Task</button>
    <form id="add-form" class="hidden" onsubmit="submitTask(event)">
      <input id="add-title" type="text" placeholder="Task title…" autocomplete="off" required>
      <div class="form-row">
        <select id="add-priority">
          <option value="1">High</option>
          <option value="2" selected>Medium</option>
          <option value="3">Low</option>
        </select>
        <input type="text" id="add-schedule" placeholder="Recurrence (e.g. 0 9 * * 1)" style="flex:1">
        <button type="submit" class="btn-primary">Add</button>
      </div>
    </form>
  </div>
  <script src="/app.js"></script>
</body>
</html>
```

- [ ] **Step 4: Create stub app.js**

```js
// ui/static/app.js
// full implementation in Task 10
```

- [ ] **Step 5: Build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add ui/
git commit -m "feat: UI embed setup, HTML shell, CSS"
```

---

### Task 10: UI — app.js (task list, add form, detail panel)

**Files:**
- Modify: `ui/static/app.js`

- [ ] **Step 1: Write the full app.js**

```js
// ui/static/app.js
let showArchive = false;
let expandedId = null;

const PRIORITY_LABEL = { 1: 'High', 2: 'Medium', 3: 'Low' };
const PRIORITY_CLASS = { 1: 'p1', 2: 'p2', 3: 'p3' };
const STATUS_OPTIONS = ['todo', 'in_progress', 'blocked'];

async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const r = await fetch(path, opts);
  if (r.status === 204) return null;
  return r.json();
}

async function loadTasks() {
  if (window.location.hash === '#add') {
    window.location.hash = '';
    showAddForm();
  }
  const url = showArchive ? '/tasks?archive=1' : '/tasks';
  const tasks = await api('GET', url);
  renderTasks(tasks || []);
}

function renderTasks(tasks) {
  const el = document.getElementById('task-list');
  if (!tasks.length) {
    el.innerHTML = '<p style="color:#636366;padding:20px 14px;font-size:13px;">No tasks.</p>';
    return;
  }

  const groups = [
    { key: 'in_progress', label: 'In Progress' },
    { key: 'todo',        label: 'Todo' },
    { key: 'blocked',     label: 'Blocked' },
    { key: 'archived',    label: 'Archived' },
  ];

  let html = '';
  for (const g of groups) {
    const items = tasks.filter(t => t.status === g.key);
    if (!items.length) continue;
    html += `<div class="group-label">${g.label}</div>`;
    for (const t of items) html += renderTaskRow(t);
  }
  el.innerHTML = html;
}

function renderTaskRow(t) {
  const prio = t.priority || 2;
  const pClass = PRIORITY_CLASS[prio] || 'p2';
  const subtaskCount = t.subtasks ? t.subtasks.length : 0;
  const subtaskDone = t.subtasks ? t.subtasks.filter(s => s.status === 'done').length : 0;
  const chip = subtaskCount ? `<span class="subtask-chip">${subtaskDone}/${subtaskCount}</span>` : '';
  const notesIcon = t.notes_path ? '📄' : '';
  const expanded = expandedId === t.id;

  let html = `
    <div class="task-row ${expanded ? 'expanded' : ''}" onclick="toggleExpand(${t.id})">
      <span class="priority-dot ${pClass}"></span>
      <span class="task-title">${escHtml(t.title)}</span>
      ${chip}
      <span class="notes-icon">${notesIcon}</span>
    </div>`;

  if (expanded) html += renderDetail(t);
  return html;
}

function renderDetail(t) {
  const prio = t.priority || 2;
  const statusOpts = STATUS_OPTIONS.map(s =>
    `<option value="${s}" ${t.status === s ? 'selected' : ''}>${s.replace('_', ' ')}</option>`
  ).join('');
  const prioOpts = [1, 2, 3].map(p =>
    `<option value="${p}" ${prio === p ? 'selected' : ''}>${PRIORITY_LABEL[p]}</option>`
  ).join('');

  let subtasksHtml = '';
  if (t.subtasks && t.subtasks.length) {
    subtasksHtml = t.subtasks.map(s => `
      <div class="subtask-item">
        <input type="checkbox" ${s.status === 'done' ? 'checked' : ''}
               onchange="setSubtaskDone(${s.id}, this.checked)">
        <span>${escHtml(s.title)}</span>
        <button class="subtask-promote" onclick="promoteSubtask(${s.id})" title="Promote to task">↑</button>
      </div>`).join('');
  }

  return `
    <div class="detail-panel">
      <div class="detail-row">
        <label>Status</label>
        <select class="inline" onchange="setStatus(${t.id}, this.value)">${statusOpts}</select>
      </div>
      <div class="detail-row">
        <label>Priority</label>
        <select class="inline" onchange="setPriority(${t.id}, this.value)">${prioOpts}</select>
      </div>
      <div class="subtask-list">${subtasksHtml}</div>
      <div class="add-subtask-row">
        <input type="text" id="sub-input-${t.id}" placeholder="Add subtask…">
        <button onclick="addSubtask(${t.id})">Add</button>
      </div>
      <button class="btn-notes" onclick="openNotes(${t.id})">📄 Notes</button>
    </div>`;
}

function toggleExpand(id) {
  expandedId = expandedId === id ? null : id;
  loadTasks();
}

async function setStatus(id, status) {
  await api('POST', `/tasks/${id}/status`, { status });
  loadTasks();
}

async function setPriority(id, priority) {
  await api('POST', `/tasks/${id}/priority`, { priority: parseInt(priority) });
  loadTasks();
}

async function setSubtaskDone(id, done) {
  await api('POST', `/tasks/${id}/status`, { done });
  loadTasks();
}

async function promoteSubtask(id) {
  await api('POST', `/tasks/${id}/promote`, {});
  expandedId = null;
  loadTasks();
}

async function openNotes(id) {
  await api('POST', `/tasks/${id}/notes`, {});
}

async function addSubtask(parentId) {
  const input = document.getElementById(`sub-input-${parentId}`);
  const title = input.value.trim();
  if (!title) return;
  await api('POST', '/tasks', { title, parent_id: parentId });
  input.value = '';
  loadTasks();
}

function showAddForm() {
  document.getElementById('add-trigger').style.display = 'none';
  document.getElementById('add-form').classList.remove('hidden');
  document.getElementById('add-title').focus();
}

async function submitTask(e) {
  e.preventDefault();
  const title = document.getElementById('add-title').value.trim();
  const priority = parseInt(document.getElementById('add-priority').value);
  const schedule = document.getElementById('add-schedule').value.trim();
  if (!title) return;

  if (schedule) {
    // create recurrence first, then the task will appear when due
    await api('POST', '/recurrences', { title, priority, schedule });
  } else {
    await api('POST', '/tasks', { title, priority });
  }

  document.getElementById('add-title').value = '';
  document.getElementById('add-schedule').value = '';
  document.getElementById('add-form').classList.add('hidden');
  document.getElementById('add-trigger').style.display = '';
  loadTasks();
}

function toggleArchive() {
  showArchive = !showArchive;
  const btn = document.getElementById('archive-toggle');
  btn.classList.toggle('active', showArchive);
  loadTasks();
}

function escHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

// Poll for changes every 30s (scheduler may add tasks)
loadTasks();
setInterval(loadTasks, 30000);
```

- [ ] **Step 2: Commit**

```bash
git add ui/static/app.js
git commit -m "feat: single-page UI with task list, detail panel, add form"
```

---

### Task 11: Wire it all together in main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Write main.go**

```go
package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/fyne-io/systray"
	"github.com/glow-mdsol/dovin/api"
	"github.com/glow-mdsol/dovin/scheduler"
	"github.com/glow-mdsol/dovin/store"
	"github.com/glow-mdsol/dovin/ui"
	"github.com/webview/webview_go"
)

//go:embed assets/icon.png
var iconData []byte

func main() {
	logFile := fmt.Sprintf("%s/Library/Logs/dovin.log", os.Getenv("HOME"))
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(f)
		defer f.Close()
	}

	s, err := store.Open()
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	srv, err := api.New(s)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	port := srv.Port()
	if err := s.ConfigSet("port", strconv.Itoa(port)); err != nil {
		log.Printf("save port: %v", err)
	}

	go func() {
		if err := srv.Serve(srv.Handler(ui.FileSystem())); err != nil {
			log.Printf("server: %v", err)
		}
	}()

	done := make(chan struct{})
	go scheduler.Run(s, done)

	systray.Run(func() {
		systray.SetIcon(iconData)
		systray.SetTooltip("Dovin")
		updateBadge(s)

		mOpen := systray.AddMenuItem("Open", "Open Dovin")
		mAdd := systray.AddMenuItem("Add Task…", "Add a new task")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit Dovin")

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					openWebview(port, "")
					updateBadge(s)
				case <-mAdd.ClickedCh:
					openWebview(port, "#add")
					updateBadge(s)
				case <-mQuit.ClickedCh:
					close(done)
					systray.Quit()
				}
			}
		}()
	}, func() {})
}

func openWebview(port int, fragment string) {
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Dovin")
	w.SetSize(420, 600, webview.HintFixed)
	url := fmt.Sprintf("http://localhost:%d/%s", port, fragment)
	w.Navigate(url)
	w.Run()
}

func updateBadge(s *store.Store) {
	n, err := s.ActiveTaskCount()
	if err != nil || n == 0 {
		systray.SetTitle("")
		return
	}
	systray.SetTitle(strconv.Itoa(n))
}
```

- [ ] **Step 2: Build**

```bash
go build -o bin/dovin .
```

Expected: binary produced. CGo will kick in for the webview — ensure Xcode Command Line Tools are installed (`xcode-select --install`).

- [ ] **Step 4: Run tests**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add main.go ui/static/app.js
git commit -m "feat: wire systray, webview, server, and scheduler in main.go"
```

---

### Task 12: Handle done→archived recurrence update in API

The `handleTaskStatus` in `api/tasks.go` has a TODO comment for recurrence `next_due_at` update. Fix it now.

**Files:**
- Modify: `api/tasks.go`

- [ ] **Step 1: Update handleTaskStatus to recalculate next_due_at on done**

Replace the done-handling block in `handleTaskStatus`:

```go
// auto-archive on done
if body.Status == "done" {
    _ = s.store.UpdateStatus(id, "archived")
    if task.RecurrenceID.Valid {
        rec, err := s.store.GetRecurrence(task.RecurrenceID.Int64)
        if err == nil && rec != nil {
            next, err := scheduler.NextAfter(rec.Schedule, time.Now())
            if err == nil {
                _ = s.store.MarkRecurrenceCompleted(rec.ID, next)
            }
        }
    }
}
```

Add the missing imports to `api/tasks.go`:

```go
import (
    "database/sql"
    "encoding/json"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/glow-mdsol/dovin/notes"
    "github.com/glow-mdsol/dovin/scheduler"
    "github.com/glow-mdsol/dovin/store"
)
```

- [ ] **Step 2: Build and test**

```bash
go build ./... && go test ./...
```

- [ ] **Step 3: Commit**

```bash
git add api/tasks.go
git commit -m "fix: recalculate recurrence next_due_at when task marked done"
```

---

### Task 13: Makefile install target and launchd plist

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Replace the install target with a shell-script approach** (the heredoc in Make is fragile — use a Go-generated plist instead)

Replace the `install` target in `Makefile`:

```makefile
install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BIN) $(INSTALL_DIR)/$(BINARY_NAME)
	@mkdir -p $(HOME)/Library/LaunchAgents
	@printf '<?xml version="1.0" encoding="UTF-8"?>\n\
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n\
<plist version="1.0">\n<dict>\n\
  <key>Label</key><string>$(PLIST_NAME)</string>\n\
  <key>ProgramArguments</key><array><string>$(INSTALL_DIR)/$(BINARY_NAME)</string></array>\n\
  <key>RunAtLoad</key><true/>\n\
  <key>KeepAlive</key><true/>\n\
  <key>StandardOutPath</key><string>$(LOG_PATH)</string>\n\
  <key>StandardErrorPath</key><string>$(LOG_PATH)</string>\n\
</dict>\n</plist>\n' > $(PLIST_PATH)
	launchctl load $(PLIST_PATH)
	@echo "Dovin installed. It will start on next login, or run: launchctl start $(PLIST_NAME)"
```

- [ ] **Step 2: Test the build target**

```bash
make build
ls -la bin/dovin
```

Expected: binary exists.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "fix: robust launchd plist generation in Makefile"
```

---

### Task 14: Manual smoke test

- [ ] **Step 1: Run the binary directly**

```bash
./bin/dovin &
```

Expected: menu bar icon appears in macOS menu bar with a blue square.

- [ ] **Step 2: Open the UI**

Click the menu bar icon → "Open". A 420×600 webview window should open showing "Dovin" with an empty task list and "+ Add Task" button.

- [ ] **Step 3: Add a task**

Click "+ Add Task", type "Test task", leave priority as Medium, click Add. The task should appear under **Todo** with an amber dot.

- [ ] **Step 4: Change status**

Click the task row to expand it. Change status to "In Progress" via the dropdown. The task should move to the **In Progress** group.

- [ ] **Step 5: Add a subtask**

With the task expanded, type "subtask one" in the "Add subtask…" field and click Add. A checkbox item should appear.

- [ ] **Step 6: Add a recurring task**

Click "+ Add Task", title "GitHub admin", enter schedule `0 9 * * 1` (every Monday 9am). Click Add. No task should appear immediately (it's invisible until due).

- [ ] **Step 7: Kill the test process**

```bash
kill %1
```

- [ ] **Step 8: Run tests one final time**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 9: Final commit**

```bash
git add -A
git commit -m "feat: dovin task manager complete — smoke tested"
```
