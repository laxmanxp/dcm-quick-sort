# QuickSort — first-PDF startup UI blocking measured; correction pending

Latest: [startup measurement](2026-11-13-sumatra-startup.md), 13 November 2026.
Native-tested source `497dbb29c66092db690c3ba501501a09f9acb64e` adds opt-in diagnostics only. No performance correction has been made.

## Verified finding

Two diagnostic runs complete four PDF opens per fresh host. First ready status: 2389/1874 ms. Longest first-open posted-queue acknowledgement: 1758/1497 ms, exceeding the provisional 250-ms diagnostic budget. `responsiveness_accepted=false` in both receipts; command PASS means measurement and safety integrity, not responsive startup.

CreateProcessW call interval accounts for 1695/1433 ms on first open, versus 14–19 ms on repeated opens. Runtime/policy validation takes 38–49 ms. This identifies the blocking UI-thread call interval, not an antivirus/cache/storage explanation. Do not explain every historical timeout from this result. The ready-status observation is not an exact first-pixel timestamp.

Fresh-process vs same-host repeats, not true cold boot/cache: private runtime is pre-read for copying/hash checks; no cache flushing or reboot. An unrelated unit session was observed before the run. Shared-machine observations, not an isolated benchmark or published service level.

Four first-run full-window screenshots were reviewed; four content crops were checked automatically, not separately inspected. Repeat/regression images not reviewed. All eight PDF opens show correct target/content, source bytes remain intact, text recovery releases the PDF, and both hosts close normally without retained private viewers/profiles. Fresh baseline370/0/1, scroll14 and resize23+12bursts pass; all24 resize raw/natural comparisons zero. External PPTX skip remains not executed.

## Next exact action

Write focused tests for nonblocking startup, then move blocking validation/process creation off the UI thread. Preserve pinned runtime/exact no-print/no-write policy and suspended process assigned to owned kill-on-close job before resume. Keep window updates on UI thread. Use selection/generation tokens, dispose of late results, and coordinate pending startup with cancellation/close so no stale document or untracked suspended process survives. Do not implement fire-and-forget process creation or bypass checksum checks. Reuse the same queue-latency probe before/after; then run loading cancellation, active-PDF shutdown, Undo and release-refusal regressions. A responsive UI and faster completed PDF are different outcomes; measure both.

DCM-SUMATRA-STARTUP-01 remains open. Do not repeat existing scrolling as a substitute for the blocking-startup correction. The diagnostic is enabled only by DCM_STARTUP_TRACE=1, records bounded numeric events in memory and flushes to the test-private TEMP at normal exit.

## Execution and identities

Code locally -> GitHub -> FlowDev update/build/test on Lenovo, one native UI suite at a time. Device `d18227f2-0b54-4b11-9026-56ac40305f02`. Workspace `D:\FlowverseReportsRepo`; candidate `D:\FlowverseReportsRepo\dcm-quicksort-candidate`; profile `tools\flowdev-dcm`; runner `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`. No remote source edits or unrelated process changes.

Runtime source `D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe`; the harness copies it privately. New selector `ui-sumatra-startup` requires DCM_BUILD_RECEIPT and DCM_SUMATRA_SOURCE_EXE like the prior suites.

Last build receipt `D:\FlowverseReportsRepo\dcm-quicksort-build-20261113-010220-8d36bdb9\receipt.json` matches497dbb29 only. App SHA256 `255e0d70ba3e052dc765b4a9229c48be87cf1ddcaab6785fd4b8ce830ccb696b`. Refresh build after metadata/source commits; never rewrite historical receipts.

Reports under D:\FlowverseReportsRepo: startup010300-2885f858 and010417-3013a7a2 (prefix dcm-quicksort-ui-sumatra-startup-20261113-); baseline010443-73f34848; scroll010542-3f4b5091; resize010815-aaa93ba1. Full paths and hashes in `evidence/sumatra-startup-summary.json`.

No numbered closure: DCM01–18 and20–27 retain completion;19 blocked;28/29 pending. Keep Sumatra opt-in/Windows.Data.Pdf default, general labels/error banner, externalPPTX, operation/path/release variants, runtime distribution/installer and actual user acceptance open. PR#1 remains draft; no installation/default association/printing/merge/publication. All prior failures and source-specific evidence retained.
