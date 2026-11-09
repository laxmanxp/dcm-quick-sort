package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShouldOfferPreviousRestoreWhenSettingsMissingAndLatestValid(t *testing.T) {
	oldConfig := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, oldConfig, "{\"default_source\":\"D:\\\\Inbox\"}\n", "Office=D:\\Office\n")
	if _, err := createConfigurationBackup(oldConfig, backupRoot, "1.0.29", "before-uninstall", time.Now()); err != nil {
		t.Fatal(err)
	}
	freshConfig := t.TempDir()
	info, ok := shouldOfferPreviousConfigurationRestore(freshConfig, backupRoot)
	if !ok || info.Product != "DCM - Quick Sort" {
		t.Fatalf("info=%#v ok=%v", info, ok)
	}
	if src := latestBackupDefaultSource(backupRoot); src != `D:\Inbox` {
		t.Fatalf("source=%q", src)
	}
}

func TestShouldNotOfferPreviousRestoreDuringNormalUpgrade(t *testing.T) {
	configDir := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, configDir, "{\"default_source\":\"C:\\\\Downloads\"}\n", "Office=C:\\Office\n")
	if _, err := createConfigurationBackup(configDir, backupRoot, "1.0.29", "current", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, ok := shouldOfferPreviousConfigurationRestore(configDir, backupRoot); ok {
		t.Fatal("restore should not be offered when current settings already exist")
	}
}

func TestRestoreLatestBackupFilesValidatesAndRestores(t *testing.T) {
	oldConfig := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, oldConfig, "{\"default_source\":\"D:\\\\Inbox\"}\n", "Office=D:\\Office\n")
	if _, err := createConfigurationBackup(oldConfig, backupRoot, "1.0.29", "before-uninstall", time.Now()); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	info, err := restoreLatestBackupFiles(dest, backupRoot)
	if err != nil {
		t.Fatal(err)
	}
	if info.ConfigVersion != 2 {
		t.Fatalf("info=%#v", info)
	}
	settingsData, err := os.ReadFile(filepath.Join(dest, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(settingsData), `D:\\Inbox`) {
		t.Fatalf("settings=%s", settingsData)
	}
	destData, err := os.ReadFile(filepath.Join(dest, "destinations.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(destData), `Office=D:\Office`) {
		t.Fatalf("destinations=%s", destData)
	}
}

func TestRestoreLatestBackupRejectsTampering(t *testing.T) {
	oldConfig := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, oldConfig, "{\"default_source\":\"D:\\\\Inbox\"}\n", "Office=D:\\Office\n")
	if _, err := createConfigurationBackup(oldConfig, backupRoot, "1.0.29", "before-uninstall", time.Now()); err != nil {
		t.Fatal(err)
	}
	latestSettings := filepath.Join(backupRoot, "latest", "settings.json")
	if err := os.WriteFile(latestSettings, []byte("{\"default_source\":\"X:\\\\Tampered\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := restoreLatestBackupFiles(t.TempDir(), backupRoot); err == nil {
		t.Fatal("expected tampered backup to be rejected")
	}
}

func TestRestoreBackupArg(t *testing.T) {
	if !hasRestoreBackupArg([]string{"/S", "/RESTOREBACKUP"}) {
		t.Fatal("restore arg not detected")
	}
	if hasRestoreBackupArg([]string{"/S"}) {
		t.Fatal("unexpected restore arg")
	}
}
