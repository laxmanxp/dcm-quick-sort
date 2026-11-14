# DCM QuickSort native qualification

Use the existing standalone FlowDev runtime with this repository's isolated profile.
Do not use FileFlow's profile, the historical DCM clone, or the installed app.

```powershell
$env:FLOWDEV_WORKSPACE_ROOT = 'C:\FlowverseReports'
Set-Location 'C:\FlowverseReports\dcm-quicksort-candidate\tools\flowdev-dcm'
$flowdev = 'C:\FlowverseReports\flowdev-standalone-validation\flowdev-test.exe'
& $flowdev update dcm-quicksort
& $flowdev test dcm-quicksort d3-journal
& $flowdev test dcm-quicksort d4-templates
& $flowdev test dcm-quicksort installer-regressions
& $flowdev test dcm-quicksort baseline
& $flowdev test dcm-quicksort package
```

The initial candidate clone is created by the committed FlowDev bootstrap selector,
before changing to this profile. The updater itself intentionally does not clone.

The runner sets `CGO_ENABLED=0` and uses the stable tooling cache
`%FLOWDEV_WORKSPACE_ROOT%\dcm-quicksort-go-cache`; each run still has separate app
configuration. `format` reports gofmt differences and emits formatted copies into
its report only. It never reformats the committed checkout on the device.

Each check requires a clean `release/v1.0.71` checkout with the expected origin. It
copies tracked source into a uniquely named report directory outside the checkout,
records commit and per-file SHA-256 identity, and verifies that the original source
stays unchanged. Native `go test -json -count=1` runs preserve raw output, individual
failures, and skips. Targeted selectors fail when they execute zero tests.

`baseline` runs all app unit tests. `regressions` runs `^Test(D[1-6]|Release)`.
`pptx-unit` runs the focused PPTX/Open XML parser tests through that same native
runner. It supplies the committed `tests/fixtures/dcm32-slides.pptx` with its
manifest, verifies both source identities, and requires the unchanged
`TestRealGeneratedPPTXIntegration` to emit one run and one pass event with no skip.
The wrapper restores any previous `DCM_REAL_PPTX` value and records a separate
`pptx-fixture-verification.json`; the native receipt and raw test log are retained
unchanged. A missing, altered or malformed fixture cannot pass by skipping the
external-fixture test. This selector does not certify visible slides.
`native-preview-rect-unit` runs the focused Windows DPI measurement tests on
owned hidden borderless windows. Non-100% scaling and supported DPI contexts are
required for qualification; skips are not accepted as a physical-coordinate pass.
`excel-preview-layout-unit` checks the Excel outer-window repair boundary using
hidden owned test windows: nine DPI geometry cases and 39 topology/handle/style
guard cases. It does not launch Excel or replace foreign-provider UI acceptance.
`installer-regressions` uses that pattern in the installer module; `installer-unit`
runs its entire suite. `installer-integration` runs the in-process installer engine
in a verified private HKCU mapping and disposable configuration. `build` produces
the app for subsequent owned UI testing. `package` runs both full suites, the
private-registry engine gate, and app/installer builds from the same snapshot.
The exact app bytes are staged as the embedded installer payload, and the package
verifier checks that those bytes occur exactly once in the Windows amd64 installer.
Hashes and receipts accompany the artifacts. The core/package runner never starts
the setup executable. `installer-ui-cancel` separately opens the hash-bound actual
setup and activates only Cancel, while proving unchanged installed state.

Passing these checks is source/package evidence. UI screenshots require separate
review, and installer registration requires a disposable Windows user/VM because
changing LOCALAPPDATA/APPDATA does not isolate HKCU integration keys.

For the focused PPTX feature, use `pptx-unit`, `ui-tooling`, and `build` after the
registered update. Set `DCM_BUILD_RECEIPT` to that exact build receipt, then run
`flowdev test dcm-quicksort ui-pptx`. The harness observes the installed Windows
handler or built-in fallback selected by the application. It does not change
Office registration. Review the actual slide captures separately before recording
DCM-32 acceptance; parser success and a ready provider footer are insufficient.
Native Page Down/Up input is confined to the observed embedded PowerPoint slide
window with revalidated process/window identity. Every capture attempt is retained;
separate image review must confirm both slides, their order, and restored content.

See `../tests/native/README.md` for `ui-tooling`, `ui-move`, `ui-batch`, `ui-pdf`,
`ui-previews`, `ui-pptx`, their exact artifact binding, and the isolated installer test plan.
