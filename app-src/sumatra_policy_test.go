package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSumatraArgumentsAndOptIn(t *testing.T) {
	root := t.TempDir()
	pdf := filepath.Join(root, "spaced file.pdf")
	profile := filepath.Join(root, "private profile")
	got, err := sumatraArguments(42, pdf, profile)
	want := []string{"-restrict", "-appdata", profile, "-view", "single page", "-zoom", "fit page", "-plugin", "42", pdf}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("args=%q err=%v", got, err)
	}
	for _, tc := range []struct {
		parent        uintptr
		file, profile string
	}{{0, pdf, profile}, {1, "relative.pdf", profile}, {1, pdf, "relative"}, {1, pdf + "\x00", profile}, {1, pdf, profile + "\x00"}, {1, filepath.Join(root, "bad.txt"), profile}} {
		if _, err := sumatraArguments(tc.parent, tc.file, tc.profile); err == nil {
			t.Fatal("unsafe args accepted")
		}
	}
	t.Setenv("DCM_PDF_PROVIDER", "")
	if useSumatraProvider() {
		t.Fatal("default changed")
	}
	t.Setenv("DCM_PDF_PROVIDER", "sumatra")
	if !useSumatraProvider() {
		t.Fatal("opt-in ignored")
	}
}
func TestSumatraPolicyFailsClosed(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "SumatraPDF.exe")
	policy := filepath.Join(root, "sumatrapdfrestrict.ini")
	if err := os.WriteFile(exe, []byte("not the runtime"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", strings.Replace(sumatraPreviewPolicy, "PrinterAccess = 0", "PrinterAccess = 1", 1), sumatraPreviewPolicy} {
		if err := os.WriteFile(policy, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := validateSumatraRuntime(exe); err == nil {
			t.Fatal("untrusted runtime or policy accepted")
		}
	}
	if _, err := validateSumatraRuntime("SumatraPDF.exe"); err == nil {
		t.Fatal("relative/PATH runtime accepted")
	}
	if !strings.Contains(sumatraPreviewPolicy, "PrinterAccess = 0") || !strings.Contains(sumatraPreviewPolicy, "DiskAccess = 0") {
		t.Fatal("restrictions weakened")
	}
}
func TestSumatraDocumentValidationIsReadOnly(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "document.pdf")
	data := []byte("%PDF-1.4\nfixture, not complete PDF validation")
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := validateSumatraDocument(p)
	if err != nil || got != p {
		t.Fatalf("%q %v", got, err)
	}
	for _, tc := range []struct {
		name string
		data []byte
	}{{"empty.pdf", nil}, {"bad.pdf", []byte("not a pdf header")}, {"other.txt", data}} {
		f := filepath.Join(root, tc.name)
		if err := os.WriteFile(f, tc.data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := validateSumatraDocument(f); err == nil {
			t.Fatalf("accepted %s", tc.name)
		}
	}
	after, err := os.ReadFile(p)
	if err != nil || string(after) != string(data) {
		t.Fatal("source changed")
	}
}

func TestSumatraToolbarClippingPolicy(t *testing.T) {
	for _, tc := range []struct {
		name               string
		y, h, parent, want int32
		move               bool
	}{
		{"page edit", -13, 25, 43, 9, true}, {"page border", -17, 33, 43, 5, true},
		{"valid find edit", 6, 25, 43, 6, false}, {"bottom clip", 30, 25, 43, 9, true},
		{"top edge", 0, 25, 43, 0, false}, {"bottom edge", 18, 25, 43, 18, false},
		{"full height", 0, 43, 43, 0, false}, {"empty", -1, 0, 43, -1, false},
		{"oversized", -13, 50, 43, -13, false}, {"invalid parent", -13, 25, 0, -13, false},
		{"unknown toolbar", -13, 25, 800, -13, false}, {"scaled", -26, 50, 86, 18, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			y, move := sumatraToolbarChildY(tc.y, tc.h, tc.parent)
			if y != tc.want || move != tc.move {
				t.Fatalf("got (%d,%v), want (%d,%v)", y, move, tc.want, tc.move)
			}
		})
	}
}
