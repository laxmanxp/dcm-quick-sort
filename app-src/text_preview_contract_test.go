package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func dcm26UTF16(text string, little bool) []byte {
	data := []byte{0xFE, 0xFF}
	if little {
		data = []byte{0xFF, 0xFE}
	}
	for _, u := range utf16.Encode([]rune(text)) {
		if little {
			data = append(data, byte(u), byte(u>>8))
		} else {
			data = append(data, byte(u>>8), byte(u))
		}
	}
	return data
}
func dcm26Read(t *testing.T, data []byte, limit int64) (string, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.txt")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readTextPreview(path, limit)
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(after, data) {
		t.Fatal("preview changed source bytes")
	}
	return got, err
}
func TestDCM26TextPreviewEncodings(t *testing.T) {
	text := "Text: café Ω ଓଡ଼ିଆ 😀\nsecond\rthird\r\nfourth"
	want := "Text: café Ω ଓଡ଼ିଆ 😀\r\nsecond\r\nthird\r\nfourth"
	for name, data := range map[string][]byte{
		"utf8": []byte(text), "utf8-bom": append([]byte{0xEF, 0xBB, 0xBF}, []byte(text)...),
		"utf16-le": dcm26UTF16(text, true), "utf16-be": dcm26UTF16(text, false),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := dcm26Read(t, data, 0)
			if err != nil || got != want {
				t.Fatalf("got %q, %v; want %q", got, err, want)
			}
		})
	}
	for name, data := range map[string][]byte{
		"invalid-utf8": {'A', 0xFF, 'B'}, "incomplete-utf8": {'A', 0xE2, 0x82},
		"utf16-odd": {0xFF, 0xFE, 'A', 0, 'B'}, "utf16-unpaired-high": {0xFF, 0xFE, 0, 0xD8, 'B', 0},
		"utf16-unpaired-low": {0xFE, 0xFF, 0xDC, 0}, "unsupported-utf32-le": {0xFF, 0xFE, 0, 0, 'A', 0, 0, 0},
		"unsupported-utf32-be": {0, 0, 0xFE, 0xFF, 0, 0, 0, 'A'},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := dcm26Read(t, data, 0)
			if err == nil || got != "" {
				t.Fatalf("malformed/unsupported encoding silently displayed: %q, %v", got, err)
			}
		})
	}
	t.Run("literal-replacement-character", func(t *testing.T) {
		got, err := dcm26Read(t, []byte("valid � character"), 0)
		if err != nil || got != "valid � character" {
			t.Fatalf("valid U+FFFD rejected: %q %v", got, err)
		}
	})
}
func TestDCM26TextPreviewLimits(t *testing.T) {
	for _, runeText := range []string{"é", "€", "😀"} {
		data := []byte("AB" + runeText + "TAIL")
		for n := 1; n < len([]byte(runeText)); n++ {
			t.Run(fmt.Sprintf("utf8-%X-cut-%d", []byte(runeText), n), func(t *testing.T) {
				limit := int64(2 + n)
				got, err := dcm26Read(t, data, limit)
				want := "AB\r\n\r\n[Preview limited to the first " + humanFileSize(limit) + ".]"
				if err != nil || got != want {
					t.Fatalf("split UTF-8 character was corrupted: %q %v; want %q", got, err, want)
				}
			})
		}
	}
	for _, le := range []bool{true, false} {
		for _, limit := range []int64{5, 6, 7} {
			t.Run(fmt.Sprintf("utf16-%t-cut-%d", le, limit), func(t *testing.T) {
				got, err := dcm26Read(t, dcm26UTF16("A😀TAIL", le), limit)
				want := "A\r\n\r\n[Preview limited to the first " + humanFileSize(limit) + ".]"
				if err != nil || got != want {
					t.Fatalf("split UTF-16 character was corrupted: %q %v; want %q", got, err, want)
				}
			})
		}
	}
	for _, extra := range []int{0, 1, 1000} {
		t.Run(fmt.Sprintf("default-limit-plus-%d", extra), func(t *testing.T) {
			data := []byte(strings.Repeat("A", int(defaultTextPreviewLimit)+extra))
			got, err := dcm26Read(t, data, 0)
			want := strings.Repeat("A", int(defaultTextPreviewLimit))
			if extra > 0 {
				want += "\r\n\r\n[Preview limited to the first 256.0 KiB.]"
			}
			if err != nil || got != want {
				t.Fatalf("wrong default bound/notice: length=%d err=%v", len(got), err)
			}
		})
	}
}
func TestDCM26TextPreviewReadSafety(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got, err := dcm26Read(t, nil, 0)
		if err != nil || got != "" {
			t.Fatalf("%q %v", got, err)
		}
	})
	t.Run("binary-controls", func(t *testing.T) {
		got, err := dcm26Read(t, []byte{'A', 0, 1, 2, 'B'}, 0)
		if err != nil || !strings.Contains(got, "Binary/non-text") || strings.ContainsRune(got, 0) || !strings.HasSuffix(got, "A···B") {
			t.Fatalf("unsafe interpretation: %q %v", got, err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		got, err := readTextPreview(filepath.Join(t.TempDir(), "missing.txt"), 0)
		if err == nil || got != "" {
			t.Fatalf("%q %v", got, err)
		}
	})
	t.Run("directory", func(t *testing.T) {
		got, err := readTextPreview(t.TempDir(), 0)
		if err == nil || got != "" {
			t.Fatalf("%q %v", got, err)
		}
	})
}
