# DCM v1.0.71 — resume after DCM-21

Latest checkpoint: [DCM-21 filters complete](2026-11-11-dcm21.md). Preserve all historical source identities and raw receipts.

Completed: **DCM-01–18, DCM-20, DCM-21**. DCM-19 is still blocked. Only DCM-21 was newly closed; five completed steps are setup/diagnostic work, not a product percentage. `DCM-UI-EMPTY-COPY-01` is fixed at this UI boundary; `DCM-UI-LABEL-01` remains open.

## Latest verified results

Tested source: `a363333ede3243ead62ce11f5a22c8665205ffe5`. Fresh baseline: **357 passed, 0 failed, 1 retained external-PPTX skip** across 358 results. Fresh ui-filters: PASS with all fixtures/source unchanged and normal exit 0; six images inspected. Fresh ui-launch: 12 cases passed, two normal exit-0 closes, children drained/source unchanged; four images inspected. Raw visual flags were not edited; separate review is in `evidence/dcm-21-native-summary.json`.

The previous Clear Filters interruption was reproduced separately as blank-destination validation after Move with no selected file. The body says `Choose an existing logical location inside the configured Vault.` then `logical address is empty`. `moveCurrent` validates destination before selection. The normal matrix seeds `Accepted` and retains dialog/file-safety guards. Diagnostic mode `DCM21_PROBE_EMPTY_DESTINATION=1` deliberately exits RED after capturing evidence and must be **unset during acceptance**. Do not count that run as passed or repeat it unnecessarily.

The wording correction was already committed before this retry. Only test diagnostics changed in this retry. Three prior DCM-21 Go tests explain the increase from DCM-20's 354 passing tests to 357; no new Go tests were added in this retry.

## Next exact runnable step — DCM-22

Inspect the four existing sorting modes and their native tests. Reuse the owned FlowDev/native adapter/FlowUIVision harness with controlled creation timestamps and explicit expected ordering. Verify that file selection, draft and visible preview remain associated with the same intended file across sort changes. Do not confuse modification timestamps (Age filter) with Windows creation timestamps (Date Created sort). Do not close DCM-23 or any other step incidentally.

Code locally, store on GitHub, then use FlowDev update/test on Lenovo; no remote source edits. Workspace `C:\FlowverseReports`, candidate `C:\FlowverseReports\dcm-quicksort-candidate`, profile directory `tools\flowdev-dcm`, runner `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`, device `d18227f2-0b54-4b11-9026-56ac40305f02`. One native UI run at a time.

Build receipt used: `C:\FlowverseReports\dcm-quicksort-build-20261111-025820-58006f2e\receipt.json`. It matches only `a363333ede3243ead62ce11f5a22c8665205ffe5`. After this documentation commit or any later revision, create a fresh matching build before UI use. App SHA-256 `55fd13c689b55c09d6e464cd75735fbff4c2becf3e9c49a0c093e61e62f28f56`.

Reports: baseline `20261111-025742-6ba62701`; ui-filters `20261111-030030-c87ccf31`; ui-launch `20261111-030053-ac91402e`; diagnostic-only RED `20261111-025935-9589eb6f`. Full paths and hashes are in the checkpoint/evidence. Preserve original failed filter run `20261111-021101-c58e206d`; its old dialog body was not captured, so do not claim retroactive observation.

DCM-19/`DCM-EXPLORER-ISOLATION-01`, clipping, external `DCM_REAL_PPTX`, remaining UI and preview workflows, final package, actual setup/uninstaller lifecycle and user acceptance remain pending. Installer evidence retains its earlier source identity; the in-process engine is not packaged E2E. No host registration/install as an isolation substitute. Keep PR #1 draft; no merge/publication without remaining gates and explicit approval.
