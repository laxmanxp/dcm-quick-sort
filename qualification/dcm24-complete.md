# DCM-24 complete — Ctrl/Shift selection and exact operation targets

11 November 2026 · standalone DCM QuickSort v1.0.71 · branch `release/v1.0.71`.
Product fix and native-tested source: `aa6dceffa1485a1867e3748c9dbe8d1ec9ade359`.

## Result

**DCM-24 passed its defined native acceptance boundary.** The selection run contains 16 successful records: 13 behavior assertions and three FlowUIVision setup/filter results. All seven selection screenshots were individually inspected. The actual Ctrl/Shift selection inputs used guarded Windows SendInput modifier/mouse events, not direct LB_SETSEL assignments. This is automated input, not recorded human user acceptance.

Only DCM-24 is newly closed. Completed: DCM-01–18 and DCM-20–24 (23 checklist steps, including five setup/diagnostic steps, not a product percentage). DCM-19 remains blocked.

## Reproduced defect and correction

The first fresh native run at `f208e3ad` failed after selecting Alpha and Charlie, then Ctrl-deselecting Alpha. Charlie remained highlighted, but the draft and visible preview identified Alpha. The saved failure screenshot independently confirms the selected-index assertion. No wrong-file move was attempted in that failing run, and none is claimed reproduced.

The Windows listbox keyboard-focus row can remain on a deselected item. The app previously used that caret as the current file and could automatically reselect the final deselected row during repaint. The correction now derives the current file from the actual selected indices, gives zero selection no current file, and renders native selection changes without resetting Windows' caret or range anchor. Explicit Next, sorting and startup retain their separate synchronization path. Move now consumes the actual selected file set, including the single-file case, and has no stale-current fallback.

New files: `source_selection.go`, `source_selection_windows.go`, and `source_selection_test.go`. One new Go parent test with ten cases covers the selection policy; existing Go and UI assertions remain unchanged. The main wiring replacement was authored locally, verified against exact input/output hashes, applied through GitHub, then updated and tested on Lenovo through FlowDev. No source was edited on Lenovo. No installer, preview engine, or UI framework change.

## Verified workflow

| Scenario | Outcome |
|---|---|
| Ctrl-click separate rows | Both intended files selected; count and draft agree. |
| Ctrl-deselect to one file | The remaining selected file supplies the draft and readable preview. |
| Ctrl-deselect the final file | Zero selection remains zero, with a blank draft and no-current guidance. |
| Forward/reverse Shift range | Exact contiguous visible range selected. |
| Remove a middle item / ordinary click | Ctrl removes only that item; ordinary click collapses the selection. |
| Next from multi-selection | Existing explicit Next becomes single-file navigation to the next file. |
| Shift range under TXT filter | Only the four visible TXT files selected; both CSV files excluded. |
| Cancel Move confirmation | Exact selection and all six file contents remain unchanged; no operation journal created. |
| Confirm two-file Move | Exactly `01-alpha.txt` and `04-delta.txt` move to the disposable destination; the other four remain in the source. All six contents match their original hashes. |

The final screenshot shows the four remaining source files and the two intended Processed records. The application exits normally with code 0, owned children drain, and source identity remains unchanged. This is representative same-volume selected-set coverage, not completion of Batch Move, Undo, conflict, cross-volume, or every other file operation gate.

## Fresh regression evidence

All native results below use `aa6dceff`:

| Selection | Result | Report suffix under `C:\FlowverseReports` |
|---|---|---|
| Full app baseline | 358 passed, 0 failed, 1 skipped; exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261111-124126-52bc061f` |
| Ctrl/Shift selection | 16 records passed; seven screenshots reviewed; normal close | `dcm-quicksort-ui-selection-20261111-124301-63c57d1b` |
| Navigation | FlowDev PASS | `dcm-quicksort-ui-navigation-20261111-124402-832423cc` |
| Sorting retry | FlowDev PASS, unchanged code/guards | `dcm-quicksort-ui-sort-20261111-124546-6419e53e` |
| Launch | FlowDev PASS | `dcm-quicksort-ui-launch-20261111-124601-94d178b7` |
| Filters | FlowDev PASS | `dcm-quicksort-ui-filters-20261111-124612-0456fa14` |

UI records are not added to the 359 Go results. The new single parent test explains the increase from 357 to 358 passing tests. The remaining skip is `TestRealGeneratedPPTXIntegration`; its identified DCM_REAL_PPTX fixture remains required before DCM-32/47. New screenshots from the four regression commands were not separately reviewed here; no new visual approval is inferred from their PASS output.

One initial sorting regression stopped with `Refusing minimized/hidden capture`; retain `dcm-quicksort-ui-sort-20261111-124427-3965ec0c`. The cause of that window state was not established. An unchanged retry passed without weakening the guard. This is not evidence of a sorting-logic failure.

The earlier DCM-24 interruption was checked first: no relevant active run or exact `dec6385a` receipt was found. Its prior outcome remains unknown; this turn ran a new serial sequence. The original selected-row RED is preserved at `dcm-quicksort-ui-selection-20261111-123543-ed7af16e`.

Local focused Go tests and Windows test cross-compilation passed. An exploratory full Linux suite failed Windows-guarded move/recovery tests and is not presented as acceptance; the complete native Windows baseline above is the authoritative result.

## Build and visual evidence

Exact build receipt: `C:\FlowverseReports\dcm-quicksort-build-20261111-124204-86887193\receipt.json`.
Receipt SHA-256: `5dae162d0d1bb1d65187d09db820a8c9cc9c86619bbee79bd95e272b4dfddad1`.
App SHA-256: `15e937e2cb81988a3cfb3a2ee1d4d6e266d2adcecdb992d08f899d4cc890005a`.
Separate review and image hashes: `evidence/dcm-24-native-summary.json`.

Raw visual flags remain unchanged; this report records the separate seven-image inspection. Raw images remain on Lenovo because a small desktop edge is visible. Existing label clipping and the zero-current FILES counter clarity issue remain open under DCM-UI-LABEL-01; functional acceptance is not final visual approval.

## Continuation

Next: **DCM-25 — refresh and folder switching without stale selection, draft or action targets**. Preserve DCM-19/disposable Explorer isolation, remaining preview/file-operation and external-PPTX gates, final package/installer lifecycle and actual user acceptance. Refresh the exact build receipt after later commits. Keep PR #1 draft; nothing merged or published.

Technical reference for caret versus selection semantics: Microsoft Learn, LB_GETCARETINDEX and LB_SETCARETINDEX (https://learn.microsoft.com/en-us/windows/win32/controls/lb-getcaretindex and https://learn.microsoft.com/en-us/windows/win32/controls/lb-setcaretindex).
