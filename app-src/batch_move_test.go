package main

import (
	"path/filepath"
	"testing"
)

func TestBuildBatchMovePlanPreservesSelectionOrderAndNames(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	dest := filepath.Join(root, "dest")
	selected := []string{
		filepath.Join(source, "b-report.pdf"),
		filepath.Join(source, "a-data.csv"),
		filepath.Join(source, "notes.md"),
	}
	plan, err := buildBatchMovePlan(selected, dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != len(selected) {
		t.Fatalf("plan length = %d, want %d", len(plan), len(selected))
	}
	for i := range selected {
		if plan[i].Source != selected[i] {
			t.Fatalf("plan[%d].Source = %q, want %q", i, plan[i].Source, selected[i])
		}
		wantTarget := filepath.Join(dest, filepath.Base(selected[i]))
		if plan[i].Target != wantTarget {
			t.Fatalf("plan[%d].Target = %q, want %q", i, plan[i].Target, wantTarget)
		}
	}
}

func TestBuildBatchMovePlanDeduplicatesSelectedPaths(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	dest := filepath.Join(root, "dest")
	a := filepath.Join(source, "a.txt")
	b := filepath.Join(source, "b.txt")
	plan, err := buildBatchMovePlan([]string{a, a, b, a}, dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 || plan[0].Source != a || plan[1].Source != b {
		t.Fatalf("unexpected deduplicated plan: %#v", plan)
	}
}

func TestBuildBatchMovePlanRejectsCurrentFolder(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	selected := []string{filepath.Join(source, "a.txt"), filepath.Join(source, "b.txt")}
	if _, err := buildBatchMovePlan(selected, source); err == nil {
		t.Fatal("expected same-folder batch move to be rejected")
	}
}
