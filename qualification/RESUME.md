# DCM v1.0.71 — DCM-19 blocked; next independent step DCM-20

Latest checkpoint: [DCM-19 discovery and registration contract](2026-11-11-dcm19.md). Preserve all earlier source/test identities. DCM-01–18 remain complete at their defined boundaries, including five setup/diagnostic steps. **No numbered step was newly closed. DCM-19 remains pending.**

## Current verified state

At `e1a8cc6c9ab8765263b8c163494f005ca80352b3`, the registered `explorer-isolation` read-only discovery completed without query errors and with unchanged source. Lenovo reports Windows 11 Home Single Language build 26200, no returned Sandbox feature or system Sandbox executable, no Hyper-V module/VM management service, and no listed VM launcher on PATH. No approved disposable account/VM is configured in this DCM profile. This is not an exhaustive inventory and does not prove no VM exists elsewhere. Do not assume an existing account or VM is disposable.

Fresh installer units: **26 passed, 0 failed, 0 skipped**, exit 0, no timeout, source unchanged. The unchanged `TestExplorerShellEntriesUsePerUserFileFolderAndBackgroundTargets` passed. The actual installer label is **Sort with DCM Quick Sort** for file, folder and folder-background entries. Commands quote the app plus `%1` (file/folder) or `%V` (background). Code-generated values are verified; real Explorer clicks are not.

Only discovery tooling and its registration were added. No app/installer implementation or previous assertions changed. No host setup, live registry edit, new account, feature enablement, reboot, VM startup, Explorer UI action or screenshot acceptance occurred. The receipt deliberately separates discovery success from `feature_accepted=false`.

Blocker: **DCM-EXPLORER-ISOLATION-01** in `todo.json`. DCM-19 and the related packaged installer gate require approved disposable Windows isolation; never substitute live host installation/registration or in-process registry checks. Keep the 50-step checklist's DCM-19 pending.

## Next exact runnable action — DCM-20

Inspect the current app's right-click selection/command path and existing test coverage. Add only the needed focused acceptance tooling locally, commit it, update through FlowDev, run tooling/build as necessary, and use a current exact `DCM_BUILD_RECEIPT`. Exercise the in-app menu against disposable files, verify that the row under the pointer is the operation target, and verify external Open using an identified owned fixture. Check screenshots and cleanup separately. Do not add printing or close unrelated steps. DCM-19 stays blocked rather than being silently waived.

Use workspace `C:\FlowverseReports`, candidate `C:\FlowverseReports\dcm-quicksort-candidate`, branch `release/v1.0.71`. Run from its `tools\flowdev-dcm` using `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`. Device `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`. Code locally, store in GitHub, update then test on Lenovo. No remote source edits. One native UI run at a time.

New registered selector: `explorer-isolation`, read-only discovery only. Do not rerun it repeatedly expecting isolation to appear without an environment change.

## Evidence and preserved gates

Discovery: `C:\FlowverseReports\dcm-quicksort-explorer-isolation-20261110-194731-af6004b0`.
Installer units: `C:\FlowverseReports\dcm-quicksort-installer-unit-20261110-194833-ca9af37a`.
Tooling: `C:\FlowverseReports\dcm-quicksort-ui-tooling-20261110-194729-5bd2d103`.
Compact evidence: `evidence/dcm-19-isolation.json`.

DCM-18 Browse remains pinned to `a3e4c185` with nine cases/six reviewed images. The app baseline remains 351 passed / 0 failed / 1 external-PPTX skip at `13f52cb0`; supply the identified `DCM_REAL_PPTX` fixture before DCM-32/47 acceptance. The in-process installer lifecycle at `a49e6595` is not packaged setup/uninstaller E2E. Do not add these historical counts to fresh results.

The incidental installer-unit app hash remains `a10e368eef249dbb38b340c52cfc8e2c2cb583758d14f07d09778c5ab9ee98c5`; do not use its installer-unit receipt as a build receipt. After any later commit, obtain a fresh matching build for UI use. Keep DCM-UI-EMPTY-COPY-01 and DCM-UI-LABEL-01 open. Final package, actual Explorer/package lifecycle and user acceptance remain pending. Keep PR #1 draft; no merge or publication without final gates and explicit approval.
