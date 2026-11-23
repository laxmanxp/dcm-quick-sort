//go:build windows

package main

import (
 "bytes"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "syscall"
 "unsafe"
)

const pdfContinuousGap int32=14

type pdfCachedPage struct {
 image uintptr
 stream *iUnknown
 width,height,scale int
}

// Neighbor images belong to the GUI thread and to the active PDF's current
// rendering settings. Only the preceding and following pages are retained.
func clearPDFNeighborCache() {
 for _,p:=range pdfPreview.neighborCache {
  if p.image!=0 {procGdipDisposeImage.Call(p.image)}
  if p.stream!=nil {releaseUnknown(p.stream)}
 }
 pdfPreview.neighborCache=nil
 pdfPreview.neighborFailed=nil
 pdfPreview.pendingDirection=0
}

func disposeCachedPDFPage(p pdfCachedPage){
 if p.image!=0 {procGdipDisposeImage.Call(p.image)}
 if p.stream!=nil {releaseUnknown(p.stream)}
}

func cachePDFNeighborImage(path string, meta PDFRenderMetadata) error {
 oldImage,oldStream:=app.image,app.imageStream
 app.image=0
 app.imageStream=nil
 err:=loadImageWithError(path)
 if err!=nil {
  releasePreviewImage()
  app.image,app.imageStream=oldImage,oldStream
  return err
 }
 page:=pdfCachedPage{image:app.image,stream:app.imageStream,width:meta.Width,height:meta.Height,scale:meta.ScalePercent}
 app.image,app.imageStream=oldImage,oldStream
 if pdfPreview.neighborCache==nil {pdfPreview.neighborCache=make(map[int]pdfCachedPage)}
 if existing,ok:=pdfPreview.neighborCache[meta.PageIndex];ok {disposeCachedPDFPage(existing)}
 pdfPreview.neighborCache[meta.PageIndex]=page
 return nil
}

func prunePDFNeighborCache(){
 for page,p:=range pdfPreview.neighborCache {
  if page<pdfPreview.page-1 || page>pdfPreview.page+1 {
   disposeCachedPDFPage(p)
   delete(pdfPreview.neighborCache,page)
  }
 }
}

func promotePDFNeighborPage(target int, scroll int32) bool {
 if !pdfPreview.active || pdfPreview.busy || target<0 || target>=pdfPreview.pageCount {return false}
 next,ok:=pdfPreview.neighborCache[target]
 if !ok || next.image==0 {return false}
 oldIndex:=pdfPreview.page
 current:=pdfCachedPage{image:app.image,stream:app.imageStream,width:pdfPreview.lastWidth,height:pdfPreview.lastHeight,scale:pdfPreview.lastScalePercent}
 delete(pdfPreview.neighborCache,target)
 if current.image!=0 {
  if old,found:=pdfPreview.neighborCache[oldIndex];found {disposeCachedPDFPage(old)}
  pdfPreview.neighborCache[oldIndex]=current
 }
 app.image,app.imageStream=next.image,next.stream
 pdfPreview.page=target
 // Refresh the navigation field when it still contains the previous page.
 // Preserve any different number the user is actively entering.
 if pdfPreview.pageEdit!=0 && strings.TrimSpace(getText(pdfPreview.pageEdit))==fmt.Sprintf("%d",oldIndex+1) {
  setText(pdfPreview.pageEdit,fmt.Sprintf("%d",target+1))
 }
 pdfPreview.lastWidth,pdfPreview.lastHeight,pdfPreview.lastScalePercent=next.width,next.height,next.scale
 pdfPreview.scrollY=scroll
 pdfPreview.pendingDirection=0
 if scroll<0 {
  _,vh:=pdfRenderViewport()
  end:=int32(next.height-vh)
  if end<0 {end=0}
  pdfPreview.scrollY=end
 }
 prunePDFNeighborCache()
 updatePDFPreviewControls()
 setText(app.statusLabel,fmt.Sprintf("DCM PDF Preview · Page %d/%d · Continuous · %d%%",pdfPreview.page+1,pdfPreview.pageCount,app.previewZoom))
 procInvalidateRect.Call(app.previewHwnd,0,1)
 startPDFNeighborPrefetch()
 return true
}

// The main page render remains highest priority. We prefetch at most one
// neighbor at a time, using the same PDFium/WinRT worker contract and the
// same shutdown counter as ordinary page renders.
func startPDFNeighborPrefetch(){
 if !pdfPreview.active || pdfPreview.viewMode!=PDFViewContinuous ||
  pdfPreview.busy || pdfPreview.neighborBusy || pdfShutdown.closing || pdfPreview.pageCount<=1 {return}
 nextIndex:=-1
 for _,candidate:=range []int{pdfPreview.page+1,pdfPreview.page-1} {
  if candidate<0 || candidate>=pdfPreview.pageCount {continue}
  if _,ok:=pdfPreview.neighborCache[candidate];ok {continue}
  if pdfPreview.neighborFailed[candidate] {continue}
  nextIndex=candidate
  break
 }
 if nextIndex<0{return}
 executable,err:=pdfRendererExecutable()
 if err!=nil{return}
 script:=""
 if !usingFlowPDFiumProvider(){
  script,err=ensurePDFRendererScript()
  if err!=nil{return}
 }
 dir:=filepath.Join(settingsDirectory(),"preview-cache")
 if strings.TrimSpace(settingsDirectory())=="" {dir=filepath.Join(os.TempDir(),"DCM Quick Sort","preview-cache")}
 if err=os.MkdirAll(dir,0o755);err!=nil{return}
 file,err:=os.CreateTemp(dir,"pdf-neighbor-*.png")
 if err!=nil{return}
 output:=file.Name()
 _=file.Close()
 _=os.Remove(output)
 vw,vh:=pdfRenderViewport()
 req:=PDFRenderRequest{
  InputPath:pdfPreview.path,OutputPath:output,PageIndex:nextIndex,
  Rotation:pdfPreview.rotation,Fit:app.previewFit,FitWidth:app.previewFitWidth,
  Zoom:app.previewZoom,ViewportWidth:vw,ViewportHeight:vh,
  Generation:pdfPreview.generation,Neighbor:true,
 }
 if !pdfShutdown.workerStarted(){return}
 tracePDFPreview(fmt.Sprintf("neighbor-prefetch-request-page-%d",nextIndex+1),req,false)
 pdfPreview.neighborBusy=true
 pdfPreview.neighborRendering=nextIndex
 pdfPreview.neighborGeneration=req.Generation
 pdfPreview.neighborPath=req.InputPath
 go func(){
  result:=PDFRenderResult{Request:req}
  if pdfiumPersistentOptIn() {
   result.Meta,_,result.Err=sharedFlowPDFiumProcess.render(executable,req,flowPDFiumRequestDeadline,nil)
   if result.Err!=nil && !errors.Is(result.Err,errFlowPDFiumCanceled) && !strings.HasPrefix(result.Err.Error(),"PDFium: ") {
    _=os.Remove(req.OutputPath)
    result.Meta,_,result.Err=flowPDFiumOneShotFallback(executable,req)
   }
  }else{
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
   if st,e:=os.Stat(req.OutputPath);e!=nil {result.Err=e} else if st.Size()==0 {result.Err=errors.New("empty adjacent PDF render")}
  }
  pdfRenderResults<-result
  procPostMessageW.Call(app.hwnd,WM_PDF_RENDER_DONE,0,0)
 }()
}

func finishPDFNeighborRender(result PDFRenderResult){
 tracePDFPreview(fmt.Sprintf("neighbor-prefetch-result-page-%d",result.Request.PageIndex+1),result.Request,result.Err!=nil)
 if pdfPreview.neighborBusy && pdfPreview.neighborRendering==result.Request.PageIndex &&
  pdfPreview.neighborGeneration==result.Request.Generation &&
  strings.EqualFold(filepath.Clean(pdfPreview.neighborPath),filepath.Clean(result.Request.InputPath)) {
  pdfPreview.neighborBusy=false
 }
 defer os.Remove(result.Request.OutputPath)
 defer func(){if pdfPreview.active && !pdfPreview.neighborBusy {startPDFNeighborPrefetch()}}()
 if !pdfPreview.active || pdfPreview.viewMode!=PDFViewContinuous ||
  result.Request.Generation!=pdfPreview.generation ||
  !strings.EqualFold(filepath.Clean(result.Request.InputPath),filepath.Clean(pdfPreview.path)) {return}
 if result.Err!=nil || result.Meta.PageIndex!=result.Request.PageIndex || result.Meta.Width<=0 || result.Meta.Height<=0 {
  if pdfPreview.neighborFailed==nil {pdfPreview.neighborFailed=make(map[int]bool)}
  pdfPreview.neighborFailed[result.Request.PageIndex]=true
  if pdfPreview.pendingDirection!=0 && pdfPreview.page+pdfPreview.pendingDirection==result.Request.PageIndex {
   direction:=pdfPreview.pendingDirection
   pdfPreview.pendingDirection=0
   goToPDFPage(direction)
  }
  return
 }
 if err:=cachePDFNeighborImage(result.Request.OutputPath,result.Meta);err!=nil{
  if pdfPreview.neighborFailed==nil {pdfPreview.neighborFailed=make(map[int]bool)}
  pdfPreview.neighborFailed[result.Request.PageIndex]=true
  if pdfPreview.pendingDirection!=0 && pdfPreview.page+pdfPreview.pendingDirection==result.Request.PageIndex {
   direction:=pdfPreview.pendingDirection
   pdfPreview.pendingDirection=0
   goToPDFPage(direction)
  }
  return
 }
 if pdfPreview.pendingDirection!=0 {
  target:=pdfPreview.page+pdfPreview.pendingDirection
  if target==result.Meta.PageIndex {
   if promotePDFNeighborPage(target,map[bool]int32{true:-1,false:0}[pdfPreview.pendingDirection<0]) {return}
  }
 }
 prunePDFNeighborCache()
 procInvalidateRect.Call(app.previewHwnd,0,1)
 startPDFNeighborPrefetch()
}

// Continuous scrolling uses real, adjacent rendered pages rather than an
// oversized image of the entire PDF. Pages may have different heights.
func scrollPDFContinuousStack(delta int)bool{
 if !pdfPreview.active || pdfPreview.viewMode!=PDFViewContinuous{return false}
 if delta==0{return true}
 var rc RECT
 procGetClientRect.Call(app.previewHwnd,uintptr(unsafe.Pointer(&rc)))
 vh:=rc.Bottom-rc.Top-pdfToolbarHeight-8
 if vh<1 {vh=1}
 if pdfPreview.lastHeight<=0 {return true}
 move:=-previewPanPixels(delta,vh)
 target:=pdfPreview.scrollY+move
 tracePDFPreview(fmt.Sprintf("continuous-wheel-page-%d-pos-%d-to-%d-nextcached-%t-neighborbusy-%t",pdfPreview.page+1,pdfPreview.scrollY,target,pdfPreview.neighborCache[pdfPreview.page+1].image!=0,pdfPreview.neighborBusy),PDFRenderRequest{InputPath:pdfPreview.path,Generation:pdfPreview.generation},false)
 boundary:=int32(pdfPreview.lastHeight)+pdfContinuousGap
 if target>=boundary && pdfPreview.page+1<pdfPreview.pageCount {
  if promotePDFNeighborPage(pdfPreview.page+1,target-boundary) {return true}
  if pdfPreview.neighborFailed[pdfPreview.page+1] {goToPDFPage(1);return true}
  pdfPreview.pendingDirection=1
  if target>boundary-1 {target=boundary-1}
  startPDFNeighborPrefetch()
 }else if target<0 && pdfPreview.page>0 {
  prev,ok:=pdfPreview.neighborCache[pdfPreview.page-1]
  if ok && prev.image!=0 {
   if promotePDFNeighborPage(pdfPreview.page-1,int32(prev.height)+pdfContinuousGap+target){return true}
  }
  if pdfPreview.neighborFailed[pdfPreview.page-1] {goToPDFPage(-1);return true}
  pdfPreview.pendingDirection=-1
  target=0
  startPDFNeighborPrefetch()
 }
 if target<0 {target=0}
 if pdfPreview.page+1==pdfPreview.pageCount {
  bottom:=int32(pdfPreview.lastHeight)-vh
  if bottom<0 {bottom=0}
  if target>bottom {target=bottom}
 }
 pdfPreview.scrollY=target
 procInvalidateRect.Call(app.previewHwnd,0,1)
 return true
}

func drawPDFContinuousStack(hdc uintptr,rc RECT){
 if app.image==0{return}
 contentTop:=pdfToolbarHeight
 bottom:=rc.Bottom
 if bottom<=contentTop{return}
 var g uintptr
 if status,_,_:=procGdipCreateFromHDC.Call(hdc,uintptr(unsafe.Pointer(&g)));status!=0 || g==0{return}
 defer procGdipDeleteGraphics.Call(g)
 procGdipSetClipRectI.Call(g,uintptr(rc.Left),uintptr(contentTop),uintptr(rc.Right-rc.Left),uintptr(bottom-contentTop),0)
 draw:=func(image uintptr,width,height int,y int32){
  if image==0 || width<=0 || height<=0{return}
  x:=(rc.Right-int32(width))/2+app.previewPanX
  procGdipDrawImageRectI.Call(g,image,uintptr(x),uintptr(y),uintptr(width),uintptr(height))
 }
 y:=contentTop-pdfPreview.scrollY
 draw(app.image,pdfPreview.lastWidth,pdfPreview.lastHeight,y)
 if next,ok:=pdfPreview.neighborCache[pdfPreview.page+1];ok {
  draw(next.image,next.width,next.height,y+int32(pdfPreview.lastHeight)+pdfContinuousGap)
 }
 if prev,ok:=pdfPreview.neighborCache[pdfPreview.page-1];ok {
  draw(prev.image,prev.width,prev.height,y-int32(prev.height)-pdfContinuousGap)
 }
}
