package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigurationSnapshotIncludesSavedSourcesWhenPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"default_source":"C:/Downloads"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "destinations.conf"), []byte("Docs=C:\\Docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sources.conf"), []byte("Downloads=C:\\Downloads\nScanner=D:\\Scans\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot, err := collectConfigurationSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range snapshot.Files {
		if f.Name == "sources.conf" {
			found = true
		}
	}
	if !found {
		t.Fatalf("sources.conf missing from configuration snapshot: %#v", snapshot.Files)
	}
}
