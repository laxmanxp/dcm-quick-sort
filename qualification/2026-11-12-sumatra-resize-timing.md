# QuickSort resize timing — queued-layout failure reproduced and repaint work reduced

12 November 2026 · DCM QuickSort v1.0.71 · `release/v1.0.71`.
Product/native-tested correction: `02e6efb39f08c5cfa29da4787cd4f41bf71b642a`.

## Diagnosis and acceptance boundary

The new diagnostic reproduced a concrete cause of an incomplete resize screenshot: queued asynchronous resize requests had not finished. At the failing capture, the host was still at the smaller size with incompletely repositioned/painted controls. Its next unassisted capture recovered at the intended restored size **before the diagnostic redraw**. The recovered image and later redraw reference have identical SHA-256 hashes. This is evidence of an unfinished resize sequence in that reproduction, not evidence that a forced redraw was necessary to recover.

The host's layout routine moved 49 controls with immediate repaint enabled, then requested another repaint at the end. The correction moves those controls without the 49 intermediate synchronous repaints and requests one queued invalidation/erase of the completed layout, including child non-client frames. It does not replace the UI framework, change Sumatra, relax file safety, add a user-resize capability or increase test waits.

The original intermittent failure remains preserved. This work establishes and corrects the reproduced queued-layout failure within the tested sequence; it does not retrospectively prove that every historical occlusion, minimization or repaint issue had this cause. General narrow-window label crowding remains open.

## Evidence, in order

1. **Instrumented startup attempt, `590db697`:** stopped at the existing synchronous file-selection deadline before resizing. It is not a resize RED. The diagnostic selector subsequently reused the already-tested posted-selection and bounded filename-read helper. The ordinary selector and all exact row/document/file assertions remained intact. Startup responsiveness remains a separate open issue.
2. **Synchronized diagnostic, `7b20667b`:** the original 23 behavior records and eight resize bursts passed when a posted queue marker preceded capture. No layout nesting above depth one was recorded. All eight raw/natural image comparisons were zero. This alone could not close the defect because synchronization can hide an unfinished resize sequence.
3. **Fixed-wait diagnostic, `c15bf4a6`:** the queue marker was moved after both unassisted captures, retaining three shrink/restore pairs with 120-ms pauses and 700 ms before capture. Unlike the ordinary Shot helper, these diagnostic captures do not restore/reposition the host and add another 400-ms delay. This stricter diagnostic sequence is identical before and after the product correction. The eighth burst failed because raw capture geometry was still 1520×945 pixels while the subsequent natural/reference captures were 1600×1025. The three images were individually inspected: incomplete controls in raw; clean controls in natural and reference. The natural image was taken after an additional 250-ms wait plus capture overhead, not after a repaint command. The final six traced layout calls took approximately 214–361 ms each. Those serialized calls were still working through the queued requests. The final burst state object was not appended because unequal geometry made the comparison throw; no missing state values are invented.
4. **Product correction, `02e6efb3`:** only the main layout routine's repaint policy changed. The same fixed-wait diagnostic harness, timings, raw capture order and guard checks were retained. The first corrected run passed all 23 existing behavior records plus all 12 added resize bursts. Each burst contains six requests, so the diagnostic exercises 72 additional size requests per run; these are repetitions, not 72 separate product features or Go tests.

The first corrected run's 24 raw/natural comparisons all show zero sampled differences from their later explicit-redraw references. All sampled raw states have completed layout, stable sequence values and no pending host paint. The completion marker comes **after** raw/natural captures and cannot make those captures settle beforehand. Existing geometry/ownership/visibility assertions were not weakened, and no raw receipts were rewritten.

## Timing observations, not a benchmark

| Run | Recorded layout calls | Mean duration | Maximum duration |
|---|---:|---:|---:|
| Fixed-wait pre-fix RED | 63 | 141.7 ms | 361.1 ms |
| First corrected fixed-wait run | 91 | 86.4 ms | 174.5 ms |

The two runs have different numbers of completed calls, tracing overhead and a shared Windows workload. These are diagnostic observations, not a general speed guarantee or an isolated performance benchmark. Maximum recorded nesting was one in both traces. The trace is an opt-in, bounded in-memory ring; it writes its numeric records only at normal shutdown into the test's private temporary directory. Ordinary launches do not enable it.

## Fresh verification

| Selection | Result | Report directory |
|---|---|---|
| Fixed-wait timing run 1 | 23 behavior records and 12 timing bursts pass | `dcm-quicksort-ui-sumatra-20261112-161805-7fe5f40e` |
| Unchanged timing repeat | 23 behavior records and 12 timing bursts pass | `dcm-quicksort-ui-sumatra-20261112-162103-83d13e18` |
| Ordinary Sumatra suite, tracing disabled | 23 records pass | `dcm-quicksort-ui-sumatra-20261112-162232-be320fb8` |
| Launch/empty-state regression | 12 records pass | `dcm-quicksort-ui-launch-20261112-162316-b7d36e84` |
| Text regression | 17 records pass | `dcm-quicksort-ui-text-20261112-162331-ebf78fb1` |
| Image regression | 23 records pass | `dcm-quicksort-ui-image-20261112-162353-ac8e67b8` |

All commands exit 0 against the same source and exact build. Both timing runs have zero differences in all 24 sampled raw/natural comparisons. They retain the old 23 behavior records; timing iterations are additional assertions, not additional parent-test totals.

The full native baseline remains **370 passed, 0 failed and 1 skipped** (371 parent results), exit 0, no timeout and unchanged source. No Go parent test was added; new coverage is native timing/visual acceptance. The retained skip is `TestRealGeneratedPPTXIntegration`, still not executed or counted as a pass.

**18 first-corrected-run images were individually inspected:** all 12 timing-burst raw captures, initial PDF, narrower host, ordinary repeated restoration, final restored text, and the last natural/reference pair. The three decisive pre-fix RED images were also inspected. This is not a claim that all 79 first-run captures or the repeat/regression images were separately reviewed. Raw review flags remain unmodified; the separate evidence JSON records names and hashes. Full-window images remain on Lenovo because they contain desktop edges.

The inspected raw frames show clean controls before diagnostic redraw, including the last burst. Narrow Size/Sort and Processed-heading overlaps are still visible and explicitly remain open. **DCM-UI-RESIZE-01 is recorded fixed for the reproduced queued-layout path and tested matrix**, not as universal interface or timing approval. The older raw failure and its cause uncertainty remain preserved.

The first corrected timing run preserves the existing PDF navigation, first/last boundaries, page entry, zoom/fit, toolbar containment, restricted internal Open, PDF-to-text release and representative Rename/Move checks. Disposable source contents remain intact, recorded viewer processes are checked for cleanup and the host exits normally. No user documents, installed runtime, global associations or unrelated applications were modified.

## Scope and remaining work

No numbered step is newly closed. DCM-01–18 and DCM-20–27 retain their earlier recorded boundaries; DCM-19 remains blocked and DCM-28/29 remain pending. Sumatra is optional and Windows.Data.Pdf stays the default. No installer/runtime package, merge or publication was made.

This does not approve all window sizes, DPI settings, physical mouse input, unbounded resize stress or general interface layout. The narrower Size/Sort row and Processed header still crowd each other. The damaged-file native banner still truncates a long path; startup responsiveness, readable-text scrolling at 100%/Fit Width, remaining operation/path cases, external PPTX, runtime distribution/installer and manual acceptance remain separate gates.

Next: verify readable document content after scrolling at 100% and Fit Width, while retaining the timing regression and the existing startup/layout issues. Rebuild against the exact new source after metadata changes; never relabel old receipts.

## Exact evidence

Reports are under `D:\FlowverseReportsRepo`:
- Startup failure: `dcm-quicksort-ui-sumatra-20261112-160104-3c67a2af`.
- Synchronized diagnostic: `dcm-quicksort-ui-sumatra-20261112-160503-8202f663`.
- Fixed-wait RED: `dcm-quicksort-ui-sumatra-20261112-161007-7ff511c9`.
- First corrected timing run: `dcm-quicksort-ui-sumatra-20261112-161805-7fe5f40e`.
- Full baseline: `dcm-quicksort-baseline-20261112-161936-784532bc`.

Build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-161742-8ecbe7d7\receipt.json`.
Build receipt SHA-256: `82b823f79254b07e8d86ef0a28fc68ab2b480f95944131595c5edc8d3dd7c232`.
337-file manifest SHA-256: `3d3e9c62d7d1ce1377448765b8d3110b11b5a838bb938c2579d7c840b583dc09`.
App SHA-256: `a02164f61a1334389649179f246a5b17a56a69771f334404b75962a212a2b1bc`.
Exact fixed-wait harness SHA-256 (same for RED and corrected runs): `fe3a07f02f8a14caf1fe12ff148b9f7056727db9d794475c215c8a83a8299a07`.
Product main file SHA-256: `99a43538f4189d9a98158a5dea08d950b8650640c285c6fe550654f6ec57c42b`.
Further receipt/trace/image hashes: `qualification/evidence/sumatra-resize-timing-summary.json`.

Source/tests authored locally, committed to GitHub with exact input/output hash guards, and updated/built/tested through FlowDev on Lenovo. No remote source edits. Earlier failures are retained under their original revisions.

Technical reference consulted: Microsoft Learn, MoveWindow (https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-movewindow), which documents immediate UpdateWindow behavior when repaint is enabled and the explicit invalidation requirement when disabled. Project-specific diagnosis above comes from the recorded trace and images, not from the API documentation alone.
