package main

import "testing"

func TestWordPreviewExperimentalMode(t *testing.T) {
 for _, tc := range []struct{path,setting,want string}{
  {"sample.docx","both","both"},
  {"sample.docx","aware","aware"},
  {"sample.DOCX","setrect","setrect"},
  {"sample.docx","invalid",""},
  {"sample.docx","",""},
  {"sample.doc","both",""},
  {"sample.pdf","both",""},
 } {
  if got:=wordPreviewExperimentalMode(tc.path,tc.setting);got!=tc.want {
   t.Errorf("%q %q got %q want %q",tc.path,tc.setting,got,tc.want)
  }
 }
}
func TestWordPreviewRectModes(t *testing.T) {
 logical:=previewRectSize{770,356}
 physical:=previewRectSize{963,445}
 for _,tc:=range []struct{mode,method string;measured bool;want previewRectSize}{
  {"","SetWindow",true,logical},
  {"both","SetWindow",true,physical},
  {"both","SetRect",true,physical},
  {"setrect","SetWindow",true,logical},
  {"setrect","SetRect",true,physical},
  {"both","SetRect",false,logical},
  {"both","SetRect",true,physical},
 } {
  if got:=wordPreviewRectForCall(tc.mode,tc.method,logical,physical,tc.measured);got!=tc.want {
   t.Errorf("%q %q measured=%v got=%+v want=%+v",tc.mode,tc.method,tc.measured,got,tc.want)
  }
 }
}
