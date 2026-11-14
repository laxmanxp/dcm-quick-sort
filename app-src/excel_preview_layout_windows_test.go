//go:build windows

package main

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

var (
	dcm31ExcelUnregisterClass = user32.NewProc("UnregisterClassW")
	dcm31ExcelGetActiveWindow = user32.NewProc("GetActiveWindow")
)

// The classes are local to this test process, with a native default window
// procedure. No Office process or preview handler is created or controlled.
func dcm31ExcelRegisterClasses(t *testing.T) func() {
	t.Helper()
	hInst, _, _ := procGetModuleHandleW.Call(0)
	var atoms []uintptr
	remove := func() {
		for i := len(atoms) - 1; i >= 0; i-- {
			if ok, _, err := dcm31ExcelUnregisterClass.Call(atoms[i], hInst); ok == 0 {
				t.Errorf("private Excel test class removal failed: %v", err)
			}
		}
	}
	for _, name := range []string{"EXCEL9", "EXCEL7"} {
		wc := WNDCLASSEXW{
			CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
			LpfnWndProc:   procDefWindowProcW.Addr(),
			HInstance:     hInst,
			LpszClassName: wstr(name),
		}
		atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		if atom == 0 {
			remove()
			t.Fatalf("private class %s registration failed; no existing class is reused: %v", name, err)
		}
		atoms = append(atoms, atom)
	}
	return remove
}

type dcm31ExcelFixture struct {
	pane, outer, inner uintptr
	windows            []uintptr
}

func (f *dcm31ExcelFixture) parent(t *testing.T) uintptr {
	t.Helper()
	hwnd := dcm31RectHiddenWindow(t)
	f.windows = append(f.windows, hwnd)
	return hwnd
}

func (f *dcm31ExcelFixture) window(t *testing.T, parent uintptr, class string, style uintptr, width, height int32) uintptr {
	t.Helper()
	hInst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, err := procCreateWindowExW.Call(0x08000080, // no activation, no taskbar entry
		uintptr(unsafe.Pointer(wstr(class))), uintptr(unsafe.Pointer(wstr("DCM31 private Excel layout test"))),
		style, 0, 0, uintptr(width), uintptr(height), parent, 0, hInst, 0)
	if hwnd == 0 {
		t.Fatalf("private %s creation failed: %v", class, err)
	}
	f.windows = append(f.windows, hwnd)
	var pid uint32
	ownerThread, _, _ := dcm31RectWindowOwner.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	currentThread, _, _ := dcm31RectThreadID.Call()
	actualParent, _, _ := procGetParent.Call(hwnd)
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	actualStyle, _, _ := spGetWindowStyle.Call(hwnd, ^uintptr(15))
	windowContext, _, _ := dcm31RectGetWindowDPI.Call(hwnd)
	if pid != uint32(os.Getpid()) || ownerThread != currentThread || actualParent != parent || visible != 0 ||
		actualStyle&(WS_CHILD|0x80000000) != style || actualStyle&(WS_VISIBLE|0x00C40000) != 0 ||
		!dcm31RectContextsEqual(windowContext, dcm31RectCurrentContext(t)) {
		t.Fatalf("private %s must retain its owned, hidden, borderless fixture identity: pid=%d thread=%d/%d parent=%x/%x visible=%d style=%x", class, pid, ownerThread, currentThread, actualParent, parent, visible, actualStyle)
	}
	return hwnd
}

func (f *dcm31ExcelFixture) child(t *testing.T, parent uintptr, class string, width, height int32) uintptr {
	t.Helper()
	return f.window(t, parent, class, WS_CHILD, width, height)
}

func (f *dcm31ExcelFixture) destroy(t *testing.T) {
	t.Helper()
	// Destroy in reverse creation order on the same locked OS thread. Do not use
	// t.Cleanup: that can outlive the caller's locked thread and DPI scope.
	for i := len(f.windows) - 1; i >= 0; i-- {
		if f.windows[i] != 0 {
			dcm31RectDestroyWindow(t, f.windows[i])
		}
	}
}

func (f *dcm31ExcelFixture) validTree(t *testing.T) {
	t.Helper()
	f.outer = f.child(t, f.pane, "EXCEL9", 240, 180)
	f.inner = f.child(t, f.outer, "EXCEL7", 320, 240)
}

type dcm31ExcelWindowState struct {
	rect                            RECT
	parent, context, style, exStyle uintptr
	visible, thread                 uintptr
	pid                             uint32
	class                           string
}

func dcm31ExcelWindowSnapshot(t *testing.T, hwnd uintptr) dcm31ExcelWindowState {
	t.Helper()
	var state dcm31ExcelWindowState
	state.rect = dcm31RectPhysicalOracle(t, hwnd)
	state.parent, _, _ = procGetParent.Call(hwnd)
	state.context, _, _ = dcm31RectGetWindowDPI.Call(hwnd)
	state.style, _, _ = spGetWindowStyle.Call(hwnd, ^uintptr(15))
	state.exStyle, _, _ = spGetWindowStyle.Call(hwnd, ^uintptr(19))
	state.visible, _, _ = procIsWindowVisible.Call(hwnd)
	state.thread, _, _ = dcm31RectWindowOwner.Call(hwnd, uintptr(unsafe.Pointer(&state.pid)))
	var class [128]uint16
	if n, _, _ := spGetClass.Call(hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class))); n == 0 {
		t.Fatal("private window class query failed")
	}
	state.class = syscall.UTF16ToString(class[:])
	if state.context == 0 || state.thread == 0 || state.pid != uint32(os.Getpid()) {
		t.Fatal("private window snapshot lost its owned HWND or DPI context")
	}
	return state
}

func (f *dcm31ExcelFixture) snapshot(t *testing.T) map[uintptr]dcm31ExcelWindowState {
	t.Helper()
	states := make(map[uintptr]dcm31ExcelWindowState)
	for _, hwnd := range f.windows {
		if hwnd != 0 {
			states[hwnd] = dcm31ExcelWindowSnapshot(t, hwnd)
		}
	}
	return states
}

func dcm31ExcelCheckUnchanged(t *testing.T, before map[uintptr]dcm31ExcelWindowState, resized uintptr) {
	t.Helper()
	for hwnd, want := range before {
		got := dcm31ExcelWindowSnapshot(t, hwnd)
		if !dcm31RectContextsEqual(got.context, want.context) {
			t.Errorf("private window %x changed DPI awareness", hwnd)
		}
		got.context = want.context // compare semantic context separately
		if hwnd == resized {
			got.rect = want.rect // only the selected outer rectangle may change
		}
		if got != want {
			t.Errorf("private window %x changed outside the requested outer rectangle: before=%+v after=%+v", hwnd, want, got)
		}
	}
}

func dcm31ExcelCallAndCheckContext(t *testing.T, pane uintptr) bool {
	t.Helper()
	before := dcm31RectCurrentContext(t)
	activeBefore, _, _ := dcm31ExcelGetActiveWindow.Call()
	focusBefore, _, _ := procGetFocus.Call()
	accepted := repairExcelPreviewOuterWindow(pane)
	after := dcm31RectCurrentContext(t)
	activeAfter, _, _ := dcm31ExcelGetActiveWindow.Call()
	focusAfter, _, _ := procGetFocus.Call()
	if !dcm31RectContextsEqual(before, after) {
		t.Error("Excel layout request changed the caller's DPI context")
	}
	if activeAfter != activeBefore || focusAfter != focusBefore {
		t.Errorf("Excel layout request changed thread activation/focus: active %x -> %x; focus %x -> %x", activeBefore, activeAfter, focusBefore, focusAfter)
	}
	return accepted
}

func TestDCM31ExcelPreviewLayout(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dcm31RectRequireContexts(t)
	dcm31RectRequireNon100Scale(t)
	removeClasses := dcm31ExcelRegisterClasses(t)
	defer removeClasses()
	for _, creator := range dcm31RectContexts {
		for _, caller := range dcm31RectContexts {
			t.Run(creator.name+"-parent/"+caller.name+"-caller", func(t *testing.T) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				restoreCreator := dcm31RectEnterContext(t, creator.value)
				defer restoreCreator()
				f := &dcm31ExcelFixture{}
				defer f.destroy(t)
				f.pane = f.parent(t)
				f.validTree(t)
				f.child(t, f.pane, "STATIC", 20, 20) // unrelated sibling must remain unchanged
				before := f.snapshot(t)
				want := before[f.pane].rect // borderless parent: independent physical client oracle
				if before[f.outer].rect == want || before[f.inner].rect != want {
					t.Fatalf("fixture must begin with an undersized outer and full-size inner: parent=%+v outer=%+v inner=%+v", want, before[f.outer].rect, before[f.inner].rect)
				}
				restoreCaller := dcm31RectEnterContext(t, caller.value)
				defer restoreCaller()
				accepted := dcm31ExcelCallAndCheckContext(t, f.pane)
				got := dcm31RectPhysicalOracle(t, f.outer)
				// These same-thread owned HWNDs apply SetWindowPos synchronously,
				// even with SWP_ASYNCWINDOWPOS. No polling can substitute for the
				// separate foreign-provider UI acceptance and image review.
				dcm31ExcelCheckUnchanged(t, before, f.outer)
				t.Logf("creator=%s caller=%s parent=%+v outer-before=%+v outer-after=%+v inner=%+v request-accepted=%t", creator.name, caller.name, want, before[f.outer].rect, got, before[f.inner].rect, accepted)
				if !accepted || got != want {
					t.Errorf("DCM31_EXCEL_PREVIEW_OUTER_LAYOUT: accepted=%t physical outer=%+v want parent client=%+v", accepted, got, want)
				}
			})
		}
	}
}

func TestDCM31ExcelPreviewLayoutGuards(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dcm31RectRequireContexts(t)
	removeClasses := dcm31ExcelRegisterClasses(t)
	defer removeClasses()
	cases := []string{
		"null-pane", "destroyed-pane", "missing-outer", "wrong-outer-class", "nested-outer", "ambiguous-outer",
		"missing-inner", "wrong-inner-class", "nested-inner", "ambiguous-inner", "wrong-inner-parent",
		"non-child-outer", "non-child-inner",
	}
	for _, caller := range dcm31RectContexts {
		for _, name := range cases {
			t.Run(caller.name+"-caller/"+name, func(t *testing.T) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				restoreCreator := dcm31RectEnterContext(t, ^uintptr(0))
				defer restoreCreator()
				f := &dcm31ExcelFixture{}
				defer f.destroy(t)
				f.pane = f.parent(t)
				f.child(t, f.pane, "STATIC", 20, 20)
				argument := f.pane
				switch name {
				case "null-pane":
					f.validTree(t)
					argument = 0
				case "destroyed-pane":
					f.validTree(t)
					argument = f.parent(t)
					dcm31RectDestroyWindow(t, argument)
					f.windows[len(f.windows)-1] = 0
					if live, _, _ := dcm31RectIsWindow.Call(argument); live != 0 {
						t.Fatal("destroyed test pane is still live")
					}
				case "missing-outer":
				case "wrong-outer-class":
					outer := f.child(t, f.pane, "STATIC", 240, 180)
					f.child(t, outer, "EXCEL7", 320, 240)
				case "nested-outer":
					wrapper := f.child(t, f.pane, "STATIC", 240, 180)
					outer := f.child(t, wrapper, "EXCEL9", 240, 180)
					f.child(t, outer, "EXCEL7", 320, 240)
				case "ambiguous-outer":
					f.validTree(t)
					other := f.child(t, f.pane, "EXCEL9", 240, 180)
					f.child(t, other, "EXCEL7", 320, 240)
				case "missing-inner":
					f.child(t, f.pane, "EXCEL9", 240, 180)
				case "wrong-inner-class":
					outer := f.child(t, f.pane, "EXCEL9", 240, 180)
					f.child(t, outer, "STATIC", 320, 240)
				case "nested-inner":
					outer := f.child(t, f.pane, "EXCEL9", 240, 180)
					wrapper := f.child(t, outer, "STATIC", 320, 240)
					f.child(t, wrapper, "EXCEL7", 320, 240)
				case "ambiguous-inner":
					f.validTree(t)
					f.child(t, f.outer, "EXCEL7", 320, 240)
				case "wrong-inner-parent":
					f.child(t, f.pane, "EXCEL9", 240, 180)
					other := f.child(t, f.pane, "STATIC", 320, 240)
					f.child(t, other, "EXCEL7", 320, 240)
				case "non-child-outer":
					outer := f.window(t, f.pane, "EXCEL9", 0x80000000, 240, 180)
					f.child(t, outer, "EXCEL7", 320, 240)
				case "non-child-inner":
					outer := f.child(t, f.pane, "EXCEL9", 240, 180)
					f.window(t, outer, "EXCEL7", 0x80000000, 320, 240)
				}
				before := f.snapshot(t)
				restoreCaller := dcm31RectEnterContext(t, caller.value)
				defer restoreCaller()
				if accepted := dcm31ExcelCallAndCheckContext(t, argument); accepted {
					t.Errorf("DCM31_EXCEL_PREVIEW_LAYOUT_GUARD: invalid %s fixture accepted a resize", name)
				}
				dcm31ExcelCheckUnchanged(t, before, 0)
			})
		}
	}
}
