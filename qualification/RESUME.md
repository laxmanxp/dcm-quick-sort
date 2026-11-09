# DCM v1.0.71 — resume after DCM-10

Latest checkpoint: [DCM-10 original-byte duplicate Undo review](2026-11-09-dcm10.md). Earlier checkpoints retain their original tested identities.

## Current state

Product/native-tested commit: `26eeaed3b9041a31b17c197ef2a095f53ee2a9e6` on `release/v1.0.71`. DCM-08, DCM-09 and DCM-10 reviewed existing Windows evidence without changing product source or running another test. Documentation commits do not replace that tested identity. Draft PR #1 remains open; no release package is qualified.

The selected native safety group remains **21 passed / 0 failed / 0 skipped**, exit 0, no timeout, source unchanged. DCM-08 reviewed 15 same-file identity subtests. DCM-09 reviewed nine D2 top-level tests and 12 nested subtests. DCM-10 reviewed eight D6 top-level tests and two nested subtests for original-byte duplicate Undo, safe refusal, batch preflight and restart handling. All are included in the existing group, not additional tests.

Both `proposed-fixes/no-replace.patch` and `proposed-fixes/duplicate-kind-gate.patch` are already applied; do not reapply either.

**Next: DCM-11**, durable batch membership. Inspect normal, replacement and duplicate-operation journal membership across the interruption boundary, together with supporting raw native evidence. Reuse results only while relevant source and tests are unchanged. A discovered defect requires assertion RED -> smallest local fix -> GitHub -> FlowDev native GREEN. Do not silently close later steps.

## Native runner

Device: `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`.

Candidate: `C:\FlowverseReports\dcm-quicksort-candidate`, origin `https://github.com/laxmanxp/dcm-quick-sort.git`.

Set `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports`, run from the candidate's `tools\flowdev-dcm` directory, and use `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.

Use `flowdev update dcm-quicksort`, then registered `flowdev test dcm-quicksort <selector>` routes when a run is needed. Record the actual updated HEAD. Code locally, store in GitHub, validate on Lenovo. Do not use the older monorepo profile or dirty v1.0.62 checkout. Run one native selector at a time.

Latest native report: `C:\FlowverseReports\dcm-quicksort-regressions-20261109-151557-80d9f368`. Compact evidence is retained in `evidence/dcm-07-native-summary.json` and the DCM-08/09/10 review JSON files. DCM-10 inspected stdout lines 66–105 and receipt lines 202–239; it did not create a run.

The last resource observation at 13:32 UTC showed 2,271,277,056 bytes free on C: and 1,920,344,064 bytes available RAM; this is historical, not a current guarantee. The 15:15 UTC regression run completed. Use `host-diagnostics` if stalls return. Do not delete user files or close unrelated applications automatically.

## Remaining qualification

Preserve original defect IDs in `../todo.json`. Bind evidence to exact tested source; keep code, UI and user-test status separate. DCM-01–10 are closed at their recorded boundaries; the first five are setup/diagnostic work, not a product percentage.

The remaining sequence includes `baseline`, `installer-unit`, `ui-tooling`, `package`, exact app/setup and embedded-payload hashes, isolated installer lifecycle, normal-app navigation/preview/rename/move/conflict/restart-Undo workflows, and reviewed images. DCM-09 is not physical second-volume UI acceptance; DCM-10 is not normal-app or crash-atomic batch-Undo acceptance. Bind `DCM_BUILD_RECEIPT` for UI selectors and `DCM_PACKAGE_RECEIPT` for `installer-ui-cancel`. Read `../tools/README.md` and `../tests/native/README.md` before those gates.

`visual_reviewed=false` is not visible-content acceptance. No user acceptance is recorded. Preserve private configuration/backup/registry isolation; actual Explorer registration requires a disposable Windows user or VM. Update evidence and release notes before `release-prepare`; any release draft must use the exact qualified package commit. Do not merge or publish until required gates and explicit publication approval are complete.
