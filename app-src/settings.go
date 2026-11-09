package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const currentAppVersion = "1.0.70"

type localSettings struct {
	DefaultSource  string `json:"default_source"`
	PDFDefaultView string `json:"pdf_default_view,omitempty"`
	VaultRoot      string `json:"vault_root,omitempty"`
}

const (
	pdfDefaultViewFitWidth = "fit_width"
	pdfDefaultViewFitPage  = "fit_page"
)

func normalizePDFDefaultView(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case pdfDefaultViewFitPage:
		return pdfDefaultViewFitPage
	case pdfDefaultViewFitWidth:
		return pdfDefaultViewFitWidth
	default:
		// New installs and older settings default to the reading-friendly width view.
		return pdfDefaultViewFitWidth
	}
}

type DestinationAlias struct {
	Name string
	Path string
}

func settingsDirectory() string {
	if base := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); base != "" {
		return filepath.Join(base, "DCM Quick Sort")
	}
	if base, err := os.UserConfigDir(); err == nil && strings.TrimSpace(base) != "" {
		return filepath.Join(base, "DCM Quick Sort")
	}
	return ""
}

func settingsFilePath() string {
	if dir := settingsDirectory(); dir != "" {
		return filepath.Join(dir, "settings.json")
	}
	return ""
}

func replacementRecoveryDirectory() string {
	if dir := settingsDirectory(); dir != "" {
		return filepath.Join(dir, "recovery", "replacements")
	}
	return filepath.Join(os.TempDir(), "DCM Quick Sort", "recovery", "replacements")
}

func learningFilePath() string {
	if dir := settingsDirectory(); dir != "" {
		return filepath.Join(dir, "learning.json")
	}
	return filepath.Join(os.TempDir(), "DCM Quick Sort", "learning.json")
}

func moveJournalFilePath() string {
	if dir := settingsDirectory(); dir != "" {
		return filepath.Join(dir, "recovery", "move-journal.jsonl")
	}
	return filepath.Join(os.TempDir(), "DCM Quick Sort", "recovery", "move-journal.jsonl")
}

func destinationConfigFilePath() string {
	if dir := settingsDirectory(); dir != "" {
		return filepath.Join(dir, "destinations.conf")
	}
	return ""
}

func loadLocalSettings() localSettings {
	path := settingsFilePath()
	if path == "" {
		return localSettings{PDFDefaultView: pdfDefaultViewFitWidth}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return localSettings{PDFDefaultView: pdfDefaultViewFitWidth}
	}
	var cfg localSettings
	if json.Unmarshal(data, &cfg) != nil {
		return localSettings{PDFDefaultView: pdfDefaultViewFitWidth}
	}
	cfg.DefaultSource = strings.TrimSpace(cfg.DefaultSource)
	cfg.PDFDefaultView = normalizePDFDefaultView(cfg.PDFDefaultView)
	cfg.VaultRoot = strings.TrimSpace(cfg.VaultRoot)
	return cfg
}

func loadDefaultSourceSetting() string {
	cfg := loadLocalSettings()
	source := strings.TrimSpace(cfg.DefaultSource)
	if source == "" {
		return ""
	}
	if st, err := os.Stat(source); err == nil && st.IsDir() {
		return source
	}
	return ""
}

func loadPDFDefaultViewSetting() string {
	return normalizePDFDefaultView(loadLocalSettings().PDFDefaultView)
}

func loadVaultRootSetting() string {
	root := strings.TrimSpace(loadLocalSettings().VaultRoot)
	if root == "" {
		return ""
	}
	if st, err := os.Stat(root); err == nil && st.IsDir() {
		return filepath.Clean(root)
	}
	return ""
}

func savePDFDefaultViewSetting(view, fallbackSource string) error {
	path := settingsFilePath()
	if path == "" {
		return fmt.Errorf("user configuration directory is unavailable")
	}
	cfg := loadLocalSettings()
	cfg.PDFDefaultView = normalizePDFDefaultView(view)
	if strings.TrimSpace(cfg.DefaultSource) == "" {
		fallbackSource = strings.TrimSpace(fallbackSource)
		if st, err := os.Stat(fallbackSource); err == nil && st.IsDir() {
			cfg.DefaultSource = fallbackSource
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
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

func saveVaultRootSetting(root, fallbackSource string) error {
	path := settingsFilePath()
	if path == "" {
		return fmt.Errorf("user configuration directory is unavailable")
	}
	root = strings.TrimSpace(root)
	if root != "" {
		st, err := os.Stat(root)
		if err != nil || !st.IsDir() {
			return fmt.Errorf("vault root must be an existing folder")
		}
		root = filepath.Clean(root)
	}
	cfg := loadLocalSettings()
	cfg.VaultRoot = root
	if strings.TrimSpace(cfg.DefaultSource) == "" {
		fallbackSource = strings.TrimSpace(fallbackSource)
		if st, err := os.Stat(fallbackSource); err == nil && st.IsDir() {
			cfg.DefaultSource = fallbackSource
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
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

func defaultDestinationAliasName(path string) string {
	name := strings.TrimSpace(filepath.Base(filepath.Clean(path)))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "Default"
	}
	return name
}

func ensureDestinationConfig(defaultPath string) error {
	path := destinationConfigFilePath()
	if path == "" {
		return fmt.Errorf("user configuration directory is unavailable")
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	defaultPath = strings.TrimSpace(defaultPath)
	var b strings.Builder
	b.WriteString("# DCM - Quick Sort Vault destination favorites\n")
	b.WriteString("# Managed by DCM. Use Vault Destinations in the app.\n")
	b.WriteString("# Format: Name=Vault/logical/location\n")
	if defaultPath != "" {
		b.WriteString(defaultDestinationAliasName(defaultPath))
		b.WriteString("=")
		b.WriteString(defaultPath)
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func loadDestinationAliases(defaultPath string) ([]DestinationAlias, error) {
	if err := ensureDestinationConfig(defaultPath); err != nil {
		return nil, err
	}
	f, err := os.Open(destinationConfigFilePath())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	aliases := make([]DestinationAlias, 0, 8)
	byName := make(map[string]int)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		name, path, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		path = strings.TrimSpace(path)
		if name == "" || path == "" {
			continue
		}
		key := strings.ToLower(name)
		if idx, exists := byName[key]; exists {
			aliases[idx].Path = path
			continue
		}
		byName[key] = len(aliases)
		aliases = append(aliases, DestinationAlias{Name: name, Path: path})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return aliases, nil
}

func normalizeDestinationPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimRight(path, "\\/")
	return path
}

func destinationAliasIndexForPath(aliases []DestinationAlias, path string) int {
	want := normalizeDestinationPath(path)
	if want == "" {
		return -1
	}
	for i, alias := range aliases {
		if strings.EqualFold(normalizeDestinationPath(alias.Path), want) {
			return i
		}
	}
	return -1
}

func validateDestinationAlias(alias DestinationAlias) error {
	alias.Name = strings.TrimSpace(alias.Name)
	alias.Path = strings.TrimSpace(alias.Path)
	if alias.Name == "" {
		return fmt.Errorf("destination name is required")
	}
	if alias.Path == "" {
		return fmt.Errorf("destination folder is required")
	}
	if strings.ContainsAny(alias.Name, "=\r\n") {
		return fmt.Errorf("destination name cannot contain = or line breaks")
	}
	return nil
}

func upsertDestinationAlias(aliases []DestinationAlias, alias DestinationAlias) ([]DestinationAlias, int, error) {
	alias.Name = strings.TrimSpace(alias.Name)
	alias.Path = strings.TrimSpace(alias.Path)
	if err := validateDestinationAlias(alias); err != nil {
		return aliases, -1, err
	}
	out := append([]DestinationAlias(nil), aliases...)
	for i := range out {
		if strings.EqualFold(strings.TrimSpace(out[i].Name), alias.Name) {
			out[i] = alias
			return out, i, nil
		}
	}
	out = append(out, alias)
	return out, len(out) - 1, nil
}

func deleteDestinationAlias(aliases []DestinationAlias, index int) []DestinationAlias {
	if index < 0 || index >= len(aliases) {
		return append([]DestinationAlias(nil), aliases...)
	}
	out := make([]DestinationAlias, 0, len(aliases)-1)
	out = append(out, aliases[:index]...)
	out = append(out, aliases[index+1:]...)
	return out
}

func writeDestinationAliases(aliases []DestinationAlias) error {
	path := destinationConfigFilePath()
	if path == "" {
		return fmt.Errorf("user configuration directory is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# DCM - Quick Sort Vault destination favorites\n")
	b.WriteString("# Managed by DCM. Use Vault Destinations in the app.\n")
	b.WriteString("# Format: Name=Vault/logical/location\n")
	for _, alias := range aliases {
		alias.Name = strings.TrimSpace(alias.Name)
		alias.Path = strings.TrimSpace(alias.Path)
		if validateDestinationAlias(alias) != nil {
			continue
		}
		b.WriteString(alias.Name)
		b.WriteString("=")
		b.WriteString(alias.Path)
		b.WriteString("\n")
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func backupCurrentConfiguration(reason string) (configurationBackupResult, error) {
	dir := settingsDirectory()
	if dir == "" {
		return configurationBackupResult{}, fmt.Errorf("user configuration directory is unavailable")
	}
	root := configurationBackupRoot()
	if root == "" {
		return configurationBackupResult{}, fmt.Errorf("backup folder is unavailable")
	}
	return createConfigurationBackup(dir, root, currentAppVersion, reason, time.Now())
}

func exportCurrentConfiguration() (configurationExportResult, error) {
	dir := settingsDirectory()
	if dir == "" {
		return configurationExportResult{}, fmt.Errorf("user configuration directory is unavailable")
	}
	root := configurationBackupRoot()
	if root == "" {
		return configurationExportResult{}, fmt.Errorf("backup folder is unavailable")
	}
	return exportConfigurationToBackupFolder(dir, root, currentAppVersion, time.Now())
}
