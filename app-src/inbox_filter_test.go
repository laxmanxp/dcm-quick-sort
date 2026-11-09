package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeInboxFixture(t *testing.T, dir, name string, size int, mod time.Time) string {
	t.Helper()
	p := filepath.Join(dir, name)
	data := make([]byte, size)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
	return p
}

func baseNames(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = filepath.Base(p)
	}
	return out
}

func TestInboxFilterCombinesNameAgeSizeTypeAndExtension(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.Local)

	writeInboxFixture(t, dir, "Invoice September.pdf", 2*1024*1024, now.Add(-2*time.Hour))
	writeInboxFixture(t, dir, "Invoice August.pdf", 2*1024*1024, now.Add(-10*24*time.Hour))
	writeInboxFixture(t, dir, "Invoice September.jpg", 2*1024*1024, now.Add(-2*time.Hour))
	writeInboxFixture(t, dir, "Receipt September.pdf", 2*1024*1024, now.Add(-2*time.Hour))
	writeInboxFixture(t, dir, "Invoice Tiny.pdf", 128*1024, now.Add(-2*time.Hour))

	got, err := listFilesInbox(dir, InboxFilter{
		Category:     CategoryDocuments,
		Extension:    ".pdf",
		NameContains: "invoice",
		Age:          InboxAgeLast7Days,
		Size:         InboxSize1To10MiB,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	names := baseNames(got)
	if len(names) != 1 || names[0] != "Invoice September.pdf" {
		t.Fatalf("combined inbox filter mismatch: %#v", names)
	}
}

func TestInboxNameFilterIsCaseInsensitiveAndMatchesFullFilename(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.Local)
	writeInboxFixture(t, dir, "BANK-Statement.PDF", 10, now)
	writeInboxFixture(t, dir, "photo.jpg", 10, now)

	got, err := listFilesInbox(dir, InboxFilter{NameContains: "statement.pdf"}, now)
	if err != nil {
		t.Fatal(err)
	}
	names := baseNames(got)
	if len(names) != 1 || names[0] != "BANK-Statement.PDF" {
		t.Fatalf("case-insensitive name filter mismatch: %#v", names)
	}
}

func TestInboxAgeFilters(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.Local)
	writeInboxFixture(t, dir, "hour.txt", 1, now.Add(-time.Hour))
	writeInboxFixture(t, dir, "three-days.txt", 1, now.Add(-72*time.Hour))
	writeInboxFixture(t, dir, "twenty-days.txt", 1, now.Add(-20*24*time.Hour))
	writeInboxFixture(t, dir, "old.txt", 1, now.Add(-45*24*time.Hour))

	cases := []struct {
		age  InboxAgeFilter
		want []string
	}{
		{InboxAgeLast24Hours, []string{"hour.txt"}},
		{InboxAgeLast7Days, []string{"hour.txt", "three-days.txt"}},
		{InboxAgeLast30Days, []string{"hour.txt", "three-days.txt", "twenty-days.txt"}},
		{InboxAgeOlder30Days, []string{"old.txt"}},
	}

	for _, tc := range cases {
		got, err := listFilesInbox(dir, InboxFilter{Age: tc.age}, now)
		if err != nil {
			t.Fatal(err)
		}
		names := baseNames(got)
		if len(names) != len(tc.want) {
			t.Fatalf("age %q: got %#v want %#v", tc.age, names, tc.want)
		}
		for i := range names {
			if names[i] != tc.want[i] {
				t.Fatalf("age %q: got %#v want %#v", tc.age, names, tc.want)
			}
		}
	}
}

func TestInboxSizeFilters(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.Local)
	writeInboxFixture(t, dir, "tiny.bin", 512*1024, now)
	writeInboxFixture(t, dir, "small.bin", 2*1024*1024, now.Add(-time.Second))
	writeInboxFixture(t, dir, "medium.bin", 20*1024*1024, now.Add(-2*time.Second))
	// Avoid allocating 100+ MiB in the test: make a sparse file.
	huge := filepath.Join(dir, "large.bin")
	f, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(120 * 1024 * 1024); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(huge, now.Add(-3*time.Second), now.Add(-3*time.Second)); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		size InboxSizeFilter
		want string
	}{
		{InboxSizeUnder1MiB, "tiny.bin"},
		{InboxSize1To10MiB, "small.bin"},
		{InboxSize10To100MiB, "medium.bin"},
		{InboxSize100MiBPlus, "large.bin"},
	}
	for _, tc := range cases {
		got, err := listFilesInbox(dir, InboxFilter{Size: tc.size}, now)
		if err != nil {
			t.Fatal(err)
		}
		names := baseNames(got)
		if len(names) != 1 || names[0] != tc.want {
			t.Fatalf("size %q: got %#v want %q", tc.size, names, tc.want)
		}
	}
}

func TestInboxFilterKeepsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.Local)
	writeInboxFixture(t, dir, "older-report.pdf", 1, now.Add(-3*time.Hour))
	writeInboxFixture(t, dir, "newer-report.pdf", 1, now.Add(-time.Hour))

	got, err := listFilesInbox(dir, InboxFilter{NameContains: "report"}, now)
	if err != nil {
		t.Fatal(err)
	}
	names := baseNames(got)
	if len(names) != 2 || names[0] != "newer-report.pdf" || names[1] != "older-report.pdf" {
		t.Fatalf("newest-first order lost: %#v", names)
	}
}

func TestInboxSortModes(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.Local)
	writeInboxFixture(t, dir, "zeta.pdf", 1, now.Add(-2*time.Hour))
	writeInboxFixture(t, dir, "alpha.jpg", 1, now.Add(-26*time.Hour))
	writeInboxFixture(t, dir, "beta.pdf", 1, now.Add(-time.Hour))
	writeInboxFixture(t, dir, "aardvark.txt", 1, now.Add(-25*time.Hour))

	cases := []struct {
		sort InboxSort
		want []string
	}{
		{InboxSortName, []string{"aardvark.txt", "alpha.jpg", "beta.pdf", "zeta.pdf"}},
		{InboxSortFileType, []string{"alpha.jpg", "beta.pdf", "zeta.pdf", "aardvark.txt"}},
		{InboxSortDateCreated, []string{"beta.pdf", "zeta.pdf", "aardvark.txt", "alpha.jpg"}},
		{InboxSortDateCreatedTypeName, []string{"beta.pdf", "zeta.pdf", "alpha.jpg", "aardvark.txt"}},
	}
	for _, tc := range cases {
		got, err := listFilesInbox(dir, InboxFilter{Sort: tc.sort}, now)
		if err != nil {
			t.Fatal(err)
		}
		names := baseNames(got)
		if len(names) != len(tc.want) {
			t.Fatalf("sort %q: got %#v want %#v", tc.sort, names, tc.want)
		}
		for i := range names {
			if names[i] != tc.want[i] {
				t.Fatalf("sort %q: got %#v want %#v", tc.sort, names, tc.want)
			}
		}
	}
}

func TestInboxSortDefaultsToDateCreated(t *testing.T) {
	if got := normalizedInboxFilter(InboxFilter{}).Sort; got != InboxSortDateCreated {
		t.Fatalf("default sort=%q want %q", got, InboxSortDateCreated)
	}
}
