# DCM-39 — Same-volume Move native UI acceptance (DESKTOP-6B6EH4T, 2026-11-16)

## Decision

**PASS — representative Windows E: same-volume engineering acceptance.** QuickSort's normal UI moves exactly one selected disposable file into the configured Vault `Accepted` location on the same E: volume. Independent checks verify source absence, destination content hash, all unselected files unchanged, durable journal completion, restart recovery and native Undo restoring the source. This is not cross-volume qualification or physical user sign-off.

## Source-bound FlowDev pipeline

- Device: `DESKTOP-6B6EH4T`; Lenovo not used.
- Repository `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`; exact tested GitHub commit `fd694c7e78ff281a679b974a547cf2eecd3585ec`.
- Isolated FlowDev registered profile runner already staged under `E:\FlowverseTesting\Builds\DCM-QuickSort\Tooling\flowdev-test.exe`.
- `flowdev-test update dcm-quicksort`: **PASS**, current branch/clean checkout and GitHub fast-forward checked.
- `flowdev-test test dcm-quicksort ui-tooling`: **PASS**, 11 trace contracts and one owned process observer.
- `flowdev-test test dcm-quicksort build`: **PASS**; source-bound build receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261116-160702-6341f9ae\receipt.json`.
- `flowdev-test test dcm-quicksort ui-move`: **PASS**, 6 case records, 5 original screenshot captures, exit code 0.
- Raw report `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-move-20261116-160754-cedcd1b6\receipt.json`; `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, `error=null`.
- No new app or test runner source code was necessary; the already registered `ui-move` selector meets DCM-39's acceptance contract.

## Exact filesystem and transaction oracles

The test generated all fixtures beneath the E: report root. Source was `<report>\inbox\01-alpha.txt`; destination was `<report>\Vault\Accepted\01-alpha.txt`, thus the same E: volume without a cross-volume copy requirement. The other inbox fixtures were `02-bravo.txt`, `03-charlie.txt`, and `08-preview.pdf`.

1. FlowUIVision entered `Accepted` in the Vault Location field and asserted the normal Move control enabled. Native selected-list readback identified **exactly `01-alpha.txt`**, not a caret row or all filtered rows.
2. The owned QuickSort Move button performed the relocation. The fixture oracle asserted the original source path absent, destination present, exact original SHA-256 preserved and every unselected file unchanged. The destination had no unexpected file.
3. The journal oracle independently read the source-bound durable `move-journal.jsonl` and required a completed entry for the expected from/to pair. The GUI displayed the matching processed transaction and moved/journaled status.
4. QuickSort was closed normally and relaunched as an owned process. The processed transaction was recovered and selected by filename. Native Undo restored the file at the original source path, removed it from the Vault target, preserved original hashes and changed the journal state to `undone`.
5. The runner checked the original checkout still clean/unchanged and all QuickSort owned processes exited normally.

## Visual review

All five actual PNGs were read and reviewed directly from the desktop: `launch-1`, `move-completed`, `launch-2`, `undo-move-completed` and `final`. The initial file selection, `Accepted` destination and processed move are visible. The restart screenshot recovers a processed transaction. Undo and final screenshots display `01-alpha.txt` restored in the source file list and no remaining active Undo transaction. During the initial move the Name filter showed zero matches, which is the correct filtered state after the selected source disappears; the physical/hash oracles separately confirm that other files remain unmodified.

## Scope and tracker

DCM-39 covers an actual same-volume E: move, physical source/destination/hash checks, retained journal, restart and Undo. DCM-40 **cross-volume movement** still needs an isolated second distinct volume and should not be inferred from this test. No arbitrary network/junction/removable storage, permission failures, or end-user manual sign-off are covered.

This report closes DCM-39 at representative automated engineering acceptance: **37/50 completed; 13 remaining.** DCM-34 and DCM-19 remain separately open.
