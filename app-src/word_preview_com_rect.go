package main

import (
 "strings"
 "path/filepath"
)

// Explicitly opt-in for one diagnostic launch. No normal installation changes
// the native Word preview COM sizing contract.
func wordPreviewExperimentalMode(path, setting string) string {
 if !strings.EqualFold(filepath.Ext(path), ".docx") { return "" }
 switch strings.ToLower(strings.TrimSpace(setting)) {
 case "setrect": return "setrect"
 case "both": return "both"
 case "aware": return "aware"
 default: return ""
 }
}

// Choose physical pane pixels only for the requested Word COM method.
// Do not touch embedded Word child windows or change DPI awareness.
func wordPreviewRectForCall(mode, method string, logical, physical previewRectSize, measured bool) previewRectSize {
 if !measured || physical.W <= 0 || physical.H <= 0 { return logical }
 if mode == "both" && (method == "SetWindow" || method == "SetRect") { return physical }
 if mode == "setrect" && method == "SetRect" { return physical }
 return logical
}

type previewRectSize struct { W, H int32 }
