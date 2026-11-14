# DCM-31 — representative XLSX preview accepted

Recorded 14 November 2026 (Asia/Kolkata). DCM QuickSort v1.0.71,
branch `release/v1.0.71`.

**Tested production source: `ec7a02349e9f53c6fcfd889cf21f800f36c68b4a`.**

This application completion record is being finalized from retained evidence.
Its own documentation commit, push and Lenovo synchronization were not yet
observed when prepared; the acceptance below belongs to the tested production
source, and does not claim those later record-delivery actions. Confirm them
before reporting the production task DONE. The separate evidence bundle is
confirmed in GitHub report commit `69fee755a94468165018b499fe7d876779e57ec8`.

## Result

DCM-31 is complete for the representative workbook workflow through Lenovo's
installed Microsoft Excel preview handler. The first-open clipping defect,
`DCM-XLSX-SIZE-01`, is fixed within the qualified scope. The worksheet fills its
preview pane on first open and reopen, with Orders/Read me tabs and both
scrollbars visible. The normal workflow also passes without diagnostic tracing.

The checklist is **30 complete / 20 remaining**. Complete IDs are DCM-01–18 and
DCM-20–31. DCM-19 remains blocked on a disposable Windows Explorer integration
environment; the next runnable step is **DCM-32 — verify PPTX preview**. These are
checklist counts, not a percentage of product development or human acceptance.

## What changed and why

The observed defect was a mismatch between the Excel outer viewport and its
already correctly sized inner worksheet. Before the repair, a 902×487 physical
client pane contained an outer `EXCEL9` window of only 723×390, while its inner
`EXCEL7` was 903×488. The first view lost its right and bottom areas, including
tabs and scrollbars. Diagnostic captures showed that waiting did not resolve it.

The app keeps logical rectangles for the preview handler's COM `SetWindow` and
`SetRect` calls. After successful Excel `SetRect`, it makes one guarded positioning
request for the outer `EXCEL9` window. It verifies the owned host pane, exact
unique direct EXCEL9/EXCEL7 hierarchy, child styles, matching renderer PID,
compatible DPI contexts, positive client dimensions, and unchanged identities
immediately before the request. The caller's DPI context is restored and checked.

The policy is enabled only for the parsed Excel preview CLSID
`{00020827-0000-0000-C000-000000000046}`. The active handler, pane and policy are
revalidated after COM returns, and unloading disables repair before cleanup.
The request preserves activation, visibility and Z order, does not resize the
inner worksheet, and does not reparent or send input to the provider. Its async
flag posts when Windows identifies different input queues; an accepted request
alone is not treated as proof of completed rendering. The feature gate measures
the resulting outer and inner windows and separately reviews actual images.

The extra initial resize is Excel-only. Other providers retain their previously
qualified initialization behavior. No Office associations, system DPI settings,
installer behavior or preview-provider installation were changed.

This is a measured compatibility repair for the observed host/provider boundary.
The evidence does not establish Excel's internal cause. Earlier uniform physical
COM rectangles fitted the outer window but oversized the inner one to 1128×609;
that failed policy was reverted and is not part of the accepted implementation.
The full chronology remains in [the sizing investigation](dcm31-preview-sizing.md).

## Pipeline and exact source identity

Development and local verification occurred in ChatGPT's development environment.
The candidate was committed and pushed before Desktop Commander updated Lenovo.
Both checkouts and FlowDev registration identify
`https://github.com/laxmanxp/dcm-quick-sort.git`, branch `release/v1.0.71`.
Production source is `app-src/`; installer source is `installer-src/`.

The registered commands, in order, were:

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort excel-preview-layout-unit
flowdev test dcm-quicksort build
flowdev test dcm-quicksort ui-xlsx-layout
flowdev test dcm-quicksort ui-xlsx
flowdev test dcm-quicksort ui-docx
flowdev test dcm-quicksort baseline
```

Every command passed at the source above. Each UI test used the exact clean-source
build receipt through `DCM_BUILD_RECEIPT`; the native tests did not edit production
source. These are isolated candidate tests, not an installed-package acceptance.

| Identity | Recorded value |
|---|---|
| Lenovo checkout | `D:\FlowverseReportsRepo\dcm-quicksort-candidate` |
| FlowDev profile | Checkout's `tools\flowdev-dcm` |
| FlowDev runtime | `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe` |
| Native build | `dcm-quicksort-build-20261114-011936-e3a17b9f` |
| Native executable | 7,365,632 bytes; Go 1.27.0 Windows/amd64 |
| Native executable SHA-256 | `7f51956e67dd7e0a33eef51a6ad8cf8a742ad6e2d64585348614eaa485e35827` |
| Tracked-source manifest | 434 files; SHA-256 `27cdc9c9ffb27681e81f94427d1b0412a7dff102d72ee250e39fb20fccca06a9` |

## Tests and reviewed feature evidence

| Gate | Actual result | Native report suffix |
|---|---|---|
| Local XLSX/Open XML units | 24 passed, 0 failed, 0 skipped | Local Go 1.27.4 Linux/amd64 |
| Local DOCX/font units | 7 passed, 0 failed, 0 skipped | Same candidate source |
| Windows application and helper tests | Cross-compilation passed; Windows binaries not run in Linux | Local compilation evidence |
| `excel-preview-layout-unit` | 2 top-level tests; 9 geometry and 39 guard cases passed; 0 skips | `20261114-011840-a131e6b7` |
| `ui-xlsx-layout` | 6 workflow records, 10 strict geometry checks; all 12 images reviewed | `20261114-012003-6ce3d109` |
| Ordinary `ui-xlsx` | 6 workflow records, 2 strict geometry checks; all 4 images reviewed | `20261114-012116-fbdc6385` |
| DOCX regression `ui-docx` | 7 workflow records; all 5 images reviewed | `20261114-012139-efd939a9` |
| Full Windows baseline | **383 passed, 0 failed, 1 skipped**; exit 0, no timeout | `20261114-012309-4ee29c99` |

The single baseline skip is `TestRealGeneratedPPTXIntegration`, because its
external fixture remains unsupplied. It is not counted as a pass and remains
under DCM-32/47. The new helper tests used actual Windows HWNDs at a measured
125% scale: 320×240 logical client units corresponded to 400×300 physical units.
The exact 48 cases and test source are unchanged from the clean RED at
`95cab1480ef04dec5d8dac44b6ba4ca436163f72`: nine intended geometry failures and
39 guard passes. The earlier popup-owner fixture setup failure is preserved
separately and was not represented as a clean RED run.

### XLSX observations

The synthetic 6,744-byte workbook contains Orders and Read me worksheets and no
formulas. Its SHA-256 is
`5059729f4390ecd2e169bb2d1eb850c768532ebbb24b9733e87b82b012d6d24f`.

All ten diagnostic workbook frames, including both captures before Restore and
the natural adjacent captures, show a fitted viewport. Both EXCEL9 and EXCEL7
measure 903×488 against a 902×487 client pane, with the unchanged four-pixel
size/origin tolerance. Both ordinary workbook frames independently pass the same
bounds. This rejects undersizing and oversizing; the upper bounds were not relaxed.

The images show the office/order/date headers, Sample Office rows, sequential
order numbers such as 0001, and the date 26-09-2026. Orders is selected, Read me
is visible, and horizontal/vertical scrollbars are within the pane. Later columns
extend naturally beyond the viewport; their full content is not claimed from
these captures. Initial text and recovered text match the alpha fixture, and
the provider footer correctly changes between TXT and XLSX.

The diagnostic trace contains 14 completed records across two Excel sessions.
Both `SetWindow` and `SetRect` receive logical 722×390 rectangles and return S_OK.
Both outer-repair requests are accepted, with observed caller/pane context 24592
unchanged before/after. The trace is below its 128-record cap. Actual geometry and
image review, rather than the BOOL trace result, establish the rendering outcome.

The workflow verifies workbook release after switching to text, original fixture
names and bytes, empty destination/journal, normal host exit 0, and unchanged
committed source. It respects the provider's programmatic-access refusal: actual
cell-content findings come from image inspection, without bypassing that refusal.

### DOCX regression observations

The installed Word provider still displays the representative letter, formatted
structured document with its table and green image, text recovery, and reopened
letter. All five frames show the intended content and complete host controls.
Both DOCX files are released after switching away; original bytes, source and
journal invariants pass. The shared Word process is not terminated or claimed to
have exited by this gate.

## Evidence preservation and limits

Separate review decisions are in
[the native summary](evidence/dcm-31-native-summary.json). Raw machine receipts
retain `feature_accepted=false` and `visual_reviewed=false`; they were not rewritten
to manufacture image review. The initial visual failure, later rejected trials,
clean RED, and final GREEN remain distinguishable.

The durable private evidence bundle is
[`flowverse-reports/apps/dcm-quicksort/dcm31-20261113-225138/outer-layout-fix-ec7a023`](https://github.com/laxmanxp/flowverse-reports/tree/69fee755a94468165018b499fe7d876779e57ec8/apps/dcm-quicksort/dcm31-20261113-225138/outer-layout-fix-ec7a023).
It preserves the 21 hash-verified current PNGs and local verification logs. The
316,274-byte exact native evidence ZIP contains 131 byte-verified text files from
the six accepted native reports, including raw receipts, test logs, source
manifests, all captured geometry and the COM trace. Its SHA-256 is
`8a403965beb5357b0114bddb3819376293be3c82b037a938849bdbacd9634ac9`.
Readable exported copies are labeled separately from these exact native bytes.

This gate covers the demonstrated representative workbook on the installed
Excel provider and observed Lenovo scaling. It does not certify native sheet
navigation, arbitrary Excel fidelity, protected/damaged files, general rendering
on all DPI/multi-monitor configurations, or headed fallback behavior without
Office. The bounded built-in worksheet renderer has focused unit coverage;
its limits include 500 rows, 64 columns, bounded cell text and cached formula
values without recalculation. Remaining preview errors and broader cleanup retain
their own gates. DCM-PREVIEW-01 remains open overall.

The separate incomplete-host-paint observation `DCM-UI-RESIZE-01` remains under
DCM-46; a clean DOCX regression does not claim to resolve it. Human acceptance
remains `not_recorded`, the final package is unqualified, and release is unpublished.
Later documentation-only commits must preserve the exact tested source above;
future production changes require a fresh build receipt before UI acceptance.

## Smallest next step

Begin DCM-32 by recovering and identifying the existing two-slide PPTX fixture,
then run its unchanged parser assertions through a focused committed FlowDev
selector. Qualify visible slide text/images afterward. Do not reinterpret the
current external-fixture skip as passing coverage or expand this completed XLSX
step with unrelated work.
