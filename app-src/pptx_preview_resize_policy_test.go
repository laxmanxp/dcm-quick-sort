package main

import "testing"

func TestPPTXPreviewPhysicalCOMGeometry(t *testing.T) {
	cases := []struct {
		name string
		logicalW, logicalH, physicalW, physicalH int32
		pptx, available bool
		wantW, wantH int32
	}{
		{"native-125percent", 722, 390, 902, 487, true, true, 902, 487},
		{"native-100percent", 902, 487, 902, 487, true, true, 902, 487},
		{"other-handler-unaffected", 722, 390, 902, 487, false, true, 722, 390},
		{"unavailable-measurement", 722, 390, 0, 0, true, false, 722, 390},
		{"invalid-physical-height", 722, 390, 902, 0, true, true, 722, 390},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, h := pptxCOMRectSize(tc.logicalW, tc.logicalH, tc.physicalW, tc.physicalH, tc.pptx, tc.available)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("got %dx%d, want %dx%d", w, h, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestPPTXDeferredPreviewResizePolicy(t *testing.T) {
	cases := []struct {
		name string
		hr uintptr
		active, running, pending, want bool
	}{
		{"com-input-sync-failure", 0x8001010d, true, false, false, true},
		{"other-handler", 0x8001010d, false, false, false, false},
		{"success", 0, true, false, false, false},
		{"other-com-error", 0x80004005, true, false, false, false},
		{"retry-running", 0x8001010d, true, true, false, false},
		{"retry-pending", 0x8001010d, true, false, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldPostPptxPreviewResize(tc.hr, tc.active, tc.running, tc.pending); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestPPTXDeferredPreviewIdentity(t *testing.T) {
	cases := []struct {
		name string
		pending, active bool
		messageEpoch, currentEpoch uint64
		messagePane, currentPane uintptr
		want bool
	}{
		{"active-original", true, true, 7, 7, 100, 100, true},
		{"unloaded", true, false, 7, 7, 100, 100, false},
		{"not-pending", false, true, 7, 7, 100, 100, false},
		{"stale-epoch", true, true, 6, 7, 100, 100, false},
		{"different-pane", true, true, 7, 7, 99, 100, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRunPptxPreviewResizeRetry(tc.pending, tc.active, tc.messageEpoch, tc.currentEpoch, tc.messagePane, tc.currentPane)
			if got != tc.want { t.Fatalf("got %v want %v", got, tc.want) }
		})
	}
}
