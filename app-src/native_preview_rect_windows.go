//go:build windows

package main

import "unsafe"

// nativePreviewPhysicalClientRect is the measurement boundary for native preview
// coordinates. This walking skeleton deliberately retains the existing logical
// GetClientRect behavior so the native DPI test can establish RED. It is not used
// by the application until the physical-coordinate implementation is qualified.
func nativePreviewPhysicalClientRect(hwnd uintptr) (RECT, bool) {
	var rc RECT
	if hwnd == 0 {
		return rc, false
	}
	ok, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	return rc, ok != 0
}
