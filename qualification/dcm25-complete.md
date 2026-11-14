# DCM-25 complete — Refresh, source switching and stale-target safety

11 November 2026 · DCM QuickSort v1.0.71 · branch `release/v1.0.71`.
Product fix and native-tested revision: `1554c9b87bbb834b14d54b231eae4b4ece27b8d2`.

## Result

**The complete native ui-refresh workflow passed all 23 recorded checks. All nine screenshots were individually opened and inspected.** The 23 records comprise 18 state/integrity checks, four actual folder-picker selections and one FlowUIVision setup result. The application closed normally with exit code 0, owned children drained, and the source remained unchanged.

Only DCM-25 is newly closed. Completed: DCM-01–18 and DCM-20–25 (24 checklist steps, including five setup/diagnostic steps, not a product-completion percentage). DCM-19 remains blocked and human user acceptance remains unrecorded.

## Reproduced product defect and correction

The focused native run at `03c9be04` temporarily moved its disposable source directory out of the configured path. Reload reported a source-read error, but left the previous list, selected row, filename draft and readable preview visible. The assertion `DCM25_UNAVAILABLE_SOURCE_STALE_TARGET` and the inspected failure screenshot independently agree. The test stopped at that assertion; no wrong-file operation or data loss after source unavailability is claimed.

The correction is confined to the failed-read branch of `reloadFilesWithFocus` in `app-src/main_windows.go`: invalidate the cached files/current index/startup preference, rebuild the empty list, unload the old preview and clear its draft through existing rendering logic, then show **Source unavailable** with the read error and restore/choose-folder/Reload guidance. It does not silently present an unreadable folder as successfully empty. Restoring the fixture directory and Reload recovers the correct list, draft and readable preview.

Code was authored locally, Windows-test-cross-compiled, stored in GitHub and applied with exact input-blob/output-hash guards before FlowDev update/test on Lenovo. No remote source editing, installer change, new preview engine or new Go parent test. The new automated regression is the native `ui-refresh` harness.

## Verified behavior

| Scenario | Result |
|---|---|
| Reload with an uncommitted filename draft | Original selected full path retained; the draft is reset to that file's actual name. |
| Reload after multi-selection | A valid single current file remains; the old multi-file selection does not survive as a stale operation set. |
| Controlled addition/removal of fixtures | Newly added file appears; a removed selected file and its draft/preview no longer remain active. |
| Cancel Browse | Original source and edited draft retained; no file operation occurs. |
| Switch A to B using actual Browse | Same basename `02-shared.txt` displays B-specific contents, not A's preview or stale draft. |
| Switch to an empty folder | List, selection and draft clear. Rename/Move/Next remain harmless with a valid disposable destination. |
| Rename and Move in B | Only B's `02-shared.txt` becomes `B-renamed.txt` and moves to Accepted. A's same-named file remains unchanged. Explicit Reload verifies the post-operation view. |
| Source becomes unavailable | No cached list/selection/draft/preview remains; truthful error guidance appears; Rename/Move do not mutate fixtures. |
| Source restored, then Reload | Correct three-file A list and matching readable preview return. |

The fixture harness intentionally adds `00-added.txt`, removes its own `03-to-remove.txt`, and moves/restores its own source directory to simulate external changes. These are declared **test setup mutations**, not application data loss. The final five expected files all match their expected hashes, including the B file moved by the application. No user documents or host settings were used.

## Fresh results and exact evidence

All final runs use the same product revision `1554c9b8`:

| Selection | Result | Directory under `C:\FlowverseReports` |
|---|---|---|
| Full app baseline | **358 passed, 0 failed, 1 skipped** across 359 Go results; exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261111-132539-4f1de17d` |
| Refresh and switching | 23 successful records; nine reviewed images; normal close | `dcm-quicksort-ui-refresh-20261111-132617-5e4dccbb` |
| Ctrl/Shift selection regression | FlowDev PASS, exit 0 | `dcm-quicksort-ui-selection-20261111-132717-40bc10d4` |
| Launch regression | FlowDev PASS, exit 0 | `dcm-quicksort-ui-launch-20261111-132739-cf097855` |

The regression commands' new screenshots were not separately reviewed here; their PASS results are not new visual acceptance. UI records are not added to the Go test count. The sole skip remains `TestRealGeneratedPPTXIntegration`, requiring its identified `DCM_REAL_PPTX` fixture before DCM-32/47 acceptance. Installer/package suites were not rerun.

Exact build: `C:\FlowverseReports\dcm-quicksort-build-20261111-132602-da7ecfec\receipt.json`.
Build-receipt SHA-256: `53a3d6d3fe41ead807006a8f46f08bf51dfe4a35203f37b6d7e3decd57ccc0a7`.
App SHA-256: `b6353d846a8c58c167032dc06a873e15e48273dbdbd68e67758877bd8b364466`.
Separate image review/hashes: `qualification/evidence/dcm-25-native-summary.json`.

## Preserved failures and test-tool corrections

The first new-harness attempt (`20261111-131534-30dfe6d4`) lacked initial FlowUIVision inspection and stopped in setup. A later harness assumption (`20261111-131738-6fd3c7d0`) incorrectly expected automatic reselection after moving the last row. That existing no-current state was retained; an independent filesystem check and explicit Reload now verify recovery without requiring an unrelated behavior change. Another attempt (`20261111-131932-722d40b4`) selected a fixture before the returned folder/list finished updating; the harness now waits for the exact source path and complete list. These are retained test-tool failures, not claimed application defects or passes.

The actual product RED is `dcm-quicksort-ui-refresh-20261111-132238-b590b5c8`. Its stale-target assertion is unchanged in the successful run. Hash, ownership, file-safety and screenshot guards were not removed.

## Visual review and continuation

Nine screenshots show correct folder-specific contents, reset drafts, empty-source behavior, representative Rename/Move results, the unavailable-source message, and successful recovery. Raw `visual_reviewed=false` fields remain untouched; the separate review record records the inspection. Raw captures stay on Lenovo because a small desktop edge is visible. Existing clipping/Processed layout issues remain open under `DCM-UI-LABEL-01`.

Next: **DCM-26 — representative supported text previews and truthful limit/encoding/failure states**. Do not count this step's TXT previews as completion of that separate matrix. Multi-selection setup here uses the existing native adapter; DCM-24's physical-input evidence remains separate. This step does not close general Rename/Move/Undo, unavailable network drives, all permission-error cases, final package/setup, or human acceptance.

Preserve DCM-19 isolation, external PPTX, remaining workflow/preview and package gates. Refresh the source-bound build after any later source/metadata commit. Leave DCM21_PROBE_EMPTY_DESTINATION unset during normal acceptance. Keep PR #1 draft; no merge or publication.
