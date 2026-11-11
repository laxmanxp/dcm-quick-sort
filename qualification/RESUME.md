# DCM v1.0.71 — DCM-24 test tooling committed; Lenovo execution unresolved

Latest work: [DCM-24 progress](2026-11-11-dcm24.md). Last completed feature: [DCM-23](2026-11-11-dcm23.md). Preserve all prior receipts and source identities.

Completed: **DCM-01–18 and DCM-20–23**. DCM-19 remains blocked. **DCM-24 remains pending; no new numbered completion.** Existing todo/checklist completion states were intentionally left unchanged.

## Current work

At `dec6385ad004c74f3cb300d37f263ca0e2f8c03a`, locally authored `tests/native/dcm-selection-input.cs` and `tools/test-ui-selection.ps1` are committed and `ui-selection` registered. No app/installer implementation changed. New harness compilation, native execution and screenshots are NOT verified. It uses SendInput Ctrl/Shift clicks with owned-window, hit-test and held-key guards, not direct selection assignment. It exercises six disposable files, exact selection sets, Cancel and a representative confirmed two-file move, without claiming full Batch Move/Undo acceptance.

Source-review hypothesis: caret and selected item can diverge after Ctrl deselection; current single-file Move uses currentFile. Obtain actual native evidence before calling this a defect or changing product code. The test includes removal to one/zero selected files.

## Resolve the uncertain execution first

Initial update/host check at `5bb5e8ac` passed (PID 9752, exit 0). The later command for update -> ui-tooling -> build at the new tooling revision returned a device timeout with no PID, so its execution/completion is unknown. It was not retried. A read-only list_sessions call `c388178d-9ac5-4205-8956-68420a48c856` also failed with device timeout. Device listing had an eight-minute-stale Online heartbeat. **ui-selection was never submitted.**

On reconnect, inspect running sessions and exact-revision receipts before retrying the interrupted sequence. Do not overlap runs or infer success from the Online label. Then use FlowDev update, ui-tooling and build, bind the matching DCM_BUILD_RECEIPT, and run ui-selection. Only close DCM-24 after passing native assertions, inspected screenshots and independently checked fixture/operation sets. No background monitoring is scheduled.

Code locally -> GitHub -> FlowDev update/test on Lenovo. Candidate `C:\FlowverseReports\dcm-quicksort-candidate`; profile `tools\flowdev-dcm`; runner `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`; workspace `C:\FlowverseReports`; device `d18227f2-0b54-4b11-9026-56ac40305f02`. One native UI run at a time; no remote source editing.

## Preserved results and gates

Last tested product fix remains `996f67208c574a0821d1dda598ffb7e0f9c775b3`: full baseline 357/0/1, navigation 22 cases/eight reviewed images, sorting and launch command PASS. These were not rerun here. Product SHA-256 `016da629bd77f57988eac7d2f162384d4c77583e052f4036e3e1581fe76e0fb0`; its old build receipt cannot certify later source revisions.

Keep DCM-19/disposable Explorer isolation, DCM-UI-LABEL-01, external DCM_REAL_PPTX, remaining workflow/preview gates, final package and actual installer lifecycle, and actual user acceptance open. Leave DCM21_PROBE_EMPTY_DESTINATION unset during acceptance. PR #1 stays draft; no merge/publication.
