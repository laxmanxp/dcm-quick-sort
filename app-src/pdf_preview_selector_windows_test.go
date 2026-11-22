//go:build windows

package main

import "testing"

func TestPDFPreviewerSessionChoice(t *testing.T) {
 for _,tc:=range []struct{env,want string}{
  {"","auto"},
  {"auto","auto"},
  {"PDFIUM","pdfium"},
  {"windows","windows"},
  {"SUMATRA","auto"},
  {"unknown","auto"},
 }{
  t.Run(tc.env,func(t *testing.T){
   t.Setenv("DCM_PDF_PROVIDER",tc.env)
   if got:=pdfPreviewChoice();got!=tc.want{t.Fatalf("choice %q: got %q, want %q",tc.env,got,tc.want)}
  })
 }
}

func TestPDFPreviewerNames(t *testing.T) {
 for _,choice:=range []string{"auto","pdfium","windows"} {
  if pdfPreviewChoiceName(choice)=="" {t.Fatal("empty menu label for "+choice)}
 }
}
