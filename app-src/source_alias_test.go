package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceConfigCreatedAndLoaded(t *testing.T) {
	local := t.TempDir()
	defaultSource := filepath.Join(t.TempDir(), "Downloads")
	if err := os.MkdirAll(defaultSource, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCALAPPDATA", local)

	aliases, err := loadSourceAliases(defaultSource)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 1 {
		t.Fatalf("got %d aliases want 1", len(aliases))
	}
	if aliases[0].Name != "Downloads" || aliases[0].Path != defaultSource {
		t.Fatalf("unexpected alias: %#v", aliases[0])
	}
	data, err := os.ReadFile(sourceConfigFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Downloads="+defaultSource) {
		t.Fatalf("created config missing default source: %s", data)
	}
}

func TestSourceAliasCRUDAndPathMatching(t *testing.T) {
	aliases := []SourceAlias{{Name: "Downloads", Path: `C:\\Users\\Me\\Downloads`}}
	aliases, idx, err := upsertSourceAlias(aliases, SourceAlias{Name: "Office Inbox", Path: `D:\\Office\\Inbox`})
	if err != nil {
		t.Fatal(err)
	}
	if idx != 1 || len(aliases) != 2 {
		t.Fatalf("unexpected add: idx=%d aliases=%#v", idx, aliases)
	}
	aliases, idx, err = upsertSourceAlias(aliases, SourceAlias{Name: "office inbox", Path: `E:\\Incoming`})
	if err != nil {
		t.Fatal(err)
	}
	if idx != 1 || aliases[1].Path != `E:\\Incoming` {
		t.Fatalf("unexpected update: %#v", aliases)
	}
	if got := sourceAliasIndexForPath(aliases, `e:\\incoming`); got != 1 {
		t.Fatalf("index=%d want 1", got)
	}
	aliases = deleteSourceAlias(aliases, 0)
	if len(aliases) != 1 || aliases[0].Name != "office inbox" {
		t.Fatalf("unexpected delete: %#v", aliases)
	}
}

func TestWriteSourceAliasesRoundTrip(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	want := []SourceAlias{{Name: "Downloads", Path: `C:\\Users\\Me\\Downloads`}, {Name: "Scanner", Path: `D:\\Scans`}}
	if err := writeSourceAliases(want); err != nil {
		t.Fatal(err)
	}
	got, err := loadSourceAliases("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestSourceAliasValidation(t *testing.T) {
	bad := []SourceAlias{{Name: "", Path: `C:\\A`}, {Name: "Bad=Name", Path: `C:\\A`}, {Name: "NoPath", Path: ""}}
	for _, alias := range bad {
		if err := validateSourceAlias(alias); err == nil {
			t.Fatalf("expected validation failure for %#v", alias)
		}
	}
}
