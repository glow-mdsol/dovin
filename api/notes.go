package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleNotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Title  string `json:"title"`
			TaskID *int64 `json:"task_id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Title == "" {
			writeError(w, http.StatusBadRequest, "title required")
			return
		}
		taskID := int64(0)
		if req.TaskID != nil {
			taskID = *req.TaskID
		}
		note, err := s.store.AddNoteToTask(taskID, req.Title)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to create note")
			return
		}
		writeJSON(w, http.StatusCreated, note)

	case http.MethodGet:
		notes, err := s.store.ListNotesAPI()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list notes")
			return
		}
		if notes == nil {
			writeJSON(w, http.StatusOK, []noteResp{})
			return
		}
		writeJSON(w, http.StatusOK, toNoteRespSlice(notes))

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleNote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		writeError(w, http.StatusBadRequest, "missing note id")
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		note, err := s.store.GetNoteById(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeJSON(w, http.StatusOK, toNoteResp(note))

	case http.MethodPut:
		var req struct {
			Content string `json:"content"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if err := s.store.SaveNote(id, req.Content); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save note")
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		if err := s.store.DeleteNote(id); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete note")
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
