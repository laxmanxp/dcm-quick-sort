# DCM-35 — same PDF preview release, Rename/Move and Undo (2026-11-16)

## Decision

**PASS — bounded, registered same-file cross-feature acceptance on Lenovo.** The previously missing direct path is now exercised: render the exact disposable `08-preview.pdf`, switch to TXT, exclusively open that same PDF, reselect and rename it with QuickSort's own UI, Undo, then repeat preview/TXT/exclusive access and Move/Undo of that exact PDF. Checks are source-bound, operation-journal bound, SHA-256 bound, and include native GUI interaction. This is **not** blanket qualification of other preview handlers or manual user acceptance. Parent DCM-35 remains in progress.

## Development and FlowDev pipeline

- Repository: `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.
- Added the registered `ui-preview-release-operations` suite to `tools/run-ui.ps1` and `tools/flowdev-dcm/config/checks.json` in GitHub. Changes to test tooling only; no production app source changed, no Lenovo production source edited.
- Tested pushed source: `605137afbeea33b04f54d7ac75a236ef1539e2ab`.
- Static source-contract validations checked the unique selector, independent file-lock oracle, both journal oracles, fixed original-file digest, and existing suite preservation before pushing.
- `flowdev update dcm-quicksort`: **PASS**, synchronized to `605137a`.
- `flowdev test dcm-quicksort ui-tooling`: **PASS**.
- `flowdev test dcm-quicksort build`: **PASS**, exact source receipt `D:\FlowverseReportsRepo\dcm-quicksort-build-20261116-102735-11b4d908\receipt.json`.
- `flowdev test dcm-quicksort ui-preview-release-operations`: **PASS**, exit code 0, 13 case records, 10 retained screenshot captures.
- Retained raw report: `D:\FlowverseReportsRepo\dcm-quicksort-ui-preview-release-operations-20261116-102855-92b21a3e\receipt.json`.
- The raw receipt reports `source_unchanged=true`, `all_owned_processes_closed=true`, no test error, and preserved original fixture bytes.

## Native file and journal oracles

**Rename cycle:** PDF rendering status showed the default `DCM PDF Preview`, Page 1/2. After selecting `01-alpha.txt`, the preview changed to normal TXT, and the original PDF opened with `FileShare.None`, demonstrating exclusive access rather than a status-only result. The PDF was reselected and QuickSort's native Rename control changed `08-preview.pdf` to `08-preview-released.pdf`. The renamed PDF SHA-256 equals the original. The durable journal recorded the pending→completed transaction; Undo restored `08-preview.pdf` with the identical SHA-256 and pending→undone journal identity.

**Move cycle:** The same restored PDF was rendered again, switched to TXT, and exclusively reopened. It was reselected and moved using QuickSort's native Move-to-Vault control to the disposable `Accepted` destination. The original source path disappeared, destination bytes matched the original digest, and the journal recorded completion. Native Undo restored the original PDF path and digest; the journal ended undone. After the final switch back to TXT, the restored PDF again opened exclusively. `Assert-Fixtures` verified the complete initial inbox set and empty final destination.

**Scope note:** To drive an unambiguous native Rename or Move, the suite reselects the PDF after verifying release on switching to TXT. It therefore checks a stronger operation with the same file selected and a settled preview, but it does not claim the UI can rename a nonselected item without selecting it. FileShare.None is an exclusive-access oracle, not itself a performed rename/move.

## Independent image review

All ten actual PNG files were individually read from Lenovo and visually reviewed:
`launch-1`, `pdf-before-rename`, `text-after-pdf-rename`, `released-pdf-renamed`, `released-pdf-rename-undone`, `pdf-before-move`, `text-after-pdf-move`, `released-pdf-moved`, `released-pdf-move-undone`, `final`.

Expected teal first-page PDF content and page navigation are clearly visible before both operations. After switching, the exact TXT fixture appears without stale PDF overlay. After Rename, the new PDF filename and Undo are visible; after Move, the original PDF is absent from source and the journal/Undo is visible. Subsequent frames show the restored `08-preview.pdf`; final TXT and original file set are visible. Renamed/Undone PDF screenshots show **transient Rendering PDF** while an asynchronous preview update is settling; this is an expected transition, not claimed as a settled rendering success. Stable preview content is evidenced by the two before-operation PDF captures. Raw captured visual flags, if false, were not rewritten; this report is the separate reviewer conclusion.

## Preserved limits

DCM-35's **default PDF same-file release→rename/move** path and prior PDF/Sumatra lifecycle gates pass. File locks for representative real installed Excel/PowerPoint/DOCX preview providers under their own active and close/switch conditions are not exhaustively proven by this selector. The older broad `ui-previews` gate remains historically RED at FlowUIVision's PPTX label mismatch; do not silently reclassify or conflate that with XLSX/PPTX rendering. User acceptance unrecorded. Keep DCM-35 **PARTIAL**, 32/50 complete, pending remaining representative non-PDF release and user acceptance scope. Do not retest already accepted normal XLSX/PPTX resizing absent a new defect.
