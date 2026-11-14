package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecoveryCenterListsAttentionFailedAndReviewedNewestFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "move-journal.jsonl")
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)

	attention, err := beginMoveJournal(path, MoveRecord{From: "a", To: "b", Kind: MoveKindMove}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := failMoveJournal(path, attention, MoveJournalAttention, "needs review", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	failed, err := beginMoveJournal(path, MoveRecord{From: "c", To: "d", Kind: MoveKindMove}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := failMoveJournal(path, failed, MoveJournalFailed, "source retained", now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}

	if err := markRecoveryEntryReviewed(path, failed.JournalID, now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}

	items, err := loadRecoveryCenterItems(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items=%#v", items)
	}
	if items[0].ID != failed.JournalID || items[0].State != MoveJournalResolved || !items[0].Reviewed {
		t.Fatalf("newest item=%#v", items[0])
	}
	if items[1].ID != attention.JournalID || items[1].State != MoveJournalAttention || !items[1].CanMarkReviewed {
		t.Fatalf("attention item=%#v", items[1])
	}
}

func TestMarkRecoveryEntryReviewedDoesNotChangeFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "dest.txt")
	path := filepath.Join(dir, "move-journal.jsonl")
	if err := os.WriteFile(src, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("dest"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, err := beginMoveJournal(path, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := failMoveJournal(path, rec, MoveJournalAttention, "manual review", time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := markRecoveryEntryReviewed(path, rec.JournalID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(src); string(got) != "source" {
		t.Fatalf("source=%q", got)
	}
	if got, _ := os.ReadFile(dst); string(got) != "dest" {
		t.Fatalf("dest=%q", got)
	}

	journal, err := loadMoveJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if journal.Entries[0].State != MoveJournalResolved {
		t.Fatalf("state=%q", journal.Entries[0].State)
	}
}

func TestCleanupResolvedRecoveryArtifactsOnlyDeletesRedundantOldBackup(t *testing.T) {
	dir := t.TempDir()
	recoveryRoot := filepath.Join(dir, "recovery")
	path := filepath.Join(dir, "move-journal.jsonl")
	src := filepath.Join(dir, "incoming.txt")
	dst := filepath.Join(dir, "report.txt")
	backup := filepath.Join(recoveryRoot, "report.txt.replaced.1")
	if err := os.MkdirAll(recoveryRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("incoming"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backup, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	old := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	rec, err := beginMoveJournal(path, MoveRecord{From: src, To: dst, Kind: MoveKindReplace, ReplacedBackup: backup}, old)
	if err != nil {
		t.Fatal(err)
	}
	if err := failMoveJournal(path, rec, MoveJournalAttention, "review", old.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := markRecoveryEntryReviewed(path, rec.JournalID, old.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}

	summary, err := cleanupResolvedRecoveryArtifacts(path, recoveryRoot, 30*24*time.Hour, time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if summary.Removed != 1 || summary.Skipped != 0 {
		t.Fatalf("summary=%#v", summary)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("backup still exists: %v", err)
	}
}

func TestCleanupResolvedRecoveryArtifactsPreservesNonRedundantOrUnresolvedBackup(t *testing.T) {
	dir := t.TempDir()
	recoveryRoot := filepath.Join(dir, "recovery")
	path := filepath.Join(dir, "move-journal.jsonl")
	if err := os.MkdirAll(recoveryRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)

	makeEntry := func(name string, state MoveJournalState, destBytes, backupBytes string) string {
		src := filepath.Join(dir, name+"-src")
		dst := filepath.Join(dir, name+"-dst")
		backup := filepath.Join(recoveryRoot, name+".replaced.1")
		_ = os.WriteFile(src, []byte("src"), 0o644)
		_ = os.WriteFile(dst, []byte(destBytes), 0o644)
		_ = os.WriteFile(backup, []byte(backupBytes), 0o644)
		rec, err := beginMoveJournal(path, MoveRecord{From: src, To: dst, Kind: MoveKindReplace, ReplacedBackup: backup}, now.Add(-60*24*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if err := failMoveJournal(path, rec, state, "review", now.Add(-59*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
		if state != MoveJournalAttention {
			return backup
		}
		return backup
	}

	unresolved := makeEntry("unresolved", MoveJournalAttention, "same", "same")
	nonredundant := makeEntry("nonredundant", MoveJournalAttention, "new", "old")
	journal, _ := loadMoveJournal(path)
	if err := markRecoveryEntryReviewed(path, journal.Entries[1].ID, now.Add(-58*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	summary, err := cleanupResolvedRecoveryArtifacts(path, recoveryRoot, 30*24*time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Removed != 0 || summary.Skipped == 0 {
		t.Fatalf("summary=%#v", summary)
	}
	if _, err := os.Stat(unresolved); err != nil {
		t.Fatalf("unresolved backup removed: %v", err)
	}
	if _, err := os.Stat(nonredundant); err != nil {
		t.Fatalf("nonredundant backup removed: %v", err)
	}
}
