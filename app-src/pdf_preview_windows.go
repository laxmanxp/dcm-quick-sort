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
	"strconv"
	"syscall"
	"unsafe"
)

const (
	WM_PDF_RENDER_DONE = 0x8007 // WM_APP + 7
	ID_PDF_PREV        = 1060
	ID_PDF_NEXT        = 1061
	ID_PDF_ROTATE      = 1062 // clockwise, retains the original button ID
	ID_PDF_ROTATE_LEFT = 1063 // counterclockwise
	ID_PDF_PAGE_EDIT = 1073
	ID_PDF_PAGE_GO = 1074
	ID_PDF_FIT_WIDTH = 1075
	ID_PDF_FIT_PAGE = 1076
	ID_PDF_ZOOM_OUT = 1077
	ID_PDF_ZOOM_IN = 1078
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
	rotation         int // clockwise preview-only degrees, retained across pages
	generation       uint64
	workerGeneration uint64 // identity of the request that owns busy/pending
	busy             bool
	pending          bool
	prevBtn          uintptr
	nextBtn          uintptr
	rotateBtn        uintptr // clockwise icon
	rotateLeftBtn    uintptr // counterclockwise icon
	pageLabel        uintptr
	pageEdit, pageGo, fitWidthBtn, fitPageBtn, zoomOutBtn, zoomInBtn uintptr
	lastWidth        int
	lastHeight       int
	lastScalePercent int
	lastFallback     string
	cursorAnchor     PDFCursorAnchor
	panAnchored      bool
}

var pdfPreview pdfPreviewState
var pdfRenderResults = make(chan PDFRenderResult, 8)
// GUI-thread-only lifecycle state: a stopped selection can still own live workers.
var pdfShutdown pdfShutdownCoordinator

func pdfToolbarHandles() []uintptr {
 return []uintptr{pdfPreview.prevBtn,pdfPreview.pageEdit,pdfPreview.pageGo,pdfPreview.pageLabel,pdfPreview.nextBtn,
 pdfPreview.fitWidthBtn,pdfPreview.fitPageBtn,pdfPreview.rotateLeftBtn,pdfPreview.rotateBtn,pdfPreview.zoomOutBtn,pdfPreview.zoomInBtn}
}

func createPDFPreviewControls() {
 if app.previewHwnd==0{return}
 parent:=app.previewHwnd
 pdfPreview.prevBtn=createControl("BUTTON","<",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,4,4,30,26,parent,ID_PDF_PREV)
 pdfPreview.pageEdit=createControl("EDIT","1",WS_CHILD|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL,38,4,42,26,parent,ID_PDF_PAGE_EDIT)
 pdfPreview.pageGo=createControl("BUTTON","Go",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,84,4,32,26,parent,ID_PDF_PAGE_GO)
 pdfPreview.pageLabel=createControl("STATIC","/ --",WS_CHILD|SS_LEFT,120,9,48,20,parent,0)
 pdfPreview.nextBtn=createControl("BUTTON",">",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,172,4,30,26,parent,ID_PDF_NEXT)
 pdfPreview.fitWidthBtn=createControl("BUTTON","Fit W",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,206,4,48,26,parent,ID_PDF_FIT_WIDTH)
 pdfPreview.fitPageBtn=createControl("BUTTON","Fit P",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,258,4,48,26,parent,ID_PDF_FIT_PAGE)
 pdfPreview.rotateLeftBtn=createControl("BUTTON","Rotate left 90 degrees",WS_CHILD|WS_TABSTOP|BS_OWNERDRAW,310,4,32,26,parent,ID_PDF_ROTATE_LEFT)
 pdfPreview.rotateBtn=createControl("BUTTON","Rotate right 90 degrees",WS_CHILD|WS_TABSTOP|BS_OWNERDRAW,346,4,32,26,parent,ID_PDF_ROTATE)
 pdfPreview.zoomOutBtn=createControl("BUTTON","-",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,382,4,30,26,parent,ID_PDF_ZOOM_OUT)
 pdfPreview.zoomInBtn=createControl("BUTTON","+",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,416,4,30,26,parent,ID_PDF_ZOOM_IN)
 for _,h:=range pdfToolbarHandles(){setExplorerTheme(h)}
 hidePDFPreviewControls()
}

func hidePDFPreviewControls() {
 for _,h:=range pdfToolbarHandles(){if h!=0 {procShowWindow.Call(h,0)}}
}

func showPDFPreviewControls() {
 layoutPDFPreviewControls()
 updatePDFPreviewControls()
}

func layoutPDFPreviewControls() {
 if app.previewHwnd==0 || pdfPreview.pageLabel==0{return}
 var rc RECT
 procGetClientRect.Call(app.previewHwnd,uintptr(unsafe.Pointer(&rc)))
 width:=rc.Right-rc.Left
 // Layout order and clipping are calculated independently of HWNDs.
 // The compact plan keeps both rotation icons before optional controls.
 handles:=pdfToolbarHandles()
 for _,h:=range handles {if h!=0 {procShowWindow.Call(h,0)}}
 for _,slot:=range pdfToolbarPlan(width) {
  h:=handles[slot.Index]
  if h==0 {continue}
  top:=int32(4)
  height:=int32(26)
  if h==pdfPreview.pageLabel {top=9;height=20}
  procMoveWindow.Call(h,uintptr(slot.X),uintptr(top),uintptr(slot.Width),uintptr(height),1)
  if pdfPreview.active {procShowWindow.Call(h,SW_SHOW)}
 }
}

func updatePDFPreviewControls() {
 if !pdfPreview.active {hidePDFPreviewControls();return}
 pageText:="/ --"
 if pdfPreview.pageCount>0 {pageText=fmt.Sprintf("/ %d",pdfPreview.pageCount)}
 if pdfPreview.rotation!=0 {pageText+=fmt.Sprintf("  %d°",pdfPreview.rotation)}
 setText(pdfPreview.pageLabel,pageText)
 // Never overwrite an in-progress page-number edit.
 if focused,_,_:=procGetFocus.Call();focused!=pdfPreview.pageEdit {
  setText(pdfPreview.pageEdit,strconv.Itoa(pdfPreview.page+1))
 }
 ready:=pdfPreview.pageCount>0
 on:=func(ok bool)uintptr{if ok{return 1};return 0}
 procEnableWindow.Call(pdfPreview.prevBtn,on(ready&&pdfPreview.page>0))
 procEnableWindow.Call(pdfPreview.nextBtn,on(ready&&pdfPreview.page+1<pdfPreview.pageCount))
 procEnableWindow.Call(pdfPreview.pageEdit,on(ready))
 procEnableWindow.Call(pdfPreview.pageGo,on(ready))
 for _,h:=range []uintptr{pdfPreview.fitWidthBtn,pdfPreview.fitPageBtn,pdfPreview.rotateLeftBtn,pdfPreview.rotateBtn,pdfPreview.zoomOutBtn,pdfPreview.zoomInBtn}{
  procEnableWindow.Call(h,1)
 }
 layoutPDFPreviewControls()
}

func goToPDFPageNumber() {
 if !pdfPreview.active || pdfPreview.pageCount<1{return}
 number,err:=strconv.Atoi(strings.TrimSpace(getText(pdfPreview.pageEdit)))
 if err!=nil||number<1||number>pdfPreview.pageCount{
  setText(pdfPreview.pageEdit,strconv.Itoa(pdfPreview.page+1))
  setText(app.statusLabel,fmt.Sprintf("Enter a page between 1 and %d",pdfPreview.pageCount))
  return
 }
 if pdfPreview.page==number-1{return}
 pdfPreview.page=number-1
 app.previewPanX,app.previewPanY=0,0
 pdfPreview.cursorAnchor=PDFCursorAnchor{}
 pdfPreview.panAnchored=false
 updatePDFPreviewControls()
 requestPDFPreviewRender()
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

// capturePDFCursorAnchor runs on the GUI thread before wheel zoom changes
// generation. The renderer may finish later; generation keeps anchors scoped
// to the corresponding displayed document.
func capturePDFCursorAnchor(pointer POINT) {
 if !usingFlowPDFiumProvider() || !pdfPreview.active || app.previewHwnd==0 {return}
 var rc RECT
 procGetClientRect.Call(app.previewHwnd,uintptr(unsafe.Pointer(&rc)))
 anchor:=pdfCaptureCursorAnchor(pointer.X,pointer.Y,int32(pdfPreview.lastWidth),int32(pdfPreview.lastHeight),
  rc.Right-rc.Left,rc.Bottom-rc.Top,pdfToolbarHeight,app.previewPanX,app.previewPanY)
 if anchor.Valid {
  anchor.Generation=pdfPreview.generation+1
  pdfPreview.cursorAnchor=anchor
 } else {
  pdfPreview.cursorAnchor=PDFCursorAnchor{}
 }
}

func pdfRendererExecutable() (string, error) {
 if usingFlowPDFiumProvider() {
  return flowPDFiumWorkerPath()
 }
 return findPDFPowerShell()
}

func pdfPreviewRenderCommand(executable, script string, req PDFRenderRequest) *exec.Cmd {
 if usingFlowPDFiumProvider() {
  cmd:=exec.Command(executable, pdfiumRendererArgs(req)...)
  cmd.Env=developmentPDFiumWorkerEnvironment()
  return cmd
 }
 return exec.Command(executable, pdfRendererArgs(script, req)...)
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
	if pdfShutdown.closing || !isPDFPath(path) {
		return false
	}
	if _, err := pdfRendererExecutable(); err != nil {
		return false
	}
	if !usingFlowPDFiumProvider() {
		if _, err := ensurePDFRendererScript(); err != nil {
			return false
		}
	}
	full, err := filepath.Abs(path)
	if err != nil {
		full = filepath.Clean(path)
	}
	pdfPreview.active = true
	pdfPreview.path = full
	pdfPreview.page = 0
	pdfPreview.pageCount = 0
	pdfPreview.rotation = 0
	pdfPreview.lastWidth = 0
	pdfPreview.lastHeight = 0
	pdfPreview.lastScalePercent = 0
	pdfPreview.lastFallback = ""
	pdfPreview.cursorAnchor = PDFCursorAnchor{}
	pdfPreview.panAnchored = false
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

func tracePDFPreview(event string, req PDFRenderRequest, failed bool) {
	recordPDFTrace(PDFTraceEvent{Event: event, Generation: req.Generation, Rotation:req.Rotation, InputPath: req.InputPath, OutputPath: req.OutputPath, Failed: failed,
		State: &PDFTraceState{Generation: pdfPreview.generation, WorkerGeneration: pdfPreview.workerGeneration, Active: pdfPreview.active, Busy: pdfPreview.busy, Pending: pdfPreview.pending, Path: pdfPreview.path}})
}

func stopPDFPreview() {
	previous := PDFRenderRequest{Generation: pdfPreview.generation, InputPath: pdfPreview.path}
	if pdfPreview.active && pdfiumPersistentOptIn() {
		// Cancel the obsolete render without waiting for the worker's
		// request mutex. Generation protection keeps the next selection safe.
		if sharedFlowPDFiumProcess.cancelThrough(previous.Generation) {
			recordPDFTrace(PDFTraceEvent{Event:"worker-cancel-requested",
				Generation:previous.Generation,InputPath:previous.InputPath})
		}
	}
	pdfPreview.generation++ // makes any in-flight renderer result stale
	pdfPreview.active = false
	pdfPreview.path = ""
	pdfPreview.page = 0
	pdfPreview.pageCount = 0
	pdfPreview.rotation = 0
	pdfPreview.busy = false
	pdfPreview.workerGeneration = 0
	pdfPreview.pending = false
	pdfPreview.lastWidth = 0
	pdfPreview.lastHeight = 0
	pdfPreview.lastScalePercent = 0
	pdfPreview.cursorAnchor = PDFCursorAnchor{}
	pdfPreview.panAnchored = false
	hidePDFPreviewControls()
	tracePDFPreview("selection-stopped", previous, false)
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
	pdfPreview.cursorAnchor = PDFCursorAnchor{}
	pdfPreview.panAnchored = false
	app.previewPanX, app.previewPanY = 0, 0
	updatePDFPreviewControls()
	requestPDFPreviewRender()
}

// Rotation changes only the current preview and is retained when navigating
// pages. Opening another PDF resets it. Multiple clicks during a render use
// the existing generation-safe queue to apply only the latest orientation.
func rotatePDFPreviewBy(degrees int) {
	if !pdfPreview.active || pdfShutdown.closing {
		return
	}
	pdfPreview.rotation = normalizePDFPreviewRotation(pdfPreview.rotation + degrees)
	pdfPreview.cursorAnchor = PDFCursorAnchor{}
	pdfPreview.panAnchored = false
	app.previewPanX, app.previewPanY = 0, 0
	updatePDFPreviewControls()
	requestPDFPreviewRender()
}

func rotatePDFPreviewClockwise() { rotatePDFPreviewBy(90) }
func rotatePDFPreviewCounterclockwise() { rotatePDFPreviewBy(-90) }

func requestPDFPreviewRender() {
	if pdfShutdown.closing || !pdfPreview.active || strings.TrimSpace(pdfPreview.path) == "" {
		return
	}
	// Coalesce rapid Ctrl+wheel/button clicks. Only one PowerShell/WinRT render
	// runs at a time per selection; changes during that run become one latest
	// render. Workers from stopped selections can finish later but own no slot.
	pdfPreview.generation++
	if pdfPreview.busy {
		pdfPreview.pending = true
		setText(app.statusLabel, "PDF zoom/page update queued…")
		return
	}
	executable, err := pdfRendererExecutable()
	if err != nil {
		finishPDFRendererUnavailable(err)
		return
	}
	script := ""
	if !usingFlowPDFiumProvider() {
		script, err = ensurePDFRendererScript()
		if err != nil {
			finishPDFRendererUnavailable(err)
			return
		}
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
		Rotation:       pdfPreview.rotation,
		Fit:            app.previewFit,
		FitWidth:       app.previewFitWidth,
		Zoom:           app.previewZoom,
		ViewportWidth:  vw,
		ViewportHeight: vh,
		Generation:     generation,
	}
	pdfPreview.busy = true
	pdfPreview.workerGeneration = req.Generation
	mode := fmt.Sprintf("%d%%", clampPDFZoom(app.previewZoom))
	if app.previewFit {
		mode = "Fit"
	} else if app.previewFitWidth {
		mode = "Fit Width"
	}
	if req.Rotation != 0 {
		mode += fmt.Sprintf(" · %d°", req.Rotation)
	}
	setText(app.statusLabel, fmt.Sprintf("Rendering PDF page %d · %s…", pdfPreview.page+1, mode))
	updatePDFPreviewControls()

	// The GUI must remain alive until this goroutine's result is consumed.
	// No other thread mutates pdfShutdown.
	if !pdfShutdown.workerStarted() {
		_ = os.Remove(outputPath)
		pdfPreview.busy = false
		pdfPreview.workerGeneration = 0
		return
	}
	go func() {
		if pdfiumPersistentOptIn() {
			result := PDFRenderResult{Request:req}
			started := func(pid int) {
				recordPDFTrace(PDFTraceEvent{Event:"worker-started",Generation:req.Generation,
					InputPath:req.InputPath,OutputPath:req.OutputPath,RendererPID:pid})
			}
			meta,pid,err:=sharedFlowPDFiumProcess.render(executable,req,flowPDFiumRequestDeadline,started)
			recordPDFTrace(PDFTraceEvent{Event:"worker-finished",Generation:req.Generation,
				InputPath:req.InputPath,OutputPath:req.OutputPath,RendererPID:pid,Failed:err!=nil})
			if errors.Is(err,errFlowPDFiumCanceled) {
				recordPDFTrace(PDFTraceEvent{Event:"worker-canceled",
					Generation:req.Generation,InputPath:req.InputPath,OutputPath:req.OutputPath,
					RendererPID:pid})
			}
			if err!=nil && !errors.Is(err,errFlowPDFiumCanceled) && !strings.HasPrefix(err.Error(),"PDFium: ") {
				// Transport failures can be recovered by an isolated one-shot
				// invocation. Decoder refusals are final and never retried.
				tracePDFPreview("persistent-transport-fallback",req,true)
				_ = os.Remove(req.OutputPath)
				meta,pid,err=flowPDFiumOneShotFallback(executable,req)
				recordPDFTrace(PDFTraceEvent{Event:"persistent-fallback-finished",Generation:req.Generation,
					InputPath:req.InputPath,OutputPath:req.OutputPath,RendererPID:pid,Failed:err!=nil})
			}
			result.Meta,result.Err=meta,err
			if result.Err==nil {
				if st,statErr:=os.Stat(req.OutputPath);statErr!=nil||st.Size()==0 {
					if statErr!=nil {result.Err=statErr} else {result.Err=errors.New("PDFium service produced an empty image")}
				}
			}
			pdfRenderResults<-result
			procPostMessageW.Call(app.hwnd,WM_PDF_RENDER_DONE,0,0)
			return
		}
		cmd := pdfPreviewRenderCommand(executable, script, req)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
		var combined bytes.Buffer
		cmd.Stdout = &combined
		cmd.Stderr = &combined
		runErr := cmd.Start()
		childPID := 0
		if runErr == nil {
			childPID = cmd.Process.Pid
			recordPDFTrace(PDFTraceEvent{Event: "worker-started", Generation: req.Generation, InputPath: req.InputPath, OutputPath: req.OutputPath, RendererPID: childPID})
			runErr = cmd.Wait()
		}
		output := combined.Bytes()
		recordPDFTrace(PDFTraceEvent{Event: "worker-finished", Generation: req.Generation, InputPath: req.InputPath, OutputPath: req.OutputPath, RendererPID: childPID, Failed: runErr != nil})
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
			// Run after the stale-result cleanup (or successful presentation)
			// has finished. Only then may a deferred WM_CLOSE proceed.
			defer func() {
				if pdfShutdown.resultConsumed() {
					procPostMessageW.Call(app.hwnd, WM_CLOSE, 0, 0)
				}
			}()
			if result.Request.Generation != pdfPreview.generation || !pdfPreview.active || !strings.EqualFold(filepath.Clean(result.Request.InputPath), filepath.Clean(pdfPreview.path)) {
				_ = os.Remove(result.Request.OutputPath)
				// A stopped selection may finish after its replacement has started.
				// Only the completing worker can release its slot. A superseded
				// zoom/page request still owns that slot and may start its queued
				// successor; a prior selection must not touch the newer worker.
				if pdfCompletionOwnsWorker(pdfPreview.busy, pdfPreview.workerGeneration, result.Request.Generation) {
					pdfPreview.busy = false
					pdfPreview.workerGeneration = 0
					if pdfPreview.active && pdfPreview.pending {
						pdfPreview.pending = false
						requestPDFPreviewRender()
					}
				}
				tracePDFPreview("result-discarded", result.Request, result.Err != nil)
				return
			}
			pdfPreview.busy = false
			pdfPreview.workerGeneration = 0
			pdfPreview.pending = false
			if result.Err != nil {
				_ = os.Remove(result.Request.OutputPath)
				tracePDFPreview("result-unavailable", result.Request, true)
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
			// Only PDFium's real mouse-wheel zoom uses pointer anchoring.
			// Fit modes retain their established centered/top-aligned layout.
			if usingFlowPDFiumProvider() && pdfAnchorApplies(pdfPreview.cursorAnchor, result.Request.Generation, pdfPreview.generation) &&
				!app.previewFit && !app.previewFitWidth {
				var rc RECT
				procGetClientRect.Call(app.previewHwnd,uintptr(unsafe.Pointer(&rc)))
				app.previewPanX,app.previewPanY=pdfRestoreCursorPan(pdfPreview.cursorAnchor,
					int32(result.Meta.Width),int32(result.Meta.Height),
					rc.Right-rc.Left,rc.Bottom-rc.Top,pdfToolbarHeight)
				pdfPreview.panAnchored=true
			}
			pdfPreview.cursorAnchor = PDFCursorAnchor{}
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
			if pdfPreview.rotation != 0 {
				mode += fmt.Sprintf(" · %d°", pdfPreview.rotation)
			}
			setText(app.statusLabel, fmt.Sprintf("DCM PDF Preview · Page %d/%d · %s", pdfPreview.page+1, pdfPreview.pageCount, mode))
			tracePDFPreview("result-applied", result.Request, false)
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
