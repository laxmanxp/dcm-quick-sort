# QuickSort — readable runtime errors verified; intermittent resize issue reopened

Latest: [readable-error retry](2026-11-12-sumatra-readable-retry.md).
Fresh native-tested source `715cd61d7da00ee0125eb006246e77711411acc2`.
No new product/test implementation in the retry: recovered the interrupted concise dependency-error fix and capture diagnostics, then freshly validated them.

## Current verified state

Two fresh error runs pass 12 records each; all ten first-run captures reviewed (six host, one password modal, three content crops). Missing runtime now reads: **SumatraPDF.exe is missing. Restore the approved private viewer, then Reload.** Underlying OS error/path is retained by Unwrap. Runtime/policy pins unchanged. Password17s/wrong/correct/cancel/recovery and source safety pass. Native damaged-file banner still truncates its long path; DCM-SUMATRA-ERROR-COPY-01 is partial, not closed.

Baseline **370 passed / 0 failed / 1 external-PPTX skip**. Lifecycle11 passes. Navigation first run fails repeated-resize repaint/background comparison; unchanged retry passes23. Do not call the whole UI stable or discard the failed raw image.

**DCM-UI-RESIZE-01 REOPENED:** `07-repeat-restored.png` in the first navigation run visibly shows incomplete/erased host controls; explicit redraw reference is clean. 32603/107276 changed samples and1746 background mismatches. Earlier07-restored is clean. Three failure/adjacent/reference images reviewed. Cause remains unknown; a subsequent unchanged pass means intermittent, not fixed. Preserve previous24ff2f24 evidence as historical.

One earlier capture obstruction is identified: lifecycle145605 at715cd61d had host PID7332 and an obstructing#32768 popup from that same PID. Why it opened is unestablished. Guard retained; no menu dismissal or relaxed capture checks in this retry. Do not attribute every previous capture failure to that one popup.

## Next exact scope

Instrument layout completion/paint ordering around the three rapid shrink/restore cycles; reproduce the raw-vs-diagnostic-redraw difference before changing the product. Distinguish unfinished asynchronous layout from lasting stale pixels, and preserve raw captures before any diagnostic redraw. Do not merely increase sleeps or close the issue after another successful retry. Then complete remaining native error-banner, startup responsiveness and readable-text scrolling checks. DCM-28/29 remain pending and Sumatra stays opt-in.

## Execution and evidence

Code locally -> GitHub -> FlowDev update/build/test on Lenovo. One native UI suite at a time. Workspace `D:\FlowverseReportsRepo`; checkout `D:\FlowverseReportsRepo\dcm-quicksort-candidate`; profile `tools\flowdev-dcm`; runner `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`. Device `d18227f2-0b54-4b11-9026-56ac40305f02`. No remote source edits. No unrelated process/window changes.

Pinned source runtime: `D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe`. Native suites copy it into private labs. Keep exact runtime/policy checks and no-print/no-write restrictions.

Last exact build receipt `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-151848-8d41b96a\receipt.json` matches715cd61d only. App hash e20c2b160f3b315d2a58b9d1bbf3effd9b105b89e03cc44b8fe92d8062b2ecab. Refresh after tracker/source changes; never rewrite receipts.

Reports under D:\FlowverseReportsRepo:
- errors reviewed: dcm-quicksort-ui-sumatra-errors-20261112-151918-97d90d78; repeat152522-13a95983.
- lifecycle: dcm-quicksort-ui-sumatra-lifecycle-20261112-152440-8d22ee27.
- navigation failure: dcm-quicksort-ui-sumatra-20261112-152144-1254fe66; unchanged pass152939-74d333d0.
- baseline: dcm-quicksort-baseline-20261112-152048-fc05ebbe.

Separate receipt/image hashes: evidence/sumatra-readable-retry-summary.json. Repeat/lifecycle/navigation-repeat images not separately reviewed. All numbered completion and human acceptance states unchanged: DCM01–18 and20–27 complete,19 blocked,28/29 pending. Keep general labels, startup, external DCM_REAL_PPTX, operation/path/release variants, runtime distribution/installer and human acceptance open. PR#1 draft; no default switch, installation, merge or publication.
