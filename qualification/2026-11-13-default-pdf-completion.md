# DCM-28 and DCM-29 — default PDF navigation and zoom verified

13 November 2026 · DCM QuickSort v1.0.71 · branch `release/v1.0.71`.
Native-tested source: `9a2ed85a38e25c3ad9698ee41ec993379dff9686`.

## Result and acceptance decision

**DCM-28 (PDF page navigation) and DCM-29 (PDF zoom and fit) are complete for the demonstrated two-page workflow.** The enhanced default-provider run passed all **19 recorded action/readback checks**. Ten full-window screenshots from that run were individually inspected. Both displayed pages, page boundaries, manual 100%/110% zoom, return to 100%, Fit Page and Fit Width were verified using actual content rather than status text alone.

**One enhanced run completed successfully. Two later unchanged repeat attempts stopped at screenshot visibility guards.** They are retained as incomplete/failed runs, not counted as passes or evidence of repeatability. This bounded closure relies on the successful source-bound workflow and reviewed images, not a claim that every attempt passed. The original, less extensive default-provider test also passed earlier in this task.

No application code or PDF-rendering implementation changed. Only the acceptance harness was extended. The built application is byte-identical to the previously qualified compact-layout build. Windows.Data.Pdf remains the default; Sumatra remains optional.

## What was verified

| Workflow | Observed result |
|---|---|
| First page | Teal page-one content and matching page label; Previous disabled and Next enabled. |
| Fit Width | Readable page-one headings appear; returning to Fit Width restores the same content. |
| Manual 100% and 110% | Real page content remains visible; owned native drag messages reveal an intact readable title. Returning to 100% reduces the displayed heading size again. |
| Fit Page | The complete teal content panel and its three headings are visible in the tested viewport. |
| Next and last page | The distinct orange page-two content and headings appear; Next is disabled and Previous enabled. |
| Previous | Page one returns with its correct content and first-page button state. |
| File access and shutdown | Exclusive access to the source PDF succeeds after rendering; no file changes or operation journal entries occur; normal host exit and owned-process cleanup pass. |

The first/last boundary checks read back the disabled navigation controls on the actual displayed pages. They do not force commands through disabled buttons. Navigation and zoom use FlowUIVision/native control actions; panning uses native messages to the identified preview window, not physical mouse-wheel/touchpad acceptance.

At centered 100% zoom, a large coloured portion initially fills the viewport and the heading is above its edge. This is **not a blank render**. Panning reveals the whole `DCM PDF PREVIEW` title at 100% and 110%. Not every document line fits simultaneously at those magnifications. The automatic checks measure page colours and state; independent screenshot review supplies the readable-text observation. No OCR or footer-only visual pass was used.

## Original DCM-PDF-01 finding

The historical November 3 v1.0.69 blank 100%/110% captures are preserved. That blank condition was **not reproduced on the current executable**. Its historical rendering-versus-capture/focus cause remains unknown; this task does not retrospectively claim a root-cause fix. DCM-PDF-01 is recorded as **tested/current visual qualification complete**, not as newly repaired application code.

The previous missing-default-provider acceptance gap is now addressed directly. Earlier Sumatra page/zoom/scroll/lifecycle evidence remains provider-specific. The fresh optional-provider regression is a separate result, not a replacement for the default-provider run.

## Test-only changes

The existing FlowDev `ui-pdf` selector now explicitly removes `DCM_PDF_PROVIDER` and `DCM_SUMATRA_EXE` from its child environment. It records the default provider, checks the exact visible page label and first/last button states, captures both full windows and content crops, rejects blank/wrong-page colours, adds Fit Page and a readable manual-zoom pan, and checks exclusive PDF access and an empty journal.

The original two-page fixture, ownership guards, readiness deadlines and capture safeguards were retained. No new PDF fixture or engine was substituted. The original capture helper already restores the owned window before capture; that behaviour was not added to conceal a failure. All source/test work was authored locally, stored in GitHub and run through FlowDev on Lenovo. No remote source edits.

The 19 records comprise eight action-script records and eleven independent content/state/safety records; they are not 19 different product features. The application-unit count is separate. Local focused PDF tests and exact patch replay passed before native testing.

## Fresh results and repeat limitation

| Validation | Result |
|---|---|
| Enhanced default-provider workflow | **19 records passed once**, normal exit 0; ten full-window images reviewed. |
| First repeat | Incomplete: a sampled screen point belonged to `XamlExplorerHostIslandWindow`, not QuickSort. |
| Unchanged repeat | Incomplete: screenshot guard reported a minimized/hidden window. |
| Complete Windows baseline | **379 passed, 0 failed, 1 skipped**, 380 parent results; no timeout. |
| Optional Sumatra regression | **23 records passed** on the same source. New images not separately reviewed. |

The skipped test remains `TestRealGeneratedPPTXIntegration`, not a pass. No new Go parent tests were added, and no application test counts are increased by UI records or repeats. The last combined validation command exited 1 because the default-PDF repeat failed its capture guard, even though the subsequent baseline and Sumatra selections passed.

Both failed repeat hosts closed normally with exit 0, without forced termination. Their final success-only source-integrity flags are unset; no such flags are inferred for those incomplete runs. The exact cause of the popup/minimization was not established. No unrelated window was modified, and no visibility guard was bypassed or relaxed. The two failed attempts remain a limitation of repeat-run evidence; a later final-candidate review must not describe them as successful.

The successful run preserved all four disposable source-file names and byte hashes, left the destination and operation journal empty, and released the PDF. Its owned-process cleanup and source-integrity checks passed. User documents, installed QuickSort, original runtime, file associations and unrelated apps were not changed.

## Visual review scope

Ten source-bound full-window images were individually opened: first boundary, Fit Width, centered 100%, panned 100%, 110%, panned 110%, return 100%, Fit Page, page-two boundary and restored first boundary. The distinct first/second pages and readable headings agree with the fixture.

The run contains 24 captures total. Content crops are automatically checked but not individually inspected. The launch/final full-window images, repeat-run captures and new optional-Sumatra images were not separately reviewed. Raw `visual_reviewed=false` fields remain untouched; separate review identities are in `evidence/default-pdf-completion-summary.json`. Full desktop-edge images remain on Lenovo rather than in the public repository.

## Exact source and build identity

Build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261113-063835-3b1e3377\receipt.json`.

- Build receipt SHA-256: `1f56aae3cba92f7757de3117a2462d869f910577ebb48efde07d8d8ebfb24bdd`.
- Application SHA-256: `9298cc004fbfc3029ad7385704702e305d6b413ccf4be8ffe124265de19763e0`, 7,336,960 bytes.
- 409-file source manifest SHA-256: `8e0091286316d0af68b3e72f3013e5602891a2d188c99644a056491bb4874113`.
- Enhanced test script SHA-256: `239d9a13476c60210187a35dc13ba2e00618647edbf91181e45f65bf163f49a7`.

Successful enhanced run: `dcm-quicksort-ui-pdf-20261113-063940-e3af9283`.
Retained repeat failures: `dcm-quicksort-ui-pdf-20261113-064146-02ff0a50` and `dcm-quicksort-ui-pdf-20261113-064608-8299c122`.
Initial unextended run at72997e46: `dcm-quicksort-ui-pdf-20261113-063047-75bdf0c4`.
All native reports are under `D:\FlowverseReportsRepo`; further receipt/image identities are in the evidence JSON.

## Boundaries and next action

Only DCM-28 and DCM-29 are newly closed. Completed numbered steps are now **DCM-01–18 and DCM-20–29**: 28 total, including five setup/diagnostic steps—not a product-completion percentage. DCM-19 stays blocked. Human acceptance stays `not_recorded`; no installer/package, physical-display-scaling, arbitrary-document/password/corruption, long-duration, full-input or release certification is inferred from this two-page qualification.

The optional runtime distribution, damaged-file banner, external PPTX, remaining operation/path variants and final-candidate review retain their own gates. No default switch, installation, association change, printing, merge or publication occurred.

**Next: DCM-30 — verify a representative DOCX's actual visible content through the existing installed-handler/fallback path, documenting layout limitations.** Do not restart this PDF profiling chain merely to generate additional green receipts. Rebuild after metadata/source changes and retain each historical result's original identity.
