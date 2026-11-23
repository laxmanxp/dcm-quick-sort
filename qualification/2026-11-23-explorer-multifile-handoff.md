# DCM QuickSort — Explorer multi-file destination handoff (foundation)

Date: 2026-11-23
Branch: `feature/quicksort-two-pane-history-tabs`

## Implemented
- An explicit `--explorer-send-request <request.json>` startup entrypoint, distinct from the existing single-file and folder argument.
- Request schema 1: `{"schema":1,"alias_name":"Saved Favorite","files":["absolute path",...]}`. The whole selection enters QuickSort as one request, capped at 256 unique absolute file paths.
- Resolve the favorite name against current `destinations.conf` at invocation time; fail closed for unavailable/removed favorites.
- Reject directories, symbolic links, missing files and invalid paths before confirmation.
- Show the selected-file count, destination, sample filenames and Move-versus-Copy warning; cancel leaves files unchanged.
- Reuse `startApprovedBatchMove`, which maintains Batch ID, per-file collision handling, journal history and Undo Batch semantics.
- A standalone `tools/explorer-send-helper` launcher accepts a saved favorite name and selected absolute paths, writes a private request manifest and starts one QuickSort process.
- QuickSort removes only launcher-owned temporary manifests after reading them. No shell registry registration was changed.

## Lenovo verification
- Helper `go test ./... -count=1`: passed.
- Main QuickSort `go test ./... -count=1`: passed.
- Helper and app dev executables: built successfully.
- Source and destination user files were not modified during the tests.

## Not yet implemented / not qualified
- No dynamic Explorer submenu/COM shell extension registered or tested. `%1` is NOT a reliable method to capture multi-select; the existing installer shell entries remain unchanged.
- An approved disposable Windows Explorer test account/VM, real Explorer multi-selection handoff, visual confirmation/cancellation, conflict-resolution and Undo Batch GUI acceptance are still required.
- No silent move, source-folder deletion, or changes to the installed QuickSort package.

## Future Explorer layer contract
A multi-selection-aware Explorer command must collect the entire shell item array, show current destination favorites, and pass all selected absolute file paths in ONE invocation to the verified helper. It must not issue one process per file or claim acceptance from registry tests alone.
