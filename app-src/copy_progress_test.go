package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileVerifiedWithProgressReportsStartAndCompletion(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "dest.bin")
	data := bytes.Repeat([]byte("abcdefgh"), 700000) // >4 MiB so at least one intermediate threshold is crossed.
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}
	type point struct{ copied, total int64 }
	var points []point
	if err := copyFileVerifiedWithProgress(src, dst, func(copied, total int64) {
		points = append(points, point{copied, total})
	}); err != nil {
		t.Fatal(err)
	}
	if len(points) < 2 {
		t.Fatalf("progress points=%v, want start and completion", points)
	}
	if points[0].copied != 0 || points[0].total != int64(len(data)) {
		t.Fatalf("first progress=%v", points[0])
	}
	last := points[len(points)-1]
	if last.copied != int64(len(data)) || last.total != int64(len(data)) {
		t.Fatalf("last progress=%v", last)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("copied bytes differ")
	}
}

func TestCopyFileVerifiedWithProgressCleansPartialTargetOnFailure(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "missing.bin")
	dst := filepath.Join(dir, "dest.bin")
	if err := copyFileVerifiedWithProgress(src, dst, func(int64, int64) {}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("partial destination should not exist; err=%v", err)
	}
}
