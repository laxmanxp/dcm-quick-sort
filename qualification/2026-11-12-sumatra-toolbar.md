# Sumatra toolbar retry — page controls corrected; broader qualification remains open

12 November 2026 IST. DCM QuickSort v1.0.71, `release/v1.0.71`.
Freshly tested source: `c53f85263963886b1038dbd8933dd3883481aabd`.

## Result

The retry recovered the toolbar implementation and expanded tests already saved by the interrupted attempt. No additional application or test implementation was changed in this retry. After a fresh FlowDev update, tooling check and source-bound build, **two consecutive `ui-sumatra` executions passed all 19 checks each** at the same revision. This is 19 distinct checks repeated twice, not 38 different features.

The original page-number clipping and redundant host PDF-sizing controls are corrected in the tested layouts. All **17 full-window captures from the first fresh run** were individually opened and inspected. Its 16 content crops were checked by the automated pixel assertions but were not separately visually inspected. Raw visual-review flags remain unchanged. This does not certify the entire UI: the resize captures expose the separate findings below.

## Observed defect and implemented correction

The preserved RED at `8666752a` (`dcm-quicksort-ui-sumatra-20261112-021413-eff1b1a5`) fails `SUMATRA_TOOLBAR_CLIPPED`. Its layout record places the numeric page field at **Y=-13** inside a toolbar 43 pixels high; the page background starts at Y=-17. This measured out-of-bounds placement explains the top clipping. The fresh GREEN layout places the page field at **Y=8** and the background at Y=5, with all seven inspected Edit/Static children inside their toolbar.

The host-side compatibility correction checks the owned viewer PID, parent and child classes, adopts the toolbar's DPI context, and asynchronously recenters only vertically clipped direct Edit/Static children. It retains child size, horizontal placement, z-order and focus. QuickSort's own five sizing controls hide while Sumatra is attached and return on switching to text. The pinned runtime and exact no-print/no-write policy are unchanged. No Sumatra binary was patched or bundled.

The retained native geometry RED and the already-committed correction are not claimed as newly authored during this retry. One added Go parent test covers toolbar-placement policy; it accounts for the baseline increase from 365 to 366 passes. Existing source, policy and capture assertions were not weakened.

## Fresh verification

| Workflow | Evidence |
|---|---|
| First/last page boundaries | Previous at page one stays on teal page one; Next at page two stays on orange page two. |
| Toolbar Next/Previous | Invokes the actual accessible toolbar action; correct page artwork and text appear. |
| Page-number input | Entering 2 and 1 shows the corresponding actual pages. |
| Zoom In/Out | Actual visible page area increases/decreases; captures show the changed scale. |
| Fit Page, 100%, Fit Width | Content remains rendered and the visible Sumatra mode notification agrees; Fit Page restores the whole page. |
| Resize smaller and restore | Host bounds change from 1280×820 to 1216×756 and back in the measured host coordinate context; the embedded frame remains contained and the page field remains readable. |
| Release, Rename and Move | Internal alternate Open stays blocked; switching to text releases the PDF; representative tracked Rename and Move retain original bytes. |
| Cleanup | Both runs exit normally with host code 0, preserve fixtures/source identity and verify that recorded viewer instances do not survive. |

Toolbar Next/Previous/Zoom/Fit actions use Windows accessibility, page entry uses text plus Enter messages, and boundary/100%/Fit Width checks use allowlisted commands to the verified viewer. These are not physical mouse/keyboard or manual-user acceptance. Only a two-page fixture and two host sizes were used. At 100% and Fit Width the captured viewport shows a magnified colored region of the page; this does not certify scroll-to-text or all-page typography at those zoom levels.

### Results at c53f8526

Reports are under `D:\FlowverseReportsRepo`.

| Selection | Result | Report directory |
|---|---|---|
| Full native application baseline | **366 passed, 0 failed, 1 skipped**, 367 parent results; exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261112-040352-443135c7` |
| Sumatra run 1 | **19 passed**, 17 full images reviewed; normal exit 0 | `dcm-quicksort-ui-sumatra-20261112-040237-691adf62` |
| Sumatra run 2, unchanged source/build | **19 passed**, normal exit 0; images not separately reviewed | `dcm-quicksort-ui-sumatra-20261112-040536-5300276f` |
| Default-provider text regression | **17 passed**, command exit 0; images not separately reviewed | `dcm-quicksort-ui-text-20261112-040435-99bf630e` |
| Default-provider image regression | **23 passed**, command exit 0; images not separately reviewed | `dcm-quicksort-ui-image-20261112-040504-1eb23a4f` |

The skip remains `TestRealGeneratedPPTXIntegration`; no fixture was supplied and it is not a pass. UI cases are separate from Go totals. Installer, old PDF UI and packaged release qualification were not rerun. Two normal runs improve repeatability evidence but are not a rapid-switch stress test.

## New visual findings — not hidden by passing automation

**DCM-SUMATRA-UI-02:** at the smaller host size, the far-right `Aa` toolbar control is clipped. The page field and tested page/zoom controls remain readable. The existing geometry assertion checks Edit/Static children, not every toolbar button; its PASS does not close this overflow issue.

**DCM-UI-RESIZE-01:** after restoring the host size, stale button/footer fragments remain visible around the right-side Browse/Recovery/Reload area and the lower status region. They persist into later captures. This is an observed repaint/layout defect, not a proven rendering-engine cause. General host label crowding also remains under the earlier layout ticket.

The original top-clipping fix is recorded under DCM-SUMATRA-UI-01 with its bounded evidence. The new findings, full-provider lifecycle checks and default/package acceptance remain open. **No numbered step is newly closed. DCM-28/29 remain pending**, with their navigation/zoom evidence updated rather than discarded. No default PDF-provider switch, printing, association/registry modification, merge or publication occurred.

## Identity and continuation

Build: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-040215-6858b471\receipt.json`.
Receipt SHA-256: `94e5052e26ab7f2906d6d4017329bd8bd53512e965b4cd2f35f06892d9ed609f`.
App SHA-256: `2b0f4e8cfe9e54909e63cf327b89dbb448ed92acfa194ea3a12f1e0ccba4ae89`, 7,229,952 bytes.
301-file source manifest: `55825569addc8c733ea658d95b5e9ec3ff52a619164501951f92ec577a336bde`.
Separate review/hashes: `qualification/evidence/sumatra-toolbar-native-summary.json`.

Preserve the clipping RED, the earlier `2fddee40` accessibility-enumeration failure and prior stage-1 failures. The fresh retry introduced no further implementation changes and no new failing native run. The interrupted attempt corrected accessibility child enumeration before this retry; it did not weaken the geometry assertion.

Next: reproduce the host resize repaint trails and narrow-toolbar overflow with focused assertions, correct them without relaxing runtime restrictions, then finish rapid switching, cancellation/error handling, Undo, release-timeout refusal and active-PDF shutdown. Keep Sumatra opt-in and the previous Windows.Data.Pdf default. Retain DCM-19, external PPTX, distribution/installer obligations and manual acceptance. Refresh build identity after later commits.
