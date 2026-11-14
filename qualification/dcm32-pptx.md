# DCM-32 — focused PPTX qualification

Recorded 14 November 2026 (Asia/Kolkata). Standalone DCM QuickSort v1.0.71,
repository `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.

**PARTIAL: native parser qualification passed; two-slide visual acceptance is
unfinished.** The numbered plan remains **30 complete / 20 remaining**, with
DCM-32 next and DCM-19 still blocked on a disposable Explorer integration
environment. The last confirmed Lenovo checkout is
`7f4690616cc50d206b3d43953a3c2a509fa8c928`. The application source is unchanged
from `ec7a02349e9f53c6fcfd889cf21f800f36c68b4a`.

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

## Confirmed native progress and retained failure

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

## Image review is incomplete

Only two of the initial run's six captures were retrieved and reviewed:

- `02-native-slide1.png`: the full window shows the actual main-slide title
  `Bridge Inspection`, date `September 2026`, and the bridge picture correctly.
- `02-native-slide1.preview.png`: the image is retained, but the old preview crop
  cuts off the title. It is not a complete slide-content capture.

Both available PNGs were hash-verified. The other four images have not been
retrieved or reviewed. No visual acceptance is claimed for slide 2, return to
slide 1, text recovery or reopening from unreviewed images. The raw six-record
PASS and its false feature/review flags remain unchanged.

The locally implemented continuation binds native input to the provider
hierarchy observed in the initial run and adds 1 -> 2 -> 1 navigation. It also
derives the preview crop from the saved full-window screenshot rather than a
separate fixed-offset capture. Native compilation, deployment and execution of
this continuation remain pending; a source change alone does not resolve the
observed crop limitation or establish navigation acceptance.

## Focused registered gates

`pptx-unit` supplies and hashes the tracked fixture, runs the existing focused
native runner, checks source-manifest membership, and requires exactly one
run/pass event for the unchanged real-file test with no skips. Its verification
receipt is separate from the runner's raw evidence.

`ui-pptx` starts from an exact clean-source build receipt. It isolates the app's
profile and fixtures, checks native ownership, provider status, actual screen
pixels and file bytes, and captures TXT → PPTX → TXT → PPTX reopening and normal
close. The first-slide colour counters reject absent picture pixels but do not
establish text semantics or full fidelity. Every image needs separate review.

The built-in fallback branch implements owned-selector 1 -> 2 -> 1 checks with
visible change/restoration; it was not the observed route in this initial run.
The installed-handler branch at `7f46906` records first/reopened slide state and
the real native hierarchy. The local continuation adds scoped native navigation
from that observation, but has not yet passed the native compile/feature gates.

## Blocker and restart

Desktop Commander stopped returning the requested observations: a three-image
read and a metadata `start_process` call timed out. The final single read-only
`.git/HEAD` probe also returned `Timeout - no response from device` at
07:18:14 UTC (12:48:14 IST) on 14 November 2026. The connection is a confirmed
current blocker. No newer device revision, update, build, compile or feature
run is inferred from these timeouts. No further production source edits were
made on Lenovo.

After connectivity is restored and the device responds, publish the local
harness continuation if not already pushed, then run from the registered profile:

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort ui-tooling
flowdev test dcm-quicksort pptx-unit
flowdev test dcm-quicksort build
flowdev test dcm-quicksort ui-pptx
```

Set `DCM_BUILD_RECEIPT` to the new exact successful build receipt before the final
command. Review every full frame and crop for slide 1, slide 2, return, text
recovery and reopening. Raw `feature_accepted` / `visual_reviewed` values stay
false; later review decisions belong in a separate evidence record. DCM-33 is a
future step; DCM-32 is not complete.

## Evidence retained and gaps

The [progress summary](evidence/dcm-32-progress-summary.json) records the exact
known run identities and the partial review. The
[preliminary report bundle](https://github.com/laxmanxp/flowverse-reports/tree/86d0e95a02f60098f34a17d6450473d8f4c2757a/apps/dcm-quicksort/dcm32-20261114-070938/preliminary-7f46906)
contains normalized receipt and native-window-tree exports, the two original
hash-verified PNGs, the local 29-test log, and the summary. These exported text
records are not represented as byte-exact native originals. The other four PNGs
and a complete exact native evidence archive have not been recovered. Their
absence is retained as an evidence gap, not filled by a successful raw flag.

## Scope limits

The built-in renderer preserves simplified shape coordinates, text, image
relationships and table text. It uses fixed fonts and does not reproduce general
themes, masters, grouped transforms, original rich formatting, table styling,
image crop/effects, charts, SmartArt, animation or audio/video. It bounds input to
200 slides, 256 shapes per slide and 16K characters per shape, with bounded ZIP
part reads. The installed handler's representative results must not be presented
as qualification of fallback rendering on a machine without Office.

Protected/damaged preview behavior and broader lifecycle remain DCM-34/35.
User acceptance and the final installer/package/release remain open.
