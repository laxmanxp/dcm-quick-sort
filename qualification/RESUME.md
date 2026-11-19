# DCM QuickSort — resume 19 November 2026

Latest dedicated acceptance: [Rotation GUI review](2026-11-19-rotation-gui.md) and [structured evidence](evidence/dcm-rotation-native-summary.json).

- **Native-tested code `31394a446f4223b7789d6287761d0fac916df5b3`**: FlowDev tooling/build and `ui-rotation` passed; 13 cases, 13 original full-window screenshots separately reviewed. Image/PDF both directions and multi-page rotation verified; sources and test fixtures unchanged; normal exit.
- Both rotation controls are a **permanent visible UI contract** protected by `TestRotationToolbarIsPermanentAndWired`. TXT and optional Sumatra show them disabled rather than disappearing. A valid Sumatra screenshot was reviewed, but the larger Sumatra regression remains incomplete owing to unrelated popup occlusion.
- Current numbered plan **44/50 complete**: DCM-19 blocked, DCM-34 in progress, DCM-47–50 pending. This focused rotation test does not close another numbered release milestone.
- PR #1 draft/unmerged, no installation or publication. FileFlow Suite final packaging waits for Flow Search and DeepSearch readiness. Human input, resizing/DPI and final-release acceptance remain separate.
- Workflow: GitHub source of truth → isolated Lenovo FlowDev update/build/GUI tests; no source code edits on Lenovo. Tested executable receipts remain bound to `31394a446f4223b7789d6287761d0fac916df5b3`; later commits may be documentation-only and are not relabeled as tested.

---

## Historical checkpoint — 13 November 2026

Latest: [default PDF completion](2026-11-13-default-pdf-completion.md).
Native-tested source `9a2ed85a38e25c3ad9698ee41ec993379dff9686`.

## Acceptance and honest limits

DCM-28 and DCM-29 are newly complete for the demonstrated two-page navigation/zoom workflow. Default Windows.Data.Pdf is now directly qualified, not inferred from optional Sumatra. Enhanced ui-pdf passes19 action/readback records once; ten full-window screenshots reviewed. Next/Previous actual pages, disabled first/last controls,100/110/return100,Fit Page/Fit Width and native pan show content. At centered100% text can be outside the viewport; panning reveals a readable title. Not every line fits at manual zoom.

Two unchanged repeats fail visibility guards: XamlExplorerHostIslandWindow occlusion, then minimized/hidden. Both normal cleanup exit0, no guard bypass or unrelated window changes. They are incomplete, not passes, and successful repeatability is not claimed. Closure is the bounded passing run plus reviewed images; final-candidate/human acceptance stays separate.

No app code changed. App matches compact build:9298cc004fbfc3029ad7385704702e305d6b413ccf4be8ffe124265de19763e0. Old v1.0.69 blank captures are not reproduced; historical root cause unknown, no new product fix claimed. DCM-PDF-01 is tested/current qualification complete, with prior evidence preserved.

Fresh baseline379/0/1 (380 parents; no timeout; externalPPTX skip remains not passed). Optional Sumatra23 regression passes on same source; new images not reviewed. Combined retry command exits1 because ui-pdf capture failed even though baseline/Sumatra passed. Do not call everything green.

## Next exact action

**DCM-30: verify representative DOCX visible content through the existing installed handler or fallback and document layout limitations.** Inspect the current selector/fixtures before testing; do not silently bundle XLSX/PPTX or restart PDF performance work. Preserve final-candidate revalidation, known error-banner, externalPPTX, operation/path variants, runtime distribution/installer and human gates. No new blockers merely to defer completed bounded PDF acceptance.

## Execution and evidence

Local code -> GitHub -> FlowDev update/build/test on Lenovo; no remote source edits. Device `d18227f2-0b54-4b11-9026-56ac40305f02`. Workspace `D:\FlowverseReportsRepo`; candidate `dcm-quicksort-candidate`; profile `tools\flowdev-dcm`; runner `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`. One UI suite at a time; leave unrelated apps alone.

Build `D:\FlowverseReportsRepo\dcm-quicksort-build-20261113-063835-3b1e3377\receipt.json` matches9a2ed85a only. Rebuild after metadata/source changes. FlowUIVision `D:\FlowverseReportsRepo\dcm-flowuivision-20261103\FlowUIVision.exe`.

Reviewed default run `dcm-quicksort-ui-pdf-20261113-063940-e3af9283`; failures064146-02ff0a50 and064608-8299c122. Baseline `dcm-quicksort-baseline-20261113-064704-4847f825`; optional Sumatra `dcm-quicksort-ui-sumatra-20261113-064842-b69cd133`. Exact identities in `evidence/default-pdf-completion-summary.json`. Raw flags/receipts remain unchanged; desktop-edge images stay on Lenovo.

Completed DCM01–18 and20–29 (28 total including5 setup steps);19 remains blocked. Next30. Windows.Data.Pdf remains default, Sumatra optional. PR#1 stays draft; no printing/installation/association/merge/publication. User acceptance not recorded.
