# DCM Quick Sort — restored PDF Preview Rotate button

**Date:** 18 November 2026
**Branch:** `feature/shared-flowpdf-pdfium-provider`
**Scope:** development branch only, view-only document rotation.

## Implementation

The native PDF page navigation row now contains a **Rotate ↻** button
(control ID 1062). Each click turns the displayed page clockwise
by 90°, cycling **0 → 90 → 180 → 270 → 0**. The page label shows the
active rotation angle. The angle is retained across pages within the
selected PDF and resets when the user selects another PDF.

Rotation is part of the generation-safe `PDFRenderRequest`, not a
direct mutation of the PDF file. The existing rendering queue
coalesces rapid button clicks, and the preview reset clears the
cursor anchor and pan before requesting the rotated bitmap.

- Shared PDFium one-shot: `-Rotation` flag (0/90/180/270)
- Shared persistent PDFium: optional JSONL `rotation` field
- PDFium native renderer: existing PDFium quarter-turn render capability
- Windows.Data.Pdf rollback: rotate its temporary PNG with
  `System.Drawing.Bitmap.RotateFlip`; Fit and Fit Width calculate
  on-screen geometry from the rotated page dimensions
- Existing source PDFs, file metadata and Move/Undo workflows are
  untouched by rotation

## GitHub Windows CI — passed

All checks below have **completed successfully**:

1. Quick Sort feature-branch Windows workflow:
   [Run 44239762865](https://github.com/laxmanxp/dcm-quick-sort/actions/runs/44239762865)
   - Ordinary Windows GUI build: PASS
   - `pdfium_dev` GUI build: PASS
   - Rotation sequencing, renderer flags, provider policy tests: PASS
   - Deterministic disposable two-page PDF created: PASS
   - **Real Windows.Data.Pdf 0/90/180/270 PNG rendering**
     with dimension swap and source-byte preservation: PASS
2. File Workspace feature-branch Windows workflow:
   [Run 44239597472](https://github.com/laxmanxp/file_workspace/actions/runs/44239597472)
   - Worker rotation/fit/cache/JSONL Go tests: PASS
   - Standard PDFium worker Windows build: PASS
   - Test-only acceptance worker Windows build: PASS

Both workflows use the feature branches, not the release branches.

## Physical GUI acceptance — pending

A native Lenovo acceptance option
`tools/test-ui-pdfium-provider.ps1 -VerifyRotation` now exercises the
actual native Rotate button through four clockwise turns, verifies
rotated page navigation, checks rendered document pixels, preserves
the original PDF hash and returns to 0° before the ordinary
navigation/zoom/Rename/Move/Undo tests.

**This native GUI test has not yet been run on Lenovo against the
rotation changes.** Lenovo's Remote Desktop Commander command
endpoint stopped responding during the first build/test attempt,
even though its directory still reported the machine as online.
Do not claim GUI screenshot acceptance or device installation.

### Recommended physical gate when Lenovo connectivity returns

From a fresh checkout of the two committed feature branches,
rebuild the regular development bundle using
`tools/build-pdfium-development.ps1`, then run the native GUI test
with `-DevDefault -VerifyRotation`. Repeat with
`-WindowsRollback -VerifyRotation`. Verify the receipts and
screenshots. Do not modify application code directly on Lenovo.

## Release decision

The button and both backend rotation contracts have passed hosted
Windows build/unit/real Windows.Data.Pdf rendering checks.
**The physical Win32 GUI and real PDFium rotated screenshot gate
remain pending**, and persistent PDFium remains the development
default only. No release-branch or installed-application changes
were made.
