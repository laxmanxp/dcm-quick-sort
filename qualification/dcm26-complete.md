# DCM-26 complete — bounded text previews and truthful error states

11 November 2026 · standalone DCM QuickSort v1.0.71 · branch `release/v1.0.71`.
Final product/native-tested source: `921f3616da39e8c67f27ba1d496eaaf3594526ad`.

## Acceptance result

The final native `ui-text` run passed **17 behavior checks** against **15 disposable files**. All **17 final screenshots** were individually opened and inspected, including the actual cutoff notice at the bottom of both long previews. The application closed normally with exit code 0, its children drained, and source/executable and fixture identities remained unchanged. Destination and operation journal stayed empty. This is automated qualification, not human user acceptance.

Only DCM-26 is newly closed. Completed: DCM-01–18 and DCM-20–26, 25 checklist steps including five setup/diagnostic steps—not a product-completion percentage. DCM-19 remains blocked.

## Reproduced failures and corrections

**Encoding and cutoff:** three new native unit-test parents first reported one pass and two failures at `9c0b9154`: encoding validation and byte-limit boundaries failed. The old decoder silently replaced invalid UTF-8, dropped an odd UTF-16 byte, accepted malformed surrogate sequences, and treated UTF-32 as another encoding. Cutting a valid multi-byte character at the preview byte limit also produced replacement characters. The corrected decoder accepts UTF-8 and BOM-marked UTF-16LE/BE, reports malformed/unsupported encodings rather than guessing, and omits only an incomplete final character cut by the known preview limit. The existing 256-KiB limit, truncation notice, newline normalization and binary-control sanitation remain intact. Unpreviewed bytes beyond the limit are not validated.

**Misleading read error:** the native UI at `b74d5f36` passed seven valid-content cases, then failed on an exclusively locked TXT file. Its screenshot showed “No Windows preview handler is installed for TXT files” with a stale success footer. The actual problem was a sharing violation. The built-in text path now retains the specific read/encoding failure and updates its status instead of falling through to the generic Windows-handler fallback. Releasing the lock and selecting the same file restores its actual contents.

**Manual status:** screenshot review of the first complete run at `8e88430f` caught the manual “Read as text” path retaining the preceding TXT status. A strengthened assertion reproduced `DCM26_MANUAL_STATUS_STALE` at `ae0c9f5e`; the final small correction updates the successful manual-mode status to its current extension. No existing assertion was removed or weakened.

Product fixes were confined to `app-src/core.go` and `app-src/main_windows.go`. Code and tests were authored locally, Windows-test-cross-compiled locally, committed/applied in GitHub with exact input-blob/output-hash guards, then updated and tested on Lenovo through FlowDev. No remote source edits, installer changes, new preview engine, UI framework migration or printing work.

## Verified matrix and limits

| Case | Verified outcome |
|---|---|
| TXT with mixed newlines | Exact displayed text matches the normalized fixture. |
| Markdown and CSV | Literal plain-text content, including markup, quotes and delimiters; no rendered-Markdown or spreadsheet-grid claim. |
| BOM UTF-8, UTF-16LE, UTF-16BE | Matching Unicode text, including accent, Greek, Odia and emoji; code-unit readback and screenshots both checked. |
| Empty file | Explicit `[Empty file]` instead of a stale preview. |
| Locked file and recovery | Specific sharing error, no previous text; correct contents return after unlock. |
| Invalid UTF-8, odd UTF-16, UTF-32 | Explicit reason, with no misleading generic handler message or fabricated decoded text. |
| Binary control bytes | Existing safe interpretation and warning displayed; source bytes unchanged. |
| Above the 256-KiB bound | Exact bounded text and cutoff notice; a UTF-8 character split at the boundary is not corrupted. Both notice lines inspected visibly. |
| Explicit Read as text for an unknown extension | Exact contents and current manual-mode status, followed by successful return to plain TXT. |

The three new Go parents contain **31 nested cases**, including incomplete UTF-8, unpaired UTF-16 surrogates, both UTF-32 byte orders, valid literal U+FFFD, UTF-8/UTF-16 cutoff boundaries, exactly-at/above-default limits, empty/binary/missing/directory input and unchanged source bytes. Those nested cases are not added again to the parent-test total. UTF-16 requires a byte-order mark; legacy ANSI/code-page detection and UTF-32 support are not advertised. Automatic built-in text routing remains TXT/MD/CSV; explicit Read as text is separately exercised on an unknown extension.

## Fresh native evidence

All final results use `921f3616da39e8c67f27ba1d496eaaf3594526ad`:

| Selection | Result | Directory under `C:\FlowverseReports` |
|---|---|---|
| Complete application baseline | 361 passed, 0 failed, 1 skipped; 362 parents, exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261111-144109-ae0c2e92` |
| Text UI | 17 passed; 17 images reviewed; normal exit 0 | `dcm-quicksort-ui-text-20261111-144427-b716381b` |
| Launch regression | FlowDev PASS, exit 0 | `dcm-quicksort-ui-launch-20261111-144456-9fa864b0` |
| Refresh regression | FlowDev PASS, exit 0 | `dcm-quicksort-ui-refresh-20261111-144512-afa1b71e` |

The regression runs’ new images were not separately reviewed; their command PASS results are not new visual approval. The sole skipped Go test remains `TestRealGeneratedPPTXIntegration`; the identified `DCM_REAL_PPTX` fixture remains required before DCM-32/47 acceptance. Installer/package suites were not rerun. Three new parent tests explain the increase from 358 to 361 passes; UI records are separate.

Build receipt: `C:\FlowverseReports\dcm-quicksort-build-20261111-144207-01065b1a\receipt.json`.
Build receipt SHA-256: `c4f3ca07b37d5ff3fcb9660f0027bfe5a77fc794aa461cbb8d55836f0c084565`.
App SHA-256: `0522a02560bea2355c6d307eca807320836a51b4e9a8553470c8107b33f2523e`; 7,183,360 bytes.
Separate image hashes/review: `qualification/evidence/dcm-26-native-summary.json`.

## Retained evidence and test-tool corrections

- Native unit RED: `dcm-quicksort-regressions-20261111-141153-7e5bea1f` (one parent pass, two failures, zero skips, no timeout).
- Initial harness failure: `dcm-quicksort-ui-text-20261111-141531-cda04fbe`. PowerShell’s wait-helper parameter shadowed the target filename; the inspected screenshot already showed correct plain content. Capturing the expected stem outside that scope corrected the test, not the app.
- Genuine locked-file UI RED: `dcm-quicksort-ui-text-20261111-142205-eb06165d`.
- Earlier 17-assertion run: `dcm-quicksort-ui-text-20261111-142616-080dd881`. Its full-content readback passed, but image review found that the long-file screenshots remained at the top and manual status was stale; it was not used to close the final gate.
- Retained capture-guard failures: `dcm-quicksort-ui-text-20261111-143806-39aac57c` and `dcm-quicksort-ui-text-20261111-144244-3cb4b562`; the latter passed on an unchanged retry. No guard was bypassed. The cause of the hidden/minimized window state was not established.
- Strengthened manual-status RED: `dcm-quicksort-ui-text-20261111-143856-cb24c34d`.

The test-only long-text helper retains a bounded 600,000-character readback, ownership and timeout checks; the general native adapter’s smaller text guard remains unchanged. Tail capture now sets the caret to the actual end and verifies a nonzero first-visible line; negative-start EM_SETSEL had merely deselected text. Full content is still compared independently, and screenshots—not the successful scroll call alone—confirm the visible notice. Raw `visual_reviewed=false` fields remain untouched; the separate summary records inspection. Captures remain on Lenovo because a small desktop edge is visible.

## Continuation

Next: **DCM-27 — representative supported image previews and existing sizing/navigation behavior**. Preserve DCM-19/disposable Explorer isolation, label-layout issues, external PPTX, remaining preview/file-operation workflows, full Batch Move/Undo, final package/setup lifecycle and actual user acceptance. The long error footer can be clipped at the tested width, but the complete error is readable in the preview pane; general layout qualification remains open. Refresh the source-bound build receipt after later commits. PR #1 stays draft; no merge or publication.

Technical reference: Microsoft Learn EM_SETSEL, https://learn.microsoft.com/en-us/windows/win32/controls/em-setsel (negative start deselects; explicit matching end positions place the caret). Project behavior and acceptance are supported by the source-bound native records above.
