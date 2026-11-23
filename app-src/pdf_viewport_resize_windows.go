//go:build windows

package main

// While the splitter is dragged, Windows sends many WM_SIZE messages. Never
// synchronously re-render in that callback: coalesce the final PDF viewport
// dimensions and let the existing generation-safe worker process the change.
const (
 pdfViewportResizeTimerID = uintptr(0xDCA4)
 pdfViewportResizeDelayMS = uintptr(160)
)
var pdfViewportResizeArmed bool

func cancelPDFViewportResize() {
 if pdfViewportResizeArmed && app.hwnd!=0 {
  nativeQueueKillTimer.Call(app.hwnd,pdfViewportResizeTimerID)
 }
 pdfViewportResizeArmed=false
}

func schedulePDFViewportResize() {
 if !pdfPreview.active || app.previewHwnd==0 || app.hwnd==0 {return}
 w,h:=pdfRenderViewport()
 if !pdfViewportNeedsRerender(pdfPreview.active,
  pdfPreview.renderViewportW,pdfPreview.renderViewportH,w,h) {
  cancelPDFViewportResize()
  return
 }
 // Rearming also handles a pending change when the drag is still in progress.
 nativeQueueKillTimer.Call(app.hwnd,pdfViewportResizeTimerID)
 timer,_,_:=nativeQueueSetTimer.Call(app.hwnd,pdfViewportResizeTimerID,pdfViewportResizeDelayMS,0)
 pdfViewportResizeArmed=timer!=0
 if !pdfViewportResizeArmed && !queueSplitterDragging {
  // If timers fail, still refresh once after an ordinary resize.
  refreshPDFViewportAfterResize()
 }
}

func refreshPDFViewportAfterResize() {
 cancelPDFViewportResize()
 if !pdfPreview.active || app.previewHwnd==0 || pdfShutdown.closing{return}
 w,h:=pdfRenderViewport()
 if !pdfViewportNeedsRerender(true,
  pdfPreview.renderViewportW,pdfPreview.renderViewportH,w,h) {return}
 // A fit-width or fit-page render must be recomputed at the new size.
 // Manual zoom keeps its selected zoom; the worker rescales accordingly.
 requestPDFPreviewRender()
}
