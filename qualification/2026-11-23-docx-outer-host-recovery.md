# DCM QuickSort native DOCX viewport recovery — Lenovo qualification (2026-11-23)

## Failure reproduced
The active development build `dcm-html-host-repair-dev.exe` (PID 32304) showed a QuickSort preview client of `1024 × 402` and Word outer host `_WwB` of `1024 × 402`, but the Word inner `_WwG` document viewport was only `806 × 308`. The same underfill occurred with a disposable fixture in a separate new instance: QuickSort/Word outer `770 × 356`, Word inner `602 × 271`, scrollbars 14px. This proves the earlier Word DPI-context-only fix was insufficient.

## Rejected attempts
- Default native `SetWindow` and `SetRect` both succeeded yet left Word's inner viewport undersized.
- Temporarily applying PMv2 context to Word's `DoPreview` as well as COM sizing (opt-in lifecycle experiment) left `_WwG=602 × 271`. This experimental branch was removed.
- A no-change `SetWindowPos` request on the Word outer `_WwB` at `770 × 356` was accepted but did not change the inner viewport.

## Successful test and implemented recovery
On the disposable DOCX instance only, an async `SetWindowPos` request to shrink `_WwB` by one pixel in each dimension, followed after 150ms by a request restoring the exact original pane size, caused Word to re-layout its inner viewport automatically. The test produced:
- Pane client `770 × 356`; Word outer `770 × 356`; Word inner `753 × 339` (17px scrollbars)
- After resizing diagnostic main window to `1100 × 760`: pane client `653 × 296`; Word outer `653 × 296`; Word inner `636 × 279`.

Implemented `app-src/word_preview_outer_repair_windows.go`:
- Only the unique verified `_WwB` immediate child of the current owned `QuickSortPreview` is targeted.
- Cross-process identity, class, parent, DPI context and measured dimensions are validated.
- Only one recovery request per active DOCX preview. The second step occurs via a 150ms one-shot UI timer; current preview handler and pane identity are checked before restoration.
- No direct `_WwG` or scrollbar manipulation, reparenting, focus/input injection or installed application changes.
- Word native COM `SetWindow`/`SetRect` is retained, and the separate HTML preview host repair is unchanged.
- `DCM_WORD_OUTER_REPAIR=off` disables the recovery for a rollback comparison.

## Verification status
- Full `go test ./... -count=1` on Laxman_Lenovo passed.
- `go build` succeeded for `dcm-docx-viewport-corrected-dev.exe`.
- The normal development build is running on Lenovo (PID 30300), without replacing the installed application.
- Automated Win32 geometry acceptance passed at two sizes in the isolated fixture, but the user's actual PFMS document and file switching still require visual acceptance.
