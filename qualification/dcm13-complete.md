# DCM-13 — installer preference and isolated lifecycle qualification complete

Date: 9 November 2026 (UTC evidence date). Branch: `release/v1.0.71`. Candidate remains unpublished.

## Result and scope

**DCM-13 is complete at its native installer-engine boundary.** `DCM-INSTALL-CLEANUP-01` is resolved. Only DCM-13 is newly closed; DCM-01–13 are recorded complete, including five setup/diagnostic steps. This is not a product-completion percentage or final release certification.

| Validation | Exact source | Result |
| --- | --- | --- |
| Complete installer unit suite, retained run | `70e8f240a0228c8548562bb627db4213cbaf1d17` | 26 passed, 0 failed, 0 skipped; exit 0, no timeout. |
| Fresh isolated installer engine and outer FlowDev validation | `a49e65951b5be9d26dc7d74a4916ae0b1b54664d` | 1 passed, 0 failed, 0 skipped; exit 0, no timeout; source unchanged. |

These are separate selections, not a combined full-app total. The source comparison from `70e8f240` to `a49e6595` changes only `tools/run-native.ps1` (seven added lines, one removed); the installer implementation and unit assertions are identical. The 26-unit result was reviewed and reused, not rerun in this final parser-fix pass.

## Reproduced causes and corrections

The original cleanup script was executed from its actual Go source expression against disposable fixture paths. Its delay failed without console input, and its self-deletion made CMD report that the batch file could not be found. Removing only the delay still reproduced exit 1 after successful removal. Removing self-deletion avoided that missing-batch error but was diagnostic only, not an accepted fix.

The installer correction replaces the console-dependent delay with a noninteractive two-second sleep, checks that the uninstaller and installation directory are absent, then deletes the script and terminates its dedicated CMD process on the same command line. Success is conditional on verified removal. An initial `exit /b` attempt still failed; the final process-exit correction is in `70e8f240`.

The focused assertions check all four outcomes:

| Fixture | Expected and observed exit | Filesystem result |
| --- | --- | --- |
| Successful cleanup | 0, no error output | Installation directory and cleanup script absent. |
| Locked uninstaller | 1 | Locked bytes preserved; directory and script retained. |
| Locked file inside installation directory | 1 | Locked bytes preserved; incomplete removal reported. |
| Locked cleanup script | 2 | Installation removed, script retained; failure not hidden. |

The outside sentinel remains unchanged. These cases are two parent tests with three nested locked-file cases, already included in the 26-unit total.

After the product fix, the native engine passed, but the outer runner failed while parsing valid JSON containing unnamed registry values. `a49e6595` changes only that evidence reader to dictionary-based deserialization. The raw JSON is not rewritten or filtered, and existing checks for engine success, private-registry cleanup, unchanged live keys and exact installed payload remain intact. The original runner blob and locally edited replacement were hash-verified before native validation.

## Fresh lifecycle evidence

Used the registered sequence `flowdev update dcm-quicksort` then `flowdev test dcm-quicksort installer-integration` on Lenovo. Both succeeded at `a49e6595`.

The engine exercised fresh installation, upgrade, restore and uninstall in disposable paths with process-local private HKCU redirection. Upgrade, restore and the retained backup have identical hashes for all five configuration files: settings, destinations, sources, learning and naming templates. The source-folder choice, PDF view and Vault preferences remain covered.

The receipt confirms `registry_override_reset=true`, `private_registry_removed=true`, `live_registry_unchanged=true`, removal of the owned install directory and cleanup script, and absence of all five shadow registration trees. Cleanup child PID 5068 was observed and the zero-exit assertion passed. No host installation was performed.

Unit report: `C:\FlowverseReports\dcm-quicksort-installer-unit-20261109-180724-83ef3b94`.
Fresh engine report: `C:\FlowverseReports\dcm-quicksort-installer-integration-20261109-181814-920da14f`.
Engine finished under the outer receipt at `2026-11-09T18:18:43.3172651Z`.
Source manifest: 206 files, SHA-256 `c72711a390f97e149c4044e4545d3dcbac2a77762aecd57de56f130b3931eaa7`.
Engine receipt SHA-256: `13323bf25b40fbc62ebc483f799d6d6326d986d61ef5a66b5a5f95be13bd5584`.

Compact, non-sensitive evidence: `evidence/dcm-13-complete.json`. Raw registry snapshots remain on Lenovo, not in this report. Earlier failed reports and `2026-11-09-dcm13.md` remain historical evidence.

## Remaining requirements

This is an in-process engine test. It uses the test executable for the uninstall self-copy; it does not launch the installed app or certify the packaged setup/uninstaller, live Explorer behavior, screenshots or actual user acceptance. Final package verification and isolated packaged-executable lifecycle remain DCM-47/48 requirements, even though the cleanup blocker is removed.

**Next: DCM-14 — run the complete native app suite using `baseline`.** DCM-15, UI workflows, final artifacts and publication are not closed here. Keep PR #1 draft; do not merge or publish without the remaining gates and explicit approval.
