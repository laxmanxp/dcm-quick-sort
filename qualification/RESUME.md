# DCM v1.0.71 — optional Sumatra provider, stage 1

Latest work: [Sumatra first native milestone](2026-11-12-sumatra-stage1.md).
Last completed numbered checkpoint remains DCM-27. Completed DCM-01–18 and DCM-20–27; DCM-19 blocked; DCM-28/29 pending. No numbered step closed here. User explicitly approved this focused optional PDF-provider integration. Do not resume the separate LibreOffice pane-divider work.

## Verified state

Native-tested source `b05024332f018b544f7d1e704c5e017fac66f364`. Full native baseline **365 pass / 0 fail / 1 retained external-PPTX skip**, 366 parents, exit 0, no timeout. Three new policy/argument/document-preflight tests. The existing Windows.Data.Pdf provider remains default; Sumatra is explicitly opt-in through DCM_PDF_PROVIDER=sumatra and a pinned absolute DCM_SUMATRA_EXE.

Sumatra ui-sumatra stage 1 passes eight checks: owned restricted embedded viewer, two actual pages, last-page boundary, Previous, internal Open blocked, switch-to-text/exclusive file release, representative tracked Rename/Move with bytes intact. Six full images/five content crops reviewed. App normal exit 0; all three recorded owned viewer processes exited. Fresh default-provider ui-text (17) and ui-image (23) pass; their new images not separately reviewed. Raw screenshot flags remain unchanged.

**Open visual blocker DCM-SUMATRA-UI-01:** page label/input clipped at toolbar top; redundant disabled QuickSort sizing controls remain above the embedded viewer. Keep PrinterAccess=0 and DiskAccess=0; do not enable printing for geometry. Do not make Sumatra default or call DCM-28 complete.

## Exact next work

Inspect and reproduce toolbar geometry under the strict no-print policy, add a focused failing UI assertion, and correct it without weakening policy or ownership guards. Then finish page-boundary/control and zoom/resize qualification, rapid switching/cancellation, failure states, Undo and release failure/active-PDF close. Missing runtime/corrupt/password paths and abnormal exit are not yet native-qualified. Maintain old provider for comparison. Packaging/distribution review DCM-SUMATRA-DISTRIBUTION-01 stays open; no binary has been bundled or published.

The first no-provider RED is `dcm-quicksort-ui-sumatra-20261111-232240-30170f6f`. First implemented run at185ba410 timed out selecting 08-preview.pdf and capture was occluded; cause not established. Passing final run followed added diagnostics and asynchronous child-hide change; do not retrospectively claim that caused the earlier timeout. Revalidate repeatability.

## Workflow and evidence

Code locally -> GitHub -> FlowDev update/test on Lenovo, serially. Canonical workspace `D:\FlowverseReportsRepo`; candidate `D:\FlowverseReportsRepo\dcm-quicksort-candidate`; profile `tools\flowdev-dcm`; runner `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`. Device `d18227f2-0b54-4b11-9026-56ac40305f02`. C:\FlowverseReports is an alias; use D: paths. No remote source edits.

`ui-sumatra` requires DCM_SUMATRA_SOURCE_EXE pointing to the existing prototype runtime `D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe`; it copies it into the private test lab and writes its own policy. Do not edit the prototype runtime/policy or interact with the unrelated LibreOffice session. Runtime hash719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0.

Last exact build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261111-233847-d6a39720\receipt.json`, matches b0502433 only. App hash6e0da23dbd7bf67d5610e9e82b7859df851690b11caeeb38c23284e61323d984. Rebuild after any later source/metadata commit; never rewrite receipts.

Native GREEN: `dcm-quicksort-ui-sumatra-20261111-233859-93834514`; baseline `dcm-quicksort-baseline-20261111-234138-747a8dff`; text `dcm-quicksort-ui-text-20261111-234201-8b3fe3f2`; image `dcm-quicksort-ui-image-20261111-234218-04fe2476`, all under D:\FlowverseReportsRepo. Separate evidence and hashes: `evidence/sumatra-stage1-native-summary.json`.

The general native Snapshot helper rejects foreign child PIDs; use the bounded Sumatra-specific inspector for cross-process children rather than relaxing the global guard. Pixel comparison must use content crops, not blue title/list chrome. Ownership/capture/content assertions remain mandatory.

Keep prior DCM-19, layout, external DCM_REAL_PPTX, full operations/Undo, final installer/package and human acceptance gates. PR #1 stays draft, no merge/publication. Preserve all original issue IDs, source identities and historical evidence.
