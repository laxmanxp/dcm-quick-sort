# QuickSort — individual active-PDF Undo and controlled failure verified

Latest: [Undo and viewer failure](2026-11-12-sumatra-undo.md).
Native-tested source `cc04ae7cd8932cffc4e13d7e181ae51eb78095fc`; product fix `0c9cdf87084bd9f18c0e062bee9f40ce8f6f0ace`.
DCM-01–18 and DCM-20–27 retain completion; DCM-19 blocked; DCM-28/29 pending. No numbered closure. Sumatra optional, Windows.Data.Pdf default, PR #1 draft; no merge/publication.

## Verified

Two consecutive `ui-sumatra-undo` runs passed nine records each. Eight full-window screenshots from the first pass reviewed; six content crops automated only. Repeat/regression images not separately reviewed. Rename/Undo Rename with active PDF, reopen moved destination PDF/Undo Move, cleared destination target, controlled owned-viewer exit error and Reload recovery, normal close and source/profile/lock cleanup pass. All original fixture bytes restored and both original journal IDs undone.

Actual RED at55e2b877 restored the file safely but retained the destination row/draft. `undoSelectedProcessed` now refreshes when the active directory matches either `rec.From` or `rec.To` parent. DCM-UNDO-REFRESH-01 fixed for this individual path; do not assume batch/alias matrices passed.

Full native baseline368 passed/0 failed/1 external-PPTX skip (369 parents). One new release-refusal parent passes six nested Windows cases covering Rename/Move/individual Undo with invalid wait handle or unsignalled event+invalid job. These are real API failures with injected test-local state, not live-process permission modification or every failure branch. Refresh23 and lifecycle11 regression commands pass. Exact build app hash81c516b0a84442dec493f21ef1ff3562b988eb66fc76c0c177959c3afcb3bdb0.

## Open startup finding and test tooling

Several runs stopped before Undo on synchronous selection/text-query deadlines. New test-only DcmUndoInput posts the real selection notification and polls only the owned filename via50ms queries inside the unchanged20s readiness window; selected-row, document identity, file/journal and pixel checks remain. Final initial readiness1437/1610ms. **DCM-SUMATRA-STARTUP-01 remains open:** this test scheduling change is not a product responsiveness fix. Measure stage costs/cold-warm behavior before making performance claims. Retain all original failures and the separate minimized-capture failure after seven checks; final unchanged retry and repeat passed. Do not disable capture/ownership guards.

## Next exact scope

Qualify valid-header damaged/password-protected PDFs and missing-runtime recovery. Keep startup responsiveness, readable text after scrolling at100%/FitWidth, general labels, broader Batch Undo/path variants, remaining release-failure cases, distribution/installer and actual user acceptance open. Do not rerun already accepted ordinary switching as a substitute. Individual Undo evidence does not close DCM-37/44/45 or full-provider DCM-28/29.

## Execution and identity

Code locally -> GitHub -> FlowDev update/build/test on Lenovo; one native suite at a time. Workspace `D:\FlowverseReportsRepo`; checkout `D:\FlowverseReportsRepo\dcm-quicksort-candidate`; profile `tools\flowdev-dcm`; runner `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`; device `d18227f2-0b54-4b11-9026-56ac40305f02`. No remote source edits; leave the separate LibreOffice session untouched.

Sumatra selectors require exact DCM_BUILD_RECEIPT and DCM_SUMATRA_SOURCE_EXE=`D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe`. Harnesses copy to private labs with the unchanged no-print/no-write policy. Last build `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-131140-4a0e53b7\receipt.json` matches cc04ae7c only. Rebuild after metadata/source changes; never rewrite receipt identities.

Reports under workspace: reviewed Undo `dcm-quicksort-ui-sumatra-undo-20261112-131419-c530aae7`; repeat `...-20261112-131747-1db5aef1`; baseline `dcm-quicksort-baseline-20261112-131448-2a7ddc9c`; Refresh `dcm-quicksort-ui-refresh-20261112-131625-5574d080`; lifecycle `dcm-quicksort-ui-sumatra-lifecycle-20261112-131711-6f3ca1c6`. Full paths/hashes, earlier failures and review in `evidence/sumatra-undo-native-summary.json`.

Keep DCM_REAL_PPTX prerequisite, DCM-19 isolation, DCM-UI-LABEL-01 (Size/Sort/Processed crowding), runtime distribution and packaged setup gates. No default switch, printing, association change or installed-working-copy replacement.
