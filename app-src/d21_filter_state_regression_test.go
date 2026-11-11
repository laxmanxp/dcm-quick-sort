package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// A wiring check is kept separate from the native screenshot/interaction gate.
func TestD21EmptyStateUsesFilterContext(t *testing.T) {
	data, err := os.ReadFile("main_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.ReplaceAll(string(data), "\r\n", "\n")
	start := strings.Index(source, "func showCurrent(")
	end := strings.Index(source, "func nextFile(")
	if start < 0 || end <= start {
		t.Fatal("cannot locate showCurrent")
	}
	body := source[start:end]
	if strings.Contains(body, `app.fallback = "Folder complete"`) {
		t.Fatal("DCM-UI-EMPTY-COPY-01: showCurrent reports Folder complete without checking filters")
	}
	for _, want := range []string{
		`inboxEmptyState(selectedInboxFilter(), len(app.files))`,
		`setText(app.currentLabel, "PREVIEW  ·  "+heading)`,
		`app.fallback = hint`,
		`setText(app.nameEdit, "")`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing empty-state wiring %q", want)
		}
	}
}

func TestD21AgeAndSizeBoundaryContract(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		age     InboxAgeFilter
		elapsed time.Duration
		want    bool
	}{
		{"all-old", InboxAgeAll, 400 * 24 * time.Hour, true},
		{"24h-edge", InboxAgeLast24Hours, 24 * time.Hour, true},
		{"24h-outside", InboxAgeLast24Hours, 24*time.Hour + time.Nanosecond, false},
		{"7d-edge", InboxAgeLast7Days, 7 * 24 * time.Hour, true},
		{"7d-outside", InboxAgeLast7Days, 7*24*time.Hour + time.Nanosecond, false},
		{"30d-edge", InboxAgeLast30Days, 30 * 24 * time.Hour, true},
		{"30d-outside", InboxAgeLast30Days, 30*24*time.Hour + time.Nanosecond, false},
		{"older-not-edge", InboxAgeOlder30Days, 30 * 24 * time.Hour, false},
		{"older-after-edge", InboxAgeOlder30Days, 30*24*time.Hour + time.Nanosecond, true},
		{"future-is-recent", InboxAgeLast24Hours, -time.Hour, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := inboxAgeMatches(now.Add(-tc.elapsed), tc.age, now); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
	const mib = int64(1024 * 1024)
	for _, tc := range []struct {
		name  string
		size  InboxSizeFilter
		bytes int64
		want  bool
	}{
		{"all-large", InboxSizeAll, 200 * mib, true},
		{"under-before", InboxSizeUnder1MiB, mib - 1, true},
		{"under-edge", InboxSizeUnder1MiB, mib, false},
		{"small-start", InboxSize1To10MiB, mib, true},
		{"small-before-end", InboxSize1To10MiB, 10*mib - 1, true},
		{"small-end", InboxSize1To10MiB, 10 * mib, false},
		{"medium-start", InboxSize10To100MiB, 10 * mib, true},
		{"medium-before-end", InboxSize10To100MiB, 100*mib - 1, true},
		{"medium-end", InboxSize10To100MiB, 100 * mib, false},
		{"large-before", InboxSize100MiBPlus, 100*mib - 1, false},
		{"large-start", InboxSize100MiBPlus, 100 * mib, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := inboxSizeMatches(tc.bytes, tc.size); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
