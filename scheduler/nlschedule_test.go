package scheduler

import (
	"testing"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		input string
		want  string
		isErr bool
	}{
		// Cron passthrough
		{"0 9 * * 1", "0 9 * * 1", false},
		{"*/15 * * * *", "*/15 * * * *", false},
		// Natural language — weekdays
		{"every wednesday", "0 9 * * 3", false},
		{"every wednesday at 3pm", "0 15 * * 3", false},
		{"every friday at 5pm", "0 17 * * 5", false},
		{"5pm on friday", "0 17 * * 5", false},
		// Natural language — daily
		{"daily", "0 9 * * *", false},
		{"daily at 7am", "0 7 * * *", false},
		// Natural language — other
		{"every weekday", "0 9 * * 1-5", false},
		{"every weekend", "0 9 * * 0,6", false},
		{"monthly", "0 9 1 * *", false},
		{"monthly at 8am", "0 8 1 * *", false},
		// One-off (should error)
		{"next tuesday", "", true},
		{"tomorrow at 5pm", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSchedule(tt.input)
			if tt.isErr {
				if err == nil {
					t.Errorf("ParseSchedule(%q) = %q, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseSchedule(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSchedule(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
