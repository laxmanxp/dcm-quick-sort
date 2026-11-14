# DCM QuickSort — project context

## Current checkpoint — 14 November 2026 (Asia/Kolkata)

The active application is **standalone DCM QuickSort v1.0.71**, repository
`https://github.com/laxmanxp/dcm-quick-sort.git`, branch `release/v1.0.71`.
Production source is `app-src/`; installer source is `installer-src/`.
The release is still unpublished. The current backlog is [todo.json](todo.json)
and the numbered qualification plan is [completion-50.json](qualification/completion-50.json).

**32 of 50 steps are complete; 18 remain.** Complete: DCM-01–18 and DCM-20–33.
DCM-19 remains blocked on a disposable Windows Explorer integration environment.
**Next runnable step: DCM-34 — preview failure and interruption states.**
DCM-33 is complete for representative installed-handler DOC/XLS/PPT previews
and the shared missing/failed-handler contract. Separate headed absent-provider
environments and broader release gates remain outside this bounded result.
These counts include setup work and are not a product-completion percentage
or user acceptance.

The DCM-31 completion record is now published at
`8671cc7bc9557ffca65de53ca79ace1156f65d00`; the registered Lenovo update received
that exact commit. Its production tests remain bound to `ec7a0234`, and its
report bundle is confirmed at `69fee755a94468165018b499fe7d876779e57ec8`.

### DCM-34 in progress: default PDF damage and recovery

The first new slice targets a header-valid damaged PDF through the default
Windows.Data.Pdf path, including its real Windows-handler fallback and exact
TXT/valid-PDF recovery. Existing text/image and optional Sumatra error evidence
remains at its recorded provider/source scope. Local focused PDF tests pass
14 top-level tests, including three new parents with 23 subcases. The Windows
test binary also cross-compiles; the new native milestone is pending. No renderer change is asserted by those tests.

A separate local probe at `9afb15b` confirms open defect
`DCM-34-DOCX-FALLBACK-01`: malformed DOCX rejected by the body renderer can regain
success through the metadata foundation fallback; empty/missing document/body
also reports zero-body success. The retained probe has 10 failing and 13 passing
leaf cases. Valid blank and recovery controls pass. This remains unfixed and
is not a claim about the installed Word handler. See
[the DCM-34 checkpoint](qualification/dcm34-preview-failures.md).
**DCM-34 remains PARTIAL; 32 complete / 18 remaining.**

### DCM-33 accepted DOC milestone

**The shared unavailable-footer fix and representative native DOC milestone
are complete at pushed and Lenovo-tested source
`b3d46b9e2003dbbfa2ae431e134e0692b8168d34`. At the DOC/XLS checkpoints, legacy
PPT was still pending and DCM-33 remained PARTIAL; completion follows below.** The registered update received
that source before the focused gates. The unavailable presenter now replaces the previous file's
success footer with the current type and missing/failed-handler outcome while
retaining the fallback body and ordinary file actions.

The native RED at `cc0948108bff8c6ccc7721a4dfda69faac59bc91`, report
`121711-6f555113`, preserves the nine intended blank/prior-TXT/prior-native footer
failures across missing/failed handlers and extensionless files. Setup and
fallback-body assertions passed. Native GREEN `122305-046d0bcc` passes **four
top-level tests and 33 leaf cases**, with zero failures/skips. `ui-tooling`
`122402-1c30522d` passes all 30 PowerShell scripts and native helper compilation.
Exact build `122409-a7d930fc` has a 454-file source manifest; its 7,367,168-byte
app SHA-256 is `d8df2a791b736ecbee517f7efaff742a5b64271516c97fed96cc604a7ee45b7b`.

The full native baseline `122523-cf1f5a00` is **388 passed / 0 failed /
0 skipped**, including the unchanged real-PPTX integration with the pinned
fixture supplied. This is a new result at `b3d46b9`; DCM-31's historical
383/0/1 baseline remains unchanged, and final package qualification stays open.

Registered `ui-legacy-doc` run `122655-7ff7ee3e` passes **eight records** and
retains **15 separately reviewed images**: three TXT frames and six
full-window/crop pairs across the unavailable control, first DOC open and
reopening. Both DOC attempts show all four required lines, including
`Inspection date: 14 November 2026`, `Bridge count: 7` and
`Status: Ready for review`. TXT initial/recovery content is correct, and the
unsupported control shows the exact unavailable body and footer. Raw
`feature_accepted=false`, `format_accepted=false` and capture
`visual_reviewed=false` remain intact; semantic acceptance is recorded
separately. Source and fixture bytes are unchanged, the journal is empty,
exclusive DOC release succeeds and the host closes normally with exit code 0.

The installed Word DOC handler was present. The harmless `.dcm33unsupported`
control exercises the shared generic unavailable path; it does not qualify a
headed missing-DOC-provider environment. No Office registrations were changed.
The DOC result is limited to this synthetic document and observed geometry;
it did not qualify legacy XLS/PPT. The later XLS result is recorded separately
below. Broader damaged/protected files and lifecycle gates, final workflow/
package, and user acceptance remain open.

See [the DCM-33 checkpoint](qualification/dcm33-legacy-office.md) and
[the DOC native summary](qualification/evidence/dcm-33-doc-native-summary.json)
for receipts, separate image review and report publication identity. The
DOC completion documentation records the tested `b3d46b9` source; publishing it
does not imply another native application acceptance run. At that milestone,
the numbered count stayed **31 complete / 19 remaining**.

### DCM-33 accepted XLS milestone

**The representative legacy XLS milestone is accepted at pushed and
Lenovo-tested source `e19ff30b1663a4b5bb385a3667ab6342f299cc1e`, with explicit
numeric equivalence for the observed locale.** It starts from completion record
`cb3b2ffabe598915d2184f7a0bbdb716a7b0bea4`. Application and installer source are
unchanged from `b3d46b9`; no new product defect, fix or RED is claimed. The
registered `flowdev update` PASS was observed in the Desktop Commander console.
No native update JSON was available; the execution observation records that
limit, while all five native test receipts bind the clean candidate source.

Registered `legacy-preview-unit` run `131559-93a60d28` passes **4 top-level /
33 leaf cases**, and `excel-preview-layout-unit` run `131705-b50e60c8` passes
**2 top-level / 48 leaf cases**, with zero failures/skips. `ui-tooling`
`131742-34c89720` passes all 31 scripts and native helper compilation. Exact build
`131747-caf81c29` has a 461-file source manifest and the same 7,367,168-byte app
SHA-256 `d8df2a791b736ecbee517f7efaff742a5b64271516c97fed96cc604a7ee45b7b`
as the accepted DOC build. The independent native evidence audit passes all
54 checks. The full 388/0/0 baseline remains the historical `b3d46b9` run and
was not rerun for this fixture/harness qualification.

Registered `ui-legacy-xls` run `132721-e381a696` passes **6 workflow records,
12 captures, 5 native attempts and 5 strict Excel bounds records**, retaining
the unchanged four-pixel tolerance. All 12 images were separately reviewed:
two TXT frames and five full-window/same-frame-crop pairs. First open becomes
stable only on attempt 3; reopening becomes stable only on attempt 2. The first
readiness frame already shows the cells, but its sheet-tab bar has not painted;
it remains unstable and is not relabeled as a stable completion frame.

All **11 cell meanings are correct; 9 display strings match exactly**. The date
reads `14 November 2026`, B7 reads `86,000`, and the labels and title are readable.
B6 displays `1,25,000` instead of the manifest's `125,000`; B8 displays
`2,11,000` instead of `211,000`. A separate read-only Lenovo observation confirms
`en-IN` grouping `[3, 2]`. Acceptance explicitly recognizes the same numeric
values, 125000 and 211000; it does not claim all 11 pinned literals matched.
The genuine BIFF8 fixture independently proves `=SUM(B6:B7)` and cache 211000;
the screenshots prove the displayed total, not a separate native recalculation.

Source and fixture bytes are unchanged, the journal is empty, exclusive workbook
release succeeds, the host exits normally with code 0 and Office registrations
are unchanged. The installed Excel handler succeeds; headed missing/failing XLS
providers and sheet navigation remain unqualified. Raw `feature_accepted=false`,
`format_accepted=false` and capture `visual_reviewed=false` remain intact;
semantic acceptance is recorded separately in the
[XLS native summary](qualification/evidence/dcm-33-xls-native-summary.json).
Publishing this documentation records the tested `e19ff30` qualification and
does not constitute another native run. **At the XLS checkpoint, DCM-33 stayed
PARTIAL, 31 complete / 19 remaining, with legacy PPT next.** The later completion
record below preserves those historical counts and scope.

### DCM-33 complete: representative legacy PPT and combined acceptance

**DCM-33 is complete within its representative DOC/XLS/PPT and shared unavailable
contract.** The final PPT source is
`8704a1bdd930886a9ae4d0849822add0088603d8`, pushed and received through registered
FlowDev before the final gates. It adds qualification assets and corrects one
harness expectation; application/installer source and recorded app bytes remain
unchanged from accepted `b3d46b9`. The genuine 510,976-byte CFBF PowerPoint fixture
has SHA-256 `b4e6a98d2f4dd27ab66566a58ae734b1b297e75a1053084b54867b5b3346a773`.
Independent validation follows the live presentation slide order, verifies both
slides and the original embedded image, and rejects renamed PPTX, a wrong amount
and reversed slide references.

Initial candidate `87788e3` passes native `legacy-preview-unit`
`143403-3dcb113d`: **4 top-level tests / 33 leaf cases, zero failures/skips**.
Its first UI run `143622-814c1f0b` stops before navigation because PowerPoint adds
` - Compatibility Mode` to the exact legacy window title. The two original
images and normal exit 0 remain preserved. The final helper pins that observed
PPT title while retaining the PPTX title and all ownership/process/input guards.
This is a harness correction, not a product rendering defect. The final delta
contains only the helper and tooling; the Go units were not rerun or rebound.

At `8704a1b`, `ui-tooling` `143912-47769611` passes **32 script parses, native
helper compilation, nine binding guards and two exact title contracts**.
Exact build `143916-26f42fd5` binds **468 source files** and the unchanged
7,367,168-byte app. Registered `ui-legacy-ppt` `143934-8e80521d` passes **eight
records, 18 captures, eight attempts and two scoped PageDown/PageUp actions**.
All 18 final images are separately reviewed: 10 full-window frames and eight
same-frame crops. The two initial-failure history images are reviewed separately.
The bridge title/date/image, Payments table `125000` / `86000`, native
**1 -> 2 -> 1**, TXT recovery and reopening are readable and correct.

Each native stage stabilizes on its second retained attempt. Slide 2 changes
26,210 sampled pixels; returning to slide 1 differs by zero. The independent
source/archive audit passes **68 checks** and image-integrity audit **185 checks**.
Source/fixtures and Office registrations are unchanged, the journal is empty,
the presentation is released and the host closes normally with exit 0.
Raw feature/format/visual-review flags remain false; semantic acceptance is
recorded separately. Unused space remains at the right/bottom of the pane;
required slide content is not clipped. Physical keyboard input, complete pane
sizing across scales and arbitrary presentation fidelity are not certified.

The combined closure uses the three separate installed-handler results, the
all-format missing/failed-handler routing/presenter units and the separately
reviewed shared unavailable control from DOC. It does not claim separate headed
absent/failing DOC/XLS/PPT provider environments. The historical **388/0/0**
baseline stays at `b3d46b9`; user acceptance, DCM-34/35/46/47 and release remain
open. See [the PPT native summary](qualification/evidence/dcm-33-ppt-native-summary.json)
for the pinned report publication and detailed limits. Documentation publication
records the accepted candidates and is not another native application run.
**32 complete / 18 remaining; next DCM-34.**

### DCM-32 qualified result

**DCM-32 is complete for the representative two-slide presentation through the
installed PowerPoint preview handler at the observed Lenovo geometry.** The
registered update received pushed source
`b3be3a016a92759ed88eeb364645f56fb5cc7916`. Application source remains unchanged
from `ec7a0234`; the intervening changes add qualification tooling and a pinned
fixture. Registered `ui-tooling` run `093847-0a9a5a0c` passes all 29 PowerShell
scripts and native C# compilation. `pptx-unit` run `093851-22afbd5a` passes
**29 tests, 0 failures, 0 skips**, and verifies exactly one run and one pass of
the unchanged real-PPTX integration. Exact build `093918-4f66ddaf` passes, with
all 444 source-manifest entries matched to Git, allowing recorded CRLF checkout
conversion. Its app SHA-256 is
`7f51956e67dd7e0a33eef51a6ad8cf8a742ad6e2d64585348614eaa485e35827`.

Registered `ui-pptx` run `094009-df395f56` passes **eight workflow records**,
retaining **18 captures: eight native full-window/crop pairs and two TXT
frames**. Two scoped PageDown/PageUp message pairs drive native 1 -> 2 -> 1
navigation; `all_fixture_slides_captured=true`. All 18 images, including readiness
attempts, accepted frames and saved-full-frame crops, were separately reviewed.
They show `Bridge Inspection`, `September 2026`, the bridge illustration, the
`Payments` table with `125000` / `86000`, return to slide 1, TXT recovery and
reopening, with the host shell intact. The raw `feature_accepted=false` and
capture `visual_reviewed=false` flags remain unchanged; semantic acceptance is
recorded separately. Source and fixture bytes are unchanged, the journal is
empty, the presentation is released and the host exits normally with code 0.

The earlier `3b2978fb` run `070557-4c20ba28` passed core tests but failed the
Windows PowerShell 5.1 source-manifest wrapper; the corrected `7f46906` unit run
`070759-9c7fff5a` and build `070857-05f16c29` remain separate evidence. Initial
UI run `070938-51df3033` remains a six-record/six-capture observation without
slide navigation. Its four previously missing images have now been recovered
and reviewed, completing that historical image inventory; the old crop's title
clipping remains recorded. The connection outage ending in a read-only HEAD
probe timeout at 07:18:14 UTC is historical: connectivity recovered for the
registered update and successful final gates above.

See [the DCM-32 checkpoint](qualification/dcm32-pptx.md) and
[the native acceptance summary](qualification/evidence/dcm-32-native-summary.json)
for exact receipts, image review, source audit and report publication identity.
The [earlier progress summary](qualification/evidence/dcm-32-progress-summary.json)
and [preliminary report bundle](https://github.com/laxmanxp/flowverse-reports/tree/86d0e95a02f60098f34a17d6450473d8f4c2757a/apps/dcm-quicksort/dcm32-20261114-070938/preliminary-7f46906)
retain the interrupted checkpoint as it was recorded. Acceptance covers the
installed native handler and scoped queued Page keys, not physical keyboard
input, headed fallback rendering, arbitrary presentations or all display
geometries. DCM-34/35/46/47, user acceptance and publication remain open.

### DCM-31 qualified result

DCM-31 is complete for representative native XLSX content and the reproduced
viewport sizing defect at pushed production source
`ec7a02349e9f53c6fcfd889cf21f800f36c68b4a`. The application keeps logical COM
SetWindow/SetRect rectangles. For the active exact Excel preview handler only,
it checks the owned pane and unique immediate EXCEL9/EXCEL7 hierarchy, then asks
Windows to size the outer viewport in the pane's DPI context. It restores the
caller context, revalidates active state after COM, and suppresses repair during
unload. It does not resize the inner worksheet or alter other providers' policy.
An accepted asynchronous request alone is not considered completed layout.

The unchanged native helper assertions pass **2 top-level tests and 48 leaf
cases**, with zero failures/skips, after the recorded RED. Registered
`ui-xlsx-layout` run `012003-6ce3d109` passes six workflow records and all ten
strict outer/inner bounds checks. Both renderer windows are 903x488 against the
902x487 pane client, within the unchanged four-pixel tolerance. All 12 diagnostic
captures were separately reviewed, including first open before Restore, natural
adjacent observations, text recovery and reopening. Orders cells, full dates,
Orders/Read me tabs and scrollbars are visible without the former unused bands.
Native sheet navigation and the Read me sheet body were not exercised.

The ordinary `ui-xlsx` regression `012116-fbdc6385` passes six records and two
bounds checks, with all four captures reviewed. `ui-docx` regression
`012139-efd939a9` passes seven records and all five captures are reviewed. The
21 current images show the expected content, correct provider footer and complete
host shell in their recorded frames. Source and fixture bytes are unchanged,
journals remain empty, files are released and all three UI hosts close normally
with exit 0. Raw machine visual-review and feature-accepted flags remain intact;
acceptance is recorded separately.

The full native app baseline `012309-4ee29c99` at that source is **383 passed,
0 failed, 1 external-PPTX skip**. The skip is `TestRealGeneratedPPTXIntegration`
because `DCM_REAL_PPTX` was not supplied. That historical skip remains unchanged;
the later focused DCM-32 parser run passes 29/0/0. DCM-32's bounded two-slide
visual acceptance is recorded above; fixture-bound final-package qualification
remains open under DCM-47.
The exact native build `011936-e3a17b9f` is 7,365,632 bytes, SHA-256
`7f51956e67dd7e0a33eef51a6ad8cf8a742ad6e2d64585348614eaa485e35827`.
Local validation passed 24 XLSX/OpenXML and seven DOCX/font tests, and both the
Windows app and test binary cross-compiled. These local artifacts are distinct
from the native build and are not release packages.

See [DCM-31 completion](qualification/dcm31-complete.md),
[the native summary](qualification/evidence/dcm-31-native-summary.json), and
[the durable evidence bundle](https://github.com/laxmanxp/flowverse-reports/tree/69fee755a94468165018b499fe7d876779e57ec8/apps/dcm-quicksort/dcm31-20261113-225138/outer-layout-fix-ec7a023).
Qualification is limited to the representative workbook and installed Excel
handler at the tested Lenovo scaling/geometry. Headed fallback behavior, native
sheet navigation/zoom, arbitrary/protected/damaged workbooks, general Excel
fidelity and all-provider/all-DPI behavior are not certified. Broader preview
failure and cleanup gates remain DCM-34/35; user acceptance and final package/
publication remain open. Excel's internal DPI behavior was not proven as a cause.

### Retained earlier evidence

The original `3d3af0fc` XLSX run passed six raw records but failed first-open
visual review. Diagnostic `b437ba2a` confirmed persistent clipping before Restore
and over five seconds later with idle host layout. The strict gate produced RED
at `45679e0`. Post-DoPreview resize `33cbe3c7` did not fix it. At `147bf909`, trace
showed successful logical 722x390 COM calls with stable caller context. Initial-only
physical SetWindow at `bcee0759` still clipped; uniform physical rectangles at
`9881724b` fitted the outer viewport but oversized the inner worksheet to 1128x609.
Those failed COM policies were reverted. Corrected hidden-window fixtures at
`95cab148` established clean outer-repair RED: nine geometry failures, all 39 guard
cases passing and zero skips. The failed trials, original image failure and first
fixture-setup failure remain recorded in
[the sizing checkpoint](qualification/dcm31-preview-sizing.md); later GREEN does
not relabel them as passing or prove Excel's internal mechanism.

DCM-30's original completion remains at production source
`8eb1470a891ea3454f1738d58191fdb90284ab42`: seven native records and five reviewed
images through the installed Word handler. Its historical baseline was 379/0/1;
the current ec7a0234 regression and 383/0/1 baseline are recorded above.
See [DCM-30 evidence and limits](qualification/dcm30-complete.md).
An earlier raw-pass DOCX run has an incomplete host-paint screenshot. The new
clean frames do not establish its cause or repair. `DCM-UI-RESIZE-01` remains
open under DCM-46, preserving the earlier bounded queued-layout fix history.

### Verified Lenovo registration and restart

- App ID: `dcm-quicksort`.
- Checkout: `D:\FlowverseReportsRepo\dcm-quicksort-candidate`.
- Profile: that checkout's `tools\flowdev-dcm`; `config/repos.list` maps to the
  exact repository and branch above. Do not substitute another QuickSort folder.
- Runtime: `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`.
- Workspace/report root: `D:\FlowverseReportsRepo`.
- XLSX selectors: `xlsx-unit`, `ui-xlsx` and diagnostic `ui-xlsx-layout`.
- `native-preview-rect-unit` is the isolated real-HWND physical measurement gate:
  RED at 147bf909, then native GREEN at bcee0759. Its qualified helper remains
  separate and unused by the accepted application sizing policy.
- `excel-preview-layout-unit` covers the guarded outer-window repair boundary:
  corrected source 95cab148 establishes clean native RED; ec7a0234 passes the same
  two top-level tests and 48 leaf cases, including nine geometry and 39 guard cases.
- Completed DOCX selector: `ui-docx` → `tools/test-ui-docx.ps1`.
- Completed PPTX selectors: `pptx-unit` and `ui-pptx`, with the tracked
  `tests/fixtures/dcm32-slides.pptx` fixture and a matching exact build receipt.
- Completed DOC milestone selectors: `legacy-preview-unit` and `ui-legacy-doc`,
  using the pinned `tests/fixtures/dcm33-legacy.doc` and matching exact build.
- Completed XLS milestone selector: `ui-legacy-xls`, using the pinned
  `tests/fixtures/dcm33-legacy-xls.xls`, focused legacy/Excel units and matching
  exact build. The separate image review records the observed en-IN grouping.

- Completed PPT milestone selector: `ui-legacy-ppt`, using the pinned
  `tests/fixtures/dcm33-legacy-ppt.ppt` and exact build. Preserve the observed
  Compatibility Mode title guard, scoped navigation and separate image review.

The smallest next task is DCM-34: select the smallest remaining damaged/protected
preview or interruption/recovery case, using the existing shared unavailable
control as prior evidence. Preserve the accepted DOC/XLS/PPT milestones; do not
start with the broad `ui-previews` workflow. DCM-19's Explorer-isolation blocker
is unchanged.

Code locally, unit test, commit/push, then use Desktop Commander for
`flowdev update dcm-quicksort` followed by the focused `flowdev test` selector.
Only one native run at a time. UI tests require `DCM_BUILD_RECEIPT` from a
successful `build` run for the exact current clean checkout; rebuild after
checkout changes. Read the selected harness before choosing fixtures or
environment variables. Review actual images separately from machine results.

## Historical checkpoint — 9 November 2026

The sections below preserve the original recovery checkpoint and historical
findings. Their then-open statuses are superseded by the current checkpoint,
`todo.json` and the numbered plan; do not restart already-closed safety work.

Checkpoint: 2026-11-09. v1.0.71 is an unpublished candidate on `release/v1.0.71`. Source recovery and native regression work are recorded below; final packaging and UI acceptance remain pending.

## Product decision

**FileFlow Suite** is the umbrella for **DCM QuickSort, Flow Search, DeepSearch and FileFlow**, with each remaining a separate application/window. DCM owns its Preview → Rename → Move → Next workflow, local learning, destinations, Processed history, recovery and FileFlow handoff.

Reuse the existing native Go implementation and established behavior. A Walk/Rust rewrite, replacement PDF renderer or shared launcher is not a prerequisite for DCM release qualification. Any future shared integration should preserve each app's independent launch and existing functionality.

## Active v1.0.71 qualification

The complete supplied v1.0.70 source was imported at `abe95acad96343e16d93396790a52bfe4d4b7b59`, preserving the original archive manifest and provenance. Archive SHA-256: `64e1cb1666acc9a70aeeb9656b4f30ca26fc30ad142ea32e154cb7a63dc7b5aa`. Current app source is in `app-src/`; installer source is in `installer-src/`. The old archive binaries are provenance only.

D3 batch journal membership and D4 template import/restore passed native regressions at `7c413fd530126398732cab7cda426b591514deb5`. The full installer unit suite at that commit passed **24 tests, zero failures and zero skips**, including D4/D5 restore and preference preservation. D1 same-file identity and D6 original-byte Undo fixes were committed at `97b90418af38989e024706c6e8c544e89d782591`. D2 guarded retirement was committed at `6ddd7da604d9181a8ea46f8b6b22e4c39ffb3154`. Their fixed-candidate native GREEN remains pending.

Two additional regression boundaries remain open: refuse an existing foreign destination on the fast rename path, and prevent a duplicate record from being reconciled as an ordinary rename chain. Their tests are committed; their implementation/validation is not closed.

Native report `20261109-043859-6216f4a1` timed out at `go version`, before product tests. Report `20261109-044942-5df3654c` reached the app regression command but exceeded its 300-second deadline; detailed partial test results were still being retrieved at this checkpoint. Do not infer a product PASS from either timeout. The completed FlowDev host diagnostic found 650,547,200 bytes free on C: and 478,658,560 bytes of available physical memory at 05:03 UTC. This resource pressure is consistent with the timeouts; it does not prove their individual causes. Restore resource headroom before further qualification. See [the diagnostic excerpt](qualification/native-host-2026-11-09.json). The source work is saved in [draft PR #1](https://github.com/laxmanxp/dcm-quick-sort/pull/1).

Exact v1.0.71 build/setup hashes, embedded-payload verification, isolated installer lifecycle, actual setup UI, visible filters/Rename/Move/Batch Undo, PDF zoom and preview checks remain outstanding. User acceptance is not recorded. The latest public release remains v1.0.63.

See [the qualification checkpoint](qualification/2026-11-09-progress.md), [draft release notes](qualification/v1.0.71-release-notes.md), [source import](provenance/v1.0.70/IMPORT.md) and [tracked work](todo.json). All product tests run on Lenovo through the registered standalone FlowDev profile.

## Historical version and source checkpoints

| Checkpoint | Evidence and limit |
|---|---|
| Latest published release | [v1.0.63](https://github.com/laxmanxp/dcm-quick-sort/releases/tag/v1.0.63), published 2026-09-23, with `DCM-QuickSort-Setup-v1.0.63.exe`. The release notes record installer SHA-256 `6be092ca2b11e8f56f4183beaa7fab59ef173ae653bdc29126fc3e60911b597c`; this checkpoint did not download or rehash it. |
| Latest identified native acceptance record | v1.0.69 on Lenovo, 2026-11-03. [The report](https://github.com/laxmanxp/dcm-quick-sort/blob/2ece9b1e044545c666b6c6296052feb6b985b750/tests/flowuivision/TEST_REPORT_2026-11-03.md) is partial, not a full release PASS. Tested EXE SHA-256: `BF93CE364909F8541D015D42CAFF3A2B0970574197CD60D0A0E756DF97E9683B`. The report commit identifies evidence, not the executable's source revision. |
| Recovered source in GitHub | [v1.0.62 recovery](https://github.com/laxmanxp/dcm-quick-sort/tree/c27d514ffeb2a76a8d77d776855ecf22b460930c/source-v1.0.62). Its transfer context records app/installer test and build passes, with live batch acceptance and final packaged-source reproducibility still requiring qualification. |
| Newer supplied archive | `DCM-QuickSort-v1.0.70-Source-and-Installer(1).zip` was audited on 2026-11-02. The complete source is now imported on `release/v1.0.71` with archive provenance. A newly built and qualified v1.0.71 app/setup pair remains pending. |

The repository main head read for this checkpoint was `e24a3a15aa88ca78a1fbe8ddecfdbd1cceae8edc`, tree `e98e21da9a92b769a40f4c2b7644bf3a4f7d986d`. Main contained release/site assets and an outdated v1.0.53 README, not a canonical current app source checkout. The QA and recovery branches are separate evidence sources. This describes the earlier main-branch state. The active release branch now contains the recovered source and qualification tooling.

## v1.0.70 archive safety findings

Source: [QuickSort code comparison, 2026-11-02](https://chatgpt.com/library?file_id=libfile_2c8db3c7893481919491f94250a2c300&account_id=personal). This historical Linux source audit reported **327 app tests passing, one real-PPTX fixture skip, 20 installer tests passing and cross-builds passing**. Eight added probes failed their intended assertions across the six findings below. Those passes do not close the added failures. The active checkpoint above records later fixes and native evidence. These historical findings are not attributed automatically to the older published v1.0.63 binary.

| ID | Finding and evidence distinction | Remaining gate |
|---|---|---|
| D1 — high | An exact same physical source/target can be treated as its own duplicate and removed; Undo cannot recover it. Exact-path deletion was reproduced in the source audit. The Windows case-only UI route was source-traced, not natively reproduced. | Guard file identity and verify exact-path and Windows alias/case variants preserve the source. |
| D2 — high | During cross-volume copy, the source may change before deletion. A controlled same-length edit left older target bytes and removed the newer source in the audit. | Preserve a changed source and qualify the copy/delete boundary with independent byte checks. |
| D3 — medium | The batch worker attaches `BatchID` late, leaving a crash interval without batch membership. Individual transaction history remains. | Make batch membership durable at the required transaction boundary and verify restart behavior. |
| D4 — medium | Naming templates are exported and included in the digest but omitted from the import allowlist, so a valid export fails checksum validation. | Qualify export/import round-trip with naming templates. |
| D5 — medium | Installer restore recognizes `default_source` but excludes `pdf_default_view` and `vault_root`. | Preserve these settings through backup/restore and installer upgrade. |
| D6 — Undo contract | Duplicate-removal Undo copies the destination's current bytes. If that destination changed, Undo restores the changed content rather than the removed original. | Define and implement truthful original-byte restoration or a refusal when original-byte recovery is unavailable. |

D3/D4/D5 now have native regression or unit GREEN at the recorded commit. D1/D2/D6 fixed-candidate validation and final release qualification remain open. Detailed implementation, UI and user-test states are separate in [todo.json](todo.json).

## Native evidence and remaining coverage

The v1.0.69 Lenovo report confirmed launch/version, filters and no-match state, sorting/navigation, TXT/PNG previews, lightweight DOCX/XLSX/PPTX fallbacks, initial PDF/Fit Width, spacing cleanup, Rename and Undo Rename. The Office results do not certify full Office page-layout fidelity. Its 53 filter/navigation steps are actions/assertions, not 53 distinct completed features.

Open native gates from that report:

- Manual PDF 100%/110% captures showed a blank document area although footer assertions passed; Fit Width restored content. Rendering versus capture/focus cause remains unresolved.
- Move to Vault did not complete in automated attempts, and the independent filesystem oracle failed. This does not prove every normal user move is broken. Undo Move was not exercised because no move was confirmed.
- Batch selection/move was not certified because the FlowUIVision adapter could not select the native ListBox. Multi-select and batch move already exist in the recovered source and the v1.0.63 release notes; this is a coverage/tooling gap, not evidence that the feature is absent.
- Legacy DOC/XLS/PPT, complex Office documents, protected/damaged files, overwrite collisions, crash recovery and physical printing were not certified in that pass.

Before a newer release: identify the exact source and installer payload; resolve D1–D6 with the appropriate regression evidence; qualify the actual Windows workflow and lifecycle; verify the packaged installer/upgrade/restore against the advertised DCM release scope. Physical printing is an unverified capability, not a newly imposed feature or release gate. Preserve both failed and successful evidence, including readable images and independent filesystem checks.

## Working rules

Code in the ChatGPT development environment; store changes in small GitHub commits; validate on Lenovo through the registered FlowDev app/feature route (`flowdev update <app>` then `flowdev test <app> <feature>`). Follow Feature → Units → Unit → Units → Feature. Record implementation, UI test and user-test status separately. Source-review probes, cross-builds, existing release notes and another app's tests must not be relabeled as current DCM native acceptance.

