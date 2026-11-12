# QuickSort — bounded Sumatra lifecycle verified; remaining provider checks

Latest: [switching, cancellation and shutdown](2026-11-12-sumatra-lifecycle.md).
Tested source `deb8373dbb52c44d963f20fce49b9ea074f7d59e`. Application unchanged from `24ff2f24`.
No numbered closure: DCM-01–18 and DCM-20–27 complete; DCM-19 blocked; DCM-28/29 pending. Keep Sumatra optional and old Windows.Data.Pdf default. Manual acceptance unrecorded.

## Completed this pass

New `ui-sumatra-lifecycle` passed 11 acceptance records twice. Each run has 40 native selections, ten abandons all observed while the host reported loading, four attached A-B-text cycles, truthful empty/invalid-header errors and recovery, and normal close with an active PDF. No private viewer process/session directory remains; both PDFs permit exclusive read access; seven fixture names/bytes and empty operation journal preserved. Recorded close times 393/374 ms are observations, not guarantees.

All seven full-window images plus three content crops of the first run were inspected. The repeat and regression images were not separately reviewed. Baseline is freshly 367 pass / 0 fail / 1 external-PPTX skip; existing ui-sumatra 23 records also passed. Only test tooling was added; no new product defect or repair claimed. General layout issues stay open.

## Next exact scope

Qualify individual Undo while a PDF is active/reopened, then safely exercise viewer failure and release-refusal, valid-header corrupt/password-file handling and missing runtime. Retain readable text after scrolling at 100%/FitWidth and end-user provider/setup/distribution work. Do not rerun already accepted ordinary switching as a substitute for these missing paths. Cancellation here means changing the selected file while loading, not a dedicated Cancel button or every decoder phase. Do not switch the default or close DCM-28/29 until the remaining acceptance is explicitly resolved.

## Workflow

Author locally -> GitHub -> FlowDev update/build/test on Lenovo, one native suite at a time. Workspace `D:\FlowverseReportsRepo`, checkout `D:\FlowverseReportsRepo\dcm-quicksort-candidate`, profile `tools\flowdev-dcm`, runner `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`. Device `d18227f2-0b54-4b11-9026-56ac40305f02`. No remote source edits. Leave the separate LibreOffice interactive session untouched.

Both `ui-sumatra` and `ui-sumatra-lifecycle` require exact DCM_BUILD_RECEIPT plus DCM_SUMATRA_SOURCE_EXE=`D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe`. Each harness makes a private runtime/policy copy; no global associations/settings. Do not relax the pin or no-print/no-write policy.

Build `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-093555-d16a9c40\receipt.json` matches deb8373d only. App SHA `a2bd4063a063431b8267a0ff50ed9157caea74ef3b965b391a6adc08d393a463`, unchanged. Refresh build after metadata/source commits; never rewrite historical receipts.

Reports under D:\FlowverseReportsRepo: reviewed lifecycle `dcm-quicksort-ui-sumatra-lifecycle-20261112-093613-56838102`, repeat `dcm-quicksort-ui-sumatra-lifecycle-20261112-093758-712f52cb`, baseline `dcm-quicksort-baseline-20261112-093728-3abdf19f`, existing regression `dcm-quicksort-ui-sumatra-20261112-093827-5a51793c`. Separate receipt/image review: `evidence/sumatra-lifecycle-native-summary.json`.

Preserve original resize reports/failures, capture guards, scoped native ownership and all earlier IDs. DCM-UI-LABEL-01 remains open, including Size/Sort and Processed crowding. MSAA toolbar coordinates alone do not prove containment. Retain DCM-19, DCM_REAL_PPTX, batch/Undo/release-failure, distribution, installer/package and human acceptance gates. PR #1 draft, no merge/publication.
