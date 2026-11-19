# DCM QuickSort — compact labels and Processed header corrected

13 November 2026 · v1.0.71 · `release/v1.0.71`.
Native-tested source: `2cafe818b16ccb2c861ac1d5cd74731ab446c7db`.

## Result

**The focused compact-label workflow passed all 24 behavior records in two consecutive executions. Each run also passed 20 label/value/overlap audits.** Nine first-run full-window screenshots and the original clipping screenshot were individually inspected. The targeted `DCM-UI-LABEL-01` defect is fixed for the tested candidate, font and window geometries; final-candidate/manual and wider display-environment acceptance remain separate.

The previous implementation was tested first. The new audit reported seven clipping findings: File Type, Extension, Sort By, the long selected sort value, New Name, Vault Location and the Processed heading/count. Its raw screenshot independently showed the clipping. The correction changes layout, not labels to abbreviated substitutes.

## What changed

When the five filter groups and Clear Filters cannot fit in one row, filters now occupy two rows: File Type / Extension / Age, followed by Size / Sort By / Clear Filters. The sort field accommodates `Date Created > Type > Name` at all three native-tested widths. At sufficient width the planner retains a one-row layout.

The Processed title/count has the full column width on its own row. Undo Batch and Undo sit below it, with the search field and history below those buttons. New Name and Vault Location labels have wider allocations. No control, caption or file-operation function was removed.

The planner recomputes from the current client width. Existing live-state dropdown guards, complete preview-pane positioning, final repaint behavior and asynchronous Sumatra ownership/cancellation remain. The extra compact filter row consumes 34 logical vertical units; the bottom workflow controls retain their bottom alignment. A separate button row consumes space above the Processed list. These are deliberate space/readability tradeoffs, not missing content or new scrolling features.

The Sumatra executable, restricted policy, default provider and settings/associations are unchanged. There is no printing, installation or package change.

## Native scope and visual evidence

The native audit reports client sizes **1264×761, 1200×697 and 1136×697**, in host logical coordinates—not assumed screenshot pixels. At each sampled state it verifies owned direct-child identities, label/field separation, selected-value width, client containment and nonoverlap of the Processed title/buttons/search/history. It runs after the raw screenshot and before any diagnostic repaint.

Text-width checking uses the application's fixed Segoe UI 16-pixel font contract and appropriate padding. It is not OCR or semantic recognition of rendered text. Separate screenshot inspection confirms the actual full wording at the tested settings. No original capture, process-ownership, source-identity or file-safety assertion was relaxed.

Nine individually reviewed GREEN screenshots cover first page, second page, 100%, Fit Width, smaller, narrower, restored, repeated-restored, and text after tracked Rename/Move. Full filter/form labels and the Processed counts **0 and 2** are readable. The PDF remains inside the pane; restoring the window leaves a clean layout, and host zoom controls return for text. The magnified 100%/Fit Width frames show a centered colored document region; those frames establish host-label containment, not fresh semantic PDF-text acceptance. The separate readable-scrolling regression passed, but its new images were not individually reviewed.

Other first-run captures, all repeat/regression images and content crops were not separately inspected. Raw `visual_reviewed=false` flags were retained; the separate evidence file lists inspected-image hashes. Desktop-edge screenshots remain on Lenovo.

Two new pure layout parent tests cover width-dependent wrapping, geometry separation, column bounds and preserved bottom alignment. Their width matrix spans 760–1920 logical units and Processed widths 200–320. **Only the three native sizes above were visually qualified.** The one-row branch has unit coverage, not a new native wide-screen screenshot. Physical monitor/scaling changes, different fonts/languages, very large history counts and arbitrary dimensions are not certified here.

## Fresh results at 2cafe818

| Selection | Result |
|---|---|
| Complete Windows application suite | **379 passed, 0 failed, 1 skipped**, 380 parents; exit 0, no timeout |
| Compact labels, first run | **24 behavior records and 20 label audits passed** |
| Unchanged compact-label repeat | **24 behavior records and 20 label audits passed** |
| Navigation/zoom/resize timing | **23 records plus 12 timing bursts passed**; all 24 raw/natural sampled paint comparisons zero |
| Startup responsiveness | Two runs, four openings each; **16.4–61.6 ms** maximum queued acknowledgements, all below the unchanged provisional 250 ms target |
| Filters / sorting / selection | Pass: **74 action/readback records / 9 checks / 16 records** |
| PDF Undo / readable scrolling | **9 / 14 checks passed** |
| Text / image / launch | **17 / 23 / 12 checks passed** |

Two new parent tests explain 377→379; nested cases, audits, repeats and UI records are not added to Go totals. The sole skip is still `TestRealGeneratedPPTXIntegration`, not a pass. Filter records include actions/readbacks and are not 74 distinct features. All scheduled regression commands completed with exit 0 on the exact source/build.

Startup ready-status observations were approximately 1.47–1.48 seconds for first opens and 0.36–0.40 seconds for repeats. These shared-machine observations preserve the earlier responsiveness correction; they are not a new performance change, a cold-boot benchmark or an exact first-pixel measurement. The performance target and startup harness were unchanged.

The focused runs retain tracked Rename/Move and byte-preservation checks; the Undo regression separately verifies restoration and journal outcomes. Preview-only regressions retain their own nonmutation checks. Private viewer cleanup succeeds and source-integrity checks remain clean. Only disposable files and test-private settings/runtime copies were used. User documents, original runtime, installed working copy and unrelated applications were not changed.

## Retained failures

At test-only source `9162bb0c`, the first attempt stopped at the visibility guard because the sampled point belonged to a `MozillaWindowClass` window, not QuickSort. No browser window was modified and no screenshot guard was bypassed. An unchanged retry reached the new label assertion and reported the seven real clipping findings. Both attempts closed their owned host normally during cleanup (exit 0). Neither is counted as a passing run.

- Visibility guard: `dcm-quicksort-ui-sumatra-20261113-053510-efa3a574`.
- Clipping RED: `dcm-quicksort-ui-sumatra-20261113-053600-eb508067`.

The failing screenshot was reviewed before the product correction. The pure layout tests and Windows cross-compilation were run locally; the exact main-file patch was replayed with input/output hashes and checked in GitHub before Lenovo validation. No source files were edited remotely.

## Exact build and reports

All report directories are under `D:\FlowverseReportsRepo`.

- Reviewed compact run: `dcm-quicksort-ui-sumatra-20261113-054106-fcd91764`.
- Compact repeat: `dcm-quicksort-ui-sumatra-20261113-054251-7d7c4d3c`.
- Resize regression: `dcm-quicksort-ui-sumatra-20261113-054340-87572ecb`.
- Baseline: `dcm-quicksort-baseline-20261113-053940-7203938f`.
- Build: `dcm-quicksort-build-20261113-054033-794d1c59\receipt.json`.

Build receipt SHA256: `9b3f0838c563d3d22490dda8e570e4c178ab08abdc0b47a09c0dd7f063fe605d`.
App SHA256: `9298cc004fbfc3029ad7385704702e305d6b413ccf4be8ffe124265de19763e0`, 7,336,960 bytes.
403-file source manifest: `1f2c2d0b5b9030d1ef421c8cbdb1f9f7a5d9bfd978e524d55c8abaaa64bddec3`.
Full receipt, screenshot and regression identities: `evidence/compact-labels-summary.json`.

## DCM-28/29 audit and next action

The optional Sumatra provider has extensive source-bound page, boundary, zoom, scrolling and lifecycle evidence. However **Windows.Data.Pdf remains the default**, and the original `DCM-PDF-01` manual-zoom visual issue concerns that provider. The current candidate's default-provider page/zoom workflow was not exercised in this task. Optional-provider success must not silently certify the default path.

**Next: run the default-provider multipage navigation and manual-zoom/fit workflow on the current candidate, verify actual displayed content, and close DCM-28/29 only to the scope that evidence supports.** Do not repeat startup profiling or reopen this bounded label fix merely to produce more passes. The final DCM-46 image review must still confirm the labels on the final release candidate.

No numbered step was closed here: DCM-01–18 and DCM-20–27 retain completion; DCM-19 remains blocked; DCM-28/29 pending. Runtime distribution/installer, error-banner presentation, external PPTX, remaining operation/path variants and actual user acceptance remain separate. Sumatra stays optional; PR #1 remains draft, with no merge or publication.
