# DCM QuickSort — Sumatra toolbar retry verified; full PDF acceptance still pending

Latest checkpoint: [toolbar/navigation/zoom retry](2026-11-12-sumatra-toolbar.md).
Tested source `c53f85263963886b1038dbd8933dd3883481aabd`.

Completed numbered steps remain DCM-01–18 and DCM-20–27 (26 including five setup steps). No numbered closure in this retry. DCM-19 remains blocked; DCM-28 and DCM-29 remain pending. Do not restart the other LibreOffice divider project.

## Verified

Recovered the already-committed toolbar correction and expanded UI test, then freshly ran FlowDev update/tooling/build. Two unchanged-source Sumatra runs each pass 19 checks; all 17 full-window images from the first run reviewed. Its 16 pixel crops were automatically checked, not separately inspected. Original top clipping is measured RED (page Y=-13) to GREEN (Y=8); redundant host sizing controls hide for Sumatra and return on text. Runtime and strict deny-print/deny-write policy are unchanged.

Actual accessible toolbar Next/Previous/Zoom and page-entry tests pass, with both boundaries, 100%/Fit modes and two-size containment checks. Native baseline 366 pass / 0 fail / 1 retained external-PPTX skip; text17/image23 regressions pass. Second Sumatra and regression images not separately reviewed. No further product/test implementation changes were made in the retry.

## Next exact work

**DCM-UI-RESIZE-01:** resizing back leaves old host button/footer fragments. **DCM-SUMATRA-UI-02:** smaller window clips far-right Aa toolbar control. Add focused failing assertions, make the smallest correction and requalify full-window images. The Edit/Static geometry check does not validate every toolbar button. Keep original page-number correction in place; do not enable printing or weaken ownership/policy checks. General DCM-UI-LABEL-01 remains open.

Then qualify rapid PDF/text/PDF switching, cancellation, failed/locked/password PDFs, Undo/release-timeout refusal and active-PDF host close. Current 19 checks do not cover these. At100%/FitWidth the viewport showed a magnified colored area; scroll/text visibility at those zooms remains unqualified. Sumatra stays opt-in; Windows.Data.Pdf remains default. Do not close DCM-28/29 or switch defaults merely because this focused run passes.

## Execution and identity

Code locally -> GitHub -> FlowDev update/test on Lenovo, one UI run at a time. Workspace D:\FlowverseReportsRepo; candidate dcm-quicksort-candidate; profile tools\flowdev-dcm; runner D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe. Device d18227f2-0b54-4b11-9026-56ac40305f02. No remote source edits.

ui-sumatra needs DCM_SUMATRA_SOURCE_EXE=D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe; it creates a private copy and policy. Do not alter the prototype/user runtime.

Build receipt dcm-quicksort-build-20261112-040215-6858b471\receipt.json matches c53f8526 only. App hash2b0f4e8cfe9e54909e63cf327b89dbb448ed92acfa194ea3a12f1e0ccba4ae89. Rebuild after any metadata/source commit.

Reports under D:\FlowverseReportsRepo: reviewed Sumatra dcm-quicksort-ui-sumatra-20261112-040237-691adf62; repeated Sumatra 20261112-040536-5300276f; baseline dcm-quicksort-baseline-20261112-040352-443135c7; text dcm-quicksort-ui-text-20261112-040435-99bf630e; image dcm-quicksort-ui-image-20261112-040504-1eb23a4f. Exact hashes in evidence/sumatra-toolbar-native-summary.json.

Keep all original checkpoints, failed runs, DCM-19, external DCM_REAL_PPTX, distribution/package/installer and user gates. Keep PR #1 draft; no merge or publication.
