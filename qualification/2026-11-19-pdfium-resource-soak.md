# DCM Quick Sort — persistent PDFium resource stability checkpoint

**Date:** 19 November 2026 (Asia/Kolkata)
**Machine:** Laxman_Lenovo
**Branch:** `feature/shared-flowpdf-pdfium-provider`
**Decision:** PASS for the bounded worker and native-GUI tests described below.
**Production:** unchanged; Windows.Data.Pdf remains default.

## A. Standalone persistent worker — 160 renders

Harness: `tools/test-pdfium-persistent-soak.ps1` (test-only, source
commit `507d3f808b4d5f3f001f281a509f31b2df83623d`).

Evidence on Lenovo:
`D:\FlowverseReportsRepo\dcm-pdfium-persistent-soak-20261119-012622-74e7e3\receipt.json`.

A single ordinary File Workspace PDFium worker, launched with
`-ServeJSONL`, rendered 160 requests over its private pipe. Four
synthetic/versioned files received 40 requests each:

- Two-page control PDF
- Synthetic 300-page mixed-layout PDF
- High-resolution raster-like scan PDF (1400 × 1800 grayscale source)
- Vector-heavy PDF containing 7000 rectangles

The script varied page, zoom and rotation; verified each PNG's
decoded dimensions against the wire result; checked the 32 MiB cache
payload bound; removed render outputs; verified original input hashes
and exclusive file access; issued the ordinary `shutdown` command;
and confirmed the exact owned worker exited normally.

**Observed result:** `PASSED=True`, 160/160 requests completed,
one worker PID **28940**, normal shutdown, zero remaining PNG files.

| Measurement | Result |
|---|---:|
| Duration | 24.91 s |
| Post-warm-up first 32-sample median private bytes | 65,142,784 |
| Last 32-sample median private bytes | 63,393,792 |
| Median private-byte change | **−1,748,992 (−1.67 MiB)** |
| Median handle count change | **0** |
| Maximum sampled private bytes | 97,878,016 (**93.34 MiB**) |
| Maximum sampled handles | 191 |
| Maximum reported PDFium PNG cache bytes | 15,312,759 |
| Cache payload limit | 33,554,432 |

The peak values were sampled at 25 ms intervals during requests and
are **not proven instantaneous peaks**. This is a bounded mixed
synthetic workload, not an overnight leak test. Source PDFs did not
change. Worker SHA-256:
`A376B587BC4D682B796CC0ABD7D7FBA158BAD0571E50FE0576A18B0A88E70A03`.

### Test harness corrections retained

The initial invocation used a three-page PDF where the harness
expected two pages and failed at the first request. A later invocation
used fractional PowerShell array indexes, failing after 14 completed
requests. The first 160-render invocation completed actual rendering,
but its summary `Measure-Object` failed on dictionary keys; that run
was marked failed and the worker was torn down by the failure path.
The corrected final run above is the **only accepted** 160-render
soak, with normal shutdown explicitly verified. All failed run
receipts remain on Lenovo; they are not relabeled as passes.

## B. Real native Quick Sort GUI — 64 PDF/TXT switch cycles

Harness: `tools/test-ui-pdfium-provider.ps1` with
`-DevDefault -SkipScreenshots -SkipPointerWheel -StressSwitches 64`.
Harness commit:
`e506fbba46d45c030aee08ffe1fe92b6718e05a5`.

Evidence:
`D:\FlowverseReportsRepo\dcm-pdfium-gui-20261119-012831-a9c62ab\receipt.json`.

The test launched an isolated native GUI with a disposable profile
and selected a PDF, then switched to TXT and back 64 times while
awaiting actual applied PDF renders. Each cycle measured both the
owned GUI process and the persistent worker's private bytes and
handle count. It then executed the ordinary page navigation,
Rename/Undo, Move/Undo, damaged/encrypted PDF refusal/recovery
and shutdown contract.

**Observed result:** `PASSED=True`, `NORMAL_EXIT=True`, exit 0.

| Measurement | Result |
|---|---:|
| PDF↔TXT switching cycles | 64 |
| Total PDF render requests over entire GUI workflow | 81 |
| Distinct persistent worker PIDs | 1 (PID 23408) |
| Unexpected one-shot fallbacks | 0 |
| Quick Sort median private-memory change | +1,679,360 bytes (**+1.60 MiB**) |
| Worker median private-memory change | −5,066,752 bytes (**−4.83 MiB**) |
| Quick Sort median handle change | **+13** |
| Worker median handle change | **0** |
| Source PDF hash, operation journal, undo | PASS |
| Exact owned worker exited after GUI close | PASS |

Quick Sort's observed handles climbed from 224 early in the run
to 243 later, then **remained at 243 across cycles 47–63**. This
is an observed plateau, not proof that a leak cannot occur under
different use. The GUI benchmark intentionally did not perform
screenshot capture or foreground-sensitive Ctrl+wheel input:
another app may be occupying the desktop. Prior native visual/zoom
gates are separate.

The test used the ordinary development application and worker,
not the tagged fault-injection worker. Application SHA-256:
`376D1E40A5720FF54A7E788DC1B87409E841FF081AAC7E029ED5D6D030FDBFF6`.
No user-owned PDFs or system-installed applications were changed.

## C. Two-axis zoom anchoring — simulated DPI geometry

Added `TestPDFCursorAnchorXYAcrossSimulatedDPIScales` and
`TestPDFCursorAnchorConstrainedPanDoesNotLeaveViewport` in
`app-src/pdf_cursor_anchor_test.go`. Tested on Lenovo on commit
`936271fa5bc6c51aa11f5fcf1983234622553b57`.

The pure Go geometry tests passed with **both X and Y drift ≤1 px**
at simulated **96, 120, 144 and 192 DPI** (100%, 125%, 150%, 200%),
using multiple document points, scaled viewport coordinates and
panning. Pan-limit cases also passed. The tests **did not change
Windows display DPI** and do not certify real multi-monitor,
mixed-DPI, physical-pointer mouse handling, or human visual smoothness.

The observed Lenovo bitmap-size discrepancy at 120 DPI was
previously explained by WinRT's logical-DIP metadata versus physical
pixel output; that distinct observation is already closed in
`qualification/evidence/dcm-34-pdf-bitmap-dpi-closeout-2026-11-16.md`.

## Acceptance scope and remaining work

This checkpoint closes the **bounded resource stability** test gate
for a single persistent PDFium worker and one native Quick Sort
instance. It does **not** complete all of DCM-34 or authorize a
release-default renderer switch.

Still pending before production promotion:

1. Real physical 100/150/200% DPI and monitor-move mouse-centered
   zoom acceptance, with independent **two-axis screenshot review**.
2. Longer-run endurance across many distinct real scanned,
   image-dense and graphics-intensive documents, including repeated
   application sessions and system memory pressure.
3. Broader Office preview/provider error behavior under DCM-34.
4. Signed managed worker/runtime packaging, licensing/notice review,
   installer/upgrade/uninstall and rollout/rollback acceptance.

The FileFlow Suite release remains gated on FlowSearch and DeepSearch
readiness. No release branch, production PDF default, installer,
or registered application was changed.


## Final regression checks (19 November 2026)

The two full Quick Sort suites ran on **Laxman_Lenovo** after
the new Go cursor-anchor geometry tests were committed. Tested
source commit `d73fd17cc8e6821383ba481acb1466941606aa61`
(the CI workflow update changes no application code).

- Untagged build, `go test ./... -count=1`: **PASS**,
  `ok quicksort 65.315s`, exit 0.
- Development-default build,
  `go test -tags pdfium_dev ./... -count=1`: **PASS**,
  `ok quicksort 35.436s`, exit 0.
- Windows hosted CI including
  `TestPDFCursorAnchorXYAcrossSimulatedDPIScales` and
  `TestPDFCursorAnchorConstrainedPanDoesNotLeaveViewport`:
  **PASS**,
  [GitHub Actions run 44437694283](https://github.com/laxmanxp/dcm-quick-sort/actions/runs/44437694283).

All added resource tests and DPI geometry tests are confined to the
development branch. No official DCM-34 completion status, user-test
acceptance, release provider setting or distribution approval is
changed by this focused result.
