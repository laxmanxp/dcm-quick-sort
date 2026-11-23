package main

import "testing"

func TestNativeHTMLPreviewDPIContextPolicy(t *testing.T) {
 cases:=[]struct{path,word,html,want string}{
  {"page.html","","","aware"},
  {"PAGE.HTM","","","aware"},
  {"page.html","","off",""},
  {"page.html","","aware","aware"},
  {"page.html","off","","aware"},
  {"page.html","","invalid",""},
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
