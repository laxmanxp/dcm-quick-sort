# DCM-34 — preview failure and interruption states

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
