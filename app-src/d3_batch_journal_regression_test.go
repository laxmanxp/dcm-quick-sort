//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Exercise the real Windows admission path without running its completion
// callback. A crash after the worker changes the files must not lose the batch
// membership that Undo Batch needs on the next launch.
func TestD3ConflictBatchJournalPrecedesUICompletion(t *testing.T) {
	for _, tc := range []struct {
		name        string
		action      BackgroundConflictActionKind
		kind        MoveKind
		sourceBytes string
		targetBytes string
	}{
		{"replace", BackgroundConflictReplace, MoveKindReplace, "NEW", "OLD"},
		{"duplicate", BackgroundConflictRemoveDuplicate, MoveKindDuplicateRemoved, "SAME", "SAME"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("LOCALAPPDATA", filepath.Join(root, "configuration"))
			source := filepath.Join(root, "incoming.txt")
			target := filepath.Join(root, "destination.txt")
			if err := os.WriteFile(source, []byte(tc.sourceBytes), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte(tc.targetBytes), 0o600); err != nil {
				t.Fatal(err)
			}

			previousApp, previousResults := app, backgroundConflictActionResults
			t.Cleanup(func() {
				app = previousApp
				backgroundConflictActionResults = previousResults
			})
			const batchID = "batch-admitted-before-files-change"
			app = appState{
				index:                -1,
				batchMoveActive:      true,
				batchMoveID:          batchID,
				pendingFileOperation: fileOperationMove,
			}
			backgroundConflictActionResults = make(chan BackgroundConflictActionResult, 1)

			// All window handles are zero. The operation still follows the normal
			// worker path, and every file it can change is inside this fixture.
			beginBackgroundConflictAction(tc.action, source, target)
			var result BackgroundConflictActionResult
			select {
			case result = <-backgroundConflictActionResults:
			case <-time.After(10 * time.Second):
				t.Fatal("conflict worker did not finish")
			}
			if result.Err != nil {
				t.Fatal(result.Err)
			}
			if result.Record.BatchID != batchID {
				t.Errorf("worker record lost admitted batch: got %q, want %q", result.Record.BatchID, batchID)
			}

			journalPath := moveJournalFilePath()
			data, err := os.ReadFile(journalPath)
			if err != nil {
				t.Fatal(err)
			}
			var pendingOnly bytes.Buffer
			events := 0
			for _, line := range bytes.Split(data, []byte{'\n'}) {
				if len(bytes.TrimSpace(line)) == 0 {
					continue
				}
				var entry MoveJournalEntry
				if err := json.Unmarshal(line, &entry); err != nil {
					t.Fatal(err)
				}
				events++
				if events == 1 && (entry.State != MoveJournalPending || entry.Stage != MoveJournalStageStarted) {
					t.Errorf("first durable event is not the operation admission: %#v", entry)
				}
				if entry.Record.BatchID != batchID {
					t.Errorf("durable event %d (%s/%s) lost batch: got %q, want %q", events, entry.State, entry.Stage, entry.Record.BatchID, batchID)
				}
				if entry.State == MoveJournalPending {
					pendingOnly.Write(line)
					pendingOnly.WriteByte('\n')
				}
			}
			if events == 0 || pendingOnly.Len() == 0 {
				t.Fatal("conflict action did not persist its pending journal events")
			}

			// Do not call finishBackgroundConflictAction: that callback is too
			// late to establish metadata needed after a crash or shutdown.
			recovered, err := recoverMoveJournal(journalPath, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			assertD3RecoveredBatch(t, recovered, tc.kind, batchID)

			// Reproduce loss of the terminal completion append as well. Existing
			// durable conflict stages must recover the same admitted batch.
			interruptedPath := filepath.Join(root, "interrupted-journal.jsonl")
			if err := os.WriteFile(interruptedPath, pendingOnly.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			recovered, err = recoverMoveJournal(interruptedPath, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if recovered.AutoFinalized != 1 {
				t.Errorf("committed conflict did not recover from its pending events: %#v", recovered)
			}
			assertD3RecoveredBatch(t, recovered, tc.kind, batchID)
		})
	}
}

func assertD3RecoveredBatch(t *testing.T, recovered MoveJournalRecovery, kind MoveKind, batchID string) {
	t.Helper()
	if len(recovered.History) != 1 {
		t.Fatalf("recovered history = %#v, want one completed conflict", recovered.History)
	}
	if recovered.History[0].Kind != kind || recovered.History[0].BatchID != batchID {
		t.Errorf("recovered conflict lost kind or batch: %#v", recovered.History[0])
	}
	if indexes := batchHistoryIndexes(recovered.History, batchID); len(indexes) != 1 {
		t.Errorf("Undo Batch cannot find recovered conflict: indexes = %v", indexes)
	}
}
