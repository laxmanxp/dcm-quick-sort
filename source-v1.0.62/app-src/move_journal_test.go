package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBeginMoveJournalPersistsPendingBeforeFilesystemChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "move-journal.json")
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	rec, err := beginMoveJournal(path, MoveRecord{From: "C:/Downloads/a.pdf", To: "D:/Docs/a.pdf", Kind: MoveKindMove}, now)
	if err != nil {
		t.Fatal(err)
	}
	if rec.JournalID == "" {
		t.Fatal("expected durable journal id")
	}
	journal, err := loadMoveJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(journal.Entries) != 1 {
		t.Fatalf("entries=%d want 1", len(journal.Entries))
	}
	got := journal.Entries[0]
	if got.State != MoveJournalPending || got.Record.JournalID != rec.JournalID || got.Record.From != rec.From || got.Record.To != rec.To {
		t.Fatalf("unexpected pending entry: %#v", got)
	}
}

func TestCompleteMoveJournalSurvivesRestartAsUndoHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "move-journal.json")
	src := filepath.Join(dir, "old", "a.txt")
	dst := filepath.Join(dir, "new", "a.txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("moved"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 22, 8, 5, 0, 0, time.UTC)
	rec, err := beginMoveJournal(path, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := completeMoveJournal(path, rec, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	recovery, err := recoverMoveJournal(path, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery.History) != 1 || recovery.History[0].JournalID != rec.JournalID {
		t.Fatalf("history=%#v", recovery.History)
	}
}

func TestRecoverPendingMoveFinalizesWhenOnlyDestinationExists(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "done", "source.txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("moved"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "journal.json")
	rec, err := beginMoveJournal(path, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	recovery, err := recoverMoveJournal(path, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if recovery.AutoFinalized != 1 || len(recovery.History) != 1 || recovery.History[0].JournalID != rec.JournalID {
		t.Fatalf("recovery=%#v", recovery)
	}
	journal, err := loadMoveJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if journal.Entries[0].State != MoveJournalCompleted {
		t.Fatalf("state=%q", journal.Entries[0].State)
	}
}

func TestRecoverPendingMoveWithSourceOnlyMarksInterruptedWithoutChangingFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "dest.txt")
	if err := os.WriteFile(src, []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "journal.json")
	if _, err := beginMoveJournal(path, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, time.Now()); err != nil {
		t.Fatal(err)
	}

	recovery, err := recoverMoveJournal(path, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery.History) != 0 || recovery.Interrupted != 1 || len(recovery.Attention) != 0 {
		t.Fatalf("recovery=%#v", recovery)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source changed: %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("destination unexpectedly exists: %v", err)
	}
	journal, err := loadMoveJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if journal.Entries[0].State != MoveJournalFailed {
		t.Fatalf("state=%q", journal.Entries[0].State)
	}
}

func TestRecoverPendingMoveWithBothFilesNeedsAttentionAndNeverDeletesSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "dest.txt")
	if err := os.WriteFile(src, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "journal.json")
	if _, err := beginMoveJournal(path, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, time.Now()); err != nil {
		t.Fatal(err)
	}

	recovery, err := recoverMoveJournal(path, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery.Attention) != 1 || len(recovery.History) != 0 {
		t.Fatalf("recovery=%#v", recovery)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source was modified/deleted: %v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("destination was modified/deleted: %v", err)
	}
	journal, err := loadMoveJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if journal.Entries[0].State != MoveJournalAttention {
		t.Fatalf("state=%q", journal.Entries[0].State)
	}
}

func TestMarkMoveJournalUndoneRemovesItFromRestartHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.json")
	rec, err := beginMoveJournal(path, MoveRecord{From: "a", To: "b", Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := completeMoveJournal(path, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := markMoveJournalUndone(path, rec.JournalID, time.Now()); err != nil {
		t.Fatal(err)
	}
	recovery, err := recoverMoveJournal(path, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery.History) != 0 {
		t.Fatalf("undone move returned to history: %#v", recovery.History)
	}
}

func TestCorruptMoveJournalFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.json")
	if err := os.WriteFile(path, []byte("{ definitely broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := beginMoveJournal(path, MoveRecord{From: "a", To: "b", Kind: MoveKindMove}, time.Now()); err == nil {
		t.Fatal("expected corrupt journal to block a new move")
	}
}

func TestRecoverCompletedMoveRecognizesFilesystemUndoIfJournalUpdateWasLost(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "dest.txt")
	path := filepath.Join(dir, "journal.json")
	rec, err := beginMoveJournal(path, MoveRecord{From: src, To: dst, Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("moved"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := completeMoveJournal(path, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
	// Simulate a successful Undo whose final journal append was lost.
	if err := os.Rename(dst, src); err != nil {
		t.Fatal(err)
	}
	recovery, err := recoverMoveJournal(path, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery.History) != 0 {
		t.Fatalf("filesystem-undone move returned to history: %#v", recovery.History)
	}
	journal, err := loadMoveJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if journal.Entries[0].State != MoveJournalUndone {
		t.Fatalf("state=%q want undone", journal.Entries[0].State)
	}
}

func TestTruncatedFinalJournalEventFallsBackToLastDurableStateAndRepairsOnAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	rec, err := beginMoveJournal(path, MoveRecord{From: "a", To: "b", Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"id":"torn"`); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	journal, err := loadMoveJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(journal.Entries) != 1 || journal.Entries[0].ID != rec.JournalID {
		t.Fatalf("journal=%#v", journal)
	}
	if err := completeMoveJournal(path, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `{"id":"torn"`) {
		t.Fatalf("truncated tail was not repaired: %q", data)
	}
}

func TestCorruptMiddleJournalEventFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	rec, err := beginMoveJournal(path, MoveRecord{From: "a", To: "b", Kind: MoveKindMove}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{broken}\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if err := completeMoveJournal(path, rec, time.Now()); err == nil {
		t.Fatal("expected corrupt middle event to block journal update")
	}
}

func TestCompletedRenameConsumedByLaterMoveDoesNotCreateFalseRecoveryAttention(t *testing.T) {
	dir := t.TempDir()
	journalPath := filepath.Join(dir, "move-journal.jsonl")
	oldPath := filepath.Join(dir, "inbox", "report-old.pdf")
	renamedPath := filepath.Join(dir, "inbox", "Report.pdf")
	finalPath := filepath.Join(dir, "records", "Report.pdf")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, []byte("report"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 23, 9, 30, 0, 0, time.UTC)
	renameRec, err := beginMoveJournal(journalPath, MoveRecord{From: oldPath, To: renamedPath, Kind: MoveKindMove}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldPath, renamedPath); err != nil {
		t.Fatal(err)
	}
	if err := completeMoveJournal(journalPath, renameRec, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	moveRec, err := beginMoveJournal(journalPath, MoveRecord{From: renamedPath, To: finalPath, Kind: MoveKindMove}, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(renamedPath, finalPath); err != nil {
		t.Fatal(err)
	}
	if err := completeMoveJournal(journalPath, moveRec, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}

	recovery, err := recoverMoveJournal(journalPath, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery.Attention) != 0 {
		t.Fatalf("rename consumed by later move must not appear in Recovery Center: %#v", recovery.Attention)
	}
	if len(recovery.History) != 2 || recovery.History[0].JournalID != renameRec.JournalID || recovery.History[1].JournalID != moveRec.JournalID {
		t.Fatalf("rename -> move undo chain must be preserved after restart; history=%#v", recovery.History)
	}
	remaining, undoneMove, err := undoHistoryAt(recovery.History, 1)
	if err != nil {
		t.Fatalf("later Move should undo first: %v", err)
	}
	if undoneMove.JournalID != moveRec.JournalID || len(remaining) != 1 {
		t.Fatalf("unexpected first undo result: undone=%#v remaining=%#v", undoneMove, remaining)
	}
	remaining, undoneRename, err := undoHistoryAt(remaining, 0)
	if err != nil {
		t.Fatalf("Rename should become undoable after Move is undone: %v", err)
	}
	if undoneRename.JournalID != renameRec.JournalID || len(remaining) != 0 {
		t.Fatalf("unexpected second undo result: undone=%#v remaining=%#v", undoneRename, remaining)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("full undo chain did not restore original file: %v", err)
	}
}

func TestLegacyFalseRenameAttentionIsRepairedWhenLaterTransactionConsumedIt(t *testing.T) {
	dir := t.TempDir()
	journalPath := filepath.Join(dir, "move-journal.jsonl")
	oldPath := filepath.Join(dir, "inbox", "report-old.pdf")
	renamedPath := filepath.Join(dir, "inbox", "Report.pdf")
	finalPath := filepath.Join(dir, "records", "Report.pdf")
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(finalPath, []byte("report"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 23, 9, 31, 0, 0, time.UTC)
	renameRec, err := beginMoveJournal(journalPath, MoveRecord{From: oldPath, To: renamedPath, Kind: MoveKindMove}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := completeMoveJournal(journalPath, renameRec, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	moveRec, err := beginMoveJournal(journalPath, MoveRecord{From: renamedPath, To: finalPath, Kind: MoveKindMove}, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := completeMoveJournal(journalPath, moveRec, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := updateMoveJournal(journalPath, renameRec.JournalID, MoveJournalAttention, renameRec, "completed journal entry no longer has source or destination; manual review required", now.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}

	recovery, err := recoverMoveJournal(journalPath, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery.Attention) != 0 {
		t.Fatalf("legacy false rename recovery item should self-heal: %#v", recovery.Attention)
	}
	if len(recovery.History) != 2 || recovery.History[0].JournalID != renameRec.JournalID || recovery.History[1].JournalID != moveRec.JournalID {
		t.Fatalf("legacy repair must restore the rename -> move undo chain; history=%#v", recovery.History)
	}
	journal, err := loadMoveJournal(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	if journal.Entries[0].State != MoveJournalCompleted {
		t.Fatalf("legacy false attention was not repaired: %#v", journal.Entries[0])
	}
	items, err := loadRecoveryCenterItems(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("repaired Rename must disappear from Recovery Center: %#v", items)
	}
}
