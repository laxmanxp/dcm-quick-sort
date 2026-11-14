# DCM-33 — legacy Office preview qualification

Started 14 November 2026 (Asia/Kolkata). Standalone DCM QuickSort v1.0.71,
repository `laxmanxp/dcm-quick-sort`, branch `release/v1.0.71`.

**IN PROGRESS. The numbered plan remains 31 complete / 19 remaining.**
DCM-33 covers DOC, XLS and PPT. This baby step starts with genuine legacy DOC;
it cannot by itself close the numbered step or certify the final package.

## Starting identity and observed contract

The development checkout and Lenovo initially match pushed completion commit
`d4edf54b60b4ce86b6642df43547a77506bc7c4a`. The registered FlowDev profile maps
`dcm-quicksort` to `D:\FlowverseReportsRepo\dcm-quicksort-candidate`, the same
GitHub repository and `release/v1.0.71` branch. The initial checkout is clean.
Read-only lookup finds Word's DOC preview handler registered at `.doc\shellex`;
registration alone does not prove content can be rendered.

Legacy DOC/XLS/PPT use an installed Windows preview handler. The built-in Open
XML renderers do not treat legacy binary files as DOCX/XLSX/PPTX. If a handler
is absent or fails, the existing fallback explains the outcome and leaves the
ordinary rename/move/open workflow available. No new rendering engine is planned.

## Narrow defect under test

The generic unavailable branch updates the pane's fallback text but leaves
`statusLabel` unchanged. It can therefore retain a prior TXT or native-preview
success caption. The regression first extracts the existing final presentation
statements without changing behavior, then exercises the actual presenter with
owned hidden Win32 controls. The behavior-preserving extraction and assertions were pushed at
`cc0948108bff8c6ccc7721a4dfda69faac59bc91` before native execution.

The intended footer distinguishes a missing handler from a registered handler
that failed and identifies the current file type. Tests cover blank, prior TXT
success and prior native success. Missing-provider behavior is exercised at the
presentation boundary; tests do not launch Office or change live registrations.

## Focused qualification path

The fixture is a locally generated, synthetic Word 97 binary document, pinned
by size and SHA-256 in its manifest. Qualification consumes the committed bytes.
The exact manifest supplies the visible text oracle and records format validation
and generation provenance. It is not a renamed RTF or DOCX file.

The headed path uses a harmless unsupported-extension control to exercise the
shared unavailable display, then previews the genuine DOC through the naturally
selected provider. It returns to TXT, checks exclusive file release, reopens DOC
and closes normally. It binds source, app and fixture hashes to the exact build,
checks an empty journal and preserves all captured frames. Actual visible words
and blank/error/success content require separate image review.

```text
flowdev update dcm-quicksort
flowdev test dcm-quicksort legacy-preview-unit
flowdev test dcm-quicksort ui-tooling
flowdev test dcm-quicksort build
flowdev test dcm-quicksort ui-legacy-doc
```

Supply `DCM_BUILD_RECEIPT` from the exact current clean build to the UI gate.
The focused unit gate precedes larger regression/feature runs. A production fix
also receives the native baseline after focused GREEN. Never infer a native test
pass from local cross-compilation, or semantic acceptance from a provider caption.

## Current evidence and remaining work

Native RED is recorded at `cc0948108bff8c6ccc7721a4dfda69faac59bc91` in
`D:\FlowverseReportsRepo\dcm-quicksort-regressions-20261114-121711-6f555113`.
The focused gate reports two top-level passes, one top-level failure, no skips
and exit 1 without timeout. All nine failures are the intended
`DCM33_UNAVAILABLE_STATUS` assertion: blank, prior TXT or prior native-success
footers survive missing-handler, failed-handler and extensionless outcomes.
All hidden-control setup and fallback-body assertions succeed. Source is
unchanged. The raw log SHA-256 is
`8a33ebc502337b9388153a8df85b1ec153517d20509c593890ca2368602116d4`.

The minimal fix is being integrated after this observed RED. Native GREEN,
the production baseline and headed DOC evidence remain pending. This checkpoint
does not claim acceptance; DOC/XLS/PPT remain pending in the numbered plan.

After the DOC milestone, continue with genuine legacy XLS and PPT in separate
baby steps. Broad damaged/protected-file and repeated lifecycle qualification
remain DCM-34/35. Final layout, package and user acceptance remain open.
