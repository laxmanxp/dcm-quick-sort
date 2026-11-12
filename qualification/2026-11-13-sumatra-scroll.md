# Sumatra readable scrolling — 100% and Fit Width verified

13 November 2026 (IST). Native receipt names use 12 November 2026 UTC.
DCM QuickSort v1.0.71 · `release/v1.0.71`.
Native-tested source: `401b44521ed36c835c04f39ffcd79ef6f3ee79af`.
Application implementation remains `02e6efb39f08c5cfa29da4787cd4f41bf71b642a`.

## Result

**The new `ui-sumatra-scroll` workflow passed all 14 acceptance records in two consecutive executions. All 12 full-window screenshots from the first passing run were individually inspected.** The actual expected headings, letters and digits are readable at both 100% and Fit Width, not merely a magnified coloured region or a changed zoom label.

No application, installer, Sumatra executable or runtime policy change was needed. This work adds a reusable native test helper, a focused FlowDev selector and a deterministic two-page PDF fixture. The application executable hash is identical to the preceding verified product build. Earlier failures in this pass were fixture/test-input/measurement problems; they are preserved and are not described as product fixes.

This closes the bounded readable-scrolling subcheck `DCM-SUMATRA-SCROLL-01`. **No numbered checklist step is newly closed.** The optional provider still has separate startup, presentation and release gates. DCM-01–18 and DCM-20–27 retain their recorded completion, DCM-19 remains blocked, and DCM-28/29 remain pending the broader provider acceptance. Windows.Data.Pdf remains the default; Sumatra stays opt-in.

## What the native workflow verifies

For each zoom mode, the test selects the same identified two-page PDF and follows this sequence:

| Check | Actual visible content or state |
|---|---|
| Page 1, top | **PAGE 1 - TOP - ALPHA**, with the complete digit sequence `0123456789`. |
| Page 1, middle | **PAGE 1 - MIDDLE - BRAVO**, reached by native line-scrolling rather than changing the PDF or its content. |
| Page 1, bottom | **PAGE 1 - BOTTOM - CHARLIE**, its numbers and the end-of-page text are readable. |
| Reverse to top | The original ALPHA section returns; the page number remains 1. |
| Page 2, top | The page-number entry selects 2 and the distinct **PAGE 2 - TOP - DELTA** section appears. |
| Page 2, bottom | **PAGE 2 - BOTTOM - FOXTROT** and page-2 end text appear, not a stale page-1 image. |

Those six checks in each of two modes form 12 records. The final two records verify switching to text/releases the PDF, then normal shutdown with original bytes, no operation history and clean private-viewer cleanup. Both runs exit normally with code 0.

The recorded page-1 vertical scrollbar positions are **0 → 656 → 1298 → 0** at 100%, and **0 → 384 → 753 → 0** at Fit Width. Page 2 uses the same top/bottom limits within each mode. These are native scrollbar units, not assumed physical screen pixels. The helper sends the real owned canvas `WM_VSCROLL` and `WM_HSCROLL` messages; it does not assign scrollbar values directly or modify document memory.

At 100%, the enlarged page is wider than the viewport. The test explicitly scrolls horizontally to the left before verifying the marked text blocks. At Fit Width, the horizontal range is zero. This is normal viewport positioning, not a correction to the application. Some long titles extend beyond the 100% viewport and a temporary zoom notification overlaps the large title in early captures. The targeted section headings, digit rows and inspected body text remain readable; this is not a claim that an entire magnified page fits simultaneously.

## Evidence stronger than colour-only assertions

The fixture contains black text and six narrow coloured markers, with distinct page/section tokens. It was generated locally, parsed independently and rendered for inspection before native use. Page size is 612×1000 points, ensuring real vertical scrolling at the tested geometry. The harness checks:

- exact source/build/runtime/policy identities and verified owned canvas/process hierarchy;
- actual vertical/horizontal position changes, expected page number, full marker geometry and rendered dark glyph pixels beside that marker;
- full-window captures under the existing occlusion/offscreen/minimization guards;
- separate inspection of the exact headings and numbers in all 12 first-run images.

The pixel checks corroborate rendered lettering but **are not semantic text recognition**. Exact wording acceptance comes from the separate screenshot review; no OCR was used and raw `visual_reviewed=false` receipt fields were not rewritten. The review record lists every inspected image and its SHA-256. The repeat's images and the resize-regression images were not separately reviewed.

The test uses direct Fit Width zoom in **single-page display mode**. It does not qualify continuous-page scrolling, mouse-wheel/trackpad/touch input, arbitrary fonts, scanned pages, every PDF or every DPI/window size. It samples page-1 top/middle/bottom and page-2 top/bottom, not every intermediate line or page-2 middle.

## File and process safety

Both runs use five disposable files, private settings and a private copy of the pinned unmodified Sumatra 3.6.1 executable. All filenames and SHA-256 values remain unchanged. The destination and operation journal remain empty. Switching to text removes the embedded viewer, and the PDF permits exclusive file access. Normal host shutdown leaves no recorded viewer process or private session profile; runtime and restriction-policy hashes remain unchanged.

The original runtime, user documents, installed working copy, associations and unrelated applications were not changed. No printing, package, default-provider switch, merge or publication occurred. The no-write/no-print restriction configuration is retained; this task does not add a security-sandbox claim.

## Preserved test-development failures

| Source / native report suffix | Failure and resolution |
|---|---|
| `ccbe90af` / `215621-6975d64b` | Stopped before app launch: the new ASCII PDF's bytes changed during Windows checkout. Read-only comparison established 103 CRLF insertions; LF normalization matched the original SHA exactly. A targeted `-text` Git attribute and regenerated byte-exact PDF/xref resolve storage without relaxing the checksum. |
| `5d739f3e` / `220123-9e07d43f` | At 100%, the first marker lay outside the horizontally centred viewport. The inspected image already had readable text. The test now performs actual horizontal scrolling to expose the left margin before checking vertical movement. No product repair claimed. |
| `c9d1bab4` / `220753-ab49aefd` | Seven records passed before the new checker falsely rejected Fit Width's middle marker. Its two-pixel sampling reported a 20×76 box, on the strict aspect boundary; the inspected BRAVO text and full marker were correct. Pixel-by-pixel edge refinement measures the real bounds while retaining the original colour, clipping-margin, aspect and ink limits. |

All report names use prefix `dcm-quicksort-ui-sumatra-scroll-20261112-`. The original fixtures, failure receipts and screenshots remain under their original source identities. Neither the application nor the existing resize test was modified to produce a pass.

## Fresh results at 401b4452

All report directories are under `D:\FlowverseReportsRepo`.

| Selection | Result | Report directory |
|---|---|---|
| Readable scrolling, reviewed run | **14 passed**; 12 captures inspected; normal exit 0 | `dcm-quicksort-ui-sumatra-scroll-20261112-221314-11122de8` |
| Unchanged readable-scrolling repeat | **14 passed**; normal exit 0; images not separately reviewed | `dcm-quicksort-ui-sumatra-scroll-20261112-221502-8c982330` |
| Complete native application baseline | **370 passed / 0 failed / 1 skipped**, 371 parent results; exit 0; no timeout; source unchanged | `dcm-quicksort-baseline-20261112-221419-946fd6f9` |
| Existing resize-timing regression | **23 behavior records + 12 timing bursts passed**; all 24 raw/natural sampled comparisons zero; normal exit 0 | `dcm-quicksort-ui-sumatra-20261112-221534-1f047660` |

The repeated 14 records are not 28 distinct features. The resize iterations, UI counts and Go baseline are separate selections. No new Go parent test was added, so the count remains 370 passes. The sole skip is still `TestRealGeneratedPPTXIntegration`, not a pass. Focused local Sumatra tests also passed; no complete local Linux app-suite acceptance is claimed. Installer/package, full Batch Undo, the default-provider PDF UI and every unrelated UI suite were not rerun here.

The resize regression retains the same fixed waits, raw-before-redraw sequence and capture/ownership assertions. Its pass preserves the earlier bounded correction, not universal timing or interface certification. Existing narrow Size/Sort and Processed-heading crowding remain open.

## Exact identities

Build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-221301-00846739\receipt.json`.
Receipt SHA-256: `e614df00beb5a89a2ccc47fcf0fb48bf9ee44e77a4fac8b13b2e444b79466423`.
App SHA-256: `a02164f61a1334389649179f246a5b17a56a69771f334404b75962a212a2b1bc`; 7,251,456 bytes, unchanged from the prior product build.
350-file source-manifest SHA-256: `193a37f4b9d07e4027d451bbbe148d997074afb1db4c2467cef3f3e966bb50e5`.
Fixture SHA-256: `05304a98a830954b4f5bfedf0187e78aa847248a80b0ca2ba2b5b5f843352bb0`; 6,214 bytes.
Runtime SHA-256: `719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0`.
Policy SHA-256: `f82ee9c3ff09da529f67f2ee73ab716dd53678d0f52b00ddcffc9d6dc0a908aa`.
Separate review, receipt and image hashes: `qualification/evidence/sumatra-scroll-native-summary.json`.

Code/test fixtures authored locally, stored in GitHub and validated via FlowDev update/tooling/build/test on Lenovo. No remote source edits. Raw desktop-edge screenshots stay on Lenovo; summary and review identities are saved to GitHub.

## Next

Measure first-PDF startup costs and host responsiveness under cold/warm conditions (`DCM-SUMATRA-STARTUP-01`). Do not use a longer readiness timeout or asynchronous harness notification as a product-performance fix. Keep readable scrolling and resize timing in regression coverage. Retain damaged-file banner, general labels, DCM-19, external PPTX, remaining operations/path/release cases, runtime distribution/installer and actual user acceptance. Rebuild after source or tracker commits; never relabel historical receipts. PR #1 remains draft.
