package main

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Repack a valid export after changing only template bytes. Recomputing ZIP's
// CRC while retaining the original manifest proves that the configuration
// checksum, rather than ZIP corruption or invalid JSON, rejects this input.
func d4AlterExportedTemplate(t *testing.T, archive string) {
	t.Helper()
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	changed := 0
	for _, entry := range reader.File {
		input, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, readErr := io.ReadAll(input)
		closeErr := input.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read exported entry %s: read=%v close=%v", entry.Name, readErr, closeErr)
		}
		if entry.Name == "naming-templates.json" {
			if bytes.Count(contents, []byte("BDO")) != 1 {
				t.Fatal("template fixture must contain exactly one BDO value")
			}
			contents = bytes.Replace(contents, []byte("BDO"), []byte("CEO"), 1)
			changed++
		}
		out, err := writer.Create(entry.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Fatalf("changed %d template entries, want exactly one", changed)
	}
	if err := os.WriteFile(archive, output.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Include every file and directory under the current configuration, not only
// settings.json, so a rejected import cannot silently add or remove files.
func d4ConfigurationTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			result[relative+"/"] = "directory"
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[relative] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestD4InvalidTemplatePackagesPreserveAllCurrentConfiguration(t *testing.T) {
	for _, damage := range []string{"archive-template-modified", "archive-truncated", "latest-template-modified"} {
		t.Run(damage, func(t *testing.T) {
			original, current := t.TempDir(), t.TempDir()
			files := map[string]string{
				"settings.json":         `{"default_source":"D:\\Inbox","pdf_default_view":"fit_page","vault_root":"E:\\Vault"}`,
				"destinations.conf":     "Office=E:\\Vault\\Office\n",
				"sources.conf":          "Inbox=D:\\Inbox\n",
				"learning.json":         `{"version":1,"total_observations":7,"destination_counts":{"Office":7}}`,
				"naming-templates.json": `{"active":"Voucher","templates":[{"name":"Voucher","pattern":"{office}-{name}","variables":[{"name":"office","value":"BDO"}]}]}`,
			}
			for name, data := range files {
				if err := os.WriteFile(filepath.Join(original, name), []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
				// Keep different, valid current values to expose accidental import.
				data = strings.NewReplacer("BDO", "OLD", `D:\\Inbox`, `C:\\Current`, `E:\\Vault`, `F:\\Current`).Replace(data)
				if err := os.WriteFile(filepath.Join(current, name), []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.MkdirAll(filepath.Join(current, "unrelated"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(current, "unrelated", "keep.txt"), []byte("leave this alone"), 0o600); err != nil {
				t.Fatal(err)
			}
			before := d4ConfigurationTree(t, current)
			backupRoot := filepath.Join(t.TempDir(), "Backup")
			now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
			var rejection error
			if damage == "latest-template-modified" {
				if _, err := createConfigurationBackup(original, backupRoot, "1.0.71", "test", now); err != nil {
					t.Fatal(err)
				}
				modified := strings.Replace(files["naming-templates.json"], "BDO", "CEO", 1)
				if err := os.WriteFile(filepath.Join(backupRoot, "latest", "naming-templates.json"), []byte(modified), 0o600); err != nil {
					t.Fatal(err)
				}
				_, rejection = restoreLatestConfiguration(current, backupRoot, "1.0.71", now.Add(time.Minute))
			} else {
				archive := filepath.Join(t.TempDir(), "settings.dcmconf")
				if _, err := exportConfigurationArchive(original, archive, "1.0.71", now); err != nil {
					t.Fatal(err)
				}
				if _, err := loadConfigurationImport(archive); err != nil {
					t.Fatalf("undamaged control export must load: %v", err)
				}
				if damage == "archive-template-modified" {
					d4AlterExportedTemplate(t, archive)
				} else {
					data, err := os.ReadFile(archive)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(archive, data[:len(data)/2], 0o600); err != nil {
						t.Fatal(err)
					}
				}
				// Match the app's load-before-apply order on disposable files.
				pkg, err := loadConfigurationImport(archive)
				rejection = err
				if rejection == nil {
					_, rejection = applyConfigurationImport(current, backupRoot, "1.0.71", pkg, aliasReplaceAll, now)
				}
			}
			if rejection == nil {
				t.Fatal("invalid package was accepted")
			}
			if damage != "archive-truncated" && !strings.Contains(strings.ToLower(rejection.Error()), "checksum") {
				t.Fatalf("modified valid template must fail checksum validation, got: %v", rejection)
			}
			if after := d4ConfigurationTree(t, current); !reflect.DeepEqual(before, after) {
				t.Fatalf("rejected package changed configuration paths or bytes: before=%#v after=%#v", before, after)
			}
		})
	}
}
