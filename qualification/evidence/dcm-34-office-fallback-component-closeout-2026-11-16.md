# DCM-34 — XLSX and PPTX fallback component closeout (2026-11-16)

## Decision

**PASS — bounded Windows owned-component failure/recovery contract for XLSX and PPTX.** Both registered FlowDev gates pass and all **32 retained PNG screenshots** were individually visually reviewed without a visible mismatch to the fixture's intended error/recovery states. This does **not** mean real installed Excel/PowerPoint providers were made to fail; the handler-present/failed inputs are simulated at an owned component boundary and registry/Office activation were not touched.

Source: `laxmanxp/dcm-quick-sort`, `release/v1.0.71`, tested and pushed `e82f295adc53a64fe977b04361c1c6f15564a3e1`. Lenovo checkout was clean. Registered `flowdev update dcm-quicksort` returned already up to date. No Lenovo production edits and no source correction.

## Registered acceptance and immutable raw reports

- `flowdev test dcm-quicksort xlsx-fallback-component`: **PASS**, 16 records, 16 captures, 0 capture errors.
  - `D:\FlowverseReportsRepo\dcm-quicksort-regressions-20261116-075402-14ba83bb\xlsx-component\receipt.json`
- `flowdev test dcm-quicksort pptx-fallback-component`: **PASS**, 16 records, 16 captures, 0 capture errors.
  - `D:\FlowverseReportsRepo\dcm-quicksort-regressions-20261116-075503-0f174057\pptx-component\receipt.json`

For both: the missing-handler-boundary and failed-handler-boundary paths exercised an invalid document → valid recovery → invalid document after success → valid second recovery sequence. Original receipts retain `feature_accepted=false`, `visual_review_required=true`, and `user_test=not_recorded`. This review is a separate bounded conclusion, not an edit to machine-generated history.

The **earlier failed XLSX run** at `dcm-quicksort-regressions-20261116-071554-d9e29222` (eight strict foreground capture failures during the second subtest) remains RED historical evidence. The new successful run supersedes that specific capture blocker without deleting it.

## Visual review of 32 exact source captures

Each of the 16 XLSX and 16 PPTX PNGs was independently read from Lenovo via Desktop Commander and inspected, including both handler-boundary scenarios and both recovery iterations.

**XLSX:** Missing-handler error visibly says no Windows XLSX preview handler installed; failed-handler error visibly says installed handler could not preview the file. Both show `corrupt.xlsx`, keep rename/move/open available, and show no stale workbook. Valid first and second recoveries show `Summary` sheet with `Bridge Work 125000 TRUE`, `Coast Road 86000`; `Data & Bills` shows `Stage Approved`, `Road 86000`; sheet 1 returns intact after sheet 2. Error-after-success states return to their correct explicit unavailable body.

**PPTX:** Missing-handler and failed-handler errors similarly distinguish no Windows handler from failing handler and show `corrupt.pptx` rather than an empty successful deck. Valid first and second recoveries show `Bridge Inspection` slide with title, September 2026, bridge image and description, then `Payments` slide with table `Bridge 125000` / `Road 86000`, and correct slide 1 return. Error-after-success states remove prior slide content. The owned preview surface clips at its right test-window edge; the meaningful fixture text and expected artwork are visible. This does not certify arbitrary display sizes or full-deck fidelity.

In both receipts no file mutations or unreleased fixture locks were reported. No Windows registry mutation and no Excel/PowerPoint application activation occurred.

## Scope and next steps

This accepts **DCM-34-XLSX-FALLBACK-01** and **DCM-34-PPTX-FALLBACK-01** only for the specified owned-component boundary contract. It is not a full real-provider failure/absence simulation and not physical end-user acceptance. It does not reopen normal native XLSX or PPTX preview qualification.

**The parent DCM-34 remains PARTIAL and the 50-step tracker stays 32 complete / 18 remaining.** Before closure, assess remaining unsupported/protected/legacy Office and lifecycle/failure contracts separately, then close only when the explicit DCM-34 acceptance criterion is supported. Preserve the separate PDF DPI and interruption closeouts and all raw reports.
