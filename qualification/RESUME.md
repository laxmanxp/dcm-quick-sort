# DCM QuickSort — live-control optimization verified; next compact-width label cleanup

Latest: [control update optimization](2026-11-13-control-update-optimization.md).
Native-tested source `cc6a1a37308ba57be37c33c81e743b668bb89a1c`, 13 November 2026.

## Retain this implementation

Eight dropdown call sites skip only exact LIVE geometry matches with same-thread/class/closed-list/equivalent-DPI guards. No geometry cache. Unknown/mixed/changed state uses original MoveWindow. Full layout calculation, preview-pane sizing and final repaint remain. Zoom updates inspect own WS_VISIBLE/WS_DISABLED and retain actual state changes. Do not reintroduce the failed whole-layout shortcut.

Two four-opening runs: UI queued responses16.8–156.3ms, all below the unchanged provisional250ms budget. Full layout9.6–52.8ms (previous113.6–246.0ms), attachment49.8–166.0ms (previous177.0–339.4ms). Every attachment skipped8 redundant dropdown moves and5 enable calls; no required hide was skipped. First PDF ready1.58–2.26s, not uniformly faster. Shared-machine measurements, not cold-boot/isolated benchmark or universal guarantee.

## Verified matrix and boundaries

Baseline377/0/1 (378 parents); four new actual-Windows-control parents. DPI awareness contexts/mixed caller context tested, not physical monitor/scaling-setting changes. Startup2x4, resize23+12bursts (all24 sampled differences zero), pending-close3,lifecycle11,Undo9,errors12,scroll14,text17,image23,sort9 and filters74 action/readback records pass. No summing repeated/parent/UI counts. External PPTX remains the sole skip, not a pass.

Nine full images reviewed:4 startup-A and5 resize/restore/text frames. Other images not separately reviewed. Known narrow Size/Sort overlap and Processed-heading clipping remain open. Raw flags unchanged. Exact evidence in `evidence/control-update-optimization-summary.json`.

## Next bounded action

Correct the existing compact-width filter row and Processed-header layout (DCM-UI-LABEL-01) with actual readable labels, dropdown usability, full preview containment and resize screenshots. Then audit remaining DCM-28/29 acceptance gaps. Keep startup/resize and relevant selection/filter regressions; do not add another profiling-only step unless a new failure warrants it. Broader physical-DPI/manual/runtime-distribution gates remain separate, not silently certified.

## Workflow / identity

Local code -> GitHub -> FlowDev update/build/test on Lenovo; no remote source edits. Device `d18227f2-0b54-4b11-9026-56ac40305f02`. Workspace `D:\FlowverseReportsRepo`; candidate `dcm-quicksort-candidate`; profile `tools\flowdev-dcm`; runner `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`. One DCM UI suite at a time; do not touch unrelated LibreOffice/other sessions.

Build receipt `D:\FlowverseReportsRepo\dcm-quicksort-build-20261113-044538-b6b19d38\receipt.json` matches cc6a1a37 only. App SHA256 `13fdc03eca1d049c8f88e6bb4ed2c402015bbb6a4ad30aa0aa4ef30c1356d5cd`.
Runtime source `D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe`; harness copies it privately. Exact hash/policy checks unchanged. Rebuild after metadata/source changes, never relabel old receipts.

Preserve native RED95ad5af2 (new redundant-control tests) and two preapplication workflow hash failures; report explains them. Earlier async/resize failures remain historical, not erased.

No numbered closure: DCM01–18 and20–27 complete,19 blocked,28/29 pending. Keep optional Sumatra/default Windows.Data.Pdf, externalPPTX, error-banner, operation/path/release variants, runtime packaging/licensing and human acceptance. PR#1 draft; no installation/association/printing/merge/publication.
