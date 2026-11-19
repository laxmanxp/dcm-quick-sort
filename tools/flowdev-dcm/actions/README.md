# DCM QuickSort qualification profile

This directory marks the isolated FlowDev profile. Checks are declared in
`../config/checks.json`; they execute from the registered DCM candidate repository.
The standalone FlowDev executable discovers this profile from the working directory.

## DOCX damaged-content regression

After the pushed candidate is received with `flowdev update dcm-quicksort`, run
`flowdev test dcm-quicksort docx-failure-unit` for the `TestDCM34DOCX` failure,
valid-blank and recovery contracts. This selector intentionally excludes older
tests: the existing native runner fails if no matching tests execute. Run
`flowdev test dcm-quicksort docx-unit` for the neighboring DOCX/Open XML
regressions after a fix. Both use the same clean-source, native-Windows runner
and retain exact source manifests, JSON test events and failure receipts.

`flowdev test dcm-quicksort ui-docx` retains its natural installed-handler-first
policy. A successful Word preview in that workflow does not certify the built-in
fallback on a missing-handler machine; record that scope separately. These
selectors do not change Office registrations or introduce a provider override.

`flowdev test dcm-quicksort docx-fallback-component` runs the exact production
fallback function and preview painter in a visible, owned test window. It saves
screen captures, control state, fixture hashes and the test executable identity
under the native report's `docx-component/` directory. A separate image review
is required. This is Windows component acceptance, not normal-application
installed/absent-provider qualification. It does not invoke Office or modify
registry keys. In other selectors the same assertions use hidden controls and
do not capture images or claim headed acceptance.
