package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJournaledReplaceSurvivesRestartWithUndoRecoveryCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in", "incoming.txt")
	dst := filepath.Join(dir, "out", "Report.txt")
	recoveryDir := filepath.Join(dir, "recovery")
	journalPath := filepath.Join(dir, "move-journal.jsonl")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, err := replaceMoveWithRecoveryJournaled(journalPath, src, dst, recoveryDir)
	if err != nil {
		t.Fatal(err)
	}
	if rec.JournalID == "" || rec.ReplacedBackup == "" {
		t.Fatalf("record=%#v", rec)
	}
	if got, _ := os.ReadFile(rec.ReplacedBackup); string(got) != "OLD" {
		t.Fatalf("backup=%q", got)
	}

	recovered, err := recoverMoveJournal(journalPath, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered.History) != 1 || recovered.History[0].Kind != MoveKindReplace {
		t.Fatalf("history=%#v", recovered.History)
	}
	if recovered.History[0].ReplacedBackup != rec.ReplacedBackup {
		t.Fatalf("backup lost across restart: %#v", recovered.History[0])
	}
	if _, err := os.Stat(rec.ReplacedBackup); err != nil {
		t.Fatalf("recovery copy missing after restart: %v", err)
	}
}

func TestRecoverPendingReplaceRestoresOldDestinationWhenCrashAfterRemoval(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "incoming.txt")
	dst := filepath.Join(dir, "Report.txt")
	backup := filepath.Join(dir, "recovery", "Report.txt.replaced.1")
	journalPath := filepath.Join(dir, "journal.jsonl")
	if err := os.WriteFile(src, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backup, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, err := beginMoveJournal(journalPath, MoveRecord{From: src, To: dst, Kind: MoveKindReplace}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	rec.ReplacedBackup = backup
	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageReplaceBackupCreated, "backup verified", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dst); err != nil {
		t.Fatal(err)
	}
	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageReplaceDestinationRemoved, "old destination removed", time.Now()); err != nil {
		t.Fatal(err)
	}

	recovered, err := recoverMoveJournal(journalPath, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered.History) != 0 || recovered.Interrupted != 1 || len(recovered.Attention) != 0 {
		t.Fatalf("recovery=%#v", recovered)
	}
	if got, _ := os.ReadFile(src); string(got) != "NEW" {
		t.Fatalf("source=%q", got)
	}
	if got, _ := os.ReadFile(dst); string(got) != "OLD" {
		t.Fatalf("destination not restored: %q", got)
	}
}

func TestRecoverPendingReplaceFinalizesWhenReplacementAlreadyCommitted(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "incoming.txt")
	dst := filepath.Join(dir, "Report.txt")
	backup := filepath.Join(dir, "recovery", "Report.txt.replaced.1")
	journalPath := filepath.Join(dir, "journal.jsonl")
	if err := os.WriteFile(src, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backup, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, err := beginMoveJournal(journalPath, MoveRecord{From: src, To: dst, Kind: MoveKindReplace}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	rec.ReplacedBackup = backup
	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageReplaceDestinationRemoved, "old destination removed", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dst); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}

	recovered, err := recoverMoveJournal(journalPath, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if recovered.AutoFinalized != 1 || len(recovered.History) != 1 || recovered.History[0].Kind != MoveKindReplace {
		t.Fatalf("recovery=%#v", recovered)
	}
	if got, _ := os.ReadFile(dst); string(got) != "NEW" {
		t.Fatalf("destination=%q", got)
	}
	if got, _ := os.ReadFile(backup); string(got) != "OLD" {
		t.Fatalf("backup=%q", got)
	}
}

func TestJournaledExactDuplicateRemovalSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "incoming.txt")
	dst := filepath.Join(dir, "Report.txt")
	journalPath := filepath.Join(dir, "journal.jsonl")
	if err := os.WriteFile(src, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, err := removeExactDuplicateSourceJournaled(journalPath, src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if rec.JournalID == "" {
		t.Fatalf("record=%#v", rec)
	}
	recovered, err := recoverMoveJournal(journalPath, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered.History) != 1 || recovered.History[0].Kind != MoveKindDuplicateRemoved {
		t.Fatalf("history=%#v", recovered.History)
	}
}

func TestRecoverCompletedDuplicateRecognizesUndoIfJournalAppendWasLost(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "incoming.txt")
	dst := filepath.Join(dir, "Report.txt")
	journalPath := filepath.Join(dir, "journal.jsonl")
	if err := os.WriteFile(src, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec, err := removeExactDuplicateSourceJournaled(journalPath, src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if err := copyFileVerifiedHash(dst, src); err != nil {
		t.Fatal(err)
	}

	recovered, err := recoverMoveJournal(journalPath, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered.History) != 0 {
		t.Fatalf("history=%#v", recovered.History)
	}
	journal, err := loadMoveJournal(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(journal.Entries) != 1 || journal.Entries[0].ID != rec.JournalID || journal.Entries[0].State != MoveJournalUndone {
		t.Fatalf("journal=%#v", journal.Entries)
	}
}

func TestCorruptJournalBlocksDuplicateRemovalBeforeSourceDeletion(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "incoming.txt")
	dst := filepath.Join(dir, "Report.txt")
	journalPath := filepath.Join(dir, "journal.jsonl")
	if err := os.WriteFile(src, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journalPath, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := removeExactDuplicateSourceJournaled(journalPath, src, dst); err == nil {
		t.Fatal("expected journal failure")
	}
	if got, _ := os.ReadFile(src); string(got) != "SAME" {
		t.Fatalf("source changed: %q", got)
	}
}

func TestCorruptJournalBlocksReplaceBeforeAnyDestinationChange(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "incoming.txt")
	dst := filepath.Join(dir, "Report.txt")
	journalPath := filepath.Join(dir, "journal.jsonl")
	recoveryDir := filepath.Join(dir, "recovery")
	if err := os.WriteFile(src, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journalPath, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := replaceMoveWithRecoveryJournaled(journalPath, src, dst, recoveryDir); err == nil {
		t.Fatal("expected journal failure")
	}
	if got, _ := os.ReadFile(src); string(got) != "NEW" {
		t.Fatalf("source changed: %q", got)
	}
	if got, _ := os.ReadFile(dst); string(got) != "OLD" {
		t.Fatalf("destination changed: %q", got)
	}
}

func TestRecoverCompletedReplaceFinishesInterruptedUndo(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "incoming.txt")
	dst := filepath.Join(dir, "Report.txt")
	journalPath := filepath.Join(dir, "journal.jsonl")
	recoveryDir := filepath.Join(dir, "recovery")
	if err := os.WriteFile(src, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec, err := replaceMoveWithRecoveryJournaled(journalPath, src, dst, recoveryDir)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate crash after the first destructive Undo step: replacement moved back
	// to source, but previous destination not restored and journal not updated.
	if err := os.Rename(dst, src); err != nil {
		t.Fatal(err)
	}

	recovered, err := recoverMoveJournal(journalPath, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered.History) != 0 {
		t.Fatalf("history=%#v", recovered.History)
	}
	if got, _ := os.ReadFile(src); string(got) != "NEW" {
		t.Fatalf("source=%q", got)
	}
	if got, _ := os.ReadFile(dst); string(got) != "OLD" {
		t.Fatalf("destination=%q", got)
	}
	if _, err := os.Stat(rec.ReplacedBackup); !os.IsNotExist(err) {
		t.Fatalf("backup should be consumed after recovered Undo, err=%v", err)
	}
	journal, err := loadMoveJournal(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	if journal.Entries[0].State != MoveJournalUndone {
		t.Fatalf("state=%q", journal.Entries[0].State)
	}
}

func TestRecoverPendingDuplicateFinalizesIfSourceDeletionCommittedBeforeJournalAppend(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "incoming.txt")
	dst := filepath.Join(dir, "Report.txt")
	journalPath := filepath.Join(dir, "journal.jsonl")
	if err := os.WriteFile(src, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}
	originalHash, err := fileSHA256(src)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := beginMoveJournal(journalPath, MoveRecord{From: src, To: dst, Kind: MoveKindDuplicateRemoved, OriginalSHA256: originalHash, OriginalSize: 4}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageDuplicateVerified, "verified", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(src); err != nil {
		t.Fatal(err)
	}

	recovered, err := recoverMoveJournal(journalPath, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if recovered.AutoFinalized != 1 || len(recovered.History) != 1 || recovered.History[0].Kind != MoveKindDuplicateRemoved {
		t.Fatalf("recovery=%#v", recovered)
	}
}
