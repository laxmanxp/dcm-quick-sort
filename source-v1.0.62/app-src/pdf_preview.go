package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	pdfPreviewMinZoom = 25
	pdfPreviewMaxZoom = 800
)

type PDFRenderRequest struct {
	InputPath      string
	OutputPath     string
	PageIndex      int
	Fit            bool
	Zoom           int
	ViewportWidth  int
	ViewportHeight int
	Generation     uint64
}

type PDFRenderMetadata struct {
	PageCount int
	PageIndex int
	Width     int
	Height    int
}

type PDFRenderResult struct {
	Request PDFRenderRequest
	Meta    PDFRenderMetadata
	Err     error
}

func isPDFPath(path string) bool {
	return strings.EqualFold(filepath.Ext(strings.TrimSpace(path)), ".pdf")
}

func clampPDFZoom(percent int) int {
	if percent < pdfPreviewMinZoom {
		return pdfPreviewMinZoom
	}
	if percent > pdfPreviewMaxZoom {
		return pdfPreviewMaxZoom
	}
	return percent
}

func pdfRendererArgs(scriptPath string, req PDFRenderRequest) []string {
	mode := "zoom"
	if req.Fit {
		mode = "fit"
	}
	return []string{
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-InputPath", req.InputPath,
		"-OutputPath", req.OutputPath,
		"-PageIndex", strconv.Itoa(req.PageIndex),
		"-Mode", mode,
		"-Zoom", strconv.Itoa(clampPDFZoom(req.Zoom)),
		"-ViewportWidth", strconv.Itoa(req.ViewportWidth),
		"-ViewportHeight", strconv.Itoa(req.ViewportHeight),
	}
}

func parsePDFRendererOutput(output string) (PDFRenderMetadata, error) {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "OK|") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 5 {
			return PDFRenderMetadata{}, fmt.Errorf("unexpected PDF renderer result: %q", line)
		}
		vals := make([]int, 4)
		for n := 0; n < 4; n++ {
			v, err := strconv.Atoi(parts[n+1])
			if err != nil {
				return PDFRenderMetadata{}, fmt.Errorf("invalid PDF renderer number %q: %w", parts[n+1], err)
			}
			vals[n] = v
		}
		meta := PDFRenderMetadata{PageCount: vals[0], PageIndex: vals[1], Width: vals[2], Height: vals[3]}
		if meta.PageCount <= 0 || meta.PageIndex < 0 || meta.PageIndex >= meta.PageCount || meta.Width <= 0 || meta.Height <= 0 {
			return PDFRenderMetadata{}, errors.New("PDF renderer returned invalid page metadata")
		}
		return meta, nil
	}
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return PDFRenderMetadata{}, errors.New("PDF renderer returned no result")
	}
	return PDFRenderMetadata{}, fmt.Errorf("PDF renderer failed: %s", trimmed)
}
