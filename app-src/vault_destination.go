package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func looksLikeAbsoluteFilesystemPath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return true
	}
	if strings.HasPrefix(value, `\\`) || strings.HasPrefix(value, `//`) {
		return true
	}
	// filepath.VolumeName does not recognize Windows drive paths when tests run
	// on a non-Windows build host. A colon is not valid in a Windows folder name,
	// so treating it as a physical-path marker is safe for DCM's Vault addresses.
	return len(value) >= 2 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':'
}

func resolveVaultDestination(vaultRoot, address string) (physical, logical string, err error) {
	vaultRoot = strings.TrimSpace(vaultRoot)
	if vaultRoot == "" {
		return "", "", fmt.Errorf("configure the Vault Library first")
	}
	st, statErr := os.Stat(vaultRoot)
	if statErr != nil || !st.IsDir() {
		return "", "", fmt.Errorf("configured Vault Library is unavailable")
	}
	address = strings.TrimSpace(address)
	if looksLikeAbsoluteFilesystemPath(address) {
		return "", "", fmt.Errorf("destination must be a Vault logical location, not a disk path")
	}
	logical, err = normalizeLogicalAddress(address)
	if err != nil {
		return "", "", err
	}
	physical = filepath.Join(filepath.Clean(vaultRoot), filepath.FromSlash(logical))
	roundTrip, err := logicalAddressForPath(vaultRoot, physical)
	if err != nil || !strings.EqualFold(roundTrip, logical) {
		return "", "", fmt.Errorf("destination escapes the configured Vault")
	}
	return filepath.Clean(physical), logical, nil
}

func resolveExistingVaultDestination(vaultRoot, address string) (physical, logical string, err error) {
	physical, logical, err = resolveVaultDestination(vaultRoot, address)
	if err != nil {
		return "", "", err
	}
	st, err := os.Stat(physical)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("Vault location does not exist: %s", logical)
		}
		return "", "", err
	}
	if !st.IsDir() {
		return "", "", fmt.Errorf("Vault destination is not a folder: %s", logical)
	}
	return physical, logical, nil
}

func normalizeVaultDestinationFavorites(vaultRoot string, aliases []DestinationAlias) []DestinationAlias {
	vaultRoot = strings.TrimSpace(vaultRoot)
	if vaultRoot == "" {
		return nil
	}
	out := make([]DestinationAlias, 0, len(aliases))
	byName := make(map[string]int, len(aliases))
	for _, alias := range aliases {
		name := strings.TrimSpace(alias.Name)
		value := strings.TrimSpace(alias.Path)
		if name == "" || value == "" {
			continue
		}
		logical := ""
		if looksLikeAbsoluteFilesystemPath(value) {
			if converted, err := logicalAddressForPath(vaultRoot, value); err == nil {
				logical = converted
			}
		} else if normalized, err := normalizeLogicalAddress(value); err == nil {
			logical = normalized
		}
		if logical == "" {
			continue
		}
		if _, _, err := resolveVaultDestination(vaultRoot, logical); err != nil {
			continue
		}
		normalized := DestinationAlias{Name: name, Path: logical}
		key := strings.ToLower(name)
		if idx, ok := byName[key]; ok {
			out[idx] = normalized
		} else {
			byName[key] = len(out)
			out = append(out, normalized)
		}
	}
	return out
}

func loadVaultDestinationIndex(vaultRoot string) (VaultAddressIndex, error) {
	idx, err := loadVaultAddressIndex(vaultRoot)
	if err != nil {
		return VaultAddressIndex{}, err
	}
	if len(idx.Entries) > 0 {
		return idx, nil
	}
	if err := rebuildVaultAddressIndex(vaultRoot); err != nil {
		return VaultAddressIndex{}, err
	}
	return loadVaultAddressIndex(vaultRoot)
}

func vaultDestinationFolderSuggestions(idx VaultAddressIndex, prefix string, limit int) []VaultAddressEntry {
	if limit <= 0 {
		return nil
	}
	prefix = strings.TrimLeft(strings.TrimSpace(strings.ReplaceAll(prefix, `\`, "/")), "/")
	for strings.Contains(prefix, "//") {
		prefix = strings.ReplaceAll(prefix, "//", "/")
	}
	if strings.Contains(prefix, "../") || strings.HasPrefix(prefix, "..") || looksLikeAbsoluteFilesystemPath(prefix) {
		return nil
	}
	folded := strings.ToLower(prefix)
	out := make([]VaultAddressEntry, 0, limit)
	for _, entry := range idx.Entries {
		if entry.Kind != vaultAddressKindFolder {
			continue
		}
		if folded != "" && !strings.HasPrefix(strings.ToLower(entry.Address), folded) {
			continue
		}
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai := strings.ToLower(out[i].Address)
		aj := strings.ToLower(out[j].Address)
		return ai < aj
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func validateVaultChildName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("new folder name is required")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("new folder name cannot be %q", name)
	}
	if strings.ContainsAny(name, `\\/:*?"<>|`) || strings.ContainsAny(name, "\r\n\x00") {
		return fmt.Errorf("new folder name must be one folder name, not a path")
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return fmt.Errorf("new folder name cannot end with a dot or space")
	}
	return nil
}

func createVaultChildFolder(vaultRoot, parentAddress, childName string) (physical, logical string, err error) {
	if err := validateVaultChildName(childName); err != nil {
		return "", "", err
	}
	parentAddress = strings.TrimSpace(parentAddress)
	parentPhysical := ""
	parentLogical := ""
	if parentAddress == "" {
		root := strings.TrimSpace(vaultRoot)
		st, statErr := os.Stat(root)
		if statErr != nil || !st.IsDir() {
			return "", "", fmt.Errorf("configured Vault Library is unavailable")
		}
		parentPhysical = filepath.Clean(root)
	} else {
		parentPhysical, parentLogical, err = resolveExistingVaultDestination(vaultRoot, parentAddress)
		if err != nil {
			return "", "", err
		}
	}
	childName = strings.TrimSpace(childName)
	logicalInput := childName
	if parentLogical != "" {
		logicalInput = parentLogical + "/" + childName
	}
	logical, err = normalizeLogicalAddress(logicalInput)
	if err != nil {
		return "", "", err
	}
	physical = filepath.Join(parentPhysical, childName)
	if _, _, err := resolveVaultDestination(vaultRoot, logical); err != nil {
		return "", "", err
	}
	if st, statErr := os.Stat(physical); statErr == nil {
		if st.IsDir() {
			return "", "", fmt.Errorf("Vault folder already exists: %s", logical)
		}
		return "", "", fmt.Errorf("a file already uses that Vault location: %s", logical)
	} else if !os.IsNotExist(statErr) {
		return "", "", statErr
	}
	if err := os.Mkdir(physical, 0o755); err != nil {
		return "", "", err
	}
	if err := updateVaultAddressIndexForMove(vaultRoot, "", physical); err != nil {
		_ = os.Remove(physical)
		return "", "", err
	}
	return filepath.Clean(physical), logical, nil
}
