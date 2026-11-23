# DCM QuickSort image preview — zoom and rotation qualification

Date: 2026-11-23

## Review findings
The native raster image viewer already decoded JPG/JPEG, PNG, BMP, GIF and TIFF using its owned GDI+ image stream. Fit, Fit Width, manual zoom and drag pan were available through the application's shared zoom controls. However, there were no native image rotation actions and Ctrl+wheel zoom discarded the existing pan position instead of anchoring to the cursor.

## Changes implemented
- Added an independent, opaque image preview toolbar with Fit, Fit W, minus, 100%, plus, Rotate L and Rotate R controls.
- Added GDI+ preview-only 90-degree left/right rotation using ImageRotateFlip; the source file is not edited or re-encoded.
- Added pointer-anchored Ctrl+wheel zoom for standalone raster images, retaining the document point near the pointer where the image extends beyond the viewport.
- Retained the existing manual zoom range, centered fit modes and drag-to-pan behavior.
- Image fit calculations and the image raster drawing clip now exclude the toolbar.
- The new toolbar is hidden and its rotation state reset when leaving an image. PDF and native Office previewers remain separate.
- Introduced pure-policy tests for pointer anchor pan and 90-degree rotation normalization.

## Lenovo verification
- `go test ./... -count=1`: passed.
- `go build` for `dcm-image-zoom-rotate-dev.exe`: passed.
- Generated a disposable 900 × 500 PNG with visible non-symmetric colored features under `C:\DCMQuickSortCollisionAcceptance\TraceImagePreview\wide-test.png`.
- Opened the fixture in the isolated development executable and verified all seven image toolbar controls were present.
- Native toolbar smoke exercised rotation right/left, Fit Width, zoom in/out, Fit and 100% and verified rotation feedback in the window.
- Source PNG SHA-256 was unchanged after all operations.
- The installed QuickSort version was not replaced.

## Open gates
Manual visual acceptance for pointer-anchored Ctrl+wheel zoom, smoothness, large images, animated GIF policy, narrow preview panes and actual JPEG/TIFF photos remains pending. Source rotation is deliberately preview-only.
