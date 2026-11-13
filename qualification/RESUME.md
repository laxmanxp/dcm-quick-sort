# DCM QuickSort — remaining attachment cost localized; no new speed fix

Latest: [attachment phase profile](2026-11-13-sumatra-attachment-profile.md), 13 November 2026.
Native-tested source `5294d6cf32e84038aeed93750800bd7308bad7ad`.

## Verified diagnosis

Two four-opening runs complete with exact selection/content, preserved fixture bytes and cleanup. Run A queue samples218/297/327/204ms: two exceed provisional250ms. Unchanged B201/246/167/140ms passes the same target. Keep both; overall performance_accepted=false. Shared-machine, opt-in tracing observations, not cold boot/cache-flushed or isolated benchmarks.

Full attachment handler177–339ms; full49-control layout114–246ms; host zoom-button visibility/enabled updates38–88ms. These two host phases account for85.6–88.6% of measured attachment time. Separate viewer resize11–18ms, first status5–11ms, toolbar repair6–19ms, final status0–2ms. The slowest handler had246ms layout plus50ms zoom updates; seven drop-down moves totaled189ms. These are elapsed call intervals, not a diagnosis of Windows/driver/scheduling internals or proof all calls are dispensable.

The new trace only records numeric per-control and phase checkpoints within a UI-thread-only attachment scope. All49 original MoveWindow argument lists/order, final redraw, full transition geometry, worker ownership, runtime restrictions and the startup harness remain unchanged. Ordinary launches leave tracing disabled. One new Windows parent verifies disabled/nested/zero-ID scope behavior.

Fresh Windows baseline373/0/1 (374 parent results), no timeout, source unchanged. Pending-close3, lifecycle11, resize23+12bursts all pass;24 repaint comparisons zero. All4 first-run full-window images individually inspected. Crops checked automatically; repeat/regression images not separately reviewed. Test fixtures only; installed runtime/user documents/associations unchanged.

## Next bounded correction

Retain full layout calculation and the required preview-container sizing. Target unnecessary native updates for unchanged drop-down geometry and unchanged host zoom-button state, with explicit tests for changed size/DPI, hidden/restored state and preview transitions. Do not restore the rejected whole-layout skip or trust cached dimensions across DPI/handle changes. Measure the smallest safe change with this same profile and queue test, then run resize, lifecycle, Undo and error regressions. No performance repair has been implemented in this diagnostic pass.

The prior optimization at948ff3f5/8e510e2e caused visible containment failure and remains removed. Async worker creation remains retained. Old failures, including the325ms-class delays, remain source-bound historical evidence. DCM-SUMATRA-STARTUP-01 stays open; no numbered/user acceptance is inferred.

## Workflow and evidence

Code locally -> GitHub -> FlowDev update/build/test on Lenovo, one native DCM UI suite at a time. Device d18227f2-0b54-4b11-9026-56ac40305f02. Workspace D:\FlowverseReportsRepo; candidate dcm-quicksort-candidate; profile tools\flowdev-dcm; runner D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe. Runtime source D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe. No remote source edits or unrelated-process changes.

Exact build: D:\FlowverseReportsRepo\dcm-quicksort-build-20261113-031920-176d38b8\receipt.json, matches5294d6cf only. App SHA25634c17d1ccc72259fffc4425210b4839674cb1d3e3a75d36b5bba6da9b9188b18. Rebuild after tracker/source commits; never relabel historical receipts.

Reports under D:\FlowverseReportsRepo: startup032014-6ac5acb9 and032205-3e83c8ae; baseline032229-c7441c1d; pending-close032325-7e5b69e1; lifecycle032340-62160fbf; resize032419-b0e1cc88 (all20261113). Full names and hashes: evidence/sumatra-attachment-profile-summary.json.

DCM01–18 and20–27 retain completion;19 blocked;28/29 pending. General labels/error banner, externalPPTX, remaining operations/path variants, distribution/installer and manual acceptance stay open. Sumatra optional, Windows.Data.Pdf default. PR#1 draft; no installation, associations, printing, merge or publication.
