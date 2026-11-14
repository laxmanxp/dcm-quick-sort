# DCM-27 complete — image previews, sizing and truthful failures

11 November 2026 · standalone DCM QuickSort v1.0.71 · branch `release/v1.0.71`.
Native-tested revision: `b92c9abdac87e7f9c937e51ca8e7023eae5e3417`.
Product correction already committed by the interrupted attempt: `4043f3e564d5c22dbac8a318f4c7fc07de970a6f`.

## Result

**All 23 native image checks passed, and all 22 final screenshots were individually opened and inspected.** The suite uses 12 disposable files: ten valid image fixtures and two invalid files. Nineteen checks compare actual screenshot pixels against known fixture colors, expected geometry and white margins; three check error states and one checks source-handle release. The window closed normally with exit 0, children drained, source/executable identity stayed unchanged, and all fixture bytes matched their original hashes. No destination files or file-operation journal entries were created.

Only DCM-27 is newly closed. Completed: DCM-01–18 and DCM-20–27, totaling 26 checklist steps including five setup/diagnostic steps, not a product-completion percentage. DCM-19 remains blocked. Human acceptance is not inferred from automation.

## Recovered work and actual defect

This retry found DCM-27 tooling and an image-error correction already committed after the last reported DCM-26 checkpoint. No further application or test implementation was changed in this retry. The existing code was reviewed, freshly built and exercised through FlowDev, then its evidence was inspected and recorded.

The retained native failure at `2dc099d5` reported `DCM27_TRUTHFUL_ERROR_MISSING: DCM-27-11-empty.png`. Its original screenshot was reopened: an empty PNG was incorrectly described as having no installed Windows preview handler, while the footer still reported a successful image Fit state. This is a reproduced error-reporting defect, not data loss.

The already-committed correction in `main_windows.go` propagates the image loader's specific empty/read/decoder error instead of falling through to the generic missing-handler message. It clears the inappropriate success state and disables image-sizing controls when loading fails. The existing GDI+ memory-stream strategy is retained, including source-handle release. The boolean loader wrapper used by PDF raster loading remains available; this does not certify PDF UI behavior before DCM-28/29.

One Windows image-loader parent test was added during the interrupted attempt. It checks missing/empty/damaged images, cleanup, valid-image recovery, decoded dimensions, original bytes, exclusive source-file access and the existing boolean wrapper. This accounts for the increase from 361 to 362 passing Go tests. No preview engine, UI framework or installer was replaced.

## Verified behavior

| Scenario | Verified outcome |
|---|---|
| PNG, JPG/JPEG, BMP, static GIF, single-page TIF/TIFF | Known image content appears with the expected orientation and aspect ratio for landscape, portrait and square fixtures. Both suffix aliases and uppercase PNG are exercised. |
| Transparent PNG quadrant | Transparent area displays the existing white background, not stale pixels. |
| Fit and Fit Width | Fit preserves the full aspect ratio; portrait Fit Width starts at the top and clips the oversized lower portion as expected. |
| 100%, 110%, minus and Fit reset | Actual rendered geometry and pixels follow the selected mode. |
| Large-image panning | The owned drag handler shifts the visible image by the expected offset. |
| Next after zoom/pan | Selects the next fixture and restores its Fit view without retaining the preceding offset. |
| Empty/corrupt/locked file | Specific error appears, no old image remains, and sizing controls are disabled. |
| Unlock and reselect | Correct image content returns. |
| File-handle safety | Exclusive read access succeeds while the image is previewed; all source bytes remain unchanged. |

The 19 rendering cases use 42–64 independent sample points each, avoiding text labels, quadrant seams and antialiased edges. Screenshots were also reviewed visually; the pixel checks do not replace that review. The native drag is simulated through owned window messages, not physical mouse-input certification. Static GIF and one-page TIFF fixtures do not establish animation or multipage support. All codec variants, EXIF orientation, color profiles, semi-transparent blending and every DPI/window size are outside this bounded matrix.

## Fresh results

All final runs are pinned to `b92c9abd` and are separate selections, not a combined test count.

| Selection | Result | Directory under `D:\FlowverseReportsRepo` |
|---|---|---|
| Complete application baseline | **362 passed, 0 failed, 1 skipped**; 363 parent results, exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261111-180442-7445a134` |
| Image acceptance | 23 checks passed; 22 images reviewed; normal exit 0 | `dcm-quicksort-ui-image-20261111-180336-6ac29363` |
| Text-preview regression | 17 cases passed; command exit 0, source unchanged | `dcm-quicksort-ui-text-20261111-180536-a3ca65d5` |
| Navigation regression | 22 cases passed; command exit 0, source unchanged | `dcm-quicksort-ui-navigation-20261111-180557-b7ac0fc1` |

The two regression runs' new screenshots were not separately reviewed here. The retained skip is `TestRealGeneratedPPTXIntegration`; its identified `DCM_REAL_PPTX` fixture remains required before DCM-32/47. Installer and packaging suites were not rerun.

Build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261111-180146-fc67e68b\receipt.json`.
Receipt SHA-256: `af23dfbaeaf5356ae4c152cd23e1ffe12fe0d354a427c87f52a783bfdbfddd5e`.
App SHA-256: `2f579ceba816c1f48853a688fe18ba0e48589f0877deaca3ca836dd47b25d5c2`; 7,185,920 bytes.
The 286-file source manifest has SHA-256 `78c26e8b8cbcb1486dcf422096d383077c4832ecab4cee9e9dcfeba2cfff97e3`.
Separate visual review and image hashes: `qualification/evidence/dcm-27-native-summary.json`.

## Retained failures and environment recovery

The interrupted attempt's product RED remains in `dcm-quicksort-ui-image-20261111-163352-b5f8d3b3`. Other unsuccessful reports are retained: `162119-9f5ce0d5` (status expectation), `162652-248acfbc`, `162931-fdfa175a`, `164121-de968742`, `164401-7b26194d` (hidden/minimized capture), and `162746-90307cb9` (window-text deadline), all dated 20261111. They are not accepted runs. This retry does not retroactively establish the cause of every earlier interruption.

The existing harness permits one recorded retry only for a hidden/minimized capture; it still requires the original capture guards and rechecks the selected target. **The final run used no retry: all 22 captures succeeded on attempt 1.** Raw `visual_reviewed=false` fields remain unchanged; this separate record documents inspection. Raw screenshots remain on Lenovo because a small desktop edge is visible.

`C:\FlowverseReports` is now a junction to `D:\FlowverseReportsRepo`. The first update attempt through the old C: path stopped with `not a repository root`; the checkout was clean. Using the canonical D: workspace resolved it without editing remote source. All final runs used FlowDev update/test from `D:\FlowverseReportsRepo\dcm-quicksort-candidate\tools\flowdev-dcm`. Historical receipt path strings were not rewritten.

## Continuation

**Next: DCM-28 — actual multipage PDF navigation, including first/last-page boundaries.** Refresh the exact source-bound build receipt after this metadata commit. Keep DCM-19/disposable Explorer isolation, label-layout cleanup, external PPTX, remaining previews/file operations, full Batch Move/Undo, final package/setup lifecycle and human acceptance open. No host installation, registry/association changes, merge or publication. PR #1 remains draft.
