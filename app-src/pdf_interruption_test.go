package main

import "testing"

func TestDCM34PDFLateResultWorkerIdentity(t *testing.T) {
	for _, tc := range []struct {
		name           string
		busy           bool
		worker, result uint64
		want           bool
	}{
		{"blank", false, 0, 0, false},
		{"invalid-zero-identity", true, 0, 0, false},
		{"old-selection", true, 43, 41, false},
		{"old-error", true, 43, 41, false},
		{"same-file-reopened", true, 45, 43, false},
		{"current-worker", true, 43, 43, true},
		{"coalesced-current-worker", true, 43, 43, true},
		{"already-finished", false, 43, 43, false},
		{"stopped-selection", false, 0, 43, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := pdfCompletionOwnsWorker(tc.busy, tc.worker, tc.result); got != tc.want {
				t.Fatalf("owns=%v want=%v", got, tc.want)
			}
		})
	}
}
