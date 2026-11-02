DCM - Quick Sort v1.0.62
Windows x64

MULTI-SELECT + BATCH MOVE
- The source File List now supports native Windows extended selection.
- Ctrl-click selects/deselects individual visible files.
- Shift-click selects a contiguous range.
- Ctrl+A selects all currently visible files.
- The focused/caret file still drives Preview and New Name.
- Rename remains a single-file action for the focused file.
- Move acts on every selected file when two or more files are selected.
- The Move button changes to "Move N Files" for a multi-selection.
- DCM asks for confirmation showing the batch size and destination before starting.
- Every file keeps its current filename during batch move.
- Conflicts are handled one file at a time using the existing duplicate/keep-both/replace rules.
- Every successful move gets its own durable journal record, Processed entry and Undo path.
- A skipped/failed item does not discard the remaining batch; DCM finishes with a moved/issues summary.

KEYBOARD-FIRST WORKFLOW
Alt+W  Focus New Name and select the filename stem
Alt+P  Focus Destination Path with the caret at the end
Alt+C  Focus Source Path with the caret at the end
Alt+L  Focus the source File List
F5     Reload the source list

MAIN WORKFLOW
Source | Source Selector | Source Location | Browse | Recovery | Reload
Name | File-name Filter | Open Folder
File Type | Extension | Age | Size | Sort By | Clear Filters

Rename | Move / Move N Files | Next | Open | Handoff to FileFlow

PREVIEW
Markdown (.md) and CSV (.csv) are always shown as plain text. PDF/image/Office preview behavior is retained.

PROCESSED HISTORY
Processed remains searchable and date-grouped/collapsible. Batch moves appear as ordinary independent transactions, so Undo and Recovery stay granular.

Qualified app SHA-256:
53ed5f61ecde3df6b2d3cd5dd8a5a08b25209ec234ef715cd05a0f6359fb5e21

Qualified installer SHA-256:
47e7e8314de69ac2ece45085e9d8854678405e7c83fe1285e29d83fb814d6533
