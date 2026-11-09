package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStartupSelectionDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := resolveStartupSelection(dir)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(dir)
	if got.SourceDir != filepath.Clean(abs) || got.PreferredFile != "" {
		t.Fatalf("directory selection mismatch: %#v", got)
	}
}

func TestResolveStartupSelectionFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "invoice.pdf")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveStartupSelection(file)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(file)
	abs = filepath.Clean(abs)
	if got.SourceDir != filepath.Dir(abs) || got.PreferredFile != abs {
		t.Fatalf("file selection mismatch: %#v", got)
	}
}

func TestResolveStartupSelectionRejectsMissingPath(t *testing.T) {
	if _, err := resolveStartupSelection(filepath.Join(t.TempDir(), "missing.pdf")); err == nil {
		t.Fatal("expected missing startup path error")
	}
}

func TestPreferredFileIndexUsesCleanCaseInsensitiveComparison(t *testing.T) {
	files := []string{`C:\\Inbox\\A.pdf`, `C:\\Inbox\\Report.PDF`}
	if got := preferredFileIndex(files, `c:\\inbox\\report.pdf`); got != 1 {
		t.Fatalf("preferred index=%d want 1", got)
	}
	if got := preferredFileIndex(files, `C:\\Inbox\\missing.pdf`); got != -1 {
		t.Fatalf("missing preferred index=%d want -1", got)
	}
}
