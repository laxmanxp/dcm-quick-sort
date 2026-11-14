package main

import (
	"reflect"
	"strings"
	"testing"
)

// DCM-34: renderer output without a usable page must remain an error. These
// protocol tests do not claim that a damaged PDF was opened by WinRT; the
// registered headed workflow supplies that separate decoder/recovery evidence.
func TestDCM34PDFRendererRejectsFailedOutput(t *testing.T) {
	for _, tc := range []struct {
		name, output, reason string
	}{
		{"blank", "", "no result"},
		{"whitespace", " \r\n\t\n", "no result"},
		{"diagnostic-only", "PDF document could not be loaded", "PDF document could not be loaded"},
		{"powershell-failure", "Exception calling LoadFromFileAsync: damaged document\r\n", "damaged document"},
		{"incomplete-result", "OK|2|0|612", "unexpected PDF renderer result"},
		{"extra-result-fields", "OK|2|0|612|792|100|unexpected", "unexpected PDF renderer result"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta, err := parsePDFRendererOutput(tc.output)
			if err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("unusable renderer output must retain its failure: metadata=%+v error=%v; want reason %q", meta, err, tc.reason)
			}
			if meta != (PDFRenderMetadata{}) {
				t.Fatalf("failed output must not advertise page metadata: %+v", meta)
			}
		})
	}
}

func TestDCM34PDFRendererRejectsInvalidMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, output string
	}{
		{"no-pages", "OK|0|0|612|792|100"},
		{"negative-page-count", "OK|-1|0|612|792|100"},
		{"negative-page-index", "OK|2|-1|612|792|100"},
		{"page-beyond-document", "OK|2|2|612|792|100"},
		{"zero-width", "OK|2|0|0|792|100"},
		{"negative-width", "OK|2|0|-612|792|100"},
		{"zero-height", "OK|2|0|612|0|100"},
		{"negative-height", "OK|2|0|612|-792|100"},
		{"invalid-count", "OK|many|0|612|792|100"},
		{"invalid-index", "OK|2|first|612|792|100"},
		{"invalid-width", "OK|2|0|wide|792|100"},
		{"invalid-height", "OK|2|0|612|tall|100"},
		{"invalid-scale", "OK|2|0|612|792|fit"},
		{"overflowing-number", "OK|99999999999999999999999999999|0|612|792|100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta, err := parsePDFRendererOutput(tc.output)
			if err == nil {
				t.Fatalf("invalid page metadata was accepted: %+v", meta)
			}
			if meta != (PDFRenderMetadata{}) {
				t.Fatalf("invalid output leaked success metadata: %+v", meta)
			}
		})
	}
}

func TestDCM34PDFRendererAcceptsSuccess(t *testing.T) {
	for _, tc := range []struct {
		name, output string
		want         PDFRenderMetadata
	}{
		{"legacy-metadata", "OK|2|0|612|792\n", PDFRenderMetadata{PageCount: 2, PageIndex: 0, Width: 612, Height: 792}},
		{"current-fit-result", "OK|2|1|306|396|50\r\n", PDFRenderMetadata{PageCount: 2, PageIndex: 1, Width: 306, Height: 396, ScalePercent: 50}},
		{"diagnostic-before-result", "Loading owned renderer\r\nOK|1|0|800|600|125\r\n", PDFRenderMetadata{PageCount: 1, PageIndex: 0, Width: 800, Height: 600, ScalePercent: 125}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePDFRendererOutput(tc.output)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("usable page metadata lost: got=%+v error=%v; want=%+v", got, err, tc.want)
			}
		})
	}
}
