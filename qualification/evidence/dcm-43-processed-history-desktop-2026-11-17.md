# DCM-43 — Processed and history displays (DESKTOP-6B6EH4T, 2026-11-17 IST)

## Decision

**PASS — representative automated engineering acceptance, with explicit partial-operation evidence reused from DCM-40.** QuickSort's Processed UI listed an actual completed move, did **not** present an explicitly refused conflict as a completed transaction, filtered and selected the correct persisted history row, and performed no filesystem or durable journal mutation merely from history search, selection, navigation, or restart. A separate, earlier real DCM-40 cross-volume partial-failure test independently established the application’s attention-required status and journal semantics; DCM-43 did **not** recreate that partial failure. Human-operated acceptance remains `not_recorded`.

## Machine and tested source

- Target: **DESKTOP-6B6EH4T**, E: disposable test fixtures; Lenovo not used.
- Repository: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`, exact tested source `fe34e913745e2adbf6339cf101b211154d7be4bc`.
- Registered selector: `flowdev-test test dcm-quicksort ui-history`, added under `tools/run-ui.ps1` and `tools/flowdev-dcm/config/checks.json`.
- `flowdev-test update dcm-quicksort`: PASS, exact branch, origin and clean source.
- `flowdev-test test dcm-quicksort ui-tooling`: PASS (11 trace contracts and owned-process check).
- `flowdev-test test dcm-quicksort build`: PASS, exact-source build receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261117-012115-7e189993\receipt.json`.
- `flowdev-test test dcm-quicksort ui-history`: **PASS**, exit code 0, **16 case records, 10 original PNG captures**.
- Immutable headed receipt: `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-history-20261117-012142-429d08f9\receipt.json`.
- Receipt: `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, `error=null`. No production application code modified.

## Direct success, refusal, read-only navigation and Undo checks

1. **Success:** through the normal Move UI, `02-bravo.txt` was relocated to the configured `Vault\Accepted` folder. The physical destination matched the source SHA-256, the journal recorded `completed`, and the Processed list presented the exact single transaction `02-bravo.txt → 02-bravo.txt`.
2. **Refused conflict:** the harness placed a different-byte disposable `Vault\Accepted\03-charlie.txt` destination and selected the original `03-charlie.txt` through QuickSort. The owned filename-conflict dialog's Cancel action preserved both file hashes. The journal SHA-256 did not change. The Processed pane **did not** gain a false completed `03-charlie` entry; status was `Filename conflict action cancelled; both files were kept unchanged`.
3. **Read-only history filter:** search for `03-charlie` displayed `PROCESSED 0 / 1`, not a false result; search for `02-bravo` displayed `PROCESSED 1 / 1`, and selecting the matching row displayed the correct source/destination path. Clearing search restored the unfiltered Processed view.
4. **No mutation on viewing:** SHA-256 snapshots of the durable `move-journal.jsonl`, the refused source, the synthetic collision and the successful destination matched **before and after** Processed searches, selected row readback and history navigation.
5. **Restart recovery:** normal owned application shutdown/restart recovered the one completed transaction. Querying history after restart again left all hashes and the journal unchanged, without adding an operation.
6. **Explicit Undo only:** once the harness had independently verified the synthetic collision unchanged, it removed only that test-owned collision fixture. The actual app Undo button restored the successful `02-bravo.txt` source bytes, cleared the Processed list, and transitioned the journal to `undone`. All four original disposable inbox fixtures and their hashes were restored.

The original, failed acceptance attempt `dcm-quicksort-ui-history-20261117-012016-f5d8d018` remains preserved. It reached the final restoration helper while the synthetic test-only collision still occupied the destination, violating the helper's standard expectation of an empty Vault. The subsequent source-controlled test correction explicitly checked its hash and removed that known test-only collision **after all read-only checks and before Undo**. The final entire suite passed. No production change or ignored assertion.

## Partial operation coverage from DCM-40 (not rerun)

The earlier registered `ui-cross-volume` source `25c9c0b` pass is recorded in `qualification/evidence/dcm-40-cross-volume-move-desktop-2026-11-16.md`, with original receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-cross-volume-20261116-163030-1b84035a\receipt.json`. On a real E:→D: copy, a deliberately locked source prevented source deletion. QuickSort preserved **both** SHA-256-identical files, displayed `Source retained; the move needs attention before retrying`, and wrote a terminal `attention` journal entry, instead of pretending a normal completed move. Its seven images were independently reviewed in the DCM-40 round. That evidence supports the DCM-43 requirement that partial states are represented honestly **at the status/recovery-journal level**. It does not establish that every partial state appears as a standalone Processed success row; an attention state is intentionally not a successful Processed transaction.

## Independent screenshot review

The ten passing DCM-43 PNGs were directly retrieved and individually reviewed on DESKTOP-6B6EH4T: `launch-1`, `dcm43-successful-processed`, `dcm43-refused-not-processed`, `dcm43-refused-history-search`, `dcm43-success-history-search`, `dcm43-history-readonly`, `launch-2`, `dcm43-restarted-history`, `dcm43-undo-completed`, and `final`. The images show the expected unfiltered and filtered counts, correct single transaction row and restored original file after Undo. The byte and journal assertions were verified independently, not inferred from screenshots.

## Qualification limits

The Processed pane represents undoable successful transactions; **refusals** are visible via immediate status and lack a false Processed success row. **Partial attention** is represented by status and durable Recovery journal rather than a completed Processed entry, as checked in DCM-40. This does not exhaust the Recovery screen's UI navigation, all partial types or arbitrary concurrent mutations; those require separate targeted acceptance. Neither direct manual user operation nor all failure and filesystem permutations are certified.

**DCM-43 complete at representative engineering acceptance. Tracker advances to 41/50 completed, 9 remaining. DCM-44 is the next independent step.** DCM-34 and DCM-19 remain separately open.
