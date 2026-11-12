# QuickSort — resize corrections verified; optional PDF qualification continues

Latest work: [Sumatra resize corrections](2026-11-12-sumatra-resize.md).
Tested source: `24ff2f243db4c5d8d29b8d64ce81b07c11112efe`.
Completed numbered steps remain DCM-01–18 and DCM-20–27. DCM-19 blocked; DCM-28/29 pending. No numbered step closed here. Human acceptance not recorded.

## Verified state

Two native ui-sumatra runs passed 23 checks each at the same source/build. All 20 unassisted full-window images from the first run reviewed. Its 19 content crops/four explicit-repaint diagnostic references and the second run's images were not separately visually reviewed. Full native baseline 367 passed / 0 failed / 1 retained external-PPTX skip (368 parents); text17/image23/launch12 commands pass. Their new images were not separately reviewed.

DCM-UI-RESIZE-01 fixed at tested sizes: before correction, raw restored screenshots differed from explicit repaint references by 2426/2171/2426 sampled pixels. After correction, all four comparisons are zero across 107276 sampled host points, repeated twice. Product queues invalidate/erase/all-children after layout; diagnostic reference repaint still occurs AFTER unassisted captures. Never count a reference-repaired screenshot as product proof.

DCM-SUMATRA-UI-02 fixed at the three tested outer host sizes 1280x820, 1216x756, 1152x756: bounded side-panel widths preserve the viewer toolbar including Aa. Original widths return after viewer release. Runtime, strict policy and fixed-window style unchanged. This is not a new user-resize feature or all-DPI qualification.

Keep DCM-UI-LABEL-01 open: smaller filter Size/Sort controls overlap and the narrowed Processed header is further compressed. Do not call the whole UI visually approved. MSAA button rectangles underreported earlier clipping; the geometry assertion is supplementary, not a substitute for actual screenshots. Improve coordinate conversion before trusting it as the sole overflow detector.

## Next exact work

Continue the optional provider lifecycle: rapid PDF/text/PDF selection, cancellation/error recovery, release failure refusal, Undo, and closing while a PDF is still active. Keep scroll-to-readable-text at100%/FitWidth and general label layout open. Do not switch the default or close DCM-28/29 merely because two ordinary runs pass. Packaging/distribution and user acceptance remain separate. No unrelated LibreOffice work, printing or association/registry modifications.

## Workflow

Code locally -> GitHub -> FlowDev update/build/test on Lenovo, serially. Canonical workspace `D:\FlowverseReportsRepo`; candidate `D:\FlowverseReportsRepo\dcm-quicksort-candidate`; cwd profile `tools\flowdev-dcm`; runner `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`. Device `d18227f2-0b54-4b11-9026-56ac40305f02`. No remote source edits; leave unrelated interactive LibreOffice session untouched.

`ui-sumatra` requires a fresh exact DCM_BUILD_RECEIPT and DCM_SUMATRA_SOURCE_EXE=`D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe`; it creates its own private runtime/policy copy. Sumatra remains opt-in, old Windows.Data.Pdf default. Runtime SHA719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0; policy SHA f82ee9c3ff09da529f67f2ee73ab716dd53678d0f52b00ddcffc9d6dc0a908aa.

Last build `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-081855-7f0de675\receipt.json` matches24ff2f24 only. App SHA a2bd4063a063431b8267a0ff50ed9157caea74ef3b965b391a6adc08d393a463. Rebuild after metadata/source changes; never rewrite source identities in old receipts.

Reports under D:\FlowverseReportsRepo: reviewed GREEN `dcm-quicksort-ui-sumatra-20261112-081912-e85ac86f`; repeat `dcm-quicksort-ui-sumatra-20261112-082224-ce406ad4`; baseline `dcm-quicksort-baseline-20261112-082052-d55bc2d2`. Separate review/hashes: `evidence/sumatra-resize-native-summary.json`.

Preserve RED `dcm-quicksort-ui-sumatra-20261112-081214-0ceefaba` at1b5cadca and the earlier insufficient-detector PASS atb0201f9b. Its preceding geometry failure080144-ddb5f6ae remains unexplained; do not invent a cause. Local undefined-function compilation failure is not native behavior evidence. All older issues/evidence remain intact. Keep DCM-19, DCM_REAL_PPTX, full operations/Undo, runtime distribution, installer/package and manual acceptance open. PR #1 stays draft; no merge/publication.
