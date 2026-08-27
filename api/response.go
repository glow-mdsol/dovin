package api

import (
	"time"

	"github.com/glow-mdsol/dovin/store"
)

// noteResp is the flat JSON shape returned to the frontend.
type noteResp struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	TaskID     *int64 `json:"task_id"`
	CreatedAt  string `json:"created_at"`
	ModifiedAt string `json:"modified_at"`
}

func toNoteResp(n *store.Note) noteResp {
	r := noteResp{
		ID:         n.ID,
		Title:      n.Title,
		Content:    n.Content,
		TaskID:     n.TaskID,
		CreatedAt:  n.CreatedAt.Format(time.RFC3339),
		ModifiedAt: n.ModifiedAt.Format(time.RFC3339),
	}
	return r
}

func toNoteRespSlice(notes []*store.Note) []noteResp {
	out := make([]noteResp, len(notes))
	for i, n := range notes {
		out[i] = toNoteResp(n)
	}
	return out
}

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
