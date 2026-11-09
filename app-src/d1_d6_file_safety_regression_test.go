package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func d16Write(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func d16WantBytes(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("file %s: got %q, err=%v; want %q", path, got, err, want)
	}
}

func d16WantAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("path must remain absent: %s, err=%v", path, err)
	}
}

func TestD1ConflictActionsRefuseSameFileIdentity(t *testing.T) {
	for _, alias := range []string{"same-path", "hard-link", "case-alias"} {
		for _, operation := range []string{"inspect", "duplicate", "journaled-duplicate", "replace", "journaled-replace"} {
			t.Run(alias+"/"+operation, func(t *testing.T) {
				if alias == "case-alias" && runtime.GOOS != "windows" {
					t.Skip("Windows case-insensitive path contract")
				}
				root := t.TempDir()
				source := filepath.Join(root, "original.txt")
				d16Write(t, source, "the only original bytes")
				target := source
				if alias == "hard-link" {
					target = filepath.Join(root, "hard-link.txt")
					if err := os.Link(source, target); err != nil {
						t.Fatal(err)
					}
				} else if alias == "case-alias" {
					target = filepath.Join(root, "ORIGINAL.TXT")
				}
				var err error
				switch operation {
				case "inspect":
					_, err = inspectDestinationConflict(source, target)
				case "duplicate":
					_, err = removeExactDuplicateSource(source, target)
				case "journaled-duplicate":
					_, err = removeExactDuplicateSourceJournaled(filepath.Join(root, "journal.jsonl"), source, target)
				case "replace":
					_, err = replaceMoveWithRecovery(source, target, filepath.Join(root, "recovery"))
				case "journaled-replace":
					_, err = replaceMoveWithRecoveryJournaled(filepath.Join(root, "journal.jsonl"), source, target, filepath.Join(root, "recovery"))
				}
				if err == nil {
					t.Error("conflict action accepted two paths for the same physical file")
				}
				d16WantBytes(t, source, "the only original bytes")
				d16WantBytes(t, target, "the only original bytes")
			})
		}
	}
}

func d6RemoveFixture(t *testing.T, contents string) (MoveRecord, string) {
	t.Helper()
	root := t.TempDir()
	source, target := filepath.Join(root, "source.txt"), filepath.Join(root, "destination.txt")
	d16Write(t, source, contents)
	d16Write(t, target, contents)
	journal := filepath.Join(root, "journal.jsonl")
	rec, err := removeExactDuplicateSourceJournaled(journal, source, target)
	if err != nil {
		t.Fatal(err)
	}
	d16WantAbsent(t, source)
	return rec, journal
}

func TestD6DuplicateUndoRefusesChangedDestination(t *testing.T) {
	for _, journaled := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "journaled"}[journaled], func(t *testing.T) {
			root := t.TempDir()
			source, target := filepath.Join(root, "source.txt"), filepath.Join(root, "target.txt")
			d16Write(t, source, "ORIGINAL")
			d16Write(t, target, "ORIGINAL")
			var rec MoveRecord
			var err error
			if journaled {
				rec, err = removeExactDuplicateSourceJournaled(filepath.Join(root, "journal.jsonl"), source, target)
			} else {
				rec, err = removeExactDuplicateSource(source, target)
			}
			if err != nil {
				t.Fatal(err)
			}
			d16Write(t, target, "MODIFIED") // Same size: a size-only guard cannot pass.
			if err := undoMove(rec); err == nil {
				t.Error("undo restored changed destination bytes as the deleted original")
			}
			d16WantAbsent(t, source)
			d16WantBytes(t, target, "MODIFIED")
		})
	}
}

func TestD6DuplicateFingerprintIsDurableBeforeDeletion(t *testing.T) {
	const contents = "ORIGINAL"
	rec, journal := d6RemoveFixture(t, contents)
	data, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := sha256.Sum256([]byte(contents))
	events := 0
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var entry struct {
			Record struct {
				OriginalSHA256 string `json:"original_sha256"`
				OriginalSize   int64  `json:"original_size"`
			} `json:"record"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatal(err)
		}
		events++
		if entry.Record.OriginalSHA256 != hex.EncodeToString(wantHash[:]) || entry.Record.OriginalSize != int64(len(contents)) {
			t.Errorf("durable event %d has no original fingerprint: %#v", events, entry.Record)
		}
	}
	if events < 3 {
		t.Fatalf("expected durable admission and completion events, got %d", events)
	}
	if err := undoMove(rec); err != nil {
		t.Fatal(err)
	}
	d16WantBytes(t, rec.From, contents)
}

func TestD6LegacyDuplicateWithoutFingerprintCannotUndo(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.txt"), filepath.Join(root, "target.txt")
	d16Write(t, target, "unproven surviving bytes")
	rec := MoveRecord{From: source, To: target, Kind: MoveKindDuplicateRemoved}
	if err := undoMove(rec); err == nil {
		t.Error("legacy record without an original fingerprint was accepted as proof")
	}
	d16WantAbsent(t, source)
	d16WantBytes(t, target, "unproven surviving bytes")
}

func TestD6EmptyDuplicateCanStillUndo(t *testing.T) {
	rec, _ := d6RemoveFixture(t, "")
	if err := undoMove(rec); err != nil {
		t.Fatal(err)
	}
	d16WantBytes(t, rec.From, "")
	d16WantBytes(t, rec.To, "")
}

func TestD6BatchPreflightRefusesChangedDuplicateBeforeAnyUndo(t *testing.T) {
	duplicate, _ := d6RemoveFixture(t, "ORIGINAL")
	duplicate.BatchID = "batch-with-duplicate"
	root := t.TempDir()
	plain := MoveRecord{From: filepath.Join(root, "incoming.txt"), To: filepath.Join(root, "sorted.txt"), Kind: MoveKindMove, BatchID: duplicate.BatchID}
	d16Write(t, plain.To, "another completed move")
	d16Write(t, duplicate.To, "MODIFIED")
	history := []MoveRecord{duplicate, plain}
	remaining, undone, err := undoHistoryBatch(history, duplicate.BatchID)
	if err == nil || !strings.Contains(err.Error(), "preflight") {
		t.Errorf("expected preflight refusal, got %v", err)
	}
	if len(remaining) != 2 || len(undone) != 0 {
		t.Errorf("refused batch consumed history: remaining=%v, undone=%v", remaining, undone)
	}
	d16WantAbsent(t, duplicate.From)
	d16WantAbsent(t, plain.From)
	d16WantBytes(t, duplicate.To, "MODIFIED")
	d16WantBytes(t, plain.To, "another completed move")
}

func TestD6RestartDoesNotInferUndoFromTwoChangedCopies(t *testing.T) {
	rec, journal := d6RemoveFixture(t, "ORIGINAL")
	d16Write(t, rec.From, "MODIFIED")
	d16Write(t, rec.To, "MODIFIED")
	recovered, err := recoverMoveJournal(journal, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	entry, err := moveJournalEntryByID(journal, rec.JournalID)
	if err != nil {
		t.Fatal(err)
	}
	if entry.State != MoveJournalAttention || len(recovered.Attention) != 1 {
		t.Errorf("changed copies were reconciled as a valid undo: entry=%#v recovery=%#v", entry, recovered)
	}
	d16WantBytes(t, rec.From, "MODIFIED")
	d16WantBytes(t, rec.To, "MODIFIED")
}

func TestD6PendingRestartRefusesChangedSurvivingCopy(t *testing.T) {
	rec, journal := d6RemoveFixture(t, "ORIGINAL")
	data, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	var pending bytes.Buffer
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var event MoveJournalEntry
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatal(err)
		}
		if event.State == MoveJournalPending {
			pending.Write(line)
			pending.WriteByte('\n')
		}
	}
	interrupted := filepath.Join(t.TempDir(), "interrupted.jsonl")
	if err := os.WriteFile(interrupted, pending.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	d16Write(t, rec.To, "MODIFIED")
	recovered, err := recoverMoveJournal(interrupted, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if recovered.AutoFinalized != 0 || len(recovered.History) != 0 || len(recovered.Attention) != 1 {
		t.Errorf("restart accepted changed bytes as original: %#v", recovered)
	}
	d16WantAbsent(t, rec.From)
	d16WantBytes(t, rec.To, "MODIFIED")
}
