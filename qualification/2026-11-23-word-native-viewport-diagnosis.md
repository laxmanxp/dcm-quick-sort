# Native Word preview viewport mismatch — Lenovo investigation (2026-11-23)

## Reproduction
Development executable: `dcm-viewport-resize-dev.exe`, GitHub branch `feature/quicksort-two-pane-history-tabs`, native DOCX handler active. On a two-pane QuickSort window, document content appears in a smaller scrollable subview and overlaps/exposes Word content in the surrounding preview region.

## Win32 evidence (read-only enumeration)
Measured with `tools/inspect-preview-viewport.ps1 -OnlyPreview` on Laxman_Lenovo:

| HWND class | Process | Screen rect (left, top, width, height) | Client |
| --- | --- | --- | --- |
| QuickSortPreview | DCM PID 31316 | (490,191,1026,404) | 1024 × 402 |
| _WwB (Word container) | WINWORD.EXE PID 24160 | (491,192,1024,402) | 1024 × 402 |
| _WwG (Word document viewport) | WINWORD.EXE PID 24160 | (491,192,806,308) | 806 × 308 |
| Word vertical scrollbar | WINWORD.EXE | (1297,192,14,308) | 14 × 308 |
| Word horizontal scrollbar | WINWORD.EXE | (491,500,806,14) | 806 × 14 |

Both app and Word child windows reported `GetDpiForWindow=96` from this inspection. This alone **does not establish** monitor DPI, so a DPI-virtualization explanation remains hypothetical. The 218px width and 94px height deficit are grounded measurements, not necessarily the cause.

The inner `_WwG` size and its scrollbars explain the apparent overlay: Word's outer container fills QuickSort, but the internal document viewport does not. `_WwG` belongs to a separate Microsoft Word process. There is only one visible `_WwB` in the enumerated host; stale duplicate Word child windows were not observed at this point.

## Relevant code checkpoints
- `app-src/main_windows.go`: QuickSort creates `QuickSortPreview` with `WS_CLIPCHILDREN`, lays out the pane in `layoutUI`, passes its rectangle to `IPreviewHandler::SetWindow` and `SetRect`, and performs a deferred/coalesced resize through `WM_SPLITTER_PREVIEW_RESIZE`.
- The previous three-column design (`8a9fa752f8` and its parent) gave the native handler a smaller outer pane; the two-pane change (`cd2fadda49`) expanded its width.
- Moving or enlarging only the host is not sufficient: measured `_WwB` already matches its host.
- The deferred resize patch was automated-test green but did not correct the measured `_WwG` viewport.

## Next diagnostic gate
Correlate `IPreviewHandler::SetWindow/SetRect` requested sizes, Word internal `_WwB/_WwG` sizes, monitor effective DPI and initial `DoPreview` timing. Test on disposable DOCX fixtures using an isolated build. Do not directly destroy or resize Word-owned child windows in production, do not replace the native Word renderer, and do not classify this as a shared preview-engine defect without evidence from other file types.
