# DCM v1.0.71 — resume after DCM-08

Latest checkpoint: [DCM-08 same-file protection](2026-11-09-dcm08.md). Earlier DCM-06 and DCM-07 checkpoints retain their original test identities.

## Current state

Product/tested commit: `26eeaed3b9041a31b17c197ef2a095f53ee2a9e6` on `release/v1.0.71`. DCM-08 reviewed existing evidence without changing product source or running another test. Later documentation commits do not replace the tested identity. Draft PR #1 remains open; no release package is qualified.

The selected native safety group remains **21 passed / 0 failed / 0 skipped**, exit 0, no timeout, source unchanged. DCM-08 explicitly reviewed the 15 same-file identity subtests inside that group: exact path, hard link and case alias across five conflict operations. All passed and preserved original fixture bytes. These are not additional top-level tests. Successful case-only Rename and visible conflict/journal behavior remain separate UI gates.

Both `proposed-fixes/no-replace.patch` and `proposed-fixes/duplicate-kind-gate.patch` are applied: do not reapply either.

**Next: DCM-09**, guarded-copy/source-retirement qualification. Inspect its supporting tests and raw native evidence for content/identity changes, open writers, stale verification and safe refusal. Reuse evidence only while the relevant product and test source is unchanged. If a coverage gap needs a fix, follow observed assertion RED -> smallest local fix -> GitHub -> FlowDev native GREEN. Continue DCM-10 onward without silently closing further steps.

## Native runner

Device: `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`.

Candidate: `C:\FlowverseReports\dcm-quicksort-candidate`, origin `https://github.com/laxmanxp/dcm-quick-sort.git`.

Set `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports`, run from the candidate's `tools\flowdev-dcm` directory, and use `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.

Use `flowdev update dcm-quicksort`, then registered `flowdev test dcm-quicksort <selector>` routes when a run is needed. Record the actual updated HEAD. Code locally, store in GitHub, validate on Lenovo. Do not use the older monorepo profile or dirty v1.0.62 checkout. Run one native selector at a time.

Latest native report: `C:\FlowverseReports\dcm-quicksort-regressions-20261109-151557-80d9f368`. Compact evidence: `evidence/dcm-07-native-summary.json` and `evidence/dcm-08-same-file-review.json`. DCM-08 inspected raw stdout lines 1–66 and the receipt's identity/result section; it did not create another run.

The last resource observation at 13:32 UTC showed 2,271,277,056 bytes free on C: and 1,920,344,064 bytes available RAM; this is historical, not a current guarantee. The 15:15 UTC regression run completed. Use `host-diagnostics` if stalls return. Do not delete user files or close unrelated applications automatically.

## Remaining qualification

Preserve original defect IDs in `../todo.json`. Bind evidence to exact tested source; keep code, UI and user-test status separate.

The remaining sequence includes `baseline`, `installer-unit`, `ui-tooling`, `package`, exact app/setup and embedded-payload hashes, isolated installer lifecycle, normal-app navigation/preview/rename/move/conflict/restart-Undo workflows, and reviewed images. Bind `DCM_BUILD_RECEIPT` for UI selectors and `DCM_PACKAGE_RECEIPT` for `installer-ui-cancel`. Read `../tools/README.md` and `../tests/native/README.md` before those gates.

`visual_reviewed=false` is not visible-content acceptance. No user acceptance is recorded. Preserve private configuration/backup/registry isolation; actual Explorer registration requires a disposable Windows user or VM. Update evidence and release notes before `release-prepare`; any release draft must use the exact qualified package commit. Do not merge or publish until required gates and explicit publication approval are complete.
