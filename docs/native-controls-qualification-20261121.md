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
