# Asynchronous PDF startup implemented; failed layout optimization removed

13 November 2026 · DCM QuickSort v1.0.71 · release/v1.0.71.
Full native UI matrix: `4c1c32d668955a2073ac496d0f3d31e5fe1cdf43`.
Restored/rebuilt application: `4e07b99b313027794c05a6bf280208513d869c34`.

## Result and boundary

Blocking document/runtime validation and suspended process creation now run in a single worker, not QuickSort's UI thread. The UI alone adopts and resumes the still-current request. Cancelled/obsolete results are disposed, pending file operations refuse until cleanup, and normal Close keeps the message loop alive while preparation drains. Exact runtime/policy checks and no-print/no-write restrictions remain.

Two completed retained-implementation runs measured first-open queued acknowledgements of **45.7 and59.1ms**, versus **1497 and1758ms** before the change. Across all eight openings in those two runs, longest acknowledgements ranged **45.7–105.4ms**. First-document ready status took **1.62–1.73s**, while repeats took **0.387–0.450s**. Ready status is not an exact first-rendered-pixel timestamp.

An earlier asynchronous run recorded275/321ms delays on two repeated opens. Those observations are retained: **the blocking-creation correction is implemented and safety-tested, but the broader shared-load250ms target remains open.** The target is provisional, not a published standard or user-agreed SLA. These are shared-machine fresh-process/repeat measurements, not cold-boot or flushed-cache benchmarks. No cache, antivirus or system-wide changes were made.

## Ownership and cancellation

One worker plus one latest queued request bounds concurrency. Workers capture request values and do not access app/window state. Cancellation cannot transfer ownership to the UI; disposal failures retain the partial resource and block operations. Process creation uses an owned kill-on-close job assigned at creation, verified before the suspended process can resume. No fallback to an unowned process was added.

The pending-close native test observed Loading, requested Close, then recorded process creation returning and cancelled-preparation disposal with **no process-resumed event**. It exited normally in1342ms and verified no private viewer/profile or PDF lock remained. This is safe cleanup, not an instant-close claim.

The new portable ownership parent contains seven cases; the new Windows pending-start parent contains Rename, Move and Close cases. Existing six release-refusal cases remain in the baseline. Focused race tests and Windows cross-compilation passed. All source was authored locally, saved in GitHub, and validated via FlowDev on Lenovo; no remote source edits.

## Native results at4c1c32d

| Selection | Result |
|---|---|
| Full Windows baseline | **372 passed,0 failed,1 skipped**;373 parents; exit0,no timeout |
| Startup measurement | Two complete runs,four opens each; both later receipts below250ms |
| Close during pending startup | **3 checks passed** |
| Switching/loading cancellation/active-PDF shutdown | **11 passed** |
| PDF Undo and controlled viewer-failure recovery | **9 passed** |
| Missing runtime/damaged/password-protected PDFs | **12 passed** |
| Readable scrolling | **14 passed** |
| Navigation/zoom/resize | **23 checks+12 bursts passed**; all24 raw/natural sampled comparisons zero |

The retained skip is TestRealGeneratedPPTXIntegration, not a pass. Two new Go parents explain the increase from370 to372; their nested cases are not added again. UI checks/repeats and Go totals are separate selections. Preview-only fixtures preserved names/bytes and empty journals; Undo has its separately verified operation/journal outcomes. Tests used disposable files and private runtime/settings copies, not user documents or the installed working copy.

| Run/open | Ready status ms | Max queue acknowledgement ms |
|---|---:|---:|
| A/first |1734.1863|45.7167|
| A/repeat1 |386.7859|75.8342|
| A/repeat2 |419.9465|76.0621|
| A/repeat3 |418.3657|76.2614|
| B/first |1623.2187|59.1361|
| B/repeat1 |419.8913|69.0331|
| B/repeat2 |402.0476|57.3876|
| B/repeat3 |449.7964|105.3915|

The timing loop/budget was unchanged. Later test-only diagnostics add failure geometry/visibility and a guarded failure screenshot, without new sleeps, restoration or relaxed assertions.

## Visual review and retained failures

One representative first-run startup screenshot was opened: selected filename, readable intended PDF and containment agree. All four full-window captures in that run have identical SHA256015a0bfc1ae69115074c6d41576384f1e08e9e5cd709d85462209544800cf5fb. The other three were not individually opened. Repeat/regression images were not separately reviewed. The lifecycle failure image was also inspected and showed the expected empty-file error. Raw visual flags are unchanged.

The initial lifecycle assertion raced asynchronous validation. Its saved failure status/image contained the exact expected nonempty-file error. The test now waits for that same exact error under the original deadline before retaining the existing viewer/file assertions. Windows subtests were pinned individually to their OS threads and fixture writes checked. These are test corrections, not error-message product fixes.

A subsequent optimization skipped full host layout at unchanged widths. It was **not accepted**. At948ff3f5 there was a minimized capture failure and then a containment failure. Diagnostics at8e510e2e reproduced actual containment failure with host_minimized=false and host_visible=true: pane(333,259,1238,792), viewer(334,260,1463,923). The deeper cause is not established; no universal DPI diagnosis is claimed.

At4e07b99b the optimization was removed, restoring exact viewer blob3b7041b3115ecbebd0c0f247f8a6e52d05aad93c from4c1c32d and deleting the unused helper. **The rebuilt application matches the UI-tested executable byte-for-byte:** SHA2566f09034154c40fca5737797cd5e341f6817d301b3bfc6e3375fe0bea0930916a,7311360bytes. The full UI suite was not rerun after rollback; earlier receipts retain their original source identities.

An unused compressed patch-transport staging attempt was replaced with readable hash-guarded source edits before any execution. A read-only evidence collector needed a syntax correction. Neither is a product fix or a discarded native test failure.

## Evidence and exact reports

All native reports are under D:\FlowverseReportsRepo:
- Baseline: dcm-quicksort-baseline-20261113-015918-9335041c.
- Startup A/B: dcm-quicksort-ui-sumatra-startup-20261113-020024-680ad7e0 and020046-5b08a6d6.
- Pending-close: dcm-quicksort-ui-sumatra-lifecycle-20261113-020107-f702acee.
- Lifecycle: dcm-quicksort-ui-sumatra-lifecycle-20261113-020120-453e2980.
- Undo: dcm-quicksort-ui-sumatra-undo-20261113-020156-63467c75.
- Errors: dcm-quicksort-ui-sumatra-errors-20261113-020221-cd50ab22.
- Scroll: dcm-quicksort-ui-sumatra-scroll-20261113-020304-8d93a62f.
- Resize: dcm-quicksort-ui-sumatra-20261113-020332-e5aab7d9.
- Tested build: dcm-quicksort-build-20261113-015835-a9d8ea4b; receipt c73347496f0771479c6f76cca3ae150579fde48b7d3db346b6ae5d17f30b32fd.
- Rollback build: dcm-quicksort-build-20261113-022343-3455c053; receipt9a5a80a192979a1f9cdf6ee161cc4b155a1b630176155f0947506ee40c2b2aad.

Further hashes, observations and failure paths: evidence/sumatra-async-startup-summary.json. All failures are retained. An unrelated LibreOffice interactive session was observed but not modified or asserted to be the cause.

## Continuation

Retain asynchronous preparation and full transition layout. Profile remaining shared-load attachment latency without skipping required geometry or changing acceptance timing. Keep all prior safety, scrolling and resize regressions. No numbered step closed: DCM01–18 and20–27 retain completion,19 blocked,28/29 pending. General labels/error banner, external PPTX, other operation/path variants, runtime distribution/package and human acceptance remain open. Sumatra remains optional; Windows.Data.Pdf remains default. PR#1 stays draft; no installation, association, printing, merge or publication.

Technical reference for creation-time job assignment: Microsoft, https://devblogs.microsoft.com/oldnewthing/20230209-00/?p=107812 . Project timings are from the native evidence, not the API documentation.
