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
  case "physical-setrect", "physical-both": return strings.ToLower(strings.TrimSpace(setting))
  }
 }
 return ""
}
func nativePreviewDPIContextMode(path, wordSetting, htmlSetting string) string {
 if mode:=wordPreviewDPIContextMode(path,wordSetting);mode!="" { return mode }
 return htmlPreviewDPIContextMode(path,htmlSetting)
}

func htmlPreviewRectForCall(mode, method string, logicalW, logicalH, physicalW, physicalH int32, measured bool) (int32,int32) {
 if !measured || physicalW <= 0 || physicalH <= 0 { return logicalW,logicalH }
 if mode == "physical-both" || (mode == "physical-setrect" && method == "SetRect") {
  return physicalW,physicalH
 }
 return logicalW,logicalH
}
