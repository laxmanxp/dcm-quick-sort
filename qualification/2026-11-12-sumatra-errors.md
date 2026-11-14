# Sumatra PDF errors — two defects corrected; repeat UI validation incomplete

12 November 2026 · DCM QuickSort v1.0.71 · `release/v1.0.71`.
Product/native-tested source: `39af1659c01cbfa6bdb38106370197001027e946`.

## Result and acceptance boundary

The new source-bound `ui-sumatra-errors` workflow completed **12 checks successfully once**, with normal host exit 0, unchanged source and unchanged fixture bytes. **All ten captures from that passing run were individually inspected: six full-host windows, one password dialog and three document-content crops.** The full native application baseline passed **369 tests, zero failures and one retained skip** (370 parent results).

The later unchanged error repeat and the navigation/lifecycle regression attempts stopped at the unchanged occluded/offscreen screenshot guard. They are **failed/incomplete runs, not passing regressions**. Their exact obstruction cause is not established. No capture guard was disabled and no raw receipt was rewritten. This checkpoint records the successful bounded test and the remaining repeat/regression requirement separately.

**No numbered step is newly closed.** DCM-01–18 and DCM-20–27 retain their recorded acceptance, DCM-19 stays blocked and DCM-28/29 remain pending. Sumatra stays optional and Windows.Data.Pdf remains the default. No runtime packaging, installed-copy change, association change, printing, merge or publication occurred.

## Two real product defects

### Damaged file advertised as ready — DCM-SUMATRA-DOCUMENT-STATE-01

The test file has a real `%PDF-1.7` header but a missing page tree/trailer; it reaches Sumatra's decoder rather than failing the host's header check. At the pre-fix source, Sumatra displayed an **Error loading** banner and had page field `0`, while QuickSort still advertised PDF navigation/zoom as ready. The native error control was read and the failure screenshot inspected.

The host now distinguishes attachment from document readiness. It reads only the pinned owned viewer's numeric page control, with a bounded 30-ms text query and fixed-size buffer. Unknown state is reported as checking the document; an empty/zero page field reports **no PDF page loaded** and directs the user to the viewer's message or another file. Non-numeric page labels are not rejected merely for being non-numeric. This is a pinned-runtime UI-state check, not a new PDF parser or a general guarantee of rendering correctness.

The passing damaged-file screenshot shows the actual decoder error together with the corrected host status. Selecting text releases the failed viewer and restores the exact expected text; all PDF fixtures then permit exclusive access.

### Password prompt killed by startup deadline — DCM-SUMATRA-PASSWORD-01

Before correction, a visible password dialog was still treated as an unattached/stuck viewer. After a 17-second test hold, it disappeared and QuickSort reported its 15-second attachment deadline had expired. The preserved pre-fix receipt records `prompt_remains=false` and that exact host error.

The host now recognizes a visible password dialog belonging to its own recorded viewer process and owned by that process's Sumatra frame. While that prompt exists, it preserves the prompt and reports **password required**; the normal bounded attachment deadline resumes after the prompt disappears. The host checks window classes/styles only for the password field; it does not read or collect its text.

In the passing run the prompt survives the 17-second hold, Cancel permits clean text recovery, an explicitly wrong fixture password is refused, and the correct known fixture password opens the expected orange page. The Remember checkbox was not selected. This is known-password testing of a generated document, not password cracking or handling user credentials.

## What the 12 records establish

| Scenario | Evidence |
|---|---|
| Text with missing runtime | Text remains readable when only the disposable runtime copy has been renamed out of place. |
| PDF with missing runtime | A missing-file error identifies the intended runtime; no viewer starts. |
| Recovery after missing runtime | Text returns; restoring that same private executable and selecting the PDF shows its original teal content. |
| Header-valid damaged PDF | Actual native decoder error plus truthful no-page host state; text recovery releases the document/viewer. |
| Password wait/cancel | Owned prompt remains beyond the old deadline; Cancel followed by text selection restores the correct preview and releases files. |
| Wrong/correct password | Wrong value retains the prompt without an attached document; correct value renders the distinct encrypted fixture's actual title/artwork. |
| Decrypted document release | Switching to text releases the encrypted PDF. |
| Final recovery and close | Original PDF renders again; host closes normally, recorded viewers/session profiles are cleaned up, all PDF files permit exclusive access. |

The six source fixtures retain their exact names and SHA-256 values. The destination and operation journal stay empty. Runtime and policy hashes match their original values at the end. Only a private test runtime was temporarily renamed; the prototype/runtime installation and user documents were untouched.

The encrypted one-page fixture was generated locally with ReportLab StandardEncryption 128-bit, then independently checked with PyMuPDF: it requires a password, rejects the known wrong password, accepts the correct fixture password and renders the expected text. Its bytes are pinned in `tests/fixtures/sumatra-password.json`, SHA-256 `cd167e6404f23dec6015d2897a9f7e462d0fec82f046c35f59afc7e9e774c24c`. The displayed decrypted content was inspected both locally and in the native viewer. This does not qualify every encryption variant, every damaged PDF, huge files or malformed-document security.

## Fresh results

All final selections below use `39af1659`; report directories are under `D:\FlowverseReportsRepo`.

| Selection | Result | Report directory |
|---|---|---|
| Full native baseline | **369 passed / 0 failed / 1 skipped**, exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261112-141022-88e2d46f` |
| Reviewed error workflow | **12 passed**, exit 0, ten captures reviewed | `dcm-quicksort-ui-sumatra-errors-20261112-140757-e0408a55` |
| Navigation/zoom/resize regression | Failed at visibility capture guard | `dcm-quicksort-ui-sumatra-20261112-141100-f448d0d0` |
| Unchanged navigation retry | Failed at visibility capture guard | `dcm-quicksort-ui-sumatra-20261112-141436-4aaab0a1` |
| Lifecycle regression | Failed at visibility capture guard; one earlier record passed | `dcm-quicksort-ui-sumatra-lifecycle-20261112-141445-386c1e93` |
| Unchanged error-workflow repeat | Failed at visibility capture guard; one earlier record passed | `dcm-quicksort-ui-sumatra-errors-20261112-141456-2c5460b9` |

The skipped parent remains `TestRealGeneratedPPTXIntegration`, not a pass. One new document-status parent with nine table entries explains the baseline increase from 368 to 369; the table entries and UI records are not added to the parent count. Focused local `TestSumatra` tests and Windows test cross-compilation passed. Earlier Linux undefined-helper compilation was a local test-development failure, not a Windows product reproduction.

## Preserved pre-fix evidence and test adjustments

- `daebe1505`, `dcm-quicksort-ui-sumatra-errors-20261112-135630-03135ea5`: missing-runtime recovery passed and damaged-file false readiness was recorded; password capture then failed as occluded/offscreen. Cleanup exit 0. The damaged-file image was inspected.
- `43b7cbd3`, `dcm-quicksort-ui-sumatra-errors-20261112-140251-caf2bc69`: both damaged-file false readiness and password-prompt timeout were recorded before a later capture failure. Cleanup exit 0. The earlier harness's cancel-labelled record did not establish actual Cancel after the prompt had timed out; the final harness requires the prompt to survive before recording Cancel acceptance.

The screenshot helper makes captured windows topmost. The test was corrected to remove only its host's topmost state before the owned password prompt and to restore that exact modal for capture. This is a scoped test-window correction, not a relaxation of visibility or process-ownership assertions. Later capture failures remain separate and unexplained; no successful result is inferred from them.

## Visual limits and next work

A separate presentation issue remains: the missing-runtime fallback includes a long technical path that clips in the pane/status bar, and the native damaged-file error banner also truncates the long fixture path. The error category/recovery guidance is visible, but this is not full error-layout approval. Record **DCM-SUMATRA-ERROR-COPY-01**, pending concise readable missing-runtime wording/path presentation under DCM-46. Existing Size/Sort and Processed-label layout findings remain open.

Raw screenshot review flags are unchanged. The separate evidence file records these ten inspections and their hashes; desktop-edge images remain on Lenovo rather than being published. Native input/known-fixture password entry is automated, not human acceptance. The test verifies Cancel followed by another selection, not all modal-focus/long-wait/operating-system shutdown conditions. The runtime/policy pin is unchanged and restricted use is not claimed to be an OS security sandbox.

Next: complete the visibility-safe unchanged error/navigation/lifecycle reruns, fix concise error presentation, then finish startup responsiveness and readable text after scrolling at 100%/Fit Width. Retain DCM-19, external PPTX, remaining operations/path cases, general layout, runtime distribution/installer and manual acceptance. The current build receipt matches only the tested source; rebuild after subsequent metadata/source commits.

## Exact identities

Build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-140740-e5a0bb21\receipt.json`.
Receipt SHA-256: `36eef2fd816869efc904c1c8d2cacd4fd078c1bc6fd83c3911a7222be2d86854`.
App SHA-256: `756beb686fb196c608e415f3b366cad93ec09af6cf32fa631a8f56a0750f0ef8`; **7,240,704 bytes**.
327-file source manifest: `a914e7ac1f06076acd547d5c3ec41879374b432e9cc620f7c03ad8b2d6a2216f`.
Runtime SHA-256: `719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0`.
Policy SHA-256: `f82ee9c3ff09da529f67f2ee73ab716dd53678d0f52b00ddcffc9d6dc0a908aa`.
Evidence/review: `qualification/evidence/sumatra-errors-native-summary.json`.
