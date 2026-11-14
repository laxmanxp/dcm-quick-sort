package main

import (
	"errors"
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

func TestSumatraSidebarWidthsReserveToolbar(t *testing.T) {
	for _, tc := range []struct{ width, list, processed int32 }{
		{1600, 220, 260}, {1264, 220, 260}, {1200, 188, 228}, {1201, 188, 229},
		{1174, 175, 215}, {1144, 160, 200}, {1136, 160, 200}, {0, 160, 200}, {-1, 160, 200},
	} {
		list, processed := sumatraSidebarWidths(tc.width)
		if list != tc.list || processed != tc.processed {
			t.Errorf("width %d: (%d,%d), want (%d,%d)", tc.width, list, processed, tc.list, tc.processed)
		}
	}
	for width := int32(1144); width <= 2000; width++ {
		list, processed := sumatraSidebarWidths(width)
		if list < 160 || list > 220 || processed < 200 || processed > 260 || width-60-list-processed < 724 {
			t.Fatalf("invalid layout width=%d left=%d right=%d", width, list, processed)
		}
	}
}

// Display text must stay short without destroying the underlying OS error.
func TestSumatraDependencyErrorsAreConcise(t *testing.T) {
	for _, policyMissing := range []bool{false, true} {
		name := "runtime"
		component := "SumatraPDF.exe"
		if policyMissing {
			name = "policy"
			component = "sumatrapdfrestrict.ini"
		}
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), strings.Repeat("long-private-path-", 7))
			if err := os.MkdirAll(root, 0700); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(root, "SumatraPDF.exe")
			if policyMissing {
				if err := os.WriteFile(exe, []byte("untrusted fixture; never executed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := validateSumatraRuntime(exe)
			want := component + " is missing. Restore the approved private viewer, then Reload."
			if err == nil || err.Error() != want {
				t.Fatalf("message=%v; want %q", err, want)
			}
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatal("missing-file cause lost")
			}
			var detail *os.PathError
			if !errors.As(err, &detail) || !strings.HasPrefix(detail.Path, root) {
				t.Fatal("original diagnostic path lost")
			}
			if strings.Contains(err.Error(), root) || len(err.Error()) > 120 {
				t.Fatal("display exposes/clips a technical path")
			}
		})
	}
	for _, tc := range []struct {
		name, reason string
		cause        error
	}{
		{"permission", " access was denied. Check file permissions, then Reload.", &os.PathError{Op: "open", Path: "private-install-path", Err: os.ErrPermission}},
		{"other read failure", " could not be read. Check the approved private viewer, then Reload.", errors.New("unbounded internal failure details")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := newSumatraDependencyError("SumatraPDF.exe", tc.cause)
			if err.Error() != "SumatraPDF.exe"+tc.reason || !errors.Is(err, tc.cause) {
				t.Fatalf("lost classification/cause: %v", err)
			}
		})
	}
}
