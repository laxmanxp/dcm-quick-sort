# DCM v1.0.71 — resume after DCM-25

Latest checkpoint: [DCM-25 complete](dcm25-complete.md). Preserve all prior records, including DCM-24's interrupted-work report and each native RED.

Completed: **DCM-01–18 and DCM-20–25**. Only DCM-25 newly closed. DCM-19 remains blocked. The 24 completed checklist steps include five setup/diagnostic steps, not a product percentage. User acceptance remains unrecorded.

## Verified state

Product/native-tested revision: `1554c9b87bbb834b14d54b231eae4b4ece27b8d2`. Native RED at 03c9be04 proved failed-source Refresh retained old list/selection/draft/preview. The failed-read branch now clears cached targets, uses existing preview cleanup and shows Source unavailable with recovery guidance. No installer/preview engine change. Do not reapply the old proposal.

Native ui-refresh: **23 successful records, nine separately reviewed images**, source/fixture-state integrity and normal exit 0 with children drained. Actual Browse switches between A, B and empty; same basenames with different contents remain distinct. Representative B-only Rename/Move leaves A intact. A controlled directory move/recovery checks unavailable-source safety. Harness creates/deletes/moves only declared disposable fixtures; these mutations are explicitly accounted for.

Full baseline: **358 passed / 0 failed / 1 external-PPTX skip**, 359 Go results, exit 0, no timeout, source unchanged. No new Go tests; native UI regression added. Fresh ui-selection and ui-launch commands PASS; their new images were not separately reviewed. Preserve the three retained harness failures and actual product RED, documented in the checkpoint.

## Next exact step — DCM-26

Inspect the advertised built-in/native text preview path, supported extensions, size limits and encoding behavior. Reuse source-bound FlowDev/native tooling with disposable fixtures, verify actual visible text and truthful empty/error/unsupported states, and inspect images before closing the step. Do not infer every text format from earlier TXT fixtures or expand preview scope silently. Keep selection/Next/Refresh safety intact.

Code locally -> GitHub -> FlowDev update/test on Lenovo, serially. Candidate `C:\FlowverseReports\dcm-quicksort-candidate`, profile `tools\flowdev-dcm`, runner `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`, workspace `C:\FlowverseReports`, device `d18227f2-0b54-4b11-9026-56ac40305f02`. No remote source editing. Check for another app's active UI test before using desktop input.

Exact build receipt: `C:\FlowverseReports\dcm-quicksort-build-20261111-132602-da7ecfec\receipt.json` matches 1554c9b8 only. App SHA-256: `b6353d846a8c58c167032dc06a873e15e48273dbdbd68e67758877bd8b364466`. Obtain a fresh source-matching build after metadata/source changes; never edit old receipts.

Reports under `C:\FlowverseReports`: refresh `20261111-132617-5e4dccbb`; baseline `20261111-132539-4f1de17d`; selection regression `20261111-132717-40bc10d4`; launch regression `20261111-132739-cf097855`; product RED refresh `20261111-132238-b590b5c8`. Full paths and hashes are in `evidence/dcm-25-native-summary.json`.

Keep DCM-19/disposable Explorer isolation, DCM-UI-LABEL-01, identified DCM_REAL_PPTX, remaining previews/workflows, full Batch Move/Undo, final package and actual setup/uninstaller lifecycle open. Leave DCM21_PROBE_EMPTY_DESTINATION unset. Previous native results retain original source identities; Shell Cut/Delete remain outside DCM Undo. PR #1 stays draft; no merge/publication without remaining gates and approval.
