# DCM-34 — PDF bitmap dimension discrepancy closeout (2026-11-16)

## Decision

**Cause confirmed and narrow discrepancy resolved.** The historical `DCM-PDF-RENDER-SIZE-01` observation of renderer metadata **612 × 1000** and decoded PNG **765 × 1250** is exactly consistent with **120 DPI / 125%** scaling on the observed Lenovo desktop. The renderer metadata width and height are logical device-independent pixels (DIPs); the WinRT-rendered bitmap is in physical pixels. This record does **not** establish that all consumers of the metadata make DPI-correct layout decisions, nor does it certify arbitrary DPI configurations or every PDF renderer/provider. No production source correction is indicated for this specific dimensions discrepancy.

## Procedure and source identity

- Application: DCM QuickSort; repository `laxmanxp/dcm-quick-sort`; branch `release/v1.0.71`.
- Tested pushed source: `e97ba4dc4b717e6cb24906ac2bf62bdca5f0b7c1`.
- `flowdev update dcm-quicksort`: **PASS, already up to date**.
- `flowdev test dcm-quicksort pdf-bitmap-dpi`: **PASS**.
- Native report: `D:\FlowverseReportsRepo\dcm-quicksort-regressions-20261116-071219-ebe56871\receipt.json`.
- DPI detail receipt: `D:\FlowverseReportsRepo\dcm-quicksort-regressions-20261116-071219-ebe56871\pdf-dpi-observation\receipt.json`.
- Production Windows.Data.Pdf rendering was exercised through the embedded `pdf_render.ps1`; no renderer substitution, synthetic DPI modification or desktop display-setting mutation.
- The pinned input fixture hash was checked. Fixture bytes preserved and PDF file exclusively released after rendering.

## Measurements

| Case | Metadata dimensions (DIP) | Decoded PNG (physical px) | Predicted at 120 DPI |
| --- | ---: | ---: | ---: |
| Zoom 100% | 816 × 1333 | 1020 × 1666 | 1020 × 1666 |
| Fit page | 428 × 700 | 535 × 875 | 535 × 875 |
| Exact historical fit-page case | **612 × 1000** | **765 × 1250** | **765 × 1250** |
| Fit width | 500 × 817 | 625 × 1021 | 625 × 1021 |
| Zoom 100%, repeat | 816 × 1333 | 1020 × 1666 | 1020 × 1666 |

Observed desktop DPI: **120**; scale factor **1.25**. Pixel dimensions are consistent with metadata multiplied by `120 / 96`, within the native test's rounding tolerance; here they match the predicted integers exactly.

The repeated zoom-100 render in *this run* had zero changed decoded pixels and maximum channel delta zero. Earlier observed intermittent raster differences must remain part of historical evidence; this one stable repeat does not prove universal pixel identity. The native evidence receipt continues to report `feature_accepted=false` and `user_test=not_recorded`; raw flags are not rewritten.

## Follow-on scope

The **size discrepancy itself** is explained and does not require changing the WinRT render script solely to force 612 × 1000 physical pixels. A separate potential issue remains if image drawing, pan offsets or viewport geometry compare **logical** metadata values directly against **physical** image dimensions without unit conversion. This focused test measures renderer output, not full GUI layout or human experience; evaluate any such problem only with a reproducible UI symptom and dedicated bound test.

The parent `DCM-34` remains PARTIAL (32/50 complete). Its next substantive work is broader Office failure/protected-file coverage. Do not reopen the accepted XLSX/PPTX preview sizing exercises or repeat this PDF raster-only investigation without a new concrete failure.
