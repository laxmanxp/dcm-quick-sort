# DCM v1.0.71 — resume after DCM-26

Latest accepted checkpoint: [DCM-26](dcm26-complete.md). Completed: **DCM-01–18 and DCM-20–26**; DCM-19 remains blocked. Only DCM-26 newly closed. The 25 checklist steps include five setup/diagnostic steps, not a product percentage. Human acceptance remains unrecorded.

## Verified state

Final tested source: `921f3616da39e8c67f27ba1d496eaaf3594526ad`. Native baseline **361 passed / 0 failed / 1 external-PPTX skip** across 362 parents; no timeout, source unchanged. Three new Go parents contain 31 nested encoding/limit/read-safety cases. Native ui-text passes 17 checks on 15 disposable fixtures; all 17 final images inspected, including visible cutoff notices and the current manual text status. Normal exit 0, children drained, source/fixture contents unchanged, no operation journal. Launch and Refresh commands also pass at the same revision; their new images were not separately reviewed.

Core decoder now accepts valid UTF-8 and BOM-marked UTF-16LE/BE, refuses malformed/unsupported encodings instead of silently replacing bytes, and drops only incomplete characters cut by the 256-KiB preview bound. Read/encoding errors remain specific rather than falling through to generic missing-handler messaging. Explicit Read as text updates its own status. Automatic text routing remains TXT/MD/CSV; Markdown and CSV are literal text, not rich rendering. No legacy encoding guessing, UTF-32 support, installer or engine changes.

Preserve all REDs/tooling failures in the report: native unit failures; locked-file misleading error; manual stale status; wait-variable shadowing; two capture-guard failures. The first 17-assertion pass was not final acceptance: image review exposed missing tail captures/manual status. The corrected test uses an actual end caret and first-visible-line check; all content/hash/ownership guards remain intact. Do not reapply historical proposals.

## Next exact step — DCM-27

Inspect the existing supported image-preview formats and qualify representative actual content plus existing sizing/navigation behavior. Use disposable known images and reviewed captures, not status-label-only assertions. Do not switch preview engines, add printing, or close other format gates incidentally.

Code locally -> GitHub -> FlowDev update/test on Lenovo, serially. Candidate `C:\FlowverseReports\dcm-quicksort-candidate`; profile `tools\flowdev-dcm`; runner `C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe`; workspace `C:\FlowverseReports`; device `d18227f2-0b54-4b11-9026-56ac40305f02`. No remote source editing. Check active sessions before any native UI input.

Last build: `C:\FlowverseReports\dcm-quicksort-build-20261111-144207-01065b1a\receipt.json`, matches 921f3616 only. App hash `0522a02560bea2355c6d307eca807320836a51b4e9a8553470c8107b33f2523e`. Obtain a fresh source-matching build after this metadata commit or any later revision; never rewrite receipt identities.

Reports under C:\FlowverseReports: baseline `dcm-quicksort-baseline-20261111-144109-ae0c2e92`; text `dcm-quicksort-ui-text-20261111-144427-b716381b`; launch `dcm-quicksort-ui-launch-20261111-144456-9fa864b0`; refresh `dcm-quicksort-ui-refresh-20261111-144512-afa1b71e`. Separate review/hashes: `evidence/dcm-26-native-summary.json`. Raw visual flags remain unchanged; images stay on Lenovo because small desktop edges are visible.

Keep DCM-19/disposable Explorer isolation, DCM-UI-LABEL-01, identified external DCM_REAL_PPTX before DCM-32/47, remaining previews and file-operation workflows, full Batch Move/Undo, final package/setup lifecycle and actual user acceptance open. Leave DCM21_PROBE_EMPTY_DESTINATION unset in normal acceptance. PR #1 remains draft; no merge/publication without final gates and approval.
