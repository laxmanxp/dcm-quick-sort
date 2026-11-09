package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestD5InstallerRestoresCurrentAndLegacySettings(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{"legacy", `{"default_source":"D:\\Inbox"}`},
		{"pdf-only", `{"default_source":"D:\\Inbox","pdf_default_view":"fit_width"}`},
		{"vault-only", `{"default_source":"D:\\Inbox","vault_root":"E:\\Vault"}`},
		{"fit-page-and-vault", `{"default_source":"D:\\Inbox","pdf_default_view":"fit_page","vault_root":"E:\\Vault"}`},
		{"fit-width-and-vault", `{"default_source":"D:\\Inbox","pdf_default_view":"fit_width","vault_root":"E:\\Vault"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := t.TempDir()
			writeTestConfig(t, original, tc.data, "")
			if !currentSettingsUsable(original) {
				t.Fatal("installer rejected settings saved by the app")
			}
			backupRoot := filepath.Join(t.TempDir(), "Backup")
			if _, err := createConfigurationBackup(original, backupRoot, "1.0.70", "before-uninstall", time.Now()); err != nil {
				t.Fatal(err)
			}
			if _, offer := shouldOfferPreviousConfigurationRestore(original, backupRoot); offer {
				t.Fatal("normal upgrade must keep current configuration instead of offering previous settings")
			}
			if source := latestBackupDefaultSource(backupRoot); source != `D:\Inbox` {
				t.Fatalf("backup source=%q, want D:\\Inbox", source)
			}
			restored := t.TempDir()
			if _, err := restoreLatestBackupFiles(restored, backupRoot); err != nil {
				t.Fatalf("restore app settings: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(restored, "settings.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, []byte(tc.data)) {
				t.Fatalf("installer changed settings during restore: got %s, want %s", got, tc.data)
			}
		})
	}
}

func TestD5InstallerUpgradePreservesPDFAndVaultSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	initial := []byte(`{"default_source":"D:\\OldInbox","pdf_default_view":"fit_page","vault_root":"E:\\Personal Vault"}`)
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}
	newSource := filepath.Join(t.TempDir(), "New Inbox")
	if err := writeSettings(path, newSource); err != nil {
		t.Fatalf("change the source during upgrade: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"default_source": filepath.Clean(newSource), "pdf_default_view": "fit_page", "vault_root": `E:\Personal Vault`}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("upgrade changed %s: got %q, want %q", key, got[key], value)
		}
	}
}

func TestD5InstallerRejectsInvalidSettingsWithoutRestoringThem(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{"unknown-key", `{"default_source":"D:\\Inbox","unrecognized_setting":"value"}`},
		{"pdf-number", `{"default_source":"D:\\Inbox","pdf_default_view":42}`},
		{"pdf-object", `{"default_source":"D:\\Inbox","pdf_default_view":{"mode":"fit_page"}}`},
		{"vault-bool", `{"default_source":"D:\\Inbox","vault_root":true}`},
		{"vault-array", `{"default_source":"D:\\Inbox","vault_root":["E:\\Vault"]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := t.TempDir()
			writeTestConfig(t, original, tc.data, "")
			backupRoot := filepath.Join(t.TempDir(), "Backup")
			if _, err := createConfigurationBackup(original, backupRoot, "1.0.70", "test", time.Now()); err != nil {
				t.Fatal(err)
			}
			current := t.TempDir()
			currentSettings := []byte(`{"default_source":"C:\\Current"}`)
			if err := os.WriteFile(filepath.Join(current, "settings.json"), currentSettings, 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := restoreLatestBackupFiles(current, backupRoot); err == nil {
				t.Fatal("installer restored invalid settings")
			}
			got, err := os.ReadFile(filepath.Join(current, "settings.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, currentSettings) {
				t.Fatalf("rejected restore overwrote existing settings: %s", got)
			}
		})
	}
}

func TestD4InstallerBackupRestoresAllExportedConfigurationFiles(t *testing.T) {
	original := t.TempDir()
	files := map[string]string{
		"settings.json":         `{"default_source":"D:\\Inbox"}`,
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
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	if _, err := createConfigurationBackup(original, backupRoot, "1.0.70", "before-uninstall", time.Now()); err != nil {
		t.Fatal(err)
	}
	restored := t.TempDir()
	if _, err := restoreLatestBackupFiles(restored, backupRoot); err != nil {
		t.Fatalf("installer must restore every file in the app configuration contract: %v", err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(restored, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, []byte(want)) {
			t.Fatalf("installer changed %s: got %s, want %s", name, got, want)
		}
	}
	if err := os.WriteFile(filepath.Join(backupRoot, "latest", "naming-templates.json"), []byte(`{"active":"Changed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := restoreLatestBackupFiles(restored, backupRoot); err == nil {
		t.Fatal("installer accepted tampered naming templates")
	}
	got, err := os.ReadFile(filepath.Join(restored, "naming-templates.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte(files["naming-templates.json"])) {
		t.Fatalf("rejected restore changed existing naming templates: %s", got)
	}
}
