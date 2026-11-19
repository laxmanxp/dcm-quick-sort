//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestD2WindowsGuardBlocksWriteRenameAndDelete(t *testing.T) {
	for _, forDelete := range []bool{false, true} {
		name := "read"
		if forDelete {
			name = "retire"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "guarded.bin")
			if err := os.WriteFile(path, []byte("ORIGINAL"), 0o600); err != nil {
				t.Fatal(err)
			}
			guard, err := openGuardedRegularFile(path, forDelete)
			if err != nil {
				t.Fatal(err)
			}
			defer guard.Close()
			if writer, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
				writer.Close()
				t.Error("guard allowed a competing writer")
			}
			if err := os.Rename(path, path+".renamed"); err == nil {
				t.Fatal("guard allowed its file to be renamed")
			}
			if err := os.Remove(path); err == nil {
				t.Fatal("guard allowed pathname deletion")
			}
			if forDelete {
				if err := deleteGuardedFile(guard); err != nil {
					t.Fatalf("exact-handle deletion failed: %v", err)
				}
				if err := guard.Close(); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("guarded file was not retired: %v", err)
				}
			} else {
				if err := guard.Close(); err != nil {
					t.Fatal(err)
				}
				assertD2FileBytes(t, path, "ORIGINAL")
			}
		})
	}
}

func TestD2WindowsRetirementRejectsReplacementIdentity(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.bin"), filepath.Join(root, "target.bin")
	if err := os.WriteFile(source, []byte("SAME"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("SAME"), 0o600); err != nil {
		t.Fatal(err)
	}
	expected, err := snapshotSource(source)
	if err != nil {
		t.Fatal(err)
	}
	previousSource := filepath.Join(root, "previous-source.bin")
	if err := os.Rename(source, previousSource); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("SAME"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := retireVerifiedSource(source, target, expected); err == nil {
		t.Error("equal bytes were accepted as proof of the original source identity")
	}
	assertD2FileBytes(t, source, "SAME")
	assertD2FileBytes(t, previousSource, "SAME")
	assertD2FileBytes(t, target, "SAME")
}

func TestD2WindowsRetirementRefusesOpenDestinationWriter(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.bin"), filepath.Join(root, "target.bin")
	for _, path := range []string{source, target} {
		if err := os.WriteFile(path, []byte("SAME"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	expected, err := snapshotSource(source)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := os.OpenFile(target, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if err := retireVerifiedSource(source, target, expected); err == nil {
		t.Error("source was retired while destination still allowed a writer")
	}
	assertD2FileBytes(t, source, "SAME")
	assertD2FileBytes(t, target, "SAME")
}

func TestD2WindowsRetirementRejectsFinalSymlink(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.bin"), filepath.Join(root, "target.bin")
	for _, path := range []string{source, target} {
		if err := os.WriteFile(path, []byte("SAME"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	expected, err := snapshotSource(source)
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "source-link.bin")
	if err := os.Symlink(source, link); err != nil {
		t.Skipf("Windows account cannot create symbolic-link fixture: %v", err)
	}
	if guard, err := openGuardedRegularFile(link, true); err == nil {
		guard.Close()
		t.Error("retirement guard followed a final-component symbolic link")
	}
	if err := retireVerifiedSource(link, target, expected); err == nil {
		t.Error("retirement followed the symbolic link and deleted its target")
	}
	assertD2FileBytes(t, source, "SAME")
	assertD2FileBytes(t, target, "SAME")
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("refused symbolic link was changed: %v", err)
	}
}

func TestD2RecoveryPartialRetirementRefusesChangedIdentityOrBytes(t *testing.T) {
	for _, change := range []string{"target-identity", "target-content", "source-content", "unverified-prefix"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			source, target := filepath.Join(root, "source.bin"), filepath.Join(root, "target.bin")
			wantSource, wantTarget := "abcdefghij", "abcde"
			if change == "unverified-prefix" {
				wantTarget = "abXde"
			}
			writeRecoveryMoveFile(t, source, wantSource)
			writeRecoveryMoveFile(t, target, wantTarget)
			expectedSource, err := snapshotSource(source)
			if err != nil {
				t.Fatal(err)
			}
			expectedTarget, err := snapshotSource(target)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "target-identity":
				if err := os.Rename(target, target+".previous"); err != nil {
					t.Fatal(err)
				}
				writeRecoveryMoveFile(t, target, wantTarget)
			case "target-content":
				wantTarget = "ABCDE"
				writeRecoveryMoveFile(t, target, wantTarget)
			case "source-content":
				wantSource = "abcdeFGHIJ"
				writeRecoveryMoveFile(t, source, wantSource)
			}
			if err := removeVerifiedPartialDestination(source, target, expectedSource, expectedTarget); err == nil {
				t.Error("partial retirement accepted changed identity, content, or an unverified prefix")
			}
			assertD2FileBytes(t, source, wantSource)
			assertD2FileBytes(t, target, wantTarget)
			if change == "target-identity" {
				assertD2FileBytes(t, target+".previous", "abcde")
			}
		})
	}
}
