# DCM-42 — Batch Move and mixed-outcome acceptance (DESKTOP-6B6EH4T, 2026-11-17 IST)

## Decision

**PASS — bounded automated engineering acceptance.** Both the existing two-file full-success batch UI selector and a new mixed-success selector passed on DESKTOP-6B6EH4T. Native selection, cancellation, exact moved files and hashes, per-file conflict Skip, journal membership, restart recovery and Undo Batch were independently verified. This is not a claim of all possible low-level I/O failure handling, arbitrary batch sizes, or manual user sign-off.

## Exact source and gates

- Repository: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`; no QuickSort production application source changes in this DCM-42 round.
- **Full-success gate:** tested pushed `66ded9cd57bdbaa6babcd8fea65b05d6a0ced1b6`. `flowdev-test update dcm-quicksort`, `ui-tooling`, and `build` PASS. Source-bound build receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261116-225333-b43457e0\receipt.json`. Registered `flowdev-test test dcm-quicksort ui-batch` **PASS**, **7 case records and 6 captured original PNGs**, exit 0; raw headed receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-batch-20261116-225403-f8fe6986\receipt.json`.
- **Mixed-result gate:** tested pushed `a7f213a7da4ed90cf1cf3afc65e0fd96b9f282f8`. `flowdev-test update dcm-quicksort`, `ui-tooling`, and `build` PASS. Exact build receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261116-225713-d86edf59\receipt.json`. Registered `flowdev-test test dcm-quicksort ui-batch-partial` **PASS**, **6 case records and 6 captured original PNGs**, exit 0; raw headed receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-batch-partial-20261116-225741-6a8d665f\receipt.json`.
- Both raw receipts report `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, `error=null`.
- All work was executed on **DESKTOP-6B6EH4T**, using disposable fixtures beneath `E:\FlowverseTesting\Builds\DCM-QuickSort`. Lenovo was not used.

## Full-success two-file batch

1. The native file-list oracle selected **exactly** `02-bravo.txt` and `03-charlie.txt` from four disposable inbox fixtures. FlowUIVision read back `2 selected` and the `Move 2 to Vault` button.
2. The first batch confirmation was canceled. Independent checks confirmed **all four original source files and hashes unchanged**.
3. The same two items were selected and confirmed. QuickSort moved exactly those two files to the configured `Vault\Accepted`, preserved their bytes, kept `01-alpha.txt` and `08-preview.pdf` untouched, and displayed `Batch move complete: 2 of 2`.
4. The independent journal oracle verified both file transactions reached `completed` with **one shared nonempty batch ID**.
5. The application closed and restarted normally. It recovered two undoable transactions; native `Undo Batch` restored both selected source paths and hashes and journal status `undone` with retained batch membership.

## Mixed-success two-file batch

1. Test seeded only the disposable destination `Vault\Accepted\03-charlie.txt` with **distinct bytes**, independently pinned by SHA-256. The original inbox source `03-charlie.txt` and its SHA-256 were unchanged.
2. The native file-list oracle again selected **exactly `02-bravo.txt` and `03-charlie.txt`**, and the operator-confirmation dialog was accepted.
3. The file-specific `Filename Conflict` dialog for `03-charlie.txt` was checked for the correct destination and choices, then canceled. QuickSort completed the other member, `02-bravo.txt`, and displayed **`Batch move finished: 1 moved; 1 skipped or failed out of 2`**. The readback showed the original `03-charlie.txt` still in the inbox and only `02-bravo.txt` in Processed.
4. Independent checks verified `02-bravo.txt` absent from inbox and present at destination with identical bytes; the skipped source and its preexisting conflicting destination retained their distinct original hashes. Unselected `01-alpha.txt` and `08-preview.pdf` remained unchanged.
5. Journal inspection proved only the successful `02-bravo.txt` had a pending→completed transaction with one nonempty `batch-...` identity. **No false transaction** was written for skipped `03-charlie.txt`.
6. Normal restart recovered **one** undoable batch member. Native `Undo Batch` restored only `02-bravo.txt`, preserving both bytes and batch journal ID/undone status. The skipped source and conflicting destination remained identical. The test removed only its own pinned collision fixture and verified the full baseline inbox restored.

This is a **user-chosen per-file conflict Skip** as a representative mixed outcome. It does not manufacture or claim a disk-full, permissions, crash, or worker I/O failure.

## Visual evidence

All **12 original PNGs** were retrieved and individually reviewed from DESKTOP-6B6EH4T, six from each passing gate.

Full success: `launch-1`, `batch-selection`, `batch-completed`, `launch-2`, `undo-batch-completed`, `final`. The images show two selected files, two Processed after completion, two recovered after restart, and the original four files after Undo Batch.

Mixed: `launch-1`, `batch-partial-selection`, `batch-partial-completed`, `launch-2`, `batch-partial-undone`, `final`. The images show two selected files, the honest `1 moved; 1 skipped or failed out of 2` message, one Processed entry, and all four files restored. Actual bytes, source/destination paths, skipped fixture safety and journal membership are established by separate machine oracles, not inferred from screenshots.

## Preserved RED setup evidence

The first mixed-outcome run at `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-batch-partial-20261116-225614-7e46962f\receipt.json` failed the harness's generic 'all confirmation dialogs closed' wait. QuickSort transitioned normally from batch confirmation to a file-specific conflict dialog; the harness incorrectly treated this expected handoff as a failure. Source-controlled test harness code was adjusted to allow that specific expected follow-up and require verification of its title and contents. The historical failed receipt was not deleted or relabeled passing.

## Qualification boundary and milestone

DCM-42's selected-file, cancellation, successful per-file actions, mixed outcome, journal batch membership, restart recovery and Undo Batch contract is **COMPLETE at representative automated engineering acceptance**, not general failure-mode certification. Actual disk-write denial, large multi-hundred-file batches, out-of-space and abrupt crash still need separate scope if required. `user_test` stays `not_recorded`.

**Tracker: 40/50 complete, 10 remaining. DCM-43 Processed and history displays is the next independent milestone.** DCM-34 and DCM-19 remain separately open.
