# QuickSort — queued resize/paint work corrected; remaining PDF qualification

Latest: [resize timing diagnosis and correction](2026-11-12-sumatra-resize-timing.md).
Tested source/product `02e6efb39f08c5cfa29da4787cd4f41bf71b642a`.
No numbered closure: DCM-01–18 and DCM-20–27 complete; DCM-19 blocked; DCM-28/29 pending. Sumatra stays optional and Windows.Data.Pdf default. No printing/merge/publication.

## Current verified result

The fixed-wait diagnostic reproduced an unfinished queued resize at c15bf4a6: a partial smaller raw screenshot, then a clean unassisted restored screenshot identical to its later diagnostic redraw reference. The final six layout calls took214–361ms each. No nesting above one was observed. This narrows the reproduced failure to queued layout completion, not a permanent paint defect requiring forced redraw. Do not assume every older occlusion/minimization shares that cause.

Product now moves49 controls without immediate per-control repaint, then queues invalidation/erase/all-children/frame repaint once at completed geometry. Same diagnostic harness SHAfe3a07f02f8a14caf1fe12ff148b9f7056727db9d794475c215c8a83a8299a07 passes twice:23 existing behavior records plus12 bursts, six size requests/burst, all24 raw/natural comparisons zero per run. Marker is AFTER raw/natural captures; timing/guards unchanged betweenRED/GREEN. First corrected trace91 layout calls, mean86.4ms/max174.5ms versus RED63calls mean141.7/max361.1ms: observations, not a benchmark.

18 first-GREEN images inspected (all12 burst raw, initial/narrow/ordinary repeat/final text and last natural/reference), plus3 decisiveRED images. Not all79 captures, repeat or regression images reviewed. Raw flags unchanged. DCM-UI-RESIZE-01 fixed only for the reproduced path/tested matrix; general labels and historical uncertainty remain.

Fresh baseline370 pass/0fail/1 external-PPTX skip. Ordinary ui-sumatra23 with trace disabled; launch12/text17/image23 pass. No new Go parent tests. Source/fixture safety and normal exit checked. Earlier startup deadline attempt preserved; diagnostic selection only reuses existing async helper, not a startup performance fix.

## Next exact scope

Qualify readable document content after scrolling at100% and FitWidth. Retain DCM-SUMATRA-STARTUP-01 and remaining native damaged-path banner/host-label layout work. Do not redo fixed error/Undo workflows instead of missing scroll/startup qualification. No default switch or DCM-28/29 closure until remaining provider acceptance is resolved.

## Workflow and diagnostic use

Code locally -> GitHub -> FlowDev update/build/test on Lenovo. Workspace D:\FlowverseReportsRepo; checkout dcm-quicksort-candidate; profile tools\flowdev-dcm; runner D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe. Device d18227f2-0b54-4b11-9026-56ac40305f02. One DCM UI suite at a time; leave unrelated work untouched. No remote source edits.

New selector ui-sumatra-resize-timing invokes the ordinary Sumatra harness with -ResizeTiming. Trace uses DCM_LAYOUT_TRACE=1 only in that child, writes numeric in-memory ring to private TEMP/dcm-layout-trace.json at shutdown. Ordinary launches do not enable it. Trace ring8192; inspect total before assuming all events retained. New tests/native/dcm-layout-probe.cs only observes owned control geometry/update regions and posts a no-paint marker.

DCM_SUMATRA_SOURCE_EXE remains D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe. Runtime/policy pin unchanged. Exact build D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-161742-8ecbe7d7\receipt.json matches02e6efb3 only. App SHAa02164f61a1334389649179f246a5b17a56a69771f334404b75962a212a2b1bc. Refresh build after metadata/source changes; never relabel receipts.

Reports under workspace: REDui-sumatra-20261112-161007-7ff511c9; GREEN161805-7fe5f40e; repeat162103-83d13e18; ordinary162232-be320fb8; baseline161936-784532bc. All use dcm-quicksort- prefix. Full names/hashes/review in evidence/sumatra-resize-timing-summary.json. Original715cd61d32,603-pixel failure and all old source identities remain preserved.

Keep DCM-19, DCM_REAL_PPTX, distribution/installer, operation/path variants, general labels and human acceptance open. No installed-working-copy or association changes. PR#1 draft.
