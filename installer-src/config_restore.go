package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxInstallerConfigFileBytes = 1 << 20

type installerConfigurationSnapshot struct {
	Files map[string][]byte
	Info  configurationBackupInfo
}

func installerSnapshotFromRaw(files map[string][]byte) (configurationSnapshot, error) {
	names := make([]string, 0, len(files))
	for _, name := range configurationFileNames() {
		if _, exists := files[name]; exists {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return configurationSnapshot{}, ErrNoConfiguration
	}
	combined := sha256.New()
	snap := configurationSnapshot{Files: make([]configurationSnapshotFile, 0, len(names))}
	for _, name := range names {
		data := files[name]
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		snap.Files = append(snap.Files, configurationSnapshotFile{Name: name, Data: data, SHA256: hash, Size: int64(len(data))})
		_, _ = io.WriteString(combined, name)
		_, _ = combined.Write([]byte{0})
		_, _ = io.WriteString(combined, hash)
		_, _ = combined.Write([]byte{0})
	}
	snap.Digest = hex.EncodeToString(combined.Sum(nil))
	return snap, nil
}

func validateInstallerSettings(data []byte) (settings, error) {
	if len(data) == 0 || len(data) > maxInstallerConfigFileBytes {
		return settings{}, fmt.Errorf("settings.json is empty or too large")
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	var cfg settings
	if err := dec.Decode(&cfg); err != nil {
		return settings{}, fmt.Errorf("invalid settings.json: %w", err)
	}
	cfg.DefaultSource = strings.TrimSpace(cfg.DefaultSource)
	if cfg.DefaultSource == "" {
		return settings{}, fmt.Errorf("settings.json does not contain a default source folder")
	}
	return cfg, nil
}

func validateInstallerDestinations(data []byte) error {
	if len(data) == 0 || len(data) > maxInstallerConfigFileBytes {
		return fmt.Errorf("destinations.conf is empty or too large")
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 4096), maxInstallerConfigFileBytes)
	seen := map[string]bool{}
	count := 0
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		name, path, ok := strings.Cut(line, "=")
		name, path = strings.TrimSpace(name), strings.TrimSpace(path)
		if !ok || name == "" || path == "" || strings.ContainsAny(name, "=\r\n") {
			return fmt.Errorf("destinations.conf line %d is invalid", lineNo)
		}
		key := strings.ToLower(name)
		if seen[key] {
			return fmt.Errorf("destinations.conf contains duplicate alias %q", name)
		}
		seen[key] = true
		count++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("destinations.conf contains no destination aliases")
	}
	return nil
}

func validateInstallerSources(data []byte) error {
	if len(data) == 0 || len(data) > maxInstallerConfigFileBytes {
		return fmt.Errorf("sources.conf is empty or too large")
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 4096), maxInstallerConfigFileBytes)
	seen := map[string]bool{}
	count := 0
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		name, path, ok := strings.Cut(line, "=")
		name, path = strings.TrimSpace(name), strings.TrimSpace(path)
		if !ok || name == "" || path == "" || strings.ContainsAny(name, "=\r\n") {
			return fmt.Errorf("sources.conf line %d is invalid", lineNo)
		}
		key := strings.ToLower(name)
		if seen[key] {
			return fmt.Errorf("sources.conf contains duplicate alias %q", name)
		}
		seen[key] = true
		count++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("sources.conf contains no source aliases")
	}
	return nil
}

func validateInstallerLearning(data []byte) error {
	if len(data) == 0 || len(data) > maxInstallerConfigFileBytes {
		return fmt.Errorf("learning.json is empty or too large")
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid learning.json: %w", err)
	}
	return nil
}

func loadLatestBackupSnapshot(backupRoot string) (installerConfigurationSnapshot, error) {
	latest := filepath.Join(strings.TrimSpace(backupRoot), "latest")
	manifestData, err := os.ReadFile(filepath.Join(latest, "backup-info.json"))
	if err != nil {
		return installerConfigurationSnapshot{}, err
	}
	var info configurationBackupInfo
	if err := json.Unmarshal(manifestData, &info); err != nil {
		return installerConfigurationSnapshot{}, fmt.Errorf("invalid backup manifest: %w", err)
	}
	if info.Product != productName {
		return installerConfigurationSnapshot{}, fmt.Errorf("backup belongs to %q", info.Product)
	}
	if info.ConfigVersion < 1 || info.ConfigVersion > configurationFormatVersion {
		return installerConfigurationSnapshot{}, fmt.Errorf("unsupported configuration format version %d", info.ConfigVersion)
	}

	files := map[string][]byte{}
	for _, name := range configurationFileNames() {
		data, err := os.ReadFile(filepath.Join(latest, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return installerConfigurationSnapshot{}, err
		}
		if len(data) > maxInstallerConfigFileBytes {
			return installerConfigurationSnapshot{}, fmt.Errorf("backup file %s is too large", name)
		}
		files[name] = data
	}
	snap, err := installerSnapshotFromRaw(files)
	if err != nil {
		return installerConfigurationSnapshot{}, err
	}
	if !strings.EqualFold(strings.TrimSpace(info.ConfigurationSHA256), snap.Digest) {
		return installerConfigurationSnapshot{}, fmt.Errorf("backup configuration checksum does not match")
	}
	manifestFiles := map[string]configurationBackupFileInfo{}
	allowedManifest := make(map[string]bool)
	for _, name := range configurationFileNames() {
		allowedManifest[name] = true
	}
	for _, item := range info.Files {
		if !allowedManifest[item.Name] {
			return installerConfigurationSnapshot{}, fmt.Errorf("backup manifest contains unsupported file %q", item.Name)
		}
		if _, exists := manifestFiles[item.Name]; exists {
			return installerConfigurationSnapshot{}, fmt.Errorf("backup manifest contains duplicate file %q", item.Name)
		}
		manifestFiles[item.Name] = item
	}
	if len(manifestFiles) != len(snap.Files) {
		return installerConfigurationSnapshot{}, fmt.Errorf("backup manifest file list does not match files")
	}
	for _, item := range snap.Files {
		mf, ok := manifestFiles[item.Name]
		if !ok || mf.Size != item.Size || !strings.EqualFold(mf.SHA256, item.SHA256) {
			return installerConfigurationSnapshot{}, fmt.Errorf("backup file %s does not match its checksum", item.Name)
		}
	}
	if data, ok := files["settings.json"]; ok {
		if _, err := validateInstallerSettings(data); err != nil {
			return installerConfigurationSnapshot{}, err
		}
	}
	if data, ok := files["destinations.conf"]; ok {
		if err := validateInstallerDestinations(data); err != nil {
			return installerConfigurationSnapshot{}, err
		}
	}
	if data, ok := files["sources.conf"]; ok {
		if err := validateInstallerSources(data); err != nil {
			return installerConfigurationSnapshot{}, err
		}
	}
	if data, ok := files["learning.json"]; ok {
		if err := validateInstallerLearning(data); err != nil {
			return installerConfigurationSnapshot{}, err
		}
	}
	return installerConfigurationSnapshot{Files: files, Info: info}, nil
}

func latestBackupDefaultSource(backupRoot string) string {
	snap, err := loadLatestBackupSnapshot(backupRoot)
	if err != nil {
		return ""
	}
	data, ok := snap.Files["settings.json"]
	if !ok {
		return ""
	}
	cfg, err := validateInstallerSettings(data)
	if err != nil {
		return ""
	}
	return cfg.DefaultSource
}

func currentSettingsUsable(configDir string) bool {
	data, err := os.ReadFile(filepath.Join(configDir, "settings.json"))
	if err != nil {
		return false
	}
	_, err = validateInstallerSettings(data)
	return err == nil
}

func shouldOfferPreviousConfigurationRestore(configDir, backupRoot string) (configurationBackupInfo, bool) {
	if currentSettingsUsable(configDir) {
		return configurationBackupInfo{}, false
	}
	snap, err := loadLatestBackupSnapshot(backupRoot)
	if err != nil {
		return configurationBackupInfo{}, false
	}
	return snap.Info, true
}

type installerRollback struct {
	path    string
	existed bool
	data    []byte
}

func restoreBackupSnapshotFiles(configDir string, snap installerConfigurationSnapshot) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	names := configurationFileNames()
	sort.Strings(names)
	rollbacks := make([]installerRollback, 0, len(names))
	staged := map[string]string{}
	for _, name := range names {
		target := filepath.Join(configDir, name)
		rb := installerRollback{path: target}
		if data, err := os.ReadFile(target); err == nil {
			rb.existed, rb.data = true, data
		} else if !os.IsNotExist(err) {
			return err
		}
		rollbacks = append(rollbacks, rb)
		if data, ok := snap.Files[name]; ok {
			tmp := target + ".restore.tmp"
			_ = os.Remove(tmp)
			if err := os.WriteFile(tmp, data, 0o644); err != nil {
				return err
			}
			staged[name] = tmp
		}
	}
	committed := 0
	rollback := func() {
		for i := 0; i < committed; i++ {
			rb := rollbacks[i]
			if rb.existed {
				_ = os.WriteFile(rb.path, rb.data, 0o644)
			} else {
				_ = os.Remove(rb.path)
			}
		}
		for _, tmp := range staged {
			_ = os.Remove(tmp)
		}
	}
	for _, name := range names {
		target := filepath.Join(configDir, name)
		if tmp, ok := staged[name]; ok {
			_ = os.Remove(target)
			if err := os.Rename(tmp, target); err != nil {
				rollback()
				return err
			}
		} else {
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				rollback()
				return err
			}
		}
		committed++
	}
	return nil
}

func restoreLatestBackupFiles(configDir, backupRoot string) (configurationBackupInfo, error) {
	snap, err := loadLatestBackupSnapshot(backupRoot)
	if err != nil {
		return configurationBackupInfo{}, err
	}
	if err := restoreBackupSnapshotFiles(configDir, snap); err != nil {
		return configurationBackupInfo{}, err
	}
	return snap.Info, nil
}

func hasRestoreBackupArg(args []string) bool {
	for _, raw := range args {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "/restorebackup", "--restore-backup", "-restore-backup":
			return true
		}
	}
	return false
}
