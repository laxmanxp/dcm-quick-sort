# DCM-38 — Destination Selection, DESKTOP-6B6EH4T (2026-11-16)

## Decision
**PASS — representative automated engineering acceptance.** The supported native QuickSort Vault destination field resolves the existing logical location `Accepted` to its actual disposable Vault folder. Nonexistent `NotConfigured` and traversal-style `../OutsideVault` inputs are rejected by an owned error dialog before any filesystem or durable journal mutation. With `Accepted` selected, QuickSort moves only the selected fixture to the expected Vault directory and Undo restores it byte-for-byte. End-user acceptance remains `not_recorded`.

## Source and device
- Machine: `DESKTOP-6B6EH4T`; no Lenovo acceptance was used.
- Candidate: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`, tested source commit `a99dc95b57e64db2bf656777fae5a609b57b09a2`.
- Workspace: `E:\FlowverseTesting\Builds\DCM-QuickSort`.
- Registered selector: `flowdev-test test dcm-quicksort ui-destination` using source-controlled `tools/run-ui.ps1` suite `destination` and `tools/flowdev-dcm/config/checks.json`.
- `flowdev-test update dcm-quicksort`: PASS.
- `flowdev-test test dcm-quicksort ui-tooling`: PASS (11 trace contracts, one owned process observer).
- `flowdev-test test dcm-quicksort build`: PASS, exact source-bound receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261116-155448-06c30566\receipt.json`.
- `flowdev-test test dcm-quicksort ui-destination`: **PASS**, **10 case records, 7 PNG captures, exit 0**.
- Raw headed receipt: `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-destination-20261116-155516-f16a5e34\receipt.json`.
- Receipt states `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, `error=null`.

## File and journal oracles
Initial disposable inbox: `01-alpha.txt`, `02-bravo.txt`, `03-charlie.txt`, `08-preview.pdf`. Fixture config defines Vault root `<report>\Vault` with existing child `Accepted`, exposed through logical destination `Accepted` and favorite `Acceptance`. Selected source was `01-alpha.txt`.

1. `NotConfigured` was entered into QuickSort's actual destination field. The native Move control was clicked; the owned dialog required an existing logical location inside the configured Vault and identified that `NotConfigured` did not exist. After the error was acknowledged, every fixture retained its exact hash and path and the journal remained unchanged.
2. `../OutsideVault` was entered and rejected through the same native UI interaction before mutation. Original source hashes, destination contents and journal state remained unchanged.
3. `Accepted` was entered. FlowUIVision confirmed the actual displayed destination field and enabled Move; a screenshot shows the favorite `Acceptance`, logical location `Accepted`, and Open Location enabled. Native QuickSort Move transferred the one selected `01-alpha.txt` to `<report>\Vault\Accepted\01-alpha.txt`. Independent checks proved source absence, destination SHA-256 equal to initial bytes, other inbox fixtures unchanged, and a completed durable Move journal record.
4. Native Undo returned the exact bytes to the inbox and marked the same journal operation undone. All four source fixtures were restored; the Vault destination contained no unexpectedly retained file. Owned process exited normally.

## Image review
All seven original captured PNGs were read directly from DESKTOP-6B6EH4T and reviewed:
`launch-1`, `dcm38-missing-logical-location-rejected`, `dcm38-invalid-traversal-location-rejected`, `dcm38-valid-location-ready`, `dcm38-valid-location-moved`, `dcm38-valid-location-undone`, `final`.

The screenshots corroborate the destination field, favorite selection, moved source list/Processed transaction, and restored `01-alpha.txt`. The final screenshots show a transient unrelated PDF rendering state; this is not a PDF rendering qualification. Hash, actual physical path and journal oracles establish DCM-38 file integrity.

## Preserved failed setup runs
- `dcm-quicksort-ui-tooling-20261116-155150-ef617848`: rejected a missing parenthesis in the newly registered PowerShell test before app launch; source corrected and subsequent tooling PASS.
- `dcm-quicksort-ui-destination-20261116-155320-2cd82837`: owned invalid-destination error was correct and captured, but test read the destroyed message-box window too early and hit a window-text deadline; a bounded 250 ms post-acknowledgement settle interval fixed the test race. No product source change; subsequent gate PASS. Neither failed receipt is reclassified as a pass.

## Scope and next work
DCM-38 is qualified at the representative native destination-field, one configured favorite, nonexistent path and traversal-style path engineering level. It does **not** establish full interactive location-manager dialog coverage, arbitrary UNC shares, broken junctions, disconnected drives, permission-denied removable folders or physical user sign-off. These limitations should not be overclaimed.

With DCM-38 complete, the numbered tracker advances to **36/50 complete; 14 remaining**. DCM-34 and DCM-19 remain separately open; next independent file-operation milestone is DCM-39.
