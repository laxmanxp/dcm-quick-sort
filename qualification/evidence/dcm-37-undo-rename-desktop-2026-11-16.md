# DCM-37 — Undo Rename occupied-original-path acceptance (DESKTOP-6B6EH4T, 2026-11-16)

## Decision

**PASS — representative automated engineering acceptance on DESKTOP-6B6EH4T.** The selected Rename transaction survives an application restart. Undo refuses when an independently created, different-content file occupies its original path. Both the collision file and the renamed file remain byte-identical, and no false journal transition occurs. Once the disposable obstruction is cleared, the normal QuickSort Undo button restores the exact original name, bytes and durable journal identity. User-operated acceptance is not recorded.

## Machine, identities and workflow

- Windows target: `DESKTOP-6B6EH4T`; Lenovo was **not** used for these gates.
- E: workspace: `E:\FlowverseTesting\Builds\DCM-QuickSort`.
- Candidate repository: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`; tested pushed commit `d5cd6df5618433d8b3806908232bb2ddef0160e6`.
- FlowDev isolated profile adapter source is committed under `tools/flowdev-dcm/cmd/flowdev-test`. The adapter Go unit test and compiled runner both passed, source-controlled at `24131cc9b0a822f7df43bfdac6641c5382d03c25`. Runner SHA-256 `4D534E7D8AE39EA333E204A46C18A2FE49DADD92772E1AEFC448831F9F1C3149`.
- The authoritative monorepo FlowDev CLI was separately compiled from `laxmanxp/flowverse` `dev` at `1e4d507f7e7b1ee4760dc1ad7dfdc6dd0ce47da8`. It does not register DCM QuickSort; it was **not** substituted for the isolated QuickSort selector runner.
- FlowUIVision helper compiled from the same `flowverse/dev` commit. SHA-256 `A516C6FD8C0F84D31316503658FBBB813E9EB0D08F91EAE6DF371E7D2B52A342`.
- `flowdev-test update dcm-quicksort`: **PASS**, source synchronized with `d5cd6df`.
- `flowdev-test test dcm-quicksort ui-tooling`: **PASS**, 11 trace contracts and one owned-process observer.
- `flowdev-test test dcm-quicksort build`: **PASS**, exact-source receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261116-151806-dbf1ced4\receipt.json`.
- `flowdev-test test dcm-quicksort ui-undo-rename`: **PASS**, 9 cases, 6 captures, zero error.
- Raw immutable test receipt: `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-undo-rename-20261116-151836-bcd20ac1\receipt.json`.
- All built/source identities remained unchanged; `source_unchanged=true` and `all_owned_processes_closed=true` in raw receipt.

## Direct acceptance

1. QuickSort's native Rename control changes `01-alpha.txt` to `01-renamed-alpha.txt`. The renamed file retains original SHA-256 and journal has the exact pending→completed transaction.
2. The normal app closes and restarts. The matching processed transaction is found and selected in the restored Processed list; the journal identity is retained.
3. The test harness creates a **different-content** file at the old original location as disposable collision setup. The native Undo button is clicked. The app displays its owned error `cannot undo: original path is already occupied` and the test dismisses it. Independently measured hashes confirm the collision file and renamed file remain unchanged. The durable journal SHA-256 is unchanged and transaction is still completed.
4. The test removes **only its own synthetic collision fixture** and again clicks QuickSort's native Undo control. Original path and hash are restored, renamed path absent, status confirms Undo, journal entry transitions pending→undone with same operation ID, and the complete fixture set is restored.
5. The runner verifies original candidate source hash/checkout unchanged and all owned QuickSort processes closed. No user data was touched.

## Separate image review

Each of the six retained original captures was read directly from the target machine and reviewed: `launch-1`, `dcm37-renamed`, `launch-2`, `dcm37-refused-occupied-original`, `dcm37-undo-restored`, and `final`. Renamed filename and Processed transaction are visible. Following refusal the transaction remains available; following successful Undo, the original `01-alpha.txt` is back in the file list and Undo is inactive. The restored capture includes a transient unrelated PDF rendering frame; the final capture shows settled valid PDF page 1. PDF rendering was not a DCM-37 gate; actual filesystem/journal assertions establish Undo safety.

## Scope and tracking

This closes DCM-37 only at representative automated Windows engineering acceptance, not human test sign-off or every possible filesystem/reparse-point configuration. The isolated profile adapter is deliberately narrow, source-controlled, and does not add unregistered shell actions. DCM-34 remains separately PARTIAL for broader real Office failure environments, and DCM-19 remains blocked by required disposable Explorer environment.

**Tracker: 35/50 complete, 15 remaining.** The next independent milestone is DCM-38 destination selection; packaging remains gated on Flow Search and DeepSearch.
