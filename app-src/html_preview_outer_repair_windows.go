//go:build windows

package main

import (
 "os"
 "runtime"
 "syscall"
 "unsafe"
)

type htmlPreviewWindowTree struct { pane, outer, inner excelLayoutWindow }

// Strictly locate the unique registered HTML preview child beneath our owned
// QuickSortPreview host. Never scan, alter, or reposition unrelated windows.
func htmlPreviewReadTree(pane uintptr) (htmlPreviewWindowTree, bool) {
 p, ok:=excelLayoutReadWindow(pane)
 if !ok || p.pid!=uint32(os.Getpid()) || p.class!="QuickSortPreview" {return htmlPreviewWindowTree{},false}
 outer,ok:=excelLayoutUniqueChild(pane,"##MimeEdit_Server")
 if !ok || outer.pid==p.pid || !excelLayoutContextsEqual(p.context,outer.context) {return htmlPreviewWindowTree{},false}
 inner,ok:=excelLayoutUniqueChild(outer.hwnd,"Internet Explorer_Server")
 if !ok || inner.pid!=outer.pid || !excelLayoutContextsEqual(p.context,inner.context) {return htmlPreviewWindowTree{},false}
 return htmlPreviewWindowTree{p,outer,inner},true
}

// The legacy Microsoft HTML native preview handler creates a foreign host
// which can be either 80% of, or larger than, the actual QuickSort pane.
// This operation requests resizing ONLY its unique outer ##MimeEdit_Server,
// and lets the HTML renderer resize its own Internet Explorer_Server child.
// It is called solely for an active HTML preview, after a successful SetRect.
func repairHTMLPreviewOuterWindow(pane uintptr) (accepted bool) {
 runtime.LockOSThread()
 defer runtime.UnlockOSThread()
 if pane==0 {return false}
 for _,proc:=range []*syscall.LazyProc{
  excelLayoutFindWindowEx,excelLayoutGetThreadDPI,excelLayoutSetThreadDPI,
  excelLayoutGetWindowDPI,excelLayoutEqualDPI,spGetPID,procGetParent,
  spGetWindowStyle,spGetClass,procGetClientRect,spSetWindowPos,
 } {if proc.Find()!=nil {return false}}
 original,_,_:=excelLayoutGetThreadDPI.Call()
 if original==0 {return false}
 tree,ok:=htmlPreviewReadTree(pane)
 if !ok {return false}
 previous,_,_:=excelLayoutSetThreadDPI.Call(tree.pane.context)
 if previous==0 {return false}
 defer func(){
  restored,_,_:=excelLayoutSetThreadDPI.Call(original)
  current,_,_:=excelLayoutGetThreadDPI.Call()
  if restored==0 || !excelLayoutContextsEqual(current,original) {accepted=false}
 }()
 current,_,_:=excelLayoutGetThreadDPI.Call()
 if !excelLayoutContextsEqual(previous,original) || !excelLayoutContextsEqual(current,tree.pane.context) {return false}
 var rc RECT
 read,_,_:=procGetClientRect.Call(pane,uintptr(unsafe.Pointer(&rc)))
 if read==0 || rc.Left!=0 || rc.Top!=0 || rc.Right<=0 || rc.Bottom<=0 {return false}
 now,ok:=htmlPreviewReadTree(pane)
 if !ok || !excelLayoutSameWindow(tree.pane,now.pane) ||
  !excelLayoutSameWindow(tree.outer,now.outer) || !excelLayoutSameWindow(tree.inner,now.inner) {return false}
 // SWP_ASYNCWINDOWPOS|SWP_NOACTIVATE|SWP_NOZORDER|SWP_NOOWNERZORDER
 // No subclassing, child destruction, reparenting, focus or input injection.
 const flags=0x4214
 result,_,_:=spSetWindowPos.Call(tree.outer.hwnd,0,0,0,uintptr(rc.Right),uintptr(rc.Bottom),flags)
 return result!=0
}
