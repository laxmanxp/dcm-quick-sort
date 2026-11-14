# DCM-33 — legacy Office preview qualification

Started 14 November 2026 (Asia/Kolkata). Standalone DCM QuickSort v1.0.71,
repository `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.

**PARTIAL overall: the shared unavailable-footer fix and representative native
DOC milestone are complete; legacy XLS/PPT remain pending.** The numbered plan
remains **31 complete / 19 remaining**. The accepted source is
`b3d46b9e2003dbbfa2ae431e134e0692b8168d34`, pushed before the registered Lenovo
update and native gates. DOC alone does not close DCM-33 or certify the final
package. The smallest next baby step is genuine legacy XLS, followed by PPT.

## Starting identity and observed contract

The development checkout and Lenovo initially match pushed completion commit
`d4edf54b60b4ce86b6642df43547a77506bc7c4a`. The registered FlowDev profile maps
`dcm-quicksort` to `D:\FlowverseReportsRepo\dcm-quicksort-candidate`, the same
GitHub repository and `release/v1.0.71` branch. The initial checkout is clean.
Read-only lookup finds Word's DOC preview handler registered at `.doc\shellex`;
registration alone does not prove content can be rendered.

Legacy DOC/XLS/PPT use an installed Windows preview handler. The built-in Open
XML renderers do not treat legacy binary files as DOCX/XLSX/PPTX. If a handler
is absent or fails, the existing fallback explains the outcome and leaves the
ordinary rename/move/open workflow available. No new rendering engine is planned.

## Reproduced footer defect and bounded fix

The former generic unavailable branch updated the pane's fallback text but left
`statusLabel` unchanged. It could therefore retain a prior TXT or native-preview
success caption. The regression first extracted the existing final presentation
statements without changing behavior, then exercised the actual presenter with
owned hidden Win32 controls. The behavior-preserving extraction and assertions were pushed at
`cc0948108bff8c6ccc7721a4dfda69faac59bc91` before native execution.

The intended footer distinguishes a missing handler from a registered handler
that failed and identifies the current file type. Tests cover blank, prior TXT
success and prior native success. Missing-provider behavior is exercised at the
presentation boundary; tests do not launch Office or change live registrations.
After the native RED below, the minimal fix makes the shared presenter assign
the current unavailable footer together with its existing body before
invalidating the pane. The fix passes native units and the visible shared
unavailable control at `b3d46b9`.

## Focused qualification path

The fixture is a locally generated, synthetic Word 97–2003 binary document,
pinned by the [fixture manifest](../tests/fixtures/dcm33-legacy.manifest.json):
**18,432 bytes**, SHA-256
`a6e70ab1c1c6db3d720f7fe12b8268e7c74f55e27c27f926be0f6bf3df2c3105`.
It has genuine CFBF magic, `WordDocument` and `1Table` streams, FIB
`wIdent=0xA5EC` and observed `nFib=0x0101`. Independent piece-table extraction
and local source/export render review validate the four synthetic paragraphs:

- `DCM-33 LEGACY DOC`
- `Inspection date: 14 November 2026`
- `Bridge count: 7`
- `Status: Ready for review`

Qualification consumes the committed DOC bytes, not a renamed RTF/DOCX or a
Lenovo-generated conversion. The manifest records source, generator, validation
and bounded reproducibility details.

The headed path uses a harmless unsupported-extension control to exercise the
shared unavailable display, then previews the genuine DOC through the naturally
selected provider. It returns to TXT, checks exclusive file release, reopens DOC
and closes normally. It binds source, app and fixture hashes to the exact build,
checks an empty journal and preserves all captured frames. Actual visible words
and blank/error/success content require separate image review.

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort legacy-preview-unit
flowdev test dcm-quicksort ui-tooling
flowdev test dcm-quicksort build
flowdev test dcm-quicksort baseline
flowdev test dcm-quicksort ui-legacy-doc
```

Supply `DCM_BUILD_RECEIPT` from the exact current clean build to the UI gate.
Supply the pinned `tests/fixtures/dcm32-slides.pptx` through `DCM_REAL_PPTX` for
the baseline; the accepted run includes that real-file integration with no
skips. The focused unit gate precedes larger regression/feature runs. Native
execution and semantic image review remain distinct from local compilation or
a provider caption.

## Retained native RED

Native RED is recorded at `cc0948108bff8c6ccc7721a4dfda69faac59bc91` in
`D:\FlowverseReportsRepo\dcm-quicksort-regressions-20261114-121711-6f555113`.
The focused gate reports two top-level passes, one top-level failure, no skips
and exit 1 without timeout. All nine failures are the intended
`DCM33_UNAVAILABLE_STATUS` assertion: blank, prior TXT or prior native-success
footers survive missing-handler, failed-handler and extensionless outcomes.
All hidden-control setup and fallback-body assertions succeed. Source is
unchanged. The raw log SHA-256 is
`8a33ebc502337b9388153a8df85b1ec153517d20509c593890ca2368602116d4`.

The later GREEN does not relabel that run or its nine intended assertions.

## Accepted native DOC milestone at b3d46b9

The registered update received pushed source
`b3d46b9e2003dbbfa2ae431e134e0692b8168d34`. All following FlowDev commands exited
0, with source unchanged:

| Gate | Native report suffix | Result |
|---|---|---|
| `legacy-preview-unit` | `122305-046d0bcc` | 4 top-level tests and 33 leaf cases pass, zero failures/skips; the nine formerly failing footer transitions are GREEN. |
| `ui-tooling` | `122402-1c30522d` | 30 PowerShell scripts parse; native helpers compile. |
| `build` | `122409-a7d930fc` | Exact-source build passes with a 454-file source manifest. |
| `baseline` | `122523-cf1f5a00` | 388 passed / 0 failed / 0 skipped, including the unchanged real-PPTX integration with the identified fixture. |
| `ui-legacy-doc` | `122655-7ff7ee3e` | 8 workflow records pass; all 15 retained images are separately reviewed. |

Reports are under `D:\FlowverseReportsRepo`, using prefixes
`dcm-quicksort-regressions-20261114-`, `dcm-quicksort-ui-tooling-20261114-`,
`dcm-quicksort-build-20261114-`, `dcm-quicksort-baseline-20261114-` and
`dcm-quicksort-ui-legacy-doc-20261114-` respectively. The UI gate consumes the
exact `122409-a7d930fc\receipt.json` build receipt. Its app is **7,367,168 bytes**,
SHA-256 `d8df2a791b736ecbee517f7efaff742a5b64271516c97fed96cc604a7ee45b7b`.

The 388/0/0 baseline is a new full-suite result at this source. The DCM-31
383/0/1 result at `ec7a0234` remains historical and unchanged. The root FlowDev
invocation supplied the tracked PPTX path; the raw receipt does not persist
`DCM_REAL_PPTX`. The raw log independently proves exactly one real-PPTX run and
one pass, without a raw input-path binding. This run does not certify the final
installer/package or imply that earlier DOCX/XLSX/PPTX headed acceptance was
repeated at `b3d46b9`.

## Separate review of all 15 images

The run retains three TXT full-window frames and six full-window/crop pairs:
two attempts each for the unsupported control, first DOC open and reopening.
The first attempt at each stage remains readiness evidence; the second is the
stable frame named by `accepted_capture`. All attempts and crops were reviewed,
without changing their original readiness classifications.

| Stage | Reviewed result |
|---|---|
| Initial TXT and both TXT recoveries | Expected plain-text fixture content and correct TXT footer; host shell remains readable. |
| Shared unsupported control | Exact unavailable body, current control filename, ordinary file-action guidance and the `Preview unavailable: DCM33UNSUPPORTED (no Windows preview handler installed)` footer; no stale TXT/native success display. |
| First DOC open and reopening | All four required DOC lines are visible in both attempts and their crops, with `Windows preview: DOC` and an intact host shell. |

Each crop is cut from the saved full-window frame using recorded physical pane
geometry. The machine also finds all four document tokens in accessibility text,
but semantic acceptance comes from the separate image review. Raw
`feature_accepted=false`, `format_accepted=false`, capture
`visual_reviewed=false` and the explicit semantic-review requirement remain
unchanged. The [DOC native summary](evidence/dcm-33-doc-native-summary.json)
records the milestone review separately.

Source and fixture bytes are preserved, the operation journal stays empty,
exclusive DOC access succeeds after TXT recovery and after close, and the owned
host exits normally with code 0. The installed Word DOC handler is present;
`missing_handler_case_exercised=false`. The control qualifies the shared generic
unavailable display, not headed behavior on a machine lacking a DOC handler.
Registry observations before and after match; no Office registrations changed.

## Evidence, limits and next step

The [DOC native summary](evidence/dcm-33-doc-native-summary.json) binds the exact
source/build/fixture identities, native receipts, raw RED and GREEN, separate
image review and report publication identity. The report path is
`apps/dcm-quicksort/dcm33-20261114-122655/legacy-doc-b3d46b9` in
`laxmanxp/flowverse-reports`. Publishing this completion record documents the
tested `b3d46b9` application; it does not create another native acceptance run.

Acceptance covers the reproduced shared footer defect and this representative
synthetic DOC through the installed Word handler at the observed geometry.
It does not certify headed missing-DOC-provider behavior, arbitrary Office
fidelity, legacy XLS/PPT, protected/damaged files or all display scales. Broad
failure and lifecycle qualification remain DCM-34/35; final workflow/layout,
package and user acceptance remain open under DCM-46/47 and the release gates.

**DCM-33 remains PARTIAL; no numbered step closes.** Continue with a genuine
legacy XLS fixture and the smallest native/unavailable gate, then legacy PPT
in a separate baby step. DCM-19's Explorer-isolation blocker is unchanged.
