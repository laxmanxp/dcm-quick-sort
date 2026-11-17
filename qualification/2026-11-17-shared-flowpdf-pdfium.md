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
