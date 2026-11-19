package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLatestUndoableBatchIDFindsMostRecentTaggedRecord(t *testing.T) {
	history := []MoveRecord{
		{From: "a", To: "b"},
		{From: "c", To: "d", BatchID: "batch-1"},
		{From: "e", To: "f"},
		{From: "g", To: "h", BatchID: "batch-2"},
	}
	if got := latestUndoableBatchID(history); got != "batch-2" {
		t.Fatalf("latestUndoableBatchID = %q, want batch-2", got)
	}
}

func TestUndoHistoryBatchRestoresOnlyThatBatch(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	makeMoved := func(name, batch string) MoveRecord {
		to := filepath.Join(dest, name)
		if err := os.WriteFile(to, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		return MoveRecord{From: filepath.Join(source, name), To: to, Kind: MoveKindMove, BatchID: batch}
	}
	old := makeMoved("old.txt", "")
	b1 := makeMoved("a.txt", "batch-7")
	b2 := makeMoved("b.txt", "batch-7")
	later := makeMoved("later.txt", "")

	remaining, undone, err := undoHistoryBatch([]MoveRecord{old, b1, b2, later}, "batch-7")
	if err != nil {
		t.Fatal(err)
	}
	if len(undone) != 2 || undone[0].To != b2.To || undone[1].To != b1.To {
		t.Fatalf("unexpected undo order: %#v", undone)
	}
	if len(remaining) != 2 || remaining[0].To != old.To || remaining[1].To != later.To {
		t.Fatalf("unexpected remaining history: %#v", remaining)
	}
	for _, rec := range []MoveRecord{b1, b2} {
		if _, err := os.Stat(rec.From); err != nil {
			t.Fatalf("restored source missing: %v", err)
		}
		if _, err := os.Stat(rec.To); !os.IsNotExist(err) {
			t.Fatalf("destination still exists: %s", rec.To)
		}
	}
}

func TestUndoHistoryBatchPreflightPreventsPartialUndo(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	a := MoveRecord{From: filepath.Join(source, "a.txt"), To: filepath.Join(dest, "a.txt"), Kind: MoveKindMove, BatchID: "batch-x"}
	b := MoveRecord{From: filepath.Join(source, "b.txt"), To: filepath.Join(dest, "b.txt"), Kind: MoveKindMove, BatchID: "batch-x"}
	if err := os.WriteFile(a.To, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b.To, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b.From, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}

	remaining, undone, err := undoHistoryBatch([]MoveRecord{a, b}, "batch-x")
	if err == nil {
		t.Fatal("expected preflight failure")
	}
	if len(undone) != 0 || len(remaining) != 2 {
		t.Fatalf("batch changed despite failed preflight: remaining=%d undone=%d", len(remaining), len(undone))
	}
	if _, err := os.Stat(a.To); err != nil {
		t.Fatalf("first destination was changed despite preflight failure: %v", err)
	}
}
