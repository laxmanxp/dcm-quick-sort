# DCM-45 — Restarted Batch Undo with occupied/changed-member preflight (DESKTOP-6B6EH4T, 2026-11-17 IST)

## Decision

**PASS at bounded automated engineering acceptance for normal restart and unsafe-member recovery; abrupt mid-operation interruption is NOT qualified.** QuickSort moved exactly two disposable files, journaled a shared batch ID with preserved original SHA-256 values, restarted normally, recovered both undoable batch members, and refused to Undo any member when the original source path was occupied or the moved destination had changed. In both refusals the original destination bytes, collision bytes and durable journal were independently checked unchanged. When the test-only obstructions were corrected, the native Undo Batch control restored both files, their SHA-256 hashes, their original pathnames and their shared journal identity.

The milestone acceptance also mentions interruption/recovery. This gate proves durable **restart recovery** but does **not** emulate sudden termination while the batch is actively moving files. Do not infer crash/kill/power-loss recovery coverage from this pass; that remains a separately identifiable residual test.

## Source and pipeline

- Host: **DESKTOP-6B6EH4T**. Lenovo was not used.
- Repository: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`. **Exact tested commit:** `87f981de341fda421c441f85fbe9f7c7f7ca7bdb`.
- The DCM-45 focused selector is committed in `tools/run-ui.ps1`, suite `batch-undo-safety`, and registered as `ui-batch-undo-safety` in `tools/flowdev-dcm/config/checks.json`.
- `flowdev-test update dcm-quicksort`: **PASS** with clean fast-forward checkout.
- `flowdev-test test dcm-quicksort ui-tooling`: **PASS**, 11 trace contracts and one owned-process observer. Raw receipt: `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-tooling-20261117-032814-4652f95f\receipt.json`.
- `flowdev-test test dcm-quicksort build`: **PASS**, source-bound build receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261117-032818-d058b417\receipt.json`.
- `flowdev-test test dcm-quicksort ui-batch-undo-safety`: **PASS**, 9 recorded cases and 9 original PNG captures, exit code zero. Raw receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-batch-undo-safety-20261117-032846-51243aff\receipt.json`. Raw fields: `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, `error=null`.
- All test files were generated inside the E: disposable report workspace. No QuickSort production code was changed in this milestone.

## Independent assertions

1. The native UI selected exactly `02-bravo.txt` and `03-charlie.txt` from four baseline fixtures. It moved only those two to `Vault\Accepted`. The independent file oracle checked source absence, destination original hashes, and two untouched unselected files.
2. The durable journal contained exactly two newly completed member transactions with matching nonempty `batch_id` and the expected `original_sha256` values. This uses the new fingerprinted-move behavior from DCM-44.
3. QuickSort closed and restarted through its owned process controls. The app reported **`Journal recovery: restored 2 undoable move(s)`**; the native Undo Batch control was enabled, and the two completed member journal records with the original batch ID remained intact. The Processed date group was initially collapsed on restart, so row visibility was not confused with absence of recovered transactions.
4. A distinct-content disposable occupant was placed at the original `inbox\03-charlie.txt` path. Native Undo Batch presented its confirmation followed by the owned **`batch undo preflight failed ... original path is already occupied`** error. The test independently checked the occupant SHA-256, both moved destination SHA-256 values and the entire journal SHA-256 unchanged. The other source remained absent: **0 of 2 restored**.
5. After removing only that test-owned occupant, the harness changed the bytes of the moved `Vault\Accepted\03-charlie.txt` to distinct contents. Native Undo Batch again refused with **`batch undo preflight failed ... moved file contents changed`**. Both sources remained absent, both moved destinations retained their prior hashes (including the modified target), and the journal SHA-256 was unchanged: **0 of 2 restored**.
6. The harness copied the original disposable `03-charlie.txt` bytes from its independently hash-verified immutable backup to the test-owned destination. Native Undo Batch now safely restored both original names, locations and hashes. The journal recorded each transaction `undone` with the same batch identity, the Vault destination emptied, the Processed panel became empty, and all four baseline fixtures matched their original hashes.

## Visual inspection

All **nine** original PNGs were retrieved directly from DESKTOP-6B6EH4T and individually reviewed: `launch-1`, `dcm45-batch-selected`, `dcm45-batch-completed`, `launch-2`, `dcm45-batch-recovered`, `dcm45-occupied-refused`, `dcm45-modified-refused`, `dcm45-batch-undone`, `final`.

The images show two selected file rows; two Processed rows after completion; a collapsed Processed date group plus recovery status and enabled Undo Batch after restart; both refusal screenshots saying **`Batch undo restored 0 file(s) before stopping safely`**, with the batch still recoverable; and final restored four-file source list with no Processed entries. Error-message fragments were checked before dismissal through the owned-dialog control oracle; screenshots taken afterwards do not depict the dialogs. Byte and journal evidence is from independent file checks, not visual inference.

## Preserved test development failures

The two earlier DCM-45 test runs are preserved, **not** recategorized as passing:

- `dcm-quicksort-ui-batch-undo-safety-20261117-032617-e6051e5e`: harness treated collapsed Processed date-group children as missing history, despite the two recovered journal entries and enabled Undo Batch control. Assertion corrected to verify persisted batch membership and available native control.
- `dcm-quicksort-ui-batch-undo-safety-20261117-032733-60be0069`: malformed PowerShell `Test-Path ... -or` expression in the test harness. Corrected by grouping boolean expressions. The final full test passed.

## Scope and remaining risk

This is representative two-file **same-volume**, normal-close/restart, occupied-original and changed-destination **all-or-nothing preflight**, byte-integrity and journal acceptance. It is not an abrupt kill/power-loss test, nor proof of atomic rollback if a filesystem error arises after all preflight checks pass and after the first member has been restored; the implementation is designed to stop and report a partial restoration in that situation. Full physical user sign-off remains `not_recorded`. The previously identified broader symlink-regression desktop permission skip remains a separate issue.

**Engineering tracker: DCM-45 complete at bounded acceptance, 43/50 milestones completed, 7 remaining.** The untested mid-batch interruption remains explicitly noted for future recovery qualification.
