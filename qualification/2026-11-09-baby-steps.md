# DCM QuickSort - 50-step continuation, 9 November 2026

## Scope and source

User approved finishing standalone DCM in 50 baby steps with its existing UI. The plan is `completion-50.json`; it supplements, not replaces, the original `todo.json` defect and release gates. The retained application is Go, centered on Preview -> Rename -> Move and essential undo/recovery. No Walk migration, new PDF engine or printing work was introduced.

Baseline source: `3f7784583b0ebc292ea1572373b6bcbea22cad44`, branch `release/v1.0.71`, draft PR #1. Canonical modules remain `app-src/` and `installer-src/`. The source checkout on Lenovo was updated through the registered DCM FlowDev profile and returned exit 0 at that exact commit.

## Actions actually completed

1. Pinned the source, scope and original defect IDs.
2. Authored and structurally validated 50 stable acceptance steps locally; stored them alongside the original tracker. The first five are setup/diagnostic steps, not five newly accepted product features.
3. Ran `flowdev update dcm-quicksort` through the existing isolated profile on Lenovo; exit 0.
4. Ran `flowdev test dcm-quicksort host-diagnostics`; the read-only observation completed. At 2026-11-09T13:32:02Z, C: had 2,271,277,056 bytes free and available physical memory was 1,920,344,064 bytes. These improved observations justified one serial targeted run; they do not certify general host or product health. No user files were deleted and no unrelated processes were closed.
5. Retrieved the interrupted old summary: zero tests, timeout true, exit 1. It supplies no assertion RED. Then ran the current `regressions` selector: 21 top-level tests, 19 passed, 2 failed, zero skips, no timeout. Both intended failures are genuine native assertion REDs.

## Native evidence locations

- Updated source: `3f7784583b0ebc292ea1572373b6bcbea22cad44`.
- Host observation: `C:\FlowverseReports\dcm-quicksort-host-diagnostics-20261109-133202-a08b76ca\receipt.json`.
- Old timeout summary: `C:\FlowverseReports\dcm-quicksort-regressions-20261109-044942-5df3654c\app-regressions.summary.json` (zero tests; timeout true).
- New regression report: `C:\FlowverseReports\dcm-quicksort-regressions-20261109-133423-16576b74`.
- New report files reviewed: `app-regressions.summary.json`, `app-regressions.stdout.log`, `app-regressions.stderr.log` (stderr empty).
- Runner: registered DCM profile under `C:\FlowverseReports\dcm-quicksort-candidate\tools\flowdev-dcm`, `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports`, executable `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.

## Assertion failures reproduced

### DCM-06 / D2 - no-overwrite fast movement

`TestD2MoveFileRefusesExistingForeignDestination` failed: the move replaced a pre-existing destination without an explicit replacement transaction, removed the source, and changed the destination hash. The regression used disposable test files, not user documents.

### DCM-07 / D6 - duplicate recovery classification

`TestD6MovedDuplicateSurvivorRequiresAttentionAfterRestart` failed: a duplicate-removal record was mistaken for an ordinary rename chain after the surviving copy moved. Recovery returned history with no attention entry.

The earlier D1 identity checks, D2 guarded-retirement checks, D3 batch-journal checks, D4 template checks and other D6 fingerprint checks passed in this 21-test selection. This is not the full app suite, not installer qualification, not UI acceptance and not user acceptance.

## Next exact step

Start DCM-06: review `qualification/proposed-fixes/no-replace.patch` against the current source, make the smallest source change locally, store it on GitHub, update through FlowDev and obtain native GREEN. Then DCM-07 uses `qualification/proposed-fixes/duplicate-kind-gate.patch` after reviewing its context. The observed assertion RED prerequisite for both now exists. Neither proposal was applied in this checkpoint.

The local environment could not resolve github.com for a Git clone; GitHub connector reads/writes remained available. This did not justify editing code on Lenovo. Only plan/checkpoint metadata was authored locally in this turn; no product source was modified.

## Release boundary

No newly built app/setup, reviewed UI screenshots, user acceptance, merge or publication is claimed. v1.0.71 remains an unpublished candidate. Do not treat the 19 passing selected tests or the first five completed setup steps as release certification. Finish all applicable product, installer and user gates before publishing.
