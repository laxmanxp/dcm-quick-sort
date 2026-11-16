//go:build windows

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Real Windows.Data.Pdf sizing contract. PdfPageRenderOptions destination
// dimensions are in device-independent pixels (DIPs), while PNG decode
// exposes physical bitmap pixels at the target display DPI.
// No renderer substitution, display-setting mutation or synthetic DPI change.
func TestDCM34PDFBitmapDPI(t *testing.T) {
	evidence := os.Getenv("DCM_PDF_DPI_EVIDENCE_ROOT")
	if evidence != "" && !filepath.IsAbs(evidence) {
		t.Fatal("PDF DPI evidence directory must be absolute")
	}
	if evidence == "" {
		evidence = t.TempDir()
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(evidence, "receipt.json")
	if _, err := os.Stat(receiptPath); err == nil {
		t.Fatal("refusing to overwrite PDF DPI receipt")
	}
	observations := []map[string]any{}
	receipt := map[string]any{
		"schema": 1, "milestone": "DCM-PDF-RENDER-SIZE-01",
		"scope": "Real WinRT PDF rasterization and decoded PNG dimensions at desktop DPI; no window GUI, synthetic DPI settings, renderer change or provider registration mutation",
		"started_utc": time.Now().UTC().Format(time.RFC3339Nano),
		"passed": false, "feature_accepted": false, "user_test": "not_recorded",
		"desktop_dpi_queried": false, "display_settings_mutated": false,
	}
	defer func() {
		receipt["passed"] = !t.Failed()
		receipt["observations"] = observations
		receipt["count"] = len(observations)
		receipt["finished_utc"] = time.Now().UTC().Format(time.RFC3339Nano)
		dcm34ComponentJSON(t, receiptPath, receipt)
		t.Logf("PDF_DPI_RECEIPT=%s", receiptPath)
	}()
	desktop, _, err := user32.NewProc("GetDesktopWindow").Call()
	if desktop == 0 {
		t.Fatalf("GetDesktopWindow failed: %v", err)
	}
	dpi, _, err := user32.NewProc("GetDpiForWindow").Call(desktop)
	if dpi < 96 || dpi > 480 {
		t.Fatalf("unexpected desktop window DPI %d: %v", dpi, err)
	}
	receipt["desktop_dpi"] = dpi
	receipt["desktop_scale"] = float64(dpi) / 96.0
	receipt["desktop_dpi_queried"] = true
	t.Logf("desktop DPI=%d (scale %.3f), WinRT DestinationWidth/Height units=DIP", dpi, float64(dpi)/96.0)

	inputPath := filepath.Join("..", "tests", "fixtures", "sumatra-scroll.pdf")
	source, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	const pinnedSHA = "05304a98a830954b4f5bfedf0187e78aa847248a80b0ca2ba2b5b5f843352bb0"
	if got := fmt.Sprintf("%x", sha256.Sum256(source)); got != pinnedSHA {
		t.Fatalf("pinned PDF changed: %s", got)
	}
	receipt["source_sha256"] = pinnedSHA
	input := filepath.Join(evidence, "dcm-pdf-dpi-control.pdf")
	if err := os.WriteFile(input, source, 0600); err != nil {
		t.Fatal(err)
	}
	renderer := filepath.Join(evidence, "production-pdf-renderer.ps1")
	if err := os.WriteFile(renderer, bundledPDFRendererScript, 0600); err != nil {
		t.Fatal(err)
	}
	receipt["renderer_sha256"] = fmt.Sprintf("%x", sha256.Sum256(bundledPDFRendererScript))
	shell, err := findPDFPowerShell()
	if err != nil {
		t.Fatal(err)
	}

	// Multiple independent modes and viewport sizes distinguish a genuine
	// logical-to-physical conversion from a coincidence at 612x1000.
	for _, tc := range []struct {
		name      string
		fit       bool
		fitWidth  bool
		zoom      int
		viewportW int
		viewportH int
	}{
		{"zoom-100", false, false, 100, 612, 1000},
		{"fit-page", true, false, 100, 500, 700},
		{"fit-width", false, true, 100, 500, 700},
		{"zoom-100-repeat", false, false, 100, 612, 1000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pngPath := filepath.Join(evidence, tc.name+".png")
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, shell, pdfRendererArgs(renderer, PDFRenderRequest{
				InputPath: input, OutputPath: pngPath, PageIndex: 0,
				Fit: tc.fit, FitWidth: tc.fitWidth, Zoom: tc.zoom,
				ViewportWidth: tc.viewportW, ViewportHeight: tc.viewportH,
			})...)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
			start := time.Now()
			output, runErr := cmd.CombinedOutput()
			obs := map[string]any{
				"case": tc.name, "duration_ms": time.Since(start).Milliseconds(),
				"viewport_width": tc.viewportW, "viewport_height": tc.viewportH,
				"fit": tc.fit, "fit_width": tc.fitWidth, "zoom": tc.zoom,
				"png": pngPath, "renderer_exit_ok": runErr == nil,
				"timed_out": ctx.Err() != nil,
			}
			observations = append(observations, obs)
			if ctx.Err() != nil || runErr != nil {
				obs["failure"] = fmt.Sprintf("renderer=%v ctx=%v output=%s", runErr, ctx.Err(), strings.TrimSpace(string(output)))
				t.Fatalf("real WinRT render failed: %s", obs["failure"])
			}
			meta, err := parsePDFRendererOutput(string(output))
			if err != nil || meta.PageCount != 2 || meta.PageIndex != 0 {
				t.Fatalf("invalid PDF output metadata=%+v err=%v", meta, err)
			}
			encoded, err := os.ReadFile(pngPath)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := png.Decode(bytes.NewReader(encoded))
			if err != nil {
				t.Fatal(err)
			}
			bounds := decoded.Bounds()
			if bounds.Empty() {
				t.Fatal("decoded PNG is empty")
			}
			// Hash decoded pixels for independent forensic comparison but DO
			// NOT turn historical intermittent pixel differences into a PASS.
			rgba := image.NewRGBA(bounds)
			draw.Draw(rgba, bounds, decoded, bounds.Min, draw.Src)
			expectedW := int(math.Round(float64(meta.Width) * float64(dpi) / 96.0))
			expectedH := int(math.Round(float64(meta.Height) * float64(dpi) / 96.0))
			obs["metadata_width_dip"] = meta.Width
			obs["metadata_height_dip"] = meta.Height
			obs["bitmap_width_pixels"] = bounds.Dx()
			obs["bitmap_height_pixels"] = bounds.Dy()
			obs["expected_width_pixels_from_desktop_dpi"] = expectedW
			obs["expected_height_pixels_from_desktop_dpi"] = expectedH
			obs["pixel_ratio_width"] = float64(bounds.Dx()) / float64(meta.Width)
			obs["pixel_ratio_height"] = float64(bounds.Dy()) / float64(meta.Height)
			obs["png_sha256"] = fmt.Sprintf("%x", sha256.Sum256(encoded))
			obs["decoded_pixel_sha256"] = fmt.Sprintf("%x", sha256.Sum256(rgba.Pix))
			obs["png_bytes"] = len(encoded)
			// A single physical pixel can be rounded differently by WinRT
			// at fractional DPI and fractional logical destination sizes.
			deltaW := bounds.Dx() - expectedW
			deltaH := bounds.Dy() - expectedH
			if deltaW < -2 || deltaW > 2 || deltaH < -2 || deltaH > 2 {
				obs["failure"] = "actual PNG dimensions are not explained by display DPI"
				t.Errorf("%s: DIP %dx%d at %d DPI -> expected ~%dx%d PNG but decoded %dx%d", tc.name, meta.Width, meta.Height, dpi, expectedW, expectedH, bounds.Dx(), bounds.Dy())
			}
			t.Logf("%s: requested=%dx%d DIP actual=%dx%d px DPI=%d ratio=%.4f/%.4f", tc.name, meta.Width, meta.Height, bounds.Dx(), bounds.Dy(), dpi, obs["pixel_ratio_width"], obs["pixel_ratio_height"])
		})
	}
	actual, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(actual, source) {
		t.Fatal("PDF fixture was altered by render")
	}
	path, err := syscall.UTF16PtrFromString(input)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(path, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("PDF source remains locked after render: %v", err)
	}
	if err := syscall.CloseHandle(h); err != nil {
		t.Fatal(err)
	}
	receipt["fixture_preserved"] = true
	receipt["source_file_exclusively_released"] = true
	receipt["original_source_unchanged"] = dcm34ComponentHash(t, inputPath) == pinnedSHA
	// No Windows DPI or display scale settings are written by the test.
}
