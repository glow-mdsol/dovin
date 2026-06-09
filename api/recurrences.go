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
