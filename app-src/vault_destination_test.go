package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveVaultDestinationUsesLogicalAddressOnly(t *testing.T) {
	root := t.TempDir()
	physical, logical, err := resolveVaultDestination(root, `BGBO/2026-27/CR/12`)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	want := filepath.Join(root, "BGBO", "2026-27", "CR", "12")
	if filepath.Clean(physical) != filepath.Clean(want) {
		t.Fatalf("physical=%q want %q", physical, want)
	}
	if logical != "BGBO/2026-27/CR/12" {
		t.Fatalf("logical=%q", logical)
	}
}

func TestResolveVaultDestinationRejectsTraversalAndAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	bad := []string{`../Outside`, `BGBO/../../Outside`, filepath.Join(root, "BGBO")}
	if runtime.GOOS != "windows" {
		bad = append(bad, `/tmp/outside`)
	}
	for _, value := range bad {
		if _, _, err := resolveVaultDestination(root, value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestResolveExistingVaultDestinationRequiresFolder(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "BGBO", "2026-27")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	physical, logical, err := resolveExistingVaultDestination(root, `BGBO/2026-27`)
	if err != nil {
		t.Fatalf("existing folder: %v", err)
	}
	if filepath.Clean(physical) != filepath.Clean(dir) || logical != "BGBO/2026-27" {
		t.Fatalf("got physical=%q logical=%q", physical, logical)
	}
	if _, _, err := resolveExistingVaultDestination(root, `BGBO/Missing`); err == nil {
		t.Fatal("missing destination should fail")
	}
}

func TestVaultDestinationFavoritesMigrateOnlyLocationsInsideVault(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "BGBO", "2026-27", "CR")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := []DestinationAlias{
		{Name: "Cases", Path: inside},
		{Name: "Logical", Path: `Accounts/2026-27`},
		{Name: "Outside", Path: filepath.Join(t.TempDir(), "Elsewhere")},
	}
	got := normalizeVaultDestinationFavorites(root, legacy)
	if len(got) != 2 {
		t.Fatalf("favorites=%#v", got)
	}
	if got[0].Name != "Cases" || got[0].Path != "BGBO/2026-27/CR" {
		t.Fatalf("inside migration=%#v", got[0])
	}
	if got[1].Name != "Logical" || got[1].Path != "Accounts/2026-27" {
		t.Fatalf("logical favorite=%#v", got[1])
	}
}

func TestVaultFolderSuggestionsContainFoldersOnly(t *testing.T) {
	idx := VaultAddressIndex{Version: 1, Entries: []VaultAddressEntry{
		{Address: "BGBO", Path: `D:\\Vault\\BGBO`, Kind: vaultAddressKindFolder},
		{Address: "BGBO/2026-27", Path: `D:\\Vault\\BGBO\\2026-27`, Kind: vaultAddressKindFolder},
		{Address: "BGBO/2026-27/Estimate.pdf", Path: `D:\\Vault\\BGBO\\2026-27\\Estimate.pdf`, Kind: vaultAddressKindFile},
	}}
	got := vaultDestinationFolderSuggestions(idx, "bgbo/2026", 10)
	if len(got) != 1 || got[0].Address != "BGBO/2026-27" || got[0].Kind != vaultAddressKindFolder {
		t.Fatalf("suggestions=%#v", got)
	}
}

func TestCreateVaultChildFolderUpdatesIndex(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "BGBO", "2026-27", "CR")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := rebuildVaultAddressIndex(root); err != nil {
		t.Fatal(err)
	}
	physical, logical, err := createVaultChildFolder(root, "BGBO/2026-27/CR", "12")
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	if logical != "BGBO/2026-27/CR/12" || filepath.Base(physical) != "12" {
		t.Fatalf("physical=%q logical=%q", physical, logical)
	}
	idx, err := loadVaultAddressIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := resolveVaultLogicalAddress(idx, logical)
	if !ok || entry.Kind != vaultAddressKindFolder {
		t.Fatalf("missing folder entry: %#v %v", entry, ok)
	}
}

func TestCreateVaultChildFolderAllowsTopLevelCreation(t *testing.T) {
	root := t.TempDir()
	physical, logical, err := createVaultChildFolder(root, "", "BGBO")
	if err != nil {
		t.Fatalf("create top-level folder: %v", err)
	}
	if logical != "BGBO" || filepath.Clean(physical) != filepath.Join(root, "BGBO") {
		t.Fatalf("physical=%q logical=%q", physical, logical)
	}
	idx, err := loadVaultAddressIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := resolveVaultLogicalAddress(idx, "BGBO")
	if !ok || entry.Kind != vaultAddressKindFolder {
		t.Fatalf("top-level folder not indexed: %#v %v", entry, ok)
	}
}

func TestCreateVaultChildFolderRejectsNestedOrTraversalName(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "BGBO"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "..", ".", "A/B", `A\\B`} {
		if _, _, err := createVaultChildFolder(root, "BGBO", name); err == nil {
			t.Fatalf("expected child name %q to fail", name)
		}
	}
}

func TestLoadVaultDestinationIndexRebuildsMissingIndex(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "BGBO", "2026-27", "CR", "12")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	idx, err := loadVaultDestinationIndex(root)
	if err != nil {
		t.Fatalf("load destination index: %v", err)
	}
	entry, ok := resolveVaultLogicalAddress(idx, "BGBO/2026-27/CR/12")
	if !ok || entry.Kind != vaultAddressKindFolder {
		t.Fatalf("missing auto-rebuilt destination folder: %#v %v", entry, ok)
	}
}

func TestVaultDestinationUIWiringMarkers(t *testing.T) {
	data, err := os.ReadFile("main_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	markers := []string{
		`"Vault &Destinations…"`,
		`"Vault Location:"`,
		`"Find location:"`,
		`"&Use Location"`,
		`"&Create Child"`,
		`"&Save Favorite"`,
		`"&Rebuild Index"`,
		`resolveExistingVaultDestination`,
		`procSetForegroundWindow.Call(destManager.hwnd)`,
		`procSetForegroundWindow.Call(hwnd)`,
	}
	for _, marker := range markers {
		if !strings.Contains(s, marker) {
			t.Fatalf("missing UI wiring marker %q", marker)
		}
	}
}

func TestMainDestinationWorkflowIsVaultOnly(t *testing.T) {
	data, err := os.ReadFile("main_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	required := []string{
		`app.destBtn = createControl("BUTTON", "&Manage…"`,
		`app.pathLabel = createControl("STATIC", "Vault Location:"`,
		`resolveExistingVaultDestination(root, strings.TrimSpace(getText(app.destEdit)))`,
		`startBatchMove(selected, physicalDest, logicalDest)`,
		`"Move to &Vault"`,
	}
	for _, marker := range required {
		if !strings.Contains(src, marker) {
			t.Fatalf("missing Vault-only destination wiring %q", marker)
		}
	}
	forbidden := []string{
		`browseFolder(hwnd, "Choose destination folder"`,
		`folderSuggestions(typed, 12)`,
		`setDestinationText(dest)`,
	}
	for _, marker := range forbidden {
		if strings.Contains(src, marker) {
			t.Fatalf("legacy arbitrary-destination wiring remains: %q", marker)
		}
	}
}
