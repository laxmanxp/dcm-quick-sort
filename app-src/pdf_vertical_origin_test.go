package main

import "testing"

func TestPDFPageTopRelativeToToolbar(t *testing.T){
 cases:=[]struct{name string;top,bottom,page,pan int32;fit,fitWidth bool;want int32}{
  {"fit-width-tall-begins-at-toolbar",34,434,800,0,false,true,34},
  {"fit-width-short-begins-at-toolbar",34,434,200,0,false,true,34},
  {"fit-width-scrolls-down",34,434,800,-140,false,true,-106},
  {"fit-width-scroll-limit",34,434,800,-900,false,true,-366},
  {"fit-width-after-resize",34,634,900,0,false,true,34},
  {"fit-width-after-resize-scroll",34,634,900,-140,false,true,-106},
  {"fit-page-centered-below-toolbar",34,434,200,0,true,false,134},
  {"manual-centered-below-toolbar",34,434,600,0,false,false,-66},
  {"toolbar-height-differs",48,448,200,0,false,true,48},
 }
 for _,tc:=range cases {
  got:=pdfPageTop(tc.top,tc.bottom,tc.page,tc.pan,tc.fit,tc.fitWidth)
  if got!=tc.want {t.Errorf("%s: got %d want %d",tc.name,got,tc.want)}
 }
}

func TestPDFTopAnchoredPan(t *testing.T) {
 for _,tc:=range []struct{page,viewport,pan,want int32}{
  {800,400,0,0},
  {800,400,-70,-70},
  {800,400,-1000,-400},
  {800,400,250,0},
  {200,400,-100,0},
  {800,600,-300,-200},
 } {
  if got:=pdfTopAnchoredPan(tc.page,tc.viewport,tc.pan);got!=tc.want{
   t.Errorf("page=%d viewport=%d pan=%d got %d want %d",tc.page,tc.viewport,tc.pan,got,tc.want)
  }
 }
}
