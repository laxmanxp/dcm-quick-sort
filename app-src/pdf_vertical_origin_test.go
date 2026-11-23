package main

import "testing"

func TestPDFPageTopRelativeToToolbar(t *testing.T){
 for _,tc:=range []struct{name string;top,bottom,page,pan int32;fit,fitWidth bool;want int32}{
  {"fit-width-tall",34,434,800,204,false,true,38},
  {"fit-width-short",34,434,200,0,false,true,38},
  {"fit-width-after-resize",34,634,900,154,false,true,38},
  {"fit-page-centered-below-toolbar",34,434,200,0,true,false,134},
  {"manual-centered-below-toolbar",34,434,600,0,false,false,-66},
  {"toolbar-offset-differs",48,448,200,0,false,true,52},
 }{
  got:=pdfPageTop(tc.top,tc.bottom,tc.page,tc.pan,tc.fit,tc.fitWidth)
  if got!=tc.want{t.Errorf("%s: got %d want %d",tc.name,got,tc.want)}
 }
}
