# Native Win32 Queue/History Controls — Qualification

Date: 2026-11-21
Branch: `feature/quicksort-two-pane-history-tabs`

## Qualified: SysTabControl32

The two temporary Queue/History buttons were replaced with native
`SysTabControl32`. `TCN_SELCHANGE` routes selection to the existing
visibility and focus logic. Existing History search, journal and Undo remain
unchanged. The Lenovo GUI acceptance passed under the normal configuration:
`D:\FlowverseTesting\TestResults\dcm-pdfium-gui-20261121-140309-ad397c7`.

## Experimental: SysListView32 report table

The new `native_queue_report_windows.go` creates a real
`SysListView32` report view with native `SysHeader32` headers and four
columns (Name, Extension, Created, Updated), only when
`DCM_NATIVE_QUEUE_LISTVIEW=1`. **It is not enabled by default.**

The legacy listbox remains the backing selection adapter so existing file
selection and transaction code is not reimplemented during the experiment.
The test verifies that the native control, header and four rows exist.
However, synthetic native ListView row selection causes the application to
exit unexpectedly. This still reproduces after:
- correcting the test so it first clears selection,
- deferring selection synchronization until after `LVN_ITEMCHANGED`
  via `wmNativeQueueSelection`.

Failed receipts:
- `D:\FlowverseTesting\TestResults\dcm-pdfium-gui-20261121-140830-35a7178`
- `D:\FlowverseTesting\TestResults\dcm-pdfium-gui-20261121-140946-ee0a587`
- `D:\FlowverseTesting\TestResults\dcm-pdfium-gui-20261121-141148-1487dd9`

**Do not enable the native report view by default or distribute it as a
qualified feature.** Before promotion: isolate whether the failure originates
in the test's `LVM_SETITEMSTATE` payload, the native notification path, or
the backing selection adapter; capture Windows exception/exit-code evidence;
qualify native header sort, single/multi-selection, keyboard navigation,
right-click Shell context menu, Rename/Move/Undo and normal shutdown.

The default native-tab + existing listbox implementation is the
established fallback. Release and Suite installations must remain unchanged.

## Follow-up qualification, same day

The original synthetic `LVM_SETITEMSTATE` test was found to be an unreliable
proxy for ordinary mouse selection: it made the process exit unexpectedly,
even when legacy synchronization was disabled. A mouse-style ListView row
selection **did** reach the existing filename/preview adapter and passed a
complete isolated PDFium GUI run. The test also incorrectly assumed a fixed
filename at row index 1 despite the default date sort; correcting the expected
filename to the actual row value yielded two consecutive native GUI passes
without tracing:

- `dcm-pdfium-gui-20261121-143649-757d33b` — PASS, normal exit
- `dcm-pdfium-gui-20261121-143655-9ec6a99` — PASS, normal exit

Native `SysHeader32` Name-header sorting also passed in those runs.

Native context-menu routing and hit testing have been implemented but **are
not qualified**: the opt-in `-VerifyNativeContext` GUI test stalled, and the
owned test harness was stopped without modifying unrelated applications. Its
last test evidence folder was
`dcm-pdfium-gui-20261121-144055-4a25243`.

Until native Ctrl/Shift multi-selection, actual Shell context menus, keyboard
navigation, Updated-column sorting and full mixed-document acceptance pass,
`DCM_NATIVE_QUEUE_LISTVIEW=1` remains a development-only opt-in, not a
production default. The regular two-pane listbox and native tabs are retained.

## Final regression checkpoint

GitHub branch source commit `9461a17` passed the full Go regression suite
(`go test -p 1 -vet=off -tags pdfium_dev ./... -count=1`) on Lenovo.
The standard two-pane Queue/History + owner-drawn four-column queue GUI
also passed with normal exit in receipt
`dcm-pdfium-gui-20261121-144446-87faaa6`.
The native ListView enabled by `DCM_NATIVE_QUEUE_LISTVIEW=1`, tested with
mouse-style row selection and native Name header sorting, passed with normal
exit in receipt `dcm-pdfium-gui-20261121-144512-5f62f5a`.

These passing runs **do not qualify** native Shell context menus or
Ctrl/Shift multi-selection. An experimental context-menu probe stalled and
was not counted as a pass. Keep the report ListView opt-in and retain the
proven owner-drawn listbox as the default.

## Native selection follow-up — 2026-11-21

The native ListView selection adapter now has successful physical-style GUI
acceptance receipts for both Ctrl-click multi-selection and Shift range
selection, without enabling it as the default:

- `dcm-pdfium-gui-20261121-145513-a1daeb7`: PASS, normal exit. Two Ctrl-selected rows propagate to the legacy selection model (`[1,2]`). Native Name/Updated column sort and PDFium also pass.
- `dcm-pdfium-gui-20261121-145620-8ee884f`: PASS, normal exit. Shift-selected rows `[0,1,2]` propagate to the legacy model and Down Arrow advances native focus to row 3.

An isolated nonmodal native context hit probe was added, but its GUI acceptance
run in `dcm-pdfium-gui-20261121-145747-7f377da` stalled without a receipt.
The owned test shell was stopped. **Context-menu support is not accepted**.
Actual Shell popup/keyboard activation still requires controlled independent
qualification. Keep `DCM_NATIVE_QUEUE_LISTVIEW` opt-in only. Production untouched.

## Nonmodal context diagnostic follow-up

The test harness was updated to avoid a blocking synchronous cross-process
`WM_CONTEXTMENU` call, then a separate development-only `WM_APP` diagnostic
message was introduced to exercise ListView pointer/keyboard hit testing
without opening the Windows Shell popup. Neither isolated diagnostic has yet
produced a passing receipt on Lenovo. In the latest attempted run,
`dcm-pdfium-gui-20261121-151008-cb0197c`, the test did not emit a receipt,
and the owned test shell was stopped. **Do not mark right-click Shell menu or
hit-test integration as qualified.** The diagnostic is gated by
`DCM_NATIVE_QUEUE_CONTEXT_PROBE=1`, and the native ListView itself remains
opt-in via `DCM_NATIVE_QUEUE_LISTVIEW=1`.

Following those changes, the default listbox-based two-pane interface passed
its isolated Go sorting/focus tests and GUI acceptance on Lenovo with normal
shutdown: `dcm-pdfium-gui-20261121-151132-2d1d991`. This confirms the
supported default workflow was not changed by the diagnostic experiment.

Next: diagnose the early exit and no-receipt case with process-exit evidence
and a narrow watchdog; qualify right-click and keyboard Shell menus through a
separate controlled physical-input test before promoting the native ListView.

## Final architecture decision — supersedes earlier experimental status

QuickSort is an action-oriented filing application, not a general-purpose file
manager. Its Windows Explorer Shell context menu was intentionally **removed**
from the QuickSort source (both the implementation and its unit tests). The
context-menu diagnostics were removed from the native queue. This feature
belongs, if required, in a separate FlowSearch-specific implementation; no
FlowSearch source was modified by this QuickSort change.

**Native `SysListView32` is now the default queue on this DEVELOPMENT branch**
for Name, Extension, Created and Updated. The native `SysTabControl32` remains
the Queue/History selector. The existing Win32 listbox is retained as a
reversible fallback and selection adapter, not the visible default.

Rollback for development/test builds:
`DCM_NATIVE_QUEUE_LISTVIEW=0` (explicitly forces the legacy owner-drawn
listbox). With the environment variable unset, the native report view is used.

Verified on Lenovo:
- Native development default: full Go regression + two-pane headed GUI,
  row selection, Ctrl/Shift multi-selection, Arrow keyboard navigation,
  Name/Updated header sort, PDFium, Rename, Move and Undo, normal exit;
  receipt `dcm-pdfium-gui-20261121-153920-e0e9a70`.
- Explicit legacy rollback: two-pane tabs and legacy four-column sorting,
  normal exit; receipt `dcm-pdfium-gui-20261121-153854-2e50f5d`.
- Extension header clipping was spotted in manual screenshot review, and its
  width was increased. Final native GUI after that fix passed with normal exit;
  receipt `dcm-pdfium-gui-20261121-154204-86d5fe2`.
  Screenshot `01-page-one.png` was inspected: all four headers are readable,
  file selection is visible, and PDF preview and controls remain aligned.

Production/release versions of QuickSort and FileFlow Suite were not changed.
The previous context-menu blocker no longer applies to QuickSort; the historic
failed experiments above are retained only as background evidence.

## Post-promotion native sorting and focus correction

Corrected the native header sorting path so it restores multi-selected files
into the visible `SysListView32` after sorting and returns focus to
`queueInputHandle()`, rather than the hidden legacy listbox. Added a headed
GUI assertion that native and backing listbox selections agree after Name and
Updated sorting, with the active rename field representing a selected file.

Lenovo receipts:
- Native default plus selection-after-sort GUI: `dcm-pdfium-gui-20261121-154945-11036cd` — PASS, normal exit.
- Full Go regression suite and legacy rollback GUI:
  `dcm-pdfium-gui-20261121-155055-76e824a` — PASS, normal exit.

The standard QuickSort workflow remains free of Windows Shell context menus;
FlowSearch should own Explorer-style context actions as an independent feature.
Production binaries/installations were not changed.

## Queue selection continuity — Reload, filters, source changes

QuickSort now identifies selected files by cleaned case-insensitive **full file
path**, not by row index. Reload and filter updates retain only those selected
files still present in the current visible queue. When the active file has
become hidden but another selected file survives, that surviving file becomes
the active preview. If none survive, QuickSort selects the first visible file;
if the queue is empty, it clears the selection and filename fields. Changing
source folders never carries a stale selection from the previous folder.
Both the native report view and the hidden legacy selection adapter are
updated after the refreshed rows are populated.

Lenovo verification:
- Full Go regression suite (`go test -p 1 -vet=off -tags pdfium_dev ./... -count=1`): PASS.
- Native headed acceptance: Ctrl-selected pair retained by Reload, name-filter
  intersection, Clear Filters, no-match empty queue, source-folder switch and
  return: PASS, normal exit, receipt
  `dcm-pdfium-gui-20261121-155949-3bd4e68`.
- Legacy listbox rollback + two-pane GUI: PASS, normal exit, receipt
  `dcm-pdfium-gui-20261121-160105-dfb09d3`.

No production or packaged FileFlow Suite installation changed.

## Native Queue preview burst coalescing

Native ListView row selection now refreshes filename, selection count and
metadata immediately while deferring the expensive preview startup by 90 ms.
Each further native row selection replaces that pending preview. The final
selection is loaded using a Win32 UI-thread timer, with timer cancellation in
ordinary `unloadPreview()` so Reload, file operations and shutdown retain
existing cleanup guarantees. Changing the multi-selection without changing
the active row updates the selection count but does not restart the preview.
The legacy listbox uses its existing immediate preview behavior.

Lenovo verification:
- Full Go regression suite and new candidate build: PASS.
- Rapid native navigation (3 row changes, 1 settled preview), native header
  sorting, Rename/Move/Undo with byte/journal checks, PDFium, normal exit:
  `dcm-pdfium-gui-20261121-173954-4e22d39` — PASS.
- Legacy listbox rollback and two-pane GUI:
  `dcm-pdfium-gui-20261121-174032-bd7a17d` — PASS, normal exit.

An initial extended run failed a brittle acceptance helper that clicked the
wrong native row in longer lists; the test-only row-coordinate calculation was
corrected and the full native GUI run passed. The production installations of
QuickSort and FileFlow Suite were not altered.

## Sustained navigation endurance — 2026-11-22

The existing GUI endurance suite was run for **120 cycles = 240 file
changes**, selecting TXT and PDF files, with persistent PDFium rendering,
final PDF file integrity and Rename/Move/Undo checks. The **complete GUI run
PASSED**, exited normally, and produced receipt
`dcm-pdfium-gui-20261121-195507-688bf22`.

Measured as median end-window minus median initial-window after warmup:
- QuickSort private bytes: **+1,191,936** (~1.14 MiB)
- PDFium worker private bytes: **-6,025,216** (~-5.75 MiB)
- QuickSort handles: **+19**; PDFium worker handles: **0**
- One persistent worker PID; **0** unplanned one-shot fallbacks
- Source PDF bytes unchanged; successful normal host/worker shutdown

An earlier full-run attempt `dcm-pdfium-gui-20261121-195258-287a453`
completed its 120-cycle endurance step (worker and memory also stable) but the
**overall run failed later** on a desktop snapshot with `Window PID mismatch`
when another foreground window was present. That run is *not* counted as an
end-to-end pass.

A new separate mixed-document keyboard scenario was added to the acceptance
harness using disposable copies of actual TXT, DOCX, XLSX and PDF fixtures:
**40 visible rows**, including 36 disposable additional documents. On the
first diagnostic run `dcm-pdfium-gui-20261121-195709-0c68ee6`, the segment
`native-keyboard-mixed-endurance` **PASSED 300 arrow-key steps**, ending on
focused row 12 after ~1.37 seconds with immediate filename agreement,
QuickSort private memory growth +2,977,792 bytes and +33 handles (under the
segment's limits). However, **that full run FAILED during cleanup**, because
a pre-existing row-click helper only handles the first 17 visible rows and
tried to click the original source at index 29. The cleanup was revised to
first filter to the original document.

Two subsequent native mixed-document acceptance attempts stalled before the
keyboard fixture stage and produced no acceptance receipt. Only their
exactly-owned test process/shell instances were stopped. **Do not claim full
40-row keyboard acceptance yet.** Qualification remains **PARTIAL** pending a
bounded, desktop-safe automation rerun through cleanup, native selection,
Rename/Move/Undo, and normal shutdown. The existing production QuickSort and
FileFlow Suite installations remain unchanged.

## Mixed-document keyboard acceptance resolved — 2026-11-22

The intermittent 40-row keyboard qualification issue was traced to brittle
test-harness ordering and fixed mouse coordinates, not a confirmed QuickSort
file-operation defect. The 40-row mixed TXT/DOCX/XLSX/PDF soak now runs
immediately after native queue initialization; ordinary file selection in
the test uses native Home/Down keyboard navigation rather than click positions.

**End-to-end Lenovo PASS**, receipt
`dcm-pdfium-gui-20261122-013229-3740530`:
- 40 visible files, including 36 temporary authentic-format mixed fixtures
- 300 Up/Down key steps; focused row 12 verified, active filename matched
- Keyboard segment elapsed ~1,497 ms, host private memory +1,875,968 bytes,
  host handles +33, within acceptance limits
- All 36 temporary fixtures removed, original four files restored
- PDFium Rename/Undo, Move/Undo, file integrity and damaged/protected PDF
  refusal/recovery acceptance passed after the keyboard soak
- Single persistent PDFium worker, 14 render requests, 0 unexpected fallbacks,
  worker owned shutdown, normal application exit

**Full Go regression suite PASS** and explicit legacy-listbox rollback GUI
PASS, receipt `dcm-pdfium-gui-20261122-013451-f1317a1`, normal exit.

The earlier failed/incomplete attempts remain documented as investigation
history, but the mixed-document keyboard acceptance is now **qualified**.
This work changes only the GitHub development branch and test fixtures; the
production QuickSort and FileFlow Suite installations remain unchanged.

## Interactive PDF Previewer selector — Lenovo dev session, 2026-11-22

The `pdfium_dev` development build now offers **Tools > PDF Previewer**:
**Auto (development default)**, **PDFium (FlowPDF)**,
**Windows.Data.Pdf**, and **SumatraPDF (restricted)**. The checked option
shows the selected backend. Switching a selection reopens the same active PDF
without restarting QuickSort. It is a **session-only** choice implemented
through the existing `DCM_PDF_PROVIDER` backend policy: no production
settings or release defaults are changed. Explicit PDFium requires the
managed DLL and persistent worker; restricted Sumatra requires its separately
approved pinned executable and no-write policy. Unavailable dependencies
are reported instead of silently substituting an unapproved viewer.

The complete Go test suite and candidate build passed on Lenovo. GUI selector
acceptance `dcm-pdfium-gui-20261122-015102-ee62116`: PASS, normal exit;
Windows.Data.Pdf, PDFium and Auto each rendered page 1 of 2 of the same
PDF; file Rename/Undo and Move/Undo preserved bytes, and the final PDFium
worker exited cleanly. Deliberate renderer changes are excluded from the
strict unexpected-worker-restart invariant **only in this comparison test**.
Sumatra integration itself was not newly qualified, and Lenovo did not have
a configured `DCM_SUMATRA_EXE` machine runtime.

The development executable
`D:\\FlowverseTesting\\Builds\\DCM-QuickSort\\two-pane-ui-20261121\\two-pane-pdfium-dev.exe`
was started interactively on Lenovo, PID 31068, with
`tests\\fixtures\\sumatra-scroll.pdf` selected. The main window was brought
to the foreground. This is an interactive dev run, not a production install.

## SumatraPDF restored to Lenovo isolated test runtime — 2026-11-22

Downloaded the official SumatraPDF 3.6.1 x64 portable archive into a
test-only staging area and verified its archive SHA-256
`98b33a518d42986856d225064b0cd2d3643ecf78cbf84ab873d26cc51877a544`.
The contained executable matched the repository's pinned SHA-256
`719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0`.
The verified private executable is at
`D:\FlowverseTesting\Runtime\SumatraPDF-3.6.1\SumatraPDF.exe`
with the exact 173-byte `sumatrapdfrestrict.ini` no-write/no-print
policy alongside it. The download staging area was deleted. No systemwide
installer, shell registration, or machine environment changes were made.

The first Sumatra GUI comparison test exposed a **125% viewer DPI** mismatch:
viewer dimensions were 1203x556 versus a 965x448 preview pane. QuickSort's
Sumatra resize now switches temporarily into the embedded viewer's DPI
awareness context before `SetWindowPos`. The subsequent GUI acceptance
**PASSED**, receipt `dcm-pdfium-gui-20261122-021014-a25e0b3`:
- Windows.Data.Pdf, PDFium and Auto each rendered the same PDF
- Sumatra's owned child `SUMATRA_PDF_FRAME` attached entirely within the
  preview (accepted size 962x445), with native control visible
- Switching away from Sumatra removed its owned process, restored Auto,
  and displayed the same PDF via PDFium
- Rename/Undo and Move/Undo preserved original bytes and journal integrity;
  application exited normally with worker cleanup

For interactive user comparison a separate development session was launched
on Lenovo, QuickSort PID **24188**, with `DCM_PDF_PROVIDER=sumatra` and
`DCM_SUMATRA_EXE` scoped to that process. Sumatra child PID **7448** was
confirmed running with `-restrict`, private `-appdata`, and `-plugin`
arguments for the sample `tests\fixtures\sumatra-scroll.pdf`. The window
was brought to the foreground and intentionally left open for manual testing.
The older development session was not disturbed. Production QuickSort and
FileFlow Suite remain untouched.

## QuickSort-owned PDF toolbar replaces Sumatra — 2026-11-22

Per product decision, the SumatraPDF choice was **removed from Tools > PDF
Previewer** and the Sumatra startup call was **removed from active PDF loading**.
The development selector now provides only Auto, PDFium (FlowPDF) and
Windows.Data.Pdf. The standalone legacy Sumatra integration source remains
inactive temporarily for historical compatibility, but no Sumatra executable
is bundled or started by the current dev build. The isolated Sumatra test
runtime was removed from Lenovo after confirming no owned viewer was in use.

A QuickSort-owned compact 34px PDF toolbar now contains: previous page,
**editable current page + Go**, total pages, next page, **Fit Width**,
**Fit Page**, **rotate counterclockwise/clockwise**, and zoom out/in.
It uses native Win32 controls and existing PDFium/Windows.Data.Pdf operations,
not copied Sumatra code or assets. On narrow panes, controls past the width
limit are hidden rather than overlapping. PDF text find is **not implemented**
in this iteration; no nonfunctional search field is exposed.

Lenovo full Go suite and development build: PASS. The GUI acceptance harness
was updated to check page number + total separately and exercise Go 1/2,
clockwise/counterclockwise rotation, fit width/page and zoom in/out.
- PDF-specific rotation and new-page-entry GUI with legacy queue fallback:
  `dcm-pdfium-gui-20261122-023520-499d33e` — PASS, normal exit.
- Native report queue full GUI with custom toolbar:
  `dcm-pdfium-gui-20261122-023654-caf75dd` — PASS, normal exit.

The updated dev app was launched interactively on Lenovo, PID **12208**, from
`D:\FlowverseTesting\Builds\DCM-QuickSort\two-pane-ui-20261121\two-pane-custom-toolbar-dev.exe`
with `DCM_PDF_PROVIDER=auto` and no Sumatra configuration. Its window was
brought to the foreground for manual inspection. Production QuickSort and
FileFlow Suite installations remain unchanged.


## Narrow PDF toolbar qualification — 2026-11-22 (PARTIAL)

Development change `7d36c8d` preserves the existing wide-toolbar order and
prioritizes previous-page, editable page/Go, next-page and both rotation icons
before Fit, Zoom and page-count decoration when the preview client is below
454 units. No renderer/transaction code or release default changed.

Targeted GUI assertions were added in `0909713`, then the test harness was
corrected in `3a5c9c4` to (a) capture the actual resized window rather than
silently restoring it, and (b) avoid demanding a new PDF render after a
pure window resize. The exact development source built successfully on Lenovo,
and the Go suite passed on the previous product-only commit `7d36c8d`.

Native receipts (retain failures, do not relabel):
- `dcm-pdfium-gui-20261122-030219-c9b8f44`: native queue run
  exited early at an owned-window/control selection check, before toolbar
  measurements. Not a toolbar acceptance.
- `dcm-pdfium-gui-20261122-030244-3010e8d`: legacy queue; both
  rotation controls visible, within preview, no overlap at requested whole-window
  widths 1350/920/800. Measured preview clients 824/655/655.
  Rendering and source PDF visual marker checks passed. Full run failed
  because the harness incorrectly required a new result-applied generation
  after a resize without a render. Its captures were made after a generic
  Restore and do **not** prove actual narrow-width appearance.
- `dcm-pdfium-gui-20261122-030419-688d9b6`: corrected width-specific
  screenshot attempt failed strict occlusion check (unrelated
  XamlExplorerHostIslandWindow in foreground); owned app shut down normally.
  Do not accept or bypass an occluded image.
- `dcm-pdfium-gui-20261122-030502-d250827`: screenshot-free legacy queue
  run passed the width-specific rotation-geometry assertions and subsequent
  page navigation, rotation, PDF preview recovery and damaged/protected
  refusal steps. Overall FAIL: unexpected persistent PDFium worker restart
  (PID 29352 -> 31776) without a matching ordered cancellation; normal app exit.
  Investigate separately before claiming a complete GUI PASS.

Important limit: QuickSort's minimum whole-window geometry clamped the
requested 800-wide test, leaving a preview of 655. Thus the
`width < 454` priority branch is not yet physically exercised by the GUI.
A deterministic narrow-preview fixture (or testable pure layout calculation),
plus an unoccluded visual capture and clean full regression, is still required
before accepting that adaptive branch. Production and Suite unchanged.


## Worker-sidecar correction and compact layout qualification — 2026-11-22

The earlier unexpected-worker-restart verdict in
`dcm-pdfium-gui-20261122-030502-d250827` was a **deployment/test-mode
misclassification**, not evidence that the persistent PDFium worker itself
restarted incorrectly. The temporary `pdfium_dev` app had been compiled in
a separate directory without its required
`flowpdf-preview-worker-persistent.exe` sidecar. With `-DevDefault`, the
app therefore used the Windows.Data.Pdf fallback. The test harness incorrectly
applied the persistent-worker PID invariant to those Windows render subprocesses.

The verified worker (SHA-256
`a376b587bc4d682b796cc0abd7d7fba158bad0571e50fe0576a18b0a88e70a03`)
was copied beside the isolated development EXE. The corrected test
deployment passed `dcm-pdfium-gui-20261122-035425-e805ce3`, with normal
exit. Commit `d05a6ed` adds a fail-closed harness preflight requiring
the expected sidecar and matching worker hash in `-DevDefault` mode.

The runtime Win32 toolbar now consumes a pure Go placement plan
(`pdf_toolbar_layout.go`), tested by
`pdf_toolbar_layout_test.go`. It preserves the full-width order and places
both rotation buttons before Fit/Zoom controls at compact widths. Tests
cover all widths 0..453 for no overlap/out-of-bounds, both rotation controls
at widths 240/260/320/400/453 and complete original ordering at 454.
This provides deterministic coverage for the compact branch which cannot
currently be reached by whole-window resizing due to minimum-size clamping.

At exact candidate `dabea236`, Lenovo Go full suite
(`go test -p 1 -vet=off -tags pdfium_dev ./... -count=1`) **PASS** and
development build **PASS**. GUI receipt
`dcm-pdfium-gui-20261122-035602-21fa13a` **PASS**, normal exit, checked
existing PDF navigation, clockwise and counterclockwise rotation, narrow
toolbar geometry at the reachable whole-window widths, file operations,
damaged/protected PDF recovery, and persistent worker ownership.
Its 33 render requests used two owned PIDs with **one accepted ordered
cancellation restart**, zero one-shot fallbacks and owned worker shutdown.

**Scope limitation:** GUI screenshots were intentionally skipped because
unrelated desktop foreground occlusion had prevented trustworthy capture.
The actual <454px preview appearance has deterministic placement tests, not
physical GUI visual acceptance. No release, production app, Suite, or
registered release checkout was changed.
