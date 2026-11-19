# DCM-34 — Office fallback headed qualification attempt (16 November 2026)

**Result: NOT ACCEPTED / foreground-contended capture gate**, not a demonstrated XLSX product defect.

Source was synced with `flowdev update dcm-quicksort` to `3f1ce140015311c1e2b3f5232ebc6817e1a5945d`. The registered `flowdev test dcm-quicksort xlsx-fallback-component` ended exit 1. The sequential PPTX fallback-component selector was **not started** after that failure.

Retained report and raw receipt:
`D:\FlowverseReportsRepo\dcm-quicksort-regressions-20261116-071554-d9e29222\receipt.json`
and `xlsx-component\receipt.json` beneath that directory.

The test targets owned component UI using simulated *missing handler* and *failed handler* inputs and real fallback parsing/rendering, without changing Office registration. `TestXLSXFallbackComponentRecovery/missing-handler-boundary` **PASS**. The `failed-handler-boundary` subtest failed eight screenshot captures, all with the same ownership guard: **"owned component is not the foreground window"**. No failure of workbook state, data content, or preview logic was reported in the log, but the full gate remains RED and cannot be called accepted.

A separate read-only Win32 foreground inspection after the test found Firefox as the active foreground process (`firefox`, PID 17412, title `Harischandra Sahayata Yojana - Mozilla Firefox`). This supports a plausible shared-desktop focus-interference explanation; it does **not** prove the identity of the foreground process at every failed capture. No foreign window was closed or focus stolen. No Lenovo source files were edited.

**Next:** only on a user-approved idle desktop, rerun the registered XLSX component gate once with uninterrupted focus and then run the PPTX component gate. Keep original failure logs and receipts. Do not replace FlowDev with ad hoc `go test`, and do not infer user acceptance or a production bug from this contention. DCM-34 remains PARTIAL, 32/50 completed.
