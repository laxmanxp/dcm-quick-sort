//go:build windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// A completion from a stopped selection must clean up only its own output.
// In particular it cannot mark a newer worker idle, including after reopening
// the SAME file. Run on the HWND owner thread; never inject foreign-window input.
func TestDCM34PDFLateResultOwnership(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	savedApp, savedPDF, savedResults := app, pdfPreview, pdfRenderResults
	defer func() { app, pdfPreview, pdfRenderResults = savedApp, savedPDF, savedResults }()
	parent := controlTestParent(t)
	defer user32.NewProc("DestroyWindow").Call(parent)
	status := createControl("STATIC", "New selected PDF is still rendering", WS_CHILD, 0, 0, 550, 30, parent, 991)
	if status == 0 {
		t.Fatal("owned status control unavailable")
	}
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.pdf")
	for _, scene := range []struct {
		name             string
		samePath, failed bool
	}{
		{"different-file/late-success", false, false},
		{"different-file/late-error", false, true},
		{"same-file-reopened/late-success", true, false},
		{"same-file-reopened/late-error", true, true},
	} {
		app = appState{statusLabel: status, fallback: "Rendering the NEW selection", previewZoom: 137, previewPanX: 17, previewPanY: 29}
		pdfRenderResults = make(chan PDFRenderResult, 8)
		pdfPreview = pdfPreviewState{active: true, path: oldPath, generation: 41, busy: true}
		oldGeneration := pdfPreview.generation
		stopPDFPreview() // actual production selection invalidation
		nextPath := filepath.Join(dir, "new.pdf")
		if scene.samePath {
			nextPath = oldPath
		}
		pdfPreview.active = true
		pdfPreview.path = nextPath
		pdfPreview.generation++
		pdfPreview.busy = true // newer asynchronous request is outstanding
		beforePDF := pdfPreview
		beforeApp := app
		footer := getText(status)
		output := filepath.Join(dir, "late.png")
		if err := os.WriteFile(output, []byte("obsolete output must be removed, never decoded"), 0600); err != nil {
			t.Fatal(err)
		}
		result := PDFRenderResult{Request: PDFRenderRequest{InputPath: oldPath, OutputPath: output, Generation: oldGeneration}, Meta: PDFRenderMetadata{PageCount: 99, PageIndex: 7, Width: 400, Height: 600, ScalePercent: 42}}
		if scene.failed {
			result.Err = errors.New("obsolete renderer failed")
		}
		pdfRenderResults <- result
		finishPDFRender()
		if !reflect.DeepEqual(pdfPreview, beforePDF) {
			t.Errorf("%s: stale completion changed NEW worker state: before=%+v after=%+v", scene.name, beforePDF, pdfPreview)
		}
		if !reflect.DeepEqual(app, beforeApp) || getText(status) != footer {
			t.Errorf("%s: stale completion changed new preview or footer", scene.name)
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Errorf("%s: obsolete output not removed: %v", scene.name, err)
		}
		if len(pdfRenderResults) != 0 {
			t.Errorf("%s: completion not consumed", scene.name)
		}
		t.Logf("%s: old generation=%d new=%d busy=%v output removed", scene.name, oldGeneration, pdfPreview.generation, pdfPreview.busy)
	}
}

func TestDCM34PDFLateResultAfterTextOrEmpty(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	savedApp, savedPDF, savedResults := app, pdfPreview, pdfRenderResults
	defer func() { app, pdfPreview, pdfRenderResults = savedApp, savedPDF, savedResults }()
	for _, text := range []bool{false, true} {
		app = appState{textPreviewActive: text, fallback: "Current selection", previewZoom: 100}
		pdfRenderResults = make(chan PDFRenderResult, 8)
		pdfPreview = pdfPreviewState{active: true, path: "old.pdf", generation: 80, busy: true}
		stopPDFPreview()
		before := pdfPreview
		beforeApp := app
		pdfRenderResults <- PDFRenderResult{Request: PDFRenderRequest{InputPath: "old.pdf", Generation: 80}, Err: errors.New("late failure")}
		finishPDFRender()
		if !reflect.DeepEqual(before, pdfPreview) || !reflect.DeepEqual(beforeApp, app) {
			t.Fatalf("text=%v: stopped completion changed empty/text state", text)
		}
	}
}
