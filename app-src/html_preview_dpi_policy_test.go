package main

import "testing"

func TestNativeHTMLPreviewDPIContextPolicy(t *testing.T) {
 cases:=[]struct{path,word,html,want string}{
  {"page.html","","","aware"},
  {"PAGE.HTM","","","aware"},
  {"page.html","","off",""},
  {"page.html","","aware","aware"},
  {"page.html","","physical-setrect","physical-setrect"},
  {"PAGE.HTM","","physical-both","physical-both"},
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

func TestHTMLExperimentalRectForCall(t *testing.T) {
 for _,tc:=range []struct{mode,method string; measured bool;w,h int32}{
  {"physical-setrect","SetWindow",true,770,356},
  {"physical-setrect","SetRect",true,963,445},
  {"physical-both","SetWindow",true,963,445},
  {"physical-both","SetRect",true,963,445},
  {"aware","SetRect",true,770,356},
  {"physical-both","SetRect",false,770,356},
 } {
  w,h:=htmlPreviewRectForCall(tc.mode,tc.method,770,356,963,445,tc.measured)
  if w!=tc.w || h!=tc.h {t.Errorf("%s %s got %dx%d want %dx%d",tc.mode,tc.method,w,h,tc.w,tc.h)}
 }
}
