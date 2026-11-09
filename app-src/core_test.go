package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTargetPreservesExtension(t *testing.T) {
	got, err := targetFor(`C:\\Downloads\\Adobe Scan 01.pdf`, `D:\\Office`, "Gurundia Road Estimate")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(got) != ".pdf" {
		t.Fatalf("extension not preserved: %q", got)
	}
}

func TestInvalidWindowsNameRejected(t *testing.T) {
	for _, s := range []string{"bad:name", "CON", "name.", "a/b"} {
		if _, err := sanitizeBaseName(s); err == nil {
			t.Fatalf("expected %q to be rejected", s)
		}
	}
}

func TestUniquePathDoesNotOverwrite(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "Report.pdf")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := uniquePath(p)
	if filepath.Base(got) != "Report (2).pdf" {
		t.Fatalf("got %q", got)
	}
}

func TestMoveAndUndo(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "Adobe Scan 01.pdf")
	if err := os.WriteFile(src, []byte("document"), 0o644); err != nil {
		t.Fatal(err)
	}
	target, err := targetFor(src, dstDir, "PMAY Gurundia Verification")
	if err != nil {
		t.Fatal(err)
	}
	target = uniquePath(target)
	if err := moveFile(src, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("target missing after move")
	}
	if err := undoMove(MoveRecord{From: src, To: target}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatal("source missing after undo")
	}
}

func TestMoveCreatesTypedFinalFolder(t *testing.T) {
	srcDir := t.TempDir()
	root := t.TempDir()
	src := filepath.Join(srcDir, "scanned 10.02.2026.pdf")
	if err := os.WriteFile(src, []byte("scan"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "Gurundia", "PMAY-G")
	target, err := targetFor(src, dest, "Beneficiary Verification")
	if err != nil {
		t.Fatal(err)
	}
	if err := moveFile(src, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("typed destination folder was not created: %v", err)
	}
}

func TestCategoryAndExtensionFilters(t *testing.T) {
	d := t.TempDir()
	for _, name := range []string{"photo.JPG", "scan.pdf", "movie.mp4", "song.mp3", "archive.zip"} {
		if err := os.WriteFile(filepath.Join(d, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	images, err := listFilesFiltered(d, CategoryImages, "All extensions")
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 || filepath.Base(images[0]) != "photo.JPG" {
		t.Fatalf("image filter mismatch: %#v", images)
	}

	docs, err := listFilesFiltered(d, CategoryDocuments, ".pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || filepath.Base(docs[0]) != "scan.pdf" {
		t.Fatalf("document extension filter mismatch: %#v", docs)
	}

	all, err := listFilesFiltered(d, CategoryAll, "All extensions")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("all filter should retain unknown extensions too; got %d", len(all))
	}
}

func TestFolderSuggestionsAreTypeAhead(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Documents", "Downloads", "Desktop", "Pictures"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := folderSuggestions(filepath.Join(root, "Do"), 12)
	if len(got) != 2 {
		t.Fatalf("expected Documents and Downloads, got %#v", got)
	}
	if filepath.Base(got[0]) != "Documents" || filepath.Base(got[1]) != "Downloads" {
		t.Fatalf("unexpected suggestions: %#v", got)
	}
}

func TestPreviewFallbackForExtensionlessFile(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "mystery-download")
	if err := os.WriteFile(p, []byte("1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := previewFallbackText(p, false)
	for _, want := range []string{"No preview available", "no extension", "mystery-download", "10 B", "add its extension"} {
		if !strings.Contains(got, want) {
			t.Fatalf("fallback missing %q: %q", want, got)
		}
	}
}

func TestPreviewFallbackForUnsupportedExtension(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "drawing.xyz")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := previewFallbackText(p, false)
	for _, want := range []string{"No preview available", "XYZ", "No Windows preview handler"} {
		if !strings.Contains(got, want) {
			t.Fatalf("fallback missing %q: %q", want, got)
		}
	}
}

func TestPreviewFallbackWhenRegisteredHandlerFails(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "document.docx")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := previewFallbackText(p, true)
	for _, want := range []string{"Preview unavailable", "DOCX", "could not preview"} {
		if !strings.Contains(got, want) {
			t.Fatalf("fallback missing %q: %q", want, got)
		}
	}
}

func TestReadTextPreviewPlainText(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "notes")
	if err := os.WriteFile(p, []byte("first line\nsecond line"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readTextPreview(p, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "first line") || !strings.Contains(got, "second line") {
		t.Fatalf("text preview mismatch: %q", got)
	}
}

func TestReadTextPreviewTruncates(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "large.txt")
	if err := os.WriteFile(p, []byte(strings.Repeat("A", 200)), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readTextPreview(p, 32)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Preview limited") {
		t.Fatalf("expected truncation notice: %q", got)
	}
}

func TestReadTextPreviewUTF16LE(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "utf16")
	data := []byte{0xFF, 0xFE, 'H', 0x00, 'i', 0x00, 0x0A, 0x00}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readTextPreview(p, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Hi") {
		t.Fatalf("UTF-16 preview mismatch: %q", got)
	}
}

func TestReadTextPreviewMakesBinaryControlsSafe(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "binary")
	if err := os.WriteFile(p, []byte{'A', 0, 1, 2, 'B'}, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readTextPreview(p, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(got, '\x00') {
		t.Fatalf("preview must not contain NUL: %q", got)
	}
	if !strings.Contains(got, "Binary/non-text") {
		t.Fatalf("expected binary warning: %q", got)
	}
}

func TestTXTMarkdownAndCSVAlwaysUsePlainTextPreview(t *testing.T) {
	cases := map[string]bool{
		"notes.txt":  true,
		"NOTES.TXT":  true,
		"notes.md":   true,
		"NOTES.MD":   true,
		"data.csv":   true,
		"DATA.CSV":   true,
		"book.xlsx":  false,
		"report.pdf": false,
	}
	for name, want := range cases {
		if got := shouldUsePlainTextPreview(name); got != want {
			t.Fatalf("shouldUsePlainTextPreview(%q) = %v; want %v", name, got, want)
		}
	}
}

func TestReadAsTextOnlyForUnknownTypes(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"mystery-download", true},
		{"payload.xyzunknown", true},
		{"photo.JPG", false},
		{"scan.pdf", false},
		{"movie.mp4", false},
		{"song.mp3", false},
		{"archive.zip", false},
		{"program.exe", false},
		{"settings.json", false},
		{"database.sqlite", false},
	}
	for _, tc := range cases {
		if got := canReadAsText(tc.name); got != tc.want {
			t.Fatalf("canReadAsText(%q) = %v; want %v", tc.name, got, tc.want)
		}
	}
}

func TestFitDimensionsUsesWholePreviewPane(t *testing.T) {
	cases := []struct {
		name                   string
		sw, sh, aw, ah, dw, dh int32
	}{
		{"landscape downscale", 2000, 1000, 1000, 500, 1000, 500},
		{"portrait downscale", 1000, 2000, 500, 1000, 500, 1000},
		{"small image upscale", 200, 100, 1000, 500, 1000, 500},
		{"letterbox landscape", 1600, 900, 1000, 1000, 1000, 562},
		{"letterbox portrait", 900, 1600, 1000, 1000, 562, 1000},
	}
	for _, tc := range cases {
		gotW, gotH := fitDimensions(tc.sw, tc.sh, tc.aw, tc.ah)
		if gotW != tc.dw || gotH != tc.dh {
			t.Fatalf("%s: got %dx%d; want %dx%d", tc.name, gotW, gotH, tc.dw, tc.dh)
		}
	}
}

func TestClampPreviewZoom(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{10, 25},
		{25, 25},
		{125, 125},
		{400, 400},
		{450, 400},
	}
	for _, tc := range cases {
		if got := clampPreviewZoom(tc.in); got != tc.want {
			t.Fatalf("clampPreviewZoom(%d) = %d; want %d", tc.in, got, tc.want)
		}
	}
}

func TestZoomDimensionsUsesActualImageScale(t *testing.T) {
	cases := []struct {
		name   string
		sw, sh int32
		zoom   int
		dw, dh int32
	}{
		{"quarter", 1600, 900, 25, 400, 225},
		{"actual pixels", 1600, 900, 100, 1600, 900},
		{"one twenty five", 800, 600, 125, 1000, 750},
		{"four hundred", 320, 200, 400, 1280, 800},
	}
	for _, tc := range cases {
		gotW, gotH := zoomDimensions(tc.sw, tc.sh, tc.zoom)
		if gotW != tc.dw || gotH != tc.dh {
			t.Fatalf("%s: got %dx%d; want %dx%d", tc.name, gotW, gotH, tc.dw, tc.dh)
		}
	}
}

func TestClampPreviewPan(t *testing.T) {
	cases := []struct {
		content, viewport, offset, want int32
	}{
		{800, 1000, 200, 0},
		{1600, 1000, 0, 0},
		{1600, 1000, 500, 300},
		{1600, 1000, -500, -300},
		{1600, 1000, 125, 125},
	}
	for _, tc := range cases {
		if got := clampPreviewPan(tc.content, tc.viewport, tc.offset); got != tc.want {
			t.Fatalf("clampPreviewPan(%d,%d,%d) = %d; want %d", tc.content, tc.viewport, tc.offset, got, tc.want)
		}
	}
}

func TestMoveRollbackIfSourceCannotBeRemoved(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "locked.jpg")
	target := filepath.Join(dstDir, "renamed.jpg")
	if err := os.WriteFile(src, []byte("image-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	renameErr := errors.New("force copy fallback")
	removeErr := errors.New("source is locked")
	err := moveFileWith(src, target,
		func(string, string) error { return renameErr },
		func(string) error { return removeErr },
	)
	if err == nil || !strings.Contains(err.Error(), "source is locked") {
		t.Fatalf("expected source removal error, got %v", err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source must remain after failed move: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("partial copied target must be rolled back; stat err=%v", err)
	}
}

func TestCopyFileVerifiedLeavesSourceUntilCleanup(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "locked.jpg")
	target := filepath.Join(dstDir, "meaningful-name.jpg")
	want := []byte("image-data-that-must-survive-the-copy")
	if err := os.WriteFile(src, want, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFileVerified(src, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source must remain until UI advances and cleanup runs: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("copied bytes differ: got %q want %q", got, want)
	}
}

func TestNextEnabledIndexForwardWrapAndSkip(t *testing.T) {
	enabled := []bool{true, false, true, true}
	if got := nextEnabledIndex(0, enabled, false); got != 2 {
		t.Fatalf("forward skip: got %d want 2", got)
	}
	if got := nextEnabledIndex(3, enabled, false); got != 0 {
		t.Fatalf("forward wrap: got %d want 0", got)
	}
}

func TestNextEnabledIndexReverseWrapAndSkip(t *testing.T) {
	enabled := []bool{true, false, true, true}
	if got := nextEnabledIndex(2, enabled, true); got != 0 {
		t.Fatalf("reverse skip: got %d want 0", got)
	}
	if got := nextEnabledIndex(0, enabled, true); got != 3 {
		t.Fatalf("reverse wrap: got %d want 3", got)
	}
}

func TestNextEnabledIndexOutsideOrderAndAllDisabled(t *testing.T) {
	if got := nextEnabledIndex(-1, []bool{false, true, false}, false); got != 1 {
		t.Fatalf("outside forward: got %d want 1", got)
	}
	if got := nextEnabledIndex(-1, []bool{false, true, false}, true); got != 1 {
		t.Fatalf("outside reverse: got %d want 1", got)
	}
	if got := nextEnabledIndex(0, []bool{false, false}, false); got != -1 {
		t.Fatalf("all disabled: got %d want -1", got)
	}
}

func TestExistingDestinationFolder(t *testing.T) {
	dir := t.TempDir()
	if got, ok := existingDestinationFolder("  " + dir + "  "); !ok || filepath.Clean(got) != filepath.Clean(dir) {
		t.Fatalf("existing directory not accepted: got=%q ok=%v", got, ok)
	}
	if _, ok := existingDestinationFolder(""); ok {
		t.Fatal("blank destination must not be openable")
	}
	if _, ok := existingDestinationFolder(filepath.Join(dir, "missing")); ok {
		t.Fatal("missing destination must not be openable")
	}
	file := filepath.Join(dir, "not-a-folder.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := existingDestinationFolder(file); ok {
		t.Fatal("regular file must not be treated as a destination folder")
	}
}

func TestAltShortcutActions(t *testing.T) {
	cases := map[rune]int{
		'B': ShortcutBrowseSource,
		'L': ShortcutFocusFileList,
		'W': ShortcutFocusNewName,
		'P': ShortcutFocusDestinationPath,
		'C': ShortcutFocusSourcePath,
		'T': ShortcutReadText,
		'D': ShortcutBrowseDestination,
		'M': ShortcutManageDestinations,
		'R': ShortcutRename,
		'V': ShortcutMove,
		'X': ShortcutNext,
		'S': ShortcutSkip,
		'U': ShortcutUndo,
		'O': ShortcutOpen,
		'F': ShortcutOpenFolder,
		'N': ShortcutOpenDestination,
		'A': ShortcutApplyRenameTool,
		'o': ShortcutOpen,
	}
	for key, want := range cases {
		if got := altShortcutAction(key); got != want {
			t.Fatalf("altShortcutAction(%q) = %d; want %d", key, got, want)
		}
	}
	if got := altShortcutAction('Q'); got != ShortcutNone {
		t.Fatalf("unmapped key should return ShortcutNone; got %d", got)
	}
}

func TestFilePositionLabelForSelectableFileList(t *testing.T) {
	cases := []struct {
		index, total int
		want         string
	}{
		{0, 5, "1 of 5"},
		{3, 5, "4 of 5"},
		{5, 5, "0 files"},
		{-1, 5, "0 files"},
		{0, 0, "0 files"},
	}
	for _, tc := range cases {
		if got := filePositionLabel(tc.index, tc.total); got != tc.want {
			t.Fatalf("filePositionLabel(%d,%d)=%q want %q", tc.index, tc.total, got, tc.want)
		}
	}
}

func TestProcessedRecordLabelShowsOldAndNewNames(t *testing.T) {
	rec := MoveRecord{From: filepath.Join("old", "R5Kzw.jpg"), To: filepath.Join("new", "Bridge Inspection.jpg")}
	got := processedRecordLabel(rec)
	if got != "R5Kzw.jpg → Bridge Inspection.jpg" {
		t.Fatalf("processedRecordLabel()=%q", got)
	}
}

func TestUndoHistoryAtSelectedTransaction(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcA := filepath.Join(srcDir, "first.pdf")
	srcB := filepath.Join(srcDir, "second.jpg")
	dstA := filepath.Join(dstDir, "Invoice.pdf")
	dstB := filepath.Join(dstDir, "Site Photo.jpg")
	if err := os.WriteFile(srcA, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcB, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := moveFile(srcA, dstA); err != nil {
		t.Fatal(err)
	}
	if err := moveFile(srcB, dstB); err != nil {
		t.Fatal(err)
	}

	history := []MoveRecord{{From: srcA, To: dstA}, {From: srcB, To: dstB}}
	remaining, undone, err := undoHistoryAt(history, 0)
	if err != nil {
		t.Fatal(err)
	}
	if undone.From != srcA || undone.To != dstA {
		t.Fatalf("wrong transaction undone: %#v", undone)
	}
	if len(remaining) != 1 || remaining[0].From != srcB {
		t.Fatalf("remaining history mismatch: %#v", remaining)
	}
	if _, err := os.Stat(srcA); err != nil {
		t.Fatalf("selected original was not restored: %v", err)
	}
	if _, err := os.Stat(dstA); !os.IsNotExist(err) {
		t.Fatalf("selected processed file should be moved back; stat err=%v", err)
	}
	if _, err := os.Stat(dstB); err != nil {
		t.Fatalf("unselected transaction must remain processed: %v", err)
	}
	if _, err := os.Stat(srcB); !os.IsNotExist(err) {
		t.Fatalf("unselected original should stay absent; stat err=%v", err)
	}
}

func TestUndoHistoryAtRejectsInvalidSelection(t *testing.T) {
	history := []MoveRecord{{From: "a", To: "b"}}
	remaining, _, err := undoHistoryAt(history, 3)
	if err == nil {
		t.Fatal("expected invalid selection error")
	}
	if len(remaining) != 1 || remaining[0] != history[0] {
		t.Fatalf("history should be unchanged: %#v", remaining)
	}
}

func TestShouldAutoFitPreviewTargetsEdgePDFOnly(t *testing.T) {
	if !shouldAutoFitPreview(`C:\\Downloads\\scan.pdf`, edgePDFPreviewHandlerCLSID) {
		t.Fatal("Edge PDF preview should request auto-fit")
	}
	if !shouldAutoFitPreview(`C:\\Downloads\\SCAN.PDF`, strings.ToLower(edgePDFPreviewHandlerCLSID)) {
		t.Fatal("auto-fit detection should be case-insensitive")
	}
	if shouldAutoFitPreview(`C:\\Downloads\\photo.jpg`, edgePDFPreviewHandlerCLSID) {
		t.Fatal("non-PDF previews must not use PDF fit shortcut")
	}
	if shouldAutoFitPreview(`C:\\Downloads\\scan.pdf`, "{00000000-0000-0000-0000-000000000000}") {
		t.Fatal("unknown PDF handlers must not receive Edge-specific shortcuts")
	}
}

func TestProcessedRecordLabelIdentifiesSameFolderRename(t *testing.T) {
	dir := t.TempDir()
	got := processedRecordLabel(MoveRecord{
		From: filepath.Join(dir, "scan.pdf"),
		To:   filepath.Join(dir, "Road Estimate.pdf"),
		Kind: MoveKindMove,
	})
	if !strings.Contains(got, "[renamed]") {
		t.Fatalf("same-folder rename should be identified in Processed history: %q", got)
	}
}

func TestConsumePreviewWheelDetentsAccumulatesRawDelta(t *testing.T) {
	rem := 0
	if got := consumePreviewWheelDetents(&rem, 60); got != 0 || rem != 60 {
		t.Fatalf("first half detent got=%d rem=%d; want 0,60", got, rem)
	}
	if got := consumePreviewWheelDetents(&rem, 60); got != 1 || rem != 0 {
		t.Fatalf("second half detent got=%d rem=%d; want 1,0", got, rem)
	}
	if got := consumePreviewWheelDetents(&rem, -360); got != -3 || rem != 0 {
		t.Fatalf("three reverse detents got=%d rem=%d; want -3,0", got, rem)
	}
}

func TestNextPreviewZoomUsesSmallProportionalSteps(t *testing.T) {
	cases := []struct {
		current, detents, want int
	}{
		{43, 1, 47},
		{47, 1, 52},
		{100, 1, 110},
		{110, -1, 100},
		{43, 2, 52},
		{25, -1, 25},
		{400, 1, 400},
	}
	for _, tc := range cases {
		if got := nextPreviewZoom(tc.current, tc.detents); got != tc.want {
			t.Fatalf("nextPreviewZoom(%d,%d)=%d; want %d", tc.current, tc.detents, got, tc.want)
		}
	}
}

func TestPreviewPanPixelsUsesViewportFractionNotOSLines(t *testing.T) {
	if got := previewPanPixels(120, 800); got != 80 {
		t.Fatalf("one wheel detent on 800px pane = %d; want 80", got)
	}
	if got := previewPanPixels(-120, 800); got != -80 {
		t.Fatalf("reverse detent = %d; want -80", got)
	}
	if got := previewPanPixels(60, 800); got != 40 {
		t.Fatalf("high-resolution half detent = %d; want 40", got)
	}
}

func TestPreviewTopPanOffsetAnchorsTallFitWidthContentToTop(t *testing.T) {
	cases := []struct {
		content, viewport, want int32
	}{
		{800, 1000, 0},
		{1000, 1000, 0},
		{1600, 1000, 300},
		{1601, 1000, 300},
	}
	for _, tc := range cases {
		if got := previewTopPanOffset(tc.content, tc.viewport); got != tc.want {
			t.Fatalf("previewTopPanOffset(%d,%d) = %d; want %d", tc.content, tc.viewport, got, tc.want)
		}
	}
}
