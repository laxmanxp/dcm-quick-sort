# Sumatra lifecycle — switching, cancellation and active-PDF shutdown verified

12 November 2026 · DCM QuickSort v1.0.71 · `release/v1.0.71`.
Native-tested revision: `deb8373dbb52c44d963f20fce49b9ea074f7d59e`.
Application implementation remains the previously accepted `24ff2f24` version.

## Result

**The new focused lifecycle check passed all 11 acceptance records in two consecutive executions.** All seven full-window screenshots and three PDF-content crops from the first execution were individually inspected. The repeat's images and the existing `ui-sumatra` regression's new images were not separately reviewed.

No application, installer or runtime changes were necessary. This work adds the reusable FlowDev `ui-sumatra-lifecycle` selector, an isolated PowerShell harness, and a bounded test-only helper that reads QuickSort-owned status controls without querying the foreign-process viewer tree. The general native adapter's ownership/capture guards are unchanged. All existing tests remain intact.

## Verified behavior

| Scenario | Actual evidence |
|---|---|
| Direct PDF A to PDF B | Distinct first-page artwork/text (teal versus orange, different title), expected document in the owned viewer command line, matching selected filename, and prior viewer exit. |
| PDF to text | Exact text-content readback, correct filename/status, no embedded PDF frame or private viewer process, and exclusive read access to both PDFs. |
| Cancel loading by selecting text | Ten immediate PDF-to-text abandons per run. All ten were observed in the host's **Loading PDF** state before cancellation in both runs. The text preview returned and no stale viewer/status reappeared during the additional one-second settle check. No new Cancel button is implied. |
| Repeated attached switching | Four A-to-B-to-text cycles per run, checking selected document/process identity and release on every cycle. This is bounded rapid-switch coverage, not a long-duration stress benchmark. |
| Empty or invalid-header PDF | Specific error shown without a viewer being started; no previous document remains visible. |
| Recover after errors | Text recovers after the empty file; a valid second PDF renders correctly after the invalid-header file. |
| Close with a PDF still active | Normal host exit 0, no surviving recorded/private viewer, no leftover session profile, and exclusive access to both PDFs. Closure was measured at 393 ms and 374 ms in these two runs, not a general performance guarantee. |

Each run makes 40 explicit file selections and records 11 attached viewer instances. The ten abandoned startups are separately evidenced by the observed loading state and post-cancellation absence checks; they are not falsely counted as individually recorded attached instances. The two cancellation sequences and repeated cycles are not added to the 11 acceptance-record count.

Seven disposable source files keep their exact names/bytes. The destination and operation journal remain empty. Runtime, policy and application hashes match before/after testing, and the tracked source stays unchanged. The existing private unmodified Sumatra 3.6.1 runtime and no-print/no-write policy are used. User documents, associations, installed applications and the unrelated LibreOffice interactive session were not changed.

## Fresh results — all at deb8373d

| Test selection | Result | Report under `D:\FlowverseReportsRepo` |
|---|---|---|
| Lifecycle, reviewed run | 11 records passed; 10 captures inspected; normal close 0 | `dcm-quicksort-ui-sumatra-lifecycle-20261112-093613-56838102` |
| Lifecycle, unchanged repeat | 11 records passed; normal close 0; 10 loading states observed | `dcm-quicksort-ui-sumatra-lifecycle-20261112-093758-712f52cb` |
| Complete native application baseline | **367 passed, 0 failed, 1 skipped**; 368 parents, exit 0, no timeout | `dcm-quicksort-baseline-20261112-093728-3abdf19f` |
| Existing Sumatra navigation/zoom/resize/operations suite | 23 records passed; exit 0 | `dcm-quicksort-ui-sumatra-20261112-093827-5a51793c` |

These are distinct selections, not a combined total. No Go parent test was added, so the baseline count is unchanged. The existing skip is `TestRealGeneratedPPTXIntegration`; it remains unexecuted, not passed. Local focused `TestSumatra` tests also passed; they are not native UI evidence. There were no failing native runs or product corrections in this pass. This qualifies existing behavior rather than claiming an invented RED-to-GREEN repair.

## Identity, review and limitations

Build: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-093555-d16a9c40\receipt.json`.
Build receipt SHA-256: `d15f6b024c9cc893e1f916813cbaeb930880408d75170e6e24f06b1f2a9cd3ab`.
App SHA-256: `a2bd4063a063431b8267a0ff50ed9157caea74ef3b965b391a6adc08d393a463`; 7,230,464 bytes, matching the prior product build.
311-file manifest SHA-256: `5d6677651bf3ab112f45f048c4d1bd86afb01216234a03bd2b8a096920563d51`.
Separate review/receipt/image hashes: `qualification/evidence/sumatra-lifecycle-native-summary.json`.

The screenshots show the correct distinct documents, restored text, clear empty/header errors and valid-PDF recovery immediately before close. Raw `visual_reviewed=false` values were not rewritten; this record documents the separate inspection. Small desktop edges in full-window captures remain local to Lenovo. Existing general label crowding is not waived.

Selection is automated through the real native list notification handler, not physical mouse input. Cancellation means changing selection after the host reports loading; it does not establish every decoder phase or a new cancellation control. The attached cycles verify identity/release, while pixel/content review is performed at representative checkpoints, not every intermediate frame. Invalid-header rejection is **not** coverage of malformed PDFs with valid headers, encrypted/password PDFs, huge files, crashes, missing runtime, denied process termination, or operating-system shutdown.

**No numbered step is newly closed.** DCM-28/29 remain pending the remaining provider checks; Sumatra remains optional, Windows.Data.Pdf stays the default, and packaging/distribution and actual user acceptance remain open. DCM-19, the external PPTX fixture and general layout issues are retained.

Next: verify individual Undo with an active/reopened PDF, then safely exercise viewer failure/release-refusal and valid-header corrupt/password-file handling. Keep scroll-to-readable-text at 100%/Fit Width and final runtime/setup distribution acceptance separate. Rebuild after later source or metadata changes; do not rewrite old receipt identities. PR #1 stays draft; no merge/publication.
