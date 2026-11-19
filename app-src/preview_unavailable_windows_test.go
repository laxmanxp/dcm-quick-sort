//go:build windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDCM33UnavailablePreviewStatusTransition(t *testing.T) {
	for _, outcome := range []struct {
		name       string
		filename   string
		registered bool
		wantStatus string
	}{
		{"missing-handler", "report.doc", false, "Preview unavailable: DOC (no Windows preview handler installed)"},
		{"registered-handler-failed", "report.DOC", true, "Preview unavailable: DOC (Windows handler failed)"},
		{"extensionless", "report", false, "Preview unavailable: file has no extension"},
	} {
		for _, prior := range []struct {
			name   string
			status string
		}{
			{"blank", ""},
			{"text-success", "Plain-text preview: TXT"},
			{"native-success", "Windows preview: DOC"},
		} {
			t.Run(outcome.name+"/"+prior.name, func(t *testing.T) {
				// Each subtest creates, uses and destroys its own HWNDs on one
				// locked thread. No Office COM activation or registry mutation
				// is needed to exercise the real unavailable-state presenter.
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				oldApp := app
				defer func() { app = oldApp }()
				parent := controlTestParent(t)
				defer func() {
					if ok, _, _ := user32.NewProc("DestroyWindow").Call(parent); ok == 0 {
						t.Error("could not destroy owned unavailable-preview test window")
					}
				}()
				preview := createControl("STATIC", "preview-test-sentinel", WS_CHILD, 0, 0, 300, 200, parent, 952)
				status := createControl("STATIC", "status-test-sentinel", WS_CHILD, 0, 210, 580, 30, parent, 953)
				if preview == 0 || status == 0 || preview == status {
					t.Fatal("private preview/status control creation failed")
				}
				if getText(parent) != "control-update-private-test" || getText(preview) != "preview-test-sentinel" || getText(status) != "status-test-sentinel" {
					t.Fatal("private test control readback failed")
				}
				app = appState{previewHwnd: preview, statusLabel: status}
				path := filepath.Join(t.TempDir(), outcome.filename)
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
				setText(status, prior.status)
				if getText(status) != prior.status {
					t.Fatal("prior footer state could not be established")
				}
				app.fallback = "previous-file fallback sentinel"
				showPreviewUnavailable(path, outcome.registered)
				if got, want := app.fallback, previewFallbackText(path, outcome.registered); got != want {
					t.Fatalf("unavailable presenter did not assign current file body: got %q; want %q", got, want)
				}
				if getText(preview) != "preview-test-sentinel" {
					t.Fatal("presenter wrote the footer to the wrong control")
				}
				if got := getText(status); got != outcome.wantStatus {
					t.Fatalf("DCM33_UNAVAILABLE_STATUS: got %q; want %q", got, outcome.wantStatus)
				}
			})
		}
	}
}
