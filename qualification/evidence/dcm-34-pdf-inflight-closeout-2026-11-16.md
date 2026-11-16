# DCM-34 — observed PDF in-flight interruption closeout (16 November 2026)

## Decision

**PASS with bounded acceptance for DCM-34-PDF-INTERRUPT-01**, the representative normal-app valid/error PDF renderer interruption, late-result rejection, and same-file reopening scenario on Lenovo. The overall numbered **DCM-34 remains PARTIAL**: broader Office damage/protection, separate PDF bitmap-size observation, human acceptance, and release gates remain open. Numbered plan remains **32 complete / 18 remaining**.

Tested source: `ed75287c58b8f0f31b4171408593cc2f9a233a7a` on `release/v1.0.71`. This is a documentation-only descendant of code `be098518e706d079f014878ae1d6eaab49219d49`. The FlowDev update synchronized the clean checkout before native tests. No production source was edited on Lenovo; no new product fix is claimed.

## Required FlowDev evidence

All selectors **PASS** at exact tested source `ed75287`:

- `update dcm-quicksort`
- `test dcm-quicksort pdf-trace-unit`
- `test dcm-quicksort pdf-interruption-unit`
- `test dcm-quicksort ui-tooling`
- `test dcm-quicksort build`
- `test dcm-quicksort ui-pdf-inflight`
- Trace-disabled `test dcm-quicksort ui-pdf`
- Trace-disabled `test dcm-quicksort ui-pdf-failure`
- Trace-disabled `test dcm-quicksort ui-pdf-protected`

Exact successful build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261116-055754-5dcc434e\receipt.json`.

Retained headed in-flight receipt: `D:\FlowverseReportsRepo\dcm-quicksort-ui-pdf-failure-20261116-055851-e3135689\receipt.json`. The native receipt has `passed=true`, with `feature_accepted=false`, `visual_review_required=true`, `user_test=not_recorded` intentionally unmodified.

## Worker and UI observations

The headed harness recorded the real renderer process and owned selection changes. Its trace confirmed these sequence orderings without synthetic completion or imposed renderer delay:

- Valid PDF: start sequence 4, stop 5, finish 6, obsolete result discarded 7.
- Damaged PDF: start 9, stop 10, finish 11, obsolete error discarded 12.
- Same-file reopen: old worker start 14, stop 15, finish 18, old result discarded 19; newer generation 13 remains selected and renders the valid page.

The harness confirmed obsolete output removal, unchanged fixture bytes, empty journal, both PDF files exclusively released, clean source, and host exit 0. These are bounded observed events, **not a claim that the renderer process was forcibly cancelled**.

## Separate screenshot review

All **12 original captures** in the in-flight report were read and visually inspected individually: six host frames and their six corresponding same-frame preview crops. The initial TXT, TXT after valid late completion, TXT after failed late completion, and final TXT states show the same three correct lines without PDF overlay. Two captured attempts of the reopened valid PDF display `DCM PDF PREVIEW`, `PAGE ONE - TEAL`, `VISIBLE DOCUMENT CONTENT`, `Page 1 of 2`, a visible Next button, and usable host preview sizing controls. No unexpected blank or corrupted settled frame was identified. The document/page is visibly clipped vertically at the observed zoom as expected in the scrollable viewport; whole-page visibility is not asserted. Original raw `visual_reviewed=false` flags are historical and not rewritten.

## Limits

This accepts representative TXT switching and same-file reopening with the real Windows PDF renderer, not arbitrary file complexity, physical mouse/keyboard acceptance, password entry/unlocking, full lifecycle cleanup across all formats, a low-latency guarantee, or hard cancellation of the renderer. Preserve the previous failed in-flight external-process-exit observation and its receipts as historical RED; the corrected observer and this later successful headed run supersede its acceptance blocker without deleting history.

Do not reopen XLSX/PPTX qualification on this basis. DCM-34 next should be bounded Office failure/protection coverage, separately resolve or explicitly limit `DCM-PDF-RENDER-SIZE-01`, then assess whether the parent step is closable. Final FileFlow Suite packaging still waits on Flow Search and DeepSearch acceptance.
