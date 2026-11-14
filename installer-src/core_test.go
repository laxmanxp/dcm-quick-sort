package main

import (
	"encoding/json"
	"testing"
)

func TestParseInstallerArgs(t *testing.T) {
	silent, uninstall, source := parseInstallerArgs([]string{`/S`, `/SOURCE=C:\Users\Me\Inbox`})
	if !silent || uninstall {
		t.Fatalf("bad flags: silent=%v uninstall=%v", silent, uninstall)
	}
	if source != `C:\Users\Me\Inbox` {
		t.Fatalf("source=%q", source)
	}
}

func TestParseUninstall(t *testing.T) {
	silent, uninstall, _ := parseInstallerArgs([]string{`/uninstall`, `/silent`})
	if !silent || !uninstall {
		t.Fatalf("silent=%v uninstall=%v", silent, uninstall)
	}
}

func TestSettingsJSON(t *testing.T) {
	data, err := settingsJSON(`C:\Downloads`)
	if err != nil {
		t.Fatal(err)
	}
	var s settings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	if s.DefaultSource == "" {
		t.Fatal("missing default source")
	}
}

func TestExplorerShellEntriesUsePerUserFileFolderAndBackgroundTargets(t *testing.T) {
	app := `C:\Users\Test User\AppData\Local\Programs\DCM Quick Sort\DCM-QuickSort.exe`
	entries := explorerShellEntries(app)
	if len(entries) != 3 {
		t.Fatalf("entries=%d want 3", len(entries))
	}
	wantCommands := map[string]string{
		`Software\Classes\*\shell\DCMQuickSort`:                    `"` + app + `" "%1"`,
		`Software\Classes\Directory\shell\DCMQuickSort`:            `"` + app + `" "%1"`,
		`Software\Classes\Directory\Background\shell\DCMQuickSort`: `"` + app + `" "%V"`,
	}
	for _, entry := range entries {
		if entry.Label != "Sort with DCM Quick Sort" {
			t.Fatalf("label=%q", entry.Label)
		}
		if entry.Icon != app {
			t.Fatalf("icon=%q want %q", entry.Icon, app)
		}
		if wantCommands[entry.Key] != entry.Command {
			t.Fatalf("key %q command=%q want %q", entry.Key, entry.Command, wantCommands[entry.Key])
		}
		delete(wantCommands, entry.Key)
	}
	if len(wantCommands) != 0 {
		t.Fatalf("missing shell entries: %#v", wantCommands)
	}
}
