# DCM-34 — preview failure and interruption states

Started 14 November 2026 (Asia/Kolkata), standalone DCM QuickSort v1.0.71,
`laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.

**PARTIAL. The numbered plan remains 32 complete / 18 remaining.** DCM-33
is complete at its recorded representative scope; DCM-19 remains blocked on
a disposable Windows Explorer environment. Physical user acceptance and release
publication remain open.

## First focused milestone: default PDF damage and recovery

Starting completion record: `9afb15b120befcf11aba5aa5b83e4ab39e598934`.
Existing DCM-26/27 evidence already covers text and image errors/recovery.
Existing Sumatra damage/password/lifecycle results apply to that opt-in provider.
The next missing headed slice is a damaged PDF through the default
Windows.Data.Pdf renderer and its actual Windows-handler fallback policy.

The new focused protocol tests cover blank/failed output, unusable page metadata
and usable legacy/current results. Local `^Test(DCM34PDFRenderer|PDF)` passes
**14 top-level tests, zero failures/skips**. Three new parent tests contain
23 subcases: 6 failed-output cases, 14 invalid-metadata cases and 3 success
cases. Together with 11 standalone existing PDF tests, the run has 34 terminal
leaf tests. These pure output-contract tests do not open PDFs in WinRT.
The Windows test binary also cross-compiles locally; execution is a separate
registered native gate.

Planned headed sequence: exact TXT baseline, known valid PDF, header-valid
damaged PDF, exact TXT recovery, valid-PDF recovery, damaged-PDF reopening,
final TXT recovery, exclusive file release and normal close. The real damaged
fixture must reach the decoder. A native Windows fallback, if attached, must
retain its own observed error/content and control state; an attachment or footer
alone is not visual acceptance. Original frames and same-frame crops need a
separate semantic review. Source/fixture bytes, destination and journal are
checked independently.

The focused FlowDev sequence is:

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort pdf-failure-unit
flowdev test dcm-quicksort ui-tooling
flowdev test dcm-quicksort build
flowdev test dcm-quicksort ui-pdf-failure
```

Set `DCM_BUILD_RECEIPT` to the successful exact-current-source build receipt
before the UI gate. Use the registered profile and Desktop Commander described
in `CONTEXT.md`. Run only one headed/native qualification at a time.

**Native acceptance for this new milestone is pending.** No production renderer
change or default-PDF defect is asserted by the local protocol checks.

## Initial headed attempt retained: TXT colour-oracle false positive

Published candidate `499b015ee11c3554f65c3a1bc9368888c16542fc` passed the registered
update, native PDF units (14 top-level tests, zero failures/skips/timeouts),
`ui-tooling` and exact-source build. An unrelated FlowDev test started after
the PDF units, so the sequence stopped at its concurrency guard and resumed
with the remaining gates after the other test finished; the units were not
rerun.

UI attempt `171254-482555d5` failed before any PDF selection, with zero accepted
cases and four retained captures. The original full frame shows the three
correct TXT lines and no PDF content. Its 905 x 490 preview crop contains 274
teal samples and 89 orange samples in the text rendering; the harness's
absolute 100-sample colour threshold incorrectly rejects the clean baseline.
The failure and original flags remain unchanged. Source, fixtures and registry
were preserved, the journal was empty, and the host closed normally with exit 0.

The correction uses the verified initial TXT image as the recovery baseline,
retaining exact text, control, ownership and provider checks. Recovery crops
must match that baseline within the existing tolerance of 30 changed sampled pixels.
The valid-PDF and damaged-PDF requirements are unchanged. This is a harness
correction, not a production PDF defect or a successful damaged-file run.
A newly pushed correction needs its own update, tooling, build and UI gates.
The unchanged Go units retain their `499b015` source attribution.

## Separate confirmed DOCX fallback defect

`DCM-34-DOCX-FALLBACK-01` is **open and unfixed**. A deterministic local probe
against unmodified `9afb15b` records 2 parent tests (1 pass / 1 fail) and 23 leaf
cases (13 pass / 10 fail / 0 skip). Empty main XML, a missing document element
and a missing body element incorrectly produce successful zero-body DOCX
previews and successful foundation metadata. Malformed XML is correctly
rejected by `renderDOCXPreview` and `docxPreviewText`, but the subsequent
`officeFoundationPreviewText` returns success with `Open XML package validated.`

The valid blank-document and missing-part guard controls pass. All five
subsequent valid-DOCX controls recover their expected heading and table. This
is function-level fallback evidence, not a claim about the installed Word
handler or headed Windows recovery. The diagnostic failing tests are retained
as report evidence rather than inserted into the accepted application's test
suite. See [the probe summary](evidence/dcm-34-docx-fallback-probe.json).

The smallest DOCX fix is to prevent a parse failure from regaining a success
presentation through the metadata-only fallback, while preserving valid blank
documents and the native-first preview policy. Handle that in a separate
TDD commit and its own source-bound acceptance.

## Remaining DCM-34 scope

- Default PDF protected-file behavior and recovery.
- Observed default-renderer loading interruption: change selection while
  loading, then prove late results cannot repaint the new preview/status.
- Resolve the confirmed DOCX fallback defect and document remaining advertised
  Office damaged/protected-handler coverage or limits.
- Keep worker/file-lock lifecycle qualification under DCM-35 and the final
  candidate/package rechecks under DCM-46/47.

A completed damaged-PDF milestone alone will not close DCM-34 or change the
numbered count. Historical DOC/XLS/PPT acceptance, original failures and the
388/0/0 baseline at `b3d46b9` retain their original source attribution.
