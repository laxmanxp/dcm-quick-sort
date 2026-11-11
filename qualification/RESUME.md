# DCM v1.0.71 — resume after DCM-22

Latest checkpoint: [DCM-22 sorting complete](2026-11-11-dcm22.md). Preserve all historical evidence and raw receipts.

Completed: **DCM-01–18 and DCM-20–22**. DCM-19 remains blocked. Only DCM-22 newly closed; 21 checklist steps include five setup/diagnostic steps, not a completion percentage.

## Latest verified evidence

Sorting UI at `238404a3f49e87ad997de3ef90b0df9294ec2eb1`: nine cases, nine separately reviewed screenshots, six disposable files unchanged including creation/modification timestamps, empty destination/journal, normal exit 0, children drained and source unchanged. All four modes preserve single-file selection, draft, counter, heading and actual preview text for two selected files. Same-day and same-timestamp ties exercised; creation dates deliberately differ from modification dates.

Full baseline at `3803c1a1e243d8cc71e04cdc806cedee22e1408b`: **357 passed, 0 failed, 1 retained external-PPTX skip**. Only the test-harness manifest assignment changed afterward; no application, installer or Go-test changes. First harness attempt failed before launch; retain `20261111-035239-fe49533e`. Do not treat it as a product defect.

Registered selector: `ui-sort`. Use `flowdev update dcm-quicksort`, `flowdev test dcm-quicksort build`, then `flowdev test dcm-quicksort ui-sort` with the current exact `DCM_BUILD_RECEIPT`. Last build receipt `C:\FlowverseReports\dcm-quicksort-build-20261111-035455-b98aca4f\receipt.json` matches `238404a3f49e87ad997de3ef90b0df9294ec2eb1` only; rebuild after metadata/source commits.

Sorting report `C:\FlowverseReports\dcm-quicksort-ui-sort-20261111-035630-12477290`; baseline `C:\FlowverseReports\dcm-quicksort-baseline-20261111-035026-33ce584f`. App SHA-256 `55fd13c689b55c09d6e464cd75735fbff4c2becf3e9c49a0c093e61e62f28f56`. Summary: `evidence/dcm-22-native-summary.json`.

## Next exact step — DCM-23

Inspect and test the existing file-navigation controls and keyboard behavior. Verify adjacent visible file, first/last boundaries, empty/no-match state, filename draft and preview consistency using disposable fixtures. Do not assume a Previous button exists or implement unrelated UI simply because the checklist uses that word; verify the actual supported previous-navigation route. Keep multi-selection DCM-24 separate.

Code locally -> GitHub -> FlowDev update/test on Lenovo. Workspace `C:\FlowverseReports`, candidate `C:\FlowverseReports\dcm-quicksort-candidate`, profile `tools\flowdev-dcm`, runner `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`, device `d18227f2-0b54-4b11-9026-56ac40305f02`. One native UI run at a time; no remote source editing.

DCM-UI-LABEL-01 includes the clipped combined-sort value; still open. DCM-UI-EMPTY-COPY-01 retains its earlier fixed boundary. Diagnostic `DCM21_PROBE_EMPTY_DESTINATION` must remain unset during acceptance. DCM-19/isolation, external DCM_REAL_PPTX before DCM-32/47, remaining workflows/previews, final package/setup lifecycle and user acceptance remain pending. No host installation/registry substitution. Keep PR #1 draft; no merge/publication.
