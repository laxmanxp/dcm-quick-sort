# DCM QuickSort — redundant control updates reduced, pane sizing preserved

13 November 2026 · v1.0.71 · `release/v1.0.71`.
Native-tested source: `cc6a1a37308ba57be37c33c81e743b668bb89a1c`.

## Result

The targeted optimization is implemented and verified on Lenovo. All eight PDF openings across two consecutive runs stayed below the unchanged provisional 250 ms posted-message response target: **16.8–156.3 ms**. The preceding diagnostic recorded **139.7–327.5 ms**, including two overruns. Full host-layout time fell from **113.6–246.0 ms** in that preceding sample to **9.6–52.8 ms** in the new sample. These are source-bound shared-machine observations, not a universal latency guarantee or isolated paired benchmark.

The complete layout calculation and preview-pane sizing remain in place. This is not the previously rejected whole-layout shortcut. The optimization omits only native dropdown moves whose live state already matches the request, and redundant host zoom-control state changes.

## Small, guarded product change

Eight dropdown call sites now inspect the current control rather than assume a cached size. A move is skipped only when the control belongs to the calling UI thread, is a closed ComboBox, has equivalent thread/control/parent DPI-awareness contexts, and its live parent-relative position, width and full dropped-down height match exactly. Unknown state, changed geometry, an open list, mirrored layout or mixed awareness contexts use the original MoveWindow path. No geometry cache survives handle recreation or display changes.

Zoom-button visibility/enabled updates compare the child's own WS_VISIBLE/WS_DISABLED styles. Ancestor visibility is not substituted for child state. The original ShowWindow/EnableWindow behavior is retained when a real state change is needed.

Each of the eight measured attachment events skipped **eight redundant dropdown moves and five redundant EnableWindow calls**. **None of the five required button-hide changes was skipped.** The remaining zoom-state work still measured 25.2–74.5 ms; this step does not claim that phase was eliminated.

All 49 layout target calculations and their argument ordering are retained: 41 original moves and eight guarded moves. Preview sizing, final repaint request, asynchronous viewer preparation, cancellation ownership, runtime checksum and restriction policy are unchanged. No UI framework migration, printing, dependency replacement or default-provider change.

## Measured results

| Run/open | Ready status (ms) | Maximum queued acknowledgement (ms) | Full layout (ms) | Attachment handler (ms) |
|---|---:|---:|---:|---:|
| A / first |2258.1232|59.8338|9.584|79.620|
| A / repeat 1 |390.1025|108.0138|24.610|102.606|
| A / repeat 2 |392.7195|79.6821|20.308|91.591|
| A / repeat 3 |391.7405|31.0088|30.186|72.696|
| B / first |1579.0698|16.8288|11.538|49.780|
| B / repeat 1 |533.8335|122.5526|52.769|118.210|
| B / repeat 2 |693.8472|156.3303|35.012|166.016|
| B / repeat 3 |486.3050|16.8132|10.306|67.565|

The entire attachment handler measured **49.8–166.0 ms**, versus **177.0–339.4 ms** previously. Ranges span different observations and must not be combined as if their extrema occurred together. Queue response, handler duration and loaded-document status are different metrics. The first PDF was reported ready in **1.58–2.26 seconds**, so completed first-document startup is not uniformly faster. The claim is reduced host UI work and successful bounded responsiveness measurements, not guaranteed faster rendering.

Both runs used a fresh host/private runtime copy and three repeated openings. OS caches were not flushed; runtime bytes were pre-read for validation. Shared workload, tracing and the small sample limit inference. The 250 ms target remains a project diagnostic criterion, not an externally prescribed or user-agreed service level. Historical overruns remain preserved under their original revisions.

## TDD and Windows control tests

The initial native RED at `95ad5af2` ran the unchanged product with unused pass-through helper implementations. Existing 373 tests passed; the two new contract parents failed on redundant dropdown moves and repeated child visibility updates with a hidden parent. That failure is retained as `dcm-quicksort-baseline-20261113-043206-3f9cba35`.

Four new Windows parent tests now pass. They cover actual dropdown geometry under unaware and per-monitor-v2 contexts, changed position/width/drop-list height, external movement, parent movement, hidden/restored controls, handle recreation, item-height changes, button-state idempotence, blank/text/image/Sumatra transitions, and a mixed caller-DPI fallback compared against the original native call. The tests lock their own OS threads and destroy their private windows.

This is **DPI-awareness-context testing, not a physical move between differently scaled monitors or changing Windows display settings**. Arbitrary monitor/scaling environments and manual interaction remain outside this bounded acceptance.

Local focused race tests and Windows test cross-compilation passed. Two preapplication GitHub checksum checks caught helper formatting/trace-counter and test-comment byte mismatches. The staged bytes were reconciled to the locally compiled source; exact checks remained enabled. Neither failure was discarded, and neither was a product runtime failure. No native timing/assertion or screenshot guard was relaxed.

## Fresh regression matrix at cc6a1a37

| Selection | Result |
|---|---|
| Complete Windows application suite | **377 passed, 0 failed, 1 skipped**, 378 parents; exit 0, no timeout |
| Startup/attachment measurement | Two complete four-opening runs; all eight acknowledgements below 250 ms |
| Close during pending startup | **3 checks passed** |
| Switching/loading cancellation/active-PDF shutdown | **11 passed** |
| Undo and controlled viewer-failure recovery | **9 passed** |
| Missing runtime/damaged/password-protected PDFs | **12 passed** |
| Readable PDF scrolling | **14 passed** |
| Navigation/zoom/resize | **23 records + 12 timing bursts passed**; all 24 sampled raw/natural paint comparisons zero |
| Text / image / sorting | **17 / 23 / 9 checks passed** |
| Filters | **Passed**, 74 recorded action/readback entries, not 74 distinct features |

Four new parent tests explain the increase from 373 to 377. The sole skip remains `TestRealGeneratedPPTXIntegration`, not a pass. UI records, repetitions and Go totals are separate selections. The full serialized validation process completed with exit 0; source-integrity checks remained clean. No installer/package or default-provider PDF acceptance is claimed.

Preview-only suites preserved fixture names/content and their empty journals; the Rename/Move/Undo suites retained their distinct operation and journal checks. Private viewer/profile cleanup remained intact. Only disposable fixtures and test-private runtime/settings copies were used. User documents, installed working copy, original runtime, associations and unrelated processes were not modified.

## Visual review

**Nine full-window images were individually inspected:** all four startup images from run A, plus smaller/restored/narrower/repeated-restored/text-restored views from the resize suite. The PDF, selected filename and containment agree. The restored host is clean and its zoom controls return for text. Raw visual-review flags were not rewritten; image names/hashes are in the separate evidence file.

The existing **narrow Size/Sort overlap and crowded Processed heading remain visible**. This optimization does not fix or approve those labels. Startup run B, content crops and other regression images were not separately inspected. Desktop-edge screenshots remain on Lenovo rather than being uploaded to the public repository.

## Exact identities and continuation

Build: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261113-044538-b6b19d38\receipt.json`.
Build receipt SHA256: `009ea9790ee3e8ef3a3b94d97245e9c5b892608ee27ac85b674d663fbde7c383`.
App SHA256: `13fdc03eca1d049c8f88e6bb4ed2c402015bbb6a4ad30aa0aa4ef30c1356d5cd`, 7,331,840 bytes.
392-file manifest: `26d659c1a3f30f0b314b8876750b9b846bc0326bf95359fb3f0eae439a31b3fc`.
All report paths, receipt/trace/image hashes and sample values: `evidence/control-update-optimization-summary.json`.

Code was authored locally, stored in GitHub and validated using FlowDev on Lenovo. No remote source edits. Rebuild after subsequent source/metadata commits; do not relabel historical receipts.

No numbered step is closed here. DCM01–18 and20–27 retain completion;19 remains blocked;28/29 remain pending broader acceptance. The specific redundant-update correction is verified, while wider environment/manual/performance guarantees remain separate. Sumatra stays optional, Windows.Data.Pdf stays default; PR #1 remains draft, with no merge or publication.

**Next: resolve the known compact-width filter/header label overlaps, retaining these startup and resize regressions, then review the remaining DCM-28/29 acceptance gaps.** Do not substitute repeated profiling for this completed bounded optimization. Preserve damaged-file banner, external PPTX, operation/path variants, distribution/installer and human-acceptance gates.
