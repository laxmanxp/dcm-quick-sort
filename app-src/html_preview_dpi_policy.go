package main

import (
 "path/filepath"
 "strings"
)

// The HTML native preview host on Lenovo measured 80% of its allocated pane
// despite successful SetWindow and SetRect calls. Match the proven DOCX DPI
// context contract while leaving the rectangle unchanged. "off" is an opt-out.
func htmlPreviewDPIContextMode(path, setting string) string {
 switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
 case ".html", ".htm":
  switch strings.ToLower(strings.TrimSpace(setting)) {
  case "", "aware": return "aware"
  }
 }
 return ""
}
func nativePreviewDPIContextMode(path, wordSetting, htmlSetting string) string {
 if mode:=wordPreviewDPIContextMode(path,wordSetting);mode!="" { return mode }
 return htmlPreviewDPIContextMode(path,htmlSetting)
}
