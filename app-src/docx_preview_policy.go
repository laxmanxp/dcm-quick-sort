package main

import (
 "path/filepath"
 "strings"
)

// The DCM DOCX text/structure view avoids native Word preview-pane overlap.
// Legacy .doc and other Office formats retain their existing preview paths.
func preferOwnedDOCXPreview(path string) bool {
 return strings.EqualFold(filepath.Ext(path), ".docx")
}
