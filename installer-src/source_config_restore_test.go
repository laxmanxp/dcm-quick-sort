package main

import "testing"

func TestInstallerConfigurationIncludesSavedSources(t *testing.T) {
	found := false
	for _, name := range configurationFileNames() {
		if name == "sources.conf" {
			found = true
		}
	}
	if !found {
		t.Fatal("sources.conf is missing from installer configuration backup/restore set")
	}
	if err := validateInstallerSources([]byte("Downloads=C:\\Downloads\nScanner=D:\\Scans\n")); err != nil {
		t.Fatal(err)
	}
}
