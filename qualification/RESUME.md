# DCM v1.0.71 — resume after DCM-12

Latest checkpoint: [DCM-12 naming-template import/restore](2026-11-09-dcm12.md). Preserve all earlier checkpoints and their original tested revisions.

## Current state

Latest native-tested repository commit: `4619e0ef2bed51b7685c5f84d0fc0f87702bb1ca` on `release/v1.0.71`. Application implementation remains unchanged from `26eeaed3`; DCM-11 and DCM-12 added regression coverage only. Later documentation commits are not the tested revision.

Fresh safety selection: **23 passed / 0 failed / 0 skipped**, exit 0, no timeout, source unchanged. Three D4 parent tests and five nested cases verify valid archive import/latest restore plus altered archive templates, truncated archives and altered latest backups. Invalid-package checks preserve the current fixture's complete file/directory paths and bytes. All earlier selected safety tests remain green. This is not the full app suite or visible UI acceptance.

Tracker: DCM-01–12 complete at their stated boundaries, including five setup/diagnostic steps. Only DCM-12 was closed in this checkpoint. **Next: DCM-13 — installer preference preservation.** Read installer regression assertions and the native runner instructions, then use the registered `installer-regressions`/`installer-unit` routes as appropriate. Do not silently count application import tests as installer lifecycle acceptance.

Both proposed product patches (`no-replace.patch`, `duplicate-kind-gate.patch`) are already applied. Do not reapply them. For a real defect use assertion RED -> smallest local fix -> GitHub -> FlowDev native GREEN. Tests of existing correct behavior may pass without a product change; do not invent a RED.

## Native runner

Device: `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`.
Candidate: `C:\FlowverseReports\dcm-quicksort-candidate`, origin `https://github.com/laxmanxp/dcm-quick-sort.git`.
Set `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports`, run from `C:\FlowverseReports\dcm-quicksort-candidate\tools\flowdev-dcm`, and use `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.

Use `flowdev update dcm-quicksort` before registered `flowdev test dcm-quicksort <selector>` runs. Run one selector at a time. Code locally, store in GitHub, validate on Lenovo. Do not edit source on Lenovo or use another app's profile.

Latest report: `C:\FlowverseReports\dcm-quicksort-regressions-20261109-170738-0bc02f13`. Complete receipt/summary and D4 stdout lines 174–207 were reviewed. Compact evidence: `evidence/dcm-12-native-summary.json`. Original logs remain on Lenovo.

The 13:32 UTC resource measurement is historical, not current host certification. The 17:07 UTC native regression completed. Use `host-diagnostics` if stalls return; never delete user files or close unrelated applications automatically.

## Remaining qualification

Keep original defect IDs in `../todo.json` and all 50 stable IDs in `completion-50.json`. Code, UI and user-test status remain separate. The installer suite's earlier 24-test PASS at `7c413fd` is not final-candidate qualification.

Remaining gates include full `baseline`, `installer-unit`, `ui-tooling`, final `package`, exact app/setup and embedded-payload hashes, isolated installer lifecycle/Explorer registration, visible navigation/preview/rename/move/conflict/restart-Undo workflows and reviewed images. Read `../tools/README.md` and `../tests/native/README.md` before those gates. Bind `DCM_BUILD_RECEIPT` for UI and `DCM_PACKAGE_RECEIPT` for setup Cancel.

No user acceptance is recorded. `visual_reviewed=false` is not visual acceptance. Preserve private configuration/backup/registry isolation; actual Explorer registration requires a disposable Windows user or VM. Keep PR #1 in draft. Do not merge or publish until required gates and explicit publication approval are complete.
