# Shared FlowPDF PDFium provider (development opt-in)

The FileFlow suite can reuse **File Workspace's PDFium engine** without copying
its Go `internal/pdfium` package into DCM Quick Sort.

Source-of-truth projects:

- File Workspace development branch: `feature/flowpdf-shared-provider-contract`
- DCM Quick Sort development branch: `feature/shared-flowpdf-pdfium-provider`

## Architecture (stage 1)

```
DCM Quick Sort PDF preview pane
  -> existing PDFRenderRequest and generation/worker-close coordinator
  -> opt-in FlowPDF command adapter
  -> flowpdf-preview-worker.exe (built from File Workspace)
  -> File Workspace internal/pdfium
  -> managed pdfium.dll
  -> temporary PNG + OK|pageCount|pageIndex|width|height|scalePercent
  -> existing Quick Sort bitmap/pan/toolbar code
```

This is a compatibility and ownership milestone, **not yet the final smooth
zoom viewer**. The worker is one process per render invocation and PDFium
currently opens the PDF for each request. A persistent worker, image cache and
cursor-anchored immediate scaling are future iterations for the native Quick
Sort UI. The FlowPDF WebView implementation has those separate UI features.

The default Quick Sort renderer is unchanged (PowerShell / Windows.Data.Pdf).
Sumatra continues to be separate. Missing opted-in worker is rejected rather
than silently falling through to an unintended binary.

## Isolated Windows acceptance

Build the worker from a File Workspace development checkout:

```powershell
$env:FILEWORKSPACE_RUNTIME_DIR = 'C:\Flowverse\runtime'
go build -o bin\flowpdf-preview-worker.exe ./cmd/flowpdf-preview-worker
```

In the Quick Sort development shell, explicitly select the worker:

```powershell
$env:DCM_PDF_PROVIDER = 'pdfium'
$env:DCM_FLOWPDF_WORKER = 'D:\FlowverseReportsRepo\fileworkspace-pdf-zoom-acceptance-20261117\bin\flowpdf-preview-worker.exe'
$env:FILEWORKSPACE_RUNTIME_DIR = 'C:\Flowverse\runtime'
$env:DCM_PDFIUM_TEST_PDF = 'D:\FlowverseReportsRepo\fileworkspace-pdf-zoom-acceptance-20261117\bin\flowpdf-worker-acceptance.pdf'
go test -run 'TestPDFiumProviderSelectionUsesExplicitWorker|TestSharedPDFiumWorkerSmoke' -v
```

Use the normal, isolated DCM test build workflow to launch the desktop GUI.
Unset `DCM_PDF_PROVIDER` and `DCM_FLOWPDF_WORKER` to return to the legacy
default. Production packaging must install the verified worker and PDFium
runtime explicitly and audit licensing notices.

## Acceptance gates before making PDFium the default

1. PDF generation/close and Rename/Move/Undo lifecycle unchanged
2. Native GUI load, page, fit and fit-width modes
3. Zoom on the pointer, fast repeated wheel, stable pan, and no blank transition
4. Scanned, graphics-heavy, malformed, password-protected and 300-page PDFs
5. Memory and process cleanup with repeated tab changes
6. Compare against Sumatra and the existing Windows renderer on Lenovo

Keep `release/v1.0.71` untouched until these gates are complete.
