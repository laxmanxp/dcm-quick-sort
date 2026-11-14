package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseImportedSettingsRequiresDefaultSource(t *testing.T) {
	if _, err := parseImportedSettings([]byte(`{"default_source":""}`)); err == nil {
		t.Fatal("expected blank default source to be rejected")
	}
	got, err := parseImportedSettings([]byte(`{"default_source":"C:\\Inbox"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultSource != `C:\Inbox` {
		t.Fatalf("default source=%q", got.DefaultSource)
	}
}

func TestParseImportedDestinationsStrict(t *testing.T) {
	aliases, err := parseImportedDestinationAliases([]byte("# comment\nOffice=C:\\Office\nPhotos=D:\\Photos\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 2 || aliases[0].Name != "Office" || aliases[1].Path != `D:\Photos` {
		t.Fatalf("aliases=%#v", aliases)
	}
	if _, err := parseImportedDestinationAliases([]byte("broken line\n")); err == nil {
		t.Fatal("expected malformed destination line to fail")
	}
}

func TestMergeDestinationAliasesConflictPolicies(t *testing.T) {
	existing := []DestinationAlias{{Name: "Office", Path: `C:\Office`}, {Name: "Bills", Path: `C:\Bills`}}
	imported := []DestinationAlias{{Name: "office", Path: `D:\Office`}, {Name: "Photos", Path: `D:\Photos`}}

	keep, conflicts, err := mergeImportedDestinationAliases(existing, imported, aliasMergeKeepExisting)
	if err != nil {
		t.Fatal(err)
	}
	if conflicts != 1 || len(keep) != 3 || keep[0].Path != `C:\Office` || keep[2].Name != "Photos" {
		t.Fatalf("keep merge=%#v conflicts=%d", keep, conflicts)
	}

	use, conflicts, err := mergeImportedDestinationAliases(existing, imported, aliasMergeUseImported)
	if err != nil {
		t.Fatal(err)
	}
	if conflicts != 1 || use[0].Path != `D:\Office` {
		t.Fatalf("use merge=%#v conflicts=%d", use, conflicts)
	}

	replace, conflicts, err := mergeImportedDestinationAliases(existing, imported, aliasReplaceAll)
	if err != nil {
		t.Fatal(err)
	}
	if conflicts != 1 || len(replace) != 2 || replace[0].Path != `D:\Office` {
		t.Fatalf("replace=%#v conflicts=%d", replace, conflicts)
	}
}

func writeDCMConfForTest(t *testing.T, path string, files map[string][]byte, mutateManifest func(*configurationBackupInfo)) {
	t.Helper()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	// Use the production snapshot digest algorithm by staging the files.
	stage := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(stage, name), files[name], 0o644); err != nil {
			t.Fatal(err)
		}
	}
	snap, err := collectConfigurationSnapshot(stage)
	if err != nil {
		t.Fatal(err)
	}
	info := makeConfigurationBackupInfo(snap, "1.0.29", "test-import", time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC))
	if mutateManifest != nil {
		mutateManifest(&info)
	}
	manifest, err := backupInfoJSON(info)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	w, err := zw.Create("backup-info.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(manifest); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDCMConfValidatesManifestAndHashes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.dcmconf")
	files := map[string][]byte{
		"settings.json":     []byte("{\"default_source\":\"C:\\\\Inbox\"}\n"),
		"destinations.conf": []byte("Office=C:\\Office\n"),
	}
	writeDCMConfForTest(t, path, files, nil)
	pkg, err := loadConfigurationImport(path)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Kind != configurationImportArchive || pkg.Settings == nil || len(pkg.Destinations) != 1 {
		t.Fatalf("pkg=%#v", pkg)
	}

	tampered := filepath.Join(t.TempDir(), "tampered.dcmconf")
	writeDCMConfForTest(t, tampered, files, func(info *configurationBackupInfo) {
		for i := range info.Files {
			if info.Files[i].Name == "settings.json" {
				info.Files[i].SHA256 = strings.Repeat("0", 64)
			}
		}
	})
	if _, err := loadConfigurationImport(tampered); err == nil {
		t.Fatal("expected tampered manifest hash to fail")
	}
}

func TestLoadConfigurationImportRejectsUnsafeArchiveEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unsafe.dcmconf")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("../settings.json")
	_, _ = w.Write([]byte(`{"default_source":"C:\\Inbox"}`))
	_ = zw.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfigurationImport(path); err == nil {
		t.Fatal("expected unsafe archive entry to fail")
	}
}

func TestStandaloneImportsOnlyContainTheirOwnArea(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"default_source":"C:\\Inbox"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg, err := loadConfigurationImport(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Settings == nil || pkg.HasDestinations {
		t.Fatalf("settings pkg=%#v", pkg)
	}

	destPath := filepath.Join(dir, "destinations.conf")
	if err := os.WriteFile(destPath, []byte("Office=C:\\Office\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg, err = loadConfigurationImport(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Settings != nil || !pkg.HasDestinations || len(pkg.Destinations) != 1 {
		t.Fatalf("dest pkg=%#v", pkg)
	}
}

func TestApplyConfigurationImportBacksUpAndMerges(t *testing.T) {
	configDir := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, configDir, "{\"default_source\":\"C:\\\\Old\"}\n", "Office=C:\\Office\nBills=C:\\Bills\n")

	importedSettings := localSettings{DefaultSource: `D:\Inbox`}
	pkg := configurationImportPackage{
		Kind:            configurationImportArchive,
		Settings:        &importedSettings,
		HasDestinations: true,
		Destinations:    []DestinationAlias{{Name: "Office", Path: `D:\Office`}, {Name: "Photos", Path: `D:\Photos`}},
	}
	now := time.Date(2026, 9, 21, 18, 0, 0, 0, time.UTC)
	result, err := applyConfigurationImport(configDir, backupRoot, "1.0.30", pkg, aliasMergeKeepExisting, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.PreImportBackupDir == "" || result.AliasConflicts != 1 || result.DestinationsImported != 2 || !result.SettingsImported {
		t.Fatalf("result=%#v", result)
	}
	raw, err := os.ReadFile(filepath.Join(configDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got localSettings
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.DefaultSource != `D:\Inbox` {
		t.Fatalf("settings=%#v", got)
	}
	aliases, err := readDestinationAliasesFromPath(filepath.Join(configDir, "destinations.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 3 || aliases[0].Path != `C:\Office` || aliases[2].Name != "Photos" {
		t.Fatalf("aliases=%#v", aliases)
	}
	if _, err := os.Stat(filepath.Join(backupRoot, "latest", "backup-info.json")); err != nil {
		t.Fatal(err)
	}
}

func TestApplyConfigurationImportStopsWhenPreBackupFails(t *testing.T) {
	configDir := t.TempDir()
	writeTestConfig(t, configDir, "{\"default_source\":\"C:\\\\Old\"}\n", "Office=C:\\Office\n")
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	imported := localSettings{DefaultSource: `D:\Inbox`}
	pkg := configurationImportPackage{Kind: configurationImportSettings, Settings: &imported}
	if _, err := applyConfigurationImport(configDir, blocker, "1.0.30", pkg, aliasMergeKeepExisting, time.Now()); err == nil {
		t.Fatal("expected backup failure")
	}
	raw, _ := os.ReadFile(filepath.Join(configDir, "settings.json"))
	if !strings.Contains(string(raw), `C:\\Old`) {
		t.Fatalf("settings changed despite failed backup: %s", raw)
	}
}

func TestLoadAndRestoreLatestBackup(t *testing.T) {
	sourceConfig := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, sourceConfig, "{\"default_source\":\"D:\\\\Restored\"}\n", "Office=D:\\Office\n")
	if _, err := createConfigurationBackup(sourceConfig, backupRoot, "1.0.29", "before-uninstall", time.Now()); err != nil {
		t.Fatal(err)
	}

	info, ok := latestConfigurationBackupInfo(backupRoot)
	if !ok || info.Product != "DCM - Quick Sort" {
		t.Fatalf("info=%#v ok=%v", info, ok)
	}

	destConfig := t.TempDir()
	result, err := restoreLatestConfiguration(destConfig, backupRoot, "1.0.30", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !result.SettingsImported || result.DestinationsImported != 1 {
		t.Fatalf("result=%#v", result)
	}
	if _, err := os.Stat(filepath.Join(destConfig, "settings.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destConfig, "destinations.conf")); err != nil {
		t.Fatal(err)
	}
}

func TestWriteConfigurationFilesTransactionRollsBackEarlierCommit(t *testing.T) {
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "destinations.conf"), []byte("Office=C:\\Office\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Force the second commit (settings.json) to fail after destinations.conf has committed.
	if err := os.Mkdir(filepath.Join(configDir, "settings.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	writes := map[string][]byte{
		"destinations.conf": []byte("Office=D:\\Office\n"),
		"settings.json":     []byte("{\"default_source\":\"D:\\\\Inbox\"}\n"),
	}
	if err := writeConfigurationFilesTransaction(configDir, writes); err == nil {
		t.Fatal("expected transaction failure")
	}
	data, err := os.ReadFile(filepath.Join(configDir, "destinations.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Office=C:\\Office\n" {
		t.Fatalf("destinations were not rolled back: %q", data)
	}
}

func TestRestoreLatestConfigurationIsExactSnapshot(t *testing.T) {
	sourceConfig := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	writeTestConfig(t, sourceConfig, "{\"default_source\":\"D:\\\\Restored\"}\n", "")
	if _, err := createConfigurationBackup(sourceConfig, backupRoot, "1.0.29", "before-uninstall", time.Now()); err != nil {
		t.Fatal(err)
	}

	destConfig := t.TempDir()
	writeTestConfig(t, destConfig, "{\"default_source\":\"C:\\\\Old\"}\n", "Stale=C:\\Stale\n")
	if _, err := restoreLatestConfiguration(destConfig, backupRoot, "1.0.30", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destConfig, "destinations.conf")); !os.IsNotExist(err) {
		t.Fatalf("stale destinations.conf should have been removed, err=%v", err)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	original := t.TempDir()
	writeTestConfig(t, original, "{\"default_source\":\"D:\\\\Inbox\"}\n", "Office=D:\\Office\nPhotos=D:\\Photos\n")
	archive := filepath.Join(t.TempDir(), "roundtrip.dcmconf")
	if _, err := exportConfigurationArchive(original, archive, "1.0.29", time.Now()); err != nil {
		t.Fatal(err)
	}
	pkg, err := loadConfigurationImport(archive)
	if err != nil {
		t.Fatal(err)
	}
	restored := t.TempDir()
	backupRoot := filepath.Join(t.TempDir(), "Backup")
	if _, err := applyConfigurationImport(restored, backupRoot, "1.0.30", pkg, aliasReplaceAll, time.Now()); err != nil {
		t.Fatal(err)
	}
	gotSettings, err := os.ReadFile(filepath.Join(restored, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var gotCfg localSettings
	if err := json.Unmarshal(gotSettings, &gotCfg); err != nil {
		t.Fatal(err)
	}
	if gotCfg.DefaultSource != `D:\Inbox` {
		t.Fatalf("restored settings=%#v", gotCfg)
	}
	gotAliases, err := readDestinationAliasesFromPath(filepath.Join(restored, "destinations.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(gotAliases) != 2 || gotAliases[0].Name != "Office" || gotAliases[1].Name != "Photos" {
		t.Fatalf("restored aliases=%#v", gotAliases)
	}
}

func TestConfigurationPackageImportsNamingTemplates(t *testing.T) {
	cfg := NamingTemplateConfig{Active: "Voucher", Templates: []NamingTemplate{{
		Name: "Voucher", Pattern: "{office}-{name}", Variables: []NamingVariable{{Name: "office", Value: "BDO"}},
	}}}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := configurationPackageFromRaw(configurationImportArchive, "test", map[string][]byte{"naming-templates.json": data}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.NamingTemplates) == 0 {
		t.Fatal("naming templates were not retained by import package")
	}
	writes, err := configurationPackageWrites(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(writes["naming-templates.json"]) == 0 {
		t.Fatal("naming templates were not emitted by import write set")
	}
}

func TestConfigurationPackageRejectsInvalidNamingTemplates(t *testing.T) {
	bad := []byte(`{"active":"Broken","templates":[{"name":"Broken","pattern":"{missing}-{name}"}]}`)
	if _, err := configurationPackageFromRaw(configurationImportArchive, "test", map[string][]byte{"naming-templates.json": bad}, nil); err == nil {
		t.Fatal("expected invalid custom variable reference to be rejected")
	}
}
