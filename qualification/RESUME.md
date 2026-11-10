# DCM v1.0.71 — resume after DCM-17

Latest checkpoint: [DCM-17 actual launch and empty-state UI](2026-11-10-dcm17.md). Preserve all previous failures, checkpoints and exact tested identities.

## Verified state

DCM-01–17 are complete at their stated boundaries; only DCM-17 was newly closed. Five completed steps are setup/diagnostic work, not a product percentage.

Latest UI-tested commit: `443153a35c09a85d2438b8aca0aba6a983c41539`. The registered `ui-launch` check passed 12 recorded acceptance cases (nine native state/filesystem checks and three FlowUIVision flows), two clean app closes and unchanged-source checks. All four actual screenshots were separately inspected. The raw automation receipt remains `visual_reviewed=false`; see the separate review in `evidence/dcm-17-native-summary.json`.

New tooling is `tools/test-ui-launch.ps1` plus the registered selector. Manifest-array and stale-inspection harness failures were corrected. Product implementation and existing assertions were unchanged. Do not confuse these harness fixes with application feature changes.

Open visual findings: `DCM-UI-EMPTY-COPY-01` (filtered no-match says Folder complete; DCM-21/46) and `DCM-UI-LABEL-01` (clipped labels/crowded Processed heading at captured scaling; DCM-46). They require resolution or explicit disposition before final release; the functional DCM-17 pass is not global visual approval.

## Next exact action — DCM-18

Verify opening a source folder through the supported UI, including spaces in its path; compare source textbox, list and displayed content with disposable fixtures. Inspect existing Browse/folder-picker coverage before extending tools. Do not certify the picker from a startup folder argument alone. New tooling must be authored locally, committed, then used through FlowDev; keep actions bound to the owned candidate and expected dialogs.

Update the candidate, run tooling validation for any changes, then `build`. The DCM-17 receipt matches only its exact tested commit; this later documentation revision requires a fresh matching receipt before further UI interaction. Never edit receipt identities or bypass source checks. Record the final binding in PR metadata to avoid a documentation/build loop. Do not close later steps silently.

## Native workflow and evidence

Device: `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`.
Candidate: `C:\FlowverseReports\dcm-quicksort-candidate`, branch `release/v1.0.71`.
Set `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports`, run from the candidate's `tools\flowdev-dcm`, using `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.
Run update before one registered selector at a time. Code locally, GitHub storage, Lenovo validation. No remote editing, host installation or unrelated cleanup.

UI evidence: `C:\FlowverseReports\dcm-quicksort-ui-launch-20261110-174648-a112f78b`.
Matching historic build receipt: `C:\FlowverseReports\dcm-quicksort-build-20261110-174601-4def7398\receipt.json`.
Final script blob: `63822da5daee4af18cb3486a7af8f331b0b15841`.
The four PNGs and raw control/FVS/receipt records remain in the exact UI report directory. Review decisions and image hashes are committed separately; user acceptance is not inferred.

Full app baseline remains 351 passed / 0 failed / 1 skipped at `13f52cb0`. The external `TestRealGeneratedPPTXIntegration` needs its identified two-slide fixture through `DCM_REAL_PPTX` before DCM-32 / DCM-PREVIEW-01 / DCM-47. Installer units remain 26/26 at `99f07c15`; in-process installer lifecycle remains separately pinned to `a49e6595`. These suites were not rerun for DCM-17. Keep PR #1 draft; final artifacts, packaged installer lifecycle in a disposable account/VM, remaining UI/Explorer checks, visual findings and actual user acceptance remain open. No merge or publication.
