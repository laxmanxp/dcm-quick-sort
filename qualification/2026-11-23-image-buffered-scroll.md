# DCM QuickSort raster-image scroll flicker — buffered rendering

Date: 2026-11-23

## Cause
QuickSort raster previews previously invalidated their complete Win32 preview window with background erasure on every wheel/drag event. The `WM_PAINT` handler painted the client area white before GDI+ rendered the updated image. Repeated wheel and drag messages could expose that empty intermediate white frame.

## Changes
- For standalone raster images only, `WM_ERASEBKGND` returns handled; wheel and drag invalidations do not request background erase.
- New `paintImagePreviewBuffered` paints the white background, opaque image toolbar strip, and GDI+ bitmap to a compatible memory DC and bitmap, then presents it through one `BitBlt`.
- Compatible bitmap and DC are released after each frame, including failure paths. If allocation fails, the normal direct paint fallback remains available.
- PDF, DOCX, HTML, Excel and PowerPoint maintain their previous drawing paths.
- Image zoom and 90-degree preview-only rotation controls are unchanged.

## Lenovo verification
- `go test ./... -count=1`: passed.
- `go build` for `dcm-image-buffered-scroll-dev.exe`: passed.
- Disposable 900x500 PNG opened in an isolated development process.
- Guarded native smoke test verified `WM_ERASEBKGND` returns handled in raster-image preview, then delivered 35 wheel events without the preview process exiting.
- Original PNG SHA-256 unchanged.
- Manual visual smoothness assessment, longer drag tests, and large photo performance remain acceptance gates.

The installed QuickSort is unchanged. Development branch: `feature/quicksort-two-pane-history-tabs`.
