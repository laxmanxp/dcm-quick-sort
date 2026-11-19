# DCM-40 — Real E: to D: cross-volume Move (DESKTOP-6B6EH4T, 2026-11-16)

## Decision

**PASS — bounded automated engineering acceptance.** On the actual `DESKTOP-6B6EH4T` Windows host, a disposable source file on E: was moved through QuickSort's normal GUI to an isolated, test-owned destination on the **different D: NTFS volume**. The destination hash matched the source, the source was removed only on verified success, the journal persisted through restart and native Undo restored the original. A second attempt held a read-sharing file handle that prevented source deletion after a successful cross-volume copy; QuickSort explicitly retained both source and copied destination, displayed an attention-required status and error, and marked the durable journal `attention`. No production application source or unrelated files were modified.

## Isolation and identities

- Source machine: `DESKTOP-6B6EH4T`; Lenovo was not used.
- Repository: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`; **exact tested pushed commit** `25c9c0be4c7857cfc6f29f508ba1f0d8a8909ec0`.
- Source/report workspace: `E:\FlowverseTesting\Builds\DCM-QuickSort`.
- E: volume ID: `\\?\Volume{46383369-0000-0000-0000-009f24000000}\`.
- D: volume ID: `\\?\Volume{8fe54965-0268-49d4-a378-18deeeec3130}\`.
- The runner verified distinct IDs and target NTFS before preparing anything.
- The **unique and absent-before-test** second-volume root was `D:\FlowverseTesting\DCM-QuickSort\DCM40-20261116-163030-1b84035a`. It contains a marker `.dcm40-disposable-test-owned` and an initially empty `Accepted` folder. No preexisting directory or user file was overwritten, and no broad cleanup command was used.
- Only the disposable test's own `settings.json` had `vault_root` temporarily pointed to that D: directory. Source-control and installed Office registrations were untouched.

## Registered pipeline

- `flowdev-test update dcm-quicksort`: **PASS**, source synced and clean.
- `flowdev-test test dcm-quicksort ui-tooling`: **PASS**, 11 trace contracts and owned-process observer.
- `flowdev-test test dcm-quicksort build`: **PASS**, exact-source receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261116-163001-5aba9c24\receipt.json`.
- `flowdev-test test dcm-quicksort ui-cross-volume`: **PASS**, exit 0, **9 case records**, **7 screenshots**.
- Raw immutable report: `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-cross-volume-20261116-163030-1b84035a\receipt.json`. Raw flags `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, `error=null`.
- Selector committed in `tools/run-ui.ps1`, registered in `tools/flowdev-dcm/config/checks.json`; no QuickSort production code edits.

## Actual cross-volume success path

1. Source was `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-cross-volume-20261116-163030-1b84035a\inbox\01-alpha.txt`; destination `D:\FlowverseTesting\DCM-QuickSort\DCM40-20261116-163030-1b84035a\Accepted\01-alpha.txt`.
2. FlowUIVision entered `Accepted` through the actual destination field; QuickSort's native file-list selection oracle asserted **exactly one selected file**.
3. Native Move triggered the cross-volume background copy and guarded source removal. The app displayed `Background move complete and journaled` with the real D: path. Independent file checks established E: source absence, D: destination existence, exact SHA-256 equality and all three other E: fixtures unchanged. Journal terminal state `completed` for the exact E:→D: pair.
4. QuickSort was closed and restarted normally; the exact processed transaction reappeared. Native Undo restored source bytes to E:, removed the D: target, and transitioned the same journal transaction to `undone`. The complete original E: file set was restored.

## Guarded partial-failure path

The test selected `01-alpha.txt` again and held a live handle opened with `FileShare.Read`. This explicitly allowed the application to **read and copy** but denied normal Windows delete/rename sharing while the handle remained active. The normal Move button was pressed; QuickSort copied and verified the destination, then rejected retirement of the locked source. The native error dialog was checked and dismissed, the visible status became `Source retained; the move needs attention before retrying`, and the journal recorded `attention` for the second transaction.

The runner verified both **source and destination still existed**, both had the original hash `ed80c15fbda9a2955adb26430305a1753afcae571f85eeee37c5d89cecbbc70d`, and the three unselected fixture hashes were unchanged. After the test exited, an **independent second desktop read** again confirmed: source exists, destination exists, marker exists, both hashes match the pinned expected hash, journal terminal state `attention`, volume IDs are distinct. The test released its lock handle in a `finally` block, but did **not** delete the retained copy or source, preserving the exact partial state as evidence in its own disposable directories. Those two identical files are *expected for this deliberately induced failure* and are not an accidental duplicate from a falsely successful operation.

## Screenshot review

All seven original PNGs were individually retrieved and reviewed from DESKTOP-6B6EH4T: `launch-1`, `dcm40-before-copy`, `dcm40-cross-volume-moved`, `launch-2`, `dcm40-cross-volume-undone`, `dcm40-source-retained-attention`, `final`. They show the selected source, `Accepted` location, successful D: path status, recovered processed transaction, restored source, and truthful attention-required partial status. These images were reviewed separately from machine receipts; the raw receipt's `visual_reviewed` flags were not rewritten.

## Boundaries and follow-on

This qualifies a real E:→D: same-machine NTFS cross-volume move and one controlled source-retirement refusal case. It does not establish disconnected network shares, disk-full, forced process termination during an actual copy, reparse-point destinations, arbitrary ACL failures or physical end-user sign-off. No simulated disk outage was invented.

**DCM-40: COMPLETE at bounded automated engineering acceptance. Tracker: 38/50 complete; 12 remaining.** DCM-34 and DCM-19 remain open. The next independent file-operation milestone is DCM-41.
