package main

import (
	"errors"
	"fmt"
	"math"
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
	Rotation       int // clockwise degrees: 0, 90, 180, or 270 (preview only)
	Fit            bool
	FitWidth       bool
	Zoom           int
	ViewportWidth  int
	ViewportHeight int
	Generation     uint64
	Neighbor bool // adjacent page prefetch; never becomes the active page automatically
}

type PDFRenderMetadata struct {
	PageCount    int
	PageIndex    int
	Width        int
	Height       int
	ScalePercent int
}

type PDFRenderResult struct {
	Request PDFRenderRequest
	Meta    PDFRenderMetadata
	Err     error
}

// PDFCursorAnchor records a document-space point under a real mouse wheel
// gesture. It is not tied to a particular renderer or window DPI.
type PDFCursorAnchor struct {
 CursorX, CursorY int32
 FractionX, FractionY float64
 Generation uint64
 Valid bool
}

// pdfCaptureCursorAnchor returns false for pointer positions outside the
// displayed PDF page. The origin uses Quick Sort's centered-pan coordinates.
func pdfCaptureCursorAnchor(cursorX, cursorY, oldWidth, oldHeight, clientWidth, clientHeight, contentTop, panX, panY int32) PDFCursorAnchor {
 if oldWidth<=0||oldHeight<=0||clientWidth<=0||clientHeight<=0{return PDFCursorAnchor{}}
 originX := float64(clientWidth-oldWidth)/2 + float64(panX)
 originY := float64(contentTop)+float64(clientHeight-contentTop-oldHeight)/2+float64(panY)
 fx:=(float64(cursorX)-originX)/float64(oldWidth)
 fy:=(float64(cursorY)-originY)/float64(oldHeight)
 if fx<0||fx>1||fy<0||fy>1{return PDFCursorAnchor{}}
 return PDFCursorAnchor{CursorX:cursorX,CursorY:cursorY,FractionX:fx,FractionY:fy,Valid:true}
}

// pdfRestoreCursorPan computes the offset required to keep the document point
// beneath the cursor fixed after a different-size PDFium bitmap arrives.
// For pages smaller than the viewport, empty margins permit limited panning.
func pdfRestoreCursorPan(anchor PDFCursorAnchor, newWidth, newHeight, clientWidth, clientHeight, contentTop int32)(int32,int32) {
 if !anchor.Valid||newWidth<=0||newHeight<=0{return 0,0}
 baseX:=float64(clientWidth-newWidth)/2
 baseY:=float64(contentTop)+float64(clientHeight-contentTop-newHeight)/2
 x:=int32(math.Round(float64(anchor.CursorX)-baseX-anchor.FractionX*float64(newWidth)))
 y:=int32(math.Round(float64(anchor.CursorY)-baseY-anchor.FractionY*float64(newHeight)))
 return pdfClampAnchorPan(newWidth,clientWidth,x),pdfClampAnchorPan(newHeight,clientHeight-contentTop,y)
}

// A coalesced render request can add a generation while a previous worker
// finishes. An anchor captured for the latest wheel gesture still belongs to
// the subsequently scheduled render, but never to a stale result.
func pdfAnchorApplies(anchor PDFCursorAnchor, renderGeneration, desiredGeneration uint64) bool {
 return anchor.Valid && anchor.Generation>0 &&
  renderGeneration==desiredGeneration && anchor.Generation<=renderGeneration
}

func pdfClampAnchorPan(content,viewport,pan int32)int32 {
 if content<=0||viewport<=0{return 0}
 limit:=(content-viewport)/2
 if limit<0{limit=-limit}
 if pan>limit{return limit}
 if pan< -limit{return -limit}
 return pan
}

// Rotation is view-only: it never writes PDF page data or changes filenames.
func normalizePDFPreviewRotation(degrees int) int {
	degrees %= 360
	if degrees < 0 {
		degrees += 360
	}
	return degrees
}

func nextPDFPreviewRotation(degrees int) int {
	return normalizePDFPreviewRotation(degrees + 90)
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
	if req.FitWidth {
		mode = "fitwidth"
	} else if req.Fit {
		mode = "fit"
	}
	return []string{
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-InputPath", req.InputPath,
		"-OutputPath", req.OutputPath,
		"-PageIndex", strconv.Itoa(req.PageIndex),
		"-Rotation", strconv.Itoa(normalizePDFPreviewRotation(req.Rotation)),
		"-Mode", mode,
		"-Zoom", strconv.Itoa(clampPDFZoom(req.Zoom)),
		"-ViewportWidth", strconv.Itoa(req.ViewportWidth),
		"-ViewportHeight", strconv.Itoa(req.ViewportHeight),
	}
}

// The shared PDFium worker uses the same OK|... output contract as
// pdf_render.ps1, but accepts native flags rather than PowerShell switches.
func pdfiumRendererArgs(req PDFRenderRequest) []string {
 mode := "zoom"
 if req.FitWidth {mode="fitwidth"} else if req.Fit {mode="fit"}
 args:=[]string{
  "-InputPath",req.InputPath,
  "-OutputPath",req.OutputPath,
  "-PageIndex",strconv.Itoa(req.PageIndex),
 }
 // Older one-shot workers do not understand -Rotation. Preserve their
 // zero-degree CLI contract while still forwarding actual quarter-turns
 // to rotation-aware workers.
 if degrees:=normalizePDFPreviewRotation(req.Rotation);degrees!=0 {
  args=append(args,"-Rotation",strconv.Itoa(degrees))
 }
 return append(args,
  "-Mode",mode,
  "-Zoom",strconv.Itoa(clampPDFZoom(req.Zoom)),
  "-ViewportWidth",strconv.Itoa(req.ViewportWidth),
  "-ViewportHeight",strconv.Itoa(req.ViewportHeight),
 )
}

func parsePDFRendererOutput(output string) (PDFRenderMetadata, error) {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "OK|") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 5 && len(parts) != 6 {
			return PDFRenderMetadata{}, fmt.Errorf("unexpected PDF renderer result: %q", line)
		}
		vals := make([]int, len(parts)-1)
		for n := 0; n < len(vals); n++ {
			v, err := strconv.Atoi(parts[n+1])
			if err != nil {
				return PDFRenderMetadata{}, fmt.Errorf("invalid PDF renderer number %q: %w", parts[n+1], err)
			}
			vals[n] = v
		}
		meta := PDFRenderMetadata{PageCount: vals[0], PageIndex: vals[1], Width: vals[2], Height: vals[3]}
		if len(vals) >= 5 {
			meta.ScalePercent = vals[4]
		}
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

// pdfCompletionOwnsWorker is independent of the current desired generation:
// a coalesced zoom/page change supersedes the desired result but not its worker.
// Stopping/reopening a selection revokes that worker's scheduler ownership.
func pdfCompletionOwnsWorker(busy bool, workerGeneration, completedGeneration uint64) bool {
	return busy && workerGeneration != 0 && workerGeneration == completedGeneration
}


// pdfShutdownCoordinator is owned exclusively by the Windows GUI thread.
// A PDF selection may be stopped while its renderer is still executing, so
// outstanding work cannot be inferred from pdfPreview.busy or active alone.
// Count a worker until its result has been consumed AND its cache cleaned.
type pdfShutdownCoordinator struct {
	outstanding int
	closing     bool
}

func (s *pdfShutdownCoordinator) workerStarted() bool {
	if s.closing {
		return false // no new renderer may be launched after close was requested
	}
	s.outstanding++
	return true
}

// requestClose returns deferClose and firstRequest.
// A repeated WM_CLOSE must not invalidate the same selection twice.
func (s *pdfShutdownCoordinator) requestClose() (bool, bool) {
	if s.outstanding == 0 {
		return false, false
	}
	if s.closing {
		return true, false
	}
	s.closing = true
	return true, true
}

// resultConsumed returns true exactly once, when the last previously launched
// worker has completed AND the GUI consumed/discarded its result.
func (s *pdfShutdownCoordinator) resultConsumed() bool {
	if s.outstanding <= 0 {
		return false // existing tests can inject synthetic result completions
	}
	s.outstanding--
	if s.outstanding == 0 && s.closing {
		s.closing = false
		return true
	}
	return false
}
