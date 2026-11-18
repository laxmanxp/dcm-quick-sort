package main

import "testing"

func TestActualFileDisplayIncludesRealExtension(t *testing.T) {
	tests := []struct{ input, filename, kind string }{
		{"", "", ""},
		{"  ", "", ""},
		{`C:\Inbox\Quarterly.Statement.FINAL.pdf`, "Quarterly.Statement.FINAL.pdf", "PDF"},
		{`C:\Inbox\Budget.xlsx`, "Budget.xlsx", "XLSX"},
		{`C:\Inbox\Invitation.DOCX`, "Invitation.DOCX", "DOCX"},
		{`C:\Inbox\Agenda.doc`, "Agenda.doc", "DOC"},
		{`C:\Inbox\invoice.csv`, "invoice.csv", "CSV"},
		{`C:\Inbox\no-extension`, "no-extension", "NO EXT."},
		{`C:\Inbox\very.long.file.name.tar.gz`, "very.long.file.name.tar.gz", "GZ"},
	}
	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			name, kind := actualFileDisplay(tc.input)
			if name != tc.filename || kind != tc.kind {
				t.Fatalf("actualFileDisplay(%q)=(%q,%q), want (%q,%q)", tc.input, name, kind, tc.filename, tc.kind)
			}
		})
	}
}
