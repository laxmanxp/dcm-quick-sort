//go:build windows

package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const (
	WM_PDF_RENDER_DONE = 0x8007 // WM_APP + 7
	ID_PDF_PREV        = 1060
	ID_PDF_NEXT        = 1061
	pdfToolbarHeight   = int32(34)
	createNoWindow     = 0x08000000
)

//go:embed assets/pdf_render.ps1
var bundledPDFRendererScript []byte

type pdfPreviewState struct {
	active           bool
	path             string
	page             int
	pageCount        int
	generation       uint64
	busy             bool
	pending          bool
	prevBtn          uintptr
	nextBtn          uintptr
	pageLabel        uintptr
	lastWidth        int
	lastHeight       int
	lastScalePercent int
	lastFallback     string
}

var pdfPreview pdfPreviewState
var pdfRenderResults = make(chan PDFRenderResult, 8)

func createPDFPreviewControls() {
	if app.previewHwnd == 0 {
		return
	}
	pdfPreview.prevBtn = createControl("BUTTON", "‹", WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON, 6, 4, 34, 26, app.previewHwnd, ID_PDF_PREV)
	pdfPreview.pageLabel = createControl("STATIC", "Page", WS_CHILD|SS_LEFT, 46, 9, 130, 20, app.previewHwnd, 0)
	pdfPreview.nextBtn = createControl("BUTTON", "›", WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON, 182, 4, 34, 26, app.previewHwnd, ID_PDF_NEXT)
	setExplorerTheme(pdfPreview.prevBtn)
	setExplorerTheme(pdfPreview.nextBtn)
	hidePDFPreviewControls()
}

func hidePDFPreviewControls() {
	for _, hwnd := range []uintptr{pdfPreview.prevBtn, pdfPreview.pageLabel, pdfPreview.nextBtn} {
		if hwnd != 0 {
			procShowWindow.Call(hwnd, 0)
		}
	}
}

func showPDFPreviewControls() {
	for _, hwnd := range []uintptr{pdfPreview.prevBtn, pdfPreview.pageLabel, pdfPreview.nextBtn} {
		if hwnd != 0 {
			procShowWindow.Call(hwnd, SW_SHOW)
		}
	}
	layoutPDFPreviewControls()
	updatePDFPreviewControls()
}

func layoutPDFPreviewControls() {
	if app.previewHwnd == 0 || pdfPreview.pageLabel == 0 {
		return
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	labelW := int32(150)
	center := (rc.Right - rc.Left) / 2
	procMoveWindow.Call(pdfPreview.prevBtn, uintptr(center-labelW/2-42), 4, 34, 26, 1)
	procMoveWindow.Call(pdfPreview.pageLabel, uintptr(center-labelW/2), 9, uintptr(labelW), 20, 1)
	procMoveWindow.Call(pdfPreview.nextBtn, uintptr(center+labelW/2+8), 4, 34, 26, 1)
}

func updatePDFPreviewControls() {
	if !pdfPreview.active {
		hidePDFPreviewControls()
		return
	}
	label := "Rendering PDF…"
	if pdfPreview.pageCount > 0 {
		label = fmt.Sprintf("Page %d of %d", pdfPreview.page+1, pdfPreview.pageCount)
	}
	setText(pdfPreview.pageLabel, label)
	prevEnabled := uintptr(0)
	nextEnabled := uintptr(0)
	if pdfPreview.page > 0 {
		prevEnabled = 1
	}
	if pdfPreview.pageCount > 0 && pdfPreview.page+1 < pdfPreview.pageCount {
		nextEnabled = 1
	}
	procEnableWindow.Call(pdfPreview.prevBtn, prevEnabled)
	procEnableWindow.Call(pdfPreview.nextBtn, nextEnabled)
}

func pdfContentRect(rc RECT) RECT {
	if pdfPreview.active {
		rc.Top += pdfToolbarHeight
		if rc.Top > rc.Bottom {
			rc.Top = rc.Bottom
		}
	}
	return rc
}

func pdfRendererRuntimePath() string {
	base := settingsDirectory()
	if strings.TrimSpace(base) == "" {
		base = filepath.Join(os.TempDir(), "DCM Quick Sort")
	}
	return filepath.Join(base, "runtime", "pdf_render.ps1")
}

func ensurePDFRendererScript() (string, error) {
	path := pdfRendererRuntimePath()
	if data, err := os.ReadFile(path); err == nil && bytes.Equal(data, bundledPDFRendererScript) {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".new"
	if err := os.WriteFile(tmp, bundledPDFRendererScript, 0o644); err != nil {
		return "", err
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return path, nil
}

func findPDFPowerShell() (string, error) {
	for _, name := range []string{"powershell.exe", "pwsh.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("Windows PowerShell was not found")
}

func pdfRenderViewport() (int, int) {
	if app.previewHwnd == 0 {
		return 900, 700
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	w := int(rc.Right - rc.Left - 12)
	h := int(rc.Bottom - rc.Top - pdfToolbarHeight - 12)
	if w < 64 {
		w = 64
	}
	if h < 64 {
		h = 64
	}
	return w, h
}

func startPDFPreview(path string) bool {
	if !isPDFPath(path) {
		return false
	}
	if _, err := findPDFPowerShell(); err != nil {
		return false
	}
	if _, err := ensurePDFRendererScript(); err != nil {
		return false
	}
	full, err := filepath.Abs(path)
	if err != nil {
		full = filepath.Clean(path)
	}
	pdfPreview.active = true
	pdfPreview.path = full
	pdfPreview.page = 0
	pdfPreview.pageCount = 0
	pdfPreview.lastWidth = 0
	pdfPreview.lastHeight = 0
	pdfPreview.lastScalePercent = 0
	pdfPreview.lastFallback = ""
	defaultView := normalizePDFDefaultView(app.pdfDefaultView)
	app.previewFitWidth = defaultView == pdfDefaultViewFitWidth
	app.previewFit = !app.previewFitWidth
	app.previewZoom = 100
	app.previewPanX, app.previewPanY = 0, 0
	app.fallback = "Rendering PDF…"
	showPDFPreviewControls()
	updatePreviewZoomControls()
	requestPDFPreviewRender()
	procInvalidateRect.Call(app.previewHwnd, 0, 1)
	return true
}

func stopPDFPreview() {
	pdfPreview.generation++ // makes any in-flight renderer result stale
	pdfPreview.active = false
	pdfPreview.path = ""
	pdfPreview.page = 0
	pdfPreview.pageCount = 0
	pdfPreview.busy = false
	pdfPreview.pending = false
	pdfPreview.lastWidth = 0
	pdfPreview.lastHeight = 0
	pdfPreview.lastScalePercent = 0
	hidePDFPreviewControls()
}

func goToPDFPage(delta int) {
	if !pdfPreview.active || delta == 0 || pdfPreview.pageCount <= 0 {
		return
	}
	next := pdfPreview.page + delta
	if next < 0 || next >= pdfPreview.pageCount || next == pdfPreview.page {
		return
	}
	pdfPreview.page = next
	app.previewPanX, app.previewPanY = 0, 0
	updatePDFPreviewControls()
	requestPDFPreviewRender()
}

func requestPDFPreviewRender() {
	if !pdfPreview.active || strings.TrimSpace(pdfPreview.path) == "" {
		return
	}
	// Coalesce rapid Ctrl+wheel/button clicks. Only one PowerShell/WinRT render
	// runs at a time; changes made while it is running become one latest render.
	pdfPreview.generation++
	if pdfPreview.busy {
		pdfPreview.pending = true
		setText(app.statusLabel, "PDF zoom/page update queued…")
		return
	}
	powershell, err := findPDFPowerShell()
	if err != nil {
		finishPDFRendererUnavailable(err)
		return
	}
	script, err := ensurePDFRendererScript()
	if err != nil {
		finishPDFRendererUnavailable(err)
		return
	}
	dir := filepath.Join(settingsDirectory(), "preview-cache")
	if strings.TrimSpace(settingsDirectory()) == "" {
		dir = filepath.Join(os.TempDir(), "DCM Quick Sort", "preview-cache")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		finishPDFRendererUnavailable(err)
		return
	}
	out, err := os.CreateTemp(dir, "pdf-page-*.png")
	if err != nil {
		finishPDFRendererUnavailable(err)
		return
	}
	outputPath := out.Name()
	_ = out.Close()
	_ = os.Remove(outputPath)

	generation := pdfPreview.generation
	vw, vh := pdfRenderViewport()
	req := PDFRenderRequest{
		InputPath:      pdfPreview.path,
		OutputPath:     outputPath,
		PageIndex:      pdfPreview.page,
		Fit:            app.previewFit,
		FitWidth:       app.previewFitWidth,
		Zoom:           app.previewZoom,
		ViewportWidth:  vw,
		ViewportHeight: vh,
		Generation:     generation,
	}
	pdfPreview.busy = true
	mode := fmt.Sprintf("%d%%", clampPDFZoom(app.previewZoom))
	if app.previewFit {
		mode = "Fit"
	} else if app.previewFitWidth {
		mode = "Fit Width"
	}
	setText(app.statusLabel, fmt.Sprintf("Rendering PDF page %d · %s…", pdfPreview.page+1, mode))
	updatePDFPreviewControls()

	go func() {
		cmd := exec.Command(powershell, pdfRendererArgs(script, req)...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
		output, runErr := cmd.CombinedOutput()
		result := PDFRenderResult{Request: req}
		if runErr != nil {
			result.Err = fmt.Errorf("%w: %s", runErr, strings.TrimSpace(string(output)))
		} else {
			result.Meta, result.Err = parsePDFRendererOutput(string(output))
			if result.Err == nil {
				if st, statErr := os.Stat(outputPath); statErr != nil || st.Size() == 0 {
					if statErr != nil {
						result.Err = statErr
					} else {
						result.Err = errors.New("PDF renderer produced an empty image")
					}
				}
			}
		}
		pdfRenderResults <- result
		procPostMessageW.Call(app.hwnd, WM_PDF_RENDER_DONE, 0, 0)
	}()
}

func finishPDFRender() {
	for {
		select {
		case result := <-pdfRenderResults:
			if result.Request.Generation != pdfPreview.generation || !pdfPreview.active || !strings.EqualFold(filepath.Clean(result.Request.InputPath), filepath.Clean(pdfPreview.path)) {
				_ = os.Remove(result.Request.OutputPath)
				pdfPreview.busy = false
				if pdfPreview.active && pdfPreview.pending {
					pdfPreview.pending = false
					requestPDFPreviewRender()
				}
				return
			}
			pdfPreview.busy = false
			pdfPreview.pending = false
			if result.Err != nil {
				_ = os.Remove(result.Request.OutputPath)
				finishPDFRendererUnavailable(result.Err)
				return
			}
			releasePreviewImage()
			if !loadImage(result.Request.OutputPath) {
				_ = os.Remove(result.Request.OutputPath)
				finishPDFRendererUnavailable(errors.New("rendered PDF page could not be decoded"))
				return
			}
			_ = os.Remove(result.Request.OutputPath)
			pdfPreview.pageCount = result.Meta.PageCount
			pdfPreview.page = result.Meta.PageIndex
			pdfPreview.lastWidth = result.Meta.Width
			pdfPreview.lastHeight = result.Meta.Height
			if result.Meta.ScalePercent > 0 {
				pdfPreview.lastScalePercent = result.Meta.ScalePercent
				app.previewZoom = result.Meta.ScalePercent
			}
			app.fallback = ""
			if app.previewFit {
				app.previewPanX, app.previewPanY = 0, 0
			} else if app.previewFitWidth {
				// Fit Width is a reading mode: enter at the top of the page.
				// The drawing model is centered, so top alignment is represented
				// by the maximum positive vertical pan offset.
				var rc RECT
				procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
				rc = pdfContentRect(rc)
				availH := rc.Bottom - rc.Top - 8
				app.previewPanX = 0
				app.previewPanY = previewTopPanOffset(int32(result.Meta.Height), availH)
			}
			showPDFPreviewControls()
			updatePreviewZoomControls()
			procInvalidateRect.Call(app.previewHwnd, 0, 1)
			mode := fmt.Sprintf("%d%%", clampPDFZoom(app.previewZoom))
			if app.previewFit {
				mode = fmt.Sprintf("Fit · %d%%", app.previewZoom)
			} else if app.previewFitWidth {
				mode = fmt.Sprintf("Fit Width · %d%%", app.previewZoom)
			}
			setText(app.statusLabel, fmt.Sprintf("DCM PDF Preview · Page %d/%d · %s", pdfPreview.page+1, pdfPreview.pageCount, mode))
			return
		default:
			return
		}
	}
}

func finishPDFRendererUnavailable(err error) {
	if !pdfPreview.active {
		return
	}
	path := pdfPreview.path
	pdfPreview.lastFallback = err.Error()
	stopPDFPreview()
	releasePreviewImage()
	if loadWindowsPreviewHandler(path) {
		app.previewActive = true
		setText(app.statusLabel, "DCM PDF renderer unavailable; using Windows PDF preview fallback")
		return
	}
	app.fallback = "PDF preview unavailable\n\n" + err.Error()
	setText(app.statusLabel, "PDF preview unavailable: "+err.Error())
	updatePreviewZoomControls()
	procInvalidateRect.Call(app.previewHwnd, 0, 1)
}

func releasePreviewImage() {
	if app.image != 0 {
		procGdipDisposeImage.Call(app.image)
		app.image = 0
	}
	if app.imageStream != nil {
		releaseUnknown(app.imageStream)
		app.imageStream = nil
	}
}
