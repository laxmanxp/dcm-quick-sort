# PDF viewport refresh after pane resizing — Lenovo qualification

Date: 2026-11-23

## Reproduced cause
QuickSort's main-window and preview child `WM_SIZE` handlers changed the preview client rectangle and rearranged the PDF toolbar but did not request a new PDF render. Since `PDFRenderRequest` records `ViewportWidth` and `ViewportHeight`, Fit Width and Fit Page could continue displaying a bitmap sized for the old pane.

## Correction
- Introduced `pdfViewportNeedsRerender` with table-driven tests to compare the effective render viewport.
- Added `pdfViewportResizeTimerID` with 160ms coalescing. It responds to main and child resize events and the splitter commit path. Dragging does not synchronously render pages.
- Track the last actually requested PDF viewport dimensions and request another generation only after they change.
- Cancel pending refresh on PDF unload. Other previewers are unaffected.
- The existing generation-safe PDF render queue performs the re-render, preserving the PDF's fit/zoom/rotation mode.

## Lenovo evidence
On the isolated `dcm-pdf-resize-refresh-dev.exe` build:
- Full Windows Go regression suite: `go test ./... -count=1` passed.
- `go build` passed.
- With `tests/fixtures/sumatra-scroll.pdf`, opt-in PDF trace showed initial generation 2 with `worker-started`, `worker-finished`, `result-applied`, all successful.
- Resizing the main window to 1100 x 760 produced a second successful render at generation 3.
- Requesting the **same dimensions again** produced no third render; the trace still had two successful render results.

## Remaining acceptance
Physical manual divider dragging and visual inspection of Fit Page, Fit Width, Continuous, rotation and zoom with real PDFs remain necessary. The installed QuickSort version was not replaced. GitHub feature branch is the source of truth.
