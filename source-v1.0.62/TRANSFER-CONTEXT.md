# DCM - Quick Sort transfer context — v1.0.62

## Current checkpoint
- Version: **1.0.62**
- Architecture: native Go Windows x64 app + native Go installer
- Base: v1.0.61 keyboard-first focus/cursor workflow

## v1.0.62 changes
1. **Native multi-selection in the source File List**
   - File list uses Windows `LBS_EXTENDEDSEL`.
   - Ctrl-click selects/deselects individual files; Shift-click selects a range.
   - `Ctrl+A` selects all currently visible files.
   - The focused/caret item remains the Preview/New Name item.
2. **Batch Move**
   - Two or more selected files make Move operate on the complete selection.
   - Move button shows `Move N Files`.
   - A confirmation dialog shows the selected count and destination before the batch starts.
   - Each file keeps its current filename; the New Name edit is deliberately not applied by Move.
3. **Per-file safety retained**
   - Batch files are processed sequentially through the existing destination-conflict inspection, SHA-256 duplicate check, durable journal, safe replace and background copy/move paths.
   - Every successful item creates its own Processed transaction and remains individually undoable.
   - Exact-duplicate, keep-both and replace behavior remains per file.
   - Skipped/failed items count as issues but do not discard the remaining batch.
4. **Selection UX**
   - Count header reports `N selected` for multi-selection.
   - Bottom hint strip now advertises Ctrl/Shift selection and Ctrl+A.
   - Rename is intentionally still single-file: it acts only on the focused item.

## Qualification
- TDD RED first for batch planning and Windows multi-selection/batch wiring.
- App tests/race/vet pass.
- Windows x64 test binary, Windows vet and GUI build pass.
- Installer tests/vet/Windows build pass.
- Embedded installer payload exactly equals the qualified app bytes.
- Final packaged-source reproducibility check is required after packaging.

## Live Windows acceptance still useful
- verify Ctrl-click and Shift-click selections at the real Windows UI;
- verify Ctrl+A selects all visible filtered files;
- verify Preview/New Name follows the focused file without collapsing the multi-selection;
- verify button changes to `Move N Files`;
- batch-move several disposable files to another folder and verify all successful files create separate Processed entries;
- exercise one filename conflict inside a batch and confirm later files continue;
- restart DCM and verify Recovery contains no false attention entries for completed batch moves.
