# QuickSort — async startup implemented; speculative transition optimization removed

13 November 2026. The current application source restores the fully exercised async implementation at `4c1c32d668955a2073ac496d0f3d31e5fe1cdf43`. Only the redundant-layout optimization was removed, not the worker/cancellation implementation. Additional test-only visibility/geometry diagnostics are retained. No native run is yet claimed against this rollback/checkpoint revision; rebuild before further qualification.

## Implemented and exercised

Validation and suspended process creation now run in a single preparation worker. UI-thread adoption resumes only the still-current request. Cancellation disposes late results; pending file operations refuse until cleanup; normal close keeps the message loop alive until the worker drains. Creation-time job assignment and exact runtime/policy checks remain in place. The new portable ownership tests, Windows pending-operation/close tests, and existing release-refusal tests pass.

At0bdfce87 the first-open queued UI delay was172ms versus the earlier1497–1758ms. The UI start handler returned in0.6–1.9ms while process creation still took1.52s on the first open. However two repeated opens measured275/321ms queue delay, so the provisional250ms responsiveness target remains open. Do not claim a universal performance pass or cold-boot benchmark.

At4c1c32d the full native baseline and startup(two runs), close-while-loading, lifecycle, Undo, errors, scrolling and resize-timing commands completed successfully. Startup command PASS means measurement/safety integrity; inspect its separate responsiveness_accepted field. No completed checklist step changes.

## Failed optimization and exact rollback

948ff3f5 skipped a full host layout at unchanged sidebar widths. Its startup verification first failed a minimized-window capture and then containment. Added diagnostics at8e510e2e reproduced actual containment failure with a visible, non-minimized host: panel(333,259,1238,792), viewer(334,260,1463,923). The scaling discrepancy is an observation, not an established root cause. Report: `D:\FlowverseReportsRepo\dcm-quicksort-ui-sumatra-startup-20261113-021713-2ce5391a`.

Restore `app-src/sumatra_preview_windows.go` to exact blob3b7041b3115ecbebd0c0f247f8a6e52d05aad93c from4c1c32d; remove unused sumatra_transition_windows.go. The original full transition layout remains. Local race tests and Windows cross-compilation pass after rollback. Keep failed receipts and diagnostic changes. Do not reintroduce the optimization merely to obtain faster timing.

## Next bounded work

First rebuild and revalidate the restored application revision. Then profile the remaining attachment/layout response delay without changing containment, capture or timing assertions. The asynchronous process-creation correction and the remaining250ms target are separate outcomes. No further optimization was accepted in this checkpoint.

## Execution

Code locally -> GitHub -> FlowDev update/build/test on Lenovo. Device d18227f2-0b54-4b11-9026-56ac40305f02. Workspace D:\FlowverseReportsRepo; candidate dcm-quicksort-candidate; profile tools\flowdev-dcm; runner D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe. Runtime source D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe. One DCM UI suite at a time; unrelated LibreOffice interactive session was observed and not modified. Do not claim it caused the failures.

Keep Sumatra optional, Windows.Data.Pdf default, runtime/no-print/no-write pins unchanged. DCM01–18 and20–27 remain complete;19 blocked;28/29 pending. General layout/error-banner, external PPTX, packaging/distribution and actual user acceptance remain open. PR#1 stays draft; no merge, publication, installation or association changes.
