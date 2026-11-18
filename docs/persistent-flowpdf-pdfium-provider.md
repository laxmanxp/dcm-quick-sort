# Persistent FlowPDF/PDFium provider — native Quick Sort integration

**Status:** Development branch candidate; native GUI acceptance passed on
Laxman_Lenovo. **Persistent PDFium is the `pdfium_dev`-tagged development
default, but it is not the release-default previewer.** See
[`pdfium-development-default.md`](pdfium-development-default.md)
for the automatic bundled-worker build and rollback qualification.

## Usage and rollback

Untagged/release builds still default to Windows.Data.Pdf. The
`pdfium_dev`-tagged development build automatically selects persistent
PDFium when it can locate its bundled worker and managed DLL, and falls
back to Windows.Data.Pdf if either is absent. Sumatra and the prior
one-shot PDFium adapter remain available.

To activate the persistent PDFium provider only in an isolated test shell:

```powershell
$env:DCM_PDF_PROVIDER = 'pdfium'
$env:DCM_FLOWPDF_WORKER = 'D:\FlowverseReportsRepo\fileworkspace-pdf-zoom-acceptance-20261117\bin\flowpdf-preview-worker-persistent.exe'
$env:DCM_FLOWPDF_PERSISTENT = '1'
$env:FILEWORKSPACE_RUNTIME_DIR = 'C:\Flowverse\runtime'
```

`DCM_FLOWPDF_PERSISTENT=0` explicitly chooses one-shot PDFium.
In a normal/untagged build an unset value also means one-shot; in a
`pdfium_dev`-tagged build an unset value means persistent PDFium.
Set `DCM_PDF_PROVIDER=windows` for an immediate Windows.Data.Pdf
rollback in either build. Unsetting PDFium options restores the Windows
default in untagged builds and the automatic development default in
tagged builds.

Only use a worker built from the File Workspace
`feature/flowpdf-shared-provider-contract` branch after verifying its
hash and matching managed PDFium runtime. No installer or user deployment
has been performed.

## Architecture

```text
DCM native PDF pane (Go/Win32)
  -> existing PDFRenderRequest and generation/busy/pending coordinator
  -> DCM_FLOWPDF_PERSISTENT=1 (opt-in)
  -> sharedFlowPDFiumProcess (one owned child, private stdin/stdout)
  -> flowpdf-preview-worker.exe -ServeJSONL
  -> File Workspace PDFium backend with bounded 32 MiB PNG cache
  -> version 1 response with matching request ID and OK|metadata
  -> existing Quick Sort PNG decoding, status, zoom/pan and cache cleanup
```

The host serializes requests, enforces a 15-second response deadline,
rejects mismatched version or response IDs, and restarts the private child
once after a transport/protocol error. It does **not** retry legitimate
PDF decoder refusals. If the persistent protocol remains unavailable,
the worker can be called through the existing one-shot PDFium flags
under a separate 15-second deadline.

Only the exact child created by Quick Sort is terminated on failed
IPC, timeout or shutdown. There is no network listener or global
shared endpoint. Main-window teardown sends a bounded `shutdown`
request after renderer results have drained, then waits for the child
to exit or terminates it.

Old renderer modes retain their code path. The existing renderer
generation check still rejects stale page results; the old
`pdfShutdownCoordinator` continues to defer close while results are
outstanding. The queue coalesces repeated zoom requests rather than
launching multiple renders simultaneously.

**Important limits:** the process persists, and its in-memory bitmap
cache persists, but PDFium currently opens/closes each PDF document
on a cache miss. Obsolete document renders now cancel by generation,
while individual zoom-tick supersession remains coalesced. Fully
certified two-axis pointer anchoring, extended memory/handle stability,
and sandboxing beyond process isolation remain future work.

## Tests — Laxman_Lenovo (17 Nov 2026)

Source: `laxmanxp/dcm-quick-sort`,
`feature/shared-flowpdf-pdfium-provider`.

- Focused persistent process tests PASS: a hung helper is terminated
  within a bounded deadline; private child PID reused across real page
  renders and encrypted-PDF refusal; a deliberately killed child is
  restarted; the old one-shot executable remains a functioning fallback.
- Full native Quick Sort `go test ./... -count=1` PASS with persistent
  environment and synthetic long/raster/vector fixtures:
  `ok quicksort 32.346s`.
- Native GUI test `tools/test-ui-pdfium-provider.ps1 -Persistent` PASS
  with `NORMAL_EXIT=True`. It exercised page navigation, native
  toolbar/Ctrl-wheel zoom, byte-safe Rename/Undo and Move/Undo,
  damaged and encrypted PDF refusal, recovery, and subsequent close.
- One PID **20448** served **19 render requests** in the healthy
  acceptance run; **zero** one-shot fallbacks were needed.
- That exact worker PID was gone after the Quick Sort GUI closed.
- Receipt/traces/12 screenshot captures:
  `D:\FlowverseReportsRepo\dcm-pdfium-gui-20261117-174423-b3ecb67\`
- Equivalent one-shot native GUI acceptance passed:
  `D:\FlowverseReportsRepo\dcm-pdfium-gui-20261117-174805-fad969d\`.

### GUI trace intervals (indicative only)

For each `result-applied` record, the interval from the same
generation's `worker-started` to result application was calculated:

| Mode | Applied renders | Median | Mean | Maximum |
|---|---:|---:|---:|---:|
| One-shot PDFium | 12 | 144.9 ms | 148.6 ms | 218.5 ms |
| Persistent PDFium | 17 | 53.6 ms | 66.4 ms | 251.2 ms |

The different number and mix of applied renders prevent treating
these medians as a paired performance benchmark. The standalone
File Workspace tests provide a stronger process-lifetime comparison:
unique zoom settings with cleared cache measured 216.6 ms one-shot
versus 66.2 ms persistent on the synthetic fixture.

## Remaining release gates

- Stale-selection cancellation and injected GUI crash/timeout recovery
  now PASS on Lenovo (see
  [the 18 November qualification](../qualification/2026-11-18-pdfium-development-default.md));
  continue monitoring these paths during release qualification.
- Longer-run process memory and handle-leak tests, real scanned PDFs,
  high zoom extremes, multiple DPI/scaling configurations.
- Complete two-axis cursor anchoring measured in native GUI and
  human visual review of captures.
- Shared package/runtime signing, attribution, redistribution licensing,
  trust boundaries and installation lifecycle.

Do not merge the feature branch into a release or switch default PDF
preview providers before these gates are complete.
