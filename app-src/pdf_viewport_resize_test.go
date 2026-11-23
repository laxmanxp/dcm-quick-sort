package main

import "testing"

func TestPDFViewportNeedsRerender(t *testing.T) {
 for _,tc:=range []struct{active bool;oldW,oldH,newW,newH int;want bool}{
  {true,700,500,700,500,false},
  {true,700,500,800,500,true},
  {true,700,500,700,400,true},
  {true,700,500,800,400,true},
  {true,0,0,700,500,true},
  {false,700,500,800,400,false},
  {true,700,500,0,400,false},
  {true,700,500,800,0,false},
 } {
  got:=pdfViewportNeedsRerender(tc.active,tc.oldW,tc.oldH,tc.newW,tc.newH)
  if got!=tc.want {t.Errorf("active=%v old=%dx%d new=%dx%d got=%v want=%v",tc.active,tc.oldW,tc.oldH,tc.newW,tc.newH,got,tc.want)}
 }
}
