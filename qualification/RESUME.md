# DCM v1.0.71 — resume after DCM-20; DCM-19 remains blocked

Latest checkpoint: [DCM-20 native context menu](2026-11-11-dcm20.md). Preserve all earlier evidence and source identities.

Completed at recorded boundaries: DCM-01–18 and DCM-20. Only DCM-20 was newly closed. DCM-19 remains pending under DCM-EXPLORER-ISOLATION-01; no host registration/installation was substituted. The first five completed steps are setup/diagnostic work, not a product percentage.

## Verified state

A native RED at `56819705` proved this candidate lacked a file-list context menu. Product fix `ded8595377d4cf187a07624b3a869cdd05494fb7` adds `app-src/context_menu_windows.go` and three Windows Go tests. It wraps the existing callback, binds Windows IContextMenu to the clicked row, rejects blank areas, distinguishes cancellation from command zero and preserves the normal handler for unrelated messages. Do not reapply historical patches.

Full native application suite at `ded8595`: **354 passed, 0 failed, 1 external-PPTX skip** across 355 top-level results. The original 351 passing tests remain; three new tests account for the increase. Later commits change only the UI-test script/adapter, not product or Go-test source.

At `fa696e12dddacefcc243941f770e3599bd8ab2ff`, `ui-context` passed four acceptance cases and all five screenshots were reviewed. Menu target/Cancel/blank-area checks preserve file bytes. Actual ordinary Open on a harmless command-file fixture produced only its unique intended marker. This is representative default .cmd handler coverage, not every document association or all Shell commands. The menu is single-file; Shell file-changing commands are outside DCM journal/Undo. No DCM printing pipeline was added.

The unchanged `ui-launch` regression passed 12 cases, two normal exit-0 closes and four separately reviewed images. All checked source/fixture identities stayed unchanged. Raw visual flags remain false; separate acceptance is in `evidence/dcm-20-native-summary.json`.

The test adapter's physical Open click originally refused mismatched logical/physical coordinates. Diagnostic evidence was retained. It now uses the popup DPI context, MenuItemFromPoint, documented logical-to-physical conversion and unchanged ownership/occlusion checks. Do not remove those guards or replace actual menu invocation with a direct process launch.

## Next exact runnable action — DCM-21

Review and exercise filters using disposable fixtures. Resolve DCM-UI-EMPTY-COPY-01: a filter with no matches must not say the physical folder is complete. Reproduce through a focused test, make the smallest change locally, store it in GitHub, update/test with FlowDev, and inspect the actual screenshots. Preserve no-selection file safety, Clear Filters recovery and the native menu integration. DCM-UI-LABEL-01 remains a separate open visual finding.

Use workspace `C:\FlowverseReports`, candidate `C:\FlowverseReports\dcm-quicksort-candidate`, branch `release/v1.0.71`. Run from `tools\flowdev-dcm` using `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`. Device `Laxman_Lenovo`: `d18227f2-0b54-4b11-9026-56ac40305f02`. Author locally -> GitHub -> FlowDev update/test on Lenovo. No remote source edits. One native UI run at a time.

The recorded build receipt matches only `fa696e12`; documentation commits also change the required source identity. Rebuild after update and set DCM_BUILD_RECEIPT to that exact successful build receipt. Never pick an arbitrary newest binary or edit receipt identities.

## Exact evidence and pending gates

Context: `C:\FlowverseReports\dcm-quicksort-ui-context-20261111-011755-9e356a18`.
Launch regression: `C:\FlowverseReports\dcm-quicksort-ui-launch-20261111-011912-47783da7`.
Baseline: `C:\FlowverseReports\dcm-quicksort-baseline-20261111-005651-f0924008`.
Build: `C:\FlowverseReports\dcm-quicksort-build-20261111-011606-8eb3acc9\receipt.json`.
App SHA-256: `500bbda9dfe4f5798cff838c182305f3f4c874d12ac5fbd3ee2737b45e9f715c`.

DCM-19 still needs a verified disposable Windows account/VM and real Explorer entry invocation; the in-app popup's existing host extension labels do not close it. Do not repeatedly run isolation discovery without an environment change. External DCM_REAL_PPTX fixture remains required for DCM-32/47. Installer units/lifecycle retain their historical revisions and were not rerun in DCM-20; packaged setup/uninstaller E2E is still required. Keep UI findings, final package and user acceptance open. PR #1 remains draft; no merge/publication without all final gates and explicit approval.
