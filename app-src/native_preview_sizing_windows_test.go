//go:build windows

package main

import (
	"runtime"
	"syscall"
	"testing"
)

func TestDCM34NativeSizingControls(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	oldApp, oldPDF, oldView := app, pdfPreview, sumatraView
	defer func() { app, pdfPreview, sumatraView = oldApp, oldPDF, oldView }()
	parent := controlTestParent(t)
	defer user32.NewProc("DestroyWindow").Call(parent)
	status := createControl("STATIC", "", WS_CHILD, 0, 90, 580, 30, parent, 959)
	pane := createControl("STATIC", "", WS_CHILD, 0, 130, 580, 300, parent, 960)
	if status == 0 || pane == 0 {
		t.Fatal("owned test controls unavailable")
	}
	app = appState{hwnd: parent, previewHwnd: pane, statusLabel: status}
	pdfPreview = pdfPreviewState{}
	sumatraView = nil
	buttons := []*uintptr{&app.zoomOutBtn, &app.zoom100Btn, &app.zoomInBtn, &app.zoomFitBtn, &app.zoomFitWidthBtn}
	for i, slot := range buttons {
		*slot = createControl("BUTTON", "Zoom", WS_CHILD|WS_VISIBLE, int32(i*70), 20, 65, 25, parent, 970+i)
		if *slot == 0 {
			t.Fatal("owned button creation failed")
		}
	}
	// These callbacks only count calls. SetFocus fails before any QueryFocus or
	// input injection, so a regressed guard cannot type into a foreign window.
	nativeCalls := 0
	vtbl := iPreviewHandlerVtbl{
		SetFocus: syscall.NewCallback(func(self uintptr) uintptr { nativeCalls++; return uintptr(0x80004005) }),
		SetRect:  syscall.NewCallback(func(self, rect uintptr) uintptr { nativeCalls++; return 0 }),
	}
	handler := &iPreviewHandler{Vtbl: &vtbl}
	for _, scene := range []struct {
		name                       string
		native, allowed, text, pdf bool
		enabled                    [5]bool
	}{
		{"blank", false, false, false, false, [5]bool{}},
		{"native-pdf-unknown-or-error", true, false, false, false, [5]bool{}},
		{"native-office", true, true, false, false, [5]bool{true, true, true, true, false}},
		{"text-recovery", false, false, true, false, [5]bool{true, true, true, true, false}},
		{"owned-pdf-recovery", false, false, false, true, [5]bool{true, true, true, true, true}},
		{"native-error-after-recovery", true, false, false, false, [5]bool{}},
		{"empty-after-error", false, false, false, false, [5]bool{}},
	} {
		t.Logf("native sizing state: %s", scene.name)
		// Keep SendMessage/SetWindowText on the thread that owns these HWNDs.
		// t.Run would move this work into a separate goroutine.
		func() {
			app.preview = nil
			if scene.native {
				app.preview = handler
			}
			app.previewHostSizingAllowed = scene.allowed
			app.textPreviewActive = scene.text
			pdfPreview.active = scene.pdf
			for repeat := 0; repeat < 2; repeat++ {
				updatePreviewZoomControls()
				for i, slot := range buttons {
					style, _, _ := spGetWindowStyle.Call(*slot, ^uintptr(15))
					if style&WS_VISIBLE == 0 || (style&0x08000000 == 0) != scene.enabled[i] {
						t.Fatalf("button %d wrong style %x; want enabled %v", i, style, scene.enabled[i])
					}
				}
			}
			if !scene.native || scene.allowed {
				return
			}
			setText(status, "DCM PDF renderer unavailable; using Windows PDF preview fallback")
			footer := getText(status)
			app.previewZoom = 173
			app.previewFit = false
			app.previewFitWidth = false
			app.previewPanX = 11
			app.previewPanY = 22
			app.previewZoomWheelRemainder = 17
			beforeCalls := nativeCalls
			setPreviewZoom100()
			fitPreview()
			fitPreviewWidth()
			adjustPreviewZoom(1)
			adjustPreviewZoom(-1)
			if sendPreviewCtrlKey(VK_0, false) {
				t.Error("native key injection allowed without sizing permission")
			}
			autoFitPreviewIfSupported("test.pdf", "{3D3B1846-CC43-42AE-BFF9-D914083C2BA3}")
			for _, id := range []int{ID_ZOOM_OUT, ID_ZOOM_100, ID_ZOOM_IN, ID_ZOOM_FIT, ID_ZOOM_FIT_WIDTH} {
				mainWindowProc(parent, WM_COMMAND, uintptr(id), 0)
			}
			if nativeCalls != beforeCalls {
				t.Errorf("disabled sizing made %d native calls", nativeCalls-beforeCalls)
			}
			if got := getText(status); got != footer {
				t.Errorf("error footer overwritten: %q", got)
			}
			if app.previewZoom != 173 || app.previewFit || app.previewFitWidth || app.previewPanX != 11 || app.previewPanY != 22 || app.previewZoomWheelRemainder != 17 {
				t.Error("disabled direct/command sizing changed viewport state")
			}
		}()
	}
}
