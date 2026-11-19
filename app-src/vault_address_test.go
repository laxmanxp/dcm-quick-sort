package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogicalAddressForPathUsesVaultRelativeSlashPath(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "BGBO", "2026-27", "CR", "12", "Estimate.pdf")
	got, err := logicalAddressForPath(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if got != "BGBO/2026-27/CR/12/Estimate.pdf" {
		t.Fatalf("got %q", got)
	}
}

func TestLogicalAddressRejectsTraversalAndOutsideVault(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside.pdf")
	if _, err := logicalAddressForPath(root, outside); err == nil {
		t.Fatal("expected outside-vault error")
	}
	if _, err := normalizeLogicalAddress("BGBO/2026-27/../Secret.pdf"); err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestVaultIndexMoveAddsFileAndParentFoldersAndResolvesExactAddress(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "BGBO", "2026-27", "CR", "12", "Estimate.pdf")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("estimate"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateVaultAddressIndexForMove(root, "", target); err != nil {
		t.Fatal(err)
	}

	idx, err := loadVaultAddressIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := resolveVaultLogicalAddress(idx, "bgbo/2026-27/cr/12/estimate.pdf")
	if !ok || filepath.Clean(got.Path) != filepath.Clean(target) || got.Kind != vaultAddressKindFile {
		t.Fatalf("unexpected resolved entry: ok=%v entry=%#v", ok, got)
	}
	folder, ok := resolveVaultLogicalAddress(idx, "BGBO/2026-27/CR/12")
	if !ok || folder.Kind != vaultAddressKindFolder {
		t.Fatalf("folder missing: %#v", folder)
	}
}

func TestVaultAddressSuggestionsPreferNearerHierarchyAndIgnoreCase(t *testing.T) {
	idx := VaultAddressIndex{Version: vaultAddressIndexVersion, Entries: []VaultAddressEntry{
		{Address: "BGBO/2026-27/CR/12/Estimate.pdf", Path: "p3", Kind: vaultAddressKindFile},
		{Address: "BGBO/2026-27", Path: "p1", Kind: vaultAddressKindFolder},
		{Address: "BGBO/2026-27/CR", Path: "p2", Kind: vaultAddressKindFolder},
	}}
	got := vaultAddressSuggestions(idx, "bgbo/2026", 10)
	if len(got) != 3 {
		t.Fatalf("got %d suggestions: %#v", len(got), got)
	}
	if got[0].Address != "BGBO/2026-27" || got[1].Address != "BGBO/2026-27/CR" {
		t.Fatalf("unexpected hierarchy order: %#v", got)
	}
}

func TestVaultIndexRenameRemovesOldAddress(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "BGBO", "2026-27", "CR", "12", "Draft.pdf")
	newPath := filepath.Join(root, "BGBO", "2026-27", "CR", "12", "Estimate.pdf")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, []byte("estimate"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateVaultAddressIndexForMove(root, "", oldPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	if err := updateVaultAddressIndexForMove(root, oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	idx, err := loadVaultAddressIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := resolveVaultLogicalAddress(idx, "BGBO/2026-27/CR/12/Draft.pdf"); ok {
		t.Fatal("stale old address remained")
	}
	if got, ok := resolveVaultLogicalAddress(idx, "BGBO/2026-27/CR/12/Estimate.pdf"); !ok || filepath.Clean(got.Path) != filepath.Clean(newPath) {
		t.Fatalf("new address missing: %#v", got)
	}
}

func TestRebuildVaultAddressIndexIndexesExistingTreeAndSkipsMetadata(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "BGBO", "2026-27", "CR", "12", "Estimate.pdf")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("estimate"), 0o644); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(root, ".flowverse", "ignore.txt")
	if err := os.MkdirAll(filepath.Dir(metadata), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadata, []byte("internal"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := rebuildVaultAddressIndex(root); err != nil {
		t.Fatal(err)
	}
	idx, err := loadVaultAddressIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := resolveVaultLogicalAddress(idx, "BGBO/2026-27/CR/12/Estimate.pdf"); !ok {
		t.Fatal("existing file not indexed")
	}
	if _, ok := resolveVaultLogicalAddress(idx, ".flowverse/ignore.txt"); ok {
		t.Fatal("vault metadata leaked into logical address index")
	}
}

func TestConfiguredVaultMoveUsesSavedVaultRoot(t *testing.T) {
	local := t.TempDir()
	root := filepath.Join(t.TempDir(), "FlowVault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCALAPPDATA", local)
	if err := saveVaultRootSetting(root, ""); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "BGBO", "2026-27", "CR", "12", "Estimate.pdf")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("estimate"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := syncConfiguredVaultMove("", target); err != nil {
		t.Fatal(err)
	}
	idx, err := loadVaultAddressIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := resolveVaultLogicalAddress(idx, "BGBO/2026-27/CR/12/Estimate.pdf"); !ok {
		t.Fatal("configured vault move was not indexed")
	}
}
