//go:build windows

package main

import (
	"runtime"
	"testing"
)

func TestControlUpdatePreviewTransitions(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	oldApp, oldView, oldPDF := app, sumatraView, pdfPreview
	defer func() { app, sumatraView, pdfPreview = oldApp, oldView, oldPDF }()
	parent := controlTestParent(t)
	defer user32.NewProc("DestroyWindow").Call(parent)
	app = appState{}
	pdfPreview.active = false
	buttons := []*uintptr{&app.zoomOutBtn, &app.zoom100Btn, &app.zoomInBtn, &app.zoomFitBtn, &app.zoomFitWidthBtn}
	for i, slot := range buttons {
		*slot = createControl("BUTTON", "Zoom", WS_CHILD|WS_VISIBLE, int32(i*50), 20, 45, 25, parent, 920+i)
		if *slot == 0 {
			t.Fatal("test button creation failed")
		}
	}
	for _, scene := range []struct {
		name                  string
		text, image, external bool
		enabled               [5]bool
	}{
		{"empty", false, false, false, [5]bool{}},
		{"text", true, false, false, [5]bool{true, true, true, true, false}},
		{"image", false, true, false, [5]bool{true, true, true, true, true}},
		{"sumatra", false, false, true, [5]bool{}},
		{"text-restored", true, false, false, [5]bool{true, true, true, true, false}},
		{"empty-restored", false, false, false, [5]bool{}},
	} {
		app.textPreviewActive = scene.text
		app.image = 0
		if scene.image {
			app.image = 1
		} // state only, never drawn or disposed
		sumatraView = nil
		if scene.external {
			sumatraView = &sumatraSession{}
		}
		updatePreviewZoomControls()
		for i, slot := range buttons {
			style, _, _ := spGetWindowStyle.Call(*slot, ^uintptr(15))
			if (style&WS_VISIBLE != 0) != !scene.external || (style&0x08000000 == 0) != scene.enabled[i] {
				t.Fatalf("%s button%d wrong style: %x", scene.name, i, style)
			}
		}
		// Repeating the update must preserve every state while parent is hidden.
		updatePreviewZoomControls()
	}
}

func TestControlUpdateMixedDPI(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	original, _, _ := spSetThreadDPIContext.Call(^uintptr(0))
	if original == 0 {
		t.Fatal("unaware context unavailable")
	}
	defer spSetThreadDPIContext.Call(original)
	parent := controlTestParent(t)
	defer user32.NewProc("DestroyWindow").Call(parent)
	h := controlTestCombo(t, parent, 941)
	moveLayoutCombo(h, 85, 13, 150, 240)
	saved, _, _ := spSetThreadDPIContext.Call(^uintptr(3))
	if saved == 0 {
		t.Fatal("per-monitor context unavailable")
	}
	updated := moveLayoutCombo(h, 85, 13, 150, 240)
	got, drop := controlTestRect(t, h, false), controlTestRect(t, h, true)
	procMoveWindow.Call(h, 85, 13, 150, 240, 0)
	matches := got == controlTestRect(t, h, false) && drop == controlTestRect(t, h, true)
	spSetThreadDPIContext.Call(saved)
	if !updated || !matches {
		t.Fatal("mixed DPI did not preserve original native behavior")
	}
	moveLayoutCombo(h, 85, 13, 150, 240)
	if moveLayoutCombo(h, 85, 13, 150, 240) {
		t.Fatal("restored DPI context did not observe live geometry")
	}
}
