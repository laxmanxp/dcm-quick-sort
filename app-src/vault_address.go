package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	vaultAddressIndexVersion = 1
	vaultAddressKindFile     = "file"
	vaultAddressKindFolder   = "folder"
)

type VaultAddressEntry struct {
	Address string `json:"address"`
	Path    string `json:"path"`
	Kind    string `json:"kind"`
}

type VaultAddressIndex struct {
	Version int                 `json:"version"`
	Entries []VaultAddressEntry `json:"entries"`
}

func normalizeLogicalAddress(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, `\`, "/"))
	value = strings.Trim(value, "/")
	if value == "" {
		return "", fmt.Errorf("logical address is empty")
	}
	parts := make([]string, 0, 8)
	for _, part := range strings.Split(value, "/") {
		if part == "" {
			continue
		}
		if part == "." || part == ".." {
			return "", fmt.Errorf("logical address contains traversal segment %q", part)
		}
		if strings.ContainsRune(part, 0) {
			return "", fmt.Errorf("logical address contains a NUL character")
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("logical address is empty")
	}
	return strings.Join(parts, "/"), nil
}

func logicalAddressForPath(vaultRoot, physicalPath string) (string, error) {
	root, err := filepath.Abs(strings.TrimSpace(vaultRoot))
	if err != nil || root == "" {
		return "", fmt.Errorf("invalid vault root")
	}
	target, err := filepath.Abs(strings.TrimSpace(physicalPath))
	if err != nil || target == "" {
		return "", fmt.Errorf("invalid physical path")
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil {
		return "", err
	}
	if rel == "." {
		return "", fmt.Errorf("vault root itself has no logical address")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path is outside the vault")
	}
	return normalizeLogicalAddress(filepath.ToSlash(rel))
}

func vaultAddressIndexPath(vaultRoot string) string {
	return filepath.Join(filepath.Clean(vaultRoot), ".flowverse", "addresses.json")
}

func loadVaultAddressIndex(vaultRoot string) (VaultAddressIndex, error) {
	idx := VaultAddressIndex{Version: vaultAddressIndexVersion, Entries: []VaultAddressEntry{}}
	data, err := os.ReadFile(vaultAddressIndexPath(vaultRoot))
	if os.IsNotExist(err) {
		return idx, nil
	}
	if err != nil {
		return idx, err
	}
	if err := json.Unmarshal(data, &idx); err != nil {
		return VaultAddressIndex{}, fmt.Errorf("read vault address index: %w", err)
	}
	if idx.Version != vaultAddressIndexVersion {
		return VaultAddressIndex{}, fmt.Errorf("unsupported vault address index version %d", idx.Version)
	}
	if idx.Entries == nil {
		idx.Entries = []VaultAddressEntry{}
	}
	return idx, nil
}

func writeVaultAddressIndex(vaultRoot string, idx VaultAddressIndex) error {
	idx.Version = vaultAddressIndexVersion
	sort.SliceStable(idx.Entries, func(i, j int) bool {
		return strings.ToLower(idx.Entries[i].Address) < strings.ToLower(idx.Entries[j].Address)
	})
	path := vaultAddressIndexPath(vaultRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func resolveVaultLogicalAddress(idx VaultAddressIndex, address string) (VaultAddressEntry, bool) {
	normalized, err := normalizeLogicalAddress(address)
	if err != nil {
		return VaultAddressEntry{}, false
	}
	for _, entry := range idx.Entries {
		if strings.EqualFold(entry.Address, normalized) {
			return entry, true
		}
	}
	return VaultAddressEntry{}, false
}

func vaultAddressSuggestions(idx VaultAddressIndex, prefix string, limit int) []VaultAddressEntry {
	prefix = strings.TrimLeft(strings.TrimSpace(strings.ReplaceAll(prefix, `\`, "/")), "/")
	for strings.Contains(prefix, "//") {
		prefix = strings.ReplaceAll(prefix, "//", "/")
	}
	if limit <= 0 || strings.Contains(prefix, "../") || strings.HasPrefix(prefix, "..") {
		return nil
	}
	folded := strings.ToLower(prefix)
	out := make([]VaultAddressEntry, 0, limit)
	for _, entry := range idx.Entries {
		if strings.HasPrefix(strings.ToLower(entry.Address), folded) {
			out = append(out, entry)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		di := strings.Count(out[i].Address, "/")
		dj := strings.Count(out[j].Address, "/")
		if di != dj {
			return di < dj
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == vaultAddressKindFolder
		}
		return strings.ToLower(out[i].Address) < strings.ToLower(out[j].Address)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func updateVaultAddressIndexForMove(vaultRoot, source, target string) error {
	vaultRoot = strings.TrimSpace(vaultRoot)
	if vaultRoot == "" {
		return nil
	}
	idx, err := loadVaultAddressIndex(vaultRoot)
	if err != nil {
		return err
	}
	changed := false
	if strings.TrimSpace(source) != "" {
		if _, err := logicalAddressForPath(vaultRoot, source); err == nil {
			changed = removeVaultEntryByPath(&idx, source) || changed
		}
	}
	if strings.TrimSpace(target) != "" {
		if _, err := logicalAddressForPath(vaultRoot, target); err == nil {
			info, statErr := os.Stat(target)
			if statErr != nil {
				return statErr
			}
			kind := vaultAddressKindFile
			if info.IsDir() {
				kind = vaultAddressKindFolder
			}
			changed = upsertVaultPathAndParents(&idx, vaultRoot, target, kind) || changed
		}
	}
	if !changed {
		return nil
	}
	return writeVaultAddressIndex(vaultRoot, idx)
}

func removeVaultEntryByPath(idx *VaultAddressIndex, physicalPath string) bool {
	physicalPath = filepath.Clean(physicalPath)
	kept := idx.Entries[:0]
	changed := false
	for _, entry := range idx.Entries {
		if strings.EqualFold(filepath.Clean(entry.Path), physicalPath) {
			changed = true
			continue
		}
		kept = append(kept, entry)
	}
	idx.Entries = kept
	return changed
}

func upsertVaultPathAndParents(idx *VaultAddressIndex, vaultRoot, target, kind string) bool {
	changed := false
	current := filepath.Clean(target)
	currentKind := kind
	for {
		address, err := logicalAddressForPath(vaultRoot, current)
		if err != nil {
			break
		}
		changed = upsertVaultAddressEntry(idx, VaultAddressEntry{Address: address, Path: current, Kind: currentKind}) || changed
		parent := filepath.Dir(current)
		if filepath.Clean(parent) == filepath.Clean(vaultRoot) || parent == current {
			break
		}
		current = parent
		currentKind = vaultAddressKindFolder
	}
	return changed
}

func upsertVaultAddressEntry(idx *VaultAddressIndex, entry VaultAddressEntry) bool {
	for i := range idx.Entries {
		if strings.EqualFold(idx.Entries[i].Address, entry.Address) {
			if idx.Entries[i] == entry {
				return false
			}
			idx.Entries[i] = entry
			return true
		}
	}
	idx.Entries = append(idx.Entries, entry)
	return true
}

func syncConfiguredVaultMove(source, target string) error {
	root := loadVaultRootSetting()
	if root == "" {
		return nil
	}
	return updateVaultAddressIndexForMove(root, source, target)
}

func rebuildVaultAddressIndex(vaultRoot string) error {
	vaultRoot = strings.TrimSpace(vaultRoot)
	if vaultRoot == "" {
		return fmt.Errorf("vault root is empty")
	}
	root, err := filepath.Abs(vaultRoot)
	if err != nil {
		return err
	}
	st, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("vault root is not a folder")
	}
	idx := VaultAddressIndex{Version: vaultAddressIndexVersion, Entries: []VaultAddressEntry{}}
	metadataDir := filepath.Join(root, ".flowverse")
	err = filepath.WalkDir(root, func(physicalPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filepath.Clean(physicalPath) == filepath.Clean(root) {
			return nil
		}
		if filepath.Clean(physicalPath) == filepath.Clean(metadataDir) && entry.IsDir() {
			return filepath.SkipDir
		}
		address, err := logicalAddressForPath(root, physicalPath)
		if err != nil {
			return err
		}
		kind := vaultAddressKindFile
		if entry.IsDir() {
			kind = vaultAddressKindFolder
		}
		upsertVaultAddressEntry(&idx, VaultAddressEntry{Address: address, Path: filepath.Clean(physicalPath), Kind: kind})
		return nil
	})
	if err != nil {
		return err
	}
	return writeVaultAddressIndex(root, idx)
}
