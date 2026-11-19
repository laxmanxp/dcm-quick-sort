# DCM QuickSort — rotation GUI acceptance (19 November 2026)

**Bounded engineering result: PASS.** Native Lenovo `ui-rotation` selector: **13/13 checks** and **13/13 original full-window screenshots reviewed separately**. Neither the native receipt nor its original `visual_reviewed=false` flag was changed.

- Native-tested source: `31394a446f4223b7789d6287761d0fac916df5b3`; application SHA-256: `57eebc831b630eb445c6d7bec02f962707f193b18aca0b6521ae9490c1e66f67`.
- Exact build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261119-075828-8dfa6ef8\receipt.json` (SHA-256 `e5109437d5e1933762a1abe2a7f7cda846fb8113f2ebfa5be7a30825e4f1338f`).
- Rotation test report: `D:\FlowverseReportsRepo\dcm-quicksort-ui-rotation-20261119-075921-99002df7`; native host exit code `0`. Source checkout, all fixture hashes and destination/journal remained unchanged.
- Permanent controls: `1049` left 90° and `1050` right 90° visible on supported PNG/PDF previews; disabled **without being hidden** on TXT. Full filename and extension indicators remained visible.
- Image left/right action and successive quarter-turns returned correct quadrant colors and rotated text at 0°, 90°, 180°, 270°; **12 pixel samples per image state** passed.
- PDF page 1 actually rotated clockwise and counterclockwise. The colored document-panel width/height ratio changed from **0.823** (portrait) to **1.202** (landscape); returning to 0° restored 0.823. Page 2 remained correct and readable in rotated view.

## Retained screenshot manifest

| Original capture | State and pixel evidence | SHA-256 |
|---|---|---|
| `image-000-original` | PNG 0°, 12 matched pixels | `74f70f01059d7d201226c334342b473592bbade2db9553f67fc2b6d3172b649d` |
| `image-270-left` | PNG 270°, 12 matched pixels | `c07983b82fe48d2e26aefddfa5111092feebd31d8f703be86aaab1a14a369a27` |
| `image-000-return` | PNG 0°, 12 matched pixels | `15b062ebd18e942724fadc4083b220af3e9b2560de6f00496ec07dfcbfb1a59a` |
| `image-090-right` | PNG 90°, 12 matched pixels | `2b02cc6c592db13e374bf3c78f0468550380ef64ae02ecd4a41542829dd98453` |
| `image-180-double` | PNG 180°, 12 matched pixels | `6360eb0fd46ed696cf684bb5f460432a4cef8eda30dbc2183deb219c4bc101ad` |
| `image-270-three` | PNG 270°, 12 matched pixels | `b5eb1fff0ca0eba58408b6bc15bd6b4821f18dd20e5cd6017630f6c080397f12` |
| `image-000-four` | PNG 0°, 12 matched pixels | `15b062ebd18e942724fadc4083b220af3e9b2560de6f00496ec07dfcbfb1a59a` |
| `text-buttons-visible-disabled` | TXT, visible disabled buttons | `e753659f6dbefad4a5c628bb71771ac1259be14aa5e48e1eb8d25e32f75037ba` |
| `pdf-p1-000-original` | PDF page 1, 0°, ratio 0.823, 7673 panel samples | `0a32924b63d4162bd80b41fe2daf2bdfc3511f966424c1a3a67eaaa8a195aa6b` |
| `pdf-p1-090-right` | PDF page 1, 90°, ratio 1.202, 13070 panel samples | `f3dd79cdfb7030958fe215c0d677196a65f1a0e6ee4d8c275ff43bd6f7349c91` |
| `pdf-p1-000-restored` | PDF page 1, 0°, ratio 0.823, 7673 panel samples | `b52381444439d5a2ae502cdaa1270b3a90010be6ac15073de6f16ab7b37c1977` |
| `pdf-p1-270-left` | PDF page 1, 270°, ratio 1.202, 13040 panel samples | `f47565db97a5e2ebff452a71de5fcfe6ef9e0a180e61393890f279ff7c2b083c` |
| `pdf-p2-270-left` | PDF page 2, 270°, ratio 1.204, 12727 panel samples | `ea6f580c76c182a27e96c3849938c2fd2320d56d21f4f7c9a9b0c4cb1c658448` |

## Optional Sumatra provider — separate incomplete run

`ui-sumatra` on the same source recorded **14 preliminary checks** and **32 screenshot artifacts**, but its overall result is **FAILED/INCOMPLETE**, not a pass. An unrelated `Xaml_WindowedPopupClass` with PID 4048 occluded a later screenshot and the normal capture guard refused it. This guard was not bypassed.

The valid first-page screenshot `01-embedded-first-page.png` (SHA-256 `54ffe8e25c6bb9898343b26241d6003f19cc060f391f87c9b6aefdad86765968`) was separately inspected: **both QuickSort rotation buttons remain visible but disabled**, and Sumatra exposes its own rotation controls. Report and failed receipts remain at `D:\FlowverseReportsRepo\dcm-quicksort-ui-sumatra-20261119-080510-edf17856`. This does not certify the full optional-provider workflow.

## Scope and release status

Native control messages are not physical mouse/touchpad user acceptance. No arbitrary DPI/window-size blanket claim; only observed Lenovo geometry was checked. The permanent-toolbar wiring is protected by `TestRotationToolbarIsPermanentAndWired`. Retain the original failed harness history (C# assembly reference and PowerShell selection-scope guard) and the later unrelated Sumatra occlusion failure.

**50-step plan unchanged: 44 completed / 6 remaining.** DCM-19 blocked; DCM-34 in progress; DCM-47–50 pending. PR #1 remains draft and unmerged. Final FileFlow Suite packaging depends on Flow Search and DeepSearch readiness; no installation, file association or publication.
