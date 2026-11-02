package main

import (
	"archive/zip"
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
	"time"
)

const (
	maxConfigurationArchiveEntries = 8
	maxConfigurationFileBytes      = 1 << 20
	maxConfigurationArchiveBytes   = 3 << 20
)

type configurationImportKind string

const (
	configurationImportArchive      configurationImportKind = "dcmconf"
	configurationImportSettings     configurationImportKind = "settings"
	configurationImportDestinations configurationImportKind = "destinations"
	configurationImportLatest       configurationImportKind = "latest-backup"
)

type aliasImportMode int

const (
	aliasReplaceAll aliasImportMode = iota
	aliasMergeKeepExisting
	aliasMergeUseImported
)

type configurationImportPackage struct {
	Kind            configurationImportKind
	SourcePath      string
	Settings        *localSettings
	HasDestinations bool
	Destinations    []DestinationAlias
	HasSources      bool
	Sources         []SourceAlias
	Learning        []byte
	NamingTemplates []byte
	BackupInfo      *configurationBackupInfo
}

type configurationImportResult struct {
	Kind                    configurationImportKind
	SettingsImported        bool
	DestinationsImported    int
	SourcesImported         int
	LearningImported        bool
	NamingTemplatesImported bool
	AliasConflicts          int
	PreImportBackupDir      string
	PostImportBackupDir     string
	Warning                 string
}

func parseImportedSettings(data []byte) (localSettings, error) {
	if len(data) == 0 || len(data) > maxConfigurationFileBytes {
		return localSettings{}, fmt.Errorf("settings.json is empty or too large")
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	var cfg localSettings
	if err := dec.Decode(&cfg); err != nil {
		return localSettings{}, fmt.Errorf("invalid settings.json: %w", err)
	}
	cfg.DefaultSource = strings.TrimSpace(cfg.DefaultSource)
	if cfg.DefaultSource == "" {
		return localSettings{}, fmt.Errorf("settings.json does not contain a default source folder")
	}
	return cfg, nil
}

func parseImportedDestinationAliases(data []byte) ([]DestinationAlias, error) {
	if len(data) == 0 || len(data) > maxConfigurationFileBytes {
		return nil, fmt.Errorf("destinations.conf is empty or too large")
	}
	aliases := make([]DestinationAlias, 0, 16)
	byName := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 4096), maxConfigurationFileBytes)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		name, path, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("destinations.conf line %d is missing '='", lineNo)
		}
		alias := DestinationAlias{Name: strings.TrimSpace(name), Path: strings.TrimSpace(path)}
		if err := validateDestinationAlias(alias); err != nil {
			return nil, fmt.Errorf("destinations.conf line %d: %w", lineNo, err)
		}
		key := strings.ToLower(alias.Name)
		if idx, exists := byName[key]; exists {
			aliases[idx] = alias
			continue
		}
		byName[key] = len(aliases)
		aliases = append(aliases, alias)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read destinations.conf: %w", err)
	}
	if len(aliases) == 0 {
		return nil, fmt.Errorf("destinations.conf contains no destination aliases")
	}
	return aliases, nil
}

func parseImportedSourceAliases(data []byte) ([]SourceAlias, error) {
	if len(data) == 0 || len(data) > maxConfigurationFileBytes {
		return nil, fmt.Errorf("sources.conf is empty or too large")
	}
	aliases := make([]SourceAlias, 0, 16)
	byName := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 4096), maxConfigurationFileBytes)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		name, path, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("sources.conf line %d is missing '='", lineNo)
		}
		alias := SourceAlias{Name: strings.TrimSpace(name), Path: strings.TrimSpace(path)}
		if err := validateSourceAlias(alias); err != nil {
			return nil, fmt.Errorf("sources.conf line %d: %w", lineNo, err)
		}
		key := strings.ToLower(alias.Name)
		if idx, exists := byName[key]; exists {
			aliases[idx] = alias
			continue
		}
		byName[key] = len(aliases)
		aliases = append(aliases, alias)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read sources.conf: %w", err)
	}
	if len(aliases) == 0 {
		return nil, fmt.Errorf("sources.conf contains no source aliases")
	}
	return aliases, nil
}

func validateImportedLearning(data []byte) error {
	if len(data) == 0 || len(data) > maxConfigurationFileBytes {
		return fmt.Errorf("learning.json is empty or too large")
	}
	var m LearningModel
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("invalid learning.json: %w", err)
	}
	if m.Version != learningModelVersion {
		return fmt.Errorf("unsupported learning model version %d", m.Version)
	}
	return nil
}

func validateImportedNamingTemplates(data []byte) error {
	if len(data) == 0 || len(data) > maxConfigurationFileBytes {
		return fmt.Errorf("naming-templates.json is empty or too large")
	}
	var cfg NamingTemplateConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("invalid naming-templates.json: %w", err)
	}
	if err := validateNamingTemplateConfig(cfg); err != nil {
		return fmt.Errorf("invalid naming-templates.json: %w", err)
	}
	return nil
}

func destinationAliasesData(aliases []DestinationAlias) ([]byte, error) {
	var b strings.Builder
	b.WriteString("# DCM - Quick Sort named destinations\n")
	b.WriteString("# Managed by DCM. Use Manage Destinations in the app.\n")
	b.WriteString("# Format: Name=Full path\n")
	seen := make(map[string]bool)
	for _, alias := range aliases {
		alias.Name = strings.TrimSpace(alias.Name)
		alias.Path = strings.TrimSpace(alias.Path)
		if err := validateDestinationAlias(alias); err != nil {
			return nil, err
		}
		key := strings.ToLower(alias.Name)
		if seen[key] {
			return nil, fmt.Errorf("duplicate destination alias %q", alias.Name)
		}
		seen[key] = true
		b.WriteString(alias.Name)
		b.WriteString("=")
		b.WriteString(alias.Path)
		b.WriteString("\n")
	}
	return []byte(b.String()), nil
}

func importedSettingsData(cfg localSettings) ([]byte, error) {
	cfg.DefaultSource = strings.TrimSpace(cfg.DefaultSource)
	if cfg.DefaultSource == "" {
		return nil, fmt.Errorf("default source folder is required")
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func readDestinationAliasesFromPath(path string) ([]DestinationAlias, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseImportedDestinationAliases(data)
}

func countAliasConflicts(existing, imported []DestinationAlias) int {
	names := make(map[string]bool, len(existing))
	for _, alias := range existing {
		names[strings.ToLower(strings.TrimSpace(alias.Name))] = true
	}
	conflicts := 0
	for _, alias := range imported {
		if names[strings.ToLower(strings.TrimSpace(alias.Name))] {
			conflicts++
		}
	}
	return conflicts
}

func mergeImportedDestinationAliases(existing, imported []DestinationAlias, mode aliasImportMode) ([]DestinationAlias, int, error) {
	for _, alias := range imported {
		if err := validateDestinationAlias(alias); err != nil {
			return nil, 0, err
		}
	}
	conflicts := countAliasConflicts(existing, imported)
	if mode == aliasReplaceAll {
		out := append([]DestinationAlias(nil), imported...)
		return out, conflicts, nil
	}
	if mode != aliasMergeKeepExisting && mode != aliasMergeUseImported {
		return nil, 0, fmt.Errorf("unsupported alias import mode")
	}

	out := append([]DestinationAlias(nil), existing...)
	index := make(map[string]int, len(out))
	for i, alias := range out {
		index[strings.ToLower(strings.TrimSpace(alias.Name))] = i
	}
	for _, alias := range imported {
		key := strings.ToLower(strings.TrimSpace(alias.Name))
		if idx, exists := index[key]; exists {
			if mode == aliasMergeUseImported {
				out[idx] = alias
			}
			continue
		}
		index[key] = len(out)
		out = append(out, alias)
	}
	return out, conflicts, nil
}

func snapshotFromRawFiles(files map[string][]byte) (configurationSnapshot, error) {
	names := make([]string, 0, len(files))
	allowed := map[string]bool{"settings.json": true, "destinations.conf": true, "sources.conf": true, "learning.json": true}
	for name := range files {
		if allowed[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return configurationSnapshot{}, ErrNoConfiguration
	}
	combined := sha256.New()
	snapshot := configurationSnapshot{Files: make([]configurationSnapshotFile, 0, len(names))}
	for _, name := range names {
		data := files[name]
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		snapshot.Files = append(snapshot.Files, configurationSnapshotFile{Name: name, Data: data, SHA256: hash, Size: int64(len(data))})
		_, _ = io.WriteString(combined, name)
		_, _ = combined.Write([]byte{0})
		_, _ = io.WriteString(combined, hash)
		_, _ = combined.Write([]byte{0})
	}
	snapshot.Digest = hex.EncodeToString(combined.Sum(nil))
	return snapshot, nil
}

func validateConfigurationManifest(info configurationBackupInfo, files map[string][]byte) error {
	if info.Product != "DCM - Quick Sort" {
		return fmt.Errorf("configuration belongs to %q, not DCM - Quick Sort", info.Product)
	}
	if info.ConfigVersion < 1 || info.ConfigVersion > configurationFormatVersion {
		return fmt.Errorf("unsupported configuration format version %d", info.ConfigVersion)
	}
	snapshot, err := snapshotFromRawFiles(files)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(info.ConfigurationSHA256), snapshot.Digest) {
		return fmt.Errorf("configuration checksum does not match the manifest")
	}
	manifestFiles := make(map[string]configurationBackupFileInfo, len(info.Files))
	allowedManifest := map[string]bool{"settings.json": true, "destinations.conf": true, "sources.conf": true, "learning.json": true}
	for _, f := range info.Files {
		if !allowedManifest[f.Name] {
			return fmt.Errorf("manifest contains unsupported file %q", f.Name)
		}
		if _, exists := manifestFiles[f.Name]; exists {
			return fmt.Errorf("manifest contains duplicate file %q", f.Name)
		}
		manifestFiles[f.Name] = f
	}
	if len(manifestFiles) != len(snapshot.Files) {
		return fmt.Errorf("manifest file list does not match configuration package")
	}
	for _, f := range snapshot.Files {
		mf, ok := manifestFiles[f.Name]
		if !ok {
			return fmt.Errorf("manifest is missing %s", f.Name)
		}
		if mf.Size != f.Size || !strings.EqualFold(mf.SHA256, f.SHA256) {
			return fmt.Errorf("%s does not match its manifest checksum", f.Name)
		}
	}
	return nil
}

func configurationPackageFromRaw(kind configurationImportKind, source string, files map[string][]byte, info *configurationBackupInfo) (configurationImportPackage, error) {
	pkg := configurationImportPackage{Kind: kind, SourcePath: source, BackupInfo: info}
	if data, ok := files["settings.json"]; ok {
		cfg, err := parseImportedSettings(data)
		if err != nil {
			return configurationImportPackage{}, err
		}
		pkg.Settings = &cfg
	}
	if data, ok := files["destinations.conf"]; ok {
		aliases, err := parseImportedDestinationAliases(data)
		if err != nil {
			return configurationImportPackage{}, err
		}
		pkg.HasDestinations = true
		pkg.Destinations = aliases
	}
	if data, ok := files["sources.conf"]; ok {
		aliases, err := parseImportedSourceAliases(data)
		if err != nil {
			return configurationImportPackage{}, err
		}
		pkg.HasSources = true
		pkg.Sources = aliases
	}
	if data, ok := files["learning.json"]; ok {
		if err := validateImportedLearning(data); err != nil {
			return configurationImportPackage{}, err
		}
		pkg.Learning = append([]byte(nil), data...)
	}
	if data, ok := files["naming-templates.json"]; ok {
		if err := validateImportedNamingTemplates(data); err != nil {
			return configurationImportPackage{}, err
		}
		pkg.NamingTemplates = append([]byte(nil), data...)
	}
	if pkg.Settings == nil && !pkg.HasDestinations && !pkg.HasSources && len(pkg.Learning) == 0 && len(pkg.NamingTemplates) == 0 {
		return configurationImportPackage{}, ErrNoConfiguration
	}
	return pkg, nil
}

func loadConfigurationArchive(path string) (configurationImportPackage, error) {
	st, err := os.Stat(path)
	if err != nil {
		return configurationImportPackage{}, err
	}
	if st.Size() <= 0 || st.Size() > maxConfigurationArchiveBytes {
		return configurationImportPackage{}, fmt.Errorf("configuration archive is empty or too large")
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return configurationImportPackage{}, fmt.Errorf("open configuration archive: %w", err)
	}
	defer zr.Close()
	if len(zr.File) == 0 || len(zr.File) > maxConfigurationArchiveEntries {
		return configurationImportPackage{}, fmt.Errorf("configuration archive contains an unexpected number of entries")
	}
	allowed := map[string]bool{"settings.json": true, "destinations.conf": true, "sources.conf": true, "learning.json": true, "naming-templates.json": true, "backup-info.json": true}
	files := make(map[string][]byte)
	for _, f := range zr.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if name != filepath.Base(name) || !allowed[name] {
			return configurationImportPackage{}, fmt.Errorf("configuration archive contains unsafe or unsupported entry %q", f.Name)
		}
		if _, exists := files[name]; exists {
			return configurationImportPackage{}, fmt.Errorf("configuration archive contains duplicate entry %q", name)
		}
		if f.UncompressedSize64 > maxConfigurationFileBytes {
			return configurationImportPackage{}, fmt.Errorf("configuration entry %s is too large", name)
		}
		rc, err := f.Open()
		if err != nil {
			return configurationImportPackage{}, err
		}
		data, readErr := io.ReadAll(io.LimitReader(rc, maxConfigurationFileBytes+1))
		closeErr := rc.Close()
		if readErr != nil {
			return configurationImportPackage{}, readErr
		}
		if closeErr != nil {
			return configurationImportPackage{}, closeErr
		}
		if len(data) > maxConfigurationFileBytes {
			return configurationImportPackage{}, fmt.Errorf("configuration entry %s is too large", name)
		}
		files[name] = data
	}
	manifestData, ok := files["backup-info.json"]
	if !ok {
		return configurationImportPackage{}, fmt.Errorf("configuration archive is missing backup-info.json")
	}
	delete(files, "backup-info.json")
	var info configurationBackupInfo
	if err := json.Unmarshal(manifestData, &info); err != nil {
		return configurationImportPackage{}, fmt.Errorf("invalid backup-info.json: %w", err)
	}
	if err := validateConfigurationManifest(info, files); err != nil {
		return configurationImportPackage{}, err
	}
	return configurationPackageFromRaw(configurationImportArchive, path, files, &info)
}

func loadConfigurationImport(path string) (configurationImportPackage, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return configurationImportPackage{}, fmt.Errorf("configuration file is required")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".dcmconf":
		return loadConfigurationArchive(path)
	case ".json":
		data, err := os.ReadFile(path)
		if err != nil {
			return configurationImportPackage{}, err
		}
		cfg, err := parseImportedSettings(data)
		if err != nil {
			return configurationImportPackage{}, err
		}
		return configurationImportPackage{Kind: configurationImportSettings, SourcePath: path, Settings: &cfg}, nil
	case ".conf":
		data, err := os.ReadFile(path)
		if err != nil {
			return configurationImportPackage{}, err
		}
		aliases, err := parseImportedDestinationAliases(data)
		if err != nil {
			return configurationImportPackage{}, err
		}
		return configurationImportPackage{Kind: configurationImportDestinations, SourcePath: path, HasDestinations: true, Destinations: aliases}, nil
	default:
		return configurationImportPackage{}, fmt.Errorf("unsupported configuration file type %q", ext)
	}
}

func loadConfigurationSnapshotDirectory(dir string, kind configurationImportKind) (configurationImportPackage, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return configurationImportPackage{}, fmt.Errorf("backup folder is unavailable")
	}
	manifestData, err := os.ReadFile(filepath.Join(dir, "backup-info.json"))
	if err != nil {
		return configurationImportPackage{}, err
	}
	var info configurationBackupInfo
	if err := json.Unmarshal(manifestData, &info); err != nil {
		return configurationImportPackage{}, fmt.Errorf("invalid backup manifest: %w", err)
	}
	files := make(map[string][]byte)
	for _, name := range configurationFileNames() {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return configurationImportPackage{}, err
		}
		if len(data) > maxConfigurationFileBytes {
			return configurationImportPackage{}, fmt.Errorf("backup file %s is too large", name)
		}
		files[name] = data
	}
	if err := validateConfigurationManifest(info, files); err != nil {
		return configurationImportPackage{}, err
	}
	return configurationPackageFromRaw(kind, dir, files, &info)
}

func latestConfigurationBackupInfo(backupRoot string) (configurationBackupInfo, bool) {
	pkg, err := loadConfigurationSnapshotDirectory(filepath.Join(strings.TrimSpace(backupRoot), "latest"), configurationImportLatest)
	if err != nil || pkg.BackupInfo == nil {
		return configurationBackupInfo{}, false
	}
	return *pkg.BackupInfo, true
}

func readExistingAliases(configDir string) ([]DestinationAlias, error) {
	path := filepath.Join(configDir, "destinations.conf")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}
	return parseImportedDestinationAliases(data)
}

func readExistingSourceAliases(configDir string) ([]SourceAlias, error) {
	path := filepath.Join(configDir, "sources.conf")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}
	return parseImportedSourceAliases(data)
}

func mergeImportedSourceAliases(existing, imported []SourceAlias, mode aliasImportMode) ([]SourceAlias, int, error) {
	names := make(map[string]bool, len(existing))
	for _, a := range existing {
		names[strings.ToLower(strings.TrimSpace(a.Name))] = true
	}
	conflicts := 0
	for _, a := range imported {
		if err := validateSourceAlias(a); err != nil {
			return nil, 0, err
		}
		if names[strings.ToLower(strings.TrimSpace(a.Name))] {
			conflicts++
		}
	}
	if mode == aliasReplaceAll {
		return append([]SourceAlias(nil), imported...), conflicts, nil
	}
	if mode != aliasMergeKeepExisting && mode != aliasMergeUseImported {
		return nil, 0, fmt.Errorf("unsupported alias import mode")
	}
	out := append([]SourceAlias(nil), existing...)
	index := map[string]int{}
	for i, a := range out {
		index[strings.ToLower(strings.TrimSpace(a.Name))] = i
	}
	for _, a := range imported {
		key := strings.ToLower(strings.TrimSpace(a.Name))
		if idx, ok := index[key]; ok {
			if mode == aliasMergeUseImported {
				out[idx] = a
			}
			continue
		}
		index[key] = len(out)
		out = append(out, a)
	}
	return out, conflicts, nil
}

type configFileRollback struct {
	path    string
	existed bool
	data    []byte
}

func writeConfigurationFilesTransaction(configDir string, writes map[string][]byte) error {
	return writeConfigurationFilesTransactionMode(configDir, writes, false)
}

func writeConfigurationFilesTransactionMode(configDir string, writes map[string][]byte, removeMissing bool) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	nameSet := make(map[string]bool)
	for name := range writes {
		nameSet[name] = true
	}
	if removeMissing {
		for _, name := range configurationFileNames() {
			nameSet[name] = true
		}
	}
	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)
	rollbacks := make([]configFileRollback, 0, len(names))
	staged := make(map[string]string, len(writes))
	for _, name := range names {
		target := filepath.Join(configDir, name)
		rb := configFileRollback{path: target}
		if data, err := os.ReadFile(target); err == nil {
			rb.existed, rb.data = true, data
		} else if !os.IsNotExist(err) {
			return err
		}
		rollbacks = append(rollbacks, rb)
		if data, ok := writes[name]; ok {
			tmp := target + ".import.tmp"
			_ = os.Remove(tmp)
			if err := os.WriteFile(tmp, data, 0o644); err != nil {
				for _, p := range staged {
					_ = os.Remove(p)
				}
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
		for _, p := range staged {
			_ = os.Remove(p)
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
		} else if removeMissing {
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				rollback()
				return err
			}
		}
		committed++
	}
	return nil
}

func configurationPackageWrites(pkg configurationImportPackage) (map[string][]byte, error) {
	writes := make(map[string][]byte)
	if pkg.Settings != nil {
		data, err := importedSettingsData(*pkg.Settings)
		if err != nil {
			return nil, err
		}
		writes["settings.json"] = data
	}
	if pkg.HasDestinations {
		data, err := destinationAliasesData(pkg.Destinations)
		if err != nil {
			return nil, err
		}
		writes["destinations.conf"] = data
	}
	if pkg.HasSources {
		data, err := sourceAliasesData(pkg.Sources)
		if err != nil {
			return nil, err
		}
		writes["sources.conf"] = data
	}
	if len(pkg.Learning) > 0 {
		if err := validateImportedLearning(pkg.Learning); err != nil {
			return nil, err
		}
		writes["learning.json"] = append([]byte(nil), pkg.Learning...)
	}
	if len(pkg.NamingTemplates) > 0 {
		if err := validateImportedNamingTemplates(pkg.NamingTemplates); err != nil {
			return nil, err
		}
		writes["naming-templates.json"] = append([]byte(nil), pkg.NamingTemplates...)
	}
	return writes, nil
}

func applyConfigurationImport(configDir, backupRoot, appVersion string, pkg configurationImportPackage, mode aliasImportMode, now time.Time) (configurationImportResult, error) {
	result := configurationImportResult{Kind: pkg.Kind}
	if pkg.Settings == nil && !pkg.HasDestinations && !pkg.HasSources && len(pkg.Learning) == 0 && len(pkg.NamingTemplates) == 0 {
		return result, ErrNoConfiguration
	}

	if current, err := collectConfigurationSnapshot(configDir); err == nil && len(current.Files) > 0 {
		backup, backupErr := createConfigurationBackup(configDir, backupRoot, appVersion, "before-import", now)
		if backupErr != nil {
			return result, fmt.Errorf("back up current configuration before import: %w", backupErr)
		}
		result.PreImportBackupDir = backup.LatestDir
	} else if err != nil && err != ErrNoConfiguration {
		return result, err
	}

	writes := make(map[string][]byte)
	if pkg.Settings != nil {
		data, err := importedSettingsData(*pkg.Settings)
		if err != nil {
			return result, err
		}
		writes["settings.json"] = data
		result.SettingsImported = true
	}
	if pkg.HasDestinations {
		existing, err := readExistingAliases(configDir)
		if err != nil {
			return result, fmt.Errorf("read existing destinations: %w", err)
		}
		merged, conflicts, err := mergeImportedDestinationAliases(existing, pkg.Destinations, mode)
		if err != nil {
			return result, err
		}
		data, err := destinationAliasesData(merged)
		if err != nil {
			return result, err
		}
		writes["destinations.conf"] = data
		result.DestinationsImported = len(pkg.Destinations)
		result.AliasConflicts += conflicts
	}

	if pkg.HasSources {
		existing, err := readExistingSourceAliases(configDir)
		if err != nil {
			return result, fmt.Errorf("read existing sources: %w", err)
		}
		merged, conflicts, err := mergeImportedSourceAliases(existing, pkg.Sources, mode)
		if err != nil {
			return result, err
		}
		data, err := sourceAliasesData(merged)
		if err != nil {
			return result, err
		}
		writes["sources.conf"] = data
		result.SourcesImported = len(pkg.Sources)
		result.AliasConflicts += conflicts
	}
	if len(pkg.Learning) > 0 {
		if err := validateImportedLearning(pkg.Learning); err != nil {
			return result, err
		}
		writes["learning.json"] = append([]byte(nil), pkg.Learning...)
		result.LearningImported = true
	}
	if len(pkg.NamingTemplates) > 0 {
		if err := validateImportedNamingTemplates(pkg.NamingTemplates); err != nil {
			return result, err
		}
		writes["naming-templates.json"] = append([]byte(nil), pkg.NamingTemplates...)
		result.NamingTemplatesImported = true
	}

	if err := writeConfigurationFilesTransaction(configDir, writes); err != nil {
		return result, fmt.Errorf("apply configuration import: %w", err)
	}
	backup, err := createConfigurationBackup(configDir, backupRoot, appVersion, "after-import", now.Add(time.Nanosecond))
	if err != nil {
		result.Warning = "Configuration imported, but the post-import backup failed: " + err.Error()
		return result, nil
	}
	result.PostImportBackupDir = backup.LatestDir
	return result, nil
}

func restoreLatestConfiguration(configDir, backupRoot, appVersion string, now time.Time) (configurationImportResult, error) {
	pkg, err := loadConfigurationSnapshotDirectory(filepath.Join(backupRoot, "latest"), configurationImportLatest)
	if err != nil {
		return configurationImportResult{}, fmt.Errorf("load latest configuration backup: %w", err)
	}
	result := configurationImportResult{Kind: configurationImportLatest, SettingsImported: pkg.Settings != nil, DestinationsImported: len(pkg.Destinations), SourcesImported: len(pkg.Sources), LearningImported: len(pkg.Learning) > 0, NamingTemplatesImported: len(pkg.NamingTemplates) > 0}
	if current, err := collectConfigurationSnapshot(configDir); err == nil && len(current.Files) > 0 {
		backup, backupErr := createConfigurationBackup(configDir, backupRoot, appVersion, "before-restore", now)
		if backupErr != nil {
			return result, fmt.Errorf("back up current configuration before restore: %w", backupErr)
		}
		result.PreImportBackupDir = backup.LatestDir
	} else if err != nil && err != ErrNoConfiguration {
		return result, err
	}
	writes, err := configurationPackageWrites(pkg)
	if err != nil {
		return result, err
	}
	if err := writeConfigurationFilesTransactionMode(configDir, writes, true); err != nil {
		return result, fmt.Errorf("restore configuration: %w", err)
	}
	backup, err := createConfigurationBackup(configDir, backupRoot, appVersion, "after-restore", now.Add(time.Nanosecond))
	if err != nil {
		result.Warning = "Configuration restored, but the post-restore backup failed: " + err.Error()
		return result, nil
	}
	result.PostImportBackupDir = backup.LatestDir
	return result, nil
}
