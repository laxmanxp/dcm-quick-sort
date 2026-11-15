//go:build windows

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Exercises the actual local PPTX fallback presenter and native owned controls.
// The registered/failed handler states are boundary inputs, not altered Office
// registrations or evidence of a missing/failing handler installed on Lenovo.
func TestPPTXFallbackComponentRecovery(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("..", "tests", "fixtures", "dcm32-slides.pptx"))
	if err != nil {
		t.Fatal(err)
	}
	const pinned = "af42fff55297109195b95039d2ac39c5d53b1fae23b59e7b40a7184d679b93f1"
	if got := dcm34ComponentHash(t, filepath.Join("..", "tests", "fixtures", "dcm32-slides.pptx")); got != pinned {
		t.Fatalf("tracked PPTX fixture changed: %s", got)
	}
	for _, tc := range []struct {
		name       string
		registered bool
	}{
		{"missing-handler-boundary", false},
		{"failed-handler-boundary", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			valid := filepath.Join(root, "valid.pptx")
			invalid := filepath.Join(root, "corrupt.pptx")
			if err := os.WriteFile(valid, original, 0600); err != nil {
				t.Fatal(err)
			}
			invalidBytes := []byte("DCM34 synthetic non-Open-XML package; not a PowerPoint file.")
			if err := os.WriteFile(invalid, invalidBytes, 0600); err != nil {
				t.Fatal(err)
			}
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			oldApp := app
			app = appState{}
			defer func() { app = oldApp }()
			if !initGDIPlus() {
				t.Fatal("GDI+ unavailable for native PPTX component")
			}
			defer func() { procGdiplusShutdown.Call(app.gpToken) }()
			destroy := dcm34ComponentWindow(t, false)
			defer func() { hidePPTXPreview(); destroy() }()
			app.pptxSlideCombo = createControl("COMBOBOX", "", WS_CHILD|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL,
				6, 4, 700, 280, app.previewHwnd, ID_PPTX_SLIDE)
			if app.pptxSlideCombo == 0 {
				t.Fatal("owned PPTX slide control creation failed")
			}
			load := func(path string) {
				t.Helper()
				// Production loadPreview state-reset preconditions. The call
				// below is the production fallback routing/presentation path.
				hideTextPreview()
				hideXLSXPreview()
				hidePPTXPreview()
				resetPreviewZoomState()
				loadOfficeFallbackPreview(path, tc.registered)
			}
			assertBad := func(stage string) {
				t.Helper()
				load(invalid)
				want := "Preview unavailable: PPTX (no Windows preview handler installed)"
				body := "No preview available\n\nNo Windows preview handler is installed for PPTX files."
				if tc.registered {
					want = "Preview unavailable: PPTX (Windows handler failed)"
					body = "Preview unavailable\n\nWindows has a preview handler for PPTX files, but it could not preview this file."
				}
				if got := getText(app.statusLabel); got != want {
					t.Errorf("%s status: %q want %q", stage, got, want)
				}
				if app.pptxPreviewActive || app.textPreviewActive || app.xlsxPreviewActive ||
					app.previewActive || len(app.pptxDeck.Slides) != 0 || len(app.pptxImages) != 0 {
					t.Errorf("%s retained active/previous preview after corrupt PPTX", stage)
				}
				if got, _, _ := procSendMessageW.Call(app.pptxSlideCombo, 0x0146 /* CB_GETCOUNT */, 0, 0); got != 0 {
					t.Errorf("%s left %d stale PPTX slide options", stage, got)
				}
				if !strings.HasPrefix(app.fallback, body) ||
					!strings.Contains(app.fallback, "Name: corrupt.pptx") ||
					strings.Contains(app.fallback, "Bridge Inspection") {
					t.Errorf("%s misleading error body: %q", stage, app.fallback)
				}
			}
			assertGood := func(stage string) {
				t.Helper()
				load(valid)
				if got := getText(app.statusLabel); !strings.HasPrefix(got, "DCM PPTX Preview: Slide 1 of 2") {
					t.Errorf("%s slide-one status: %q", stage, got)
				}
				if !app.pptxPreviewActive || app.textPreviewActive || app.xlsxPreviewActive ||
					app.previewActive || app.fallback != "" || len(app.pptxDeck.Slides) != 2 ||
					app.pptxSlideIndex != 0 {
					t.Errorf("%s failed to enter real PPTX fallback state", stage)
					return
				}
				if got, _, _ := procSendMessageW.Call(app.pptxSlideCombo, 0x0146 /* CB_GETCOUNT */, 0, 0); got != 2 {
					t.Errorf("%s slide selector count %d, want 2", stage, got)
				}
				if app.pptxDeck.Slides[0].Title != "Bridge Inspection" ||
					app.pptxDeck.Slides[1].Title != "Payments" {
					t.Errorf("%s wrong slide content/order", stage)
				}
				showPPTXSlide(1)
				if app.pptxSlideIndex != 1 ||
					!strings.HasPrefix(getText(app.statusLabel), "DCM PPTX Preview: Slide 2 of 2") {
					t.Errorf("%s slide-two selection/display failed", stage)
				}
				showPPTXSlide(0)
				if app.pptxSlideIndex != 0 ||
					!strings.HasPrefix(getText(app.statusLabel), "DCM PPTX Preview: Slide 1 of 2") {
					t.Errorf("%s slide-one return failed", stage)
				}
			}
			assertBad("initial-error")
			assertGood("first-recovery")
			assertBad("error-after-success")
			assertGood("second-recovery")
			hidePPTXPreview()
			if app.pptxPreviewActive || len(app.pptxImages) != 0 {
				t.Error("PPTX image resources survived unload")
			}
			for _, file := range []struct{ path string; original []byte }{{valid, original}, {invalid, invalidBytes}} {
				data, err := os.ReadFile(file.path)
				if err != nil || !bytes.Equal(data, file.original) {
					t.Errorf("fixture was modified: %s: %v", filepath.Base(file.path), err)
				}
				if !dcm34ComponentExclusive(file.path) {
					t.Errorf("fixture still locked: %s", filepath.Base(file.path))
				}
			}
		})
	}
}
