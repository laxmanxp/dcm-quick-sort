package main

import (
 "path/filepath"
 "strings"
)

// HTML's native Windows preview host needs its outer child resized to the
// QuickSort pane; PMv2 COM calls alone did not work in Lenovo qualification.
// "off" restores the unmodified HTML handler for A/B testing.
func htmlPreviewDPIContextMode(path, setting string) string {
 switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
 case ".html", ".htm":
  switch strings.ToLower(strings.TrimSpace(setting)) {
  case "", "repair": return "repair"
  case "aware": return "aware" // isolated diagnostic; not production default
  }
 }
 return ""
}
func nativePreviewDPIContextMode(path, wordSetting, htmlSetting string) string {
 if mode:=wordPreviewDPIContextMode(path,wordSetting);mode!="" {return mode}
 return htmlPreviewDPIContextMode(path,htmlSetting)
}
