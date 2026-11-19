# Sumatra retry — readable dependency errors verified; intermittent resize issue reopened

12 November 2026 · DCM QuickSort v1.0.71 · branch `release/v1.0.71`.
Fresh native-tested source: `715cd61d7da00ee0125eb006246e77711411acc2`.

## Result

**The missing-runtime message is now fully readable in the tested preview pane and status bar. Two fresh error-workflow runs passed all 12 records each, and all ten captures from the first run were inspected.** The full Windows baseline reports **370 passed, 0 failed, 1 skipped**; the lifecycle regression also passes all 11 records.

The navigation/zoom/resize suite initially failed its repeated-resize repaint comparison. Its raw screenshot visibly contains incomplete/stale controls, while the later diagnostic redraw reference is clean. An unchanged retry passes all 23 records, but **DCM-UI-RESIZE-01 is reopened as intermittent**. A later pass does not erase the failed image or establish the cause.

No numbered checklist step is closed. DCM-01–18 and DCM-20–27 retain their recorded acceptance; DCM-19 remains blocked and DCM-28/29 remain pending. Sumatra stays optional and the default Windows.Data.Pdf path is unchanged. No installer, runtime distribution, default association, printing, merge or publication change.

## Recovered implementation, not duplicated work

The interrupted attempt had already committed the concise dependency-error wrapper, one Go parent test, a focused native assertion and capture diagnostics. This retry reviewed those changes and freshly ran FlowDev update, tooling, build and validation; **no additional application or test implementation was changed in this retry**. The report/tracker update is authored locally and saved in GitHub. Lenovo is used for validation, not source editing.

The visible message is:

> SumatraPDF.exe is missing. Restore the approved private viewer, then Reload.

The code also classifies missing policy, denied access and other read failures. The underlying OS error and full diagnostic path remain accessible through `Unwrap`/`errors.Is`/`errors.As`; only display text is shortened. Runtime checksum and exact restriction-policy checks remain unchanged. No PATH search or fallback to an arbitrary installed viewer was added.

The preserved test-only RED at `5c851e84`, `dcm-quicksort-ui-sumatra-errors-20261112-144741-f11a6204`, failed `RUNTIME_ERROR_COPY_NOT_CONCISE`. This is recovered historical evidence, not a new failing run claimed by this retry. The fresh successful runs use the existing corrected `715cd61d` implementation.

## Error workflow and image review

The two fresh runs verify text while the private runtime is unavailable; truthful PDF error with no viewer; recovery when that same private runtime is restored; a header-valid damaged PDF with truthful no-page status; password-prompt survival for 17 seconds; Cancel/text recovery; rejection of a wrong fixture password; rendering with the known correct fixture password; document release and normal shutdown.

All six disposable source files keep their names and original bytes. Source checkout, runtime and policy remain unchanged; destination/journal remain empty. The temporarily renamed test-private runtime is restored. Host exits normally with code 0 and the harness verifies viewer/profile/file-handle cleanup. No user documents or installed runtime are changed.

All **ten first-run captures** were opened: six full host windows, one owned password modal and three content crops. The missing-runtime message and recovery guidance are fully visible; recovered PDF and decrypted fixture content are correct. Raw review flags remain untouched; separate hashes/inspection are recorded in `evidence/sumatra-readable-retry-summary.json`. Second-error-run and fresh lifecycle/navigation-repeat images were not separately inspected.

**DCM-SUMATRA-ERROR-COPY-01 remains partially open:** Sumatra's native damaged-document banner still truncates a long path. The host's no-page status and selected filename are readable. This bounded missing-runtime fix does not certify every error presentation or all display widths. General Size/Sort and Processed-label layout findings remain open.

## Screenshot obstruction: one cause identified, not all old failures explained

The saved diagnostic in `dcm-quicksort-ui-sumatra-lifecycle-20261112-145605-a92e04eb` reports an obstructing window of class `#32768` with PID **7332**, matching that test's QuickSort host PID. It identifies a QuickSort-owned popup window at the sampled capture point, not a foreign PDF-renderer window. Why that popup appeared is not established. This evidence must not be generalized to every earlier occlusion or minimization.

The guard still refuses an obscured/offscreen capture. This retry did not dismiss menus, relax the check or repair historical receipts. Its error/lifecycle captures completed under the unchanged guards.

## Repeated-resize failure retained and reopened

At the first fresh navigation run, `07-repeat-restored.png` shows displaced/partially erased host controls after repeated shrink/restore. The explicit-redraw reference restores the normal screen. The comparison reports **32,603 changed samples out of 107,276**, and the background check reports **1,746 bad samples**. `07-restored.png` from the preceding checkpoint is clean. All three images and the layout record were inspected.

The raw screenshot precedes the diagnostic redraw, so the failure is not concealed by a test-assisted repaint. Twenty-one earlier behavior records passed, including page/zoom and representative Rename/Move, but the suite's final result is **failed**, even though the app itself exited normally and fixtures were preserved.

The same source and executable then passed an unchanged 23-record navigation retry, with zero sampled paint differences at its comparison checkpoints. This establishes intermittency, not a fix. The exact cause remains unestablished: instrument layout/message completion and paint ordering rather than assume the dependency-error change caused it or increase sleeps until it passes. Preserve the earlier `24ff2f24` successful resize evidence under its original identity, but do not continue treating the issue as universally resolved.

## Fresh results — same source and matching build

Reports are under `D:\FlowverseReportsRepo`.

| Selection | Result | Report directory |
|---|---|---|
| Full Windows baseline | **370 passed / 0 failed / 1 skipped**; 371 parents, exit 0, no timeout, source unchanged | `dcm-quicksort-baseline-20261112-152048-fc05ebbe` |
| Reviewed errors run | **12 passed**, host exit 0; all ten captures reviewed | `dcm-quicksort-ui-sumatra-errors-20261112-151918-97d90d78` |
| Errors unchanged repeat | **12 passed**, host exit 0; images not separately reviewed | `dcm-quicksort-ui-sumatra-errors-20261112-152522-13a95983` |
| Lifecycle regression | **11 passed**, host exit 0; images not separately reviewed | `dcm-quicksort-ui-sumatra-lifecycle-20261112-152440-8d22ee27` |
| Navigation/zoom/resize first run | **Failed** repeated-resize repaint/background assertions after 21 behavior records; app exit 0 | `dcm-quicksort-ui-sumatra-20261112-152144-1254fe66` |
| Navigation unchanged retry | **23 passed**, host exit 0; images not separately reviewed | `dcm-quicksort-ui-sumatra-20261112-152939-74d333d0` |

The sole skip remains `TestRealGeneratedPPTXIntegration`, not a pass. One recovered new dependency-message parent test explains the increase from 369 to 370 passing parents; its cases are not counted again. UI records and repeat executions are separate selections. No fresh installer/package, full Batch Undo, default-provider PDF UI, text/image or general physical-input qualification is claimed.

## Exact identities and continuation

Build receipt: `D:\FlowverseReportsRepo\dcm-quicksort-build-20261112-151848-8d41b96a\receipt.json`.
Receipt SHA-256: `29aa8731636035b9824da6bc195564f43e53f655fc7842081e51f27ce900294f`.
App SHA-256: `e20c2b160f3b315d2a58b9d1bbf3effd9b105b89e03cc44b8fe92d8062b2ecab`; 7,242,752 bytes.
331-file source-manifest SHA-256: `b1ae57b1e4b52ed0f3edca91062ad2591dd396858fc490a2de65871267163eb1`.
Runtime/policy hashes remain the pinned values recorded in the evidence JSON.

Next: diagnose the intermittent repeated-resize raw-capture failure with layout/paint timing evidence, then address the remaining native damaged-file banner and startup/scroll-to-readable-text checks. Do not rerun already-passing error checks as a substitute for that diagnosis. Keep DCM-19, external PPTX, general layout, remaining operation/path cases, runtime distribution/installer and human acceptance open. Rebuild after later source/metadata commits; never rewrite old receipt identities. PR #1 remains draft.
