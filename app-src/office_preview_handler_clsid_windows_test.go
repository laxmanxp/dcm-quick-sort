//go:build windows

package main

import "testing"

// Guard both exact installed Office preview CLSIDs. A single missing Data4
// zero disables Excel's established outer-window repair without a compile
// error; PPTX must remain a distinct handler with its own sizing policy.
func TestXLSXKnownOfficePreviewHandlerClassIDs(t *testing.T) {
	cases := []struct {
		name, text string
		got GUID
	}{
		{"excel", "{00020827-0000-0000-C000-000000000046}", clsidExcelPreviewHandler},
		{"powerpoint", "{65235197-874B-4A07-BDC5-E65EA825B718}", clsidPptxPreviewHandler},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var expected GUID
			if !guidFromString(tc.text, &expected) {
				t.Fatalf("could not parse pinned %s CLSID", tc.name)
			}
			if tc.got != expected {
				t.Fatalf("%s handler CLSID changed: got %#v, expected %#v", tc.name, tc.got, expected)
			}
		})
	}
}
