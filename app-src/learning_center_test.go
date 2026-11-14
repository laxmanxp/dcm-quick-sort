package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLearningCenterProjectsExplainablePatterns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning.json")
	now := time.Now()
	moves := []MoveRecord{
		{From: "5th_SFC_order.pdf", To: filepath.Join("Office", "5th SFC", "2026-09-22 - 5th SFC Order.pdf"), Kind: MoveKindMove},
		{From: "5th_SFC_release.pdf", To: filepath.Join("Office", "5th SFC", "2026-09-22 - 5th SFC Release.pdf"), Kind: MoveKindMove},
		{From: "5th_SFC_statement.pdf", To: filepath.Join("Office", "5th SFC", "2026-09-22 - 5th SFC Statement.pdf"), Kind: MoveKindMove},
		{From: "other.pdf", To: filepath.Join("Misc", "Other.pdf"), Kind: MoveKindMove},
	}
	for i, rec := range moves {
		if err := observeCompletedMoveLearning(path, rec, now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	items, state, err := loadLearningCenterItems(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.TotalObservations != 4 {
		t.Fatalf("observations=%d", state.TotalObservations)
	}
	found := false
	for _, item := range items {
		if item.Kind == LearningPatternDestination && item.Feature == "token:sfc" && filepath.Base(item.Choice) == "5th SFC" {
			found = true
			if item.Count != 3 || item.Total != 3 || item.Confidence != 1 {
				t.Fatalf("unexpected sfc pattern: %#v", item)
			}
		}
	}
	if !found {
		t.Fatalf("expected explainable token:sfc destination pattern in %#v", items)
	}
}

func TestPauseAndResetLearning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning.json")
	rec := MoveRecord{From: "invoice.pdf", To: filepath.Join("Accounts", "Invoice.pdf"), Kind: MoveKindMove}
	if err := observeCompletedMoveLearning(path, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := setLearningPaused(path, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	model, _ := loadLearningModel(path)
	if !model.Paused {
		t.Fatal("expected paused")
	}
	if err := resetLearningModel(path, time.Now()); err != nil {
		t.Fatal(err)
	}
	model, _ = loadLearningModel(path)
	if model.TotalObservations != 0 {
		t.Fatalf("reset observations=%d", model.TotalObservations)
	}
	if !model.Paused {
		t.Fatal("reset should preserve paused preference")
	}
}
