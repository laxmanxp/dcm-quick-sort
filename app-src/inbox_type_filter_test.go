package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandedInboxTypeFamilies(t *testing.T) {
	cases := []struct {
		name string
		want FileCategory
	}{
		{"bundle.zip", CategoryArchives},
		{"setup.msi", CategoryPrograms},
		{"script.ps1", CategoryCodeText},
		{"source.go", CategoryCodeText},
		{"drawing.dwg", CategoryDesignCAD},
		{"font.otf", CategoryDesignCAD},
		{"book.epub", CategoryDocuments},
	}
	for _, tc := range cases {
		if got := categoryForPath(tc.name); got != tc.want {
			t.Fatalf("%s: got category %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestOtherUnknownInboxCategory(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"mystery.foo", "plain", "known.zip", "known.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := listFilesFiltered(dir, CategoryOther, "All extensions")
	if err != nil {
		t.Fatal(err)
	}
	names := baseNames(got)
	if len(names) != 2 {
		t.Fatalf("other/unknown expected 2 files, got %#v", names)
	}
	seen := map[string]bool{}
	for _, n := range names {
		seen[n] = true
	}
	if !seen["mystery.foo"] || !seen["plain"] {
		t.Fatalf("other/unknown mismatch: %#v", names)
	}
}

func TestExpandedExtensionsAppearInAllAndFamilyFilters(t *testing.T) {
	archives := extensionsForCategory(CategoryArchives)
	programs := extensionsForCategory(CategoryPrograms)
	code := extensionsForCategory(CategoryCodeText)
	design := extensionsForCategory(CategoryDesignCAD)
	all := extensionsForCategory(CategoryAll)

	contains := func(values []string, want string) bool {
		for _, v := range values {
			if v == want {
				return true
			}
		}
		return false
	}
	for _, pair := range []struct {
		values []string
		want   string
	}{
		{archives, ".zip"}, {programs, ".exe"}, {code, ".go"}, {design, ".dwg"},
		{all, ".zip"}, {all, ".exe"}, {all, ".go"}, {all, ".dwg"},
	} {
		if !contains(pair.values, pair.want) {
			t.Fatalf("missing extension %s in %#v", pair.want, pair.values)
		}
	}
}
