package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultSourceSetting(t *testing.T) {
	local := t.TempDir()
	source := filepath.Join(t.TempDir(), "Inbox")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCALAPPDATA", local)
	settingsDir := filepath.Join(local, "DCM Quick Sort")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"default_source":"` + filepath.ToSlash(source) + `"}`)
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadDefaultSourceSetting(); got != filepath.ToSlash(source) {
		t.Fatalf("got %q want %q", got, filepath.ToSlash(source))
	}
}

func TestLoadDefaultSourceSettingIgnoresMissingFolder(t *testing.T) {
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	settingsDir := filepath.Join(local, "DCM Quick Sort")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(`{"default_source":"Z:/definitely/missing"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadDefaultSourceSetting(); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestDestinationConfigCreatedAndLoaded(t *testing.T) {
	local := t.TempDir()
	defaultDest := filepath.Join(t.TempDir(), "Documents")
	if err := os.MkdirAll(defaultDest, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCALAPPDATA", local)

	aliases, err := loadDestinationAliases(defaultDest)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 1 {
		t.Fatalf("got %d aliases want 1", len(aliases))
	}
	if aliases[0].Name != "Documents" || aliases[0].Path != defaultDest {
		t.Fatalf("unexpected alias: %#v", aliases[0])
	}
	data, err := os.ReadFile(destinationConfigFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Documents="+defaultDest) {
		t.Fatalf("created config missing default destination: %s", data)
	}
}

func TestDestinationConfigParsesNamedPaths(t *testing.T) {
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	settingsDir := filepath.Join(local, "DCM Quick Sort")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "# Name=Full path\nOffice=C:\\Office\\Files\nPhotos=D:\\Pictures\nOffice=C:\\Office\\Updated\nBadLine\n"
	if err := os.WriteFile(filepath.Join(settingsDir, "destinations.conf"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	aliases, err := loadDestinationAliases("")
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 2 {
		t.Fatalf("got %d aliases want 2: %#v", len(aliases), aliases)
	}
	if aliases[0].Name != "Office" || aliases[0].Path != `C:\Office\Updated` {
		t.Fatalf("unexpected office alias: %#v", aliases[0])
	}
	if aliases[1].Name != "Photos" || aliases[1].Path != `D:\Pictures` {
		t.Fatalf("unexpected photos alias: %#v", aliases[1])
	}
}

func TestDestinationAliasForPath(t *testing.T) {
	aliases := []DestinationAlias{
		{Name: "Office", Path: `C:\Office\Files`},
		{Name: "Photos", Path: `D:\Pictures`},
	}
	if got := destinationAliasIndexForPath(aliases, `d:\pictures`); got != 1 {
		t.Fatalf("got index %d want 1", got)
	}
	if got := destinationAliasIndexForPath(aliases, `C:\Elsewhere`); got != -1 {
		t.Fatalf("got index %d want -1", got)
	}
}

func TestUpsertDestinationAliasAddsAndUpdates(t *testing.T) {
	aliases := []DestinationAlias{{Name: "Documents", Path: `C:\\Users\\Me\\Documents`}}
	aliases, idx, err := upsertDestinationAlias(aliases, DestinationAlias{Name: "Photos", Path: `D:\\Pictures`})
	if err != nil {
		t.Fatal(err)
	}
	if idx != 1 || len(aliases) != 2 || aliases[1].Name != "Photos" {
		t.Fatalf("unexpected add result: idx=%d aliases=%#v", idx, aliases)
	}
	aliases, idx, err = upsertDestinationAlias(aliases, DestinationAlias{Name: "photos", Path: `E:\\Images`})
	if err != nil {
		t.Fatal(err)
	}
	if idx != 1 || len(aliases) != 2 || aliases[1].Path != `E:\\Images` {
		t.Fatalf("unexpected update result: idx=%d aliases=%#v", idx, aliases)
	}
}

func TestDeleteDestinationAlias(t *testing.T) {
	aliases := []DestinationAlias{{Name: "A", Path: `C:\\A`}, {Name: "B", Path: `C:\\B`}, {Name: "C", Path: `C:\\C`}}
	got := deleteDestinationAlias(aliases, 1)
	if len(got) != 2 || got[0].Name != "A" || got[1].Name != "C" {
		t.Fatalf("unexpected delete result: %#v", got)
	}
}

func TestWriteDestinationAliasesRoundTrip(t *testing.T) {
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	want := []DestinationAlias{{Name: "Photos", Path: `D:\\Pictures`}, {Name: "Invoices", Path: `D:\\Accounts\\Invoices`}}
	if err := writeDestinationAliases(want); err != nil {
		t.Fatal(err)
	}
	got, err := loadDestinationAliases("")
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

func TestDestinationAliasValidation(t *testing.T) {
	bad := []DestinationAlias{
		{Name: "", Path: `C:\\A`},
		{Name: "Bad=Name", Path: `C:\\A`},
		{Name: "NoPath", Path: ""},
	}
	for _, alias := range bad {
		if err := validateDestinationAlias(alias); err == nil {
			t.Fatalf("expected validation failure for %#v", alias)
		}
	}
}

func TestMoveJournalFilePathUsesLocalAppDataRecoveryFolder(t *testing.T) {
	old := os.Getenv("LOCALAPPDATA")
	t.Cleanup(func() { _ = os.Setenv("LOCALAPPDATA", old) })
	base := t.TempDir()
	if err := os.Setenv("LOCALAPPDATA", base); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "DCM Quick Sort", "recovery", "move-journal.jsonl")
	if got := moveJournalFilePath(); got != want {
		t.Fatalf("move journal path=%q want %q", got, want)
	}
}
