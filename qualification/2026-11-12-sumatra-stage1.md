# Sumatra PDF integration — first native milestone

12 November 2026 IST. Native receipts below are dated 11 November UTC.
DCM QuickSort v1.0.71, branch `release/v1.0.71`.
Tested source: `b05024332f018b544f7d1e704c5e017fac66f364`.

## Result and boundaries

Sumatra now embeds inside QuickSort's actual preview pane as an **optional PDF provider**. The focused native test passed **8 behavior checks**. Six full-window screenshots and five content crops were individually inspected. The existing Windows.Data.Pdf provider remains the default; no Sumatra executable has been bundled or published.

**DCM-28 and DCM-29 are not complete. No numbered step is newly closed.** Completed steps remain DCM-01–18 and DCM-20–27; DCM-19 remains blocked. This is an implemented and partially qualified integration, not final visual approval or user acceptance.

The user explicitly approved a focused Sumatra PDF integration after DCM-27. That authorizes this PDF-provider change; it does not authorize a UI-framework migration, printing subsystem, suite merger, production installation or publication. Text and image providers are unchanged.

## Implementation

The provider launches the pinned, unmodified Sumatra 3.6.1 executable as a separate process using `-plugin <owned preview HWND>`, a private profile, `-restrict`, and an exact adjacent policy denying disk writes/open-file commands, printing, registry preference changes, external navigation and copying. Policy/runtime checks fail closed; neither PATH nor the user's default installed viewer is used.

Startup creates the process suspended and assigns it to an owned kill-on-close Windows job before resuming. A host timer observes attachment/liveness rather than waiting for rendering. Child-window operations verify its PID and immediate parent. Release first requests closure, then can terminate only the owned job and requires process-exit confirmation before the tracked file operation proceeds. Asynchronous child hiding avoids a synchronous cross-process hide call. The normal/default PDF path remains available.

Integration hooks cover selection/unload, pane sizing, Rename/Move admission and individual/batch Undo release. **Hook coverage is not the same as native acceptance of every route**: this milestone exercises representative Rename/Move, not full Undo, forced-failure recovery, every close/cancel path or every file-operation variant.

Opt-in currently uses `DCM_PDF_PROVIDER=sumatra` plus an absolute `DCM_SUMATRA_EXE` pointing to the exact runtime with the exact no-print policy. This is a development/provider-selection mechanism, not a finished end-user Settings screen. A dedicated FlowDev `ui-sumatra` check creates its own isolated runtime copy and fixtures. No original runtime, installed app, association or Explorer registration was changed.

## Native checks that passed

| Check | Observed result |
|---|---|
| Embedded viewer | Exact test runtime, expected host parent process and child window; PDF page one visible. |
| Next page | Page two's orange content replaces page one's teal content. |
| Last-page boundary | Next at the last page stays on page two. |
| Previous page | Page one's actual contents return. |
| Internal Open command | No file picker opens and the same page remains displayed. |
| Switch PDF to text | The owned viewer exits and exclusive access to the PDF succeeds. |
| Rename PDF | QuickSort releases its old viewer, renames the intended PDF, and reopens the renamed document; contents remain unchanged. |
| Move PDF | QuickSort releases the viewer, moves the renamed PDF to the isolated Vault destination, and preserves its contents. |

The three unselected text files remained byte-identical. The app exited normally with code 0 and none of the three recorded owned Sumatra processes remained alive with its original identity. The source checkout remained unchanged. The final screenshot shows the two tracked Rename/Move entries and restored text preview.

Navigation was invoked through bounded commands to the verified owned Sumatra frame, not physical mouse clicks or human testing. Actual screenshot content and colors were checked independently. Only a two-page fixture was used; the first-page Previous boundary, page-number editing, zoom, scrolling, repeated resizing and rapid-switch stress are still pending.

The exact policy sets `PrinterAccess=0`; no print command or physical printing was attempted. Only the internal Open route was actively challenged here, so the run does not constitute exhaustive validation of all restricted commands or an operating-system security sandbox.

## Visual finding: keep optional

The page-number label/input at the top of the embedded Sumatra toolbar is visibly clipped. QuickSort's own disabled sizing controls also remain above the viewer, making the integration visually unfinished. This is recorded as **DCM-SUMATRA-UI-01**. The exact clipping cause is not yet established. The strict no-print/no-write policy stays in force; printing will not be enabled as a layout workaround.

The document pages themselves are readable and contained within the preview pane. Passing content checks do not waive the toolbar issue or complete DCM-28/29.

## Fresh regression results

All final results below use `b0502433`:

| Selection | Result | Report under `D:\FlowverseReportsRepo` |
|---|---|---|
| Full native application suite | **365 passed, 0 failed, 1 skipped**; 366 parent results, exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261111-234138-747a8dff` |
| Optional Sumatra stage 1 | 8 behavior checks passed; 11 captures inspected; normal exit 0 | `dcm-quicksort-ui-sumatra-20261111-233859-93834514` |
| Default-provider text regression | 17 cases passed; command exit 0, source unchanged | `dcm-quicksort-ui-text-20261111-234201-8b3fe3f2` |
| Default-provider image regression | 23 cases passed; command exit 0, source unchanged | `dcm-quicksort-ui-image-20261111-234218-04fe2476` |

Three new Go parent tests account for the increase from 362 to 365 passing tests. They exercise argument/path validation, opt-in behavior, rejection of altered policy/untrusted runtime and read-only document preflight. Local focused tests and Windows test cross-compilation also passed. Existing assertions were not removed.

The sole skip remains `TestRealGeneratedPPTXIntegration`; the identified external fixture is still required. The text/image regression images were not separately reviewed in this milestone. UI checks are separate from Go counts; installer/package and the old PDF UI suite were not rerun.

## Build, failures and evidence

Exact build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261111-233847-d6a39720\receipt.json`.
Receipt SHA-256: `e7429e5a22abcd1923dc2e1ca4224a28693c600afbee65e127f042ffbc777713`.
App SHA-256: `6e0da23dbd7bf67d5610e9e82b7859df851690b11caeeb38c23284e61323d984`; 7,223,296 bytes.
Runtime SHA-256: `719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0`.
Policy SHA-256: `f82ee9c3ff09da529f67f2ee73ab716dd53678d0f52b00ddcffc9d6dc0a908aa`.
Separate review: `evidence/sumatra-stage1-native-summary.json`.

The initial test-only RED at `3f6b1b42` stopped at `Deadline: embedded Sumatra child`, before the provider existed (`dcm-quicksort-ui-sumatra-20261111-232240-30170f6f`). The first implemented run at `185ba410` passed the Go baseline but stopped at `Deadline: selection 08-preview.pdf`; its diagnostic screenshot was refused as occluded/offscreen (`dcm-quicksort-ui-sumatra-20261111-233015-40552c48`). That selection timeout's exact cause was not established. Later diagnostic capture and asynchronous hiding changes preceded the passing run; they do not establish a retrospective root cause. Repeatability remains part of further qualification.

The pixel assertion was corrected to use a separate preview-content crop, excluding blue title/list chrome. This retains the expected-page color checks and full-window screenshots rather than bypassing capture guards. All 11 final captures were individually inspected; raw `visual_reviewed=false` values remain untouched and the separate review record records the inspection. Images remain on Lenovo because full-window captures contain small desktop edges.

## Next

Resolve the clipped page controls without relaxing restrictions; then finish DCM-28/29 with actual visible controls, first/last boundaries, zoom/resize, rapid switching and error/cancel recovery. Verify Undo, release-timeout refusal and closing with a PDF still active. Default-provider selection and managed-runtime packaging follow only after these gates pass.

Before bundling or publishing, qualify the exact runtime notices/source distribution obligations and installer payload/policy lifecycle (**DCM-SUMATRA-DISTRIBUTION-01**). This milestone neither distributes a runtime nor resolves licensing by assuming separate processes are exempt. Keep DCM-19, external PPTX, general layout, final installer/package and actual user acceptance open. PR #1 remains draft; no merge or publication.
