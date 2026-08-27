package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/glow-mdsol/dovin/scheduler"
	"github.com/glow-mdsol/dovin/store"
)

type recurrenceResp struct {
	ID              int64   `json:"id"`
	Title           string  `json:"title"`
	Priority        int     `json:"priority"`
	Schedule        string  `json:"schedule"`
	NextDueAt       *string `json:"next_due_at"`
	LastCompletedAt *string `json:"last_completed_at"`
	Active          bool    `json:"active"`
}

func toRecurrenceResp(r *store.Recurrence) recurrenceResp {
	resp := recurrenceResp{
		ID:       r.ID,
		Title:    r.Title,
		Priority: r.Priority,
		Schedule: r.Schedule,
		Active:   r.Active,
	}
	if r.NextDueAt.Valid {
		s := r.NextDueAt.Time.UTC().Format(time.RFC3339)
		resp.NextDueAt = &s
	}
	if r.LastCompletedAt.Valid {
		s := r.LastCompletedAt.Time.UTC().Format(time.RFC3339)
		resp.LastCompletedAt = &s
	}
	return resp
}

func toRecurrenceRespSlice(recs []store.Recurrence) []recurrenceResp {
	out := make([]recurrenceResp, len(recs))
	for i := range recs {
		out[i] = toRecurrenceResp(&recs[i])
	}
	return out
}

func (s *Server) handleRecurrences(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	switch r.Method {
	case http.MethodGet:
		recs, err := s.store.ListRecurrences()
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, toRecurrenceRespSlice(recs))
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
		cronExpr, err := scheduler.ParseSchedule(body.Schedule)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		body.Schedule = cronExpr
		firstDue, err := scheduler.NextAfter(cronExpr, time.Now())
		if err != nil {
			writeError(w, 400, "invalid schedule: "+err.Error())
			return
		}
		rec, err := s.store.CreateRecurrence(body.Title, body.Priority, body.Schedule, firstDue)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		// Create first task immediately; DueRecurrences guards against duplicates
		// while a todo/in_progress task is pending.
		_, _ = s.store.CreateTask(rec.Title, rec.Priority,
			sql.NullInt64{},
			sql.NullInt64{Valid: true, Int64: rec.ID},
		)
		writeJSON(w, 201, toRecurrenceResp(rec))
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
	switch r.Method {
	case http.MethodDelete:
		if err := s.store.DeleteRecurrence(id); err != nil {
			writeError(w, 500, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPost:
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
		rec, err := s.store.GetRecurrence(id)
		if err != nil || rec == nil {
			writeError(w, http.StatusInternalServerError, "could not reload recurrence")
			return
		}
		writeJSON(w, 200, toRecurrenceResp(rec))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
