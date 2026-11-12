# QuickSort — PDF error fixes verified once; UI reruns remain incomplete

Latest: [damaged/password/missing-runtime PDF handling](2026-11-12-sumatra-errors.md).
Product/native-tested source `39af1659c01cbfa6bdb38106370197001027e946`.
No numbered closure: DCM-01–18 and DCM-20–27 complete; DCM-19 blocked; DCM-28/29 pending. Keep Sumatra optional and Windows.Data.Pdf default; PR #1 draft, no merge/publication.

## Actual verified state

New `ui-sumatra-errors` completed 12 records once. All ten captures inspected: six host windows, one password dialog, three content crops. Missing private runtime does not prevent text; missing-runtime error and restoration recover correctly. A header-valid damaged file reaches the real decoder and shows a load error. Its host status no longer falsely advertises navigation/zoom. A known encrypted fixture's password prompt survives a17-second hold beyond the former15-second deadline; Cancel/text recovery, wrong-password rejection and correct-password rendering/release pass. All six source names/bytes retained; empty destination/journal; normal close0; private runtime/policy restored.

Full Windows baseline369 pass /0 fail /1 external-PPTX skip,370 parents. One new document-status parent with nine table entries. Local focused Sumatra tests and Windows test cross-compilation passed. Product checks only owned pinned viewer controls, with bounded text reads; password text is never read by the host. No runtime binary or restriction change.

**Later unchanged UI runs did not pass**: navigation, its retry, lifecycle and error repeat all stopped at the occluded/offscreen capture guard. Do not silently treat the first passing error run as repeat/regression acceptance. Exact obstruction cause unknown; guards retained. Do not repeatedly launch suites without restoring a usable test display or understanding the obstruction, and do not touch unrelated user windows/processes.

## Next exact work

Reestablish a visible owned test window and finish errors/navigation/lifecycle reruns. Then correct `DCM-SUMATRA-ERROR-COPY-01`: missing-runtime technical path clips; preserve truthful detail but provide concise readable guidance. Existing native damaged-file banner truncates long paths too. Preserve the decoder-error and password-timeout RED evidence; do not claim all layouts approved. Continue startup responsiveness `DCM-SUMATRA-STARTUP-01` and scroll-to-readable-text at100%/FitWidth; decide remaining bounded DCM-28/29 acceptance explicitly rather than endlessly rerunning ordinary successes.

## Workflow and identities

Code locally -> GitHub -> FlowDev update/build/test on Lenovo, serially. Workspace `D:\FlowverseReportsRepo`; checkout `D:\FlowverseReportsRepo\dcm-quicksort-candidate`; profile `tools\flowdev-dcm`; runner `D:\FlowverseReportsRepo\flowdev-standalone-validation\flowdev-test.exe`. Device `d18227f2-0b54-4b11-9026-56ac40305f02`. No remote source edits or unrelated LibreOffice changes.

All Sumatra selectors need `DCM_BUILD_RECEIPT` and `DCM_SUMATRA_SOURCE_EXE=D:\Flowverse\sumatra-preview-prototype\runtime\sumatra-3.6.1\SumatraPDF-3.6.1-64.exe`. Error test copies that runtime into its lab before temporarily renaming its private copy. Never rename the original runtime. Only the generated fixture's known password is used; never collect user passwords.

Build `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-140740-e5a0bb21\receipt.json` matches39af1659 only. App SHA756beb686fb196c608e415f3b366cad93ec09af6cf32fa631a8f56a0750f0ef8. Rebuild after metadata/source updates, do not rewrite receipts.

Reports under D:\FlowverseReportsRepo:
Reviewed errors `dcm-quicksort-ui-sumatra-errors-20261112-140757-e0408a55`.
Baseline `dcm-quicksort-baseline-20261112-141022-88e2d46f`.
Failed navigation `dcm-quicksort-ui-sumatra-20261112-141100-f448d0d0` and `141436-4aaab0a1`.
Failed lifecycle `dcm-quicksort-ui-sumatra-lifecycle-20261112-141445-386c1e93`.
Failed errors repeat `dcm-quicksort-ui-sumatra-errors-20261112-141456-2c5460b9`.
Pre-fix reports `dcm-quicksort-ui-sumatra-errors-20261112-135630-03135ea5` and `140251-caf2bc69`.
Review/hashes: `evidence/sumatra-errors-native-summary.json`.

Preserve all earlier Undo/release/lifecycle/layout evidence at its original source identity. Keep external DCM_REAL_PPTX, DCM-19, general labels, remaining Batch Undo/path/release cases, runtime distribution/installer and manual acceptance open. No default switch, printing, associations or installed-working-copy replacement.
