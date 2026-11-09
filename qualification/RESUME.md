# DCM v1.0.71 — resume after DCM-13

Latest checkpoint: [DCM-13 complete](dcm13-complete.md). Preserve the earlier partial report and every prior source/test identity.

## Verified state

DCM-01–13 are complete at their defined boundaries, including five setup/diagnostic steps. Only DCM-13 was newly closed. `DCM-INSTALL-CLEANUP-01` is fixed; final packaged-executable qualification remains open.

Product/installer cleanup source: `70e8f240a0228c8548562bb627db4213cbaf1d17`. Its full installer unit suite passed 26 tests, with no failures or skips. The successful cleanup case and three locked-target cases were checked; success is not an unconditional exit.

Latest native-tested source: `a49e65951b5be9d26dc7d74a4916ae0b1b54664d`. Only the engine evidence reader changed from the unit-tested source. Fresh `installer-integration` passed both its single native lifecycle test and outer FlowDev validation: exit 0, no timeout, source unchanged. The reader preserves empty registry-value names rather than changing the raw evidence. All cleanup, registry and payload checks remain intact.

The engine proves matching configuration through upgrade, restore and retained backup, removal of its installation/script/shadow registrations, reset of HKCU redirection, private-key removal and unchanged live DCM registry keys. It is not packaged setup/uninstaller E2E or visible UI acceptance.

## Next exact action

**DCM-14:** update through FlowDev, then run `baseline` to execute the full native app suite. Investigate any failures and explicitly disposition skips before closing the step. Do not infer full-app acceptance from the earlier 23/23 safety selection. Then continue DCM-15 (`installer-unit` and `ui-tooling`), reusing evidence only when its relevant source and assertions are unchanged.

## Native workflow and evidence

Device: `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`.
Candidate: `C:\FlowverseReports\dcm-quicksort-candidate`, branch `release/v1.0.71`.
Set `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports`, run from `C:\FlowverseReports\dcm-quicksort-candidate\tools\flowdev-dcm`, and use `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.
Run `flowdev update dcm-quicksort`, then one `flowdev test dcm-quicksort <selector>` at a time. Code locally, store in GitHub, validate on Lenovo. Never edit the device checkout or substitute a host installation.

Unit evidence: `C:\FlowverseReports\dcm-quicksort-installer-unit-20261109-180724-83ef3b94` at `70e8f240`.
Fresh engine evidence: `C:\FlowverseReports\dcm-quicksort-installer-integration-20261109-181814-920da14f` at `a49e6595`.
Compact evidence: `evidence/dcm-13-complete.json`. Do not publish raw host registry snapshots.

Both earlier move/recovery proposals and the installer cleanup correction are already applied. Do not reapply them. Read `../tools/README.md` and `../tests/native/README.md` before UI/package gates. Bind exact `DCM_BUILD_RECEIPT` and `DCM_PACKAGE_RECEIPT` values; reviewed images and user testing remain separate from automated assertions. Keep PR #1 draft. No merge or publication before remaining gates and explicit approval.
