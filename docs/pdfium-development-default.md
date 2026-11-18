# DCM Quick Sort — development-default persistent PDFium

**Scope:** the PDFium-default **development build** only. The existing
untagged/release Go build continues using Windows.Data.Pdf by default.
This development feature does not update an installed application.

## Build and run

In the Quick Sort feature-branch checkout, run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\tools\build-pdfium-development.ps1 `
  -FileWorkspaceCheckout 'D:\FlowverseReportsRepo\fileworkspace-pdf-zoom-acceptance-20261117' `
  -RunPolicyTests
```

This requires the File Workspace development branch
`feature/flowpdf-shared-provider-contract` and Quick Sort branch
`feature/shared-flowpdf-pdfium-provider`. It builds a fresh worker from
the File Workspace checkout and the Quick Sort executable with Go build
tag `pdfium_dev`. Both binaries are placed together in a separate
`out\pdfium-dev` directory with a SHA-256 manifest. No production
binary or release branch is overwritten. `CGO_ENABLED=0` is preserved
for the build.

### Selection policy

| Build and settings | PDF preview provider |
|---|---|
| Normal/untagged build, no override | Windows.Data.Pdf, unchanged |
| `pdfium_dev` tagged build with bundled worker and available managed DLL | Persistent PDFium automatically |
| `pdfium_dev` tagged build with missing worker or DLL | Windows.Data.Pdf automatically |
| `DCM_PDF_PROVIDER=windows` | Windows.Data.Pdf, explicit rollback |
| `DCM_PDF_PROVIDER=pdfium` | PDFium, explicit (missing worker reports configuration error) |
| PDFium selected + `DCM_FLOWPDF_PERSISTENT=0` | One-shot PDFium |
| PDFium selected + `DCM_FLOWPDF_PERSISTENT=1` | Persistent PDFium |
| `DCM_PDF_PROVIDER=sumatra` | Existing Sumatra selection, unchanged |

An empty `DCM_FLOWPDF_PERSISTENT` value uses the development build
default (persistent), while normal builds preserve the former one-shot
opt-in behavior.

The worker is discovered as
`flowpdf-preview-worker-persistent.exe` **beside the executable**,
or beneath the explicitly configured managed runtime directory. The
runtime DLL is checked using `FILEWORKSPACE_PDFIUM` or
`FILEWORKSPACE_RUNTIME_DIR\pdfium\bin\pdfium.dll`; when neither is
configured in the tagged development build, the documented
`C:\Flowverse\runtime\pdfium\bin\pdfium.dll` is checked.

When the default managed runtime is used, its directory is passed to
the worker process without changing the parent environment.

**Automatic fallback is a selection-time safeguard:** if the worker or
DLL is missing before opening a PDF, the development build uses
Windows.Data.Pdf. If a running persistent worker suffers IPC failure,
the existing supervised retry/one-shot PDFium fallback still applies.
Damaged/password-protected decoder refusals do not trigger automatic
Windows re-rendering. This difference is intentional until runtime
failure recovery is separately qualified.

## Lenovo acceptance evidence — 18 November 2026

Device: `Laxman_Lenovo`. All screenshots, fixtures, logs and output
executables are located in an isolated testing checkout under
`D:\FlowverseReportsRepo`, not a production installation.

- Developer-tagged policy tests: PASS
- Untagged/release policy tests: PASS
- Tagged native GUI with *no* PDF provider, worker-path, persistent-mode,
  runtime-dir or explicit-DLL environment overrides: PASS
- Bundled development executable and colocated worker built by
  `tools/build-pdfium-development.ps1`: PASS
- The actual bundled development binary's native GUI test: PASS
- Persistent-process reuse: **19** render requests, **1** worker PID,
  **0** unexpected one-shot fallbacks; worker exited after host close
- Page navigation, toolbar/Ctrl-wheel zoom, Rename/Move/Undo,
  damaged/protected-PDF refusal and recovery, normal host close: PASS
- Explicit `DCM_PDF_PROVIDER=windows` native GUI rollback: PASS,
  with Windows.Data.Pdf rendering and normal close

Primary evidence:
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261118-014901-79a6adf\receipt.json`
and
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261118-014741-c7f87c2\receipt.json`.

The Windows rollback test initially captured page 2 before the
PowerShell renderer had applied its bitmap, producing a stale page-1
screenshot and a color-assertion failure. The acceptance harness was
corrected to wait for an actual `result-applied` trace event.
The subsequent rollback test passed. This was a test synchronization
defect, not a demonstrated PDF decoding defect.

The development build manifest is at
`D:\FlowverseReportsRepo\dcm-pdfium-provider-acceptance-20261117\out\pdfium-dev\pdfium-development-manifest.json`.
It records the source commits and worker/executable SHA-256 hashes.

## Release decision and pending gates

PDFium is **not** promoted as the release default. Production selection
remains Windows.Data.Pdf. The existing Sumatra and one-shot PDFium
alternatives remain available.

Stale-selection cancellation, injected crash/restart, and two-attempt
timeout followed by one-shot fallback have now passed the native GUI
acceptance gates (see
[the updated qualification evidence](../qualification/2026-11-18-pdfium-development-default.md)).
Production promotion is still blocked by sustained
process/handle/memory pressure across many PDFs, complete two-axis
cursor anchoring at multiple DPIs, diverse real-world scanned and
vector-heavy PDFs, packaged runtime signing, trust boundaries,
redistribution notice/license review and installer acceptance.
