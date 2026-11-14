//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	nativePreviewRectGetThreadDPI = user32.NewProc("GetThreadDpiAwarenessContext")
	nativePreviewRectSetThreadDPI = user32.NewProc("SetThreadDpiAwarenessContext")
	nativePreviewRectEqualDPI     = user32.NewProc("AreDpiAwarenessContextsEqual")
)

// nativePreviewPhysicalClientRect measures only in a temporary PMv2 context.
// It restores and verifies the caller's original context before returning a
// usable rectangle; it never changes the window or calls a preview handler.
func nativePreviewPhysicalClientRect(hwnd uintptr) (rc RECT, ok bool) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if hwnd == 0 {
		return RECT{}, false
	}
	for _, p := range []*syscall.LazyProc{nativePreviewRectGetThreadDPI, nativePreviewRectSetThreadDPI, nativePreviewRectEqualDPI, procGetClientRect} {
		if p.Find() != nil {
			return RECT{}, false
		}
	}
	original, _, _ := nativePreviewRectGetThreadDPI.Call()
	if original == 0 {
		return RECT{}, false
	}
	const pmv2 = ^uintptr(3)
	previous, _, _ := nativePreviewRectSetThreadDPI.Call(pmv2)
	if previous == 0 {
		return RECT{}, false
	}
	defer func() {
		// Use the saved caller context even if the switch's previous-context
		// invariant fails. A failed restoration invalidates the measurement.
		restored, _, _ := nativePreviewRectSetThreadDPI.Call(original)
		current, _, _ := nativePreviewRectGetThreadDPI.Call()
		equal, _, _ := nativePreviewRectEqualDPI.Call(current, original)
		if restored == 0 || current == 0 || equal == 0 {
			rc, ok = RECT{}, false
		}
	}()
	if equal, _, _ := nativePreviewRectEqualDPI.Call(previous, original); equal == 0 {
		return RECT{}, false
	}
	applied, _, _ := nativePreviewRectGetThreadDPI.Call()
	equal, _, _ := nativePreviewRectEqualDPI.Call(applied, pmv2)
	if applied == 0 || equal == 0 {
		return RECT{}, false
	}
	if read, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc))); read == 0 {
		return RECT{}, false
	}
	return rc, true
}
