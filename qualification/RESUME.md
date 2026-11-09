# DCM v1.0.71 — resume DCM-13; installer cleanup blocker

Latest checkpoint: [DCM-13 partial results](2026-11-09-dcm13.md). DCM-01–12 remain complete; DCM-13 is deliberately still pending. Preserve earlier checkpoint identities.

## Actual current state

Latest native-tested repository commit: `2a302e38c92b271e97d484e7eb77010559dfba03`. Only the isolated installer test harness changed; application/installer implementation is unchanged. The full installer unit suite passed 24/24 at `1c5bbcae5eaf21f147958dab52e5fdc949df4dd0`. The initial integration attempt failed before any installer call because of a 64-bit HKCU handle representation. That harness boundary is corrected and the new run gets through fresh install, upgrade and restore.

**The latest installer engine still FAILS:** `owned uninstall cleanup exited 1`. Preferences/configuration hashes match after upgrade and restore, but final uninstall assertions are not reached. Registry override reset, private-key deletion and unchanged live DCM keys are proven. Do not label this partial engine result green. Do not erase or weaken its assertion.

Open item: `DCM-INSTALL-CLEANUP-01` in `../todo.json`; linked to DCM-13 and packaging/lifecycle gates DCM-47/48. The 50-step count and completed steps are unchanged. Full app-suite and UI acceptance remain separate.

## Next exact action

Diagnose the cleanup child exit code using the existing owned fixture and production `uninstall` in `installer-src/main_windows.go` (lines 376–412). The CMD script delays, deletes the uninstall copy, removes the owned installation folder and deletes itself. The post-run listing found installation folder/script absent, but that is not proof of a clean exit. Capture/reproduce per-command outcomes in a locally authored, committed FlowDev check before deciding whether product or test needs correction. No direct remote coding. Do not solve this by unconditional `exit 0`, ignoring return codes, or changing the test to accept failure.

Latest engine report: `C:\FlowverseReports\dcm-quicksort-installer-integration-20261109-173550-9f982691`.
Initial failed isolation report: `C:\FlowverseReports\dcm-quicksort-installer-integration-20261109-172606-e2b5f2f5`.
Successful installer-unit report: `C:\FlowverseReports\dcm-quicksort-installer-unit-20261109-172517-35406408`.
Compact evidence: `evidence/dcm-13-native-summary.json`. Full raw logs and registry snapshots remain on Lenovo; do not publish the host registry contents.

## Registered native workflow

Device: `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`.
Candidate: `C:\FlowverseReports\dcm-quicksort-candidate`, expected origin `https://github.com/laxmanxp/dcm-quick-sort.git` and branch `release/v1.0.71`.
Set `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports`, run from the candidate's `tools\flowdev-dcm` directory, using `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.

Run `flowdev update dcm-quicksort` before one registered test selector at a time. Relevant selectors: `installer-unit` and `installer-integration`. Code locally, store in GitHub, validate on Lenovo. No edits in the candidate checkout; no host setup or unrelated cleanup. Both earlier product proposals are already applied; do not reapply them.

## Remaining gates

After DCM-13 is closed, continue DCM-14 (`baseline`) and DCM-15 (`installer-unit`, `ui-tooling`) with exact-source evidence. Current installer units can be reused only while their source remains unchanged. Build/package receipts are not UI acceptance. Read `../tools/README.md` and `../tests/native/README.md` for isolation, artifact binding and cleanup rules. Use `DCM_BUILD_RECEIPT` for UI selectors and `DCM_PACKAGE_RECEIPT` for setup Cancel. Keep screenshot review separate from automated checks and user acceptance separate from both.

An in-process engine test is not packaged setup/uninstaller E2E. Actual integration belongs in a disposable Windows account/VM, not a host install. Full suites, visual workflows, exact final payload checks, actual installer lifecycle and user acceptance remain required. Keep PR #1 draft; no merge or publication without final gates and explicit approval.
