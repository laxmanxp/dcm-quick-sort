# DCM-41 — Native UI conflict choices on DESKTOP-6B6EH4T (2026-11-17 IST)

## Decision

**PASS — bounded representative automated engineering acceptance.** The existing native Windows QuickSort destination conflict dialogs exercised Skip/Cancel, Keep Both, Replace, exact-duplicate No (keep source), and exact-duplicate Yes (remove verified source) against disposable TXT fixtures. Independent filesystem SHA-256, destination contents, journal operation kind/ID, and Undo checks passed. This is not manual end-user sign-off.

## Tested GitHub and FlowDev identities

- Repository: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.
- **Exact tested commit:** `d668dfb138b2b86e9d826ab5c28b43581165386a`. This changes registered test code, not QuickSort production code.
- Machine: **DESKTOP-6B6EH4T**, with E: test storage. Lenovo was not used.
- `flowdev-test update dcm-quicksort`: PASS, clean fast-forward checkout at tested commit.
- `flowdev-test test dcm-quicksort ui-tooling`: PASS (11 trace contracts, owned-process observer).
- `flowdev-test test dcm-quicksort build`: PASS, exact-source receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261116-200342-41a0d8fd\receipt.json`.
- `flowdev-test test dcm-quicksort ui-conflicts`: **PASS**, exit code 0, **18/18 case records**, **10 original PNGs**.
- Raw immutable headed receipt: `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-conflicts-20261116-200411-0aab38e6\receipt.json`.
- Receipt reports `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, `error=null`.
- Source-controlled selector: `tools/run-ui.ps1` suite `conflicts`, registered in `tools/flowdev-dcm/config/checks.json`.

## Actual source/destination contract

Each scenario starts from the isolated generated fixture `inbox\01-alpha.txt`, with an unoccupied `Vault\Accepted` destination. The test harness then creates only a **known disposable destination collision** and captures its SHA-256. All Move clicks, dialog choices and Undo operations go through QuickSort's owned UI controls; the harness never substitutes a filesystem move for an application operation.

1. **Skip/Cancel different-file conflict:** The test seeds a destination file with different bytes, selects the source, clicks Move and explicitly chooses Cancel from the `Filename Conflict` dialog. QuickSort displays `Filename conflict action cancelled; both files were kept unchanged`. Both independently measured hashes and the durable journal hash remain exactly unchanged.
2. **Keep Both:** Against a different-byte collision, the dialog's Yes / Keep Both option moves the source to an automatically numbered target (observed `01-alpha (2).txt`). The existing `01-alpha.txt` at the destination retains its exact original hash, the numbered new target has the source's original hash, and the source path disappears. The corresponding current transaction's journal goes pending→completed. Native Undo restores the source, removes the numbered target, keeps the pre-existing destination bytes and marks the **same journal ID** undone.
3. **Replace:** Against a different-byte collision, the dialog's No / Replace option moves the source to `Vault\Accepted\01-alpha.txt`. Before mutation, QuickSort creates an independently hash-verified recovery backup of the displaced destination, and that backup is present in the completed journal record. The new destination matches the source's original SHA-256. Native Undo restores the original source and the displaced destination's prior bytes, with the transaction journal moving to undone.
4. **Exact duplicate, No:** Against a byte-identical pre-existing destination, the `Exact Duplicate` dialog's No option leaves both files and durable journal unchanged. Native status confirms the source was kept.
5. **Exact duplicate, Yes:** The `Exact Duplicate` dialog's Yes option performs background size/SHA-256 verification and safely removes only the redundant source. The destination's bytes remain identical. The journal records kind `duplicate-removed`, pending→completed. Native Undo restores the source without touching the retained destination and finishes the same current journal transaction as undone.
6. After each scenario the harness checks that the original `inbox` fixtures and all unrelated files match their pinned baseline hashes, then removes **only its own test-seeded destination file**, never a user file.

For repeated operations against the same logical path, the final acceptance runner scopes journal assertions by the current **transaction ID** rather than an earlier conflict entry. It also waits for durable journal finalization after Undo, rather than assuming the filesystem and journal become visible at the exact same instant.

## Independent visual review

All ten original PNGs from the passing report were directly read from DESKTOP-6B6EH4T and inspected: `launch-1`, `dcm41-skip`, `dcm41-keep-both-completed`, `dcm41-keep-both-undone`, `dcm41-replace-completed`, `dcm41-replace-undone`, `dcm41-duplicate-no`, `dcm41-duplicate-yes-completed`, `dcm41-duplicate-yes-undone`, and `final`. They corroborate the source list, processed transaction labels, numbered Keep Both name, replaced-existing label, retained source after Skip/duplicate-No, duplicate-removed label, and restored original file after Undo.

The screenshots do not directly show the displaced destination bytes or durable journal ID. Those claims are grounded in independent SHA-256, recovery-copy and journal file oracles, not images.

## Retained earlier failed-run evidence

The preceding test development attempts remain untouched and are **not** relabeled passing:

- `dcm-quicksort-ui-conflicts-20261116-200010-b74d35eb`: final duplicate-removal Undo performed safely and journal contained the correct undone event, but harness compared its ID with an earlier Replace transaction.
- `dcm-quicksort-ui-conflicts-20261116-200136-f2af4222`: Replace restored both files and journal eventually contained the undone event; harness observed it before journal finalization.
- `dcm-quicksort-ui-conflicts-20261116-200300-4f1562ed`: duplicate-removal journal kind serialized as `duplicate-removed`, while the harness expected `duplicate_removed`.

All three were **harness assertions**, corrected in source-controlled GitHub commits. Final pass at `d668dfb` proves the entire registered suite. No product application source was edited in this qualification round.

## Scope and tracking

This acceptance covers disposable **same-volume, single-file** native conflict choices, destination backup integrity and session Undo. It does not cover network-share conflicts, large-file races, power loss during replacement, malicious destination substitutions, or all restart recovery contingencies; those require separate gates. `user_test` remains `not_recorded`.

**DCM-41 complete at bounded automated engineering acceptance. Milestone count: 39/50 complete, 11 remaining.** DCM-34's broader installed-provider failure scope and DCM-19's Explorer-dependent environment remain separate open items.
