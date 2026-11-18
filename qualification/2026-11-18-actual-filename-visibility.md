# DCM Quick Sort — visible actual filename and file type

**Device:** Laxman_Lenovo (18 November 2026)
**Branch:** `feature/shared-flowpdf-pdfium-provider`
**Scope:** development only; no installed/release build changed.

## User-facing change

The existing New Name field intentionally edits only the filename
**stem**, preserving the source extension. The file list can clip
longer names and its preview header competes with toolbar controls.

A dedicated row between the preview and New Name fields now shows:

- **Actual File:** read-only copyable Windows edit control, with the
  real on-disk basename *including its original extension*.
- **Type:** separate always-visible extension indicator (e.g.
  `Type: PDF`, `Type: DOCX`, `Type: XLSX`, `Type: TXT`).
- **New Name:** remains an editable extension-free stem, as before.

The preview header is now simply `PREVIEW`; filename identity
is in the separate lower row. The preview gives up 34px of vertical
space so the existing New Name, destination, action and status rows
keep their old positions. The type indicator remains independently
visible even when a long filename cannot fully fit in the edit field.

The actual-name row updates through the existing selection/reload
lifecycle, including Rename/Undo, Move/Undo, changing source folder,
filtered/empty source states, and opening another document. The
display is read-only, so it never becomes an alternate rename input.
Source files are not modified to display this information.

Implementation:
`app-src/main_windows.go`,
`app-src/actual_filename.go`,
`app-src/actual_filename_test.go`.

## Native Lenovo acceptance

Evidence:
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261118-141235-782c978\receipt.json`.

**PASSED=True, NORMAL_EXIT=True, process exit 0.**

The isolated native UI test verified:

| Actual File value | Fixed indicator | Editable New Name |
|---|---|---|
| `08-preview.pdf` | `Type: PDF` | `08-preview` |
| `01-alpha.txt` | `Type: TXT` | `01-alpha` |
| `undo-preview.pdf` after Rename | `Type: PDF` | `undo-preview` |
| `13-Letter.DOCX` | `Type: DOCX` | `13-Letter` |
| `14-Budget.xlsx` | `Type: XLSX` | `14-Budget` |
| No file in source folder | blank | blank |

The test additionally confirmed the empty state reads
`No file selected`; the actual filename field has the Windows
`ES_READONLY` style, the type indicator does not overlap it,
and the filename row is above—not overlapping—the editable New Name
row. The field measured 995px in the tested window. The PDF
Rename/Undo and Move/Undo source bytes and journal integrity passed.
The healthy PDFium worker served 19 renders on one PID, without
unexpected one-shot fallback, and shut down normally.

**Limitations of this run:** the host desktop was shared with another
foreground application. The native screenshot guard correctly
refused obstructed captures; the final focused filename UI acceptance
used `-SkipScreenshots -SkipPointerWheel` to avoid disturbing other
windows. It exercised actual native control text, layout, read-only
style, selection and file lifecycle instead of claiming screenshot
acceptance. Native PDF pointer-wheel anchoring had been tested in
separate earlier receipts.

Initial test failures were unrelated to the feature itself:
PowerShell's `Wait` helper shadowed a local `$name` variable,
and a foreground-sensitive wheel/capture assertion was triggered
while other application windows occupied the desktop. The test was
fixed/scoped; the final native filename-specific acceptance passed.

A pre-existing full-suite legacy-worker compatibility defect was
also uncovered: default zero-degree PDFium rendering was passing
`-Rotation 0` to a worker predating rotation support. The provider
now omits this optional argument for zero rotation, retaining old
one-shot worker support while preserving nonzero rotation and the
Windows.Data.Pdf API's dedicated rotation parameter.

## Release decision

Keep changes on the development feature branch pending normal
promotion procedures. The production-default renderer remains
Windows.Data.Pdf. No user files or installed applications were
changed during this work.
