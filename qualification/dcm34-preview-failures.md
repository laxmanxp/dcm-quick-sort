# DCM-34 — preview failure and interruption states

## 16 November 2026 — observed PDF interruption accepted with limits

The previously blocked corrected headed retry now **passes** at pushed FlowDev-synchronized source `ed75287c58b8f0f31b4171408593cc2f9a233a7a`. Both valid success and damaged error renderers started before TXT selection, finished after it, and their obsolete results were discarded; a same-file reopen retained the newer generation and rendered the correct page. Three focused/production-prep native checks, exact build, in-flight GUI and all three trace-disabled normal/damaged/protected PDF regressions passed. All 12 original in-flight captures were individually reviewed. Both PDFs were exclusively released; fixture bytes/source were unchanged and the host exited normally. No production source fix or renderer process hard-cancellation is claimed.

See [bounded in-flight closeout](evidence/dcm-34-pdf-inflight-closeout-2026-11-16.md), committed after its tested source. Raw `feature_accepted=false` and user-test flags are unchanged. The earlier failed external-process observation and its failed receipts remain historical evidence.

**DCM-34 remains PARTIAL, 32/50 complete:** wider Office error/protection coverage and `DCM-PDF-RENDER-SIZE-01` still need resolution or explicit limitation; end-user and suite release acceptance remain open. Next work should target those distinct cases rather than repeat successful XLSX/PPTX sizing and PDF interruption tests.

## 15 November 2026 - live PDF observation recorded; corrected headed run pending

`DCM-34-PDF-INTERRUPT-01` remains **PARTIAL**. Opt-in, bounded JSONL tracing
and the registered `ui-pdf-inflight` harness are implemented. At `3f07fe2c0e60c63240101ef642100bba4a7dfb1c`,
the real valid-PDF renderer starts, the selection changes to TXT while it is
still running, and its later successful result is consumed and discarded. The
original headed run still **failed** its external-process exit-code check after
that observation; this is not full workflow or visual acceptance.

A disposable diagnostic identified a null external ExitCode without a retained
process handle. The observer correction at `32b906ed818bc8b9d57af751fd5c423a3555a8d5` retains the live handle
and passes the native tooling regression: 11 trace-contract checks plus one
process-observer check. The exact build and three worker-ownership tests pass.
The **409 passed / 0 failed / 0 skipped** Windows baseline and four trace tests
retain their actual source `3f07fe2c0e60c63240101ef642100bba4a7dfb1c`. Application/tests/fixtures/native
runner identities are unchanged by the observer correction, and the rebuilt
executable hash matches. No fresh baseline at the corrected source is claimed.

The corrected headed retry **did not start**: `flowdev run unoembed` remained
on the shared desktop through a bounded wait. No foreign process was stopped.
Original failed receipts and raw visual flags remain unchanged. All 501 source
files were rehashed; 185 raw evidence files are published in [report `f65c71c5`](https://github.com/laxmanxp/flowverse-reports/tree/f65c71c547ba4fafd8d48ed27683e5fd0a1fb7bd/apps/dcm-quicksort/dcm34-20261115-inflight/32b906e).
See `qualification/evidence/dcm-34-pdf-inflight-progress.json` for exact source attribution and scope.

**Next:** on an idle desktop, use a fresh exact-current-source FlowDev build, bind
`DCM_BUILD_RECEIPT`, run `ui-pdf-inflight`, then the trace-disabled normal PDF,
damaged-PDF and protected-PDF regressions. Retain every failed observation.
No artificial renderer delay, suspension, substitution or process cancellation
was added. Keep DCM-34 PARTIAL and **32/50 complete**. The bitmap-size observation,
human acceptance and Flow Search/DeepSearch suite-readiness gate remain open.

Earlier preparation and acceptance notes below retain their original scope.

## 15 November 2026 - PDF worker ownership fix accepted; interruption remains partial

Native RED at `92a5023` reproduced four old success/error completions clearing a
newer worker's busy flag, including same-file reopen. The worker-generation fix
is accepted at `594709d1277ec28f155a2a54162122414c48678c`: three focused tests,
14 PDF failure tests, full Windows baseline 405 passed / 0 failed / 0 skipped,
and exact build/tooling pass. Headed regression records: normal PDF 19, damaged
PDF 9, protected PDF 9. These are normal completed-state regressions, not an
observed normal-app in-flight switch. No process cancellation was added.

The first baseline (404 passed / 1 skipped) remains retained; the full baseline
was repeated with the committed PPTX fixture bound and its integration run/pass
verified. Raw receipt and visual flags remain unchanged.

Evidence: `qualification/evidence/dcm-34-pdf-interruption-worker.json`; immutable
report commit `d1c44657fb9c27bb328e08d7950d56519a27c878` in `laxmanxp/flowverse-reports`,
prefix `apps/dcm-quicksort/dcm34-20261115-interruption/594709d`.

**Next:** observe the real application's rendering request in flight, switch
selection, and prove that the obsolete completion cannot change the new preview.
Keep DCM-34-PDF-INTERRUPT-01 and DCM-34 PARTIAL, 32/50 complete, the open
DCM-PDF-RENDER-SIZE-01 observation, and the Flow Search/DeepSearch release gate.

## 15 November 2026 - bounded protected-PDF milestone accepted

`DCM-34-PDF-PROTECTED-01` is **complete within its bounded no-password contract**. The pending `ui-pdf-error-controls` damaged-file regression passed at `42a6c1da83604284c2c2e97b341df98e93c62d0e` after the shared desktop became idle. It used the reverified exact-source build: 9 records, 34 retained captures, 14 readiness attempts, two native errors with all five host sizing controls disabled, correct TXT/PDF recovery, exclusive release and normal exit 0. Both TXT recovery pixel deltas are zero. Seven accepted-state full frames were separately reviewed; remaining transitions/crops are retained without separate acceptance and raw visual flags are unchanged.

The protected-file UI and 402/0/0 Windows baseline retain their actual source `1170e844c56ec019ae219abc8e3248092482e54d`; neither was rerun for this closeout. Application implementation is unchanged. Original failed runs, concurrency holds and the prior partial checkpoint remain historical evidence.

Evidence: `qualification/evidence/dcm-34-pdf-protected-closeout.json` and [immutable regression reports `dca83055`](https://github.com/laxmanxp/flowverse-reports/tree/dca830554ba1b3305a6efef1f0e1e7b657830700/apps/dcm-quicksort/dcm34-20261115-protected/damage-regression-42a6c1d-083159).

**Keep 32/50 complete and full DCM-34 PARTIAL.** `DCM-PDF-RENDER-SIZE-01` remains open: reported 612x1000 versus decoded 765x1250, with cause/impact unestablished and no size-equivalence claim. **Next DCM-34 task:** observed in-flight preview interruption and late-result rejection. Password entry/unlocking and human acceptance are unqualified. Final suite packaging and installation remain gated on Flow Search and DeepSearch readiness.

The earlier pending-regression statements below are historical and superseded by this checkpoint.

## 15 November 2026 - protected-PDF path passes; regression still pending

`DCM-34-PDF-PROTECTED-01` remains **PARTIAL**, with meaningful native progress:
three focused renderer tests and five renderer phases pass at `0786a42342f99dd059cd41d1078bbc33c8a43a2b`;
the final baseline at `1170e844c56ec019ae219abc8e3248092482e54d` passes **402/0/0**. The exact-source
build/tooling and protected UI pass: **nine records, two refusals, valid TXT/PDF
recovery, exclusive release and normal exit 0**. All seven accepted-state full
frames were separately reviewed. No password was supplied to the application.
The application binary is unchanged from the previously accepted implementation.

The additional `ui-pdf-error-controls` damaged-fixture regression **did not run**:
another UnoEmbed FlowDev GUI test held the shared session. The protected workflow
had already passed; the final wrapper hold is not a failure of that result.
No foreign process was stopped. Original native sizing-assertion and headed
phrase-miss failures are preserved, not relabeled as passing runs.

Evidence: [qualification/evidence/dcm-34-pdf-protected-progress.json](evidence/dcm-34-pdf-protected-progress.json); immutable raw report commit
`0d0ca7b2871190bae66affd159d5f8fcf7c69b14` in `laxmanxp/flowverse-reports`, prefix `apps/dcm-quicksort/dcm34-20261115-protected/1170e84`.
Raw visual flags remain unchanged. Transitions/crops are retained, not separately
accepted. `DCM-PDF-RENDER-SIZE-01` remains open: reported 612x1000 versus decoded
765x1250, with cause and production impact unestablished. No sizing fix is claimed.

**Next:** obtain an exact FlowDev build of the current pushed checkpoint, bind
`DCM_BUILD_RECEIPT`, then run `flowdev test dcm-quicksort ui-pdf-error-controls`
when the shared desktop is idle. Do not use an older-revision build receipt.
Keep **32/50 complete and DCM-34 PARTIAL**; password entry, in-flight interruption,
user acceptance and final suite packaging are not accepted by this result.
The Flow Search / DeepSearch suite-readiness gate is unchanged.

Earlier preparation notes below are historical and superseded by this checkpoint.

## 15 November 2026 - PDF error controls accepted with limits

`DCM-34-PDF-CONTROLS-01` is **fixed** at `c20ad2649df9f78847dab2058c27ab8852e0062e`.
The registered native RED at `4f29d69` displayed an explicit failed PDF but
left minus, 100%, plus and Fit enabled. The production correction is
`bece916`; `c20ad26` only corrects thread ownership in the new native test.

Fresh registered Lenovo gates pass: 4 control tests, 14 PDF failure tests,
**399 baseline tests / 0 failures / 0 skips**, tooling, exact build,
9 error/recovery UI records, 19 normal-PDF records and 7 DOCX records.
All 34 GREEN full-window frames are covered by separate visual review of
25 exact-hash groups; the original RED image was also inspected. Preview
crops remain preserved but were not separately reviewed. Raw flags are unchanged.

Both damaged openings have all five host sizing controls disabled. Direct
and WM_COMMAND sizing routes preserve the error footer/viewport and make
no native sizing/focus calls in the owned component test. TXT and primary
PDF recovery restore their supported controls.

**Conservative limit:** host sizing is disabled for all attached native PDF
fallbacks, even valid PDFs, until readiness can be established. The primary
Windows.Data.Pdf renderer, native-provider controls and ordinary SetRect layout
are unchanged. Valid native-fallback rendering was not separately requalified.
Zoom/pan captures demonstrate a bounded viewport, not whole-page visibility.

[Immutable RED/GREEN evidence](https://github.com/laxmanxp/flowverse-reports/tree/acf8d7287a227f2cbd9703afcbfe7f71f55954fa/apps/dcm-quicksort/dcm34-20261115-pdf-controls/c20ad26) preserves 405 original files; all
410 publication blobs were verified before push. The 489-file source manifest
and built executable were independently rehashed. See
[`dcm-34-pdf-controls-native-summary.json`](evidence/dcm-34-pdf-controls-native-summary.json).

Earlier open-control/DOCX statements below are historical and superseded by
this milestone and the DOCX closeout; their original evidence is not rewritten.
**Full DCM-34 remains PARTIAL: 32/50 complete.** Next is default-PDF protected-file
refusal/recovery, followed by observed in-flight interruption and remaining
Office failures. Human acceptance is not recorded. Final suite packaging and
installation still wait for Flow Search and DeepSearch readiness.

## 15 November 2026 - DOCX closeout and suite release decision

DCM-34-DOCX-FALLBACK-01 is fixed at bfbb731 within the built-in fallback contract: native RED preserved; 5 focused and 23 existing DOCX/Open XML tests pass; baseline 396/0/0; 16 headed component states and 7 normal-app DOCX records pass with all 21 captures separately reviewed. The missing/failed-handler inputs are owned component boundary inputs, not separate absent/failing Word environments. Raw visual/acceptance flags remain unchanged. Existing PDF milestone remains separately attributed. Full DCM-34 is PARTIAL, 32 complete / 18 remaining; protected PDF, in-flight interruption, PDF error-control readiness and broader Office failure coverage remain. User acceptance and release remain open.

Original RED/GREEN evidence: [f0caef5efb0f4f3ac370b412577d38ab94aa3fdb](https://github.com/laxmanxp/flowverse-reports/tree/f0caef5efb0f4f3ac370b412577d38ab94aa3fdb/apps/dcm-quicksort/dcm34-20261114-232225/docx-fallback-bfbb731). The separate acceptance and 21-image review are in [qualification/evidence/dcm-34-docx-fallback-native-summary.json](https://github.com/laxmanxp/dcm-quick-sort/blob/release/v1.0.71/qualification/evidence/dcm-34-docx-fallback-native-summary.json). This records existing tests; it does not claim a new application/UI run. Earlier DOCX open/unfixed statements below describe the pre-fix PDF/probe checkpoint and are superseded here.

**Suite decision:** FileFlow Suite includes DCM QuickSort, Flow Search, DeepSearch and FileFlow as separate applications/windows. Final suite packaging and installation are deferred until Flow Search and DeepSearch are complete and accepted for the intended scope. Their readiness was not checked in this DCM step. QuickSort development, source-bound builds and focused tests continue independently. No UI/codebase merger or installer topology is assumed. Publication still requires explicit approval.

**Next QuickSort work:** continue the remaining DCM-34 preview cases; do not recreate the already-pushed DOCX fix or restart completed source recovery.

Started 14 November 2026 (Asia/Kolkata), standalone DCM QuickSort v1.0.71,
`laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.

**PARTIAL. The numbered plan remains 32 complete / 18 remaining.** DCM-33
is complete at its recorded representative scope; DCM-19 remains blocked on
a disposable Windows Explorer environment. Physical user acceptance and release
publication remain open.

## Accepted bounded milestone: default PDF damage and recovery

The milestone is **accepted with limits** at pushed and Lenovo-tested source
`3847a157feba71b4d34dd2c2ba35ed2f0214cc0d`. The normal default
Windows.Data.Pdf path displays the representative valid PDF, then uses its
naturally selected Windows preview fallback for the pinned damaged PDF. The
observed Edge/WebView error reads `We can't open this file`; exact TXT and
valid-PDF recovery, damaged-file reopening, release and normal close pass.
This accepts one damaged-file/completed-state recovery flow. Full DCM-34,
protected PDF, in-flight interruption/cancellation, broader Office failures,
physical user acceptance and release remain open.

### Exact source and gate attribution

| Evidence | Tested source | Result |
| --- | --- | --- |
| Registered PDF unit gate `170709-24e362ea` | `499b015ee11c3554f65c3a1bc9368888c16542fc` | 14 top-level tests; 23 nested subcases + 11 standalone tests = 34 terminal cases; zero failures/skips/timeouts. |
| Final `ui-tooling` | `3847a157feba71b4d34dd2c2ba35ed2f0214cc0d` | 33 PowerShell scripts parse, native helpers compile, nine existing binding guards and two title contracts pass. |
| Final build `173029-fcc69290` | `3847a157feba71b4d34dd2c2ba35ed2f0214cc0d` | Exact 475-file manifest and application receipt. |
| Final UI `173102-5c08f5b2` | `3847a157feba71b4d34dd2c2ba35ed2f0214cc0d` | Nine cases, 34 captures, 14 readiness attempts; accepted with the separate semantic reviews below. |

The PDF unit gate was not rerun after either harness correction. The native
execution receipts explicitly reuse its original receipt and verify unchanged
application, installer, test/fixture/helper, native runner, tooling, selector
and attribute inputs. The later commits change the harness and qualification
documentation. No production PDF defect or renderer fix is claimed.

The retained local protocol proof also passes 14 top-level tests. Its three
new parents contain 6 failed-output, 14 invalid-metadata and 3 success subcases;
11 standalone existing PDF tests make 34 terminal cases. These tests validate
renderer output, not WinRT decoding. The native UI is the separate observed
format/recovery evidence. No full baseline rerun is claimed; historical
388/0/0 at `b3d46b9` remains historical.

### Observed UI sequence and bounded acceptance

The nine records cover exact TXT baseline, valid default PDF page one,
damaged PDF, TXT recovery, valid-PDF recovery, damaged-PDF reopening, final
TXT recovery, exclusive release of both PDFs, and normal close/original bytes.
The four PDF states become eligible and stable on attempts **3 / 4 / 3 / 4**.
Both damaged selections finish in `windows_fallback` with an explicit visible
error; the valid selections use `default_page`. Both TXT recoveries match their
initial preview with **zero changed sampled pixels**, within the unchanged
30-sample recovery tolerance.

All **34 final original captures** are covered by separate semantic reports:
17 full frames through 10 directly inspected exact-hash representatives, and
17 preview crops through six representatives. **Seven accepted preview states
pass bounded semantics**: three clean TXT states, two readable valid-PDF states
and two explicit native errors. Retained blank/grey/loading transitions are
not promoted to successful final document or error states.

The source/fixture bytes and registry are unchanged, the destination/journal
guards pass, both PDFs are exclusively released, and the owned host closes
normally with exit 0 and no forced teardown. Original `feature_accepted`,
`visual_accepted` and capture `visual_reviewed` fields stay false. Acceptance
is recorded in the separate derived reviews and this summary.

**Control limit:** four generic host sizing controls (`zoom_out`,
`zoom_actual`, `zoom_in`, `fit`) remain enabled on the native error while Width
is disabled. Their loaded-document readiness and click behavior remain
unaccepted; `native_error_control_readiness_accepted` remains false. This
milestone does not silently close that remaining DCM-34 issue.

### Evidence integrity and publication

The independent source/archive audit records **172 PASS / 0 FAIL / 1 LIMIT**.
Each unit/build source manifest contains 475 files: 456 match only the
observed text LF-to-CRLF checkout conversion and 19 match exact Git blobs.
The 59-byte damaged PDF is pinned with `-text` and remains byte-identical.
The original history and final bundles contain **241 + 108 = 349 files**,
all preserved exactly; both archive/manifest/transfer hash sets are pinned.
All three build receipts record the same 7,367,168-byte application SHA-256:
`d8df2a791b736ecbee517f7efaff742a5b64271516c97fed96cc604a7ee45b7b`.
Executable bytes were not part of the report transfer; this is receipt-bound
artifact identity, not an independent rehash of a transferred executable.

The single audit limitation concerns the first failed UI wrapper: stderr
stopped it before its CLI command row/stdout was saved. Its original FlowDev
envelope and failed UI receipt remain hash-bound. It is preserved as failed
history, separately from the earlier concurrency hold and later phrase miss.

See [the native summary](evidence/dcm-34-pdf-damage-native-summary.json).
The report repository is `laxmanxp/flowverse-reports`, prefix
`apps/dcm-quicksort/dcm34-20261114-173102/default-pdf-3847a15`.
Its immutable report commit is `d65d3bae8cd5e9e5db773d93cace361ac8c361d3`.
On 15 November, all 398 published file paths, blob hashes, sizes and modes
were independently verified against the local evidence bundle.
Documentation synchronization records these native results and is not another
application acceptance run.

### Reproduction and source policy

For a change affecting PDF unit inputs, use the registered sequence:

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort pdf-failure-unit
flowdev test dcm-quicksort ui-tooling
flowdev test dcm-quicksort build
flowdev test dcm-quicksort ui-pdf-failure
```

Set `DCM_BUILD_RECEIPT` to the successful exact-current-source build receipt
before the UI gate. Use Desktop Commander and the registered profile described
in `CONTEXT.md`, with one active native qualification at a time. The recorded
harness-only retries reused the unchanged unit result with explicit source
comparison; this is not a general permission to reuse stale unit evidence.

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
At that historical checkpoint, the correction still needed its own update,
tooling, build and UI gates. The subsequent results are recorded above. The
unchanged Go units retain their `499b015` source attribution.

## Second attempt retained: explicit native error phrase missing from the harness

At pushed correction `dd1f03ae42b44d0003df176a79eb591ac2a08f06`, the registered
update, tooling and fresh build pass. Unchanged Go/unit/fixture/helper inputs
retain the passing units at `499b015`; no unit rerun is claimed.

UI `172254-9e118143` passes the initial TXT baseline and valid PDF, then stops
on the first damaged PDF. Its 66 original captures and 31 readiness attempts
(4 valid-PDF attempts and 27 damaged-PDF attempts) are retained. From damaged
attempt 3 onward the state identifies the real Windows fallback. The scoped
Edge/WebView accessibility text and the reviewed original full frame both show
`We can't open this file`, followed by `Something went wrong.` and `Refresh`.
The prior teal PDF is absent. On the final attempt, the sole failed condition
is that the harness's explicit-error allowlist omits that exact phrase; pixel
and identity conditions pass, with zero sampled differences from the previous
frame. Only the exact observed phrase is added. The footer alone and
`Something went wrong.` alone remain insufficient.

The original failure remains failed, with two accepted cases; this historical
run did not complete the recovery sequence. Native source/fixtures/registry remain unchanged,
the journal is empty and the host exits normally with code 0. Four generic
host sizing controls remain enabled on the native error while Width is disabled;
this is recorded separately and is not accepted as loaded-document readiness.
No production PDF bug or renderer change is inferred from the phrase mismatch.

## Historical DOCX fallback probe - superseded by accepted fix above

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

- Enabled generic sizing controls on the native error: loaded-document
  readiness and click behavior remain unaccepted.
- Default PDF protected-file behavior and recovery.
- Observed default-renderer loading interruption: change selection while
  loading, then prove late results cannot repaint the new preview/status.
- Resolve the confirmed DOCX fallback defect and document remaining advertised
  Office damaged/protected-handler coverage or limits.
- Keep worker/file-lock lifecycle qualification under DCM-35 and the final
  candidate/package rechecks under DCM-46/47.

This accepted damaged-PDF milestone does not close DCM-34 or change the
numbered count: **32 complete / 18 remaining; next DCM-34**. Historical DOC/XLS/PPT acceptance, original failures and the
388/0/0 baseline at `b3d46b9` retain their original source attribution.
