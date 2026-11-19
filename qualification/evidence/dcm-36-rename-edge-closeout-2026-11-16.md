# DCM-36 — normal-UI Rename filename edge acceptance (16 November 2026)

## Decision
**PASS — bounded representative engineering acceptance.** DCM-36's normal-UI Rename scenarios (spaces, Unicode, case-only spelling, invalid name and existing-name conflict) passed without data loss on Lenovo. User-operated acceptance is **not recorded**. No XLSX/PPTX rendering scope was reopened.

## Tested source and mandatory pipeline
- Repository: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.
- Exact tested source: `e477bc3de0c3de90d067871ab857c148451ec88f`.
- `flowdev update dcm-quicksort`: PASS.
- `flowdev test dcm-quicksort ui-tooling`: PASS.
- `flowdev test dcm-quicksort build`: PASS; exact build receipt `D:\FlowverseReportsRepo\dcm-quicksort-build-20261116-123909-6a9b754f\receipt.json`.
- `flowdev test dcm-quicksort ui-rename-edge`: **PASS**, exit code 0, **19/19 cases**, **10 original PNG captures**, no error.
- Headed receipt: `D:\FlowverseReportsRepo\dcm-quicksort-ui-rename-edge-20261116-124028-ccbf9d35\receipt.json`; `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`.
- The registered full `flowdev test dcm-quicksort regressions` passed at earlier pushed application-code-identical `0a918212efefdfd51858c5ae9a97057b868a05cd`, before subsequent changes exclusively to the headed test runner. Do **not** attribute a fresh full regression run to the final documentation/test-only descendant.

## Native GUI and safety oracles
All operations targeted the disposable `01-alpha.txt` through QuickSort's owned native Rename and Undo controls. Assertions checked exact enumerated Windows directory entry case, source and destination SHA-256, identity-preserving pending→completed→undone durable rename journal transitions, complete fixture restoration and absence of unintended destination changes.

1. **Spaces:** `01-alpha.txt` → `Odisha Project Notes.txt` → Undo; exact filename and bytes restored.
2. **Unicode:** `01-alpha.txt` → `Café Odisha 2026.txt` → Undo. The Unicode input was constructed as `'Caf'+[char]0x00E9` to bypass legacy PowerShell script encoding problems. UTF-8 receipt was read and independently inspected: character code sequence includes **233 = U+00E9** at the expected position; the corresponding GUI screenshot visibly shows `Café`, not `CafÃ©`.
3. **Case-only:** `01-alpha.txt` → `01-ALPHA.txt` → Undo. Exact directory-entry case and journal were verified; test did not mistake Windows case-insensitive existence checks for proof.
4. **Invalid:** entered `bad:name`; observed the owned error dialog `name contains an invalid character: ':'` and dismissed its actual owned OK control. No filesystem or journal mutation.
5. **Conflict:** attempted `02-bravo.txt` (already present), explicitly canceled the owned filename-conflict dialog. Both source and conflicting-file SHA-256 stayed unchanged; journal hash unchanged. GUI status confirmed both files kept unchanged.

## Product correction
Baseline case-only rename was reproducibly rejected as an unsafe same-file conflict. A disposable Windows diagnostic confirmed `MoveFileExW` with no-replace flags changes only directory-entry casing while preserving bytes. Product code in `app-src/core.go` now narrowly permits same-file *case-only same-directory* operations only when exact source directory entry is verified, preserving refusal of hardlinks/other aliases and destructive duplicate/replacement operations. Guarded Undo verifies expected target casing. `app-src/move_journal.go` reconciles pending/completed case-only operations by actual directory-entry casing, treating ambiguities as requiring attention. Source-controlled regression tests cover Undo and restart recovery. Existing safety regression was adjusted **only for nondestructive case-only inspection** and continues refusing duplicate-removal and replacement against same-file aliases.

## Visual evidence and preserved RED history
All ten original latest-run PNGs were independently read and inspected: initial, spaces/Undo, Unicode/Undo, case-only/Undo, invalid rejected, conflict canceled, final. They show the expected file list/history and restored original state; some Undo captures show an unrelated PDF rendering transition after selection changes and are not claimed as settled PDF previews. Native receipt, SHA and journal oracles are authoritative for file operations.

Earlier failures remain immutable:
- `dcm-quicksort-ui-rename-edge-20261116-121905-2b9c7d9d`: genuine case-only rejection before fix.
- `dcm-quicksort-regressions-20261116-122548-a3956e59`: outdated same-file inspection safety assertion; subsequent full native regressions PASS with destructive protections preserved.
- `dcm-quicksort-ui-rename-edge-20261116-123425-37333310`: harness used wrong OK button ID; actual owned dialog ID 2.
- `dcm-quicksort-ui-rename-edge-20261116-123702-0d924947`: machine PASS, screenshot exposed mojibake Unicode script input; fixed in final run.
No failed receipt was relabeled, deleted or weakened.

## Scope and tracker
The tracked DCM-36 criterion is qualified at a representative automated engineering level, not exhaustive arbitrary Unicode normalization, case-sensitive Windows directories, symlinks, all Windows filesystem types, or physical end-user acceptance. Original user-test states remain `not_recorded`.

**Close DCM-36 as complete: 34 of 50 steps complete, 16 remain.** DCM-34 remains PARTIAL for genuinely absent/failing installed Office provider environments beyond completed bounded fallback and PDF qualification; DCM-19 remains blocked. Do not reopen previously accepted Office resize or cleanup work without a new reproducible defect.
