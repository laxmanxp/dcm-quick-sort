//go:build windows

package main

import "unsafe"

var (
 imageCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
 imageCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
 imageSelectObject = gdi32.NewProc("SelectObject")
 imageDeleteDC = gdi32.NewProc("DeleteDC")
 imageBitBlt = gdi32.NewProc("BitBlt")
)

// Raster images are painted offscreen so a wheel/drag frame is never shown as
// an empty white pane before GDI+ finishes drawing. The toolbar is drawn into
// the same bitmap; child toolbar controls remain Windows-owned and are not
// captured or modified. No bitmap/DC is retained across window resizes.
func paintImagePreviewBuffered(hdc uintptr, bounds RECT) bool {
 width,height:=bounds.Right-bounds.Left,bounds.Bottom-bounds.Top
 if !imagePreviewActive() || hdc==0 || width<=0 || height<=0 {return false}
 memory,_,_:=imageCreateCompatibleDC.Call(hdc)
 if memory==0 {return false}
 defer imageDeleteDC.Call(memory)
 bitmap,_,_:=imageCreateCompatibleBitmap.Call(hdc,uintptr(width),uintptr(height))
 if bitmap==0 {return false}
 defer procDeleteObject.Call(bitmap)
 previous,_,_:=imageSelectObject.Call(memory,bitmap)
 if previous==0 || previous==^uintptr(0) {return false}
 defer imageSelectObject.Call(memory,previous)
 white,_,_:=procGetStockObject.Call(WHITE_BRUSH)
 procFillRect.Call(memory,uintptr(unsafe.Pointer(&bounds)),white)
 bar:=RECT{Left:bounds.Left,Top:bounds.Top,Right:bounds.Right,Bottom:bounds.Top+imageToolbarHeight}
 if bar.Bottom>bounds.Bottom {bar.Bottom=bounds.Bottom}
 face,_,_:=procGetSysColorBrush.Call(15)
 procFillRect.Call(memory,uintptr(unsafe.Pointer(&bar)),face)
 drawImagePreview(memory,bounds)
 const srccopy=0x00CC0020
 ok,_,_:=imageBitBlt.Call(hdc,uintptr(bounds.Left),uintptr(bounds.Top),
  uintptr(width),uintptr(height),memory,0,0,srccopy)
 return ok!=0
}
