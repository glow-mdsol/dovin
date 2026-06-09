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
