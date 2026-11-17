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


## Native Rename / Move / Undo acceptance — Lenovo

The isolated PDFium candidate passed real Win32 GUI file-operation acceptance
on **Laxman_Lenovo** using only disposable fixtures. No release build,
installed application, or user document was changed.

- Source branch: `feature/shared-flowpdf-pdfium-provider`
- Test harness/qualification commit: `a212a139fd32ef41a218726570c0f39b207be9da`
- Device evidence directory:
  `D:\FlowverseReportsRepo\dcm-pdfium-gui-20261117-165735-4034632`
- Result: `PASSED=True`, `NORMAL_EXIT=True`, `GUI_EXIT=0`
- Native evidence: `receipt.json`, `pdf-trace.jsonl`, and ten
  full-window PNG captures (visual review not independently completed).

The test selected the active two-page PDFium-rendered document, renamed
`08-preview.pdf` to `undo-preview.pdf`, verified the new file's byte
hash and completed journal entry, reopened its page, then used the
application's own Undo control to restore the original path and verify the
journal state `undone`.

It subsequently moved the PDF to the disposable `Vault\Accepted` location,
verified the completed move journal and matching bytes, opened the document
from the destination, then used the application's own Undo control to restore
the source path. The destination selection cleared correctly. The final
source folder contained the original `08-preview.pdf` and the unchanged
three TXT fixtures. The destination was empty. Both operation journal records
ended in the `undone` state.

The worker's ability to release file handles for these operations is supported
by the successful on-disk transitions and explicit exclusive-file access checks.

**Qualified scope:** small deterministic two-page fixture, regular operation
completion, existing native preview, local filesystem. **Still pending:**
large scanned and graphics-heavy PDFs; corrupt/password-protected documents;
failure injection and render-timeout handling; repeated tab/file switching
under memory pressure; persistent worker performance and packaging; complete
two-axis cursor anchoring and independent screenshot review.

Do not make PDFium the release default based solely on this scope.


## PDFium resilience and memory acceptance — 17 November 2026

Machine: `Laxman_Lenovo`. Worker from File Workspace's
`feature/flowpdf-shared-provider-contract`; DCM branch
`feature/shared-flowpdf-pdfium-provider`. All inputs are versioned
synthetic fixtures or test-generated PDFs.

### Native worker decoder gate

Added `app-src/pdfium_resilience_windows_test.go`.
Test `TestFlowPDFiumWorkerResilience`: PASS.

- Valid control before failures: PASS (819 ms cold measured)
- Damaged PDF: explicit load error, no output PNG, input intact: PASS (135 ms)
- Valid second page after damaged failure: PASS (483 ms)
- Encrypted PDF without a password: explicit refusal, no output PNG: PASS (135 ms)
- Valid control after protected failure: PASS (304 ms)
- Ten subsequent renders alternating pages and zoom percentages: PASS
- All synthetic PDFs retained original byte hashes; file access verified after worker exit

`TestFlowPDFiumWorkerLongDocument`: PASS.
Synthetic mixed-layout 300-page PDF renders at indices 0, 149, 299, 0:
130, 146, 175, 211 ms respectively. These are worker-process runtimes
for this document, not general scanned-PDF performance benchmarks.

### Worker process footprint

Added `tools/test-pdfium-worker-memory.ps1`, revised to explicitly
provide `FILEWORKSPACE_RUNTIME_DIR`. The first run failed due to the
missing test runtime setting; it produced no benchmark result. The
corrected 20-invocation run passed.

Evidence: `D:\FlowverseReportsRepo\dcm-pdfium-worker-memory-20261117-170743-e92137\receipt.json`.

- 20/20 individual worker invocations returned usable PNGs and exited
- Median wall time 199 ms; maximum 484 ms
- Maximum **sampled** private memory 38.5 MB
- Render PNG outputs removed after each test; no leftover PNGs
- These observations cannot establish a continuously running worker's
  memory growth or true instantaneous peak; sampling may miss short peaks

### Actual native GUI failure and recovery

Extended `tools/test-ui-pdfium-provider.ps1` to select each malformed or
encrypted PDF inside the disposable Quick Sort GUI, confirm the
`result-unavailable` trace event, then select the valid PDF and confirm
real rendered teal pixels. The first expanded run failed because the
test incorrectly required exclusive file access while the valid PDF
was still actively previewed. That assertion was removed. The corrected
full GUI test passed.

Evidence: `D:\FlowverseReportsRepo\dcm-pdfium-gui-20261117-171115-00e7eda\receipt.json`.

- Candidate branch commit `5d1a6a2ff15c7033188a0ce331e1b37a4faa89b6`
- `PASSED=True`, `NORMAL_EXIT=True`, `GUI_EXIT=0`
- Damaged PDF: real worker failure observed, input hash unchanged, exclusive
  file access after failure: PASS
- Valid PDF renders following damage: PASS
- Protected PDF: real worker failure observed, input hash unchanged,
  exclusive file access after failure: PASS
- Valid PDF renders following protected refusal: PASS
- Rename/Undo, Move/Undo and final journal integrity remained PASS
- 12 saved native GUI captures; not independently visually reviewed

**Remaining:** raster-heavy scanned and graphics-intensive PDFs,
extended tab/file switching under memory pressure, controlled
renderer crash/timeout injection, persistent worker service benchmark,
full two-axis/DPI cursor-anchor certification, packaging and license
audit. This does not authorize making PDFium the default provider.
