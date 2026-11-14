package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func processedFixture(from, to string, at time.Time) MoveRecord {
	return MoveRecord{
		From:      from,
		To:        to,
		Kind:      MoveKindMove,
		JournalID:  journalIDForProcessedTime(at),
	}
}

func TestProcessedViewGroupsByDateAndStartsCollapsed(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local)
	history := []MoveRecord{
		processedFixture("a.txt", "A/a.txt", now.Add(-2*time.Hour)),
		processedFixture("b.txt", "B/b.txt", now.Add(-26*time.Hour)),
		processedFixture("c.txt", "C/c.txt", now.Add(-time.Hour)),
	}
	collapsed := map[string]bool{}
	ensureProcessedGroupState(history, collapsed)
	rows := buildProcessedView(history, "", collapsed, now)
	if len(rows) != 2 {
		t.Fatalf("collapsed date groups should show two headers only, got %#v", rows)
	}
	if !rows[0].Header || !strings.Contains(rows[0].Label, "23 Sep 2026") || !strings.Contains(rows[0].Label, "(2)") {
		t.Fatalf("newest group header mismatch: %#v", rows[0])
	}
	if !rows[1].Header || !strings.Contains(rows[1].Label, "22 Sep 2026") {
		t.Fatalf("older group header mismatch: %#v", rows[1])
	}
}

func TestProcessedViewExpandsGroupAndMapsRowsToHistory(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local)
	history := []MoveRecord{
		processedFixture("older.txt", "D/older.txt", now.Add(-3*time.Hour)),
		processedFixture("newer.txt", "D/newer.txt", now.Add(-time.Hour)),
	}
	key := processedDateKey(history[0], now)
	collapsed := map[string]bool{key: false}
	rows := buildProcessedView(history, "", collapsed, now)
	if len(rows) != 3 || !rows[0].Header {
		t.Fatalf("expanded group rows=%#v", rows)
	}
	if rows[1].HistoryIndex != 1 || rows[2].HistoryIndex != 0 {
		t.Fatalf("records should be newest-first within date group: %#v", rows)
	}
}

func TestProcessedSearchIgnoresCollapsedStateAndMatchesPaths(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local)
	history := []MoveRecord{
		processedFixture(filepath.Join("Downloads", "invoice.pdf"), filepath.Join("Accounts", "Invoice.pdf"), now),
		processedFixture(filepath.Join("Downloads", "photo.jpg"), filepath.Join("Photos", "photo.jpg"), now),
	}
	collapsed := map[string]bool{}
	ensureProcessedGroupState(history, collapsed)
	rows := buildProcessedView(history, "accounts", collapsed, now)
	if len(rows) != 2 || !rows[0].Header || rows[1].HistoryIndex != 0 {
		t.Fatalf("search should show matching record even in collapsed group: %#v", rows)
	}
}

func TestProcessedUnknownDateUsesEarlierGroup(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local)
	history := []MoveRecord{{From: "legacy.txt", To: "archive/legacy.txt", Kind: MoveKindMove}}
	collapsed := map[string]bool{}
	ensureProcessedGroupState(history, collapsed)
	rows := buildProcessedView(history, "", collapsed, now)
	if len(rows) != 1 || !strings.Contains(rows[0].Label, "Earlier / unknown date") {
		t.Fatalf("legacy group mismatch: %#v", rows)
	}
}
