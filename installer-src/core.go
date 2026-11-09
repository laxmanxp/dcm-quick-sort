package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	productName    = "DCM - Quick Sort"
	productVersion = "1.0.71"
	publisherName  = "Laxman Mahanto"
)

type settings struct {
	DefaultSource  string `json:"default_source"`
	PDFDefaultView string `json:"pdf_default_view,omitempty"`
	VaultRoot      string `json:"vault_root,omitempty"`
}

func parseInstallerArgs(args []string) (silent, uninstall bool, source string) {
	for _, raw := range args {
		a := strings.TrimSpace(raw)
		lower := strings.ToLower(a)
		switch lower {
		case "/s", "/silent", "--silent", "-silent":
			silent = true
		case "/uninstall", "--uninstall", "-uninstall":
			uninstall = true
		default:
			for _, prefix := range []string{"/source=", "--source=", "-source="} {
				if strings.HasPrefix(lower, prefix) {
					source = strings.Trim(strings.TrimSpace(a[len(prefix):]), `"`)
					break
				}
			}
		}
	}
	return
}

func defaultSourceFolder(home string) string {
	p := filepath.Join(home, "Downloads")
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p
	}
	return home
}

func settingsJSON(source string) ([]byte, error) {
	return json.MarshalIndent(settings{DefaultSource: filepath.Clean(source)}, "", "  ")
}

func writeSettings(path, source string) error {
	var cfg settings
	if existing, err := os.ReadFile(path); err == nil {
		cfg, err = validateInstallerSettings(existing)
		if err != nil {
			return fmt.Errorf("read existing settings: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read existing settings: %w", err)
	}
	// An upgrade changes the source folder while keeping the app's preferences.
	cfg.DefaultSource = filepath.Clean(source)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

type explorerShellEntry struct {
	Key     string
	Label   string
	Icon    string
	Command string
}

func explorerShellEntries(appExe string) []explorerShellEntry {
	quoted := `"` + appExe + `"`
	const label = "Sort with DCM Quick Sort"
	return []explorerShellEntry{
		{Key: `Software\Classes\*\shell\DCMQuickSort`, Label: label, Icon: appExe, Command: quoted + ` "%1"`},
		{Key: `Software\Classes\Directory\shell\DCMQuickSort`, Label: label, Icon: appExe, Command: quoted + ` "%1"`},
		{Key: `Software\Classes\Directory\Background\shell\DCMQuickSort`, Label: label, Icon: appExe, Command: quoted + ` "%V"`},
	}
}
