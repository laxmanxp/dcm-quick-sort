package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectDestinationConflictAvailable(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "src.pdf")
	target := filepath.Join(d, "dest", "Report.pdf")
	if err := os.WriteFile(src, []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := inspectDestinationConflict(src, target)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ConflictNone {
		t.Fatalf("kind=%v want none", got.Kind)
	}
}

func TestInspectDestinationConflictExactDuplicateUsesSHA256(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "src.pdf")
	target := filepath.Join(d, "Report.pdf")
	data := []byte("same bytes")
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := inspectDestinationConflict(src, target)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ConflictExactDuplicate {
		t.Fatalf("kind=%v want exact duplicate", got.Kind)
	}
	if got.SourceSHA256 == "" || got.TargetSHA256 == "" || got.SourceSHA256 != got.TargetSHA256 {
		t.Fatalf("hashes not equal: %#v", got)
	}
}

func TestInspectDestinationConflictDifferentSameSize(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "src.pdf")
	target := filepath.Join(d, "Report.pdf")
	if err := os.WriteFile(src, []byte("AAAA"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("BBBB"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := inspectDestinationConflict(src, target)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ConflictDifferent {
		t.Fatalf("kind=%v want different", got.Kind)
	}
	if got.SourceSHA256 == got.TargetSHA256 {
		t.Fatalf("hashes unexpectedly equal: %#v", got)
	}
}

func TestInspectDestinationConflictDifferentSizeAvoidsFalseDuplicate(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "src.pdf")
	target := filepath.Join(d, "Report.pdf")
	if err := os.WriteFile(src, []byte("short"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("much longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := inspectDestinationConflict(src, target)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ConflictDifferent {
		t.Fatalf("kind=%v want different", got.Kind)
	}
	if got.SourceSize == got.TargetSize {
		t.Fatalf("expected different sizes: %#v", got)
	}
}

func TestInspectDestinationConflictRejectsDirectoryAtTarget(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "src.pdf")
	target := filepath.Join(d, "Report.pdf")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectDestinationConflict(src, target); err == nil {
		t.Fatal("expected directory target to be rejected")
	}
}

func TestKeepBothTargetIsUniqueWithoutOverwriting(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "Report.pdf")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "Report (2).pdf"), []byte("old2"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := keepBothTarget(target)
	if filepath.Base(got) != "Report (3).pdf" {
		t.Fatalf("got %q", got)
	}
}

func TestReplaceMoveCanBeUndoneAndRestoresPreviousDestination(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	recovery := t.TempDir()
	src := filepath.Join(srcDir, "incoming.txt")
	target := filepath.Join(dstDir, "Report.txt")
	if err := os.WriteFile(src, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, err := replaceMoveWithRecovery(src, target, recovery)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Kind != MoveKindReplace || rec.ReplacedBackup == "" {
		t.Fatalf("bad record: %#v", rec)
	}
	if got, _ := os.ReadFile(target); string(got) != "NEW" {
		t.Fatalf("target=%q", got)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be moved, err=%v", err)
	}
	if got, _ := os.ReadFile(rec.ReplacedBackup); string(got) != "OLD" {
		t.Fatalf("backup=%q", got)
	}

	if err := undoMove(rec); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(src); string(got) != "NEW" {
		t.Fatalf("source after undo=%q", got)
	}
	if got, _ := os.ReadFile(target); string(got) != "OLD" {
		t.Fatalf("target after undo=%q", got)
	}
	if _, err := os.Stat(rec.ReplacedBackup); !os.IsNotExist(err) {
		t.Fatalf("backup should be removed after undo, err=%v", err)
	}
}

func TestReplaceMoveRollsBackWhenSourceMoveFails(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	recovery := t.TempDir()
	src := filepath.Join(srcDir, "incoming.txt")
	target := filepath.Join(dstDir, "Report.txt")
	if err := os.WriteFile(src, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make source impossible to move after the destination is safely backed up
	// by using the injectable helper.
	_, err := replaceMoveWithRecoveryUsing(src, target, recovery, func(_, _ string) error {
		return os.ErrPermission
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	if got, _ := os.ReadFile(src); string(got) != "NEW" {
		t.Fatalf("source changed: %q", got)
	}
	if got, _ := os.ReadFile(target); string(got) != "OLD" {
		t.Fatalf("old target not restored: %q", got)
	}
}

func TestExactDuplicateRemovalUndoRestoresSourceWithoutRemovingDestination(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "incoming.txt")
	target := filepath.Join(dstDir, "Report.txt")
	if err := os.WriteFile(src, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("SAME"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, err := removeExactDuplicateSource(src, target)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Kind != MoveKindDuplicateRemoved {
		t.Fatalf("bad kind: %#v", rec)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source not removed: %v", err)
	}
	if err := undoMove(rec); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(src); string(got) != "SAME" {
		t.Fatalf("source=%q", got)
	}
	if got, _ := os.ReadFile(target); string(got) != "SAME" {
		t.Fatalf("destination should remain: %q", got)
	}
}

func TestRemoveExactDuplicateSourceRefusesNonDuplicate(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "src")
	target := filepath.Join(d, "target")
	_ = os.WriteFile(src, []byte("A"), 0o644)
	_ = os.WriteFile(target, []byte("B"), 0o644)
	if _, err := removeExactDuplicateSource(src, target); err == nil || !strings.Contains(strings.ToLower(err.Error()), "not an exact duplicate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessedLabelsDescribeReplaceAndDuplicateRemoval(t *testing.T) {
	rep := processedRecordLabel(MoveRecord{From: "a.txt", To: "b.txt", Kind: MoveKindReplace})
	if !strings.Contains(strings.ToLower(rep), "replace") {
		t.Fatalf("replace label=%q", rep)
	}
	dup := processedRecordLabel(MoveRecord{From: "a.txt", To: "b.txt", Kind: MoveKindDuplicateRemoved})
	if !strings.Contains(strings.ToLower(dup), "duplicate") {
		t.Fatalf("duplicate label=%q", dup)
	}
}
