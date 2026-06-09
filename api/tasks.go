package api

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

// handleTask routes /tasks/{id}[/status|/promote|/notes|/priority]
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
		// auto-archive on done + update recurrence next_due_at
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

// Stubs — implemented in api/recurrences.go (Task 8)
func (s *Server) handleRecurrences(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (s *Server) handleRecurrence(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
