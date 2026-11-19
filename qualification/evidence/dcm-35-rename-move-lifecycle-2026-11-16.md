# DCM-35 — rename/move and preview lifecycle acceptance checkpoint (16 November 2026)

## Result and decision

**PARTIAL — bounded gates passed, broad preview-to-operation gate still RED.** No source code was changed or reworked in this round. The verified source is `laxmanxp/dcm-quick-sort`, `release/v1.0.71`, commit `049b6cb21ba744041eb5ec7260046f8a62718414`; the source checkout stayed clean. The previous bounded real Windows.Data.Pdf close-in-flight case remains accepted separately in `dcm-35-pdf-close-inflight-2026-11-16.md`.

Required Lenovo FlowDev pipeline:
- `flowdev update dcm-quicksort`: **PASS**, already up to date.
- `flowdev test dcm-quicksort ui-tooling`: **PASS**.
- `flowdev test dcm-quicksort build`: **PASS**, exact build receipt `D:\FlowverseReportsRepo\dcm-quicksort-build-20261116-092226-1958df81\receipt.json`.
- `flowdev test dcm-quicksort ui-rename`: **PASS**, 8 cases, 5 captures, source unchanged and owned hosts closed. Receipt `D:\FlowverseReportsRepo\dcm-quicksort-ui-rename-20261116-092323-08c27077\receipt.json`.
- `flowdev test dcm-quicksort ui-move`: **PASS**, 6 cases, 5 captures, source unchanged and owned hosts closed. Receipt `D:\FlowverseReportsRepo\dcm-quicksort-ui-move-20261116-092336-304e63cf\receipt.json`.
- `flowdev test dcm-quicksort ui-previews`: **FAIL** after 17 passing cases out of 18 observed; stopped at `pptx-select-1`. Raw receipt `D:\FlowverseReportsRepo\dcm-quicksort-ui-previews-20261116-092348-0c424bb8\receipt.json`.
- `flowdev test dcm-quicksort ui-sumatra-lifecycle`: **PASS**, 11 cases, 10 captures, fixtures preserved, source unchanged, journal empty, no private viewers remaining, exit 0. Receipt `D:\FlowverseReportsRepo\dcm-quicksort-ui-sumatra-lifecycle-20261116-092554-d37c3593\receipt.json`.
- `flowdev test dcm-quicksort ui-sumatra-pending-close`: **PASS**, 3 cases, zero captures, fixtures preserved, source unchanged, no private viewers remaining, exit 0. Receipt `D:\FlowverseReportsRepo\dcm-quicksort-ui-sumatra-lifecycle-20261116-092700-afcf79d0\receipt.json`.

The Sumatra runtime used by both tests is a hash-verified pinned copy of `SumatraPDF.exe` (SHA-256 `719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0`) from an earlier disposable test report, not an installation or production source mutation.

## What the passing checks establish

- UI rename: ordinary disposable TXT selected, renamed with unchanged byte hash; completed durable rename journal entry; after host restart, the processed record can be selected and Undone, restoring original bytes and operation identity.
- UI move: ordinary disposable TXT selected and physically relocated to the accepted destination; source absence, destination bytes, completed journal, restart recovery and Undo restoration verified.
- Sumatra lifecycle: private runtime provider; PDF-A → PDF-B switching releases prior viewer; PDF → TXT frees both PDFs; ten immediate abandoned loads and four attached PDF/PDF/TXT sequences do not leave stale viewer/status or locks. Empty and invalid PDFs show errors and recover. Host close with active PDF drains private processes/profiles and file handles. This verifies exclusive file-open/read (no sharing), not a performed rename or move of those previewed PDFs.
- Pending-close: normal close during asynchronous Sumatra loading drains private resources, with normal exit and preserved fixtures.
- Previously accepted default Windows.Data.Pdf close-in-flight: natural worker exit, stale result discard, cache cleanup and exclusive release; separate evidence.

## Preserved RED preview suite — exact limitation

The `ui-previews` FlowUIVision run passed default PDF zoom/page assertions, then TXT, PNG, DOCX, XLSX and initial PPTX preview selection and second-slide observation. At `pptx-select-1`, the observed combobox had two entries, but FlowUIVision refused the selection because the requested label, containing garbled quotation/dash glyphs, did not equal its captured item text. Error: `combobox value ... not found`. No assertion of a PPTX image/render failure was recorded in this run. This is a **test-command/label mismatch hypothesis**, not a proven product bug; the entire selector remains **RED**, and no workaround, source edit, or test relaxation was applied on Lenovo. Preserve raw `pptx-select-1.vision.json` and receipt.

## What remains unproven

The parent DCM-35 criterion is **after switching/closing previews, owned workers and file locks are released so rename/move can proceed**. Separate successful normal rename/move and preview lifecycle tests do not fully prove *the same previewed file* can be renamed/moved immediately after switching away. Current evidence is strong but not equivalent to this cross-feature operation; acceptance must explicitly combine preview-open → release → rename/move of the same file, confirm SHA-256, and check no outstanding workers or stale locks. This remains **DCM-35 IN PROGRESS**, not complete.

Additional limits: no arbitrary Office provider process/COM failure or protected Office document acceptance, no end-user physical acceptance, no 50-step increment; DCM-34 remains separately partial. Retained machine receipts and visual flags must not be edited or recategorized, and the RED preview suite must not be presented as a pass.

**Suggested next development:** In ChatGPT's development environment add a focused registered `ui-preview-release-operations` FlowDev selector that uses an exact source-bound build and disposable files, performs selected PDF (and representative Office fallback) preview → TXT selection → exclusive open → native rename/move with hashes/journal and Undo, checking worker and cache cleanup. Unit/selector tests, commit, push, `flowdev update`, `flowdev test`, then bounded image review. Do not modify Lenovo source or restart normal XLSX/PPTX resizing.
