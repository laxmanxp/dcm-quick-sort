package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestD4NamingTemplatesSurviveExportImportAndLatestRestore(t *testing.T) {
	for _, mode := range []string{"archive-import", "latest-restore"} {
		t.Run(mode, func(t *testing.T) {
			original := t.TempDir()
			files := map[string]string{
				"settings.json":         `{"default_source":"D:\\Inbox","pdf_default_view":"fit_page","vault_root":"E:\\Vault"}`,
				"destinations.conf":     "Office=E:\\Vault\\Office\n",
				"sources.conf":          "Inbox=D:\\Inbox\n",
				"learning.json":         `{"version":1,"total_observations":7,"destination_counts":{"Office":7}}`,
				"naming-templates.json": `{"active":"Voucher","templates":[{"name":"Voucher","pattern":"{office}-{name}","variables":[{"name":"office","value":"BDO"}]}]}`,
			}
			for name, data := range files {
				if err := os.WriteFile(filepath.Join(original, name), []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			restored := t.TempDir()
			writeTestConfig(t, restored, `{"default_source":"C:\\Old"}`, "Old=C:\\Old\n")
			backupRoot := filepath.Join(t.TempDir(), "Backup")
			now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
			var result configurationImportResult
			var err error
			if mode == "archive-import" {
				archive := filepath.Join(t.TempDir(), "all-settings.dcmconf")
				if _, err := exportConfigurationArchive(original, archive, "1.0.70", now); err != nil {
					t.Fatal(err)
				}
				pkg, loadErr := loadConfigurationImport(archive)
				if loadErr != nil {
					t.Fatalf("the app must import its own export with naming templates: %v", loadErr)
				}
				result, err = applyConfigurationImport(restored, backupRoot, "1.0.70", pkg, aliasReplaceAll, now)
			} else {
				if _, err := createConfigurationBackup(original, backupRoot, "1.0.70", "before-uninstall", now); err != nil {
					t.Fatal(err)
				}
				result, err = restoreLatestConfiguration(restored, backupRoot, "1.0.70", now.Add(time.Minute))
			}
			if err != nil {
				t.Fatalf("restore a complete configuration: %v", err)
			}
			if !result.SettingsImported || !result.LearningImported || !result.NamingTemplatesImported || result.DestinationsImported != 1 || result.SourcesImported != 1 {
				t.Fatalf("some exported settings were omitted: %#v", result)
			}
			for _, name := range []string{"naming-templates.json", "learning.json"} {
				got, err := os.ReadFile(filepath.Join(restored, name))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, []byte(files[name])) {
					t.Fatalf("%s changed during restore: got %s, want %s", name, got, files[name])
				}
			}
			settingsData, err := os.ReadFile(filepath.Join(restored, "settings.json"))
			if err != nil {
				t.Fatal(err)
			}
			var cfg localSettings
			if err := json.Unmarshal(settingsData, &cfg); err != nil {
				t.Fatal(err)
			}
			if cfg.DefaultSource != `D:\Inbox` || cfg.PDFDefaultView != "fit_page" || cfg.VaultRoot != `E:\Vault` {
				t.Fatalf("settings changed during restore: %#v", cfg)
			}
		})
	}
}

func TestD4TamperedNamingTemplateBackupLeavesCurrentConfigurationUntouched(t *testing.T) {
	original := t.TempDir()
	writeTestConfig(t, original, `{"default_source":"D:\\Inbox"}`, "")
	templates := []byte(`{"active":"Default","templates":[{"name":"Default","pattern":"{name}"}]}`)
	if err := os.WriteFile(filepath.Join(original, "naming-templates.json"), templates, 0o644); err != nil {
		t.Fatal(err)
	}
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	if _, err := createConfigurationBackup(original, backupRoot, "1.0.70", "test", now); err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(templates, []byte("{name}"), []byte("{original}"), 1)
	if err := os.WriteFile(filepath.Join(backupRoot, "latest", "naming-templates.json"), tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	current := t.TempDir()
	currentSettings := []byte(`{"default_source":"C:\\Current"}`)
	if err := os.WriteFile(filepath.Join(current, "settings.json"), currentSettings, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := restoreLatestConfiguration(current, backupRoot, "1.0.70", now.Add(time.Minute)); err == nil {
		t.Fatal("restore accepted naming templates whose checksum no longer matches")
	}
	got, err := os.ReadFile(filepath.Join(current, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, currentSettings) {
		t.Fatalf("rejected restore changed existing settings: %s", got)
	}
	if _, err := os.Stat(filepath.Join(current, "naming-templates.json")); !os.IsNotExist(err) {
		t.Fatalf("rejected restore created naming templates: %v", err)
	}
}
