package main

import "testing"

func TestPDFViewModeNormalization(t *testing.T) {
 if normalizePDFViewMode("unexpected")!=PDFViewSingle {t.Fatal("invalid mode not reset")}
 if normalizePDFViewMode(PDFViewContinuous)!=PDFViewContinuous {t.Fatal("continuous mode lost")}
}
func TestPDFScrollStepBoundaries(t *testing.T) {
 cases:=[]struct{page,count int;pos,viewport,content,delta int32;wantPage int;wantPos int32}{
  {0,3,0,400,900,100,0,100},
  {0,3,490,400,900,30,1,0},
  {1,3,0,400,900,-5,0,-1},
  {0,3,0,400,900,-80,0,0},
  {2,3,500,400,900,80,2,500},
  {0,1,0,400,300,60,0,0},
  {0,2,0,400,300,60,1,0},
  {0,2,0,400,300,-60,0,0},
 }
 for _,c:=range cases {
  p,pos:=pdfScrollStep(c.page,c.count,c.pos,c.viewport,c.content,c.delta)
  if p!=c.wantPage || pos!=c.wantPos {t.Errorf("got %d,%d want %d,%d for %+v",p,pos,c.wantPage,c.wantPos,c)}
 }
}
