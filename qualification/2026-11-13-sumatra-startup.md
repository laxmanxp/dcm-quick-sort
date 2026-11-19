# DCM QuickSort — first-PDF startup measured; UI blocking confirmed

13 November 2026 · v1.0.71 · `release/v1.0.71`.
Native-tested source: `497dbb29c66092db690c3ba501501a09f9acb64e`.

## Result

The new `ui-sumatra-startup` diagnostic completed twice, opening the same disposable PDF four times per fresh QuickSort process. Functional document identity, rendered page, release, shutdown and file-integrity checks passed. **Startup responsiveness did not pass the diagnostic's provisional 250 ms maximum posted-message response target.** This target is an engineering diagnostic threshold, not a published standard or a user-agreed release service level.

The first PDF selection in each process delayed a queued UI acknowledgement by approximately **1.76 s and 1.50 s**, respectively. The measured synchronous process-creation interval accounts for **1.70 s and 1.43 s**. Repeated opens in those same processes were much quicker. A blocking startup path is now measured rather than inferred from old harness timeouts.

**No performance correction is claimed.** This change adds opt-in diagnostic instrumentation, a background observation thread in the test process and a registered FlowDev selector. It does not move production launch work off the UI thread, weaken validation, switch the default provider or alter Sumatra.

## Measurements

“Ready” below means the host's first loaded-document status, observed through a numeric property, not a precisely timed first displayed pixel. Subsequent screenshots and content checks independently establish that the expected PDF rendered.

| Run | Open | Host ready status (ms) | Longest queued acknowledgement (ms) | Process creation (ms) |
|---|---|---:|---:|---:|
| A | First in fresh host | 2389.32 | 1757.76 | 1695.00 |
| A | Repeat 1 | 554.90 | 162.77 | 16.58 |
| A | Repeat 2 | 547.71 | 201.68 | 19.05 |
| A | Repeat 3 | 643.19 | 202.05 | 13.74 |
| B | First in fresh host | 1874.46 | 1497.25 | 1432.65 |
| B | Repeat 1 | 512.83 | 153.98 | 13.57 |
| B | Repeat 2 | 555.15 | 153.75 | 17.91 |
| B | Repeat 3 | 560.09 | 202.78 | 15.41 |

Runtime checksum/policy validation took **38.3–48.8 ms**, and attachment-time host layout took **97.1–183.2 ms** across these eight observations. The first synchronous `startSumatraPreview` calls took 1747.21 and 1482.54 ms before returning to the UI loop. Later synchronous starts took approximately 54–67 ms. The application then waited for the separate viewer to attach and report a loaded page.

The longest first-open interval is between the trace immediately before `CreateProcessW` and its return. The process is created suspended, assigned to its own kill-on-close job and resumed as before. **This identifies the blocking call interval, not why Windows spent that time in it.** Antivirus scanning, storage/cache effects and system load were not isolated and are not asserted as causes. This does not establish the cause of every historical native-message timeout.

## How responsiveness was measured

An opt-in host handler acknowledges a private posted message by setting a numeric window property. A dedicated test thread posts one token at a time, checks acknowledgement roughly every 5 ms, and waits 25 ms before the next token. Only the identified QuickSort window/PID is addressed. No process injection, foreign-process memory reads or arbitrary input are involved.

This measures whether QuickSort services its posted-message queue, not just whether a synchronous window-message call can return. Synchronous sends can be serviced in circumstances that do not demonstrate ordinary posted-queue progress. The test retains all samples and calculates the longest acknowledged delay overlapping each selection-to-ready interval. Polling, scheduling and shared workload add overhead; these are not exact physical mouse/key response timings. The 20-second observation deadline is not a responsiveness pass criterion. No prior timeout was increased to hide a product stall.

The Go trace records only sequence numbers, named stages and monotonic elapsed times. It is bounded to 1024 events and written only at normal shutdown into the private test TEMP directory. It contains no passwords or document contents. Ordinary launches leave the probe disabled.

## Cold/warm limits

Each run starts a new QuickSort process and uses a new private runtime copy; each PDF opening creates a new Sumatra process. First opening is compared with three repeated openings in the same host.

**This is not a true cold-boot or flushed-storage-cache test.** The harness reads and copies the runtime before launching, which can warm caches. No reboot, cache flushing, antivirus changes or system-wide settings occurred. An unrelated `unoembed` unit session was observed on Lenovo before measurement. This was not an isolated benchmark, and the small sample does not establish percentiles or a general startup guarantee.

## Functional and safety results

All eight selections displayed the intended teal page of `08-preview.pdf` with the matching selected filename. Each transition back to text restored its exact content, removed the viewer/profile and permitted exclusive PDF access. Four source-file names/hashes remained unchanged in both runs; destination and operation journal remained empty. Private runtime/policy and app hashes were unchanged. Both host instances exited normally with code 0; recorded viewer instances were checked for cleanup.

**Four full-window screenshots from run A were individually inspected.** All show the correct selected file and readable PDF page. Its four content crops were checked by pixel assertions but not separately inspected; run B and regression images were not separately reviewed. Original raw visual flags remain untouched; separate review hashes are in the evidence JSON. Desktop-edge screenshots remain on Lenovo.

## Regressions

Full native Windows baseline: **370 passed, 0 failed, 1 skipped**, 371 parent results, exit 0, no timeout, source unchanged. The skip remains `TestRealGeneratedPPTXIntegration`, not a pass. The **14-check readable-scroll regression passed**. The existing resize-timing regression passed **23 behavior checks plus 12 timing bursts**, with zero differences in all 24 raw/natural sampled comparisons. Counts are distinct selections, not summed.

No Go parent test was added. Startup diagnostic command success means measurements and safety checks completed; both receipts separately record `responsiveness_accepted=false`. Focused local Sumatra tests and Windows cross-compilation passed before Lenovo validation. A local patch uniqueness assertion was corrected before publication; the input helper was named `SelectFile` before execution to avoid PowerShell's `Select` alias. Neither is a product repair. No native startup diagnostic failure was discarded to obtain these two completed measurements.

## Exact evidence

Reports under `D:\FlowverseReportsRepo`:

- Run A: `dcm-quicksort-ui-sumatra-startup-20261113-010300-2885f858`.
- Run B: `dcm-quicksort-ui-sumatra-startup-20261113-010417-3013a7a2`.
- Baseline: `dcm-quicksort-baseline-20261113-010443-73f34848`.
- Scroll: `dcm-quicksort-ui-sumatra-scroll-20261113-010542-3f4b5091`.
- Resize: `dcm-quicksort-ui-sumatra-20261113-010815-aaa93ba1`.
- Build: `dcm-quicksort-build-20261113-010220-8d36bdb9\receipt.json`.

App SHA-256 `255e0d70ba3e052dc765b4a9229c48be87cf1ddcaab6785fd4b8ce830ccb696b`, 7,257,600 bytes. Build receipt SHA-256 `0cd4a1652151bad515685cac2d2d49c4cb172a08c54f7654ae8f11f480eef93a`. 359-file manifest `b91d9775472d040c132e63c0850fa2bc0b380a6279b2dd560f52d4938d977741`.

Runtime/policy hashes remain the pinned values. Measurements, derived stage durations and receipt/image/trace hashes: `qualification/evidence/sumatra-startup-summary.json`. Code/tests authored locally, stored in GitHub and validated through FlowDev on Lenovo; no remote source edits.

## Next bounded correction

Move blocking validation/process creation off the UI thread while preserving executable/policy pins and suspended-process/job ownership. Keep window updates on the UI thread. Use generation/selection tokens so cancelled or stale launches cannot attach to the wrong file; dispose of late results and coordinate pending creation with shutdown. Do not leave an untracked suspended process if cancellation/exit races creation. Reuse this queue-latency diagnostic before and after the correction, followed by switching/cancellation, shutdown, Undo and release-refusal regressions.

The objective is to remove the UI pause even where Windows process-creation duration remains high; do not promise a faster completed PDF without measuring it. `DCM-SUMATRA-STARTUP-01` remains open. No numbered step closed: DCM-28/29 pending, DCM-19 blocked. General labels, error-banner presentation, external PPTX, operation/path variants, distribution/package and human acceptance remain separate gates. Sumatra stays optional, Windows.Data.Pdf stays default, PR #1 stays draft, no merge/publication.

Technical references: Microsoft Learn, About Messages and Message Queues, https://learn.microsoft.com/en-us/windows/win32/winmsg/about-messages-and-message-queues ; Microsoft The Old New Thing, How can I know when a window has processed a message that I posted to it?, https://devblogs.microsoft.com/oldnewthing/20241125-00/?p=110568 . Project-specific timings come from the recorded native measurements, not these references alone.
