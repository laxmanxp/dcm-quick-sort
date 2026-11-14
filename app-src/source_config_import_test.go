package main

import "testing"

func TestParseImportedSourceAliases(t *testing.T) {
	aliases, err := parseImportedSourceAliases([]byte("# saved sources\nDownloads=C:\\Users\\Me\\Downloads\nScanner=D:\\Scans\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 2 || aliases[0].Name != "Downloads" || aliases[1].Path != `D:\Scans` {
		t.Fatalf("aliases=%#v", aliases)
	}
}

func TestMergeImportedSourceAliasesUsesConflictPolicy(t *testing.T) {
	existing := []SourceAlias{{Name: "Downloads", Path: `C:\Old`}}
	imported := []SourceAlias{{Name: "downloads", Path: `D:\New`}, {Name: "Scanner", Path: `E:\Scans`}}
	merged, conflicts, err := mergeImportedSourceAliases(existing, imported, aliasMergeUseImported)
	if err != nil {
		t.Fatal(err)
	}
	if conflicts != 1 || len(merged) != 2 || merged[0].Path != `D:\New` || merged[1].Name != "Scanner" {
		t.Fatalf("merged=%#v conflicts=%d", merged, conflicts)
	}
}
