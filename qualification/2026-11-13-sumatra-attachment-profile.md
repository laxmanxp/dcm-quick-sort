# DCM QuickSort — attachment delay localized without changing pane sizing

13 November 2026 · v1.0.71 · `release/v1.0.71`.
Native-tested diagnostic source: `5294d6cf32e84038aeed93750800bd7308bad7ad`.

## Result

The remaining response delay reproduced with the retained asynchronous startup implementation. Two complete four-opening diagnostic runs produced usable phase traces and passed document identity, rendering, file preservation and cleanup checks. **Run A exceeded the provisional 250 ms queue-response target on two repeat opens (297.0 and 327.5 ms); the unchanged run B stayed below it (139.7–246.5 ms).** Both outcomes remain recorded. A completed diagnostic command is not a performance pass, and the later run does not erase the earlier overruns.

The expensive attachment work is primarily **QuickSort's own control layout and zoom-control updates**, not the page-status read alone. These two phases account for approximately **85.6–88.6% of the timed attachment handler** across the eight observations. This pass adds diagnosis, not a product-speed correction.

## Measured phases

| Phase | Observed range across eight opens |
|---|---:|
| Full host layout, including all 49 control moves | 113.6–246.0 ms |
| Host zoom-control visibility/enabled updates | 38.0–87.9 ms |
| Separate viewer resize request | 11.2–17.9 ms |
| Initial loaded-document status handling | 5.4–10.7 ms |
| Embedded toolbar compatibility repair | 5.7–18.9 ms |
| Final document-status handling | 0.0–1.8 ms |
| Entire recorded attachment handler | 177.0–339.4 ms |

The ranges are from different observations and must not be added as if their minima/maxima occurred together. Zero values reflect measured timestamp granularity, not proof of zero execution cost. Queue acknowledgements and handler duration are different measurements: a posted token may arrive after a handler starts, and loaded-document status is set before some tail work. No exact first-rendered-pixel timing is claimed.

In the slowest handler sample, full layout took **246.036 ms** and zoom-control updates **49.623 ms**, out of **339.372 ms** total. Seven drop-down `MoveWindow` calls alone accounted for **189.085 ms**: extension43.359, size33.474, source alias28.063, rename tool23.687, age22.641, sort19.005 and category18.856 ms. The next-largest recorded call was the processed list at8.775 ms. This locates elapsed time at specific call sites; it does not prove why Windows/scheduling spent that time, nor that every move can safely be omitted.

| Run / open | Ready-status time (ms) | Maximum queued acknowledgement (ms) | Attachment handler (ms) |
|---|---:|---:|---:|
| A / first |1860.902|218.449|225.683|
| A / repeat1 |692.352|297.032|315.184|
| A / repeat2 |874.658|327.496|339.372|
| A / repeat3 |736.489|203.643|203.746|
| B / first |1844.537|200.743|206.515|
| B / repeat1 |888.973|246.485|263.378|
| B / repeat2 |605.330|167.498|196.351|
| B / repeat3 |595.046|139.725|176.998|

These are shared-machine first-process/repeat observations with tracing overhead, not a cold-boot/cache-flushed benchmark or a general latency guarantee. No system settings, cache flushing, antivirus settings or unrelated processes were changed.

## What changed, and what explicitly did not

Added a UI-thread-only, opt-in attachment scope that records before/after timestamps for the existing control moves and the remaining attachment phases. It reuses the bounded1024-event startup trace and flushes numeric events only at normal exit. Each of the eight openings records all49 moves. Workers keep their separate request IDs and do not access the attachment scope.

The existing **49 MoveWindow argument lists and ordering**, full transition layout, final redraw request, startup timing harness, worker ownership/cancellation logic and runtime/policy checks are unchanged. The failed optimization that skipped whole host layouts remains removed. Nothing was resized by the measurement code, and no guard, wait or target was relaxed.

A new Windows parent test verifies disabled tracing, nested scope restoration, outside-scope silence and zero-ID behavior. Local Windows test cross-compilation and focused portable Sumatra/ownership race tests passed. The exact patch was replayed against its original input hashes locally before publication and cross-compiled by GitHub before application. Lenovo validation used FlowDev update/tooling/build/test; no source files were edited on Lenovo.

A local patch-generator uniqueness assertion was corrected before publication. A read-only PowerShell summary collector initially used dictionaries where Measure-Object expected properties; its output conversion was corrected. Neither was an application failure or a discarded native result; raw receipts/traces were not modified.

## Native verification and image review

| Selection | Result |
|---|---|
| Startup/attachment measurement | Two completed runs, four PDF openings each; run A responsiveness fails, run B passes the provisional target |
| Complete Windows application suite | **373 passed,0 failed,1 skipped**,374 parent results; exit0,no timeout,source unchanged |
| Close during pending startup | **3 checks passed**, normal exit0, cleanup confirmed |
| Switching/loading cancellation/active-PDF shutdown | **11 checks passed**, normal exit0 |
| Navigation/zoom/resize timing | **23 behavior checks plus12 timing bursts passed**; all24 raw/natural sampled comparisons zero |

The new diagnostic-scope parent explains the increase from372 to373 passing tests; its internal cases are not counted again. The sole skip remains `TestRealGeneratedPPTXIntegration`, not a pass. UI repetitions, timing bursts and application-unit totals are separate selections. Full Undo/error/scroll and unrelated preview/installer suites were not freshly rerun in this diagnosis-only step.

**All four full-window startup images from run A were individually opened and inspected.** They show the correct selected filename, readable intended PDF and contained viewer. The four captures are byte-identical, but all were opened; raw review flags remain unchanged. First-run content crops were automatically checked, not separately inspected. Run B and regression images were not separately reviewed. Desktop-edge images remain on Lenovo.

Startup runs preserved all four disposable file names/bytes, empty destination/journal and private viewer/profile cleanup. Lifecycle and resize runs retained their own fixture checks; representative Rename/Move belongs to the resize suite and does not imply an empty journal for that suite. No user documents, installed runtime, working copy or associations were changed.

## Exact evidence

Reports under `D:\FlowverseReportsRepo`:
- A: `dcm-quicksort-ui-sumatra-startup-20261113-032014-6ac5acb9`.
- B: `dcm-quicksort-ui-sumatra-startup-20261113-032205-3e83c8ae`.
- Baseline: `dcm-quicksort-baseline-20261113-032229-c7441c1d`.
- Pending close: `dcm-quicksort-ui-sumatra-lifecycle-20261113-032325-7e5b69e1`.
- Lifecycle: `dcm-quicksort-ui-sumatra-lifecycle-20261113-032340-62160fbf`.
- Resize: `dcm-quicksort-ui-sumatra-20261113-032419-b0e1cc88`.
- Build: `dcm-quicksort-build-20261113-031920-176d38b8\receipt.json`.

Build receipt SHA256 `e114fadaaf3434992802194722bc8c8c102f95576787155232b296eb660e2ba1`.
App SHA256 `34c17d1ccc72259fffc4425210b4839674cb1d3e3a75d36b5bba6da9b9188b18`,7,327,232bytes.
383-file manifest SHA256 `9c9109c92fa556ae46c7f85049859739468c6bd81fc504468838f316f8323a9d`.
Further phase values, receipt/trace and reviewed-image hashes: `evidence/sumatra-attachment-profile-summary.json`.

## Next bounded correction

Keep the complete layout calculation and always-correct preview sizing. First target unnecessary native updates to unchanged drop-down geometry and unchanged host zoom-button state, with explicit tests for resize/DPI changes, hidden/restored controls and preview transitions. Do not repeat the rejected whole-layout shortcut or assume cached dimensions are valid across DPI/handle changes. Apply the smallest proven optimization, then rerun this same phase/queue test and resize/lifecycle/Undo/error regressions.

No performance fix or numbered completion is claimed here. DCM-SUMATRA-STARTUP-01 remains open; DCM28/29 pending,19 blocked,01–18 and20–27 retain their earlier boundaries. General labels/error banner, externalPPTX, remaining operation/path variants, distribution/installer and human acceptance remain separate. Sumatra stays optional, Windows.Data.Pdf default, PR#1 draft; no merge/publication.

Technical context checked: Microsoft Learn MoveWindow documents that moving a control delivers window-position/size messages, even apart from immediate repaint; SendMessageTimeout documents cross-thread waiting until response/timeout. These APIs motivate measuring actual call intervals, but the diagnosis above is from project traces, not documentation alone. References: https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-movewindow and https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendmessagetimeoutw .
