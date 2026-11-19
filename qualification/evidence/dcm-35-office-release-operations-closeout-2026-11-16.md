# DCM-35 — Office preview release and same-file operation closeout (2026-11-16)

## Engineering acceptance

**PASS, bounded — DCM-35 representative engineering acceptance COMPLETE.** The tracked criterion is: *After switching/closing previews, owned workers and file locks are released so rename/move can proceed.*

This decision combines separately attributed native preview-release evidence, default PDF in-flight shutdown/rename/move evidence, Sumatra subprocess lifecycle evidence, and a new direct **DOCX, XLSX and PPTX same-file preview → TXT → exclusive access → native Rename/Undo → preview → TXT → exclusive access → native Move/Undo** acceptance selector on disposable fixtures. No new QuickSort production application source was changed. This is not end-user acceptance, arbitrary-file/provider certification, or a claim that the newly combined Office selector activated installed Word/Excel/PowerPoint preview handlers.

## Pushed source and mandatory pipeline

Repository `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`. Tested pushed source `25d257482567050eadecc3cd52b3014a7a14997d`.

- `flowdev update dcm-quicksort`: PASS, synchronized to tested commit.
- `flowdev test dcm-quicksort ui-tooling`: PASS.
- `flowdev test dcm-quicksort build`: PASS; receipt `D:\FlowverseReportsRepo\dcm-quicksort-build-20261116-120504-55a8f9fe\receipt.json`.
- `flowdev test dcm-quicksort ui-office-preview-release-operations`: **PASS**, exit code 0, **37 of 37 cases**, **26 retained images**.
- Exact headed report: `D:\FlowverseReportsRepo\dcm-quicksort-ui-office-preview-release-operations-20261116-120612-925021fd\receipt.json`.
- The raw receipt reports `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, no test error.

The new selector is source-controlled in `tools/run-ui.ps1` and `tools/flowdev-dcm/config/checks.json`. All three Office inputs were pinned by SHA-256, and were read from existing historical disposable fixtures (DOCX/XLSX) or the committed `tests/fixtures/dcm32-slides.pptx`. The runner used the current installed-handler-or-fallback selection mechanism; **on this run the screenshots show the built-in DOCX OpenXML text fallback, XLSX table fallback and PPTX slide fallback**. The test did not edit Office registration, installed application source, or Lenovo production source.

## Required cross-feature behaviors directly observed

For each `05-letter.docx`, `06-orders.xlsx`, `07-slides.pptx`:

1. Select the exact source file and observe a truthful format-specific preview status, with representative visible content in a settled before-operation screenshot.
2. Select `01-alpha.txt` and verify the normal TXT state.
3. Open that **same source file** with `FileShare.None` within a bounded wait, then confirm unchanged SHA-256.
4. Reselect it and use QuickSort's own Rename UI; verify original source path absence, new path, unchanged bytes, pending→completed durable journal entry.
5. Use QuickSort Undo; verify original path and hash restored and journal pending→undone.
6. Repeat preview→TXT→exclusive access; reselect and perform QuickSort Move to the disposable Accepted destination. Assert source absence, destination hash and completed journal, then Undo and verify complete source/destination state and hash.
7. Finish on TXT, exclusively reopen all three files, and verify the entire disposable inbox restored, no unexpected destination files, source checkout unchanged, and owned QuickSort processes closed.

The direct new run has **6 preview/release/operation cycles**, **6 completed journal operations**, and **6 Undo restorations** across the three formats, plus final exclusive file accesses. These are not mere enabled-button checks.

## Independent screenshot review

All **26 original screenshots** were read directly from the Lenovo report and inspected: initial/final frames plus eight per format. DOCX shows the FlowDoc/Gurundia Block content; XLSX shows the Orders sheet; PPTX shows Bridge Inspection, September 2026 and the image. After each preview, the TXT fixture is visible without stale prior content. Rename frames display renamed files and processed history; Move frames show source list removal and Undo, and the original filenames are restored afterward. Some operation/Undo frames are **transient**: the preview can be blank, display 'Preview', or show another newly selected file while state settles. They do not independently prove all post-operation frames are fully rendered, nor are they treated as rendering defects. Hashes, filesystem state and journal oracles establish the operations.

## Separate native-provider and PDF corroboration

Installed native Word/Excel/PowerPoint handler evidence is held in previous independently qualified receipts:
- Native DOCX: `D:\FlowverseReportsRepo\dcm-quicksort-ui-docx-20261115-152925-08feeec9\receipt.json`, PASS, explicitly reports installed Windows DOCX preview handler.
- Native XLSX: `D:\FlowverseReportsRepo\dcm-quicksort-ui-xlsx-20261116-041317-7805bb81\receipt.json`, PASS, installed Windows XLSX preview handler.
- Native PPTX: `D:\FlowverseReportsRepo\dcm-quicksort-ui-pptx-20261116-045200-6e624785\receipt.json`, PASS, installed Windows PPTX preview handler.

These previously accepted native runs include exclusive source access after preview switching and after close plus normal host close. **They do not perform the new same-file native Rename/Move sequence while that installed provider is active.** Do not conflate these evidence types.

PDF and independent lifecycle evidence:
- `qualification/evidence/dcm-35-pdf-close-inflight-2026-11-16.md`: real renderer drains after normal close, obsolete result discarded, temporary PNG and source lock released.
- `qualification/evidence/dcm-35-rename-move-lifecycle-2026-11-16.md`: native normal rename/move, Sumatra repeated switching and pending-close process cleanup.
- `qualification/evidence/dcm-35-same-pdf-release-rename-move-2026-11-16.md`: 13-case same-file default PDF preview-release Rename/Undo Move/Undo acceptance with 10 separately reviewed screenshots.

## Preserved failures and scope limits

Two earlier Office selector runs are historical RED, retained:
- `dcm-quicksort-ui-office-preview-release-operations-20261116-120111-c31b31b8`: fixture pinning included unrelated PNG, failed setup before app test.
- `dcm-quicksort-ui-office-preview-release-operations-20261116-120303-b1cca991`: DOCX Rename/Undo and release passed, then generic descendant snapshot hit a foreign native-provider PID boundary; corrected to the established host-owned statics observation without relaxing ownership guards.
- The separately registered broad `ui-previews` suite remains historical RED at FlowUIVision PPTX dropdown-label mismatch. It is **not** silently relabeled PASS.

DCM-35 is complete **only within representative automated engineering acceptance**, not arbitrary application interoperability, all Office installations, physical keyboard/mouse acceptance, or unsaved external Office documents. `user_test=not_recorded` is preserved. DCM-34 remains separately PARTIAL; DCM-19 remains blocked; release packaging still depends on the other FileFlow Suite components.

The project tracker can now advance from **32/50 to 33/50 complete**, with **17 remaining**. This is a milestone count, not an estimate of workload percent.
