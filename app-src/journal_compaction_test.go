package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompactMoveJournalKeepsLatestStateAndShrinksEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "move-journal.jsonl")
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	rec, err := beginMoveJournal(path, MoveRecord{From: "a", To: "b", Kind: MoveKindMove}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := updateMoveJournalStage(path, rec, MoveJournalStageStarted, "stage", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := completeMoveJournal(path, rec, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := markMoveJournalUndone(path, rec.JournalID, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}

	before, err := countMoveJournalEvents(path)
	if err != nil {
		t.Fatal(err)
	}
	if before < 4 {
		t.Fatalf("before=%d", before)
	}

	summary, err := compactMoveJournal(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Compacted || summary.BeforeEvents != before || summary.AfterEvents != 1 {
		t.Fatalf("summary=%#v", summary)
	}
	journal, err := loadMoveJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(journal.Entries) != 1 || journal.Entries[0].State != MoveJournalUndone {
		t.Fatalf("journal=%#v", journal.Entries)
	}
}

func TestCompactMoveJournalSkipsBelowThreshold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "move-journal.jsonl")
	if _, err := beginMoveJournal(path, MoveRecord{From: "a", To: "b", Kind: MoveKindMove}, time.Now()); err != nil {
		t.Fatal(err)
	}
	summary, err := compactMoveJournal(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Compacted {
		t.Fatalf("summary=%#v", summary)
	}
}

func TestLoadMoveJournalRestoresCompactionBackupIfActiveFileMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "move-journal.jsonl")
	rec, err := beginMoveJournal(path, MoveRecord{From: "a", To: "b", Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	backup := path + ".compactbak.123"
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}

	journal, err := loadMoveJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(journal.Entries) != 1 || journal.Entries[0].ID != rec.JournalID {
		t.Fatalf("journal=%#v", journal)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("active journal not restored: %v", err)
	}
}
