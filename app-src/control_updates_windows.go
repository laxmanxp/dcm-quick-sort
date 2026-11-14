//go:build windows

package main

import (
	"strings"
	"unsafe"
)

var controlGetThreadDPI = user32.NewProc("GetThreadDpiAwarenessContext")
var controlEqualDPI = user32.NewProc("AreDpiAwarenessContextsEqual")

// This is deliberately not a geometry cache. Every decision uses the current
// HWND, parent-relative position and the combo's complete dropped-down extent.
// Changed geometry, unknown state, mixed DPI contexts or an open drop list use
// the original MoveWindow call. Preview-pane sizing never uses this helper.
func moveLayoutCombo(hwnd, x, y, width, height uintptr) bool {
	if hwnd == 0 {
		return false
	}
	if layoutComboAlreadyMatches(hwnd, x, y, width, height) {
		recordAttachmentTrace("combo-noop")
		return false
	}
	procMoveWindow.Call(hwnd, x, y, width, height, 0)
	return true
}

func layoutComboAlreadyMatches(hwnd, x, y, width, height uintptr) bool {
	thread, _, _ := spGetPID.Call(hwnd, 0)
	current, _, _ := layoutTraceThread.Call()
	if thread == 0 || thread != current || !strings.EqualFold(sumatraClass(hwnd), "ComboBox") {
		return false
	}
	parent, _, _ := procGetParent.Call(hwnd)
	if parent == 0 || width == 0 || height == 0 {
		return false
	}
	threadDPI, _, _ := controlGetThreadDPI.Call()
	windowDPI, _, _ := spGetWindowDPIContext.Call(hwnd)
	parentDPI, _, _ := spGetWindowDPIContext.Call(parent)
	equalWindow, _, _ := controlEqualDPI.Call(threadDPI, windowDPI)
	equalParent, _, _ := controlEqualDPI.Call(threadDPI, parentDPI)
	if threadDPI == 0 || windowDPI == 0 || parentDPI == 0 || equalWindow == 0 || equalParent == 0 {
		return false
	}
	exStyle, _, _ := spGetWindowStyle.Call(parent, ^uintptr(19)) // GWL_EXSTYLE
	if exStyle&0x00400000 != 0 {                                 // WS_EX_LAYOUTRTL: do not guess mirrored coordinates
		return false
	}
	opened, _, _ := procSendMessageW.Call(hwnd, 0x0157, 0, 0) // CB_GETDROPPEDSTATE
	if opened != 0 {
		return false
	}
	var collapsed, dropped RECT
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&collapsed))); ok == 0 {
		return false
	}
	if ok, _, _ := procSendMessageW.Call(hwnd, 0x0152, 0, uintptr(unsafe.Pointer(&dropped))); ok == 0 {
		return false
	}
	point := POINT{X: collapsed.Left, Y: collapsed.Top}
	if ok, _, _ := spScreenToClient.Call(parent, uintptr(unsafe.Pointer(&point))); ok == 0 {
		return false
	}
	return point.X == int32(x) && point.Y == int32(y) &&
		collapsed.Right-collapsed.Left == int32(width) &&
		dropped.Left == collapsed.Left && dropped.Top == collapsed.Top &&
		dropped.Right-dropped.Left == int32(width) && dropped.Bottom-dropped.Top == int32(height)
}

// Test the child's own style, not ancestor visibility. A hidden parent must not
// make a visible child look as though its WS_VISIBLE state needs changing.
func setPreviewControlVisible(hwnd uintptr, show bool) bool {
	if hwnd == 0 {
		return false
	}
	style, _, _ := spGetWindowStyle.Call(hwnd, ^uintptr(15)) // GWL_STYLE
	if style != 0 && (style&WS_VISIBLE != 0) == show {
		recordAttachmentTrace("visibility-noop")
		return false
	}
	command := uintptr(0)
	if show {
		command = 5 // preserve the original SW_SHOW behavior
	}
	procShowWindow.Call(hwnd, command)
	return true
}
func setPreviewControlEnabled(hwnd uintptr, enabled bool) bool {
	if hwnd == 0 {
		return false
	}
	style, _, _ := spGetWindowStyle.Call(hwnd, ^uintptr(15)) // GWL_STYLE
	if style != 0 && (style&0x08000000 == 0) == enabled {    // WS_DISABLED
		recordAttachmentTrace("enabled-noop")
		return false
	}
	value := uintptr(0)
	if enabled {
		value = 1
	}
	procEnableWindow.Call(hwnd, value)
	return true
}
