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
	dcm31RectGetThreadDPI = user32.NewProc("GetThreadDpiAwarenessContext")
	dcm31RectSetThreadDPI = user32.NewProc("SetThreadDpiAwarenessContext")
	dcm31RectGetWindowDPI = user32.NewProc("GetWindowDpiAwarenessContext")
	dcm31RectEqualDPI     = user32.NewProc("AreDpiAwarenessContextsEqual")
	dcm31RectValidDPI     = user32.NewProc("IsValidDpiAwarenessContext")
	dcm31RectDestroy      = user32.NewProc("DestroyWindow")
	dcm31RectIsWindow     = user32.NewProc("IsWindow")
	dcm31RectWindowOwner  = user32.NewProc("GetWindowThreadProcessId")
	dcm31RectThreadID     = kernel32.NewProc("GetCurrentThreadId")
)

var dcm31RectContexts = []struct {
	name  string
	value uintptr
}{
	{"unaware", ^uintptr(0)},
	{"system", ^uintptr(1)},
	{"pmv2", ^uintptr(3)},
}

func dcm31RectRequireContexts(t *testing.T) {
	t.Helper()
	for _, p := range []*syscall.LazyProc{dcm31RectGetThreadDPI, dcm31RectSetThreadDPI, dcm31RectGetWindowDPI, dcm31RectEqualDPI, dcm31RectValidDPI} {
		if err := p.Find(); err != nil {
			t.Skipf("DCM31 physical-coordinate qualification unavailable: %s: %v", p.Name, err)
		}
	}
	for _, ctx := range dcm31RectContexts {
		if valid, _, _ := dcm31RectValidDPI.Call(ctx.value); valid == 0 {
			t.Skipf("DCM31 physical-coordinate qualification unavailable: %s DPI context unsupported", ctx.name)
		}
	}
}

func dcm31RectCurrentContext(t *testing.T) uintptr {
	t.Helper()
	ctx, _, _ := dcm31RectGetThreadDPI.Call()
	if ctx == 0 {
		t.Fatal("GetThreadDpiAwarenessContext failed")
	}
	return ctx
}

func dcm31RectContextsEqual(a, b uintptr) bool {
	equal, _, _ := dcm31RectEqualDPI.Call(a, b)
	return a != 0 && b != 0 && equal != 0
}

// All callers keep their OS thread locked until this deferred restoration and
// every owned HWND destruction have finished. Do not use t.Cleanup for HWNDs.
func dcm31RectEnterContext(t *testing.T, ctx uintptr) func() {
	t.Helper()
	before := dcm31RectCurrentContext(t)
	previous, _, _ := dcm31RectSetThreadDPI.Call(ctx)
	if previous == 0 {
		t.Fatal("SetThreadDpiAwarenessContext failed for a supported context")
	}
	current, _, _ := dcm31RectGetThreadDPI.Call()
	if !dcm31RectContextsEqual(before, previous) || !dcm31RectContextsEqual(current, ctx) {
		dcm31RectSetThreadDPI.Call(before)
		t.Fatal("DPI context switch did not preserve the expected previous/current contexts")
	}
	return func() {
		if result, _, _ := dcm31RectSetThreadDPI.Call(before); result == 0 {
			t.Error("DPI context restoration failed")
			return
		}
		current, _, _ := dcm31RectGetThreadDPI.Call()
		if !dcm31RectContextsEqual(current, before) {
			t.Error("DPI context restoration returned a different context")
		}
	}
}

func dcm31RectHiddenWindow(t *testing.T) uintptr {
	t.Helper()
	// WS_POPUP has no frame; no WS_VISIBLE/ShowWindow or activation is used.
	// Thus the PMv2 GetWindowRect size is an independent physical client oracle.
	const popup = 0x80000000
	const noActivateToolWindow = 0x08000080
	hInst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, err := procCreateWindowExW.Call(noActivateToolWindow,
		uintptr(unsafe.Pointer(wstr("STATIC"))), uintptr(unsafe.Pointer(wstr("DCM31 private rectangle test"))),
		popup, 64, 64, 320, 240, 0, 0, hInst, 0)
	if hwnd == 0 {
		t.Fatalf("hidden borderless STATIC creation failed: %v", err)
	}
	var pid uint32
	ownerThread, _, _ := dcm31RectWindowOwner.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	currentThread, _, _ := dcm31RectThreadID.Call()
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	style, _, _ := spGetWindowStyle.Call(hwnd, ^uintptr(15)) // GWL_STYLE
	const frameOrVisible = 0x00C40000 | WS_VISIBLE           // caption, thick frame, visible
	if pid != uint32(os.Getpid()) || ownerThread != currentThread || visible != 0 || style&frameOrVisible != 0 {
		dcm31RectDestroy.Call(hwnd)
		t.Fatalf("test HWND must be owned, hidden and borderless: pid=%d thread=%d/%d visible=%d style=%x", pid, ownerThread, currentThread, visible, style)
	}
	return hwnd
}

func dcm31RectDestroyWindow(t *testing.T, hwnd uintptr) {
	t.Helper()
	if ok, _, err := dcm31RectDestroy.Call(hwnd); ok == 0 {
		t.Errorf("owned test window destruction failed: %v", err)
	}
}

func dcm31RectPhysicalOracle(t *testing.T, hwnd uintptr) RECT {
	t.Helper()
	restore := dcm31RectEnterContext(t, ^uintptr(3))
	defer restore()
	var rc RECT
	if ok, _, err := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc))); ok == 0 {
		t.Fatalf("physical GetWindowRect oracle failed: %v", err)
	}
	if rc.Right <= rc.Left || rc.Bottom <= rc.Top {
		t.Fatalf("physical oracle has no area: %+v", rc)
	}
	return rc
}

func dcm31RectRequireNon100Scale(t *testing.T) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	restore := dcm31RectEnterContext(t, ^uintptr(0))
	defer restore()
	hwnd := dcm31RectHiddenWindow(t)
	defer dcm31RectDestroyWindow(t, hwnd)
	var logical RECT
	if ok, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&logical))); ok == 0 {
		t.Fatal("logical scale-witness rectangle unavailable")
	}
	physical := dcm31RectPhysicalOracle(t, hwnd)
	w, h := physical.Right-physical.Left, physical.Bottom-physical.Top
	if w == logical.Right-logical.Left && h == logical.Bottom-logical.Top {
		t.Skipf("DCM31 non-100%% scaling not observed on the owned test window's monitor: logical=%+v physical=%+v; this does not qualify DPI conversion", logical, physical)
	}
	t.Logf("DCM31 actual non-100%% scale witness: unaware client=%dx%d PMv2 borderless window=%dx%d", logical.Right-logical.Left, logical.Bottom-logical.Top, w, h)
}

func TestDCM31NativePreviewPhysicalRect(t *testing.T) {
	dcm31RectRequireContexts(t)
	dcm31RectRequireNon100Scale(t)
	for _, creator := range dcm31RectContexts {
		for _, caller := range dcm31RectContexts {
			t.Run(creator.name+"-window/"+caller.name+"-caller", func(t *testing.T) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				restoreCreator := dcm31RectEnterContext(t, creator.value)
				defer restoreCreator()
				hwnd := dcm31RectHiddenWindow(t)
				defer dcm31RectDestroyWindow(t, hwnd)
				windowContext, _, _ := dcm31RectGetWindowDPI.Call(hwnd)
				if !dcm31RectContextsEqual(windowContext, creator.value) {
					t.Fatal("owned window was not created in the requested DPI context")
				}
				physicalBefore := dcm31RectPhysicalOracle(t, hwnd)
				want := RECT{Right: physicalBefore.Right - physicalBefore.Left, Bottom: physicalBefore.Bottom - physicalBefore.Top}
				restoreCaller := dcm31RectEnterContext(t, caller.value)
				defer restoreCaller()
				callerBefore := dcm31RectCurrentContext(t)
				got, ok := nativePreviewPhysicalClientRect(hwnd)
				callerAfter := dcm31RectCurrentContext(t)
				windowAfter, _, _ := dcm31RectGetWindowDPI.Call(hwnd)
				if !dcm31RectContextsEqual(callerBefore, callerAfter) {
					t.Error("measurement changed the caller's DPI context")
				}
				if !dcm31RectContextsEqual(windowContext, windowAfter) {
					t.Error("measurement changed the owned window's DPI awareness")
				}
				if after := dcm31RectPhysicalOracle(t, hwnd); after != physicalBefore {
					t.Errorf("measurement changed window geometry: before=%+v after=%+v", physicalBefore, after)
				}
				if visible, _, _ := procIsWindowVisible.Call(hwnd); visible != 0 {
					t.Error("measurement made the private test window visible")
				}
				t.Logf("creator=%s caller=%s physical oracle=%+v measured=%+v ok=%t", creator.name, caller.name, want, got, ok)
				if !ok || got != want {
					t.Errorf("DCM31_NATIVE_PREVIEW_PHYSICAL_RECT: got=%+v ok=%t want physical client=%+v", got, ok, want)
				}
			})
		}
	}
}

func TestDCM31NativePreviewPhysicalRectInvalidHWND(t *testing.T) {
	dcm31RectRequireContexts(t)
	for _, caller := range dcm31RectContexts {
		t.Run(caller.name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			restore := dcm31RectEnterContext(t, caller.value)
			defer restore()
			hwnd := dcm31RectHiddenWindow(t)
			toDestroy := hwnd
			defer func() {
				if toDestroy != 0 {
					dcm31RectDestroyWindow(t, toDestroy)
				}
			}()
			dcm31RectDestroyWindow(t, hwnd)
			if live, _, _ := dcm31RectIsWindow.Call(hwnd); live != 0 {
				t.Fatal("destroyed test HWND is still live; cannot use it as an invalid-handle oracle")
			}
			toDestroy = 0
			for _, invalid := range []uintptr{0, hwnd} {
				before := dcm31RectCurrentContext(t)
				got, ok := nativePreviewPhysicalClientRect(invalid)
				after := dcm31RectCurrentContext(t)
				if ok || got != (RECT{}) {
					t.Errorf("invalid HWND %x returned a rectangle: %+v ok=%t", invalid, got, ok)
				}
				if !dcm31RectContextsEqual(before, after) {
					t.Errorf("invalid HWND %x changed caller DPI context", invalid)
				}
			}
		})
	}
}
