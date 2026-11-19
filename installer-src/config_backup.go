package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const configurationFormatVersion = 2

var ErrNoConfiguration = errors.New("no DCM configuration files found")

type configurationSnapshotFile struct {
	Name   string
	Data   []byte
	SHA256 string
	Size   int64
}

type configurationSnapshot struct {
	Files  []configurationSnapshotFile
	Digest string
}

type configurationBackupFileInfo struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type configurationBackupInfo struct {
	Product             string                        `json:"product"`
	ConfigVersion       int                           `json:"config_version"`
	AppVersion          string                        `json:"app_version"`
	CreatedAt           string                        `json:"created_at"`
	Reason              string                        `json:"reason"`
	ConfigurationSHA256 string                        `json:"configuration_sha256"`
	Files               []configurationBackupFileInfo `json:"files"`
}

type configurationBackupResult struct {
	BackupRoot     string
	LatestDir      string
	HistoryDir     string
	Digest         string
	HistoryCreated bool
}

type configurationExportResult struct {
	Path   string
	Digest string
}

func configurationFileNames() []string {
	return []string{"destinations.conf", "learning.json", "naming-templates.json", "settings.json", "sources.conf"}
}

func collectConfigurationSnapshot(configDir string) (configurationSnapshot, error) {
	configDir = strings.TrimSpace(configDir)
	if configDir == "" {
		return configurationSnapshot{}, fmt.Errorf("configuration directory is unavailable")
	}
	names := configurationFileNames()
	sort.Strings(names)
	files := make([]configurationSnapshotFile, 0, len(names))
	combined := sha256.New()
	for _, name := range names {
		path := filepath.Join(configDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return configurationSnapshot{}, fmt.Errorf("read %s: %w", name, err)
		}
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		files = append(files, configurationSnapshotFile{Name: name, Data: data, SHA256: hash, Size: int64(len(data))})
		_, _ = io.WriteString(combined, name)
		_, _ = combined.Write([]byte{0})
		_, _ = io.WriteString(combined, hash)
		_, _ = combined.Write([]byte{0})
	}
	if len(files) == 0 {
		return configurationSnapshot{}, ErrNoConfiguration
	}
	return configurationSnapshot{Files: files, Digest: hex.EncodeToString(combined.Sum(nil))}, nil
}

func makeConfigurationBackupInfo(snapshot configurationSnapshot, appVersion, reason string, now time.Time) configurationBackupInfo {
	files := make([]configurationBackupFileInfo, 0, len(snapshot.Files))
	for _, f := range snapshot.Files {
		files = append(files, configurationBackupFileInfo{Name: f.Name, SHA256: f.SHA256, Size: f.Size})
	}
	return configurationBackupInfo{
		Product:             "DCM - Quick Sort",
		ConfigVersion:       configurationFormatVersion,
		AppVersion:          strings.TrimSpace(appVersion),
		CreatedAt:           now.Format(time.RFC3339Nano),
		Reason:              strings.TrimSpace(reason),
		ConfigurationSHA256: snapshot.Digest,
		Files:               files,
	}
}

func backupInfoJSON(info configurationBackupInfo) ([]byte, error) {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func writeConfigurationSnapshotDirectory(dir string, snapshot configurationSnapshot, info configurationBackupInfo) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, f := range snapshot.Files {
		if err := atomicWriteFile(filepath.Join(dir, f.Name), f.Data, 0o644); err != nil {
			return err
		}
	}
	manifest, err := backupInfoJSON(info)
	if err != nil {
		return err
	}
	if err := atomicWriteFile(filepath.Join(dir, "backup-info.json"), manifest, 0o644); err != nil {
		return err
	}
	return nil
}

func readConfigurationBackupInfo(path string) (configurationBackupInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return configurationBackupInfo{}, err
	}
	var info configurationBackupInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return configurationBackupInfo{}, err
	}
	return info, nil
}

func replaceSnapshotDirectory(target string, snapshot configurationSnapshot, info configurationBackupInfo) error {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(parent, ".latest.tmp-"+fmt.Sprintf("%d", time.Now().UnixNano()))
	_ = os.RemoveAll(tmp)
	if err := writeConfigurationSnapshotDirectory(tmp, snapshot, info); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	_ = os.RemoveAll(target)
	if err := os.Rename(tmp, target); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	return nil
}

func historySnapshotName(now time.Time, digest string) string {
	short := digest
	if len(short) > 12 {
		short = short[:12]
	}
	return now.Format("2006-01-02_150405.000") + "-" + short
}

func createConfigurationBackup(configDir, backupRoot, appVersion, reason string, now time.Time) (configurationBackupResult, error) {
	snapshot, err := collectConfigurationSnapshot(configDir)
	if err != nil {
		return configurationBackupResult{}, err
	}
	backupRoot = strings.TrimSpace(backupRoot)
	if backupRoot == "" {
		return configurationBackupResult{}, fmt.Errorf("backup folder is unavailable")
	}
	info := makeConfigurationBackupInfo(snapshot, appVersion, reason, now)
	latestDir := filepath.Join(backupRoot, "latest")
	historyRoot := filepath.Join(backupRoot, "history")

	previousDigest := ""
	if previous, err := readConfigurationBackupInfo(filepath.Join(latestDir, "backup-info.json")); err == nil {
		previousDigest = previous.ConfigurationSHA256
	}

	result := configurationBackupResult{BackupRoot: backupRoot, LatestDir: latestDir, Digest: snapshot.Digest}
	if previousDigest != snapshot.Digest {
		historyDir := filepath.Join(historyRoot, historySnapshotName(now, snapshot.Digest))
		if err := writeConfigurationSnapshotDirectory(historyDir, snapshot, info); err != nil {
			return configurationBackupResult{}, fmt.Errorf("write history backup: %w", err)
		}
		result.HistoryDir = historyDir
		result.HistoryCreated = true
	}

	if err := replaceSnapshotDirectory(latestDir, snapshot, info); err != nil {
		return configurationBackupResult{}, fmt.Errorf("write latest backup: %w", err)
	}
	return result, nil
}

func configurationExportFileName(now time.Time) string {
	return "DCM-QuickSort-Configuration-" + now.Format("2006-01-02_150405") + ".dcmconf"
}

func exportConfigurationArchive(configDir, destination, appVersion string, now time.Time) (configurationExportResult, error) {
	snapshot, err := collectConfigurationSnapshot(configDir)
	if err != nil {
		return configurationExportResult{}, err
	}
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return configurationExportResult{}, fmt.Errorf("export path is unavailable")
	}
	if !strings.EqualFold(filepath.Ext(destination), ".dcmconf") {
		destination += ".dcmconf"
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return configurationExportResult{}, err
	}
	tmp := destination + ".tmp"
	_ = os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return configurationExportResult{}, err
	}
	zw := zip.NewWriter(f)
	writeEntry := func(name string, data []byte) error {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetModTime(now)
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	for _, item := range snapshot.Files {
		if err := writeEntry(item.Name, item.Data); err != nil {
			_ = zw.Close()
			_ = f.Close()
			_ = os.Remove(tmp)
			return configurationExportResult{}, err
		}
	}
	manifest, err := backupInfoJSON(makeConfigurationBackupInfo(snapshot, appVersion, "manual-export", now))
	if err == nil {
		err = writeEntry("backup-info.json", manifest)
	}
	if closeErr := zw.Close(); err == nil {
		err = closeErr
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return configurationExportResult{}, err
	}
	_ = os.Remove(destination)
	if err := os.Rename(tmp, destination); err != nil {
		_ = os.Remove(tmp)
		return configurationExportResult{}, err
	}
	return configurationExportResult{Path: destination, Digest: snapshot.Digest}, nil
}

func configurationBackupRootForHome(home string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		return ""
	}
	return filepath.Join(home, "Documents", "DCM Quick Sort", "Backup")
}

func configurationBackupRoot() string {
	if override := strings.TrimSpace(os.Getenv("DCM_QUICKSORT_BACKUP_DIR")); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return configurationBackupRootForHome(home)
}

func exportConfigurationToBackupFolder(configDir, backupRoot, appVersion string, now time.Time) (configurationExportResult, error) {
	if strings.TrimSpace(backupRoot) == "" {
		return configurationExportResult{}, fmt.Errorf("backup folder is unavailable")
	}
	exportDir := filepath.Join(backupRoot, "exports")
	return exportConfigurationArchive(configDir, filepath.Join(exportDir, configurationExportFileName(now)), appVersion, now)
}
