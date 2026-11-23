# Native HTML preview viewport — Lenovo qualification (2026-11-23)

## Defect
On Laxman_Lenovo, the registered `htmlfile` native preview handler (`{f8b8412b-dea3-4130-b36c-5e8be73106ac}`) created `##MimeEdit_Server` and `Internet Explorer_Server` windows sized `616 × 285` inside a QuickSort preview client of `770 × 356`, an approximately 80% scaling discrepancy. QuickSort's native `SetWindow` and `SetRect` requests were both `770 × 356` and returned `S_OK`.

The registered HTML handler is hosted out of process. Win32 inspection identified `QuickSortPreview` as the owned pane, and `##MimeEdit_Server -> Internet Explorer_Server` as its cross-process children. The window inspection showed DPI awareness 0 on both sides in this environment.

## Rejected approaches
- Applying Word's PMv2 COM DPI-context correction to HTML did not change the `616 × 285` HTML window geometry.
- Supplying physical rectangles to the HTML `SetRect` method produced `962 × 445`, larger than the `770 × 356` pane. Those experimental physical-sizing branches were removed from the normal code.

## Qualified correction
The caller keeps the original logical `SetWindow`/`SetRect` parameters. After successful HTML `SetRect`, QuickSort requests `SetWindowPos` only on the unique, verified `##MimeEdit_Server` directly under its owned preview pane. Before the request, it verifies parentage, class, unique child match, process identity, HWND identity, DPI-awareness context, and an `Internet Explorer_Server` immediate child belonging to the same external process. The native HTML handler resizes its own inner child. No foreign inner child is touched, no subclassing or reparenting, no input or focus changes, and no active documents are edited. A scoped environment variable `DCM_HTML_COM_RECT_MODE=off` disables the repair for A/B comparison. Word retains its prior separate DPI-aware COM correction.

## Measured result
- Normal test opening, maximized: QuickSort client `1024 × 402`; HTML outer `1024 × 402`; HTML inner `1024 × 402`.
- Resized diagnostic window to `1100 × 760` outer: QuickSort client `615 × 296`; HTML outer `615 × 296`; HTML inner `615 × 296`.
- Diagnostic COM trace: 25 native preview calls across window changes. `SetWindow`, `DoPreview`, and each `SetRect` returned `0x00000000`; seven `HTMLOuterRepairRequest` entries returned accepted=true with caller DPI-awareness 0 before and after.
- `go test ./... -count=1` passed on Lenovo; `go build` produced `dcm-html-host-repair-dev.exe`.

## Scope and open gates
Qualification covers the disposable HTML fixture on Lenovo, initial/multi-size window geometry, and a clean child-window hierarchy. Manual visual inspection on the user's HTML documents, divider-drag behavior, switching HTML/DOCX/PDF and non-100% scaling environments remain outstanding. The installed QuickSort version was not replaced. All tests use the isolated `C:\DCMQuickSortCollisionAcceptance` checkout.
