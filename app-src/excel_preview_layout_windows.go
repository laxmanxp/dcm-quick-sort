//go:build windows

package main

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	excelLayoutFindWindowEx = user32.NewProc("FindWindowExW")
	excelLayoutGetThreadDPI = user32.NewProc("GetThreadDpiAwarenessContext")
	excelLayoutSetThreadDPI = user32.NewProc("SetThreadDpiAwarenessContext")
	excelLayoutGetWindowDPI = user32.NewProc("GetWindowDpiAwarenessContext")
	excelLayoutEqualDPI     = user32.NewProc("AreDpiAwarenessContextsEqual")
)

type excelLayoutWindow struct {
	hwnd, parent, thread, context uintptr
	pid                           uint32
	class                         string
	child                         bool
}

type excelLayoutTree struct {
	pane, outer, inner excelLayoutWindow
}

func excelLayoutContextsEqual(a, b uintptr) bool {
	if a == 0 || b == 0 {
		return false
	}
	equal, _, _ := excelLayoutEqualDPI.Call(a, b)
	return equal != 0
}

func excelLayoutReadWindow(hwnd uintptr) (excelLayoutWindow, bool) {
	if hwnd == 0 {
		return excelLayoutWindow{}, false
	}
	w := excelLayoutWindow{hwnd: hwnd}
	w.thread, _, _ = spGetPID.Call(hwnd, uintptr(unsafe.Pointer(&w.pid)))
	w.context, _, _ = excelLayoutGetWindowDPI.Call(hwnd)
	if w.thread == 0 || w.pid == 0 || w.context == 0 {
		return excelLayoutWindow{}, false
	}
	w.parent, _, _ = procGetParent.Call(hwnd)
	style, _, _ := spGetWindowStyle.Call(hwnd, ^uintptr(15)) // GWL_STYLE
	w.child = style&WS_CHILD != 0
	var class [128]uint16
	n, _, _ := spGetClass.Call(hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
	if n == 0 || n >= uintptr(len(class)-1) {
		return excelLayoutWindow{}, false
	}
	w.class = syscall.UTF16ToString(class[:])
	return w, true
}

func excelLayoutUniqueChild(parent uintptr, class string) (excelLayoutWindow, bool) {
	if parent == 0 {
		return excelLayoutWindow{}, false
	}
	className := wstr(class)
	// FindWindowEx searches immediate children only. A nil title avoids asking
	// the foreign provider for window text; no input is sent to its controls.
	hwnd, _, _ := excelLayoutFindWindowEx.Call(parent, 0, uintptr(unsafe.Pointer(className)), 0)
	if hwnd == 0 {
		return excelLayoutWindow{}, false
	}
	other, _, _ := excelLayoutFindWindowEx.Call(parent, hwnd, uintptr(unsafe.Pointer(className)), 0)
	runtime.KeepAlive(className)
	if other != 0 {
		return excelLayoutWindow{}, false
	}
	w, ok := excelLayoutReadWindow(hwnd)
	// FindWindowEx compares class names without case sensitivity; retain the
	// exact observed Excel classes and explicitly exclude owned popup windows.
	if !ok || w.parent != parent || !w.child || w.class != class {
		return excelLayoutWindow{}, false
	}
	return w, true
}

func excelLayoutReadTree(pane uintptr) (excelLayoutTree, bool) {
	p, ok := excelLayoutReadWindow(pane)
	if !ok || p.pid != uint32(os.Getpid()) {
		return excelLayoutTree{}, false
	}
	outer, ok := excelLayoutUniqueChild(pane, "EXCEL9")
	if !ok || !excelLayoutContextsEqual(p.context, outer.context) {
		return excelLayoutTree{}, false
	}
	inner, ok := excelLayoutUniqueChild(outer.hwnd, "EXCEL7")
	if !ok || inner.pid != outer.pid || !excelLayoutContextsEqual(p.context, inner.context) {
		return excelLayoutTree{}, false
	}
	return excelLayoutTree{pane: p, outer: outer, inner: inner}, true
}

func excelLayoutSameWindow(a, b excelLayoutWindow) bool {
	return a.hwnd == b.hwnd && a.parent == b.parent && a.thread == b.thread &&
		a.pid == b.pid && a.class == b.class && a.child == b.child &&
		excelLayoutContextsEqual(a.context, b.context)
}

// repairExcelPreviewOuterWindow requests a resize of only the unique Excel
// outer child of an owned pane. The caller gates this on its active Excel
// handler. True means the request was accepted and the original thread DPI
// context restored, not that asynchronous layout or document rendering passed.
func repairExcelPreviewOuterWindow(pane uintptr) (accepted bool) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if pane == 0 {
		return false
	}
	for _, p := range []*syscall.LazyProc{
		excelLayoutFindWindowEx, excelLayoutGetThreadDPI, excelLayoutSetThreadDPI,
		excelLayoutGetWindowDPI, excelLayoutEqualDPI, spGetPID, procGetParent,
		spGetWindowStyle, spGetClass, procGetClientRect, spSetWindowPos,
	} {
		if p.Find() != nil {
			return false
		}
	}
	original, _, _ := excelLayoutGetThreadDPI.Call()
	if original == 0 {
		return false
	}
	tree, ok := excelLayoutReadTree(pane)
	if !ok {
		return false
	}
	previous, _, _ := excelLayoutSetThreadDPI.Call(tree.pane.context)
	if previous == 0 {
		return false
	}
	defer func() {
		restored, _, _ := excelLayoutSetThreadDPI.Call(original)
		current, _, _ := excelLayoutGetThreadDPI.Call()
		if restored == 0 || !excelLayoutContextsEqual(current, original) {
			accepted = false
		}
	}()
	current, _, _ := excelLayoutGetThreadDPI.Call()
	if !excelLayoutContextsEqual(previous, original) || !excelLayoutContextsEqual(current, tree.pane.context) {
		return false
	}
	var rc RECT
	if read, _, _ := procGetClientRect.Call(pane, uintptr(unsafe.Pointer(&rc))); read == 0 ||
		rc.Left != 0 || rc.Top != 0 || rc.Right <= 0 || rc.Bottom <= 0 {
		return false
	}
	// Recheck uniqueness, direct parents, process/thread identities, styles and
	// DPI contexts immediately before touching the selected outer window.
	now, ok := excelLayoutReadTree(pane)
	if !ok || !excelLayoutSameWindow(tree.pane, now.pane) ||
		!excelLayoutSameWindow(tree.outer, now.outer) || !excelLayoutSameWindow(tree.inner, now.inner) {
		return false
	}
	// ASYNCWINDOWPOS | NOACTIVATE | NOZORDER | NOOWNERZORDER. No SHOWWINDOW,
	// reparenting, inner resize, COM call or timer is involved. The asynchronous
	// flag posts only when Windows finds different input queues.
	const flags = 0x4214
	result, _, _ := spSetWindowPos.Call(tree.outer.hwnd, 0, 0, 0, uintptr(rc.Right), uintptr(rc.Bottom), flags)
	return result != 0
}
