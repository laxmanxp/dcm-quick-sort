package main

import (
 "math"
 "testing"
 "fmt"
)

func TestPDFCursorAnchorKeepsDocumentPointUnderPointer(t *testing.T) {
 // Image 800x1000, viewport 600x700, PDF toolbar 34px.
 oldW,oldH:=int32(800),int32(1000)
 newW,newH:=int32(1200),int32(1500)
 cursorX,cursorY:=int32(250),int32(290)
 anchor:=pdfCaptureCursorAnchor(cursorX,cursorY,oldW,oldH,600,700,34,0,0)
 if !anchor.Valid{t.Fatal("pointer expected over PDF page")}
 panX,panY:=pdfRestoreCursorPan(anchor,newW,newH,600,700,34)
 beforeX:=float64(600-oldW)/2+anchor.FractionX*float64(oldW)
 beforeY:=34+float64(700-34-oldH)/2+anchor.FractionY*float64(oldH)
 afterX:=float64(600-newW)/2+float64(panX)+anchor.FractionX*float64(newW)
 afterY:=34+float64(700-34-newH)/2+float64(panY)+anchor.FractionY*float64(newH)
 if math.Abs(afterX-beforeX)>1||math.Abs(afterY-beforeY)>1{
  t.Fatalf("cursor drift: x=%v y=%v old x=%v y=%v",afterX,afterY,beforeX,beforeY)
 }
}

func TestPDFCursorAnchorSmallPageUsesAvailableMargins(t *testing.T) {
 // A 300x240 image fully fits 600x700, but pan may use empty margins
 // to keep the document coordinate under the pointer while zooming.
 anchor:=pdfCaptureCursorAnchor(230,240,300,240,600,700,34,0,0)
 if !anchor.Valid {
  // At default centered layout the image begins at x=150,y=247.
  anchor=pdfCaptureCursorAnchor(230,300,300,240,600,700,34,0,0)
 }
 if !anchor.Valid{t.Fatal("expected inside centered small PDF")}
 x,y:=pdfRestoreCursorPan(anchor,450,360,600,700,34)
 if x==0&&y==0{t.Fatalf("small page anchor did not offset despite noncentral pointer")}
 if x>75||x< -75||y>153||y< -153{t.Fatalf("pan outside permitted margins x=%d y=%d",x,y)}
}

func TestPDFCursorAnchorRejectsOutsidePageAndInvalidGeometry(t *testing.T) {
 if pdfCaptureCursorAnchor(-300,-300,800,1000,600,700,34,0,0).Valid {t.Fatal("mouse outside page accepted")}
 if pdfCaptureCursorAnchor(300,300,0,1000,600,700,34,0,0).Valid {t.Fatal("invalid size accepted")}
 if pdfClampAnchorPan(900,500,999)!=200 {t.Fatal("large page pan not bounded")}
 if pdfClampAnchorPan(200,500,999)!=150 {t.Fatal("small page pan not bounded")}
}

func TestPDFCursorAnchorCoalescedGeneration(t *testing.T) {
 anchor:=PDFCursorAnchor{Valid:true,Generation:7}
 if !pdfAnchorApplies(anchor,8,8) {t.Fatal("coalesced final render dropped live pointer anchor")}
 if pdfAnchorApplies(anchor,7,8) {t.Fatal("stale worker used pointer anchor")}
 if pdfAnchorApplies(anchor,8,9) {t.Fatal("superseded worker used pointer anchor")}
 if pdfAnchorApplies(PDFCursorAnchor{},8,8) {t.Fatal("empty anchor accepted")}
 if pdfAnchorApplies(PDFCursorAnchor{Valid:true,Generation:9},8,8) {t.Fatal("future anchor accepted")}
}

func TestPDFCursorAnchorXYAcrossSimulatedDPIScales(t *testing.T) {
 // In-memory geometry qualification only. All sizes and pointer positions
 // are scaled together as Win32 physical pixels, never mixing PDF DIPs
 // with decoded PNG dimensions. This does not change desktop DPI.
 for _,dpi:=range []int{96,120,144,192}{
  t.Run(fmt.Sprintf("dpi-%d",dpi),func(t *testing.T){
   scale:=float64(dpi)/96
   px:=func(v int32)int32{return int32(math.Round(float64(v)*scale))}
   viewportW,viewportH,toolbar:=px(900),px(700),px(34)
   oldW,oldH:=px(1000),px(1000)
   newW,newH:=px(1500),px(1500)
   for _,pt:=range []struct{x,y int32}{
    {px(260),px(220)},
    {px(380),px(350)},
    {px(490),px(470)},
    {px(635),px(295)},
   }{
    oldPanX,oldPanY:=px(15),px(-12)
    anchor:=pdfCaptureCursorAnchor(pt.x,pt.y,oldW,oldH,viewportW,viewportH,toolbar,oldPanX,oldPanY)
    if !anchor.Valid{t.Fatalf("DPI %d cursor %v outside displayed document",dpi,pt)}
    panX,panY:=pdfRestoreCursorPan(anchor,newW,newH,viewportW,viewportH,toolbar)
    newOriginX:=float64(viewportW-newW)/2+float64(panX)
    newOriginY:=float64(toolbar)+float64(viewportH-toolbar-newH)/2+float64(panY)
    restoredX:=newOriginX+anchor.FractionX*float64(newW)
    restoredY:=newOriginY+anchor.FractionY*float64(newH)
    if dx,dy:=math.Abs(restoredX-float64(pt.x)),math.Abs(restoredY-float64(pt.y));dx>1||dy>1{
     t.Fatalf("DPI %d pointer(%d,%d), restored(%0.2f,%0.2f), XY drift(%0.3f,%0.3f)",dpi,pt.x,pt.y,restoredX,restoredY,dx,dy)
    }
   }
  })
 }
}

func TestPDFCursorAnchorConstrainedPanDoesNotLeaveViewport(t *testing.T){
 // When an enlarged image clips the viewport, panning cannot exceed
 // the centered image edge bounds. A clamped gesture may necessarily
 // move the document point away from the cursor.
 for _,tc:=range []struct{content,viewport,requested int32;want int32}{
  {1200,600,1000,300},{1200,600,-1000,-300},
  {350,600,1000,125},{350,600,-1000,-125},
  {900,900,999,0},
 }{
  if got:=pdfClampAnchorPan(tc.content,tc.viewport,tc.requested);got!=tc.want{
   t.Fatalf("clamp (%d,%d,%d)=%d want=%d",tc.content,tc.viewport,tc.requested,got,tc.want)
  }
 }
}
