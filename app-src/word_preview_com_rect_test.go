package main

import "testing"

func TestWordPreviewDPIContextMode(t *testing.T) {
 for _, tc := range []struct{path,setting,want string}{
  {"sample.docx","","aware"},
  {"sample.docx","aware","aware"},
  {"sample.DOCX","AWARE","aware"},
  {"sample.docx","off",""},
  {"sample.docx","setrect",""},
  {"sample.docx","both",""},
  {"sample.docx","invalid",""},
  {"sample.doc","aware",""},
  {"sample.pdf","",""},
  {"sample.docx.tmp","",""},
 } {
  if got:=wordPreviewDPIContextMode(tc.path,tc.setting);got!=tc.want {
   t.Errorf("%q %q got %q want %q",tc.path,tc.setting,got,tc.want)
  }
 }
}
