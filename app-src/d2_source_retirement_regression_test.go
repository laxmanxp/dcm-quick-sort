package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestD2BackgroundMoveRefusesChangedFilesBeforeSourceRetirement(t *testing.T) {
	for _, worker := range []string{"plain", "progress"} {
		for _, change := range []string{"source-content", "source-identity", "destination-content"} {
			t.Run(worker+"/"+change, func(t *testing.T) {
				root := t.TempDir()
				source := filepath.Join(root, "source.bin")
				target := filepath.Join(root, "target.bin")
				if err := os.WriteFile(source, []byte("ORIGINAL"), 0o600); err != nil {
					t.Fatal(err)
				}
				originalInfo, err := os.Stat(source)
				if err != nil {
					t.Fatal(err)
				}
				wantSource, wantTarget := "ORIGINAL", "ORIGINAL"
				var mutationErr error
				mutate := func() error {
					switch change {
					case "source-content":
						wantSource = "MODIFIED"
						if err := os.WriteFile(source, []byte(wantSource), 0o600); err != nil {
							return err
						}
						// Size and modification time alone cannot establish that
						// the copied file is still the file being retired.
						return os.Chtimes(source, originalInfo.ModTime(), originalInfo.ModTime())
					case "source-identity":
						if err := os.Rename(source, filepath.Join(root, "original-retained.bin")); err != nil {
							return err
						}
						if err := os.WriteFile(source, []byte(wantSource), 0o600); err != nil {
							return err
						}
						return os.Chtimes(source, originalInfo.ModTime(), originalInfo.ModTime())
					default:
						wantTarget = "CHANGED!"
						return os.WriteFile(target, []byte(wantTarget), 0o600)
					}
				}
				copyThenChange := func(source, target string, progress CopyProgressFunc) error {
					if err := copyFileVerifiedWithProgress(source, target, progress); err != nil {
						return err
					}
					mutationErr = mutate()
					return mutationErr
				}

				var got BackgroundMoveResult
				if worker == "plain" {
					got = executeBackgroundCopyMove(source, target, func(source, target string) error {
						return copyThenChange(source, target, nil)
					}, retireVerifiedSource)
				} else {
					results := startBackgroundCopyMoveWithProgress(source, target, copyThenChange, retireVerifiedSource, nil)
					select {
					case got = <-results:
					case <-time.After(10 * time.Second):
						t.Fatal("background copy did not finish")
					}
				}
				if mutationErr != nil {
					t.Fatalf("could not reproduce the edit after copying: %v", mutationErr)
				}
				if got.Status != BackgroundMoveCopiedSourceRetained || got.Err == nil {
					t.Errorf("changed file was reported as a completed move: status=%q err=%v", got.Status, got.Err)
				}
				if got.Record != (MoveRecord{}) {
					t.Errorf("refused retirement produced a successful move record: %#v", got.Record)
				}
				assertD2FileBytes(t, source, wantSource)
				assertD2FileBytes(t, target, wantTarget)
				if change == "source-identity" {
					assertD2FileBytes(t, filepath.Join(root, "original-retained.bin"), "ORIGINAL")
				}
			})
		}
	}
}

func TestD2RecoveryRestartRetainsSourceEditedAtCopyCompletion(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.bin"), filepath.Join(root, "target.bin")
	journal := filepath.Join(root, "journal.jsonl")
	writeRecoveryMoveFile(t, source, "abcdefghij")
	writeRecoveryMoveFile(t, target, "abcde")
	rec := attentionMove(t, journal, source, target)
	mutated := false
	var mutationErr error
	result := executeInterruptedMoveRecovery(journal, rec.JournalID, RecoveryMoveRestart, func(copied, total int64) {
		if !mutated && total > 0 && copied == total {
			mutated = true
			mutationErr = os.WriteFile(source, []byte("ABCDEFGHIJ"), 0o600)
		}
	})
	if !mutated || mutationErr != nil {
		t.Fatalf("source edit was not reproduced: attempted=%v err=%v", mutated, mutationErr)
	}
	if result.Err == nil {
		t.Error("recovery deleted a source changed after its bytes were copied")
	}
	assertD2FileBytes(t, source, "ABCDEFGHIJ")
	assertD2FileBytes(t, target, "abcdefghij")
	assertD2RecoveryAttention(t, journal, rec.JournalID)
}

func TestD2RecoveryCompletionRejectsStaleVerification(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.bin"), filepath.Join(root, "target.bin")
	journal := filepath.Join(root, "journal.jsonl")
	writeRecoveryMoveFile(t, source, "abcdefghij")
	writeRecoveryMoveFile(t, target, "abcdefghij")
	rec := attentionMove(t, journal, source, target)
	analysis, err := inspectInterruptedStandardMove(source, target)
	if err != nil || !analysis.CanFinish {
		t.Fatalf("fixture did not reach verified Finish boundary: analysis=%#v err=%v", analysis, err)
	}
	expected, err := snapshotSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("ABCDEFGHIJ"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := completeRecoveredMoveAfterVerifiedDestination(journal, rec, "previous verification is now stale", expected); err == nil {
		t.Error("recovery trusted stale verification and removed the edited source")
	}
	assertD2FileBytes(t, source, "ABCDEFGHIJ")
	assertD2FileBytes(t, target, "abcdefghij")
	assertD2RecoveryAttention(t, journal, rec.JournalID)
}

func assertD2FileBytes(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Errorf("file %s = %q, err=%v; want preserved bytes %q", path, got, err, want)
	}
}

func assertD2RecoveryAttention(t *testing.T, journal, id string) {
	t.Helper()
	entry, err := moveJournalEntryByID(journal, id)
	if err != nil {
		t.Fatal(err)
	}
	if entry.State != MoveJournalAttention {
		t.Errorf("refused source retirement journal state=%q, want attention", entry.State)
	}
}
