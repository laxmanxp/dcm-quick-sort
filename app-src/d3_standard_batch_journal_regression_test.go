//go:build windows

package main

import (
	"path/filepath"
	"testing"
	"time"
)

// Check ordinary move membership at the journal boundary independently of any
// UI completion callback. All mutations use disposable fixtures.
func TestD3StandardBatchJournalSurvivesInterruption(t *testing.T) {
	for _, phase := range []string{"before-move", "after-move-before-completion", "completed"} {
		t.Run(phase, func(t *testing.T) {
			root := t.TempDir()
			journalPath := filepath.Join(root, "journal.jsonl")
			source := filepath.Join(root, "incoming.txt")
			target := filepath.Join(root, "sorted", "incoming.txt")
			const original = "original batch file bytes"
			const batchID = "standard-batch-admitted-before-move"
			d16Write(t, source, original)
			rec, err := beginMoveJournal(journalPath, MoveRecord{
				From: source, To: target, Kind: MoveKindMove, BatchID: batchID,
			}, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			journal, err := loadMoveJournal(journalPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(journal.Entries) != 1 || journal.Entries[0].State != MoveJournalPending ||
				journal.Entries[0].Record.BatchID != batchID || journal.Entries[0].ID != rec.JournalID {
				t.Fatalf("admission did not persist the original batch: %#v", journal)
			}
			d16WantBytes(t, source, original)
			d16WantAbsent(t, target)

			if phase != "before-move" {
				if err := moveFile(source, target); err != nil {
					t.Fatal(err)
				}
				d16WantAbsent(t, source)
				d16WantBytes(t, target, original)
			}
			if phase == "completed" {
				if err := completeMoveJournal(journalPath, rec, time.Now()); err != nil {
					t.Fatal(err)
				}
			}
			// In the middle case there is deliberately no completion append and
			// no UI callback; recovery has only the pending batch admission.
			recovered, err := recoverMoveJournal(journalPath, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if len(recovered.Attention) != 0 {
				t.Fatalf("ordinary fixture unexpectedly needs attention: %#v", recovered)
			}
			if phase == "before-move" {
				if recovered.Interrupted != 1 || recovered.AutoFinalized != 0 || len(recovered.History) != 0 {
					t.Fatalf("unstarted move was reported as completed: %#v", recovered)
				}
				d16WantBytes(t, source, original)
				d16WantAbsent(t, target)
				return
			}
			wantFinalized := 0
			if phase == "after-move-before-completion" {
				wantFinalized = 1
			}
			if recovered.AutoFinalized != wantFinalized || recovered.Interrupted != 0 {
				t.Fatalf("incorrect ordinary recovery outcome: %#v", recovered)
			}
			assertD3RecoveredBatch(t, recovered, MoveKindMove, batchID)
			if recovered.History[0].JournalID != rec.JournalID {
				t.Fatal("recovery substituted a different transaction")
			}
			d16WantAbsent(t, source)
			d16WantBytes(t, target, original)
		})
	}
}
