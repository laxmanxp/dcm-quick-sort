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
 default: return ""
 }
}

 
// The Word child viewport needs a one-time outer-host layout nudge in the
// tested two-pane configuration. Keep it separate from the COM DPI policy.
func wordPreviewOuterRepairEnabled(path, setting string) bool {
 if !strings.EqualFold(filepath.Ext(strings.TrimSpace(path)), ".docx") { return false }
 return !strings.EqualFold(strings.TrimSpace(setting), "off")
}
