package main

import "testing"

func TestPDFToolbarPlanCompactRetainsBothRotationButtons(t *testing.T) {
 for _,width:=range []int32{240,260,320,400,453} {
  slots:=pdfToolbarPlan(width)
  found:=map[int]bool{}
  previousEnd:=int32(0)
  for _,s:=range slots {
   if s.X<previousEnd {t.Fatalf("width %d: overlap at index %d",width,s.Index)}
   if s.X+s.Width+4>width {t.Fatalf("width %d: index %d exceeds pane",width,s.Index)}
   previousEnd=s.X+s.Width
   found[s.Index]=true
  }
  for _,i:=range []int{0,1,2,4,7,8} {
   if !found[i] {t.Errorf("width %d: essential control %d hidden",width,i)}
  }
 }
}

func TestPDFToolbarPlanWidePreservesOriginalOrder(t *testing.T) {
 slots:=pdfToolbarPlan(454)
 if len(slots)!=11 {t.Fatalf("wide pane has %d controls, want 11",len(slots))}
 for i,s:=range slots {
  if s.Index!=i {t.Fatalf("wide control %d in position %d",s.Index,i)}
 }
}

func TestPDFToolbarPlanNeverOverlapsAtTinyWidths(t *testing.T) {
 for width:=int32(0);width<454;width++ {
  previousEnd:=int32(0)
  for _,s:=range pdfToolbarPlan(width) {
   if s.X<previousEnd || s.X+s.Width+4>width {
    t.Fatalf("width %d invalid slot %+v",width,s)
   }
   previousEnd=s.X+s.Width
  }
 }
}
