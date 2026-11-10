# DCM v1.0.71 — resume after DCM-18

Latest checkpoint: [DCM-18 Browse workflow](2026-11-10-dcm18.md). Preserve earlier reports and source identities.

DCM-01–18 are complete at their defined boundaries; only DCM-18 was newly closed. Five are setup/diagnostic steps, not a percentage. The actual Browse dialog was exercised, not replaced with a startup path. Nine cases passed at `a3e4c185dd00bf75222694db7a324b86de50bc86`, all six screenshots were separately reviewed, all three fixture names/hashes stayed unchanged, and the owned app closed normally (exit 0, children drained, source unchanged).

UI report: `C:\FlowverseReports\dcm-quicksort-ui-folder-20261110-181633-dfd65522`.
Matching build receipt: `C:\FlowverseReports\dcm-quicksort-build-20261110-181432-f131c8f5\receipt.json`.
App SHA-256: `a10e368eef249dbb38b340c52cfc8e2c2cb583758d14f07d09778c5ab9ee98c5`.
Evidence: `evidence/dcm-18-native-summary.json`. Raw visual flags remain false; separate image review is recorded. Preserve the failed capture run `20261110-181547-2391fd35`: the exact obstruction cause was not established, and the unchanged retry passed without relaxing the capture guard.

## Next exact action — DCM-19

Verify Windows Explorer folder/background integration in a disposable Windows account or VM. Inspect isolation availability and actual menu labels and quoted-path registration. A startup argument or reading registry values alone does not certify the Explorer click workflow. Do not overwrite live host registrations or install the candidate on the host as a substitute. Missing isolation leaves the gate pending; do not manufacture completion.

Use the existing FlowDev profile: workspace `C:\FlowverseReports`, candidate `C:\FlowverseReports\dcm-quicksort-candidate`, branch `release/v1.0.71`; run from its `tools\flowdev-dcm` with `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`. Device `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`. Code locally, store in GitHub, update then test on Lenovo; no remote source edits. Only one native UI run at a time.

The DCM-18 build receipt matches only `a3e4c185dd00bf75222694db7a324b86de50bc86`. Any later code/documentation revision requires a fresh source-matching build before UI use. Do not edit receipts or bypass checks. Newly registered selectors: `ui-folder-probe` (discovery/cancel only) and `ui-folder` (full DCM-18 workflow).

Earlier full app baseline: 351 passed, 0 failed, 1 skipped at `13f52cb0`; external `DCM_REAL_PPTX` fixture still required under DCM-32/47. Installer units remain 26/26 at `99f07c15`; in-process engine at `a49e6595` is not packaged-executable E2E. These suites were not rerun here.

DCM-UI-EMPTY-COPY-01 and DCM-UI-LABEL-01 remain open. Remaining UI workflows, Explorer integration, external PPTX test, final package/payload, actual packaged setup/uninstaller lifecycle and user acceptance remain pending. Keep PR #1 draft; no merge/publication without the remaining gates and explicit approval.
