# DCM-32 — focused PPTX qualification

Recorded 14 November 2026 (Asia/Kolkata). Standalone DCM QuickSort v1.0.71,
repository `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.

**PARTIAL: the existing PPTX preview is under focused qualification.** The
numbered plan remains 30 complete / 20 remaining. No new numbered step closes
until the exact-source native tests and actual slide captures are reviewed.

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

The existing local parser/Open XML gate passes **29 top-level tests, 0 failures,
0 skips**, including the unchanged real-file integration test. This is parser
coverage; it does not establish visible native rendering.

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

The built-in fallback branch checks its owned slide selector through 1 → 2 → 1
and requires visible change/restoration. The initial installed-handler branch
records the real native window hierarchy and first/reopened slide without
guessing foreign controls. Its native slide navigation remains unexercised;
that branch alone cannot close the two-slide gate. Any further native navigation
adapter must be based on the actual observed embedded provider.

Run in order from the registered profile:

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort ui-tooling
flowdev test dcm-quicksort pptx-unit
flowdev test dcm-quicksort build
flowdev test dcm-quicksort ui-pptx
```

Set `DCM_BUILD_RECEIPT` to the exact successful build receipt before the final
command. Native execution and image review are pending at this preparation
checkpoint. Raw `feature_accepted` / `visual_reviewed` values stay false; later
review decisions belong in a separate evidence record.

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
