# DCM QuickSort — project context

Checkpoint: 2026-11-09. This is a documentation checkpoint; it does not certify a new build or record new tests.

## Product decision

**FileFlow Suite** is the umbrella for **DCM QuickSort, Flow Search, DeepSearch and FileFlow**, with each remaining a separate application/window. DCM owns its Preview → Rename → Move → Next workflow, local learning, destinations, Processed history, recovery and FileFlow handoff.

Reuse the existing native Go implementation and established behavior. A Walk/Rust rewrite, replacement PDF renderer or shared launcher is not a prerequisite for DCM release qualification. Any future shared integration should preserve each app's independent launch and existing functionality.

## Version and source checkpoints

| Checkpoint | Evidence and limit |
|---|---|
| Latest published release | [v1.0.63](https://github.com/laxmanxp/dcm-quick-sort/releases/tag/v1.0.63), published 2026-09-23, with `DCM-QuickSort-Setup-v1.0.63.exe`. The release notes record installer SHA-256 `6be092ca2b11e8f56f4183beaa7fab59ef173ae653bdc29126fc3e60911b597c`; this checkpoint did not download or rehash it. |
| Latest identified native acceptance record | v1.0.69 on Lenovo, 2026-11-03. [The report](https://github.com/laxmanxp/dcm-quick-sort/blob/2ece9b1e044545c666b6c6296052feb6b985b750/tests/flowuivision/TEST_REPORT_2026-11-03.md) is partial, not a full release PASS. Tested EXE SHA-256: `BF93CE364909F8541D015D42CAFF3A2B0970574197CD60D0A0E756DF97E9683B`. The report commit identifies evidence, not the executable's source revision. |
| Recovered source in GitHub | [v1.0.62 recovery](https://github.com/laxmanxp/dcm-quick-sort/tree/c27d514ffeb2a76a8d77d776855ecf22b460930c/source-v1.0.62). Its transfer context records app/installer test and build passes, with live batch acceptance and final packaged-source reproducibility still requiring qualification. |
| Newer supplied archive | `DCM-QuickSort-v1.0.70-Source-and-Installer(1).zip` was audited on 2026-11-02. Full v1.0.70 source has not been imported into the canonical source branch, and its source-to-installer provenance and native release qualification remain unresolved. |

The repository main head read for this checkpoint was `e24a3a15aa88ca78a1fbe8ddecfdbd1cceae8edc`, tree `e98e21da9a92b769a40f4c2b7644bf3a4f7d986d`. Main contained release/site assets and an outdated v1.0.53 README, not a canonical current app source checkout. The QA and recovery branches are separate evidence sources. Reconcile source provenance before fixing or qualifying a new release.

## v1.0.70 archive safety findings

Source: [QuickSort code comparison, 2026-11-02](https://chatgpt.com/library?file_id=libfile_2c8db3c7893481919491f94250a2c300&account_id=personal). This historical Linux source audit reported **327 app tests passing, one real-PPTX fixture skip, 20 installer tests passing and cross-builds passing**. Eight added probes failed their intended assertions across the six findings below. Those passes do not close the added failures. No later fix or Windows qualification is verified here, and these findings are not attributed automatically to the older published v1.0.63 binary.

| ID | Finding and evidence distinction | Remaining gate |
|---|---|---|
| D1 — high | An exact same physical source/target can be treated as its own duplicate and removed; Undo cannot recover it. Exact-path deletion was reproduced in the source audit. The Windows case-only UI route was source-traced, not natively reproduced. | Guard file identity and verify exact-path and Windows alias/case variants preserve the source. |
| D2 — high | During cross-volume copy, the source may change before deletion. A controlled same-length edit left older target bytes and removed the newer source in the audit. | Preserve a changed source and qualify the copy/delete boundary with independent byte checks. |
| D3 — medium | The batch worker attaches `BatchID` late, leaving a crash interval without batch membership. Individual transaction history remains. | Make batch membership durable at the required transaction boundary and verify restart behavior. |
| D4 — medium | Naming templates are exported and included in the digest but omitted from the import allowlist, so a valid export fails checksum validation. | Qualify export/import round-trip with naming templates. |
| D5 — medium | Installer restore recognizes `default_source` but excludes `pdf_default_view` and `vault_root`. | Preserve these settings through backup/restore and installer upgrade. |
| D6 — Undo contract | Duplicate-removal Undo copies the destination's current bytes. If that destination changed, Undo restores the changed content rather than the removed original. | Define and implement truthful original-byte restoration or a refusal when original-byte recovery is unavailable. |

All six remain open qualification items. Their detailed follow-up is tracked in [todo.json](todo.json).

## Native evidence and remaining coverage

The v1.0.69 Lenovo report confirmed launch/version, filters and no-match state, sorting/navigation, TXT/PNG previews, lightweight DOCX/XLSX/PPTX fallbacks, initial PDF/Fit Width, spacing cleanup, Rename and Undo Rename. The Office results do not certify full Office page-layout fidelity. Its 53 filter/navigation steps are actions/assertions, not 53 distinct completed features.

Open native gates from that report:

- Manual PDF 100%/110% captures showed a blank document area although footer assertions passed; Fit Width restored content. Rendering versus capture/focus cause remains unresolved.
- Move to Vault did not complete in automated attempts, and the independent filesystem oracle failed. This does not prove every normal user move is broken. Undo Move was not exercised because no move was confirmed.
- Batch selection/move was not certified because the FlowUIVision adapter could not select the native ListBox. Multi-select and batch move already exist in the recovered source and the v1.0.63 release notes; this is a coverage/tooling gap, not evidence that the feature is absent.
- Legacy DOC/XLS/PPT, complex Office documents, protected/damaged files, overwrite collisions, crash recovery and physical printing were not certified in that pass.

Before a newer release: identify the exact source and installer payload; resolve D1–D6 with the appropriate regression evidence; qualify the actual Windows workflow and lifecycle; verify the packaged installer/upgrade/restore against the advertised DCM release scope. Physical printing is an unverified capability, not a newly imposed feature or release gate. Preserve both failed and successful evidence, including readable images and independent filesystem checks.

## Working rules

Code in the ChatGPT development environment; store changes in small GitHub commits; validate on Lenovo through the registered FlowDev app/feature route (`flowdev update <app>` then `flowdev test <app> <feature>`). Follow Feature → Units → Unit → Units → Feature. Record implementation, UI test and user-test status separately. Source-review probes, cross-builds, existing release notes and another app's tests must not be relabeled as current DCM native acceptance.
