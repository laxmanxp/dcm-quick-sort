# DCM QuickSort 1.0.69 — Lenovo FlowUIVision feature test

Date: 2026-11-03. Device: Laxman_Lenovo.

## Scope and identity

Tested a byte-identical copy of the installed DCM QuickSort executable, not FileFlow Walk and not the older 1.0.62 recovery build. FlowUIVision identified the running window as version 1.0.69.

Executable SHA-256: `BF93CE364909F8541D015D42CAFF3A2B0970574197CD60D0A0E756DF97E9683B`.

The existing FlowUIVision source at `C:\Flowverse\apps\flowuivision` was tested and built into the isolated lab. The checkout reported HEAD `54395443`. Its desktopdriver, script, and vision package tests passed; the command package had no test files. The Windows agent health check returned ready.

Test root: `C:\FlowverseReports\dcm-flowuivision-20261103`.

The copy used a separate LOCALAPPDATA profile, eight disposable inbox files, and an empty test vault. No production application code was changed. Test scripts are stored on `qa/flowuivision-v1069`; main is untouched.

## Confirmed results

| Feature | Result | Evidence |
|---|---|---|
| Launch/version/control discovery | PASS | Actual 1.0.69 window inspected through FlowUIVision |
| Name filter and no-match state | PASS | Known name returns one; absent name returns zero |
| File type, extension, size, age filters | PASS | Images=1, TXT=3, 1–10 MiB=0, older-than-30-days returns the one aged fixture |
| Clear Filters, reload, sort selection, Next | PASS | Filter/navigation script completed all 53 steps |
| TXT preview | PASS, visually reviewed | Exact ALPHA fixture text visible |
| PNG preview | PASS, visually reviewed | Orange IMAGE PROBE image visible |
| DOCX built-in fallback | PASS, visually reviewed | Heading/body/currency/signature text present; not Word page-layout certification |
| XLSX built-in fallback | PASS, visually reviewed | Orders grid; switching to Read me displays different worksheet content |
| PPTX built-in fallback | PASS, visually reviewed | Slide 1, Slide 2, and return to Slide 1 show the expected distinct content |
| Initial PDF and Fit Width | PASS, visually reviewed | Print test content and page 1/1 visible |
| Clean spacing naming tool | PASS | Repeated spaces become single spaces in the proposed name |
| Rename and Undo Rename | PASS, filesystem verified | Original filename restored; SHA-256 unchanged |

The 53 steps are script actions/assertions, not 53 independent features. These results apply to the tested fixtures and configuration, not all possible files.

## Remaining failures and limitations

1. **Manual PDF zoom repaint is NOT certified.** Status checks for 100%, 110%, and Fit Width passed after waiting for readiness. However, full-window captures after 100%/110% showed a blank document area, while Fit Width visibly restored Print test. Additional screen cross-checks were invalid because the test window was minimized by capture time. This could be a rendering/repaint or capture/interaction issue; root cause is not established. Do not use the passing footer assertions as a visual PASS.
2. **Move to Vault did not complete in the automated attempts.** The destination edit displayed the isolated vault path, but the source stayed in the inbox and the vault stayed empty. One run failed the Undo-enabled assertion; another failed the independent filesystem oracle. Destination configuration and native interaction/focus require investigation. This is not sufficient evidence to declare all normal user moves broken. Undo Move was not tested because no move was confirmed.
3. **Multi-file selection and batch move are not certified.** The current FlowUIVision select adapter does not support DCM's native ListBox; it returned `control does not support select`. This is a tooling gap, not proof that QuickSort lacks multi-selection.
4. The installed FlowUIVision parser rejects empty quoted text tokens. The harness used Clear Filters instead of an empty type action. Some initial test assumptions about selection and copied-file dates were corrected; those were harness issues, not application defects.
5. Initial desktop screenshots and several minimized-window captures were rejected. Visual conclusions use readable PID-verified QuickSort window captures. A minimized/blank image is never treated as proof of rendered content.
6. Legacy DOC/XLS/PPT, complex Office fidelity, password-protected or damaged files, crash recovery, overwrite collisions, and physical printing were not certified in this pass.

PPTX is a useful lightweight fallback: expected slide text and approximate placement rendered, but the original blue shape styling was not reproduced. DOCX and XLSX similarly used DCM's structured fallback presentation during this run. These results do not close the separate FileFlow Walk COM-host bug.

## Evidence locations on Lenovo

All paths below are relative to the test root.

- `filters-verified.json`: final 53-step filter/navigation PASS.
- `features-20261103-173437`: six format preview captures; clean-spacing, rename, and Undo Rename results with filesystem/hash checks.
- `features-20261103-173915`: initial navigation checks, PDF status results, multi-selection tool-coverage failure, and unsuccessful move attempt.
- `features-20261103-174255`: restored-window slide 2, return to slide 1, second worksheet captures, and PDF toolbar readiness checks. Review pixels separately from JSON booleans.
- `features-20261103-174752`: final PDF cross-check attempt and move filesystem failure. The screen captures here are invalid/minimized, not visual evidence of success.
- `final-safety.json`: final identity, profile, fixture, and shutdown checks.

Each feature directory contains generated .fvs scripts, raw FlowUIVision JSON, stderr, screenshots, and summary JSON. A summary entry with visual=`not reviewed` is an automation placeholder; the explicit visual conclusions above record subsequent manual image inspection. Runner exit code alone is not the result; per-case JSON and visual evidence take precedence.

## Final safety checks

The copied test application closed gracefully. All eight fixtures remained in the inbox, the renamed temporary name was absent, and the test vault contained zero files. The five copied image/Office/PDF fixtures still matched their original source bytes. The installed QuickSort executable still matched the recorded SHA-256.

The regular user's five profile files — destinations.conf, fileflow-drafts.json, learning.json, settings.json, and sources.conf — all matched their pre-test hashes. No production profile file was overwritten. Reports and disposable test files remain available for inspection.

## Next testing gates

Improve FlowUIVision ListBox multi-selection and minimized-window detection; configure a genuine isolated Vault destination through the supported UI and certify move/undo; add a pixel-level PDF zoom regression. Preserve the working DCM Office fallbacks as the reuse baseline.
