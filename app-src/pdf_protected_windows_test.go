//go:build windows

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Real Windows.Data.Pdf invocation, not a mocked decoder or an installed viewer.
// Each invocation is bounded and uses disposable paths and the bundled production script.
func TestDCM34PDFProtectedRenderer(t *testing.T) {
	shell, err := findPDFPowerShell()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	script := filepath.Join(root, "pdf_render.ps1")
	if err := os.WriteFile(script, bundledPDFRendererScript, 0600); err != nil {
		t.Fatal(err)
	}
	locked := dcmProtectedFixture(t)
	good, err := os.ReadFile(filepath.Join("..", "tests", "fixtures", "sumatra-scroll.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(good)) != "05304a98a830954b4f5bfedf0187e78aa847248a80b0ca2ba2b5b5f843352bb0" {
		t.Fatal("valid control fixture changed")
	}
	for _, f := range []struct {
		name string
		data []byte
	}{{"locked.pdf", locked}, {"valid.pdf", good}} {
		if err := os.WriteFile(filepath.Join(root, f.name), f.data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for i, tc := range []struct {
		name, file string
		refusal    bool
	}{
		{"valid-before", "valid.pdf", false}, {"protected-no-password", "locked.pdf", true},
		{"valid-recovery", "valid.pdf", false}, {"protected-repeat", "locked.pdf", true},
		{"valid-second-recovery", "valid.pdf", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(root, fmt.Sprintf("page-%d.png", i))
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, shell, pdfRendererArgs(script, PDFRenderRequest{InputPath: filepath.Join(root, tc.file), OutputPath: out, Fit: true, Zoom: 100, ViewportWidth: 612, ViewportHeight: 1000})...)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
			started := time.Now()
			output, runErr := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("renderer timed out: %v", ctx.Err())
			}
			meta, parseErr := parsePDFRendererOutput(string(output))
			t.Logf("duration=%s process_error=%v metadata=%+v output=%s", time.Since(started), runErr, meta, strings.TrimSpace(string(output)))
			if tc.refusal {
				if runErr == nil || parseErr == nil || meta != (PDFRenderMetadata{}) {
					t.Fatalf("protected document falsely ready: process=%v parse=%v metadata=%+v", runErr, parseErr, meta)
				}
				if _, err := os.Stat(out); !os.IsNotExist(err) {
					t.Fatalf("refusal created output page: %v", err)
				}
			} else {
				if runErr != nil || parseErr != nil || meta.PageCount != 2 || meta.PageIndex != 0 {
					t.Fatalf("valid control failed: process=%v parse=%v meta=%+v", runErr, parseErr, meta)
				}
				image, err := os.ReadFile(out)
				if err != nil {
					t.Fatal(err)
				}
				cfg, err := png.DecodeConfig(bytes.NewReader(image))
				if err != nil || cfg.Width != meta.Width || cfg.Height != meta.Height {
					t.Fatalf("invalid actual page PNG: %+v %v", cfg, err)
				}
			}
		})
	}
	for _, f := range []struct {
		name string
		data []byte
	}{{"locked.pdf", locked}, {"valid.pdf", good}} {
		actual, err := os.ReadFile(filepath.Join(root, f.name))
		if err != nil || !bytes.Equal(actual, f.data) {
			t.Fatal("renderer changed input bytes")
		}
		// Exclusive-open proof uses the native API; no share-read/write/delete.
		path, err := syscall.UTF16PtrFromString(filepath.Join(root, f.name))
		if err != nil {
			t.Fatal(err)
		}
		h, err := syscall.CreateFile(path, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
		if err != nil {
			t.Fatalf("renderer retained fixture lock: %v", err)
		}
		if err := syscall.CloseHandle(h); err != nil {
			t.Fatal(err)
		}
	}
}
