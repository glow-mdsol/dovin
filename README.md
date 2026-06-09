# Dovin

A macOS menu bar task manager. Always running, always one click away.

## Features

- **Task lifecycle**: todo → in_progress → blocked → done → archived
- **Priorities**: high / medium / low, colour-coded in the UI
- **Subtasks**: one level deep; subtasks can be promoted to top-level tasks
- **Recurring tasks**: cron-based schedule; tasks appear automatically when due
- **Notes**: lazy-created Markdown file per task, opened in `$EDITOR`
- **Menu bar badge**: count of active (todo + in_progress) tasks

## Requirements

- macOS 12+
- Go 1.22+ (`brew install go`)
- Xcode Command Line Tools (`xcode-select --install`) — required for the WebKit webview

## Build & Run

```bash
# Clone
git clone https://github.com/glow-mdsol/dovin
cd dovin

# Build
make build          # produces bin/dovin

# Run directly (foreground, logs to stdout)
./bin/dovin
```

The menu bar icon appears immediately. Click it to open the task panel.

## Install as a background service

```bash
make install
```

This:
1. Copies the binary to `~/.local/bin/dovin`
2. Writes a launchd plist to `~/Library/LaunchAgents/com.glow.dovin.plist`
3. Loads it via `launchctl` (starts immediately, restarts on crash, launches at login)

```bash
make uninstall      # stops the service and removes all installed files
```

Logs are written to `~/Library/Logs/dovin.log`.

## Usage

### Menu bar

| Menu item | Action |
|---|---|
| **Open** | Opens the task panel |
| **Add Task…** | Opens the panel with the add form focused |
| **Quit** | Exits the app |

The title next to the icon shows the count of active tasks (todo + in_progress). It clears when everything is done.

### Task panel

**Adding a task**

Click **+ Add Task** at the bottom. Fill in:

- **Title** (required)
- **Priority**: high / medium (default) / low
- **Schedule** (optional): a cron expression makes this a recurring task (e.g. `0 9 * * 1` for Monday 9am). Leave blank for a one-off task.

**Task lifecycle**

Click any task row to expand it. Use the **Status** dropdown to advance the task:

| Status | Meaning |
|---|---|
| `todo` | Not started |
| `in_progress` | Being worked on |
| `blocked` | Waiting on something |
| `done` | Complete — auto-archives immediately |
| `archived` | Hidden from main view |

Any task can be discarded directly to `archived` regardless of current status.

**Subtasks**

Expand a task and use the **Add subtask** field at the bottom of the detail panel. Subtasks have a title and a checkbox (todo / done). Checking all subtasks does not auto-close the parent — that remains a manual step.

Click **↑** next to a subtask to promote it to a standalone task (inherits medium priority).

**Notes**

Click the document icon on a task row (or the **Notes** button in the detail panel) to open — or create — a Markdown notes file for that task. The file lives at `~/.config/dovin/notes/{id}-{title}.md` and opens in `$EDITOR` (falls back to `open`). Notes files are never deleted automatically.

**Archive view**

Toggle **Show archive** at the top of the panel to see completed tasks.

### Recurring tasks

When you submit a task with a schedule, a recurrence record is created instead of a task. The scheduler checks every 60 seconds and creates a `todo` task when the next due time arrives. Completing that task (marking done) automatically updates the next due time so the cycle continues.

Cron format: `minute hour day-of-month month day-of-week`, e.g.:

| Schedule | Meaning |
|---|---|
| `0 9 * * 1` | Every Monday at 9am |
| `0 9 * * 1-5` | Weekdays at 9am |
| `0 9 1 * *` | First of every month at 9am |
| `0 9 * * *` | Every day at 9am |

## Storage

All data is stored in `~/.config/dovin/dovin.db` (SQLite). Notes files live in `~/.config/dovin/notes/`. Neither is touched by `make uninstall`.

## Development

```bash
make test           # run all tests
make build          # compile to bin/dovin
```

Tests cover store CRUD (tasks, recurrences, config), state machine transitions, and cron scheduling logic. The webview layer is thin and tested manually.
