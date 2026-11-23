package main

import "testing"

func TestPreferOwnedDOCXPreview(t *testing.T) {
 for _, tc := range []struct{path string; want bool}{
  {"report.docx",true},
  {"REPORT.DOCX",true},
  {`C:\Inbox\memo.Docx`,true},
  {"report.doc",false},
  {"slides.pptx",false},
  {"report.docx.tmp",false},
  {"report.pdf",false},
  {"",false},
 } {
  if got:=preferOwnedDOCXPreview(tc.path); got!=tc.want {
   t.Errorf("preferOwnedDOCXPreview(%q)=%v want %v",tc.path,got,tc.want)
  }
 }
}
