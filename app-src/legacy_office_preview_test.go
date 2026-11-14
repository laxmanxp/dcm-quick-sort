package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDCM33LegacyOfficeRequiresHandler(t *testing.T) {
	for _, name := range []string{"report.doc", "report.DOC", "book.xls", "book.XLS", "slides.ppt", "slides.PPT"} {
		t.Run(name, func(t *testing.T) {
			if categoryForPath(name) != CategoryDocuments || !isKnownFileType(name) {
				t.Fatal("legacy Office file must remain a known document type")
			}
			if shouldUsePlainTextPreview(name) || canReadAsText(name) {
				t.Fatal("legacy Office binary must not be offered as a plain-text preview")
			}
			if kind, ok := openXMLKindForPath(name); ok || kind != OpenXMLNone {
				t.Fatalf("legacy Office file was classified as Open XML: %q, %v", kind, ok)
			}
			if _, _, ok := docxPreviewText(name); ok {
				t.Fatal("DOCX fallback accepted a legacy Office file")
			}
			if _, ok := xlsxPreviewForPath(name); ok {
				t.Fatal("XLSX fallback accepted a legacy Office file")
			}
			if _, ok := pptxPreviewForPath(name); ok {
				t.Fatal("PPTX fallback accepted a legacy Office file")
			}
			if _, ok := officeFoundationPreviewText(name); ok {
				t.Fatal("Open XML foundation fallback accepted a legacy Office file")
			}
		})
	}
}

func TestDCM33LegacyOfficeUnavailableText(t *testing.T) {
	for _, file := range []struct {
		name string
		kind string
		data []byte
	}{
		{"report.doc", "DOC", []byte("metadata-only unit fixture")},
		{"empty.DOC", "DOC", nil},
		{"book.xls", "XLS", []byte("metadata-only unit fixture")},
		{"slides.PPT", "PPT", []byte("metadata-only unit fixture")},
	} {
		for _, registered := range []bool{false, true} {
			state := "missing-handler"
			if registered {
				state = "registered-handler-failed"
			}
			t.Run(file.name+"/"+state, func(t *testing.T) {
				// These bytes exercise metadata and preservation only. They are
				// never represented as a valid legacy Office document or opened
				// through an Office handler.
				path := filepath.Join(t.TempDir(), file.name)
				if err := os.WriteFile(path, file.data, 0600); err != nil {
					t.Fatal(err)
				}
				got := previewFallbackText(path, registered)
				for _, want := range []string{
					"Name: " + file.name,
					"Size: " + humanFileSize(int64(len(file.data))),
					"Modified: ",
					"You can still rename, move, open, or open its folder.",
				} {
					if !strings.Contains(got, want) {
						t.Fatalf("unavailable preview missing %q: %q", want, got)
					}
				}
				if registered {
					want := "Preview unavailable\n\nWindows has a preview handler for " + file.kind + " files, but it could not preview this file."
					if !strings.HasPrefix(got, want) || strings.Contains(got, "No Windows preview handler is installed") {
						t.Fatalf("registered-handler failure was misrepresented: %q", got)
					}
				} else {
					want := "No preview available\n\nNo Windows preview handler is installed for " + file.kind + " files."
					if !strings.HasPrefix(got, want) || strings.Contains(got, "but it could not preview this file") {
						t.Fatalf("missing handler was misrepresented: %q", got)
					}
				}
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(after, file.data) {
					t.Fatalf("unavailable preview changed source bytes: %v", err)
				}
			})
		}
	}
}
