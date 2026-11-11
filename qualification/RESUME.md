# DCM v1.0.71 — resume after DCM-27

Latest accepted checkpoint: [DCM-27 image previews](dcm27-complete.md). Completed: **DCM-01–18 and DCM-20–27**; DCM-19 remains blocked. Only DCM-27 newly closed. The 26 checklist steps include five setup/diagnostic steps, not a product percentage. Human acceptance remains unrecorded.

## Verified state

Native-tested revision: `b92c9abdac87e7f9c937e51ca8e7023eae5e3417`. Image-error fix `4043f3e5` and the image tools were already committed by the interrupted attempt; this retry made no further product/test changes. Empty image previously showed a missing Windows handler and stale success footer; the fixed image loader retains specific errors and disables sizing controls. Existing GDI+ memory-stream loading and the PDF bool wrapper remain intact.

Fresh baseline: **362 passed / 0 failed / 1 retained external-PPTX skip** across 363 parent tests; exit 0, no timeout, source unchanged. Image workflow: 23 checks on 12 fixtures, 22 individually reviewed screenshots, normal exit 0, children drained and original bytes unchanged. All 22 captures succeeded on first attempt. Nineteen render cases verify actual screenshot pixels, plus three error states and source-handle release. Text and navigation regression commands also passed; their new images were not separately reviewed.

See `dcm27-complete.md` and `evidence/dcm-27-native-summary.json` for exact runs, hashes, historical failures and scope. Static GIF/single-page TIFF only; no animation/multipage TIFF, general EXIF/color-profile, all DPI or human-input claim. Panning uses owned native messages.

## Next exact step — DCM-28

Qualify a known multipage PDF through the actual candidate: verify each visible page and next/previous at first/last boundaries. Reuse the existing native tools and PDF path. Do not add printing or replace the renderer. DCM-29 zoom is separate. Obtain a fresh matching build receipt before UI input; no arbitrary newest binary or edited receipt identities.

## Canonical Lenovo workspace

Use **D:\FlowverseReportsRepo**. `C:\FlowverseReports` is a junction to it and FlowDev update through that alias returned `not a repository root`. Do not undo the user's relocation or mix aliases when binding source/build identity.

Candidate: `D:\FlowverseReportsRepo\dcm-quicksort-candidate`.
Profile: candidate `tools\flowdev-dcm`.
Runner: `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`.
Set `FLOWDEV_WORKSPACE_ROOT=D:\FlowverseReportsRepo` and `DCM_FLOWUIVISION_EXE=D:\FlowverseReportsRepo\dcm-flowuivision-20261103\FlowUIVision.exe`.
Device: `d18227f2-0b54-4b11-9026-56ac40305f02` (Laxman_Lenovo).
Code locally -> GitHub -> FlowDev update/test on Lenovo. Check active sessions before any native UI run; one UI test at a time. No remote source edits.

Last successful build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261111-180146-fc67e68b\receipt.json`, matching b92c9abd only. App SHA-256 `2f579ceba816c1f48853a688fe18ba0e48589f0877deaca3ca836dd47b25d5c2`. A later documentation commit also requires a fresh build for exact receipt binding.

Keep DCM-19/disposable Explorer isolation, DCM-UI-LABEL-01, identified DCM_REAL_PPTX before DCM-32/47, remaining workflows/previews, final package/actual installer lifecycle and human acceptance open. Preserve historical C: receipt strings and all prior evidence. Leave DCM21_PROBE_EMPTY_DESTINATION unset. No merge/publication without final gates and approval.
