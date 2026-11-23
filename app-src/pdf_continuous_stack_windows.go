//go:build windows

package main

import (
 "bytes"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "syscall"
 "unsafe"
)

// Continuous PDF mode keeps at most the previous and next page in memory.
// The active page remains owned by app.image and app.imageStream.
const pdfContinuousGap int32 = 14
type pdfCachedPage struct {
 image uintptr
 stream *iUnknown
 width, height int
 scale int
}
func (p pdfCachedPage) dispose() {
 if p.image!=0 {procGdipDisposeImage.Call(p.image)}
 if p.stream!=nil {releaseUnknown(p.stream)}
}
func clearPDFNeighborCache() {
 for _,p:=range pdfPreview.neighborCache {p.dispose()}
 pdfPreview.neighborCache=nil
 pdfPreview.neighborFailed=nil
 pdfPreview.neighborGeneration=0
 // Do not clear neighborBusy here: the running worker still owns a shutdown
 // ticket and must complete before another neighbor worker may start.
}
func loadPDFCachedBitmap(path string)(pdfCachedPage,error){
 data,err:=os.ReadFile(path)
 if err!=nil{return pdfCachedPage{},err}
 if len(data)==0{return pdfCachedPage{},errors.New("empty PDF page image")}
 global,_,_:=procGlobalAlloc.Call(GMEM_MOVEABLE,uintptr(len(data)))
 if global==0{return pdfCachedPage{},errors.New("could not allocate page bitmap")}
 mem,_,_:=procGlobalLock.Call(global)
 if mem==0 {procGlobalFree.Call(global);return pdfCachedPage{},errors.New("could not lock page bitmap")}
 copy(unsafe.Slice((*byte)(unsafe.Pointer(mem)),len(data)),data)
 procGlobalUnlock.Call(global)
 var stream *iUnknown
 hr,_,_:=procCreateStreamOnHGlobal.Call(global,1,uintptr(unsafe.Pointer(&stream)))
 if hresultFailed(hr)||stream==nil {
  procGlobalFree.Call(global)
  return pdfCachedPage{},fmt.Errorf("page image stream HRESULT %#x",hr)
 }
 var image uintptr
 status,_,_:=procGdipLoadImageStream.Call(uintptr(unsafe.Pointer(stream)),uintptr(unsafe.Pointer(&image)))
 if status!=0||image==0 {
  if image!=0 {procGdipDisposeImage.Call(image)}
  releaseUnknown(stream)
  return pdfCachedPage{},fmt.Errorf("page image decode status %d",status)
 }
 var width,height uint32
 procGdipGetImageWidth.Call(image,uintptr(unsafe.Pointer(&width)))
 procGdipGetImageHeight.Call(image,uintptr(unsafe.Pointer(&height)))
 return pdfCachedPage{image:image,stream:stream,width:int(width),height:int(height)},nil
}

func startPDFNeighborPrefetch() {
 if !pdfPreview.active || pdfPreview.viewMode!=PDFViewContinuous ||
  pdfPreview.busy || pdfPreview.neighborBusy || pdfShutdown.closing ||
  pdfPreview.pageCount<2 || pdfPreview.lastHeight<=0{return}
 if pdfPreview.neighborCache==nil {pdfPreview.neighborCache=make(map[int]pdfCachedPage)}
 if pdfPreview.neighborFailed==nil {pdfPreview.neighborFailed=make(map[int]bool)}
 target:=-1
 for _,p:=range []int{pdfPreview.page+1,pdfPreview.page-1}{
  if p<0||p>=pdfPreview.pageCount{continue}
  if _,ok:=pdfPreview.neighborCache[p];!ok&&!pdfPreview.neighborFailed[p] {target=p;break}
 }
 if target<0{return}
 executable,err:=pdfRendererExecutable()
 if err!=nil {pdfPreview.neighborFailed[target]=true;return}
 script:=""
 if !usingFlowPDFiumProvider(){
  script,err=ensurePDFRendererScript()
  if err!=nil {pdfPreview.neighborFailed[target]=true;return}
 }
 dir:=filepath.Join(settingsDirectory(),"preview-cache")
 if strings.TrimSpace(settingsDirectory())=="" {dir=filepath.Join(os.TempDir(),"DCM Quick Sort","preview-cache")}
 if err=os.MkdirAll(dir,0755);err!=nil {pdfPreview.neighborFailed[target]=true;return}
 f,err:=os.CreateTemp(dir,"pdf-neighbor-*.png")
 if err!=nil {pdfPreview.neighborFailed[target]=true;return}
 out:=f.Name()
 f.Close()
 os.Remove(out)
 vw,vh:=pdfRenderViewport()
 req:=PDFRenderRequest{InputPath:pdfPreview.path,OutputPath:out,PageIndex:target,
  Rotation:pdfPreview.rotation,Fit:app.previewFit,FitWidth:app.previewFitWidth,
  Zoom:app.previewZoom,ViewportWidth:vw,ViewportHeight:vh,
  Generation:pdfPreview.generation,Neighbor:true}
 if !pdfShutdown.workerStarted(){return}
 pdfPreview.neighborBusy=true
 pdfPreview.neighborRendering=target
 pdfPreview.neighborGeneration=req.Generation
 pdfPreview.neighborPath=req.InputPath
 go func(){
  result:=PDFRenderResult{Request:req}
  if pdfiumPersistentOptIn(){
   result.Meta,_,result.Err=sharedFlowPDFiumProcess.render(executable,req,flowPDFiumRequestDeadline,nil)
   if result.Err!=nil && !errors.Is(result.Err,errFlowPDFiumCanceled) &&
    !strings.HasPrefix(result.Err.Error(),"PDFium: "){
    os.Remove(req.OutputPath)
    result.Meta,_,result.Err=flowPDFiumOneShotFallback(executable,req)
   }
  } else {
   cmd:=pdfPreviewRenderCommand(executable,script,req)
   cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:createNoWindow}
   var output bytes.Buffer
   cmd.Stdout=&output
   cmd.Stderr=&output
   if err:=cmd.Run();err!=nil {result.Err=fmt.Errorf("%w: %s",err,strings.TrimSpace(output.String()))} else {
    result.Meta,result.Err=parsePDFRendererOutput(output.String())
   }
  }
  if result.Err==nil {
   if stat,err:=os.Stat(out);err!=nil {result.Err=err} else if stat.Size()==0 {
    result.Err=errors.New("empty adjacent-page render")
   }
  }
  pdfRenderResults<-result
  procPostMessageW.Call(app.hwnd,WM_PDF_RENDER_DONE,0,0)
 }()
}
func finishPDFNeighborRender(result PDFRenderResult){
 // A neighbor result cannot change the current page or replace app.image.
 valid:=pdfPreview.active&&pdfPreview.viewMode==PDFViewContinuous &&
  strings.EqualFold(filepath.Clean(result.Request.InputPath),filepath.Clean(pdfPreview.path)) &&
  result.Request.Generation==pdfPreview.generation
 page:=result.Request.PageIndex
 pdfPreview.neighborBusy=false
 pdfPreview.neighborGeneration=0
 if valid && result.Err==nil {
  cached,err:=loadPDFCachedBitmap(result.Request.OutputPath)
  if err==nil {
   cached.scale=result.Meta.ScalePercent
   if pdfPreview.neighborCache==nil {pdfPreview.neighborCache=make(map[int]pdfCachedPage)}
   if old,ok:=pdfPreview.neighborCache[page];ok {old.dispose()}
   pdfPreview.neighborCache[page]=cached
   procInvalidateRect.Call(app.previewHwnd,0,1)
  } else {
   if pdfPreview.neighborFailed==nil {pdfPreview.neighborFailed=make(map[int]bool)}
   pdfPreview.neighborFailed[page]=true
  }
 } else if valid {
  if pdfPreview.neighborFailed==nil {pdfPreview.neighborFailed=make(map[int]bool)}
  pdfPreview.neighborFailed[page]=true
 }
 os.Remove(result.Request.OutputPath)
 if valid {startPDFNeighborPrefetch()}
}

// Promote a prefetched bitmap without waiting on a renderer process.
// A previous-page copy remains cached for reverse scrolling.
func promotePDFNeighborPage(next int, position int32)bool{
 cached,ok:=pdfPreview.neighborCache[next]
 if !ok || cached.image==0 {return false}
 oldPage:=pdfPreview.page
 old:=pdfCachedPage{image:app.image,stream:app.imageStream,
  width:pdfPreview.lastWidth,height:pdfPreview.lastHeight,scale:pdfPreview.lastScalePercent}
 delete(pdfPreview.neighborCache,next)
 if previous,has:=pdfPreview.neighborCache[oldPage];has {previous.dispose()}
 pdfPreview.neighborCache[oldPage]=old
 app.image=cached.image
 app.imageStream=cached.stream
 pdfPreview.page=next
 pdfPreview.lastWidth=cached.width
 pdfPreview.lastHeight=cached.height
 pdfPreview.lastScalePercent=cached.scale
 pdfPreview.scrollY=position
 pdfPreview.cursorAnchor=PDFCursorAnchor{}
 pdfPreview.panAnchored=false
 updatePDFPreviewControls()
 procInvalidateRect.Call(app.previewHwnd,0,1)
 startPDFNeighborPrefetch()
 return true
}

func scrollPDFContinuousStack(delta int)bool{
 if !pdfPreview.active || pdfPreview.viewMode!=PDFViewContinuous {return false}
 if delta==0{return true}
 _,vh:=pdfRenderViewport()
 viewport:=int32(vh)
 if viewport<=0{return true}
 height:=int32(pdfPreview.lastHeight)
 if height<=0{return true}
 // Wheel up makes the content move down; wheel down makes it move up.
 step:=previewPanPixels(delta,viewport)
 target:=pdfPreview.scrollY-step
 if target<0 {
  prev:=pdfPreview.page-1
  if prev>=0 {
   if cached,ok:=pdfPreview.neighborCache[prev];ok {
    if promotePDFNeighborPage(prev,int32(cached.height)+pdfContinuousGap+target){return true}
   }
   pdfPreview.pendingDirection=-1
   goToPDFPage(-1)
   return true
  }
  target=0
 }
 maxScroll:=height-viewport
 if maxScroll<0 {maxScroll=0}
 if pdfPreview.page+1<pdfPreview.pageCount{
  maxScroll=height+pdfContinuousGap
  if target>=maxScroll {
   next:=pdfPreview.page+1
   if promotePDFNeighborPage(next,target-maxScroll){return true}
   pdfPreview.pendingDirection=1
   goToPDFPage(1)
   return true
  }
 }
 if target>maxScroll{target=maxScroll}
 pdfPreview.scrollY=target
 procInvalidateRect.Call(app.previewHwnd,0,1)
 startPDFNeighborPrefetch()
 return true
}

func drawPDFContinuousStack(hdc uintptr,window RECT){
 if app.image==0{return}
 area:=pdfContentRect(window)
 if area.Bottom<=area.Top{return}
 var g uintptr
 if status,_,_:=procGdipCreateFromHDC.Call(hdc,uintptr(unsafe.Pointer(&g)));status!=0||g==0{return}
 defer procGdipDeleteGraphics.Call(g)
 procGdipSetClipRectI.Call(g,uintptr(area.Left),uintptr(area.Top),
  uintptr(area.Right-area.Left),uintptr(area.Bottom-area.Top),0)
 width:=int32(pdfPreview.lastWidth)
 height:=int32(pdfPreview.lastHeight)
 if width<=0||height<=0{return}
 x:=area.Left+(area.Right-area.Left-width)/2
 y:=area.Top+4-pdfPreview.scrollY
 // Draw the previous page if it intersects the current viewport.
 if prev,ok:=pdfPreview.neighborCache[pdfPreview.page-1];ok&&prev.image!=0{
  prevX:=area.Left+(area.Right-area.Left-int32(prev.width))/2
  prevY:=y-int32(prev.height)-pdfContinuousGap
  if prevY+int32(prev.height)>=area.Top {
   procGdipDrawImageRectI.Call(g,prev.image,uintptr(prevX),uintptr(prevY),
    uintptr(prev.width),uintptr(prev.height))
  }
 }
 if y+height>=area.Top&&y<=area.Bottom{
  procGdipDrawImageRectI.Call(g,app.image,uintptr(x),uintptr(y),uintptr(width),uintptr(height))
 }
 if next,ok:=pdfPreview.neighborCache[pdfPreview.page+1];ok&&next.image!=0{
  nextX:=area.Left+(area.Right-area.Left-int32(next.width))/2
  nextY:=y+height+pdfContinuousGap
  if nextY<=area.Bottom {
   procGdipDrawImageRectI.Call(g,next.image,uintptr(nextX),uintptr(nextY),
    uintptr(next.width),uintptr(next.height))
  }
 }
}
