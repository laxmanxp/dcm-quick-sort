//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	spEnumWindows    = user32.NewProc("EnumWindows")
	spGetWindowOwner = user32.NewProc("GetWindow")
	spGetWindowStyle = user32.NewProc("GetWindowLongPtrW")
	spReadWindowText = user32.NewProc("SendMessageTimeoutW")
)

// Observe only our pinned process's own password dialog, never a different app.
// Do not read password text or collect/decrypt documents in the host.
type sumatraPasswordQuery struct {
	viewer *sumatraSession
	dialog uintptr
	found  bool
}

var sumatraFindPasswordEdit = syscall.NewCallback(func(hwnd, context uintptr) uintptr {
	q := (*sumatraPasswordQuery)(unsafe.Pointer(context))
	parent, _, _ := procGetParent.Call(hwnd)
	if parent != q.dialog || !sumatraHasPID(hwnd, q.viewer.pid) || sumatraClass(hwnd) != "Edit" {
		return 1
	}
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	style, _, _ := spGetWindowStyle.Call(hwnd, ^uintptr(15)) // GWL_STYLE = -16
	if visible != 0 && style&0x20 != 0 {
		q.found = true
		return 0
	} // ES_PASSWORD
	return 1
})
var sumatraFindPasswordWindow = syscall.NewCallback(func(hwnd, context uintptr) uintptr {
	q := (*sumatraPasswordQuery)(unsafe.Pointer(context))
	if !sumatraHasPID(hwnd, q.viewer.pid) || sumatraClass(hwnd) != "#32770" {
		return 1
	}
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	owner, _, _ := spGetWindowOwner.Call(hwnd, 4) // GW_OWNER
	if visible == 0 || !sumatraHasPID(owner, q.viewer.pid) || sumatraClass(owner) != "SUMATRA_PDF_FRAME" {
		return 1
	}
	q.dialog = hwnd
	spEnumChildren.Call(hwnd, sumatraFindPasswordEdit, context)
	if q.found {
		return 0
	}
	return 1
})

func sumatraWaitingForPassword(v *sumatraSession) bool {
	if v == nil || sumatraView != v {
		return false
	}
	q := &sumatraPasswordQuery{viewer: v}
	spEnumWindows.Call(sumatraFindPasswordWindow, uintptr(unsafe.Pointer(q)))
	runtime.KeepAlive(q)
	return q.found
}

type sumatraPageQuery struct {
	viewer  *sumatraSession
	toolbar uintptr
	known   bool
	label   string
}

var sumatraReadPageField = syscall.NewCallback(func(hwnd, context uintptr) uintptr {
	q := (*sumatraPageQuery)(unsafe.Pointer(context))
	parent, _, _ := procGetParent.Call(hwnd)
	if parent != q.toolbar || !sumatraHasPID(hwnd, q.viewer.pid) || sumatraClass(hwnd) != "Edit" {
		return 1
	}
	style, _, _ := spGetWindowStyle.Call(hwnd, ^uintptr(15))
	if style&0x2000 == 0 {
		return 1
	} // Pinned runtime's page field: ES_NUMBER, not search/password.
	var text [96]uint16
	var copied uintptr
	ok, _, _ := spReadWindowText.Call(hwnd, 0xD, uintptr(len(text)), uintptr(unsafe.Pointer(&text[0])), 0x2, 30, uintptr(unsafe.Pointer(&copied)))
	if ok != 0 {
		q.known = true
		q.label = syscall.UTF16ToString(text[:])
	}
	runtime.KeepAlive(text)
	return 0
})

func sumatraCurrentDocumentStatus(v *sumatraSession) string {
	if !sumatraOwnsWindow(v, v.child) {
		return sumatraDocumentStatus(false, "")
	}
	toolbar := &sumatraToolbarQuery{viewer: v}
	spEnumChildren.Call(v.child, sumatraFindToolbar, uintptr(unsafe.Pointer(toolbar)))
	q := &sumatraPageQuery{viewer: v, toolbar: toolbar.toolbar}
	if q.toolbar != 0 {
		spEnumChildren.Call(q.toolbar, sumatraReadPageField, uintptr(unsafe.Pointer(q)))
	}
	runtime.KeepAlive(toolbar)
	runtime.KeepAlive(q)
	return sumatraDocumentStatus(q.known, q.label)
}
func updateSumatraStatus(v *sumatraSession, status string) {
	recordStartupReady(v.traceID, status)
	if sumatraView == v && v.status != status {
		v.status = status
		if v.child == 0 {
			app.fallback = status
			procInvalidateRect.Call(app.previewHwnd, 0, 1)
		}
		setText(app.statusLabel, status)
	}
}
