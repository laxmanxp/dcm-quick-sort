package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileFlowMetadataDraftStagesDistinctValues(t *testing.T) {
	var d FileFlowMetadataDraft
	d.AddTag(" PMAY-G ")
	d.AddTag("pmay-g")
	d.AddTodo("Send compliance report")
	d.AddNote("Received from Block office")
	if len(d.Tags) != 1 || d.Tags[0] != "PMAY-G" {
		t.Fatalf("unexpected tags %#v", d.Tags)
	}
	if len(d.Todos) != 1 || d.Todos[0] != "Send compliance report" {
		t.Fatalf("unexpected todos %#v", d.Todos)
	}
	if len(d.Notes) != 1 || d.Notes[0] != "Received from Block office" {
		t.Fatalf("unexpected notes %#v", d.Notes)
	}
	if d.Empty() {
		t.Fatal("draft with metadata must not be empty")
	}
}

func TestFileFlowDraftStoreSurvivesReloadRenameAndClear(t *testing.T) {
	root := t.TempDir()
	store := filepath.Join(root, "fileflow-drafts.json")
	from := filepath.Join(root, "Downloads", "abc.pdf")
	renamed := filepath.Join(root, "Downloads", "Sanction Order.pdf")
	draft := FileFlowMetadataDraft{Tags: []string{"sanction"}, Todos: []string{"Reply"}, Notes: []string{"Important"}}
	if err := saveFileFlowDraft(store, from, draft); err != nil {
		t.Fatal(err)
	}
	got, ok, err := loadFileFlowDraft(store, from)
	if err != nil || !ok || len(got.Tags) != 1 {
		t.Fatalf("load failed: draft=%#v ok=%v err=%v", got, ok, err)
	}
	if err := migrateFileFlowDraft(store, from, renamed); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := loadFileFlowDraft(store, from); err != nil || ok {
		t.Fatalf("old path should no longer own draft, ok=%v err=%v", ok, err)
	}
	if got, ok, err = loadFileFlowDraft(store, renamed); err != nil || !ok || got.Todos[0] != "Reply" {
		t.Fatalf("renamed draft missing: %#v ok=%v err=%v", got, ok, err)
	}
	if err := clearFileFlowDraft(store, renamed); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := loadFileFlowDraft(store, renamed); err != nil || ok {
		t.Fatalf("draft should be cleared, ok=%v err=%v", ok, err)
	}
}

func TestFileFlowMetadataUpdateQueuesFinalMovedPath(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sorted", "Sanction Order.pdf")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := MoveRecord{From: filepath.Join(root, "Downloads", "Sanction Order.pdf"), To: target, Kind: MoveKindMove, JournalID: "move-meta-1"}
	draft := FileFlowMetadataDraft{Tags: []string{"PMAY-G"}, Todos: []string{"Send reply"}, Notes: []string{"Received today"}}
	queue := filepath.Join(root, "handoff", "fileflow.jsonl")
	added, err := queueFileFlowMetadataUpdate(queue, rec, draft, time.Date(2026, 9, 23, 4, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("metadata update should be queued")
	}
	events, err := loadFileFlowHandoffs(queue)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	e := events[0]
	if e.Action != fileFlowActionEnsureMetadata || e.CurrentPath != target {
		t.Fatalf("wrong event %#v", e)
	}
	if len(e.Tags) != 1 || e.Tags[0] != "PMAY-G" || len(e.Todos) != 1 || len(e.Notes) != 1 {
		t.Fatalf("metadata not carried %#v", e)
	}
}

func TestFileFlowMetadataUpdateAllowsExactDuplicateDestination(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sorted", "same.pdf")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := MoveRecord{From: filepath.Join(root, "incoming", "same.pdf"), To: target, Kind: MoveKindDuplicateRemoved, JournalID: "dup-meta"}
	draft := FileFlowMetadataDraft{Tags: []string{"duplicate-reviewed"}}
	if _, err := queueFileFlowMetadataUpdate(filepath.Join(root, "queue.jsonl"), rec, draft, time.Now()); err != nil {
		t.Fatalf("metadata should be applicable to retained exact-duplicate target: %v", err)
	}
}
