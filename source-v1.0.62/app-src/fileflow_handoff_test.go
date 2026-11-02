package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileFlowHandoffQueuesExistingCompletedTargetOnce(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "old", "invoice.pdf")
	target := filepath.Join(root, "sorted", "invoice.pdf")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("invoice"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := MoveRecord{From: source, To: target, Kind: MoveKindMove, JournalID: "move-123"}
	queue := filepath.Join(root, "handoff", "fileflow.jsonl")
	now := time.Date(2026, 9, 22, 18, 0, 0, 0, time.UTC)

	added, err := queueFileFlowHandoff(queue, rec, now)
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("first handoff should be queued")
	}
	added, err = queueFileFlowHandoff(queue, rec, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if added {
		t.Fatal("same journal transaction must be deduplicated")
	}

	events, err := loadFileFlowHandoffs(queue)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if events[0].CurrentPath != target || events[0].OriginalPath != source || events[0].ID == "" {
		t.Fatalf("unexpected event %#v", events[0])
	}
}

func TestFileFlowHandoffRefusesMissingTarget(t *testing.T) {
	root := t.TempDir()
	queue := filepath.Join(root, "fileflow.jsonl")
	rec := MoveRecord{From: filepath.Join(root, "source.pdf"), To: filepath.Join(root, "missing.pdf"), Kind: MoveKindMove}
	if _, err := queueFileFlowHandoff(queue, rec, time.Now()); err == nil {
		t.Fatal("missing completed target must not be handed off")
	}
	if _, err := os.Stat(queue); !os.IsNotExist(err) {
		t.Fatalf("queue should not be created on refusal, stat err=%v", err)
	}
}

func TestFileFlowHandoffRefusesDuplicateRemovalTransaction(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "existing.pdf")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := MoveRecord{From: filepath.Join(root, "duplicate.pdf"), To: target, Kind: MoveKindDuplicateRemoved}
	if _, err := queueFileFlowHandoff(filepath.Join(root, "queue.jsonl"), rec, time.Now()); err == nil {
		t.Fatal("exact duplicate cleanup is not a new FileFlow handoff")
	}
}

func TestFileFlowHandoffLoaderAcceptsLegacyVersionOneEvents(t *testing.T) {
	root := t.TempDir()
	queue := filepath.Join(root, "fileflow.jsonl")
	legacy := `{"version":1,"id":"journal:legacy","created_at":"2026-09-22T12:00:00Z","original_path":"C:/old.pdf","current_path":"C:/new.pdf","kind":"move","display_name":"new.pdf"}` + "\n"
	if err := os.WriteFile(queue, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	events, err := loadFileFlowHandoffs(queue)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != fileFlowActionHandoff {
		t.Fatalf("legacy event was not normalized: %#v", events)
	}
}
