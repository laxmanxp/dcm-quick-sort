//go:build windows

package main

// repairExcelPreviewOuterWindow will request a bounded resize of the unique
// Excel outer child of an owned preview pane. A true result will mean that the
// positioning request was accepted and the caller's DPI context was restored;
// it will not certify asynchronous completion or rendered document content.
//
// This deliberately inactive scaffold establishes the native geometry RED
// before any implementation or application call site is introduced.
func repairExcelPreviewOuterWindow(pane uintptr) bool {
	return false
}
