# Sumatra Undo and controlled viewer failure — verified

12 November 2026 · DCM QuickSort v1.0.71 · branch `release/v1.0.71`.
Tested source: `cc04ae7cd8932cffc4e13d7e181ae51eb78095fc`.
Product correction: `0c9cdf87084bd9f18c0e062bee9f40ce8f6f0ace`.

## Result

Two consecutive completed `ui-sumatra-undo` runs passed all **nine acceptance records**. All **eight full-window screenshots from the first passing run** were opened and inspected. Its six content crops were checked automatically, not separately visually inspected. Repeat and regression images were not separately reviewed. Raw visual flags remain unchanged.

The full native application suite passed **368 parent tests, zero failures, one skip** (369 parent results). The new Windows release-refusal parent test passed all six nested cases. Refresh/folder-switching and PDF lifecycle regressions also passed. No numbered checklist step is closed: DCM-01–18 and DCM-20–27 retain their recorded completion; DCM-19 stays blocked and DCM-28/29 remain pending. Sumatra stays optional; no package/default switch or release.

## Actual product defect, reproduced before correction

At `55e2b877`, the test renamed a previewed PDF, undid Rename, moved the original PDF, opened its destination folder and previewed the moved file. **Undo Move correctly restored the source bytes and updated the original journal transaction to undone, but the destination view retained its old file row, selection and filename draft.** The viewer had exited, leaving a generic blank Preview. The test failed `destination clears stale selection after Undo`; its failure screenshot was inspected. No wrong-file operation or data loss was observed or claimed.

`undoSelectedProcessed` refreshed the list only when the currently viewed directory matched the original source directory. The minimal correction refreshes when the active directory matches **either** the original source or destination. The passing screenshot now shows a genuinely empty destination, zero files, no selected filename and the normal empty-folder message. Reopening the original source displays the restored PDF. Track this as **DCM-UNDO-REFRESH-01**, fixed for the individual-Undo workflow exercised here. Batch Undo and all path-alias variants are not certified by this result.

## What the native workflow verifies

| Scenario | Evidence |
|---|---|
| Active-PDF Rename and Undo Rename | Old viewer released; renamed document reopens; Undo restores original name and SHA-256; original transaction ID becomes undone and disappears from Processed. |
| Open moved PDF from destination, then Undo Move | Viewer released; original file and bytes restored; destination file removed; same transaction ID becomes undone. |
| Destination view after Undo | Zero file rows, cleared filename draft and no stale viewer; original PDF allows exclusive access. |
| Controlled viewer failure | Only the test's recorded private viewer is forcibly terminated after verifying executable, PID, start time and parent. Host reports viewer exit, without changing the PDF or journal. |
| Reload after viewer exit | A new owned viewer displays the correct original PDF content. |
| Normal close | Host exit 0; no recorded owned viewer or session profile remains; exclusive access succeeds; original four-file set restored and both original transactions remain undone. |

The controlled failure is a forced process exit, **not** a reproduced decoder crash. The test operates on one generated two-page PDF and three text fixtures, with private application settings/runtime/policy. User files, installed copies, global associations and the unrelated LibreOffice session are untouched.

## Release refusal: six Windows unit cases, separate from UI acceptance

`TestSumatraReleaseFailurePreservesOperations` tests Rename admission, Move admission and individual Undo under two test-local failure states: an invalid wait handle, and an unsignalled event plus an invalid job handle causing termination to fail. The production operation handlers refuse admission, retain the unconfirmed viewer session/profile, surface an error and leave file bytes, paths, history and durable journal unchanged.

The tests use real Windows API failure returns but injected test-local handle state. They do **not** alter a running user's process permissions or add a production failure switch. They do not establish every release failure, successful termination followed by an exhausted exit wait, or an end-to-end GUI denial of termination against a live Sumatra process. One parent contains the six cases; those cases are already included in the baseline and are not added again to its total.

## Fresh results at cc04ae7c

All report directories below are under `D:\FlowverseReportsRepo`.

| Selection | Result | Report directory |
|---|---|---|
| Full native baseline | **368 passed / 0 failed / 1 skipped**, exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261112-131448-2a7ddc9c` |
| Reviewed Undo/failure run | **9 passed**, normal exit 0, eight full images reviewed | `dcm-quicksort-ui-sumatra-undo-20261112-131419-c530aae7` |
| Unchanged Undo/failure repeat | **9 passed**, normal exit 0; images not separately reviewed | `dcm-quicksort-ui-sumatra-undo-20261112-131747-1db5aef1` |
| Refresh/folder switching | **23 passed**, FlowDev exit 0, source unchanged; images not separately reviewed | `dcm-quicksort-ui-refresh-20261112-131625-5574d080` |
| PDF switching/cancellation/shutdown | **11 passed**, exit 0, source/fixtures unchanged; images not separately reviewed | `dcm-quicksort-ui-sumatra-lifecycle-20261112-131711-6f3ca1c6` |

The retained skip is `TestRealGeneratedPPTXIntegration`, not a pass. One new release-refusal parent explains the increase from 367 to 368 passes. UI records, repeat runs and nested cases are separate counts. Installer, package, old-provider PDF UI and full Batch Undo were not rerun. Local focused tests and Windows test cross-compilation passed; a broader Linux Undo selection encountered existing Windows-only guard limitations and is not represented as a full local pass.

## Startup timeouts and test corrections are retained

Before reaching Undo, two runs at `7b0aca98` timed out on the harness's synchronous selection notification (native message 273). The test-only input helper now posts the real list-selection notification, retaining exact selected-row, draft, document identity and pixel assertions. Two later runs at product-fix source `0c9cdf87` timed out querying filename text during initial PDF startup (message 14). The helper now polls only that owned filename control with short bounded queries inside the existing 20-second readiness deadline. This is a test input/readiness correction, **not** an application startup-performance fix.

Initial filename readiness in the two passing runs was **1,437 ms and 1,610 ms**; subsequent recorded selections ranged from 340 to 543 ms across the passing/preceding partial run. These limited observations neither establish general responsiveness nor explain every earlier timeout. **DCM-SUMATRA-STARTUP-01 remains open** to measure the actual startup costs and responsiveness rather than hiding them with longer waits.

An earlier final-source run reached seven successful records, including Undo and controlled viewer exit, but failed when the capture guard found the window minimized during Reload recovery. It remains a failed run; an unchanged retry completed. The cause of minimization was not established and capture guards were not disabled.

Preserved failures:
- `124531-b1b33c46` and `124705-2ef93a76` on 20261112: synchronous selection deadlines at `7b0aca98`.
- `125742-ee1f4895`: actual stale-destination RED at `55e2b877`; both file reversals and journal updates had succeeded, normal cleanup 0.
- `130158-54c9fdd9` and `130457-ec8ed176`: initial text-query deadlines at `0c9cdf87`, before acceptance records.
- `131234-d8f1e1a0`: final-source capture failure after seven records; normal cleanup 0.

All use the `dcm-quicksort-ui-sumatra-undo-20261112-` report prefix. Raw receipts/screenshots remain unmodified. Full-window screenshots contain desktop edges and remain on Lenovo; only review/hash summaries are saved to GitHub.

## Exact identities and continuation

Build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-131140-4a0e53b7\receipt.json`.
Receipt SHA-256: `5e4d210d3c78e2a6b8c36ed524ecedac4e88e72f71381562142837dd9763458e`.
App SHA-256: `81c516b0a84442dec493f21ef1ff3562b988eb66fc76c0c177959c3afcb3bdb0`; 7,230,976 bytes.
318-file source-manifest SHA-256: `e6899c708207f4c62d56cd429a03326d6b01d40a4dc0628fd4b6b8341169825a`.
Runtime/policy hashes are unchanged from previous milestones.
Evidence summary: `qualification/evidence/sumatra-undo-native-summary.json`.

Next: valid-header damaged/password-protected PDF and missing-runtime handling, while retaining startup responsiveness, readable text after scrolling at 100%/Fit Width, general labels, DCM-19, external PPTX, remaining Batch Undo/path variants, runtime distribution/installer and human acceptance gates. No numbered closure, default switch, printing, merge or publication. Rebuild after later metadata/source commits; do not rewrite receipt identities.
