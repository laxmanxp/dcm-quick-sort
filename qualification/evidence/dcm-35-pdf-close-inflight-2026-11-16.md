# DCM-35 — bounded PDF close-during-render lifecycle qualification (2026-11-16)

## Decision

**PASS for one bounded DCM-35 PDF lifecycle subcase; parent DCM-35 remains IN PROGRESS.** This is direct evidence of real normal-app close while a WinRT PDF renderer worker is active, its natural completion and obsolete-output cleanup, and source-file lock release. It does not certify the broader preview switch/close and rename/move matrix across all formats, manual user acceptance, or forced worker cancellation.

## Identity, mandatory pipeline and receipts

- Repository: `laxmanxp/dcm-quick-sort`; branch `release/v1.0.71`.
- Tested pushed source: `72ad64be9f2a2b5b3e9a06dff304a12425f110e6`. Candidate clean and tracking origin.
- `flowdev update dcm-quicksort`: PASS, already up to date.
- `flowdev test dcm-quicksort pdf-shutdown-unit`: PASS.
- `flowdev test dcm-quicksort ui-tooling`: PASS.
- `flowdev test dcm-quicksort build`: PASS. Exact build receipt `D:\FlowverseReportsRepo\dcm-quicksort-build-20261116-081353-6976d517\receipt.json`.
- `flowdev test dcm-quicksort ui-pdf-close-inflight`: **PASS**, exit code 0.
- Headed report: `D:\FlowverseReportsRepo\dcm-quicksort-ui-pdf-failure-20261116-081450-bfad6020\receipt.json`, `passed=true`, four machine-accepted cases.
- Detailed process trace: same report directory, `close-inflight.events.json`.
- No production source edited on Lenovo. No new product-code change required for this qualification.

## Observed real process lifecycle

The owned renderer PID **12236** was alive when normal WM_CLOSE was posted to the uniquely identified disposable QuickSort host. No artificial render delay, worker suspension, substituted renderer, synthetic completion, forced kill, or foreign-window interaction was used.

The retained ordered trace for the same worker generation is:
`worker-started` → `selection-stopped` → `worker-finished` → `result-discarded`.

The renderer exited naturally with code **0** before the host's final exit, and the host exited with code **0**. The stale result was not applied. The temporary rendered PNG and remaining owned PDF cache images were absent after close. Both PDF source fixtures were exclusively readable, original fixture bytes were preserved, source checkout unchanged, and the application journal empty under the registered harness.

Machine cases passed: `01-initial-text`, `inflight-valid-PDF-close-drains-renderer`, `08-both-pdfs-exclusively-released`, and `09-normal-close-and-original-bytes`.

## Separate limited screenshot review

Both retained PNG captures (`01-initial-text.png`, `01-initial-text.preview.png`) were individually read and visually reviewed. The disposable app displays the exact expected three-line TXT fixture, without stale PDF content, before the close test. The actual close lifecycle is established by process and trace evidence, **not a screenshot of a closed window**. Original `feature_accepted=false`, `visual_review_required=true`, `user_test=not_recorded` receipt fields remain historical raw outputs and are not rewritten.

## Open scope

DCM-35's full acceptance requires switching/closing the supported preview types and demonstrating file release in real workflows sufficiently to permit rename/move. The current PDF-only close test proves exclusive file-read/open and cache cleanup but does not itself perform rename/move. Broader native Office provider lifecycle cases, multiple rapid selections, and human acceptance remain outstanding. Do not mark DCM-35 completed or increment the 50-step count on this evidence alone.

DCM-34 remains separately PARTIAL: installed Office handler missing/failure environments were not physically qualified, despite passing representative owned-component fallback tests. Final suite packaging continues to depend on Flow Search/DeepSearch readiness.
