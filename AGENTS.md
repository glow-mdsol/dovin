# Dovin Custom Agents

Specialized agents for the dovin task manager project. Route tasks to the agent that owns the relevant code—multi-layer work uses **fullstack**, single-layer changes use the focused agent.

---

## agent-fullstack

**When to use:** Features and fixes that span backend + frontend, or when uncertain which layer to address.

**Expertise:**
- Full-stack feature implementation
- Go API + JavaScript coordination
- Database-to-UI data flow
- Cross-layer refactoring

**Owns:** Backend HTTP handlers, frontend event loops, database layer, and application lifecycle  
**Files:** `api/**/*.go`, `ui/static/**`, `store/**/*.go`, `main.go`

---

## agent-backend

**When to use:** API design, server logic, database queries, business rules, and performance tuning.

**Expertise:**
- RESTful HTTP API (net/http)
- SQLite queries and schema
- Go error handling and testing
- Performance and security (CORS, input validation)

**Owns:** All server-side code and configuration  
**Files:** `api/**/*.go`, `store/**/*.go`, `scheduler/**/*.go`, `notes/**/*.go`, `main.go`, `go.mod`, `Makefile`

---

## agent-frontend

**When to use:** UI bugs, visual styling, interactivity, responsive layout, and user experience improvements.

**Expertise:**
- Vanilla JavaScript (no frameworks)
- DOM manipulation and event handling
- Markdown preview (marked.js) and syntax highlighting (Prism.js)
- CSS dark theme and split-pane UI
- Form validation and auto-save

**Owns:** All client-side code and styling  
**Files:** `ui/static/app.js`, `ui/static/index.html`, `ui/static/style.css`, `ui/embed.go`

---

## agent-tasks

**When to use:** Task CRUD operations, status/priority management, subtasks, archiving, and task-related workflows.

**Expertise:**
- Task state management
- Status transitions and filtering
- Subtask hierarchies
- Task-to-note linking
- Archiving and cleanup logic

**Owns:** Task subsystem across all layers  
**Files:** `api/tasks.go`, `store/tasks.go`, task handlers in `ui/static/app.js` and `ui/static/index.html`

---

## agent-notes

**When to use:** Markdown editing, note storage/retrieval, real-time preview, and note-to-task associations.

**Expertise:**
- Markdown editor UI and live preview
- Note CRUD operations
- Content persistence and sync
- Task-note relationships
- External note file handling

**Owns:** Notes subsystem across all layers  
**Files:** `api/notes.go`, `store/notes.go`, `notes/notes.go`, note handlers in `ui/static/app.js` and `ui/static/index.html`

---

## agent-scheduler

**When to use:** Recurring task scheduling, cron rules, natural language scheduling, and schedule management.

**Expertise:**
- Natural language schedule parsing
- Cron expression generation and execution
- Recurring task lifecycle
- Schedule-to-task synchronization

**Owns:** Scheduling and recurrence subsystem  
**Files:** `scheduler/**/*.go`, `api/recurrences.go`, `store/recurrences.go`

---

## agent-database

**When to use:** Schema design, migrations, query optimization, indexing, and data consistency issues.

**Expertise:**
- SQLite schema design
- Query optimization and performance profiling
- Migration strategies
- Referential integrity and constraints
- Index management and analysis

**Owns:** Database layer and testing  
**Files:** `store/*.go`, `store/**/*_test.go`

---

## Usage

Invoke an agent by name with your task:

```
@agent-fullstack Build recurring task feature end-to-end
@agent-frontend Fix dark theme styling in the notes editor
@agent-backend Optimize task list query performance
@agent-tasks Add task archival workflow
@agent-notes Improve markdown preview sync
@agent-scheduler Add weekly recurrence support
@agent-database Add index for task status filtering
```

Or use VS Code's agent selector when working on a specific area.
