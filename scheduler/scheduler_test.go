// scheduler/scheduler_test.go
package scheduler_test

import (
	"testing"
	"time"

	"github.com/glow-mdsol/dovin/scheduler"
)

func TestNextAfter(t *testing.T) {
	// "0 9 * * 1" = every Monday at 09:00
	// Use a known Monday + advance to find next Monday
	monday := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC) // a Monday
	next, err := scheduler.NextAfter("0 9 * * 1", monday)
	if err != nil {
		t.Fatalf("NextAfter: %v", err)
	}
	// next Monday 09:00 after midnight Monday 8 June = same day at 09:00
	want := time.Date(2026, 6, 8, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}

	// after 09:00 on Monday, next should be following Monday
	next2, _ := scheduler.NextAfter("0 9 * * 1", time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC))
	want2 := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	if !next2.Equal(want2) {
		t.Errorf("next2 = %v, want %v", next2, want2)
	}
}

func TestInvalidSchedule(t *testing.T) {
	_, err := scheduler.NextAfter("not-a-cron", time.Now())
	if err == nil {
		t.Fatal("expected error for invalid cron, got nil")
	}
}
