# DCM-30 — representative DOCX preview accepted

Recorded 14 November 2026 (Asia/Kolkata). DCM QuickSort v1.0.71, branch `release/v1.0.71`.
Tested production source: `8eb1470a891ea3454f1738d58191fdb90284ab42`.

## Result

**DCM-30 is complete for the demonstrated representative DOCX content workflow through Lenovo's installed Microsoft Word preview handler.** The fresh registered FlowDev run passed seven recorded checks. All five full-window screenshots were inspected separately and show the intended document/text content and complete shell controls in those frames.

The checklist is now **29 complete / 21 remaining**: DCM-01–18 and DCM-20–30 are complete. DCM-19 remains environment-blocked. The next numbered step is **DCM-31 — verify XLSX preview**. These are checklist counts, not a percentage of product development or human acceptance.

The current continuation records acceptance and durable context. The production fix was already pushed at `8eb1470a`: after a Windows preview handler loads successfully, the footer identifies its current type (`Windows preview: DOCX`) rather than retaining `Plain-text preview: TXT` from the previously selected file.

## Exact pipeline and identity

The local checkout and the dedicated Lenovo FlowDev registration both resolve to `https://github.com/laxmanxp/dcm-quick-sort.git`, branch `release/v1.0.71`. Production source is `app-src/`; installer source is `installer-src/`.

The fresh device commands were:

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort ui-docx
```

Update reported already up to date; the feature check returned PASS at `8eb1470a`, exit 0. The registered selector invokes `tools/test-ui-docx.ps1`. The previously successful exact-source build receipt was reused after its identity checks; this was not a new build or an installed-app test.

- Profile: `D:\FlowverseReportsRepo\dcm-quicksort-candidate\tools\flowdev-dcm`.
- Runtime: `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`.
- Build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261113-074831-dccd7e8e\receipt.json`.
- Native EXE SHA-256: `8c682baab23b4ea831314ae0ff1bc2c9a98a1180d05365fb6968014bee9f9ef5`.
- Native EXE size: 7,336,960 bytes; Go 1.27.0 Windows/amd64.
- Build source manifest: 422 tracked files, SHA-256 `e6f9c84e1c2b84fe184ae5b00469301cae5a1651ad98cb27684b0100ecc80f6c`.
- Fresh UI report: `dcm-quicksort-ui-docx-20261113-183530-acd8af62`.
- UI interval: 13 November 2026, 18:35:30–18:35:55 UTC; **14 November, 00:05:30–00:05:55 IST**.

No production source was edited on Lenovo. The harness used its existing disposable fixture/configuration directories and original capture/ownership safeguards. It did not change Office associations or install a preview provider.

## What the screenshots and readbacks establish

| Stage | Reviewed observation |
|---|---|
| Initial text | The selected alpha fixture and its plain-text footer agree; shell controls are painted. |
| Historical letter DOCX | The actual Government of Odisha — Gurundia Block heading, Road Repair subject, ₹25,000, FlowDoc line and Block Development Officer signature are readable. The footer identifies DOCX. |
| Structured uppercase `.DOCX` | DCM-30 FIELD REPORT, bold and regular text, accented Café, table rows Bolts/12 and Panels/3, a green embedded image and END DCM-30 ALPHA are visible. |
| Return to text | The correct alpha text returns without stale DOCX content; the native preview subtree is detached. |
| Reopen the letter | The original letter and DOCX footer return, with the file list, labels and buttons fully painted in the current frame. |

The script also verifies exclusive access to both DOCX files after switching away, unchanged names/bytes for all six disposable source fixtures, an empty destination and operation journal, normal host exit 0, and unchanged committed source. It checks direct owned children; it does not terminate or claim shutdown of a shared Microsoft Word host.

The historical letter is pinned to SHA-256 `11ca20995a5da66033c8dd75cc55a145bc6179bf189ac7d8bfaf9d90562a72a7`. The second document comes from the committed `tests/native/docx-content-fixture.json`. Each generated ZIP's byte hash is retained in its own receipt; differing ZIP timestamps between runs do not imply that fixture checks within a run were waived.

Raw receipts retain `visual_reviewed=false`. The independent decisions and image hashes are recorded in [the review summary](evidence/dcm-30-native-summary.json); the raw receipt was not rewritten to manufacture review.

## Tests and their limits

| Check | Result | Timing/scope |
|---|---|---|
| Focused local DOCX/font units | **7 passed, 0 failed, 0 skipped** | Fresh local run at `8eb1470a`, Go 1.27.4 Linux/amd64. |
| Local Windows x64 cross-build | **PASS** | Compilation only; different toolchain/artifact from Lenovo. |
| Registered Lenovo `ui-docx` | **7 records passed; 5 images reviewed** | Fresh current continuation, exact native EXE above. |
| Full Windows app baseline | **379 passed, 0 failed, 1 skipped** | Recovered completed receipt at the same `8eb1470a`; not rerun in this continuation. |

The seven local units cover style mapping, heading recognition, body/table/image-description output, body ordering, non-Word Open XML rejection, bounded output and text font sizing. The native full baseline is `dcm-quicksort-baseline-20261113-075058-dd0f60d2`; it exited 0 without timeout and left source unchanged. The skip is `TestRealGeneratedPPTXIntegration`, whose external fixture remains required under DCM-32/47. The skip is not a pass.

The native UI route selected the installed Word handler naturally. The fallback's structured-text renderer has unit coverage, but this run does not certify its headed behavior on a machine without the handler. Native DOCX zoom/pagination, general Word layout fidelity, arbitrary complex documents, protected/damaged inputs and broad failure/cleanup coverage remain outside this gate; DCM-34/35 retain their own work.

## Earlier failure evidence is retained

At `c0c6e4c5`, report `dcm-quicksort-ui-docx-20261113-074320-590bcdc7` failed `DCM30_STALE_PROVIDER_STATUS`. The reviewed image shows the DOCX letter with the incorrect previous TXT footer. The one-line `8eb1470a` change corrects that observed status defect; the fresh images show the correct footer.

Two later earlier-session runs at `8eb1470a` reported seven machine checks passed. One of them, `dcm-quicksort-ui-docx-20261113-075025-5212636c`, contains an incomplete-paint `10-native-letter-reopened.png`: the letter content is readable, but the surrounding shell has blank controls and smaller internal geometry. All five images from that run were inspected here. **That run is not accepted as a complete visual pass.** The earlier `074859-1d0d1f66` receipt was read, but its images were not reviewed in this continuation.

The fresh unchanged-source reproduction has five clean images. This supports the bounded DOCX content acceptance above, but **does not establish the cause or repair of the earlier partial frame**. `DCM-UI-RESIZE-01` is reopened for this observation under the already-pending DCM-46 final UI gate, preserving its previous narrower queued-layout fix history. Release remains blocked on its resolution or explicit evidence-based disposition.

An asynchronous restore/layout sequence is a source-informed hypothesis, not an observed diagnosis. If the incomplete frame recurs, preserve it, record `DcmLayoutProbe.Observe` before/after capture and take a natural adjacent capture without restore or forced redraw. Existing bounded layout traces can distinguish pending geometry/paint from a completed but broken frame. Do not call an increased sleep or a later clean run a product fix.

## Durable evidence and continuation

The private report bundle is under [`flowverse-reports/apps/dcm-quicksort/dcm30-20261113-183530`](https://github.com/laxmanxp/flowverse-reports/tree/main/apps/dcm-quicksort/dcm30-20261113-183530). It contains the five current image files, the retained incomplete-paint and stale-footer images, receipt readbacks, local unit output, native baseline/build receipts and a separate review summary. Source-native files remain on Lenovo; JSON/text copies are exported readbacks, while the PNG bytes were verified against recorded SHA-256 values.

User acceptance remains `not_recorded`; the final package is unqualified and the release is unpublished. The next task is DCM-31 only. Update through the same profile, rebuild after any checkout change, and bind each next UI run to that exact build receipt. Preserve this report's tested source identity when subsequent documentation commits advance the branch.
