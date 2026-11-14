//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// A blocked preparation is injected only in this test binary. Production has
// no artificial-delay switch. These tests exercise the real operation guard and
// close policy while the resource worker has not yet returned.
func TestSumatraPendingStartupGuards(t *testing.T) {
	oldApp, oldView := app, sumatraView
	oldPending, oldQueued, oldRetired, oldClosing := sumatraPending, sumatraQueued, sumatraRetired, sumatraClosing
	defer func() {
		app = oldApp
		sumatraView = oldView
		sumatraPending = oldPending
		sumatraQueued = oldQueued
		sumatraRetired = oldRetired
		sumatraClosing = oldClosing
	}()
	for _, operation := range []string{"rename", "move", "close"} {
		t.Run(operation, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			app = appState{}
			sumatraView = nil
			sumatraQueued = nil
			sumatraRetired = nil
			sumatraClosing = false
			hidden := createControl("STATIC", "", 0, 0, 0, 200, 100, 0, 0)
			if hidden == 0 {
				t.Fatal("hidden window unavailable")
			}
			defer user32.NewProc("DestroyWindow").Call(hidden)
			app.hwnd, app.previewHwnd = hidden, hidden
			gate := make(chan struct{})
			disposed := make(chan struct{})
			sumatraPending = newPreparedResource(func(context.Context) (*sumatraSession, error) { <-gate; return &sumatraSession{}, nil }, func(*sumatraSession) error { close(disposed); return nil })
			root := t.TempDir()
			src, dst := filepath.Join(root, "source.pdf"), filepath.Join(root, "target.pdf")
			if err := os.WriteFile(src, []byte("original bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			if operation == "close" {
				if closeSumatraPreview() {
					t.Fatal("close admitted before cleanup")
				}
				if !sumatraClosing {
					t.Fatal("closing state lost")
				}
			} else {
				mode := fileOperationRename
				if operation == "move" {
					mode = fileOperationMove
				}
				beginFileOperation(mode, src, dst)
				if app.pendingFileOperation != fileOperationNone || app.backgroundMoveBusy {
					t.Fatal("file operation admitted while worker pending")
				}
				if !strings.Contains(app.fallback, "being cancelled") {
					t.Fatal("missing truthful retry message")
				}
			}
			data, err := os.ReadFile(src)
			if err != nil || string(data) != "original bytes" {
				t.Fatal("source changed", err)
			}
			if _, err = os.Stat(dst); !os.IsNotExist(err) {
				t.Fatal("target created")
			}
			close(gate)
			<-disposed
			until := time.Now().Add(time.Second)
			for sumatraPending != nil {
				if err := stopSumatraPreview(); err != nil {
					t.Fatal(err)
				}
				if time.Now().After(until) {
					t.Fatal("cleanup did not drain")
				}
				time.Sleep(time.Millisecond)
			}
			if operation == "close" && !closeSumatraPreview() {
				t.Fatal("cleaned startup still blocks close")
			}
			sumatraClosing = false
		})
	}
}
