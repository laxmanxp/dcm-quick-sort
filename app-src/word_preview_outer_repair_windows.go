//go:build windows

package main

import (
 "os"
 "runtime"
 "unsafe"
)

const wordOuterRepairTimerID = uintptr(0xDCA3)
const wordOuterRepairDelayMS = uintptr(150)

var wordOuterRepairPending bool
var wordOuterRepairDone bool
var wordOuterRepairPane uintptr
var wordOuterRepairHandler uintptr
var wordOuterRepairHost excelLayoutWindow

func cancelWordOuterRepair() {
 if app.hwnd!=0 {spKillTimer.Call(app.hwnd,wordOuterRepairTimerID)}
 wordOuterRepairPending=false
 wordOuterRepairDone=false
 wordOuterRepairPane=0
 wordOuterRepairHandler=0
 wordOuterRepairHost=excelLayoutWindow{}
}

// Only the outer window of the currently active, registered native Word
// handler may be nudged. Do not touch Word's document or scrollbar HWNDs.
func readWordOuterHost(pane uintptr) (excelLayoutWindow,bool) {
 p,ok:=excelLayoutReadWindow(pane)
 if !ok || p.class!="QuickSortPreview" || p.pid!=uint32(os.Getpid()) {return excelLayoutWindow{},false}
 w,ok:=excelLayoutUniqueChild(pane,"_WwB")
 if !ok || w.pid==p.pid || !excelLayoutContextsEqual(p.context,w.context) {return excelLayoutWindow{},false}
 return w,true
}

func beginWordOuterRepair(pane,handler uintptr) bool {
 if pane==0 || handler==0 || app.hwnd==0 || wordOuterRepairDone || wordOuterRepairPending {return false}
 runtime.LockOSThread()
 defer runtime.UnlockOSThread()
 original,_,_:=excelLayoutGetThreadDPI.Call()
 if original==0 {return false}
 host,ok:=readWordOuterHost(pane)
 if !ok {return false}
 previous,_,_:=excelLayoutSetThreadDPI.Call(host.context)
 if previous==0 {return false}
 defer excelLayoutSetThreadDPI.Call(original)
 if !excelLayoutContextsEqual(previous,original) {return false}
 var rc,outer RECT
 if ok,_,_:=procGetClientRect.Call(pane,uintptr(unsafe.Pointer(&rc)));ok==0 || rc.Right<200 || rc.Bottom<100 {return false}
 if ok,_,_:=procGetClientRect.Call(host.hwnd,uintptr(unsafe.Pointer(&outer)));ok==0 || outer.Right!=rc.Right || outer.Bottom!=rc.Bottom {return false}
 // Cross-process HWND request; SWP_ASYNCWINDOWPOS|NOACTIVATE|NOZORDER|NOOWNERZORDER
 const flags=0x4214
 accepted,_,_:=spSetWindowPos.Call(host.hwnd,0,0,0,uintptr(rc.Right-1),uintptr(rc.Bottom-1),flags)
 if accepted==0 {return false}
 wordOuterRepairPending=true
 wordOuterRepairDone=true
 wordOuterRepairPane=pane
 wordOuterRepairHandler=handler
 wordOuterRepairHost=host
 timer,_,_:=spSetTimer.Call(app.hwnd,wordOuterRepairTimerID,wordOuterRepairDelayMS,0)
 if timer==0 {
  // Best effort immediate restoration; never leave the outer child shrunken.
  spSetWindowPos.Call(host.hwnd,0,0,0,uintptr(rc.Right),uintptr(rc.Bottom),flags)
  wordOuterRepairPending=false
  return false
 }
 return true
}

func finishWordOuterRepair() {
 spKillTimer.Call(app.hwnd,wordOuterRepairTimerID)
 if !wordOuterRepairPending {return}
 wordOuterRepairPending=false
 pane,handler,host:=wordOuterRepairPane,wordOuterRepairHandler,wordOuterRepairHost
 if app.preview==nil || uintptr(unsafe.Pointer(app.preview))!=handler || app.previewHwnd!=pane {return}
 runtime.LockOSThread()
 defer runtime.UnlockOSThread()
 original,_,_:=excelLayoutGetThreadDPI.Call()
 if original==0 {return}
 previous,_,_:=excelLayoutSetThreadDPI.Call(host.context)
 if previous==0 {return}
 defer excelLayoutSetThreadDPI.Call(original)
 now,ok:=readWordOuterHost(pane)
 if !ok || !excelLayoutSameWindow(now,host) {return}
 var rc RECT
 if read,_,_:=procGetClientRect.Call(pane,uintptr(unsafe.Pointer(&rc)));read==0 || rc.Right<200 || rc.Bottom<100 {return}
 const flags=0x4214
 spSetWindowPos.Call(host.hwnd,0,0,0,uintptr(rc.Right),uintptr(rc.Bottom),flags)
}
