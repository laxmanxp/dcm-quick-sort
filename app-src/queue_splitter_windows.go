//go:build windows

package main

import "unsafe"

// toggleSplitterGuide is a reversible XOR outline drawn in the free gap.
// The ListView and embedded preview are not resized during WM_MOUSEMOVE.
// It keeps pointer movement responsive even with costly Office providers.
func toggleSplitterGuide(hwnd uintptr, width int32) {
 var bounds RECT
 procGetClientRect.Call(hwnd,uintptr(unsafe.Pointer(&bounds)))
 w,h:=bounds.Right-bounds.Left,bounds.Bottom-bounds.Top
 if w<760 {w=760}
 if h<650 {h=650}
 headerY:=compactFilterLayout(w).HeaderY
 top:=headerY+28
 height:=h-369-(headerY-120)
 if height<245 {height=245}
 left:=int32(20)+queueSplitterClamp(w,width)+4
 rect:=RECT{Left:left,Top:top+4,Right:left+2,Bottom:top+height-4}
 hdc,_,_:=procGetDC.Call(hwnd)
 if hdc==0 {return}
 procDrawFocusRect.Call(hdc,uintptr(unsafe.Pointer(&rect)))
 procReleaseDC.Call(hwnd,hdc)
}
