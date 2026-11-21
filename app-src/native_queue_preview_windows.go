//go:build windows

package main

import "strings"

// Native ListView selection events arrive in bursts during rapid keyboard
// navigation. Keep filename/selection UI immediate, but only start the
// expensive preview after the active row settles briefly.
const (
 nativeQueuePreviewTimerID = 0xDCA1
 nativeQueuePreviewDelayMS = 90
)

var (
 nativeQueueSetTimer = user32.NewProc("SetTimer")
 nativeQueueKillTimer = user32.NewProc("KillTimer")
)

func cancelNativeQueuePreview() {
 if app.nativePreviewTimerArmed && app.hwnd!=0 {
  nativeQueueKillTimer.Call(app.hwnd,nativeQueuePreviewTimerID)
 }
 app.nativePreviewTimerArmed=false
 app.nativePreviewPendingPath=""
}

func scheduleNativeQueuePreview(path string) {
 cancelNativeQueuePreview()
 if app.hwnd==0 || strings.TrimSpace(path)=="" {return}
 app.nativePreviewPendingPath=path
 timer,_,_:=nativeQueueSetTimer.Call(app.hwnd,nativeQueuePreviewTimerID,nativeQueuePreviewDelayMS,0)
 if timer==0 {
  // No preview should be lost merely because a Windows timer is unavailable.
  app.nativePreviewPendingPath=""
  unloadPreview()
  loadPreview(path)
  return
 }
 app.nativePreviewTimerArmed=true
}

func flushNativeQueuePreview() {
 if !app.nativePreviewTimerArmed {return}
 path:=app.nativePreviewPendingPath
 cancelNativeQueuePreview()
 if path=="" || currentFile()!=path {return}
 unloadPreview()
 loadPreview(path)
}
