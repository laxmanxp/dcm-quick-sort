# DCM-46 — Integrated final-candidate normal workflow and image review (DESKTOP-6B6EH4T, 2026-11-17)

## Decision

**PASS — bounded engineering acceptance on the tested final candidate.** One continuous owned native QuickSort session completed **readable TXT preview → real PDF pages 1 and 2 → Rename → Move to configured Vault → normal close/restart → Undo Move → Undo Rename**, restoring the original filename, file content hashes, Vault contents, Processed state and durable operation identities. The workflow also captured an explicit empty/no-match filter state. Both PDF page renderings had independent nonblank pixel oracles.

**Visual/layout limitations remain:** the long source-folder path is horizontally scrolled in the source-path field, and long Processed transaction rows are clipped on the right at the captured window geometry. Those captures are not evidence that every layout/resolution/DPI case is fixed. The PDF preview shows the first portion of a document page in the viewport at Fit Width; scrolling remains separately qualified. No claimed physical user sign-off.

## Exact machine and source-bound FlowDev pipeline

- Device: **DESKTOP-6B6EH4T**. No Lenovo acceptance was used.
- Repository: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`; **exact tested pushed source `ef7bd87d8a55264ef682738d8f412c86bf4151ee`**.
- New integrated registered selector: `flowdev-test test dcm-quicksort ui-final-workflow`, mapping to `tools/run-ui.ps1 -Suite final-workflow`, registered in `tools/flowdev-dcm/config/checks.json`. This is acceptance test code; no QuickSort production app source was changed in this milestone.
- `flowdev-test update dcm-quicksort`: **PASS**, clean origin/branch checkout with source pin.
- `flowdev-test test dcm-quicksort ui-tooling`: **PASS**, 11 trace contracts and owned-process observer. Receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-tooling-20261117-035221-ddac49b4\receipt.json`.
- `flowdev-test test dcm-quicksort build`: **PASS**, exact-source build receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-build-20261117-035225-bd522c77\receipt.json`.
- `flowdev-test test dcm-quicksort ui-final-workflow`: **PASS**, exit code zero, **19 case records and 13 original screenshots**. Raw receipt `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-final-workflow-20261117-035254-97a5d2b2\receipt.json`; raw fields `passed=true`, `source_unchanged=true`, `all_owned_processes_closed=true`, `error=null`, `visual_reviewed=false`. The original receipt is deliberately not rewritten to claim an in-harness visual sign-off; screenshot review was done separately.
- All file operations used disposable report-root fixtures on E:.

## Direct file and rendering acceptance

1. The native file list selected `01-alpha.txt`, and an owned controls snapshot verified the visible preview text **`Exact bytes must survive Move and Undo.`**. The original SHA-256 was retained as the independent oracle.
2. The same session opened the disposable native `08-preview.pdf` with the real default PDF provider. Page 1 displayed `DCM PDF PREVIEW / PAGE ONE - TEAL`; the document-only pixels contained at least 100 actual teal samples. The owned Next Page control then showed page 2, with `PAGE TWO - ORANGE` and at least 100 orange samples. Both actual PNGs were separately reviewed.
3. Native Rename changed `inbox\01-alpha.txt` to `inbox\01-final-alpha.txt`, preserving original SHA-256. The durable journal recorded pending→completed under one specific Rename operation ID; the Processed pane displayed the renamed transaction. No filesystem rename shortcut was substituted for the app.
4. Native Move of the newly renamed file to `Vault\Accepted\01-final-alpha.txt` preserved source bytes, removed only the selected inbox entry and recorded a completed Move transaction under a different durable ID with saved destination SHA-256. Every unrelated inbox fixture remained hash-identical.
5. QuickSort was closed normally and launched again from the same exact build. The app displayed **`Journal recovery: restored 2 undoable move(s)`** and enabled Undo, while the durable journal retained both exact completed transaction IDs. The Processed date group was initially collapsed and is not a missing-history result.
6. The Move record was found through Processed search, selected via the native list control and explicitly Undone. `01-final-alpha.txt` returned to the inbox, disappeared from the Vault, retained its SHA-256, and the Move journal ID transitioned to `undone`.
7. The Rename record was then selected and explicitly Undone. `01-alpha.txt` returned with the original SHA-256, `01-final-alpha.txt` disappeared and its Rename journal ID transitioned to `undone`. All four baseline inbox fixtures were independently rechecked, and the Processed count became zero.
8. Finally, the Name filter was set to `ZZZ_DCM46_NOT_A_REAL_FILENAME`. The test verified zero matching file rows; the visible UI displayed **`No files match the current filters`** and **`Clear or change filters to show files`**, with no stale preview.

## Independent visual review

All 13 original PNG captures were retrieved and individually reviewed from DESKTOP-6B6EH4T:

- `launch-1`
- `dcm46-text-preview`
- `dcm46-pdf-page-one`
- `dcm46-pdf-page-one-document`
- `dcm46-pdf-page-two-document`
- `dcm46-rename-completed`
- `dcm46-move-completed`
- `launch-2`
- `dcm46-restarted-recovery`
- `dcm46-undo-move`
- `dcm46-final-restored`
- `dcm46-empty-filter`
- `final`

Observations: TXT and PDF document content are visible and not blank; the two PDF pages' colors/content differ as expected; Rename and Move status/messages correctly identify physical target paths; recovered Processed state is present after restart; after Undo there are no active Processed transactions. The explicit no-match state is helpful and readable. Rightmost Processed labels are cropped at the captured geometry; the long source path horizontally scrolls. These UI issues remain retained visual follow-ups and are not a substitute for later physical scaling/resize acceptance. No unexpected error dialogs appeared in this success-path suite; existing error-path evidence is maintained under other milestones.

## Preserved failed setup receipt

The initial tooling check `dcm-quicksort-ui-tooling-20261117-035125-de5864cd\receipt.json` failed before app launch because Windows PowerShell 5 parsed literal UTF-8 arrow symbols in the new test script with a legacy ANSI encoding, corrupting a history-row comparison. The source-controlled test was corrected by constructing the arrow using `[char]0x2192` at runtime. The entire corrected tooling/build/integrated suite then passed at pushed `ef7bd87d`. The failed receipt is preserved and is not reclassified.

## Qualification boundary and progress

DCM-46 is accepted **at the single tested window geometry, representative TXT/PDF fixtures and isolated native workflow**. This does not establish all Office native-provider conditions, every display scale/size, manual mouse/keyboard acceptance, all preview error states, crash/power-loss interruption, or universal layout approval. Keep `DCM-UI-RESIZE-01`, `DCM-UI-LABEL-01` where long rows are still clipped, and the retained historical DOCX incomplete-paint follow-up explicitly dispositioned before release; do not incorrectly reopen already accepted ordinary spreadsheet/presentation previews.

**Engineering tracker: 44/50 completed; 6 remain.** DCM-19 and DCM-34 remain separately partially blocked/limited, and physical user acceptance is not recorded.
