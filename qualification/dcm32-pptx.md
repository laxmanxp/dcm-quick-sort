# DCM-32 — focused PPTX qualification

Recorded 14 November 2026 (Asia/Kolkata). Standalone DCM QuickSort v1.0.71,
repository `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.

**COMPLETE for the representative two-slide fixture through the installed native
PowerPoint preview handler at the observed Lenovo geometry.** The numbered plan
is **31 complete / 19 remaining**, with DCM-33 next and DCM-19 still blocked on a
disposable Explorer integration environment. The registered Lenovo update and
final gates ran at pushed source `b3be3a016a92759ed88eeb364645f56fb5cc7916`.
The application source is unchanged from
`ec7a02349e9f53c6fcfd889cf21f800f36c68b4a`. User acceptance and the final package
remain separate.

## Recovered starting point

DCM-31's completion records were published at
`8671cc7bc9557ffca65de53ca79ace1156f65d00` and delivered through the registered
`flowdev update dcm-quicksort` on Lenovo. The XLSX implementation remains the
accepted `ec7a02349e9f53c6fcfd889cf21f800f36c68b4a` source, with retained native
evidence; documentation publication does not imply another product test run.

The historical Lenovo `07-slides.pptx` is 37,897 bytes with SHA-256
`c8d9b1597e64de6426356a923c3fb5dc288a1841391431be72c6b2f6b517477d`.
It contains `FILEFLOW OFFICE PREVIEW` and `SECOND SLIDE`, and no embedded media.
The unchanged `TestRealGeneratedPPTXIntegration` correctly fails against it:
the expected `Bridge Inspection` / `September 2026` content is absent. That
is a fixture mismatch, not a demonstrated product regression. The test's
assertions were preserved.

## Reproducible representative fixture

The committed [fixture manifest](../tests/fixtures/dcm32-slides.manifest.json)
pins `tests/fixtures/dcm32-slides.pptx`: **31,109 bytes**, SHA-256
`af42fff55297109195b95039d2ac39c5d53b1fae23b59e7b40a7184d679b93f1`.
The [generator](../tests/fixtures/generate-pptx-fixture.py) uses fixed package
timestamps and synthetic content; regeneration was verified byte-identical.
Qualification consumes the committed file instead of regenerating it on Lenovo.

- Slide 1: `Bridge Inspection`, `September 2026`, and a real embedded bridge
  illustration with teal water and an orange deck.
- Slide 2: `Payments` and a real table containing `Scheme` / `Amount`,
  `Bridge` / `125000`, and `Road` / `86000`.

The local and corrected native parser/Open XML gates each pass **29 top-level
tests, 0 failures, 0 skips**, including the unchanged real-file integration
test. This is parser coverage; visible native rendering has its own gate.

## Final registered native acceptance

After connectivity recovered, `flowdev update dcm-quicksort` received the pushed
`b3be3a016a92759ed88eeb364645f56fb5cc7916` qualification source. The following
registered selectors ran serially through the standalone FlowDev profile. Every
FlowDev command exited 0.

| Gate | Native report suffix | Result |
|---|---|---|
| `ui-tooling` | `093847-0a9a5a0c` | PASS: 29 PowerShell scripts parse without errors; the native C# helpers, including scoped PPTX input, compile. |
| `pptx-unit` | `093851-22afbd5a` | 29 passed / 0 failed / 0 skipped. The unchanged real-file test emits exactly 1 run / 1 pass; separate fixture verification passes and preserves the file. |
| `build` | `093918-4f66ddaf` | PASS at exact source; 444 source-manifest entries match Git with recorded CRLF checkout conversion. |
| `ui-pptx` | `094009-df395f56` | Eight passing workflow records, 18 captures, eight native capture attempts, two scoped queued key actions and `all_fixture_slides_captured=true`. All images separately reviewed. |

All reports are under `D:\FlowverseReportsRepo`, using prefixes
`dcm-quicksort-ui-tooling-20261114-`, `dcm-quicksort-regressions-20261114-`,
`dcm-quicksort-build-20261114-` and `dcm-quicksort-ui-pptx-20261114-` respectively.
The UI run consumes the exact `093918-4f66ddaf\receipt.json` build receipt. Its
native application is 7,365,632 bytes, SHA-256
`7f51956e67dd7e0a33eef51a6ad8cf8a742ad6e2d64585348614eaa485e35827`.
The source and fixture bytes are unchanged, the operation journal stays empty,
exclusive presentation access succeeds after switching to TXT, and the host
closes normally with exit code 0. This is an application qualification build,
not final installer/package qualification.

## Retained preliminary progress and failure

| Phase | Source / native report suffix | Result and limit |
|---|---|---|
| First `pptx-unit` wrapper | `3b2978fb50d207de9b96dec29ba535483e1fcbb3` / `070557-4c20ba28` | Core tests passed, but the wrapper failed while handling the source-manifest array on Windows PowerShell 5.1. Preserve both outcomes. |
| Corrected `pptx-unit` | `7f4690616cc50d206b3d43953a3c2a509fa8c928` / `070759-9c7fff5a` | 29 native tests passed, zero failures/skips; the real-file test emitted exactly one run and one pass. The separate fixture-verification wrapper passed. |
| Exact native build | `7f46906` / `070857-05f16c29` | PASS. This receipt belongs to that checkout and cannot be reused after the harness changes. |
| Initial `ui-pptx` observation | `7f46906` / `070938-51df3033` | Raw PASS: six machine records and six captures. `all_fixture_slides_captured=false`; native slide navigation was not exercised. |

All report paths are under `D:\FlowverseReportsRepo`. The unit reports use
`dcm-quicksort-regressions-20261114-`, the build uses
`dcm-quicksort-build-20261114-`, and the UI report uses
`dcm-quicksort-ui-pptx-20261114-` before the suffixes above.

Commit `7f46906` changes only the wrapper's enumeration of the parsed JSON source
manifest. It preserves the original test assertions and the raw core receipt.
The earlier wrapper failure is a tooling failure, not a failed parser test, and
is not retrospectively relabeled as success.

The old full-suite result at `ec7a0234` remains 383 passed, zero failed and one
external-PPTX skip. The focused 29/0/0 result resolves that fixture prerequisite
for this parser gate; it does not change the historical baseline or certify a
zero-skip final package.

## Separate image review and navigation acceptance

All **18 final-run images** were retrieved and reviewed: two TXT frames plus a
full-window image and a crop for each of eight native capture attempts. Each of
the four native stages has two attempts. The first is retained as readiness
evidence; the second is the stable capture referenced by that case's
`accepted_capture`. Reviewing a readiness frame does not change its recorded
`stable=false` result. The accepted native pairs and TXT frames show:

| Stage | Reviewed visible result |
|---|---|
| First slide | `Bridge Inspection`, `September 2026` and the embedded bridge illustration, with the title contained in the new crop. |
| Second slide | `Payments`, `Scheme` / `Amount`, `Bridge` / `125000` and `Road` / `86000`. |
| Return to first slide | The title, date and bridge are restored after the scoped PageUp action. |
| TXT recovery and PPTX reopening | The expected TXT content returns; reopening presents slide 1 with its title, date and picture. The full frames retain the host shell. |

The two native input actions queue PageDown and PageUp messages only to the
bound PowerPoint `childClass` inside the owned preview pane, with process,
thread, window and fixture identity checks. Pixel change/restoration and stable
adjacent captures pass; the separate image review establishes the semantic
slide identities. This is scoped message-driven 1 -> 2 -> 1 navigation, not a
physical-keyboard test. Each new crop comes from its saved full-window frame
using the native child bounds, rather than a second fixed-offset screenshot.

Raw `feature_accepted=false`, capture `visual_reviewed=false` and the requirement
for separate semantic review remain intact in the machine receipt. The
[native summary](evidence/dcm-32-native-summary.json) records the review decision
separately; user-test status remains `not_recorded`.

The preliminary run initially had only `02-native-slide1.png` and its crop
retrieved. The full frame shows the title, date and bridge correctly, while the
old preview crop clips the title. The other four historical images have now
been recovered and reviewed, so all six are accounted for. The old crop
limitation, six-record raw result and `all_fixture_slides_captured=false` remain
historical evidence; the preliminary run did not exercise native slide navigation.

## Focused registered gates

`pptx-unit` supplies and hashes the tracked fixture, runs the existing focused
native runner, checks source-manifest membership, and requires exactly one
run/pass event for the unchanged real-file test with no skips. Its verification
receipt is separate from the runner's raw evidence.

`ui-pptx` starts from an exact clean-source build receipt. It isolates the app's
profile and fixtures, checks native ownership, provider status, actual screen
pixels and file bytes, and captures TXT → PPTX slides 1 → 2 → 1 → TXT → PPTX
reopening and normal close. The first-slide colour counters reject absent
picture pixels but do not establish text semantics or full fidelity. Every
image needs separate review, completed for this final run.

The built-in fallback branch implements owned-selector 1 -> 2 -> 1 checks with
visible change/restoration; it was not exercised by the accepted native run.
At `b3be3a0`, the installed-handler branch's scoped navigation and crop changes
pass native compilation and the registered feature gate, with the separate
review above. This does not qualify headed fallback rendering without Office.

## Historical interruption and next step

Desktop Commander stopped returning the requested observations: a three-image
read and a metadata `start_process` call timed out. The final single read-only
`.git/HEAD` probe also returned `Timeout - no response from device` at
07:18:14 UTC (12:48:14 IST) on 14 November 2026. This was a confirmed blocker at
the preliminary checkpoint, and no product failure or newer device revision was
inferred from the timeouts. Connectivity subsequently recovered. The successful
update, native compilation, build and accepted UI run above resolve that
continuation blocker. No production source was edited on Lenovo.

The reproducible DCM-32 sequence remains:

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort ui-tooling
flowdev test dcm-quicksort pptx-unit
flowdev test dcm-quicksort build
flowdev test dcm-quicksort ui-pptx
```

Set `DCM_BUILD_RECEIPT` to a successful build receipt for the exact current clean
checkout before the final command; do not reuse an earlier checkout's receipt.
Retain and review every full frame and crop. Raw review flags stay unchanged and
review decisions belong in a separate record. DCM-32 is complete at the bounded
accepted source above. The smallest next DCM-33 task is representative legacy
DOC behavior: inspect the existing native-handler/unavailable contract and add
the smallest focused fixture and gate. Legacy XLS/PPT follow in separate baby
steps; DCM-19 remains blocked on Explorer isolation.

## Evidence retained

The [native acceptance summary](evidence/dcm-32-native-summary.json) binds the
final FlowDev/native receipts, source manifest, pinned fixture, all 18 new
images, separate semantic review and recovered preliminary history. Its report
publication identity points to
`apps/dcm-quicksort/dcm32-20261114-094009/native-navigation-b3be3a0` in
`laxmanxp/flowverse-reports`. Raw machine results are retained alongside the
separate review, including the original wrapper failure.

The [progress summary](evidence/dcm-32-progress-summary.json) preserves the
earlier interrupted checkpoint's known run identities and partial review. The
[preliminary report bundle](https://github.com/laxmanxp/flowverse-reports/tree/86d0e95a02f60098f34a17d6450473d8f4c2757a/apps/dcm-quicksort/dcm32-20261114-070938/preliminary-7f46906)
contains normalized receipt and native-window-tree exports, the two original
hash-verified PNGs, the local 29-test log, and the summary. These exported text
records are not represented as byte-exact native originals. The later recovery
of the other four historical PNGs and native originals is recorded in the final
evidence; it does not change the preliminary bundle's original contents.

## Scope limits

The built-in renderer preserves simplified shape coordinates, text, image
relationships and table text. It uses fixed fonts and does not reproduce general
themes, masters, grouped transforms, original rich formatting, table styling,
image crop/effects, charts, SmartArt, animation or audio/video. It bounds input to
200 slides, 256 shapes per slide and 16K characters per shape, with bounded ZIP
part reads. The installed handler's representative results must not be presented
as qualification of fallback rendering on a machine without Office.

Qualification is limited to the installed PowerPoint handler, the representative
two slides, scoped queued Page keys and the observed window/display geometry.
It does not certify physical keyboard input, arbitrary presentation fidelity,
all providers or display scales. Protected/damaged preview behavior and broader
lifecycle remain DCM-34/35; the final normal-app workflow remains DCM-46.
User acceptance and the final installer/package/release remain open under
DCM-47 and the remaining release steps.
