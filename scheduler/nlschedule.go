package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
	"github.com/robfig/cron/v3"
)

var nlParser *when.Parser

func init() {
	nlParser = when.New(nil)
	nlParser.Add(en.All...)
	nlParser.Add(common.All...)
}

// ParseSchedule converts a human-readable schedule string to a 5-field cron expression.
// Valid cron expressions are passed through unchanged.
// Natural language must describe a recurring pattern.
func ParseSchedule(input string) (string, error) {
	input = strings.TrimSpace(input)

	// Pass through valid cron expressions unchanged
	specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := specParser.Parse(input); err == nil {
		return input, nil
	}

	lower := strings.ToLower(input)

	if !isRecurring(lower) {
		return "", fmt.Errorf("schedule %q does not describe a recurring pattern; try 'every wednesday at 9am' or a cron expression like '0 9 * * 3'", input)
	}

	// Hourly shorthand
	if strings.Contains(lower, "hour") {
		return "0 * * * *", nil
	}

	// Monthly shorthand — day-of-month=1
	if strings.Contains(lower, "month") {
		hour, minute := extractTime(lower, 9, 0)
		return fmt.Sprintf("%d %d 1 * *", minute, hour), nil
	}

	// Extract hour/minute; strip recurring keywords so `when` focuses on time/day tokens
	stripped := stripRecurringKeywords(lower)
	hour, minute := extractTime(stripped, 9, 0)

	// Build day-of-week field
	cronDay := "*"
	switch {
	case strings.Contains(lower, "weekday"):
		cronDay = "1-5"
	case strings.Contains(lower, "weekend"):
		cronDay = "0,6"
	default:
		if d := namedWeekday(lower); d >= 0 {
			cronDay = strconv.Itoa(d)
		}
	}

	return fmt.Sprintf("%d %d * * %s", minute, hour, cronDay), nil
}

// extractTime uses the when library to pull hour/minute from s, falling back to defaults.
// It only uses the parsed result if the matched text contains an explicit time token
// (am/pm or HH:MM), to avoid using "next wednesday" style matches that carry no time intent.
func extractTime(s string, defaultHour, defaultMinute int) (int, int) {
	r, _ := nlParser.Parse(s, time.Now())
	if r != nil && hasExplicitTime(r.Text) {
		return r.Time.Hour(), r.Time.Minute()
	}
	return defaultHour, defaultMinute
}

// hasExplicitTime reports whether a matched text string contains an explicit time reference.
func hasExplicitTime(text string) bool {
	t := strings.ToLower(text)
	return strings.Contains(t, "am") || strings.Contains(t, "pm") ||
		strings.ContainsAny(t, "0123456789") && strings.Contains(t, ":")
}

var recurringKeywords = []string{
	"every", "each", "daily", "weekly", "monthly", "hourly", "weekday", "weekend",
}

var weekdayNames = []string{
	"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
	"mon", "tue", "wed", "thu", "fri", "sat", "sun",
}

func isRecurring(s string) bool {
	for _, kw := range recurringKeywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	// A bare day name (without "next"/"this") is implicitly recurring: "5pm on friday"
	if !strings.Contains(s, "next") && !strings.Contains(s, "this") && !strings.Contains(s, "tomorrow") {
		for _, day := range weekdayNames {
			if strings.Contains(s, day) {
				return true
			}
		}
	}
	return false
}

func stripRecurringKeywords(s string) string {
	r := strings.NewReplacer(
		"every ", "",
		"each ", "",
		"daily", "today",
		"weekly", "",
		"monthly", "",
		"hourly", "",
	)
	return r.Replace(s)
}

var weekdays = map[string]int{
	"sunday": 0, "sun": 0,
	"monday": 1, "mon": 1,
	"tuesday": 2, "tue": 2,
	"wednesday": 3, "wed": 3,
	"thursday": 4, "thu": 4,
	"friday": 5, "fri": 5,
	"saturday": 6, "sat": 6,
}

func namedWeekday(s string) int {
	// Longest match first to avoid "tue" matching inside "tuesday"
	order := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday",
		"sun", "mon", "tue", "wed", "thu", "fri", "sat"}
	for _, name := range order {
		if strings.Contains(s, name) {
			return weekdays[name]
		}
	}
	return -1
}
