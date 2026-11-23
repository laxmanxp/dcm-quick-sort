//go:build windows

package main

import (
 "fmt"
 "unsafe"
)

const (
 imageToolbarHeight int32=34
 idImageRotateLeft=1080
 idImageRotateRight=1081
 idImageFit=1082
 idImageFitWidth=1083
 idImageZoomOut=1084
 idImageZoomIn=1085
 idImageZoomActual=1086
)

var imagePreviewControls struct{
 rotateLeft,rotateRight,fit,fitWidth,zoomOut,zoomIn,actual uintptr
 rotation int
}

var gdipImageRotateFlip=gdiplus.NewProc("GdipImageRotateFlip")

func imagePreviewActive()bool{return app.image!=0&&!pdfPreview.active&&!app.pptxPreviewActive}
func imageContentRect(rc RECT)RECT{
 if imagePreviewActive() {
  rc.Top+=imageToolbarHeight
  if rc.Top>rc.Bottom {rc.Top=rc.Bottom}
 }
 return rc
}
func imageToolbarHandles()[]uintptr{
 c:=imagePreviewControls
 return []uintptr{c.fit,c.fitWidth,c.zoomOut,c.actual,c.zoomIn,c.rotateLeft,c.rotateRight}
}
func createImagePreviewControls(){
 if app.previewHwnd==0{return}
 parent:=app.previewHwnd
 imagePreviewControls.fit=createControl("BUTTON","Fit",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,4,4,42,26,parent,idImageFit)
 imagePreviewControls.fitWidth=createControl("BUTTON","Fit W",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,50,4,50,26,parent,idImageFitWidth)
 imagePreviewControls.zoomOut=createControl("BUTTON","-",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,104,4,30,26,parent,idImageZoomOut)
 imagePreviewControls.actual=createControl("BUTTON","100%",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,138,4,50,26,parent,idImageZoomActual)
 imagePreviewControls.zoomIn=createControl("BUTTON","+",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,192,4,30,26,parent,idImageZoomIn)
 imagePreviewControls.rotateLeft=createControl("BUTTON","Rotate L",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,226,4,68,26,parent,idImageRotateLeft)
 imagePreviewControls.rotateRight=createControl("BUTTON","Rotate R",WS_CHILD|WS_TABSTOP|BS_PUSHBUTTON,298,4,68,26,parent,idImageRotateRight)
 for _,h:=range imageToolbarHandles(){setExplorerTheme(h)}
 hideImagePreviewControls()
}
func hideImagePreviewControls(){
 for _,h:=range imageToolbarHandles(){if h!=0 {procShowWindow.Call(h,0)}}
}
func layoutImagePreviewControls(){
 handles:=imageToolbarHandles()
 if app.previewHwnd==0{return}
 for _,h:=range handles{if h!=0 {procShowWindow.Call(h,0)}}
 if !imagePreviewActive(){return}
 var rc RECT
 procGetClientRect.Call(app.previewHwnd,uintptr(unsafe.Pointer(&rc)))
 widths:=[]int32{42,50,30,50,30,68,68}
 x:=int32(4)
 for i,h:=range handles {
  if h==0{continue}
  if x+widths[i]+4>rc.Right {break}
  procMoveWindow.Call(h,uintptr(x),4,uintptr(widths[i]),26,1)
  procShowWindow.Call(h,SW_SHOW)
  x+=widths[i]+4
 }
}
func showImagePreviewControls(){
 imagePreviewControls.rotation=0
 layoutImagePreviewControls()
}
func resetImagePreviewControls(){
 hideImagePreviewControls()
 imagePreviewControls.rotation=0
}
func rotateImagePreview(degrees int){
 if !imagePreviewActive(){return}
 mode:=uintptr(1) // GDI+ Rotate90FlipNone
 if degrees<0 {mode=3} // Rotate270FlipNone
 status,_,_:=gdipImageRotateFlip.Call(app.image,mode)
 if status!=0 {
  setText(app.statusLabel,fmt.Sprintf("Image rotation unavailable (GDI+ status %d)",status))
  return
 }
 imagePreviewControls.rotation=nextImagePreviewRotation(imagePreviewControls.rotation,degrees)
 app.previewPanX,app.previewPanY=0,0
 if app.previewFit {app.previewZoom=imageFitZoomPercent()}
 if app.previewFitWidth {app.previewZoom=imageFitWidthZoomPercent()}
 procInvalidateRect.Call(app.previewHwnd,0,1)
 setText(app.statusLabel,fmt.Sprintf("Image preview rotated %d° · %d%% · original unchanged",imagePreviewControls.rotation,app.previewZoom))
}

// Zoom around the pointer rather than recentering the whole image. This path
// is only for standalone raster images, never PDFium-owned page bitmaps.
func zoomImagePreviewAt(detents int, screenPoint POINT){
 if detents==0||!imagePreviewActive()||app.previewHwnd==0{return}
 point:=screenPoint
 if ok,_,_:=procScreenToClient.Call(app.previewHwnd,uintptr(unsafe.Pointer(&point)));ok==0 {adjustPreviewZoomInternal(detents);return}
 var rc RECT
 procGetClientRect.Call(app.previewHwnd,uintptr(unsafe.Pointer(&rc)))
 rc=imageContentRect(rc)
 var iw,ih uint32
 procGdipGetImageWidth.Call(app.image,uintptr(unsafe.Pointer(&iw)))
 procGdipGetImageHeight.Call(app.image,uintptr(unsafe.Pointer(&ih)))
 if iw==0||ih==0{return}
 viewW,viewH:=rc.Right-rc.Left,rc.Bottom-rc.Top
 if viewW<=0||viewH<=0{return}
 var oldW,oldH int32
 if app.previewFit {oldW,oldH=fitDimensions(int32(iw),int32(ih),viewW-8,viewH-8)
 }else if app.previewFitWidth {oldW,oldH=fitWidthDimensions(int32(iw),int32(ih),viewW-8)
 }else {oldW,oldH=zoomDimensions(int32(iw),int32(ih),app.previewZoom)}
 base:=app.previewZoom
 if app.previewFit {base=imageFitZoomPercent()}
 if app.previewFitWidth {base=imageFitWidthZoomPercent()}
 next:=nextPreviewZoom(base,detents)
 newW,newH:=zoomDimensions(int32(iw),int32(ih),next)
 x,y:=imageZoomAnchorPan(oldW,oldH,newW,newH,viewW,viewH,
  point.X-rc.Left,point.Y-rc.Top,app.previewPanX,app.previewPanY)
 app.previewFit=false
 app.previewFitWidth=false
 app.previewZoom=next
 app.previewPanX,app.previewPanY=x,y
 procInvalidateRect.Call(app.previewHwnd,0,1)
 setText(app.statusLabel,fmt.Sprintf("Image preview · %d%% · Ctrl+wheel to zoom at pointer",next))
}
