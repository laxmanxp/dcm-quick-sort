package main

import "testing"

func TestWordPreviewDPIContextMode(t *testing.T) {
 for _, tc := range []struct{path,setting,want string}{
  {"sample.docx","","aware"},
  {"sample.docx","aware","aware"},
  {"sample.pdf","lifecycle",""},
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

func TestWordPreviewOuterRepairOptOut(t *testing.T) {
 cases:=[]struct{path,env string;want bool}{
  {"sample.docx","",true},
  {"sample.DOCX","",true},
  {"sample.docx","off",false},
  {"sample.docx","OFF",false},
  {"sample.docx"," off ",false},
  {"sample.docx","on",true},
  {"sample.doc","",false},
  {"sample.html","",false},
  {"sample.pdf","",false},
 }
 for _,tc:=range cases {
  if got:=wordPreviewOuterRepairEnabled(tc.path,tc.env);got!=tc.want {
   t.Errorf("%q %q got=%v want=%v",tc.path,tc.env,got,tc.want)
  }
 }
}
