package main

import (
	"strings"
	"testing"
)

func TestDCM34NativeSizingPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		allowed    bool
	}{
		{"blank", "", false}, {"whitespace", "  ", false},
		{"pdf", "broken.pdf", false}, {"uppercase-pdf", "valid.PDF", false},
		{"spaced-path", "C:/test folder/report.pdf", false},
		{"word", "report.docx", true}, {"legacy-word", "report.DOC", true},
		{"excel", "report.xlsx", true}, {"powerpoint", "report.pptx", true},
		{"not-pdf-extension", "report.pdf.docx", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := nativePreviewHostSizingAllowed(tc.path); got != tc.allowed {
				t.Fatalf("host sizing for %q = %v; want %v", tc.path, got, tc.allowed)
			}
		})
	}
}

func TestDCM34NativeSizingWiring(t *testing.T) {
	src := windowsMainSource(t)
	start := strings.Index(src, "func loadWindowsPreviewHandler(path string)")
	end := strings.Index(src[start:], "func createPreviewHandler")
	if start < 0 || end < 0 {
		t.Fatal("native loader missing")
	}
	loader := src[start : start+end]
	policy := strings.Index(loader, "app.previewHostSizingAllowed = nativePreviewHostSizingAllowed(path)")
	autoFit := strings.Index(loader, "autoFitPreviewIfSupported(path, clsidText)")
	if policy < 0 || autoFit < 0 || policy > autoFit {
		t.Fatal("native sizing permission must bind actual loaded path before automatic native sizing")
	}
	start = strings.Index(src, "func unloadPreview()")
	end = strings.Index(src[start:], "func previewWindowProc")
	unload := src[start : start+end]
	if !strings.Contains(unload, "app.previewHostSizingAllowed = false") {
		t.Fatal("native sizing permission must be cleared when preview retires")
	}
}
