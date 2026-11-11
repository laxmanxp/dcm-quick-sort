package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

type MoveKind string

const (
	MoveKindMove             MoveKind = "move"
	MoveKindReplace          MoveKind = "replace"
	MoveKindDuplicateRemoved MoveKind = "duplicate-removed"
)

type MoveRecord struct {
	From           string   `json:"from"`
	To             string   `json:"to"`
	Kind           MoveKind `json:"kind"`
	ReplacedBackup string   `json:"replaced_backup,omitempty"`
	JournalID      string   `json:"journal_id,omitempty"`
	BatchID        string   `json:"batch_id,omitempty"`
	OriginalSHA256 string   `json:"original_sha256,omitempty"`
	OriginalSize   int64    `json:"original_size,omitempty"`
}

type DestinationConflictKind int

const (
	ConflictNone DestinationConflictKind = iota
	ConflictExactDuplicate
	ConflictDifferent
)

type DestinationConflictInfo struct {
	Kind         DestinationConflictKind
	Source       string
	Target       string
	SourceSize   int64
	TargetSize   int64
	SourceSHA256 string
	TargetSHA256 string
}

type FileCategory string

const (
	CategoryAll       FileCategory = "All files"
	CategoryImages    FileCategory = "Images"
	CategoryDocuments FileCategory = "Documents"
	CategoryVideo     FileCategory = "Video"
	CategoryAudio     FileCategory = "Audio"
	CategoryArchives  FileCategory = "Archives"
	CategoryPrograms  FileCategory = "Programs"
	CategoryCodeText  FileCategory = "Code & text"
	CategoryDesignCAD FileCategory = "Design & CAD"
	CategoryOther     FileCategory = "Other / unknown"
)

var categoryOrder = []FileCategory{
	CategoryAll, CategoryImages, CategoryDocuments, CategoryVideo, CategoryAudio,
	CategoryArchives, CategoryPrograms, CategoryCodeText, CategoryDesignCAD, CategoryOther,
}

var categorizedFileTypes = []FileCategory{
	CategoryImages, CategoryDocuments, CategoryVideo, CategoryAudio,
	CategoryArchives, CategoryPrograms, CategoryCodeText, CategoryDesignCAD,
}

var categoryExtensions = map[FileCategory][]string{
	CategoryImages:    {".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".tif", ".tiff", ".heic", ".svg"},
	CategoryDocuments: {".pdf", ".doc", ".docx", ".xls", ".xlsx", ".xlsm", ".ppt", ".pptx", ".txt", ".rtf", ".csv", ".md", ".odt", ".ods", ".odp", ".epub"},
	CategoryVideo:     {".mp4", ".mkv", ".mov", ".avi", ".wmv", ".webm", ".m4v", ".mpeg", ".mpg", ".3gp"},
	CategoryAudio:     {".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg", ".wma", ".opus"},
	CategoryArchives:  {".zip", ".7z", ".rar", ".tar", ".gz", ".bz2", ".xz", ".cab", ".iso"},
	CategoryPrograms:  {".exe", ".dll", ".msi", ".msix", ".appx", ".sys", ".com"},
	CategoryCodeText:  {".bat", ".cmd", ".ps1", ".vbs", ".sh", ".log", ".ini", ".cfg", ".conf", ".json", ".xml", ".yaml", ".yml", ".html", ".htm", ".css", ".js", ".ts", ".go", ".rs", ".py", ".php", ".java", ".c", ".h", ".cpp", ".hpp", ".sql", ".db", ".sqlite", ".sqlite3"},
	CategoryDesignCAD: {".psd", ".ai", ".cdr", ".dwg", ".dxf", ".ttf", ".otf", ".woff", ".woff2"},
}

// knownExtraExtensions covers common file types outside Quick Sort's primary
// filter categories. "Read as text" is meant as an escape hatch for files whose
// type is genuinely unknown, not as an alternate renderer for known formats.
var knownExtraExtensions = map[string]bool{
	// Archives / disk images
	".zip": true, ".7z": true, ".rar": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true, ".cab": true, ".iso": true,
	// Executables / installers / scripts
	".exe": true, ".dll": true, ".msi": true, ".msix": true, ".appx": true, ".sys": true, ".com": true,
	".bat": true, ".cmd": true, ".ps1": true, ".vbs": true, ".sh": true,
	// Structured text / source code
	".log": true, ".ini": true, ".cfg": true, ".conf": true, ".json": true, ".xml": true, ".yaml": true, ".yml": true,
	".html": true, ".htm": true, ".css": true, ".js": true, ".ts": true, ".go": true, ".rs": true, ".py": true,
	".php": true, ".java": true, ".c": true, ".h": true, ".cpp": true, ".hpp": true, ".sql": true,
	// Databases / fonts / design / CAD / ebooks
	".db": true, ".sqlite": true, ".sqlite3": true, ".ttf": true, ".otf": true, ".woff": true, ".woff2": true,
	".psd": true, ".ai": true, ".cdr": true, ".dwg": true, ".dxf": true, ".epub": true,
}

var reservedWindowsNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

func sanitizeBaseName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("name cannot be empty")
	}
	if strings.HasSuffix(s, ".") || strings.HasSuffix(s, " ") {
		return "", errors.New("name cannot end with a dot or space")
	}
	for _, r := range s {
		if r < 32 || strings.ContainsRune(`<>:"/\\|?*`, r) || unicode.IsControl(r) {
			return "", fmt.Errorf("name contains an invalid character: %q", r)
		}
	}
	upper := strings.ToUpper(s)
	if dot := strings.IndexByte(upper, '.'); dot >= 0 {
		upper = upper[:dot]
	}
	if reservedWindowsNames[upper] {
		return "", fmt.Errorf("%q is reserved by Windows", s)
	}
	return s, nil
}

func targetFor(source, destDir, requestedBase string) (string, error) {
	if source == "" || destDir == "" {
		return "", errors.New("source and destination are required")
	}
	base, err := sanitizeBaseName(requestedBase)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(source)
	return filepath.Join(destDir, base+ext), nil
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(filepath.Base(path), ext)
	for n := 2; ; n++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, n, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func inspectDestinationConflict(source, target string) (DestinationConflictInfo, error) {
	info := DestinationConflictInfo{Kind: ConflictNone, Source: source, Target: target}
	srcInfo, err := os.Stat(source)
	if err != nil {
		return info, err
	}
	if !srcInfo.Mode().IsRegular() {
		return info, errors.New("source is not a regular file")
	}
	info.SourceSize = srcInfo.Size()
	tgtInfo, err := os.Stat(target)
	if os.IsNotExist(err) {
		return info, nil
	}
	if err != nil {
		return info, err
	}
	if !tgtInfo.Mode().IsRegular() {
		return info, errors.New("destination path exists but is not a regular file")
	}
	if os.SameFile(srcInfo, tgtInfo) {
		return info, errors.New("source and destination refer to the same file; no conflict action is safe")
	}
	info.TargetSize = tgtInfo.Size()
	if info.SourceSize != info.TargetSize {
		info.Kind = ConflictDifferent
		return info, nil
	}
	info.SourceSHA256, err = fileSHA256(source)
	if err != nil {
		return info, err
	}
	info.TargetSHA256, err = fileSHA256(target)
	if err != nil {
		return info, err
	}
	if info.SourceSHA256 == info.TargetSHA256 {
		info.Kind = ConflictExactDuplicate
	} else {
		info.Kind = ConflictDifferent
	}
	return info, nil
}

func keepBothTarget(target string) string { return uniquePath(target) }

func copyFileVerifiedHash(source, target string) error {
	want, err := fileSHA256(source)
	if err != nil {
		return err
	}
	if err := copyFileVerified(source, target); err != nil {
		return err
	}
	got, err := fileSHA256(target)
	if err != nil {
		_ = os.Remove(target)
		return err
	}
	if got != want {
		_ = os.Remove(target)
		return errors.New("SHA-256 verification failed after copy")
	}
	return nil
}

func replacementBackupPath(recoveryDir, target string) (string, error) {
	if err := os.MkdirAll(recoveryDir, 0o755); err != nil {
		return "", err
	}
	base := filepath.Base(target)
	for n := 1; ; n++ {
		name := fmt.Sprintf("%s.replaced.%d", base, n)
		p := filepath.Join(recoveryDir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p, nil
		}
	}
}

func replaceMoveWithRecovery(source, target, recoveryDir string) (MoveRecord, error) {
	return replaceMoveWithRecoveryUsing(source, target, recoveryDir, moveFile)
}

func replaceMoveWithRecoveryUsing(source, target, recoveryDir string, moveFn func(string, string) error) (MoveRecord, error) {
	rec := MoveRecord{From: source, To: target, Kind: MoveKindReplace}
	info, err := inspectDestinationConflict(source, target)
	if err != nil {
		return rec, err
	}
	if info.Kind == ConflictNone {
		return rec, errors.New("replace requested but destination does not exist")
	}
	backup, err := replacementBackupPath(recoveryDir, target)
	if err != nil {
		return rec, err
	}
	if err := copyFileVerifiedHash(target, backup); err != nil {
		return rec, fmt.Errorf("could not back up existing destination: %w", err)
	}
	rec.ReplacedBackup = backup
	if err := os.Remove(target); err != nil {
		_ = os.Remove(backup)
		return rec, fmt.Errorf("could not prepare destination for replacement: %w", err)
	}
	if err := moveFn(source, target); err != nil {
		// Roll back the existing destination before returning. The recovery copy
		// is retained if restoration itself fails so no bytes are silently lost.
		if restoreErr := copyFileVerifiedHash(backup, target); restoreErr == nil {
			_ = os.Remove(backup)
			rec.ReplacedBackup = ""
		} else {
			return rec, fmt.Errorf("replacement failed: %v; previous destination is preserved at %s but could not be restored automatically: %v", err, backup, restoreErr)
		}
		return rec, err
	}
	return rec, nil
}

func removeExactDuplicateSource(source, target string) (MoveRecord, error) {
	rec := MoveRecord{From: source, To: target, Kind: MoveKindDuplicateRemoved}
	info, err := inspectDestinationConflict(source, target)
	if err != nil {
		return rec, err
	}
	if info.Kind != ConflictExactDuplicate {
		return rec, errors.New("source is not an exact duplicate of the destination")
	}
	expected, err := snapshotSource(source)
	if err != nil {
		return rec, err
	}
	rec.OriginalSHA256, rec.OriginalSize = expected.SHA256, expected.Size
	if err := retireVerifiedSource(source, target, expected); err != nil {
		return rec, err
	}
	return rec, nil
}

func moveFile(source, target string) error {
	return moveFileWith(source, target, renameFileNoReplace, retireVerifiedSource)
}

// copyFileVerified stages a safe copy while leaving the source untouched.
// The destination is created exclusively, flushed to disk, and checked against
// the source byte count. This supports the Quick Sort workflow where the UI can
// advance away from the source before Windows is asked to delete it.
type CopyProgressFunc func(copied, total int64)

func copyFileVerified(source, target string) error {
	return copyFileVerifiedWithProgress(source, target, nil)
}

func copyFileVerifiedWithProgress(source, target string, progress CopyProgressFunc) error {
	if source == target {
		return errors.New("source and target are the same file")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(target)
		}
	}()
	total := info.Size()
	if progress != nil {
		progress(0, total)
	}
	buf := make([]byte, 1024*1024)
	var written int64
	var nextReport int64 = 4 * 1024 * 1024
	for {
		n, readErr := in.Read(buf)
		if n > 0 {
			wn, writeErr := out.Write(buf[:n])
			written += int64(wn)
			if writeErr != nil {
				return writeErr
			}
			if wn != n {
				return io.ErrShortWrite
			}
			if progress != nil && (written >= nextReport || written == total) {
				progress(written, total)
				for nextReport <= written {
					nextReport += 4 * 1024 * 1024
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if written != total {
		return fmt.Errorf("copy verification failed: wrote %d of %d bytes", written, total)
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	st, err := os.Stat(target)
	if err != nil {
		return err
	}
	if st.Size() != total {
		return fmt.Errorf("copy verification failed: destination size %d, source size %d", st.Size(), total)
	}
	if progress != nil && written == 0 {
		progress(0, total)
	}
	ok = true
	return nil
}

func moveFileWith(source, target string, renameFn func(string, string) error, retireFn verifiedRetireFunc) error {
	if source == target {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := renameFn(source, target); err == nil {
		return nil
	}

	expected, err := snapshotSource(source)
	if err != nil {
		return err
	}
	if err := copyFileVerified(source, target); err != nil {
		return err
	}
	// A completed copy is retained if retirement refuses. Removing target by
	// pathname here could discard a destination that another process changed.
	return retireFn(source, target, expected)
}

func validateUndoMove(rec MoveRecord) error {
	switch rec.Kind {
	case MoveKindDuplicateRemoved:
		if _, err := os.Lstat(rec.From); err == nil {
			return errors.New("cannot undo: original path is already occupied")
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("cannot inspect original path for duplicate undo: %w", err)
		}
		if err := verifyDuplicateOriginal(rec.To, rec); err != nil {
			return fmt.Errorf("cannot undo duplicate removal: %w", err)
		}
		return nil
	case MoveKindReplace:
		if rec.ReplacedBackup == "" {
			return errors.New("cannot undo replacement: recovery copy is missing")
		}
		if _, err := os.Stat(rec.From); err == nil {
			return errors.New("cannot undo: original path is already occupied")
		}
		if _, err := os.Stat(rec.To); err != nil {
			return errors.New("cannot undo replacement: current destination no longer exists")
		}
		if _, err := os.Stat(rec.ReplacedBackup); err != nil {
			return errors.New("cannot undo replacement: previous destination recovery copy is missing")
		}
		return nil
	default:
		if _, err := os.Stat(rec.From); err == nil {
			return errors.New("cannot undo: original path is already occupied")
		}
		if _, err := os.Stat(rec.To); err != nil {
			return errors.New("cannot undo: moved file no longer exists")
		}
		return nil
	}
}

func undoMove(rec MoveRecord) error {
	if err := validateUndoMove(rec); err != nil {
		return err
	}
	switch rec.Kind {
	case MoveKindDuplicateRemoved:
		return restoreDuplicateOriginal(rec)
	case MoveKindReplace:
		if err := moveFile(rec.To, rec.From); err != nil {
			return err
		}
		if err := copyFileVerifiedHash(rec.ReplacedBackup, rec.To); err != nil {
			_ = moveFile(rec.From, rec.To)
			return fmt.Errorf("could not restore previous destination: %w", err)
		}
		_ = os.Remove(rec.ReplacedBackup)
		return nil
	default:
		return moveFile(rec.To, rec.From)
	}
}

func sameMovePath(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func isRenameOnlyRecord(rec MoveRecord) bool {
	if rec.Kind != "" && rec.Kind != MoveKindMove {
		return false
	}
	if strings.TrimSpace(rec.From) == "" || strings.TrimSpace(rec.To) == "" {
		return false
	}
	return sameMovePath(filepath.Dir(rec.From), filepath.Dir(rec.To)) && !strings.EqualFold(filepath.Base(rec.From), filepath.Base(rec.To))
}

func processedRecordLabel(rec MoveRecord) string {
	switch rec.Kind {
	case MoveKindReplace:
		return fmt.Sprintf("%s → %s  [replaced existing]", filepath.Base(rec.From), filepath.Base(rec.To))
	case MoveKindDuplicateRemoved:
		return fmt.Sprintf("%s  [exact duplicate removed; matches %s]", filepath.Base(rec.From), filepath.Base(rec.To))
	default:
		if filepath.Clean(filepath.Dir(rec.From)) == filepath.Clean(filepath.Dir(rec.To)) && filepath.Base(rec.From) != filepath.Base(rec.To) {
			return fmt.Sprintf("%s → %s  [renamed]", filepath.Base(rec.From), filepath.Base(rec.To))
		}
		return fmt.Sprintf("%s → %s", filepath.Base(rec.From), filepath.Base(rec.To))
	}
}

func undoHistoryAt(history []MoveRecord, index int) ([]MoveRecord, MoveRecord, error) {
	if index < 0 || index >= len(history) {
		return history, MoveRecord{}, errors.New("select a processed transaction to undo")
	}
	rec := history[index]
	if err := undoMove(rec); err != nil {
		return history, rec, err
	}
	remaining := make([]MoveRecord, 0, len(history)-1)
	remaining = append(remaining, history[:index]...)
	remaining = append(remaining, history[index+1:]...)
	return remaining, rec, nil
}

func extensionsForCategory(category FileCategory) []string {
	if category == CategoryAll {
		seen := map[string]bool{}
		var out []string
		for _, c := range categorizedFileTypes {
			for _, ext := range categoryExtensions[c] {
				if !seen[ext] {
					seen[ext] = true
					out = append(out, ext)
				}
			}
		}
		return out
	}
	return append([]string(nil), categoryExtensions[category]...)
}

func categoryForPath(path string) FileCategory {
	ext := strings.ToLower(filepath.Ext(path))
	for _, category := range categorizedFileTypes {
		for _, candidate := range categoryExtensions[category] {
			if ext == candidate {
				return category
			}
		}
	}
	return ""
}

func isKnownFileType(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return false
	}
	if categoryForPath(path) != "" {
		return true
	}
	return knownExtraExtensions[ext]
}

func canReadAsText(path string) bool {
	return path != "" && !isKnownFileType(path)
}

func shouldUsePlainTextPreview(path string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".txt", ".md", ".csv":
		return true
	default:
		return false
	}
}

func fitDimensions(sourceW, sourceH, availW, availH int32) (int32, int32) {
	if sourceW <= 0 || sourceH <= 0 || availW <= 0 || availH <= 0 {
		return 0, 0
	}
	wScale := float64(availW) / float64(sourceW)
	hScale := float64(availH) / float64(sourceH)
	scale := wScale
	if hScale < scale {
		scale = hScale
	}
	dw := int32(float64(sourceW) * scale)
	dh := int32(float64(sourceH) * scale)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	return dw, dh
}

func fitWidthDimensions(sourceW, sourceH, availW int32) (int32, int32) {
	if sourceW <= 0 || sourceH <= 0 || availW <= 0 {
		return 0, 0
	}
	dw := availW
	dh := int32((int64(sourceH)*int64(availW) + int64(sourceW)/2) / int64(sourceW))
	if dh < 1 {
		dh = 1
	}
	return dw, dh
}

const (
	previewZoomMin = 25
	previewZoomMax = 400
	wheelDelta     = 120
)

func clampPreviewZoom(percent int) int {
	if percent < previewZoomMin {
		return previewZoomMin
	}
	if percent > previewZoomMax {
		return previewZoomMax
	}
	return percent
}

// consumePreviewWheelDetents turns raw WM_MOUSEWHEEL deltas into logical
// detents while retaining high-resolution/trackpad remainder. It deliberately
// does not consult Windows' "scroll N lines" preference: preview zoom is a
// graphical operation, not a text-line scroll operation.
func consumePreviewWheelDetents(remainder *int, delta int) int {
	if remainder == nil || delta == 0 {
		return 0
	}
	total := *remainder + delta
	detents := total / wheelDelta
	*remainder = total % wheelDelta
	return detents
}

func nextPreviewZoomBounded(current, detents, minZoom, maxZoom int) int {
	if minZoom <= 0 {
		minZoom = 1
	}
	if maxZoom < minZoom {
		maxZoom = minZoom
	}
	if current < minZoom {
		current = minZoom
	}
	if current > maxZoom {
		current = maxZoom
	}
	stepUp := func(v int) int {
		n := (v*110 + 50) / 100
		if n <= v {
			n = v + 1
		}
		if n > maxZoom {
			n = maxZoom
		}
		return n
	}
	stepDown := func(v int) int {
		// Inverse of the 10% growth step, rounded to nearest integer. This
		// makes 100 -> 110 -> 100 feel symmetric instead of drifting.
		n := (v*100 + 55) / 110
		if n >= v {
			n = v - 1
		}
		if n < minZoom {
			n = minZoom
		}
		return n
	}
	for detents > 0 {
		current = stepUp(current)
		detents--
	}
	for detents < 0 {
		current = stepDown(current)
		detents++
	}
	return current
}

func nextPreviewZoom(current, detents int) int {
	return nextPreviewZoomBounded(current, detents, previewZoomMin, previewZoomMax)
}

// previewPanPixels maps wheel motion to a fraction of the visible preview
// height/width rather than to Windows' configured text-line scroll count.
// One conventional detent pans 10% of the viewport; partial deltas remain
// proportional for high-resolution mice and touchpads.
func previewPanPixels(delta int, viewport int32) int32 {
	if delta == 0 || viewport <= 0 {
		return 0
	}
	return int32((int64(delta)*int64(viewport) + signedRoundingBias(int64(delta), int64(wheelDelta*10))) / int64(wheelDelta*10))
}

func signedRoundingBias(value, divisor int64) int64 {
	if value > 0 {
		return divisor / 2
	}
	if value < 0 {
		return -divisor / 2
	}
	return 0
}

func zoomDimensions(sourceW, sourceH int32, zoomPercent int) (int32, int32) {
	if sourceW <= 0 || sourceH <= 0 {
		return 0, 0
	}
	zoomPercent = clampPreviewZoom(zoomPercent)
	dw := int32((int64(sourceW)*int64(zoomPercent) + 50) / 100)
	dh := int32((int64(sourceH)*int64(zoomPercent) + 50) / 100)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	return dw, dh
}

func clampPreviewPan(content, viewport, offset int32) int32 {
	if content <= viewport || viewport <= 0 {
		return 0
	}
	limit := (content - viewport) / 2
	if offset > limit {
		return limit
	}
	if offset < -limit {
		return -limit
	}
	return offset
}

// previewTopPanOffset returns the centered-coordinate pan offset that places
// the top edge of content at the top edge of the viewport. Fit Width uses
// the same symmetric pan model as manual zoom, so the top position is the
// maximum positive vertical pan rather than zero.
func previewTopPanOffset(content, viewport int32) int32 {
	if content <= viewport || viewport <= 0 {
		return 0
	}
	return (content - viewport) / 2
}

func matchesFilter(path string, category FileCategory, extension string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if extension != "" && extension != "All extensions" && ext != strings.ToLower(extension) {
		return false
	}
	if category == "" || category == CategoryAll {
		return true
	}
	if category == CategoryOther {
		return categoryForPath(path) == ""
	}
	return categoryForPath(path) == category
}

func filePositionLabel(index, total int) string {
	if total <= 0 || index < 0 || index >= total {
		return "0 files"
	}
	return fmt.Sprintf("%d of %d", index+1, total)
}

func listFilesFiltered(dir string, category FileCategory, extension string) ([]string, error) {
	return listFilesInbox(dir, InboxFilter{Category: category, Extension: extension}, time.Now())
}

func listFiles(dir string) ([]string, error) {
	return listFilesFiltered(dir, CategoryAll, "All extensions")
}

func humanFileSize(bytes int64) string {
	const unit = int64(1024)
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := unit, 0
	for n := bytes / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func previewFallbackText(path string, handlerRegistered bool) string {
	base := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(base))

	var heading, reason, hint string
	switch {
	case ext == "":
		heading = "No preview available"
		reason = "Unknown file type — this file has no extension."
		hint = "If you know the file type, add its extension in the Name box (for example .pdf or .jpg)."
	case handlerRegistered:
		heading = "Preview unavailable"
		reason = fmt.Sprintf("Windows has a preview handler for %s files, but it could not preview this file.", strings.ToUpper(strings.TrimPrefix(ext, ".")))
		hint = "You can still rename, move, open, or open its folder."
	default:
		heading = "No preview available"
		reason = fmt.Sprintf("No Windows preview handler is installed for %s files.", strings.ToUpper(strings.TrimPrefix(ext, ".")))
		hint = "You can still rename, move, open, or open its folder."
	}

	metadata := []string{"Name: " + base}
	if info, err := os.Stat(path); err == nil {
		metadata = append(metadata, "Size: "+humanFileSize(info.Size()))
		metadata = append(metadata, "Modified: "+info.ModTime().Format("02 Jan 2006 15:04"))
	}

	return heading + "\n\n" + reason + "\n\n" + strings.Join(metadata, "\n") + "\n\n" + hint
}

const defaultTextPreviewLimit int64 = 256 * 1024

func readTextPreview(path string, limit int64) (string, error) {
	if limit <= 0 {
		limit = defaultTextPreviewLimit
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return "", err
	}
	truncated := int64(len(data)) > limit
	if truncated {
		data = data[:limit]
	}

	text, err := decodePreviewTextBytes(data, truncated)
	if err != nil {
		return "", err
	}
	text, suspicious := sanitizePreviewText(text)
	if suspicious {
		text = "[Binary/non-text bytes detected. Showing a safe text interpretation.]\n\n" + text
	}
	if truncated {
		text += fmt.Sprintf("\n\n[Preview limited to the first %s.]", humanFileSize(limit))
	}
	return normalizePreviewNewlines(text), nil
}

// Decode only the bounded preview prefix. A character cut by our byte limit
// is omitted; a malformed sequence in the available prefix is reported, not
// silently replaced. BOM-less legacy encodings and UTF-32 are not guessed.
func decodePreviewTextBytes(data []byte, truncated bool) (string, error) {
	if len(data) >= 4 && ((data[0] == 0xFF && data[1] == 0xFE && data[2] == 0 && data[3] == 0) ||
		(data[0] == 0 && data[1] == 0 && data[2] == 0xFE && data[3] == 0xFF)) {
		return "", fmt.Errorf("unsupported UTF-32 encoding; use UTF-8 or BOM-marked UTF-16")
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return decodePreviewUTF16(data[2:], true, truncated)
	}
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodePreviewUTF16(data[2:], false, truncated)
	}
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	for i := 0; i < len(data); {
		value, size := utf8.DecodeRune(data[i:])
		if value == utf8.RuneError && size == 1 {
			if truncated && !utf8.FullRune(data[i:]) {
				return string(data[:i]), nil
			}
			return "", fmt.Errorf("unsupported or invalid text encoding; expected UTF-8 or BOM-marked UTF-16")
		}
		i += size
	}
	return string(data), nil
}

func decodePreviewUTF16(data []byte, littleEndian, truncated bool) (string, error) {
	if len(data)%2 != 0 && !truncated {
		return "", fmt.Errorf("invalid UTF-16 encoding: incomplete code unit")
	}
	units := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		var u uint16
		if littleEndian {
			u = uint16(data[i]) | uint16(data[i+1])<<8
		} else {
			u = uint16(data[i])<<8 | uint16(data[i+1])
		}
		units = append(units, u)
	}
	for i := 0; i < len(units); i++ {
		u := units[i]
		if u >= 0xD800 && u <= 0xDBFF {
			if i+1 == len(units) && truncated {
				units = units[:i]
				break
			}
			if i+1 == len(units) || units[i+1] < 0xDC00 || units[i+1] > 0xDFFF {
				return "", fmt.Errorf("invalid UTF-16 encoding: unpaired surrogate")
			}
			i++
		} else if u >= 0xDC00 && u <= 0xDFFF {
			return "", fmt.Errorf("invalid UTF-16 encoding: unpaired surrogate")
		}
	}
	return string(utf16.Decode(units)), nil
}

func sanitizePreviewText(s string) (string, bool) {
	var b strings.Builder
	bad := 0
	total := 0
	for _, r := range s {
		total++
		switch r {
		case '\r', '\n', '\t':
			b.WriteRune(r)
		default:
			if r == 0 || (r < 32 && unicode.IsControl(r)) {
				b.WriteRune('·')
				bad++
			} else {
				b.WriteRune(r)
			}
		}
	}
	suspicious := total > 0 && bad*20 > total // more than 5% control bytes
	return b.String(), suspicious
}

func normalizePreviewNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func folderSuggestions(input string, limit int) []string {
	input = strings.TrimSpace(input)
	if input == "" || limit <= 0 {
		return nil
	}
	trailingSeparator := strings.HasSuffix(input, string(os.PathSeparator))
	if os.PathSeparator == '\\' {
		trailingSeparator = trailingSeparator || strings.HasSuffix(input, "/")
	}
	dir := input
	prefix := ""
	if !trailingSeparator {
		dir = filepath.Dir(input)
		prefix = filepath.Base(input)
		if dir == "." && filepath.VolumeName(input) == "" {
			return nil
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	prefixLower := strings.ToLower(prefix)
	out := make([]string, 0, limit)
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(strings.ToLower(e.Name()), prefixLower) {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
		if len(out) == limit {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out
}

const edgePDFPreviewHandlerCLSID = "{3A84F9C2-6164-485C-A7D9-4B27F8AC009E}"

// shouldAutoFitPreview identifies the Microsoft Edge PDF preview handler,
// which supports Ctrl+\ as a fit-to-page / fit-to-width toggle. Other
// preview handlers do not expose a standard zoom/fitting API through
// IPreviewHandler. Quick Sort can send conventional zoom shortcuts on a
// best-effort basis, while only the Edge PDF handler gets the Fit command.
func shouldAutoFitPreview(path, clsid string) bool {
	return strings.EqualFold(filepath.Ext(path), ".pdf") &&
		strings.EqualFold(strings.TrimSpace(clsid), edgePDFPreviewHandlerCLSID)
}

func existingDestinationFolder(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return path, true
}

// Alt-shortcut actions are kept in the cross-platform core so the mapping can
// be regression-tested even when the Windows UI cannot be executed here.
const (
	ShortcutNone = iota
	ShortcutBrowseSource
	ShortcutReload
	ShortcutReadText
	ShortcutBrowseDestination
	ShortcutManageDestinations
	ShortcutRename
	ShortcutMove
	ShortcutNext
	ShortcutSkip
	ShortcutUndo
	ShortcutOpen
	ShortcutOpenFolder
	ShortcutOpenDestination
	ShortcutApplyRenameTool
	ShortcutFocusNewName
	ShortcutFocusDestinationPath
	ShortcutFocusSourcePath
	ShortcutFocusFileList
)

func altShortcutAction(key rune) int {
	switch unicode.ToUpper(key) {
	case 'B':
		return ShortcutBrowseSource
	case 'L':
		return ShortcutFocusFileList
	case 'W':
		return ShortcutFocusNewName
	case 'P':
		return ShortcutFocusDestinationPath
	case 'C':
		return ShortcutFocusSourcePath
	case 'T':
		return ShortcutReadText
	case 'D':
		return ShortcutBrowseDestination
	case 'M':
		return ShortcutManageDestinations
	case 'R':
		return ShortcutRename
	case 'V':
		return ShortcutMove
	case 'X':
		return ShortcutNext
	case 'S':
		return ShortcutSkip
	case 'U':
		return ShortcutUndo
	case 'O':
		return ShortcutOpen
	case 'F':
		return ShortcutOpenFolder
	case 'N':
		return ShortcutOpenDestination
	case 'A':
		return ShortcutApplyRenameTool
	default:
		return ShortcutNone
	}
}

// nextEnabledIndex returns the next enabled item in a wrapping focus order.
// current may be -1 when focus is outside the managed order.
func nextEnabledIndex(current int, enabled []bool, reverse bool) int {
	if len(enabled) == 0 {
		return -1
	}
	if current < -1 || current >= len(enabled) {
		current = -1
	}
	idx := current
	if reverse && idx == -1 {
		idx = 0
	}
	for attempts := 0; attempts < len(enabled); attempts++ {
		if reverse {
			idx--
			if idx < 0 {
				idx = len(enabled) - 1
			}
		} else {
			idx++
			if idx >= len(enabled) {
				idx = 0
			}
		}
		if enabled[idx] {
			return idx
		}
	}
	return -1
}
