package main

import (
	"path/filepath"
	"strings"
)

// actualFileDisplay separates the on-disk filename from the editable rename
// stem. The extension remains visible as a short, independent file-type label
// even when the filename itself is wider than the read-only display field.
func actualFileDisplay(path string) (filename, fileType string) {
	if strings.TrimSpace(path) == "" {
		return "", ""
	}
	filename = filepath.Base(path)
	if filename == "." || filename == string(filepath.Separator) || filename == "" {
		return "", ""
	}
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "" {
		return filename, "NO EXT."
	}
	return filename, strings.ToUpper(ext)
}
