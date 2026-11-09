# DCM v1.0.71 — resume after DCM-06

Latest checkpoint: [DCM-06 native GREEN](2026-11-09-dcm06.md). Earlier records in `2026-11-09-progress.md` and `2026-11-09-baby-steps.md` remain historical evidence.

## Current state

Source/tested commit: `c88abef150de794a028b466b6a0a28a5fab7707d` on `release/v1.0.71`. Later documentation commits do not change this tested source identity. Draft PR #1 remains open; no release or installer is qualified.

The no-overwrite proposal is now applied to both core and normal UI fast paths. `TestD2MoveFileRefusesExistingForeignDestination` passed natively. The selected group completed **20 passed / 1 failed / 0 skipped**, no timeout, with unchanged source. Its overall exit is still 1.

**Next: DCM-07.** `TestD6MovedDuplicateSurvivorRequiresAttentionAfterRestart` is the sole remaining failure in this selected group. Review and apply `proposed-fixes/duplicate-kind-gate.patch` locally. The assertion RED prerequisite exists in both the earlier `20261109-133423-16576b74` report and the latest `20261109-141655-4b4f9621` report. Do not reapply `no-replace.patch` or mistake the earlier zero-test timeout for RED.

## Native runner

Device: `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`.

Candidate: `C:\FlowverseReports\dcm-quicksort-candidate`, origin `https://github.com/laxmanxp/dcm-quick-sort.git`.

Set `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports`, run from the candidate's `tools\flowdev-dcm` directory, and use `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.

Use `flowdev update dcm-quicksort`, then `flowdev test dcm-quicksort regressions`. Record the actual updated HEAD. Code locally, store in GitHub, and use Lenovo only for validation. Do not use the older monorepo profile or dirty v1.0.62 checkout. Run one native selector at a time.

Latest native report: `C:\FlowverseReports\dcm-quicksort-regressions-20261109-141655-4b4f9621`. Its receipt, summary and relevant raw stdout were retrieved. A compact summary is in `evidence/dcm-06-native-summary.json`.

The last resource observation (13:32 UTC, not a current guarantee) showed 2,271,277,056 bytes free on C: and 1,920,344,064 bytes available RAM. The later bounded regression run completed. Recheck with `host-diagnostics` if stalls return. Never delete user files or close unrelated applications automatically.

## After DCM-07

Continue `completion-50.json`, not a UI migration. Preserve original defect IDs in `../todo.json`. All affected evidence must be tied to the candidate tested, and source, UI and user acceptance remain separate.

The remaining sequence includes final `baseline`, `installer-unit`, `ui-tooling`, and `package`; exact app/setup and embedded-payload hashes; isolated installer lifecycle; normal-app filters, navigation, Preview/Rename/Move, conflict and restart Undo workflows; and actual reviewed images. Bind `DCM_BUILD_RECEIPT` for UI selectors and `DCM_PACKAGE_RECEIPT` for `installer-ui-cancel`. Read `../tools/README.md` and `../tests/native/README.md` before running those gates.

`visual_reviewed=false` is not visible-content acceptance. No user acceptance is recorded. Keep private configuration/backup/registry isolation intact; actual Explorer registration requires a disposable Windows user or VM. Update evidence and release notes before `release-prepare`; any `release-draft` requires the exact expected package commit. Do not merge or publish until all required gates and publication approval are complete.
