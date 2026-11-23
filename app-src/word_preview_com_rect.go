package main

import (
 "path/filepath"
 "strings"
)

// Word's native preview process requires a PMv2 caller DPI context when the
// host's logical pane rectangle is sent to IPreviewHandler. The logical rect
// is never scaled or enlarged. An explicit "off" preserves the old behavior
// for acceptance comparisons; other file types remain unchanged.
func wordPreviewDPIContextMode(path, setting string) string {
 if !strings.EqualFold(filepath.Ext(path), ".docx") { return "" }
 switch strings.ToLower(strings.TrimSpace(setting)) {
 case "", "aware": return "aware"
 case "lifecycle": return "lifecycle"
 default: return ""
 }
}
