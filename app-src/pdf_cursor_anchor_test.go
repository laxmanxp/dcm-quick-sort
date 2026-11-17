package main

import (
 "math"
 "testing"
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
