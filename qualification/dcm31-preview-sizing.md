# DCM-31 — XLSX content renders; first-open sizing remains open

Recorded 14 November 2026 (Asia/Kolkata). Tested source:
`3d3af0fc3d67effdd7184251deb4ff234008dc1c`, branch `release/v1.0.71`.

**DCM-31 is PARTIAL. The tracker remains 29 complete / 21 remaining.**
The exact-source FlowDev workflow passed six machine records, but review of all
four full-window images found the initial Excel preview clipped to roughly 80%
of the pane width and height. A later reopened view fills the pane. Reopening
does not establish a repair; the first-open observation is `DCM-XLSX-SIZE-01`.

## Completed pipeline and evidence

The correct repository is `laxmanxp/dcm-quick-sort`; production is `app-src/`.
The registered Lenovo profile at
`D:\FlowverseReportsRepo\dcm-quicksort-candidate\tools\flowdev-dcm` maps to that
repository and branch. Tooling was developed locally, reviewed, committed and
pushed before Desktop Commander ran these registered commands:

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort xlsx-unit
flowdev test dcm-quicksort ui-tooling
flowdev test dcm-quicksort build
flowdev test dcm-quicksort ui-xlsx
```

| Gate | Recorded result |
|---|---|
| Local XLSX/Open XML units | 24 passed, 0 failed, 0 skipped; Go 1.27.4 Linux/amd64, source `45f86d9` with unchanged relevant files |
| Lenovo `xlsx-unit` | 24 passed, 0 failed, 0 skipped; Go 1.27.0 Windows/amd64 at `3d3af0fc` |
| Lenovo `ui-tooling` | 27 scripts parsed; new worksheet helper compiled |
| Native build | PASS; source unchanged; 426-file manifest |
| Native `ui-xlsx` | Six records passed; four images reviewed; first-open image fails complete-pane presentation |

Native build: `dcm-quicksort-build-20261113-225251-10a6593e`.
EXE: 7,336,960 bytes, SHA-256
`8c682baab23b4ea831314ae0ff1bc2c9a98a1180d05365fb6968014bee9f9ef5`.
This is byte-identical to the prior DCM-30 executable; no app or installer source
changed after `8eb1470a891ea3454f1738d58191fdb90284ab42`.
Manifest SHA-256: `47f62f19636b6386c605b789d697149073643d3303fcb576360879897b20b343`.
The complete manifest was recovered and its hash verified; all 426 entries match
the committed checkout, including its Windows CRLF conversion.

UI run: `dcm-quicksort-ui-xlsx-20261113-225846-094bf40a`,
22:58:46–22:59:02 UTC on 13 November. The selected provider was the installed
Windows Excel preview handler. Office associations were observed, not changed.

## Reviewed images

| Image | Observation |
|---|---|
| `01-initial-text.png` | Correct alpha fixture and TXT footer; complete host controls. |
| `02-native-workbook.png` | Actual office/order/date headers, Sample Office and sequential order numbers are visible. Excel occupies approximately 80% of the pane in both axes, leaving unused white right/bottom bands. Dates are clipped; sheet tabs and scrollbars are absent from the captured area. |
| `04-text-recovery.png` | Correct alpha text and TXT footer return; no stale worksheet body. |
| `06-native-workbook-reopened.png` | Excel fills the pane. Orders and Read me tabs, scrollbars, order numbers and date 26-09-2026 are visible. Recipient columns extend horizontally beyond the viewport. |

Native accessibility returns a programmatic-access refusal for this preview,
so cell-content findings come from direct image inspection. No attempt was made
to bypass that refusal. Native sheet navigation was not exercised. The source
workbook is synthetic, contains no formulas, and is pinned to SHA-256
`5059729f4390ecd2e169bb2d1eb850c768532ebbb24b9733e87b82b012d6d24f`.

The raw receipt remains `feature_accepted=false`, with unchanged
`visual_reviewed=false` capture flags. Separate review is recorded in
[the initial summary](evidence/dcm-31-initial-summary.json). Files, destination,
journal, exact source and original workbook bytes were preserved. The host exited
normally with code 0; the workbook allowed exclusive read access after switching
away and after closing. This does not claim termination of every Office process.

## Investigation boundary

Pre-capture geometry differs between the Excel outer and inner windows:
`EXCEL9`/`EXCEL7` are 578x312/722x390 on first open, then 722x390/578x312 on reopen.
Those snapshots precede accessibility, restoration and capture, so they cannot
identify the geometry in the captured frame. The approximately 0.8 ratio makes
DPI virtualization plausible; asynchronous initial sizing is another hypothesis.
Neither is diagnosed yet. The host shell is fully painted in these four images;
do not equate this native-child observation with the older host-paint follow-up.

The next registered check, `ui-xlsx-layout`, preserves the ordinary raw capture
and adds bounded before-restore/natural adjacent images, paired window geometry
under explicit DPI contexts, and the existing host layout trace. It sends no
native Excel input or forced redraw. A later clean frame is diagnostic evidence,
not a product fix. Keep DCM-31 open until the first-open defect is explained and
resolved or explicitly dispositioned with adequate evidence.

Microsoft's [preview-host sequence](https://learn.microsoft.com/en-us/windows/win32/shell/preview-handlers)
and [SetRect contract](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nf-shobjidl_core-ipreviewhandler-setrect)
describe setting and updating the preview area. Its
[GetWindowRect documentation](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getwindowrect)
states that window rectangles are DPI-virtualized. These contracts guide the
diagnostic; they do not prove the cause of this particular frame.

## Bounded diagnostic result

The registered `ui-xlsx-layout` run
`dcm-quicksort-ui-xlsx-20261113-231011-787600e7` at `b437ba2a6604bce82090d1b7f24ca9b0cb2851bb`
retained twelve images and 24 paired geometry observations. Production bytes are
unchanged. Initial and reopened views remain clipped before Restore, in the raw
frame and after natural adjacent captures. The initial raw pre-capture observation
is 23:10:22.9657292 UTC; the last natural post-capture observation is
23:10:28.3989448 UTC, approximately 5.43 seconds later. This is not explained by a
brief capture immediately before the viewer settled.

In the PMv2 observer context, the owned pane client is 902x487, the Excel outer
`EXCEL9` window is 723x390 and its `EXCEL7` child is 903x488. The host-context
measurements are 722x390, 578x312 and 722x390 respectively. Host sequence remains 2,
depth 0 and host pending-paint false in the inspected paired observations. All
three windows report DPI 96 / DPI-unaware. The outer viewer therefore clips an
otherwise larger inner view. These measurements identify the sizing mismatch;
they do not prove every internal Excel initialization cause.

The registered `ui-xlsx` and `ui-xlsx-layout` selectors now require a geometry
assertion for the observed installed Excel route: both native viewport windows
must match the pane client dimensions within four pixels, with the expected
parent relationship and border-aligned origin. The gate retains capture,
recovery and normal-close evidence before reporting failure. Cell readability
still requires separate image review. Establish this new assertion's native RED
before changing production initialization. No numbered step closes here.

## Coverage limits and durable material

The fallback parser/grid has focused unit coverage for blank/sparse sheets,
shared/inline strings, dates, cached formulas, bounds and malformed packages.
This installed-handler run does not certify the fallback's headed behavior.
The fallback is a bounded grid (500 rows, 64 columns, 24 displayed characters per
cell); it does not recalculate formulas or reproduce general Excel layout.
Protected/damaged and broader cleanup gates remain DCM-34/35. Human acceptance,
final package and release publication remain pending.

The evidence bundle is
[`flowverse-reports/apps/dcm-quicksort/dcm31-20261113-225138`](https://github.com/laxmanxp/flowverse-reports/tree/main/apps/dcm-quicksort/dcm31-20261113-225138).
It retains the exact four PNGs, synthetic workbook, raw readbacks, local/native
unit output, build/tooling receipts, exact source manifest and separate review.
JSON/text exports may have normalized line endings; PNG/fixture/manifest hashes
are verified. The next numbered step remains DCM-31; no numbered step closes here.
