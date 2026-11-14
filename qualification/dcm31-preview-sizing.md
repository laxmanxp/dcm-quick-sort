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

## Native assertion RED and candidate correction

The strengthened registered `ui-xlsx` test at
`45679e0380ab9ee2d912203f64469239d5864068` ran as
`dcm-quicksort-ui-xlsx-20261113-231926-395a05c1` and exited 1 with
`DCM31_XLSX_NATIVE_BOUNDS`. The first Excel outer window is 723x390 versus the
902x487 pane client; its inner window is 903x488. Reopening has both native windows
at 903x488 and passes the same four-pixel allowance. The six ordinary workflow
records still pass, and the failure is reported only after preserved-byte checks
and normal exit 0. This is an actual assertion RED, not a timeout or inferred
failure from a missing screenshot.

The candidate production change calls the existing `resizePreviewHandler()` once
after successful `DoPreview` and ownership assignment, before automatic fit and
toolbar updates. It rereads the current pane client rectangle and uses the
handler's normal `SetRect` interface, following the documented host sequence.
It adds no scaling constant, timer, forced redraw or direct Excel-window input.
The existing initialization-failure and unload paths are unchanged. This remains
a candidate until the same native bounds gate and reviewed images pass. Regression
scope includes the focused XLSX/Open XML and DOCX/font units, Windows cross-build,
XLSX first-open/reopen and DOCX UI checks, followed by the native app baseline.

## Post-initialization resize alone is ineffective

Candidate `33cbe3c7d6ae8f00250079da456695a22ac5095d` passed the 24 XLSX/Open XML
and seven DOCX/font local units and Windows cross-build. It was pushed before
FlowDev update/build and the same native `ui-xlsx` gate. The new native app hash is
`2f230b189c49289cbcce6c48adea2c9b8acddf94083f047a08fa6cda5f7cbaf2`.
Run `dcm-quicksort-ui-xlsx-20261113-232554-5572d8ff` still exits 1 with
`DCM31_XLSX_NATIVE_BOUNDS` for first open: outer 723x390, inner 903x488, expected
client 902x487. Reopening passes both native rectangles. Source, files, journal
and normal close remain preserved. The post-initialization call is consistent
with the documented lifecycle but does not resolve this defect.

The next diagnostic records the supplied rectangle, GetClientRect success,
caller DPI context and COM HRESULT rather than assuming the ignored SetRect
result was successful. A separate real-HWND Windows unit will establish whether
the proposed measurement boundary returns physical client dimensions and restores
its caller context. A physical-rectangle experiment must be scoped to the observed
Excel CLSID `{00020827-0000-0000-C000-000000000046}` and prove both first open and
reopen; the currently correct inner/reopened windows must not become oversized.
The preview-handler documentation specifies an area in the parent HWND but does
not establish an unconditional physical-pixel contract for all providers.

## Native call trace and measurement RED

At pushed source `147bf90977d1a9162b3e5fc12a88364dcf68b244`, registered
`native-preview-rect-unit` run `20261113-234157-ea8a13d9` produced one top-level
pass, one assertion failure and zero skips. The actual scale witness was an
unaware 320x240 client versus a 400x300 PMv2 borderless physical window. All three
unaware-caller matrix cases returned logical dimensions and failed; the six other
matrix cases and all three invalid-handle context cases passed. This establishes
actual RED for the deliberately logical measurement boundary.

The exact native build passed (app SHA-256
`e37ec98fd1c16a37629e865e7121e73a6943f589c4f40fa41191a7deaa77a7ec`).
Registered `ui-xlsx-layout` run `20261114-003449-c091c991` again fails all ten
workbook bounds observations while its six ordinary workflow records, preserved
source/fixture bytes, empty journal and normal close pass. The COM trace has all
12 records, below its 128-record cap. Both sessions activate the selected Excel
CLSID via LOCAL_SERVER, return true for GetClientRect (722x390), and return S_OK
from SetWindow, DoPreview and SetRect. Caller thread and pane remain context24592,
awareness0 before and after every observed call. The physical pane is902x487;
outerEXCEL9 is723x390 and innerEXCEL7 is903x488 throughout. A failed COM HRESULT
or differing caller context does not explain these observed sessions.

The next candidate scopes PMv2 to physical client measurement only and restores
its caller before returning. Initial SetWindow uses that result only for the
parsed Excel CLSID; the existing logical post-DoPreview and later SetRect policy
is retained for the first compatibility experiment. This is a bounded trial,
not a universal COM coordinate convention or a proven clipping repair. The same
upper/lower bounds gate must reject oversizing as well as clipping. Native
measurement GREEN and first-open/reopen feature acceptance remain pending.

## Physical initial rectangle alone is ineffective

At pushed source `bcee0759ab95d01a6563d2d0a79a8ad80ae14959`, native
`native-preview-rect-unit` run `20261114-004101-9b8149b0` passes both top-level tests
and all twelve leaf cases, with zero skips. Physical dimensions, invalid handles,
unchanged HWND awareness/geometry and restored caller context are now qualified
on the observed 125% Lenovo display. The existing31 local units and Windows app/
test cross-builds also pass. Native build SHA-256 is
`443e57ddaa35bb358c984158a511bc4e384833810633a9f2707b6fca268d4df2`.

However, registered `ui-xlsx-layout` run `20261114-004206-3cfa9591` still fails all
ten workbook viewport observations. Both sessions trace successful physical
902x487 initial SetWindow followed by successful logical722x390 SetRect, with the
original unaware caller context restored before COM. The final observed outer
window remains723x390 and the inner903x488 against902x487 pane client. These data
do not show geometry between the two COM methods, so they do not prove exactly
when Excel applies the wrong size. The initial-only policy does not fix clipping.
All six ordinary records, source/fixture preservation and normal close still pass.

The next trial retains the verified physical measurement and applies it to both
SetWindow and SetRect for the active Excel handler. Its policy is latched with the
successfully initialized handler and cleared on unload; it is not inferred from
the currently selected filename. A failed live measurement leaves the current
preview intact without sending a zero rectangle. Other providers keep their
existing logical policy. Unchanged upper/lower native bounds and image review
must establish first-open/reopen GREEN before this trial can be called a repair.

## Uniform physical coordinates oversize the inner worksheet

Pushed source `9881724b25695ca154c2ab9d4aae02c614ace296` passes31 focused local
units and the Windows app cross-build. Native build passes with app SHA-256
`a53791f85645abf1c2d1c007d0c94d72f632f87bed444f7574825b7a43ae13b4`.
The unchanged registered `ui-xlsx-layout` gate (`20261114-004947-cef87ff6`) rejects
all ten workbook observations for a different reason: the outerEXCEL9 now fits
at903x488, but innerEXCEL7 is1128x609 against the902x487 pane client. All twelve
images were reviewed; workbook content fills the pane, while tabs and scrollbars
remain outside the visible area. Both sessions trace physical902x487 arguments
for SetWindow and SetRect, with S_OK returns and restored caller contexts. Six
ordinary workflow records and preserved bytes/source/normal close still pass.
A uniform scale does not size both observed levels correctly. This candidate is
not a fix and the upper-bound assertions must not be relaxed to accept it.

The failed physical COM wiring is now reverted to the exact147bf909 main source:
normal logical SetWindow/SetRect, the post-DoPreview resize and opt-in trace.
The next TDD boundary is an initially unused outer-window repair helper. Its native
unit uses hidden, owned test windows to require correct outer physical bounds,
unchanged inner geometry, preserved window identity/context and caller-context
restoration across three parent by three caller DPI modes. Wrong, missing,
ambiguous, nested or invalid child identities must be refused without mutation.
After actual native RED and correction, the proposed app integration is restricted
to the active selected Excel handler: one asynchronous SetWindowPos request for
its verified EXCEL9 direct child, scoped to the owned pane DPI context, after a
successful normal SetRect. It must not resize EXCEL7, activate or reparent windows,
or change process-wide awareness. Revalidate active handler/pane/policy after COM
and disable the repair policy when unload starts. Actual Excel geometry and
reviewed images remain the acceptance authority; request acceptance alone is not
feature success. No repair is implemented or qualified by this checkpoint.

Microsoft documents different-DPI cross-process parenting as a possible cause of
forced awareness resets, and explains that SetWindowPos may post across input
queues. Those rules support this bounded compatibility experiment; there is no
recorded proof of Excel's internal reset or a guaranteed provider response.
References: [SetParent](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setparent),
[SetWindowPos](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setwindowpos),
and [Office DPI guidance](https://learn.microsoft.com/en-us/office/client-developer/ddpi/handle-high-dpi-and-dpi-scaling-in-your-office-solution).

## Coverage limits and durable material

The diagnostic at147bf909 added bounded, opt-in native-call records to the existing
`DCM_LAYOUT_TRACE=1` output. It preserves the first 128 activation/GetClientRect/
SetWindow/DoPreview/SetRect records, including the selected CLSID, caller and pane
DPI contexts, rectangle arguments and exact HRESULTs. It does not alter DPI
contexts or native call ordering. A separate, initially unwired measurement
boundary deliberately retained logical GetClientRect behavior so the registered
`native-preview-rect-unit` could establish a real RED on Lenovo before correction.
The hidden owned HWND tests cover nine creator/caller DPI combinations, invalid
handles, unchanged window geometry/awareness and caller-context restoration.
An unavailable API or absent non-100% scaling is an explicit unqualified skip.
The combined diagnostic/scaffold passes the existing 24 XLSX/Open XML and seven
DOCX/font local units; both the Windows app and Windows test binary cross-compile.
Native RED and subsequent helper GREEN are recorded above. The consistent-policy
candidate also passes all31 focused local units and the Windows GUI cross-build;
its native bounds and visual acceptance remain pending.

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
