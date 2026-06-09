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
