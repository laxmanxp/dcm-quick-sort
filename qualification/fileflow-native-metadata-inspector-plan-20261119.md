# DCM QuickSort — native FileFlow metadata inspector integration plan

**Date:** 19 November 2026
**Target candidate:** `release/v1.0.71` (not the published installer)
**Status:** DESIGN / NOT IMPLEMENTED / NOT QUALIFIED

## Objective
Provide QuickSort users the same file-associated Tags, Notes and Todos CRUD as FlowSearch, while keeping QuickSort's primary **Preview -> Rename -> Move -> Next** workflow and independent native window. TaskFlow is a separate application and is **not** a dependency of FileFlow Suite.

## Reference implementation
FlowSearch native editor: `laxmanxp/fileflow-suite`, `classic/metadata_editor_windows.go`, feature branch `feature/classic-search-two-pane-preview`; tabbed inspector acceptance `classic/FLOWSEARCH-METADATA-TABBED-ACCEPTANCE-20261119.md`.

Source of truth remains **FileFlowService** metadata operations, not file paths alone and not a second QuickSort database. The prototype's browser `localStorage` behavior is demonstrative only and must not be introduced to native applications.

## Proposed QuickSort implementation
1. Identify the canonical `release/v1.0.71` UI source and existing file-selection command handling. Keep production installer and installed service unchanged.
2. Add an unobtrusive **Metadata** action for the currently selected incoming file; show **Tags / Todos / Notes** in a separately owned native inspector or collapsible pane. Never displace sorting/navigation shortcuts or preview controls.
3. Reuse the FileFlowService wire contract and validated entitlement/authorization behavior from FlowSearch. For an old, unavailable or unverified provider, show an explicit unavailable/conditional state; do not fabricate Pro entitlement or fall back to a private metadata DB.
4. Resolve logical file identity through FileFlowService. After a successful QuickSort rename or move, refresh metadata using the resulting path/logical identity. Do not write metadata optimistically during an uncommitted move, replacement or undo.
5. Tags: add/rename/remove through the current `SetTags` authoritative set; preserve server-side normalization. Todos: add/update/complete/reopen/delete. Notes: add/update/delete. No priority or deadline fields until the service contract actually supports them.
6. Preserve drafts during reload/unrelated writes; provider-confirmed write, reload and error states must be truthful. Do not weaken destructive file-operation safeguards or changes required by outstanding DCM-34 issues.
7. Isolated acceptance: create a single test file; save tags, a note and a todo in QuickSort, confirm they appear in FlowSearch for the **same logical file**, then update them in FlowSearch and re-open from QuickSort. Test move/rename/undo identity, cancellation, Free/Pro/missing-provider and source-file bytes/ACL invariance.
8. Keep QuickSort UI test receipts and release gates separate from FlowSearch's acceptance. Do not certify or publish a new DCM release by analogy.

## Open dependency
The independent QuickSort candidate remains subject to its recorded source-bound qualification and release safety gates. This document does **not** assert that the native inspector is present in QuickSort.
