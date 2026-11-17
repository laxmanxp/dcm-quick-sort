# Shared FlowPDF/PDFium worker acceptance — 2026-11-17

## Scope

This is an **opt-in backend integration**, not a release-default provider
switch and not a completed smooth-zoom qualification of the DCM native viewer.

- File Workspace source: `laxmanxp/file_workspace`,
  `feature/flowpdf-shared-provider-contract` at `32920beccdc95570ab399890ed65a8365074f126`.
- Quick Sort source: `laxmanxp/dcm-quick-sort`,
  `feature/shared-flowpdf-pdfium-provider`.
- Acceptance device: `Laxman_Lenovo`.
- Isolated File Workspace checkout:
  `D:\FlowverseReportsRepo\fileworkspace-pdf-zoom-acceptance-20261117`.
- Isolated Quick Sort checkout:
  `D:\FlowverseReportsRepo\dcm-pdfium-provider-acceptance-20261117`.
- Managed PDFium: `C:\Flowverse\runtime\pdfium\bin\pdfium.dll`.

## Tests observed

1. `go test ./cmd/flowpdf-preview-worker -count=1` — PASS
2. Built `bin\flowpdf-preview-worker.exe` from File Workspace — PASS
3. Generated a real three-page mixed-format PDF and rendered its second
   page in `fitwidth` mode using the worker — PASS
4. Worker stdout:
   `OK|3|1|800|1036|98`, corresponding output PNG exists — PASS
5. Quick Sort provider args and metadata parser tests — PASS
6. Quick Sort provider selection and missing-worker tests — PASS
7. Quick Sort native worker integration smoke (`TestSharedPDFiumWorkerSmoke`)
   — PASS; small fixture round-trip approximately 0.10 seconds
8. Quick Sort candidate Windows binary build — PASS
9. Full Quick Sort `go test ./... -count=1` with PDFium opt-in environment
   — PASS; approx. 34.8 seconds

Binary under `D:\FlowverseReportsRepo\dcm-pdfium-provider-acceptance-20261117\bin\DCMQuickSort-pdfium-candidate.exe`.

## Not yet qualified

- Physical GUI PDF preview in DCM, including Ctrl+wheel mouse anchoring
- Large scanned PDFs, graphics-heavy PDFs, protected/malformed PDFs
- File locks on Rename/Move/Undo under PDFium provider
- Rapid document switching, renderer crash/timeout and shutdown
- Persistent shared process and cache across applications
- Packaging and distribution (managed runtime, worker executable and notices)

**Do not merge or change the PDF default** before the missing GUI and lifecycle
gates pass. The existing PowerShell/Windows.Data.Pdf provider and Sumatra route
remain unmodified by default.


## Native PDFium preview acceptance continuation (17 November 2026)

**Tested on Laxman_Lenovo, isolated candidate only.** The native GUI
acceptance harness and the cursor-anchoring candidate were added on the
feature branch. The installed and release applications were not changed.

- Native harness: `tools/test-ui-pdfium-provider.ps1`
- Owned Ctrl-wheel input adapter: `tests/native/dcm-pdfium-wheel-input.cs`
- Pointer-zoom geometry: `app-src/pdf_preview.go`,
  `app-src/pdf_preview_windows.go`, `app-src/main_windows.go`
- Pure geometry/generation tests: `app-src/pdf_cursor_anchor_test.go`
- Final product-code commit verified: `3fd13cde16fd39057440dd8448ea9b14a585f6fd`
- Final acceptance harness instrumentation commit: `68468d6775000679b0a3333cbe7747a718765e32`

### Observed passing gates

- First/last page controls and real teal/orange page pixels: PASS
- Return from page 2 to page 1 and PDF reopening: PASS
- Native toolbar Zoom In, owned Ctrl+wheel, and PDFium render completion: PASS
- Ctrl+wheel gesture coalesced into a final render without crash: PASS
- Exclusive read/write handle acquisition after PDF deselection: PASS
- Normal GUI shutdown: PASS, exit 0 without forced termination
- Full DCM `go test ./... -count=1`: PASS (approximately 34.4 seconds)
- Native candidate rebuild: PASS

The final native GUI acceptance run completed with `PASSED=True`,
`NORMAL_EXIT=True`, `GUI_EXIT=0`.

Evidence locations on Lenovo:
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261117-162450-34126c3`
(initial controlled UI);
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261117-162820-483dd3a`
(owned Ctrl+wheel);
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261117-163732-5d3302c`
(rebuilt cursor anchor);
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261117-163936-2ac68f9`
(horizontal pixel metric). Each has `receipt.json`, renderer trace, and
captured PNG evidence. Images are **not yet independently human-reviewed**.

### Cursor anchoring scope and unresolved gates

The native PDFium provider now computes the pointer's fractional position in
the currently visible bitmap and reapplies it to the completed render, with
generation-safe handling for coalesced wheel requests. New pure-Go tests pass.

A provisional screenshot metric found horizontal fractional drift
`0.039770114942528745`, satisfying its `< 0.04` diagnostic threshold.
This metric uses a coloured-content bounding box and **does not establish
complete two-axis cursor-anchor acceptance**; cropping, viewport constraints,
different DPI arrangements and other page layouts remain untested. Do not
describe pointer-zoom smoothness as fully qualified.

**Still pending**: real Rename/Move/Undo workflow on the PDFium provider
(the exclusive file-release check is narrower); scans/graphics/protected/
malformed PDFs; repeated tab changes; memory-pressure measurements; crash
and timeout containment; long-lived PDFium worker design and packaging
security/redistribution compliance.

No PDF default-provider switch, release merge, installer change, or
production application modification was made.
