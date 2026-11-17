# DCM-44 — Single Undo Move destination safety (DESKTOP-6B6EH4T, 2026-11-17 IST)

## Engineering decision

**PASS — bounded automated engineering acceptance for new fingerprinted Move records.** QuickSort now verifies the saved destination fingerprint before allowing a newly completed single Move to be undone. In a normal native UI test, an occupied original path and a changed moved destination both caused a refusal **without changing either protected file or the durable journal**. Once both disposable test obstructions were removed/restored and the destination matched its journaled original SHA-256, native Undo restored the exact original path and bytes, and marked the same transaction undone.

**Separate regression gate remains NOT ACCEPTED on this desktop:** the registered broad `regressions` suite detected one `TestD2WindowsRetirementRejectsFinalSymlink` skip due to Windows symlink-creation capability and intentionally returned failure even though the Go test command exited 0 and reported 26 passes, 0 failures, 1 skip in the targeted D1–D6/Release tests. Do not relabel that receipt as PASS.

## Source-controlled production correction

- Repository `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`; **exact tested pushed source:** `35c4391074e8963c100627f0c5b127bc45567dcf`.
- In `app-src/main_windows.go`, the normal same-volume `renameFileNoReplace` success path now captures the verified moved destination bytes using `snapshotSource(target)`, and records `OriginalSHA256` and `OriginalSize` with the completed durable journal transaction and Processed entry. If the fingerprint cannot be obtained, QuickSort marks that move as needing attention instead of claiming a normal safely undoable completion.
- In `app-src/core.go`, `verifyUndoMovedBytes` checks a fingerprinted moved destination's size and SHA-256 before Undo; mismatched contents are explicitly refused with `cannot undo: moved file contents changed after the move`. The check also applies to newly fingerprinted case-only renames.
- The previous occupied-original protection remains effective. No operation silently overwrites the colliding original.
- An isolated `TestUndoMoveWithRecordedFingerprintRejectsChangedDestination` Go unit test proves edited-target refusal, occupied-original refusal, and safe restored-byte Undo.
- Older legacy Move records **without a stored digest** retain previous Undo behavior; do not claim this upgrade retroactively protects them. Synchronous target fingerprinting may affect very large-file UI latency, which this small fixture does not quantify.

## Exact registered FlowDev runs

Device: **DESKTOP-6B6EH4T**; workspace `E:\FlowverseTesting\Builds\DCM-QuickSort`; Lenovo was not used. Source was pushed to GitHub before deployment; candidate checkout remained clean.

- `flowdev-test update dcm-quicksort`: **PASS**, source pin above.
- `flowdev-test test dcm-quicksort regressions`: **NOT ACCEPTED**; raw receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-regressions-20261117-021249-26a3faf7\receipt.json`. Existing symlink test skipped; tool correctly failed on skipped targeted coverage, with 26 passing targeted D1–D6/Release test cases. This suite's regex does not include the new `TestUndoMoveWithRecordedFingerprintRejectsChangedDestination` test, so the separate gate below is required.
- `flowdev-test test dcm-quicksort undo-move-safety-unit`: **PASS**, exact source-pinned Go unit. Receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-undo-move-unit-20261117-021439-a610a242\receipt.json`.
- `flowdev-test test dcm-quicksort ui-tooling`: **PASS**, 11 trace contracts and owned-process observer.
- `flowdev-test test dcm-quicksort build`: **PASS**, source-bound build receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261117-021454-615cd997\receipt.json`.
- `flowdev-test test dcm-quicksort ui-undo-move-safety`: **PASS**, exit 0, **7 case records, 7 PNG captures**. Raw receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-undo-move-safety-20261117-021527-eebeda0f\receipt.json`, `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, `error=null`.
- Post-fix existing `flowdev-test test dcm-quicksort ui-move`: **PASS**, receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-move-20261117-021635-1bfac57a\receipt.json`.
- Post-fix existing `flowdev-test test dcm-quicksort ui-batch`: **PASS**, receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-batch-20261117-021642-6beb1c72\receipt.json`.

All headed gates used the same exact-source successful build receipt, the existing pinned FlowUIVision helper, a clean GitHub checkout and disposable E: fixtures.

## Independent filesystem/journal oracles

1. QuickSort's native UI moved only `inbox\01-alpha.txt` into `Vault\Accepted\01-alpha.txt`; test checked the exact selected source, original SHA-256, and `completed` durable journal entry with an `original_sha256` and `original_size` corresponding to the moved file.
2. The owned app was closed and restarted. The matching Processed transaction reappeared and was selected.
3. The test harness created a **distinct-content disposable occupant** at the original `inbox\01-alpha.txt` location. Native Undo displayed `cannot undo: original path is already occupied`. Both occupant and destination SHA-256 and the journal SHA-256 remained unchanged.
4. The harness removed only that disposable occupant, then edited **only the test-owned moved destination** to distinct bytes. Native Undo displayed `cannot undo: moved file contents changed after the move`. The edited destination hash and journal SHA-256 were unchanged, and original source path remained absent.
5. The harness restored the disposable destination bytes from an **immutable original fixture copy kept under the report root**, whose SHA-256 was independently checked against the pinned original. Native Undo then restored the source path and original hash, removed the destination and wrote `undone` to the same journal transaction ID while retaining the saved fingerprint.
6. The final complete fixture oracle verified all original disposable files intact; application and owned processes shut down normally, without altering production source.

## Screenshot review

All **seven original UI captures** were retrieved from DESKTOP-6B6EH4T and independently reviewed: `launch-1`, `dcm44-move-completed`, `launch-2`, `dcm44-occupied-original-refused`, `dcm44-changed-destination-refused`, `dcm44-single-undo-restored`, `final`.

They show the expected single completed Processed record, persisted history after restart, retained active transaction after each refusal, restored source list and empty Processed after successful Undo. Error dialogs were checked and dismissed through the owned-dialog text/control assertions; they are not present in the subsequent post-dismissal screenshots. File bytes and durable journal evidence come from hash/file oracles, not visual inference.

## Qualification limits

DCM-44 is closed for **new fingerprinted, representative single-file native Moves** with explicit occupied-original and changed-destination refusal, plus success/restart/Undo. It is **not** certification of legacy no-digest records, atomic race freedom across concurrent external writers, all reparse-point/ACL cases, very large file hashing latency or human operation. Broad D1–D6 regression coverage remains **blocked on symlink privilege**, with the untouched failing receipt retained. No user sign-off is recorded.

**Milestones: 42/50 completed, 8 remaining** at bounded engineering acceptance. DCM-34, DCM-19 and broader symlink environment issues remain separate.
