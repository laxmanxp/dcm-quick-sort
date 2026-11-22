//go:build windows

package main

import "unsafe"

// layoutSplitterDrag moves only controls affected by the pane boundary.
// In particular it avoids full layoutUI, column recalculation, tab visibility
// changes and Office COM calls on every mouse event.
func layoutSplitterDrag(hwnd uintptr) {
 var rc RECT
 procGetClientRect.Call(hwnd,uintptr(unsafe.Pointer(&rc)))
 w,h:=rc.Right-rc.Left,rc.Bottom-rc.Top
 if w<760 {w=760}
 if h<650 {h=650}
 headerY:=compactFilterLayout(w).HeaderY
 top:=headerY+28
 height:=h-369-(headerY-120)
 if height<245 {height=245}
 listW:=queueSplitterPosition(w)
 previewX:=int32(30)+listW
 previewW:=w-20-previewX
 // Move both active and inactive queue variants so switching tabs later
 // keeps exactly the same geometry.
 procMoveWindow.Call(app.queueTabControl,20,uintptr(headerY-3),uintptr(listW),27,0)
 procMoveWindow.Call(app.countLabel,184,uintptr(headerY),uintptr(listW-164),24,0)
 nameW,extW,createdW,updatedW:=queueColumnWidths(listW)
 procMoveWindow.Call(app.queueNameHeader,20,uintptr(top),uintptr(nameW),25,0)
 procMoveWindow.Call(app.queueExtHeader,uintptr(20+nameW),uintptr(top),uintptr(extW),25,0)
 procMoveWindow.Call(app.queueCreatedHeader,uintptr(20+nameW+extW),uintptr(top),uintptr(createdW),25,0)
 procMoveWindow.Call(app.queueUpdatedHeader,uintptr(20+nameW+extW+createdW),uintptr(top),uintptr(updatedW),25,0)
 procMoveWindow.Call(app.fileList,20,uintptr(top+27),uintptr(listW),uintptr(height-27),0)
 procMoveWindow.Call(app.nativeFileList,20,uintptr(top),uintptr(listW),uintptr(height),0)
 procMoveWindow.Call(app.processedSearchEdit,20,uintptr(top),uintptr(listW),27,0)
 procMoveWindow.Call(app.processedList,20,uintptr(top+33),uintptr(listW),uintptr(height-73),0)
 procMoveWindow.Call(app.undoBatchBtn,20,uintptr(top+height-34),95,30,0)
 procMoveWindow.Call(app.undoBtn,123,uintptr(top+height-34),74,30,0)
 procMoveWindow.Call(app.previewHwnd,uintptr(previewX),uintptr(top),uintptr(previewW),uintptr(height),0)
 // Keep preview controls on the resized pane; avoid changing focus.
 procMoveWindow.Call(app.currentLabel,uintptr(previewX),uintptr(headerY),uintptr(maxInt32(90,previewW-410)),24,0)
 zoomOutW,zoom100W,zoomInW,zoomFitW,zoomFitWidthW:=int32(34),int32(56),int32(34),int32(44),int32(54)
 zoomX:=previewX+previewW-(zoomOutW+zoom100W+zoomInW+zoomFitW+zoomFitWidthW+16)
 procMoveWindow.Call(app.readTextCheck,uintptr(zoomX-113),uintptr(headerY-4),105,27,0)
 for _,control:=range []struct{hwnd uintptr;width int32}{
  {app.zoomOutBtn,zoomOutW},{app.zoom100Btn,zoom100W},{app.zoomInBtn,zoomInW},
  {app.zoomFitBtn,zoomFitW},{app.zoomFitWidthBtn,zoomFitWidthW},
 } {
  procMoveWindow.Call(control.hwnd,uintptr(zoomX),uintptr(headerY-4),uintptr(control.width),27,0)
  zoomX+=control.width+4
 }
 procInvalidateRect.Call(hwnd,0,0)
 procInvalidateRect.Call(app.previewHwnd,0,0)
}

func maxInt32(a,b int32) int32 {if a>b{return a};return b}
