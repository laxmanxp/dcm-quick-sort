package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeRecoveryMoveFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func attentionMove(t *testing.T, journalPath, src, dst string) MoveRecord {
	t.Helper()
	rec, err := beginMoveJournal(journalPath, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := failMoveJournal(journalPath, rec, MoveJournalAttention, "interrupted copy", time.Now()); err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestInspectInterruptedMoveRecognizesVerifiedPartialPrefix(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	writeRecoveryMoveFile(t, dst, "abcde")

	got, err := inspectInterruptedStandardMove(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if !got.SourceExists || !got.TargetExists || !got.Partial || !got.PrefixMatches {
		t.Fatalf("analysis=%#v", got)
	}
	if !got.CanResume || !got.CanRestart || !got.CanRemovePartial || got.CanFinish {
		t.Fatalf("actions=%#v", got)
	}
}

func TestInspectInterruptedMoveRejectsMismatchedPartialDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	writeRecoveryMoveFile(t, dst, "abXde")

	got, err := inspectInterruptedStandardMove(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if got.PrefixMatches || got.CanResume || got.CanRestart || got.CanRemovePartial || got.CanFinish {
		t.Fatalf("unsafe actions offered for mismatched file: %#v", got)
	}
}

func TestInspectInterruptedMoveRecognizesCompleteIdenticalDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	writeRecoveryMoveFile(t, dst, "abcdefghij")

	got, err := inspectInterruptedStandardMove(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ExactMatch || !got.CanFinish || got.CanResume || got.CanRemovePartial {
		t.Fatalf("analysis=%#v", got)
	}
}

func TestRecoveryResumeAppendsVerifiedRemainderThenDeletesSource(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "move-journal.jsonl")
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	writeRecoveryMoveFile(t, dst, "abcde")
	rec := attentionMove(t, journal, src, dst)

	result := executeInterruptedMoveRecovery(journal, rec.JournalID, RecoveryMoveResume, nil)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be deleted only after verification: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "abcdefghij" {
		t.Fatalf("destination=%q err=%v", data, err)
	}
	j, err := loadMoveJournal(journal)
	if err != nil {
		t.Fatal(err)
	}
	if j.Entries[0].State != MoveJournalCompleted {
		t.Fatalf("state=%q", j.Entries[0].State)
	}
}

func TestRecoveryResumeMismatchPreservesBothFiles(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "move-journal.jsonl")
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	writeRecoveryMoveFile(t, dst, "abXde")
	rec := attentionMove(t, journal, src, dst)

	result := executeInterruptedMoveRecovery(journal, rec.JournalID, RecoveryMoveResume, nil)
	if result.Err == nil {
		t.Fatal("expected mismatched partial copy to be rejected")
	}
	if got, _ := os.ReadFile(src); string(got) != "abcdefghij" {
		t.Fatalf("source changed: %q", got)
	}
	if got, _ := os.ReadFile(dst); string(got) != "abXde" {
		t.Fatalf("destination changed: %q", got)
	}
}

func TestRecoveryRestartRemovesOnlyVerifiedPartialThenCopiesFresh(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "move-journal.jsonl")
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	writeRecoveryMoveFile(t, dst, "abcde")
	rec := attentionMove(t, journal, src, dst)

	result := executeInterruptedMoveRecovery(journal, rec.JournalID, RecoveryMoveRestart, nil)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "abcdefghij" {
		t.Fatalf("destination=%q", got)
	}
}

func TestRecoveryRemovePartialLeavesSourceAndResolvesJournal(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "move-journal.jsonl")
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	writeRecoveryMoveFile(t, dst, "abcde")
	rec := attentionMove(t, journal, src, dst)

	result := executeInterruptedMoveRecovery(journal, rec.JournalID, RecoveryMoveRemovePartial, nil)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if got, _ := os.ReadFile(src); string(got) != "abcdefghij" {
		t.Fatalf("source changed: %q", got)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("partial destination still exists: %v", err)
	}
	j, err := loadMoveJournal(journal)
	if err != nil {
		t.Fatal(err)
	}
	if j.Entries[0].State != MoveJournalResolved {
		t.Fatalf("state=%q", j.Entries[0].State)
	}
}

func TestRecoveryFinishRequiresExactDestinationBeforeDeletingSource(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "move-journal.jsonl")
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	writeRecoveryMoveFile(t, dst, "abcdefghij")
	rec := attentionMove(t, journal, src, dst)

	result := executeInterruptedMoveRecovery(journal, rec.JournalID, RecoveryMoveFinish, nil)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be removed after exact verification: %v", err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "abcdefghij" {
		t.Fatalf("destination=%q", got)
	}
}

func TestSecondPowerFailureDuringResumeReturnsToAttentionWithoutDeletingFiles(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "move-journal.jsonl")
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	writeRecoveryMoveFile(t, dst, "abcdefg")
	rec, err := beginMoveJournal(journal, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := updateMoveJournalStage(journal, rec, MoveJournalStageRecoveryResumeStarted, "resume in progress", time.Now()); err != nil {
		t.Fatal(err)
	}

	recovery, err := recoverMoveJournal(journal, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery.Attention) != 1 {
		t.Fatalf("recovery=%#v", recovery)
	}
	if got, _ := os.ReadFile(src); string(got) != "abcdefghij" {
		t.Fatalf("source changed: %q", got)
	}
	if got, _ := os.ReadFile(dst); string(got) != "abcdefg" {
		t.Fatalf("partial destination changed: %q", got)
	}
}

func TestPowerFailureAfterRestartRemovedPartialLeavesSourceAuthoritative(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "move-journal.jsonl")
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, src, "abcdefghij")
	rec, err := beginMoveJournal(journal, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := updateMoveJournalStage(journal, rec, MoveJournalStageRecoveryPartialRemoved, "partial removed before fresh copy", time.Now()); err != nil {
		t.Fatal(err)
	}

	recovery, err := recoverMoveJournal(journal, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if recovery.Interrupted != 1 || len(recovery.Attention) != 0 {
		t.Fatalf("recovery=%#v", recovery)
	}
	if got, _ := os.ReadFile(src); string(got) != "abcdefghij" {
		t.Fatalf("source changed: %q", got)
	}
}

func TestPowerFailureAfterFinishDeletedSourceAutoFinalizes(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "move-journal.jsonl")
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	writeRecoveryMoveFile(t, dst, "abcdefghij")
	rec, err := beginMoveJournal(journal, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := updateMoveJournalStage(journal, rec, MoveJournalStageRecoveryFinishStarted, "exact copy verified; source removal in progress", time.Now()); err != nil {
		t.Fatal(err)
	}

	recovery, err := recoverMoveJournal(journal, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if recovery.AutoFinalized != 1 || len(recovery.History) != 1 {
		t.Fatalf("recovery=%#v", recovery)
	}
}
