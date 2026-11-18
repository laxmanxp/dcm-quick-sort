# DCM Quick Sort — PDFium development default acceptance

**Acceptance date:** 18 November 2026 (Asia/Kolkata)
**Device:** Laxman_Lenovo
**Branch:** `feature/shared-flowpdf-pdfium-provider`
**Decision:** Qualified as the default **only for tagged development builds**.
**Production default:** Windows.Data.Pdf (unchanged).

## Product change

`go build -tags pdfium_dev` compiles the development-default policy;
ordinary `go build` leaves the Windows.Data.Pdf default unchanged.

A tagged development executable now discovers its colocated
`flowpdf-preview-worker-persistent.exe`. It automatically selects
persistent PDFium only if that qualified worker and a managed PDFium
DLL are available. Missing worker/DLL falls back to Windows.Data.Pdf.
An explicit `DCM_PDF_PROVIDER=pdfium` still fails closed when the
worker is absent; `DCM_PDF_PROVIDER=windows` forces immediate rollback;
`DCM_FLOWPDF_PERSISTENT=0` selects one-shot PDFium.

The managed DLL can be supplied by `FILEWORKSPACE_PDFIUM`,
`FILEWORKSPACE_RUNTIME_DIR`, or the documented development location
`C:\Flowverse\runtime\pdfium\bin\pdfium.dll`. The latter is inherited
by the worker only, not written into the parent environment.

A reproducible build script,
`tools/build-pdfium-development.ps1`, builds the shared worker from
the File Workspace development checkout, builds the tagged Quick Sort
application, colocates them, and records source commits and binary
SHA-256 hashes in `pdfium-development-manifest.json`. It does not
replace installed or production binaries.

## Verification on Lenovo

| Gate | Result |
|---|---|
| Untagged provider-policy tests | PASS |
| Tagged provider-policy tests | PASS |
| Missing-worker and missing-runtime selection checks | PASS |
| Forced Windows and one-shot PDFium selection checks | PASS |
| Native GUI automatic PDFium default, with PDF-specific variables absent | PASS |
| Native GUI from reproducible development bundle | PASS |
| Native GUI explicit Windows.Data.Pdf rollback | PASS |
| Full untagged Quick Sort Go suite | PASS — 47.949 s |
| Full `pdfium_dev` tagged Quick Sort Go suite | PASS — 38.895 s |
| Persistent worker normal cleanup at GUI shutdown | PASS |

### Native GUI evidence

The final reproducible development bundle's GUI test:
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261118-014901-79a6adf\receipt.json`

- `PASSED=True`, `NORMAL_EXIT=True`, exit code 0
- 19 native PDF render requests using one worker PID **25484**
- Zero unexpected one-shot fallback events
- Owned worker had exited after Quick Sort closed
- Real PDF navigation, Ctrl+wheel zoom, Rename/Undo, Move/Undo,
  synthetic damaged/protected PDF recovery, byte/journal integrity: PASS
- 12 captured GUI screenshots, not independently human-reviewed

The explicit Windows rollback GUI test:
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261118-014741-c7f87c2\receipt.json`

- `PASSED=True`, `NORMAL_EXIT=True`, exit code 0
- Used the legacy Windows renderer and completed the same core
  navigation, file-operation, and recovery acceptance

The initial Windows rollback run failed a page-2 color assertion
because the test captured before Windows.Data.Pdf had applied the
rendered bitmap. The GUI harness was corrected to wait for the
`result-applied` trace event. The corrected rerun passed.

The first tagged full-suite run also revealed an outdated assumption
in `TestPersistentPDFiumOptInIsStrict`: an empty PDF-provider variable
was previously treated as non-PDFium, while it now intentionally
means the development default. The test was changed to assert
explicit `DCM_PDF_PROVIDER=windows`; the full tagged suite then
passed on commit `2c9ba942866278dc6e92f54ab9654589af9ef397`.

### Reproducible development artifact

Bundle location on Lenovo:
`D:\FlowverseReportsRepo\dcm-pdfium-provider-acceptance-20261117\out\pdfium-dev\`.

The manifest records:

- Quick Sort build commit `064fadf51800f80dd8a633450987f68d99785fe9`
- File Workspace worker build commit
  `268afcdc0aa674ebc8542d31633c5033f7593983`
- Worker SHA-256
  `D5A26A7D77B1CF69EC10457CE11DA3902185EF1A69A1F0B63A08298CF2F8D5A1`
- App SHA-256
  `04684922332CFDD10E49C9C9FF1572BA23CE3F9D122CCF53E3797509C48FDA44`

The build and test actions used isolated Lenovo checkouts. Source
changes were committed through GitHub; no on-device source edits,
production installation, release merge, or default release-provider
switch occurred.

## Remaining production-release gates

Persistent PDFium remains **development-default only** until the
following are qualified: stale-render cancellation on fast file
switching; injected worker crash/timeout in the live GUI; sustained
memory/handle/process pressure and shutdown; complete two-axis
mouse-centered zoom across DPI settings; diverse real scanned and
graphics-intensive PDFs; packaging/runtime signing, security boundaries,
third-party notices/licensing and installer acceptance.
