# Dovin — Personal Task Manager: Design Spec

**Date:** 2026-06-09  
**Status:** Approved

---

## Overview

Dovin is a personal task manager that runs as a macOS menu bar app. It is always running, surfaces recurring tasks automatically when they are due, and provides a rich but minimal UI for managing tasks through a defined lifecycle.

---

## Platform & Tech Stack

| Concern | Choice |
|---|---|
| Language | Go |
| Menu bar | `fyne.io/systray` |
| UI popup | `github.com/webview/webview` (embedded webview ~420×600px, uses CGo/WKWebView on macOS) |
| Local API | `net/http` (Go stdlib, loopback only) |
| Database | SQLite via `modernc.org/sqlite` (pure Go, no CGo required) |
| Frontend | Embedded HTML/CSS/JS via `embed.FS` |
| Startup | launchd plist (`~/Library/LaunchAgents/com.glow.dovin.plist`) |

---

## Architecture

```
main.go
├── starts HTTP server (loopback, random port stored in SQLite config)
├── registers systray icon + menu
│   ├── "Open" → opens webview popup
│   ├── "Add Task…" → opens webview with add form pre-focused
│   └── "Quit"
├── launches scheduler goroutine (ticks every minute)
└── blocks on systray.Run()

store/        — SQLite queries (tasks, recurrences, config)
api/          — HTTP handlers: /tasks, /tasks/{id}, /tasks/{id}/status, /recurrences
scheduler/    — promotes due recurring tasks to todo
ui/           — embedded static assets (index.html, app.js, style.css)
```

The webview loads `http://localhost:{port}` on open. The frontend is a single-page app that talks to the Go HTTP API via `fetch`. No external network access is required or used.

---

## Data Model

### `tasks`

| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| title | TEXT | e.g. "GitHub admin" |
| status | TEXT | `todo` \| `in_progress` \| `blocked` \| `done` \| `archived` |
| notes | TEXT | optional free text |
| recurrence_id | INTEGER FK | null for one-off tasks |
| created_at | DATETIME | |
| updated_at | DATETIME | |

### `recurrences`

| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| title | TEXT | task template name |
| schedule | TEXT | cron expression e.g. `0 9 * * 1` (Mon 9am) |
| next_due_at | DATETIME | scheduler inserts a new todo task when `next_due_at ≤ now` |
| last_completed_at | DATETIME | updated when linked task is marked done |
| active | BOOLEAN | pause a recurrence without deleting it |

---

## Task Lifecycle

```
todo ──► in_progress ──► done ──► archived (auto, immediate)
  │           │
  │           ▼
  └──────► blocked ──► in_progress
               │
               └──► todo (reset)

any state ──► archived (manual discard)
```

**Allowed transitions:**

| From | To |
|---|---|
| `todo` | `in_progress`, `blocked`, `archived` |
| `in_progress` | `blocked`, `done` |
| `blocked` | `in_progress`, `todo` |
| `done` | `archived` (auto, no manual step) |
| `archived` | — (terminal) |

---

## Recurring Task Flow

1. Task is marked **done** → status set to `archived`, `last_completed_at` updated on its recurrence row, `next_due_at` recalculated from the cron expression.
2. Scheduler goroutine ticks every 60 seconds. For each active recurrence where `next_due_at ≤ now` and no linked task exists with status `todo` or `in_progress`, it inserts a new `todo` task linked to that recurrence.
3. The new task appears in the UI on the next data fetch.

Recurring tasks are **invisible until due** — they do not appear in the task list until the scheduler promotes them.

---

## UI Layout

Single floating webview panel (~420×600px), positioned below the menu bar icon.

```
┌─────────────────────────────────┐
│  Dovin                    [×]   │
├─────────────────────────────────┤
│  IN PROGRESS                    │
│  ▎ GitHub admin                 │
│                                 │
│  TODO                           │
│  ▎ Review PRs                   │
│  ▎ Update dependencies          │
│                                 │
│  BLOCKED                        │
│  ▎ Deploy staging               │
├─────────────────────────────────┤
│  [+ Add Task]                   │
└─────────────────────────────────┘
```

- Tasks are grouped by status: **In Progress** → **Todo** → **Blocked**
- Each task row: click to expand detail/edit panel inline
- Status can be changed via a dropdown on the task row
- Archive view accessible via a toggle at the top (hidden by default)
- Closing the window (×) hides it; does not quit the app

### Add Task form (inline, expands at bottom)

Fields: title (required), notes (optional), recurrence schedule (optional cron or human-readable picker: daily / weekly / monthly / custom).

---

## Menu Bar Icon

- Default: app icon
- Badge: count of `todo` + `in_progress` tasks (shown as a number overlay)
- Updates after any task state change

---

## Installation & Startup

```makefile
make build    # builds ./bin/dovin
make install  # copies to ~/.local/bin/dovin, writes launchd plist, loads it
make uninstall # unloads plist, removes binary and plist file
```

launchd plist path: `~/Library/LaunchAgents/com.glow.dovin.plist`  
The plist sets `RunAtLoad true` and `KeepAlive true` so the process restarts if it crashes.

---

## Error Handling

- All DB errors are logged to `~/Library/Logs/dovin.log` (launchd stdout/stderr redirect)
- API errors return JSON `{"error": "message"}` with appropriate HTTP status codes
- Scheduler errors are logged and skipped — a failed tick does not crash the app
- If the webview fails to open, the systray menu remains functional

---

## Testing

- Unit tests on `store/` package: SQL queries for task CRUD, recurrence promotion logic
- Unit tests on `scheduler/` package: due-date calculation from cron expressions
- No UI tests; the webview layer is thin and manually verified
- `go test ./...` must pass before `make install`
