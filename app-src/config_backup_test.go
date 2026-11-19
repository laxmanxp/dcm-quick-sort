package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTestConfig(t *testing.T, dir, settingsText, destinationsText string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if settingsText != "" {
		if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(settingsText), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if destinationsText != "" {
		if err := os.WriteFile(filepath.Join(dir, "destinations.conf"), []byte(destinationsText), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConfigurationSnapshotDigestStable(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, `{"default_source":"C:/Downloads"}\n`, "Office=C:\\Office\n")
	a, err := collectConfigurationSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := collectConfigurationSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest == "" || a.Digest != b.Digest {
		t.Fatalf("digest not stable: %q %q", a.Digest, b.Digest)
	}
	if len(a.Files) != 2 || a.Files[0].Name != "destinations.conf" || a.Files[1].Name != "settings.json" {
		t.Fatalf("unexpected snapshot order: %#v", a.Files)
	}
}

func TestConfigurationSnapshotDigestChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, `{"default_source":"C:/Downloads"}\n`, "Office=C:\\Office\n")
	a, err := collectConfigurationSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "destinations.conf"), []byte("Office=D:\\Office\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := collectConfigurationSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest == b.Digest {
		t.Fatal("configuration digest did not change")
	}
}

func TestConfigurationSnapshotRequiresAtLeastOneKnownFile(t *testing.T) {
	_, err := collectConfigurationSnapshot(t.TempDir())
	if !errors.Is(err, ErrNoConfiguration) {
		t.Fatalf("got %v want ErrNoConfiguration", err)
	}
}

func TestBackupCreatesLatestAndHistory(t *testing.T) {
	configDir := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, configDir, `{"default_source":"C:/Downloads"}\n`, "Office=C:\\Office\n")
	now := time.Date(2026, 9, 21, 16, 30, 45, 123000000, time.FixedZone("IST", 5*3600+1800))

	result, err := createConfigurationBackup(configDir, backupRoot, "1.0.29", "test", now)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HistoryCreated || result.HistoryDir == "" {
		t.Fatalf("history not created: %#v", result)
	}
	for _, dir := range []string{result.LatestDir, result.HistoryDir} {
		for _, name := range []string{"settings.json", "destinations.conf", "backup-info.json"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				t.Fatalf("missing %s in %s: %v", name, dir, err)
			}
		}
	}
	data, err := os.ReadFile(filepath.Join(result.LatestDir, "backup-info.json"))
	if err != nil {
		t.Fatal(err)
	}
	var info configurationBackupInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatal(err)
	}
	if info.Product != "DCM - Quick Sort" || info.ConfigVersion != 2 || info.AppVersion != "1.0.29" || info.Reason != "test" {
		t.Fatalf("unexpected metadata: %#v", info)
	}
	if info.ConfigurationSHA256 != result.Digest || info.ConfigurationSHA256 == "" {
		t.Fatalf("bad metadata digest: %#v result=%#v", info, result)
	}
}

func TestBackupSuppressesDuplicateHistoryButRefreshesLatest(t *testing.T) {
	configDir := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, configDir, `{"default_source":"C:/Downloads"}\n`, "Office=C:\\Office\n")
	firstTime := time.Date(2026, 9, 21, 16, 30, 0, 0, time.UTC)
	secondTime := firstTime.Add(10 * time.Minute)

	first, err := createConfigurationBackup(configDir, backupRoot, "1.0.29", "first", firstTime)
	if err != nil {
		t.Fatal(err)
	}
	second, err := createConfigurationBackup(configDir, backupRoot, "1.0.29", "second", secondTime)
	if err != nil {
		t.Fatal(err)
	}
	if !first.HistoryCreated || second.HistoryCreated {
		t.Fatalf("duplicate suppression failed: first=%#v second=%#v", first, second)
	}
	entries, err := os.ReadDir(filepath.Join(backupRoot, "history"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("history entries=%d want 1", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(backupRoot, "latest", "backup-info.json"))
	if err != nil {
		t.Fatal(err)
	}
	var info configurationBackupInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatal(err)
	}
	if info.Reason != "second" || info.CreatedAt != secondTime.Format(time.RFC3339Nano) {
		t.Fatalf("latest was not refreshed: %#v", info)
	}
}

func TestBackupCreatesHistoryAfterConfigurationChange(t *testing.T) {
	configDir := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, configDir, `{"default_source":"C:/Downloads"}\n`, "Office=C:\\Office\n")
	now := time.Date(2026, 9, 21, 16, 30, 0, 0, time.UTC)
	if _, err := createConfigurationBackup(configDir, backupRoot, "1.0.29", "first", now); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "destinations.conf"), []byte("Office=D:\\Office\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := createConfigurationBackup(configDir, backupRoot, "1.0.29", "changed", now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !result.HistoryCreated {
		t.Fatal("changed configuration did not create history")
	}
	entries, err := os.ReadDir(filepath.Join(backupRoot, "history"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("history entries=%d want 2", len(entries))
	}
}

func TestExportConfigurationArchiveContainsManifestAndConfig(t *testing.T) {
	configDir := t.TempDir()
	writeTestConfig(t, configDir, `{"default_source":"C:/Downloads"}\n`, "Office=C:\\Office\n")
	out := filepath.Join(t.TempDir(), "config.dcmconf")
	now := time.Date(2026, 9, 21, 16, 45, 0, 0, time.UTC)
	result, err := exportConfigurationArchive(configDir, out, "1.0.29", now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Digest == "" || result.Path != out {
		t.Fatalf("bad export result: %#v", result)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	got := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		got[f.Name] = string(b)
	}
	for _, name := range []string{"settings.json", "destinations.conf", "backup-info.json"} {
		if _, ok := got[name]; !ok {
			t.Fatalf("archive missing %s: %#v", name, got)
		}
	}
	if !strings.Contains(got["backup-info.json"], `"config_version": 2`) || !strings.Contains(got["backup-info.json"], `"reason": "manual-export"`) {
		t.Fatalf("bad manifest: %s", got["backup-info.json"])
	}
}

func TestDefaultExportFileNameIsSafeAndTraceable(t *testing.T) {
	now := time.Date(2026, 9, 21, 16, 45, 6, 0, time.UTC)
	got := configurationExportFileName(now)
	if got != "DCM-QuickSort-Configuration-2026-09-21_164506.dcmconf" {
		t.Fatalf("got %q", got)
	}
}

func TestConfigurationSnapshotIncludesLearningModelWhenPresent(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, `{"default_source":"C:/Downloads"}\n`, "Office=C:\\Office\n")
	if err := os.WriteFile(filepath.Join(dir, "learning.json"), []byte(`{"version":1,"total_observations":7}`), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := collectConfigurationSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Files) != 3 {
		t.Fatalf("files=%d want 3: %#v", len(snapshot.Files), snapshot.Files)
	}
	if snapshot.Files[0].Name != "destinations.conf" || snapshot.Files[1].Name != "learning.json" || snapshot.Files[2].Name != "settings.json" {
		t.Fatalf("unexpected snapshot order: %#v", snapshot.Files)
	}
}

func TestNamingTemplatesIncludedInConfigurationBackupSet(t *testing.T) {
	found := false
	for _, name := range configurationFileNames() {
		if name == "naming-templates.json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("naming-templates.json is not included in configuration backup set")
	}
}
