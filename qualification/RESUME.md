# DCM QuickSort — async preparation retained; unsafe layout optimization reverted

Latest report: [asynchronous startup](2026-11-13-sumatra-async-startup.md).
Full native UI matrix:4c1c32d668955a2073ac496d0f3d31e5fe1cdf43.
Restored/build-verified source:4e07b99b313027794c05a6bf280208513d869c34.

## State

Document/runtime validation and suspended creation run off the UI thread. One worker/latest queued request; cancelled results disposed; creation-time owned job; UI-only adoption/resume; pending operations refuse and normal Close waits responsively for cleanup. Hash/policy restrictions unchanged. Native baseline372/0/1; startup2x4,pending-close3,lifecycle11,Undo9,errors12,scroll14,resize23+12bursts passed at4c1c32d.

Two later startup runs: first queue45.7/59.1ms, all eight46–105ms; first ready1.62–1.73s. Earlier async repeat delays275/321ms remain evidence: broader250ms target is still open. This is not cold-boot/cache or isolated performance qualification.

Only one representative first-run startup screenshot was directly opened; its other three full-window images are byte-identical, not independently inspected. Regression/repeat images not separately reviewed. Raw review flags unchanged.

## Rollback must remain

Skipping full layout at unchanged sidebar widths caused real visible containment failure at8e510e2e: parent333,259,1238,792; viewer334,260,1463,923; not minimized. Do not infer a complete DPI root cause. At4e07b99b restored exact viewer blob3b7041b3115ecbebd0c0f247f8a6e52d05aad93c from4c1c32d and removed sumatra_transition_windows.go. No startup worker code was reverted.

Rollback build `D:\FlowverseReportsRepo\dcm-quicksort-build-20261113-022343-3455c053\receipt.json` passed and matches tested app byte-for-byte:6f09034154c40fca5737797cd5e341f6817d301b3bfc6e3375fe0bea0930916a,7311360bytes. Full UI matrix was not rerun after rollback; do not relabel old receipts. Detailed evidence: evidence/sumatra-async-startup-summary.json.

## Next bounded work

Keep original full transition layout. Profile remaining attachment/layout response under shared load without skipping required geometry or loosening guards. The worker correction is implemented; broader response/release acceptance is not. Rebuild after metadata/source commits, then use existing startup, pending-close,lifecycle,Undo,error,scroll and resize selectors as needed. No fresh unrelated UI cycle just to obtain more green receipts.

## Workflow

Code locally -> GitHub -> FlowDev update/build/test on Lenovo. Device d18227f2-0b54-4b11-9026-56ac40305f02. Workspace D:\FlowverseReportsRepo; candidate dcm-quicksort-candidate; profile tools\flowdev-dcm; runner D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe. Original runtime D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe. One native DCM UI suite at a time. No remote source edits or unrelated-process changes.

DCM01–18 and20–27 retain completion;19 blocked;28/29 pending. Sumatra optional, Windows.Data.Pdf default. Keep layout/error-banner, externalPPTX, remaining operations/path variants, distribution/package and manual acceptance open. PR#1 draft; no installation/associations/printing/merge/publication.
