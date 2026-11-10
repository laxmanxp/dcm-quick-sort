# DCM v1.0.71 — resume after DCM-14

Latest checkpoint: [DCM-14 full native baseline](2026-11-10-dcm14.md). Preserve all earlier source identities and failed evidence.

## Current state

DCM-01–14 are complete at their defined boundaries, including five setup/diagnostic steps. Only DCM-14 was closed in this continuation.

Latest full-app tested source: `13f52cb0ed5e494bd838062c6b7e3fcca424f068`. `baseline`: **351 passed, 0 failed, 1 skipped; 352 top-level results**, exit 0, no timeout, source unchanged. Seven initial failures were test-fixture defects: unset Windows creation timestamps and CRLF/LF source comparisons. Both were reproduced in native diagnostic RED before test-only corrections. All original ordering and wiring expectations remain intact; application/installer implementation was unchanged.

**Do not call this zero-skip acceptance.** `TestRealGeneratedPPTXIntegration` did not execute because `DCM_REAL_PPTX` was not set. It remains open under DCM-32 / DCM-PREVIEW-01 / DCM-47. Supply an identified external two-slide fixture through committed FlowDev tooling and run its unchanged assertions before PPTX/final-package qualification. Visible screenshots are separate.

## Next exact action

**DCM-15:** verify relevant installer source is unchanged before reusing its 26-test result, and run registered `ui-tooling` through FlowDev after update. Do not infer UI acceptance from source assertions or harness compilation. Then DCM-16 creates the exact source-bound UI build. Do not silently close later steps.

## Native workflow

Device `Laxman_Lenovo`: `d18227f2-0b54-4b11-9026-56ac40305f02`.
Checkout: `C:\FlowverseReports\dcm-quicksort-candidate`, branch `release/v1.0.71`.
Set `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports`; run from the checkout's `tools\flowdev-dcm`, using `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.
Use `flowdev update dcm-quicksort` then one registered `flowdev test dcm-quicksort <selector>` at a time. Code locally; store in GitHub; validate on Lenovo. No remote source edits or host setup replacement.

Final app report: `C:\FlowverseReports\dcm-quicksort-baseline-20261110-030347-e80c1e53`.
Initial failed full-app report: `C:\FlowverseReports\dcm-quicksort-baseline-20261110-025222-2c14b054`.
Diagnostic RED report: `C:\FlowverseReports\dcm-quicksort-regressions-20261110-025903-c4ae5422`.
Compact evidence: `evidence/dcm-14-native-summary.json`.

## Prior installer result and release boundary

DCM-13 remains closed at the in-process engine boundary; `DCM-INSTALL-CLEANUP-01` is fixed. Installer units: 26 passed at `70e8f240a0228c8548562bb627db4213cbaf1d17`, report `C:\FlowverseReports\dcm-quicksort-installer-unit-20261109-180724-83ef3b94`. Isolated lifecycle plus receipt reader: 1 passed at `a49e65951b5be9d26dc7d74a4916ae0b1b54664d`, report `C:\FlowverseReports\dcm-quicksort-installer-integration-20261109-181814-920da14f`. Keep those identities, not the latest documentation revision. The test uses its own executable for uninstall self-copy; packaged executable E2E remains open.

Both old move/recovery proposals and installer cleanup corrections are already applied. Read `../tools/README.md` and `../tests/native/README.md` before UI/package gates. Bind exact `DCM_BUILD_RECEIPT` / `DCM_PACKAGE_RECEIPT`; review images and user feedback separately. Actual packaged lifecycle and Explorer integration need a disposable Windows account/VM. Preserve raw host registry snapshots on Lenovo. No merge or publication before all remaining gates and explicit approval.
