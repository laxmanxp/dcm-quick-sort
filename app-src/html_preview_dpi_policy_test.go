package main

import "testing"

func TestNativeHTMLPreviewHostRepairPolicy(t *testing.T) {
 cases:=[]struct{path,word,html,want string}{
  {"page.html","","","repair"},
  {"PAGE.HTM","","","repair"},
  {"page.html","","repair","repair"},
  {"page.html","","off",""},
  {"page.html","","aware","aware"},
  {"page.html","off","","repair"},
  {"page.html","","physical-setrect",""},
  {"page.html","","physical-both",""},
  {"page.html","","physical-window",""},
  {"document.docx","","","aware"},
  {"document.docx","off","",""},
  {"document.docx","","off","aware"},
  {"page.pdf","","",""},
  {"page.html.bak","","",""},
 } 
 for _,tc:=range cases {
  if got:=nativePreviewDPIContextMode(tc.path,tc.word,tc.html);got!=tc.want {
   t.Errorf("nativePreviewDPIContextMode(%q,%q,%q)=%q want %q",tc.path,tc.word,tc.html,got,tc.want)
  }
 }
}
