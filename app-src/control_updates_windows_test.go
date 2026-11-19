//go:build windows

package main

import (
	"runtime"
	"testing"
	"unsafe"
)

func controlTestParent(t *testing.T) uintptr {
	t.Helper()
	h := createControl("STATIC", "control-update-private-test", 0, 0, 0, 600, 500, 0, 0)
	if h == 0 {
		t.Fatal("private test parent creation failed")
	}
	return h
}
func controlTestRect(t *testing.T, h uintptr, dropped bool) RECT {
	t.Helper()
	var rc RECT
	var ok uintptr
	if dropped {
		ok, _, _ = procSendMessageW.Call(h, 0x0152, 0, uintptr(unsafe.Pointer(&rc)))
	} else {
		ok, _, _ = procGetWindowRect.Call(h, uintptr(unsafe.Pointer(&rc)))
	}
	if ok == 0 {
		t.Fatal("live rectangle unavailable")
	}
	return rc
}
func controlTestCombo(t *testing.T, parent uintptr, id int) uintptr {
	t.Helper()
	h := createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 20, 20, 120, 200, parent, id)
	if h == 0 {
		t.Fatal("combo creation failed")
	}
	for _, s := range []string{"One", "Two", "Three", "Four", "Five", "Six"} {
		procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(s))))
	}
	return h
}

func TestControlUpdateGeometry(t *testing.T) {
	for _, ctx := range []struct {
		name  string
		value uintptr
	}{{"unaware", ^uintptr(0)}, {"per-monitor-v2", ^uintptr(3)}} {
		t.Run(ctx.name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			previous, _, _ := spSetThreadDPIContext.Call(ctx.value)
			if previous == 0 {
				t.Fatal("required test DPI context unavailable")
			}
			defer spSetThreadDPIContext.Call(previous)
			parent := controlTestParent(t)
			defer user32.NewProc("DestroyWindow").Call(parent)
			h := controlTestCombo(t, parent, 911)
			if !moveLayoutCombo(h, 85, 13, 150, 240) {
				t.Fatal("changed geometry not applied")
			}
			rc, drop := controlTestRect(t, h, false), controlTestRect(t, h, true)
			if moveLayoutCombo(h, 85, 13, 150, 240) {
				t.Fatalf("redundant dropdown move was not skipped; collapsed=%+v dropped=%+v", rc, drop)
			}
			// Changed inputs, including the drop-list height, must still reach Windows.
			for _, r := range [][4]uintptr{{90, 13, 150, 240}, {90, 18, 150, 240}, {90, 18, 170, 240}, {90, 18, 170, 300}, {85, 13, 150, 240}} {
				if !moveLayoutCombo(h, r[0], r[1], r[2], r[3]) {
					t.Fatalf("changed geometry skipped: %v", r)
				}
				got, gotDrop := controlTestRect(t, h, false), controlTestRect(t, h, true)
				procMoveWindow.Call(h, r[0], r[1], r[2], r[3], 0)
				if got != controlTestRect(t, h, false) || gotDrop != controlTestRect(t, h, true) {
					t.Fatalf("helper differs from original MoveWindow for %v", r)
				}
			}
			// An external change cannot be hidden by an old requested geometry.
			procMoveWindow.Call(h, 120, 70, 180, 280, 0)
			if !moveLayoutCombo(h, 85, 13, 150, 240) {
				t.Fatal("external movement was not repaired")
			}
			// Parent movement changes screen coordinates, not child-relative layout.
			procMoveWindow.Call(parent, 45, 60, 620, 520, 0)
			if moveLayoutCombo(h, 85, 13, 150, 240) {
				t.Fatal("parent-only move invalidated matching relative geometry")
			}
			procShowWindow.Call(h, 0)
			if moveLayoutCombo(h, 85, 13, 150, 240) {
				t.Fatal("hidden unchanged combo should not move")
			}
			procShowWindow.Call(h, 5)
			if moveLayoutCombo(h, 85, 13, 150, 240) {
				t.Fatal("restored unchanged combo should not move")
			}
			user32.NewProc("DestroyWindow").Call(h)
			h = controlTestCombo(t, parent, 911)
			if !moveLayoutCombo(h, 85, 13, 150, 240) {
				t.Fatal("recreated combo failed to receive changed geometry")
			}
			// A font-driven height change must be observed fresh, without caching.
			procSendMessageW.Call(h, 0x0153, ^uintptr(0), 32) // CB_SETITEMHEIGHT, selection field
			gotBefore := controlTestRect(t, h, false)
			moveLayoutCombo(h, 85, 13, 150, 240)
			got, gotDrop := controlTestRect(t, h, false), controlTestRect(t, h, true)
			procMoveWindow.Call(h, 85, 13, 150, 240, 0)
			if got != controlTestRect(t, h, false) || gotDrop != controlTestRect(t, h, true) {
				t.Fatalf("font/item-height update differs: before=%+v", gotBefore)
			}
		})
	}
}

func TestControlUpdateState(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	parent := controlTestParent(t)
	defer user32.NewProc("DestroyWindow").Call(parent)
	h := createControl("BUTTON", "Zoom", WS_CHILD|WS_VISIBLE, 10, 10, 80, 25, parent, 912)
	if h == 0 {
		t.Fatal("button creation failed")
	}
	// The parent is hidden, but the child's own WS_VISIBLE state is set.
	if setPreviewControlVisible(h, true) {
		t.Fatal("hidden ancestor caused redundant child ShowWindow")
	}
	if !setPreviewControlVisible(h, false) || setPreviewControlVisible(h, false) {
		t.Fatal("hide transition/idempotence failed")
	}
	if !setPreviewControlVisible(h, true) || setPreviewControlVisible(h, true) {
		t.Fatal("show transition/idempotence failed")
	}
	if setPreviewControlEnabled(h, true) {
		t.Fatal("redundant enable was not skipped")
	}
	if !setPreviewControlEnabled(h, false) || setPreviewControlEnabled(h, false) {
		t.Fatal("disable transition/idempotence failed")
	}
	if !setPreviewControlEnabled(h, true) || setPreviewControlEnabled(h, true) {
		t.Fatal("enable transition/idempotence failed")
	}
	procShowWindow.Call(h, 0)
	procEnableWindow.Call(h, 0)
	if !setPreviewControlVisible(h, true) || !setPreviewControlEnabled(h, true) {
		t.Fatal("external control state not restored")
	}
	if setPreviewControlVisible(0, true) || setPreviewControlEnabled(0, true) || moveLayoutCombo(0, 0, 0, 0, 0) {
		t.Fatal("null handles must be no-op")
	}
}
