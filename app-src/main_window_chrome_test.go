package main

import "testing"

func TestPDFMainWindowChromeHasMaximizeAndResize(t *testing.T) {
	const overlapped, visible, clipChildren = uint32(0x00CF0000), uint32(0x10000000), uint32(0x02000000)
	style := mainWindowChromeStyle(overlapped, visible, clipChildren)
	for _, flag := range []uint32{0x00010000, 0x00040000, visible, clipChildren} {
		if style&flag != flag {
			t.Fatalf("main window missing style %#x in %#x", flag, style)
		}
	}
	if style != overlapped|visible|clipChildren {
		t.Fatalf("main chrome unexpectedly strips flags: %#x", style)
	}
}

func TestPDFMainWindowMinimumTrackSize(t *testing.T) {
	x, y := minimumMainWindowTrackSize()
	if x < 1080 || y < 720 {
		t.Fatalf("min tracking geometry too small: %dx%d", x, y)
	}
	if x > 1280 || y > 820 {
		t.Fatalf("min tracking geometry prevents restore to initial window: %dx%d", x, y)
	}
}
