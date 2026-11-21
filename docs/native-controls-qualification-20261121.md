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
