package handlers

import (
	"testing"
	"time"
)

func TestDueDateSummaryAt(t *testing.T) {
	now := time.Date(2026, 6, 17, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		due  time.Time
		want string
	}{
		{name: "today", due: time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC), want: "Today · Jun 17"},
		{name: "tomorrow", due: time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC), want: "1 Day · Jun 18"},
		{name: "five days", due: time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC), want: "5 Days · Jun 22"},
		{name: "fifty nine days", due: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), want: "59 Days · Aug 15"},
		{name: "sixty days", due: time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC), want: "Aug 16"},
		{name: "yesterday", due: time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC), want: "1 Day ago · Jun 16"},
		{name: "three days ago", due: time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC), want: "3 Days ago · Jun 14"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			due := tt.due
			if got := dueDateSummaryAt(&due, now); got != tt.want {
				t.Fatalf("dueDateSummaryAt() = %q, want %q", got, tt.want)
			}
		})
	}
}
