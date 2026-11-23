# PDF Single Page / Continuous preview — development qualification

Date: 2026-11-23

## Scope
QuickSort's owned PDFium/WinRT preview now exposes Single and Continuous viewing modes in its native toolbar. Continuous mode draws the active page and up to two prefetched neighboring page bitmaps in a clipped vertical stack. Wheel scrolling crosses page boundaries without loading the whole PDF into memory. The active page remains highest priority in the existing render scheduler; the neighboring worker uses generation and source-path checks.

## Implemented
- Responsive native "Single / Continuous" mode control (single page remains the default).
- Fit-width continuous reading, page gap, clipping below the toolbar, previous/next visible-page cache, and page-boundary wheel handling.
- Page indicator and status refresh on cached-page promotion.
- Fallback to normal page navigation when adjacent-page prefetch fails.
- Existing PDF zoom, rotation, page navigation and single-page rendering retained.
- Controlled PDF trace events for diagnosing prefetch and wheel navigation; logs are opt-in.

## Tests and measured evidence
- Full Lenovo Windows Go regression: `go test ./... -count=1` passed.
- Windows development build: `go build` passed.
- Disposable `tests/fixtures/sumatra-scroll.pdf` has 2 pages (`/Count 2`).
- Native control smoke test: "Single" toggled to "Continuous" and remained active.
- Opt-in trace confirmed neighbor page 2 was requested, completed without a renderer error, and was cached. A long wheel sequence recorded a transition from `continuous-wheel-page-1` to `continuous-wheel-page-2` without a new main-page render.
- Found and corrected a stale page-number field/status during cached promotion, followed by another full passing Go suite.
- On Lenovo the final separate acceptance executable is `C:\DCMQuickSortCollisionAcceptance\dcm-pdf-continuous-acceptance-dev.exe`, verified running as PID 8680.

## Outstanding
User visual acceptance is pending. Check multi-page boundary appearance, smoothness on mouse/touchpad, page indicator, divider resizing, zoom, both rotations, and transition back to Single Page. Evaluate heterogeneous page sizes and large PDFs before release. This is not a claim of parity with SumatraPDF's full continuous reading engine.

The installed QuickSort was not replaced. GitHub feature branch `feature/quicksort-two-pane-history-tabs` remains the source of truth.
