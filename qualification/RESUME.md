# DCM v1.0.71 — resume native qualification

This checkpoint is blocked by the native host, not complete. The source review is [draft PR #1](https://github.com/laxmanxp/dcm-quick-sort/pull/1). Preserve the user's test-first sequence: native RED, smallest fix, native GREEN; then full units and actual visible workflow. User acceptance remains unrecorded.

## Host and source

The retained [diagnostic excerpt](native-host-2026-11-09.json) records 650,547,200 bytes free on C: and 478,658,560 bytes available physical memory at 05:03 UTC. The connection briefly dropped and file reads were very slow. Restore resource headroom and a reliable Desktop Commander connection before resuming. Do not delete user files or close unrelated applications automatically.

Native device: `Laxman_Lenovo`, `d18227f2-0b54-4b11-9026-56ac40305f02`.

Candidate: `C:\FlowverseReports\dcm-quicksort-candidate`, branch `release/v1.0.71`, origin `https://github.com/laxmanxp/dcm-quick-sort.git`.

FlowDev executable: `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`.

Set `FLOWDEV_WORKSPACE_ROOT=C:\FlowverseReports` and run from the candidate's `tools\flowdev-dcm` directory. Use `flowdev update dcm-quicksort` and registered `flowdev test dcm-quicksort <selector>` routes. Do not use the unrelated older monorepo FlowDev executable or dirty v1.0.62 checkout.

The final confirmed native update was `59b9d9cd4611ae71edd139913fdae51a5d93286d`. Later context commits may exist on the release branch; always record the exact updated HEAD. No qualified v1.0.71 executable or setup has been produced.

## Resume in order

1. Retrieve the interrupted regression receipt and logs from `C:\FlowverseReports\dcm-quicksort-regressions-20261109-044942-5df3654c`. The app-regressions child hit its 300-second deadline on `a88d4f28ab801113487248d833c898c344138f0c`; its partial tests remain unknown. Do not assume a timeout is the required assertion RED.
2. Update the candidate and run `host-diagnostics` if resource recovery is uncertain. The previous diagnostic command completed successfully; no restart of that old command is needed.
3. Run `regressions` and obtain actual assertion results for `TestD2MoveFileRefusesExistingForeignDestination` and `TestD6MovedDuplicateSurvivorRequiresAttentionAfterRestart`. D1/D2/D6's earlier defects already have native RED evidence, and their main fixes are committed.
4. After the two additional assertion REDs, apply the reviewed proposals in `qualification/proposed-fixes/no-replace.patch` and `qualification/proposed-fixes/duplicate-kind-gate.patch` to local product source. They are deliberately **not applied** at this checkpoint. Recheck patch context, commit small changes, update through FlowDev, and run regressions again. Do not weaken the original-byte or existing-destination expectations.
5. Run the final full app suite (`baseline`), current `ui-tooling`, and `package`. Package includes app/installer units, private-registry installer engine lifecycle, fresh app/setup builds and exact embedded-payload verification. Preserve receipts and failures; fix any real problem before proceeding.
6. Bind `DCM_BUILD_RECEIPT` to the successful build/package receipt and run `ui-filters`, `ui-rename`, `ui-move`, `ui-batch`, `ui-pdf`, `ui-previews`. Bind `DCM_PACKAGE_RECEIPT` for `installer-ui-cancel`. Review the actual retained images; `visual_reviewed=false` is not human visual acceptance. Keep private app/backup/registry isolation intact. Installer UI checks open and cancel the actual setup; the installer engine test separately exercises install/upgrade/restore/uninstall in a private registry scope.
7. Update the checkpoint, todo, release notes and both shared project contexts with exact source identities, test totals, artifact hashes, evidence and material preview limitations. Keep feature implementation, UI and user-test statuses distinct.
8. Run `release-prepare` with the exact package receipt and final notes. `gh.exe` availability/authentication are unverified. If supported, `release-draft` requires `DCM_EXPECTED_COMMIT` matching the package. It creates only a release draft with the exact two assets; it does not publish. Do not merge or publish while any native release gate is open.

## Outstanding observations

The read-only host diagnostic and its separate receipt-read process completed. The two asynchronous reads for the old regression summary/stdout were still unresolved at the checkpoint; check their status before repeating those exact calls:

- Summary: `cb3e2cc9-876d-483d-a2d9-e8cdc2f3da8f`.
- Stdout: `1760efa6-e7c9-4060-bec6-7a5d388f44be`.

The stderr read failed with `Result could not be stored (TypeError: fetch failed)`; it supplied no product result. No cause beyond the observed resource shortage and unreliable access is asserted.
