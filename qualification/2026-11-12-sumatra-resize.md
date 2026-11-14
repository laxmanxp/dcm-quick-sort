# Sumatra resize corrections — toolbar space and repaint trails

12 November 2026 · DCM QuickSort v1.0.71 · `release/v1.0.71`.
Product/native-tested source: `24ff2f243db4c5d8d29b8d64ce81b07c11112efe`.

## Result

The narrow-preview toolbar and host repaint corrections are implemented and verified at the tested sizes. **Two consecutive focused runs passed 23 checks each**, with the same executable/source. All **20 unassisted full-window captures from the first passing run** were individually inspected. Their 19 PDF-content crops and four explicitly repainted diagnostic references were checked by the test but were not separately visually reviewed. The second run's images were not separately reviewed.

**DCM-SUMATRA-UI-02 and DCM-UI-RESIZE-01 are recorded fixed within this tested matrix, not as universal layout approval.** No numbered step is newly closed. DCM-01–18 and DCM-20–27 remain complete; DCM-19 blocked; DCM-28/29 pending broader provider qualification. Sumatra stays opt-in, the old Windows.Data.Pdf route stays default, and no installer/runtime release occurred.

## Reproduced repaint defect and correction

The first bare-background detector was insufficient: it excluded every current child rectangle, so it missed stale paint *inside* buttons and static footer text. Its successful run at `b0201f9b` was rejected as defect acceptance after screenshot review still showed the trails. No application fix is inferred from that early PASS.

The strengthened test compares an unassisted screenshot with a separately captured explicit-redraw reference. It excludes the PDF and dynamic editable/list regions, but includes the affected host buttons, labels and background. The initial, non-resized screen calibrated at zero changed samples. Before the product fix, the restored screens differed at **2,426**, **2,171** and **2,426** sampled pixels respectively, out of **107,276** host samples per comparison. The native run at `1b5cadca` therefore failed with `HOST_REPAINT_DIFFERENCE`, while preserving fixtures/source and closing normally. The raw restored screenshot and its repaint reference were both opened: the reference removes the visibly stale Reload/button/footer fragments.

The application now invalidates and erases the completed host layout and its children after moving the controls. The repaint is queued, not a synchronous wait on the foreign PDF renderer. At the final source, the same initial/restored/repeated comparisons all report **zero changed sampled pixels**, and the unassisted screenshots themselves are clean. A diagnostic redraw is still performed only *after* each raw screenshot; the test cannot pass simply by repairing the screen before measuring it. Original raw evidence is retained.

## Narrow-toolbar correction and its tradeoff

The host reserves more preview width while Sumatra is attached by borrowing width from the two side panels, with bounded minimum widths. Original panel sizing returns after the PDF viewer is released. The Sumatra binary, its commands and no-print/no-write policy are unchanged. The rightmost **Aa / Match Case** control is fully visible in the reviewed smaller and narrower captures, whereas the preserved pre-fix captures clipped it (and, at the narrowest size, another find control).

The tested outer host sizes, in the host's coordinate context, were **1280×820**, **1216×756** and **1152×756**, followed by restoration and three additional shrink/restore cycles. The fixed-window style was not changed; automated resizing does not add a new user resize feature. This does not certify every DPI, screen or width.

**Remaining layout tradeoff:** narrowing the side panels makes the existing Processed-heading crowding more pronounced; the narrow filter row also still has overlapping Size/Sort content. These remain explicitly open under `DCM-UI-LABEL-01` / DCM-46. The full application UI is not visually approved, and the sidebar minima are engineering bounds, not proof that every sidebar label is readable.

The additional MSAA toolbar-bounds detector is supplementary only. Its returned button coordinates did not reliably identify the observed far-right clipping at the tested scaling, even after moving proxy construction into a physical-DPI context. It must not replace screenshot review or be used as sole proof that every icon is fully visible. The retained command-set check verifies that visible command names were not removed. Future improvements should qualify the coordinate conversion before relying on that geometry assertion alone.

## Fresh native results

All final runs below use `24ff2f24`; counts are separate selections, not added together.

| Selection | Result | Report under `D:\FlowverseReportsRepo` |
|---|---|---|
| Full application baseline | **367 passed, 0 failed, 1 skipped**; 368 parent results; exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261112-082052-d55bc2d2` |
| Focused Sumatra run 1 | **23 checks passed**; normal host exit 0; 20 full captures reviewed | `dcm-quicksort-ui-sumatra-20261112-081912-e85ac86f` |
| Unchanged Sumatra repeat | **23 checks passed**; normal host exit 0; zero repaint differences | `dcm-quicksort-ui-sumatra-20261112-082224-ce406ad4` |
| Text regression | **17 checks passed**, command exit 0, source unchanged | `dcm-quicksort-ui-text-20261112-082129-e66a09e9` |
| Image regression | **23 checks passed**, command exit 0, source unchanged | `dcm-quicksort-ui-image-20261112-082149-e002f5b0` |
| Launch/empty-state regression | **12 checks passed**, command exit 0, source unchanged | `dcm-quicksort-ui-launch-20261112-082213-7ab24fa1` |

One new Go parent test covers sidebar allocation through nine explicit width cases and an invariant loop over 857 widths. That explains the increase from 366 to 367 passing parents; iterations are not additional test totals. Five focused local parents passed and the full Windows test executable cross-compiled locally. The initial local undefined-helper compile failure is preserved, but is not claimed as a native behavior reproduction. The native repaint RED is the actual product-failure evidence.

The single skipped test remains `TestRealGeneratedPPTXIntegration`; the identified external `DCM_REAL_PPTX` fixture still must run before its preview/package acceptance. Text/image/launch regression images were not separately reviewed here. Installer, packaged release and default-provider PDF UI suites were not rerun.

The focused workflow retains page entry, both page boundaries, accessible navigation/zoom/Fit, actual page-content checks, resize containment, internal Open refusal, PDF-to-text release, and representative Rename/Move byte preservation. All test fixtures and source identities remained intact; the host exited normally and recorded owned viewer processes were checked for cleanup. These are automated fixture tests, not human acceptance or full Batch Move/Undo qualification. No actual printer activity occurred.

## Preserved attempts

- `b0201f9b`, `dcm-quicksort-ui-sumatra-20261112-080144-ddb5f6ae`: initial frame geometry assertion failed before behavior checks; cleanup exit 0. Exact cause is not established. No guard was bypassed.
- Same source, `dcm-quicksort-ui-sumatra-20261112-080357-8a3f02e9`: early 21-check PASS with insufficient paint/toolbar geometry detectors; two resized screenshots reviewed and rejected as defect closure.
- `1b5cadca`, `dcm-quicksort-ui-sumatra-20261112-081214-0ceefaba`: strengthened comparison reproduces repaint RED; 21 preceding behavior records passed, final acceptance failed. Raw/reference/narrow screenshots reviewed. Normal exit 0 and unchanged fixtures/source.
- Final GREEN and unchanged repeat are separate new reports, not edited old receipts. Raw visual flags remain unchanged; this review and image hashes are separate. Full-window images stay on Lenovo because desktop edges are present.

Code/tests were authored locally, applied in GitHub with exact before-blob and after-SHA guards, then updated/built/tested on Lenovo through FlowDev. No remote source edits. The unrelated LibreOffice interactive session was left untouched.

## Identity and continuation

Build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-081855-7f0de675\receipt.json`.
Receipt SHA-256: `8fd19a9201b5482228ca94c6f63f3a9f342126d94aef9387bddbfa795b5435b2`.
App SHA-256: `a2bd4063a063431b8267a0ff50ed9157caea74ef3b965b391a6adc08d393a463`; **7,230,464 bytes**.
305-file source manifest SHA-256: `247d67faed56c67b277f5ee536839311942562db5122502b84faa9e10a533e1d`.
Runtime SHA-256: `719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0`.
Restriction policy SHA-256: `f82ee9c3ff09da529f67f2ee73ab716dd53678d0f52b00ddcffc9d6dc0a908aa`.
Separate evidence: `qualification/evidence/sumatra-resize-native-summary.json`.

Next: provider lifecycle qualification—rapid PDF/text/PDF switching, cancellation/error recovery, Undo, release-timeout refusal and closing with a PDF still active. Retain the narrower host label layout issue and scroll-to-readable-text qualification at 100%/Fit Width. Keep DCM-28/29 pending until their remaining provider acceptance is recorded. No default switch, printing, association/registry changes, installer replacement, merge or publication. DCM-19, external PPTX, distribution/installer obligations and actual user acceptance remain open. Rebuild after later metadata/source commits; do not rewrite receipt identities.

Technical reference consulted for repaint flags: Microsoft Learn, RedrawWindow (https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-redrawwindow). Project conclusions above come from the identified native tests and reviewed images, not from the API documentation alone.
