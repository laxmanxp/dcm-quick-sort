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


## Persistent PDFium selection cancellation and fault recovery — Lenovo

Acceptance device: `Laxman_Lenovo`, isolated test checkout
`D:\FlowverseReportsRepo\dcm-pdfium-provider-acceptance-20261117`.
Tested application-code commit
`b3f8d50322b07c8be15ce24c9101a719e6d323f3`.
Corresponding File Workspace worker commit
`cb3ba0dde970c224f7a67ad73acc238744818691`.

### Engineering change

`app-src/pdfium_persistent_windows.go` now tracks the exact active
worker process, request generation and canceled-through boundary with
a separate small mutex. `stopPDFPreview()` can cancel the obsolete
process when switching selections without waiting for the serial IPC
mutex. The boundary also rejects requests that had been queued but
not started. Later generations are unaffected, and an explicitly
canceled request does not perform another persistent retry or invoke
one-shot PDFium fallback.

Cancellation is different from a worker crash: an unexpected crash
still uses supervised restart, and repeated transport timeouts still
use the bounded one-shot fallback. Ordinary decoder refusals remain
errors, not backend switches.

### Native cancellation unit tests

`TestPersistentPDFiumCancelInFlightOnFileSwitch`: PASS; deliberately
stalled worker PID 19900 was canceled in about **32 ms** in one Go
test run. `TestPersistentPDFiumCancelQueuedRenderBeforeStart`: PASS;
no obsolete child launched. `TestPersistentPDFiumCancelAndResumeWithLiveWorker`:
PASS; a higher-generation PDF page rendered successfully. Existing
timeout/restart/refusal tests remained green.

### Native GUI fault-injection tests

`tools/test-ui-pdfium-worker-faults.ps1` launches a verified owned
Quick Sort candidate in a private, disposable profile and source
folder. The test-only worker is built with **`pdfium_acceptance`**
from File Workspace: it can delay or intentionally exit on a chosen
synthetic PDF and writes a one-use marker for deterministic test
synchronization. The normal worker build does **not** contain this
fault-injection code.

| Scenario | Result | Evidence |
|---|---|---|
| Cancel an 8s delayed PDF render on switching to a second PDF | PASS | `D:\FlowverseReportsRepo\dcm-pdfium-fault-cancel-20261118-034202-d1a0b2` |
| Crash a persistent worker once during native PDF preview, then restart | PASS | `D:\FlowverseReportsRepo\dcm-pdfium-fault-crash-20261118-034232-7e7205` |
| Stall both persistent attempts beyond the 15s response deadline, then recover through one-shot PDFium | PASS | `D:\FlowverseReportsRepo\dcm-pdfium-fault-timeout-20261118-034259-2d8fae` |

Each evidence folder has `receipt.json`, renderer trace and
sampled actual PDF screenshot pixels. All three tests completed
with `PASSED=True`, `NORMAL_EXIT=True`, and test exit 0.
Both synthetic input PDF hashes were retained, and no owned worker
process remained after test shutdown.

- **Cancellation:** `worker-cancel-requested`, `worker-canceled`,
  and `result-discarded` were observed for the obsolete document.
  The replacement PDF `result-applied` event occurred **519 ms**
  after the user-selection operation began. Zero one-shot fallbacks.
  Reopening the original PDF then succeeded.
- **Crash:** first worker PID 18436 exited intentionally; PID 31388
  restarted and rendered the PDF. No one-shot fallback was required.
  Another PDF opened successfully, and both owned PIDs exited.
- **Timeout:** two separate persistent attempts exceeded their
  response deadline; exactly **one** one-shot fallback rendered the
  original PDF. The follow-up PDF then rendered in the recovered
  persistent worker. No worker remained after shutdown.

The initial GUI cancellation harness mistakenly read the
previous-page button instead of the page-status label. A subsequent
run included screenshot processing time in a render latency
threshold. Both test-only issues were corrected; the final three
scenario runs passed and the trace confirms cancellation before
the replacement render was applied.

### Regression and ordinary development bundle

Full Quick Sort Go regressions on Lenovo following the changes:

- Untagged: `go test ./... -count=1` — **PASS**, 53.193s
- Development-tagged:
  `go test -tags pdfium_dev ./... -count=1` — **PASS**, 43.662s

After rebuilding the normal, non-fault-injection development bundle,
`tools/test-ui-pdfium-provider.ps1 -DevDefault` also passed:
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261118-034708-6103cc5`.
The normal GUI ran **19** PDF requests through one worker PID **3828**,
with zero unplanned fallbacks, correct Rename/Move/Undo journal and
byte preservation, damaged/protected-PDF recovery, and clean shutdown.

Rebuilt ordinary bundle manifest:
`D:\FlowverseReportsRepo\dcm-pdfium-provider-acceptance-20261117\out\pdfium-dev\pdfium-development-manifest.json`.

- Quick Sort commit:
  `b3f8d50322b07c8be15ce24c9101a719e6d323f3`
- File Workspace worker commit:
  `cb3ba0dde970c224f7a67ad73acc238744818691`
- Quick Sort executable SHA-256:
  `D231B2393E2152AC4973F6C1A6820002F60076CEA65BAE9289AA57E76E57B97A`
- Normal (untagged) worker SHA-256:
  `37C54DC9F1FC72CBB8A73539E2814DEC899669FA2B2B5487FD44C95694DA04D9`

### Release decision

**The three cancellation/fault-recovery gates passed, but PDFium is
still the tagged development default only.** No production provider
switch, installation or release merge was performed. Remaining
release gates include longer-running memory/handle pressure,
complete multi-DPI two-axis cursor anchoring, diverse real-world
scanned documents, PDFium binary/runtime signing, security review,
redistribution notices/licensing and installer acceptance. Zoom
supersession is currently coalesced; the new immediate cancellation
targets obsolete *document selections*, not every wheel tick.
