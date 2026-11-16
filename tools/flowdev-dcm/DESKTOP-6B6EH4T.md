# Isolated DCM FlowDev acceptance on DESKTOP-6B6EH4T

The compiled Flowverse monorepo FlowDev (`laxmanxp/flowverse`, `dev`) has its own application registry and does **not** register `dcm-quicksort`. Do not modify the Flowverse monorepo's deployed registry on a test computer to make this pass.

DCM QuickSort instead owns the narrowly scoped source-controlled profile adapter in `tools/flowdev-dcm/cmd/flowdev-test`. It discovers only this repository's `tools/flowdev-dcm/config/checks.json` and `repos.list`, validates candidate origin, branch and clean source, permits only `update dcm-quicksort` (fast-forward only) and `test dcm-quicksort <registered selector>`, runs the selector directly without a shell, then verifies the candidate commit and cleanliness remain unchanged. It is **not** a claim of full parity with the older standalone FlowDev runtime or all monorepo FlowDev actions. The registered PowerShell test runner owns fixture and build-receipt validation and generates its own report.

## Desktop paths

- Checkout: `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-candidate` (`release/v1.0.71`)
- Workspace: `E:\FlowverseTesting\Builds\DCM-QuickSort`
- Stable tools: `E:\FlowverseTesting\Builds\DCM-QuickSort\Tooling\flowdev-test.exe` and `FlowUIVision.exe`
- Source-controlled profile: `<candidate>\tools\flowdev-dcm`
- Source for the separately built full FlowDev and FlowUIVision helpers: sparse checkout of `laxmanxp/flowverse` `dev` under `E:\FlowverseTesting\Temp\flowverse-flowdev-source`

## Build and run

Build adapter from the **clean committed** candidate without editing the source on the test machine:

```powershell
$env:PATH = 'C:\Program Files\Git\cmd;C:\Program Files\Git\ucrt64\bin;' + $env:PATH
$env:FLOWDEV_WORKSPACE_ROOT = 'E:\FlowverseTesting\Builds\DCM-QuickSort'
$root = Join-Path $env:FLOWDEV_WORKSPACE_ROOT 'dcm-quicksort-candidate'
Set-Location (Join-Path $root 'tools\flowdev-dcm')
go test ./cmd/flowdev-test
go build -trimpath -o 'E:\FlowverseTesting\Builds\DCM-QuickSort\Tooling\flowdev-test.exe' ./cmd/flowdev-test
$flowdev = 'E:\FlowverseTesting\Builds\DCM-QuickSort\Tooling\flowdev-test.exe'
& $flowdev update dcm-quicksort
& $flowdev test dcm-quicksort ui-tooling
& $flowdev test dcm-quicksort build
```

Read the *fresh* successful `dcm-quicksort-build-*\receipt.json` for the same pushed source commit. Set `DCM_BUILD_RECEIPT` to its exact absolute path and `DCM_FLOWUIVISION_EXE` to the compiled, pinned helper `Tooling\FlowUIVision.exe` before running headed gates. For example:

```powershell
$env:DCM_BUILD_RECEIPT = '<full successful build receipt path>'
$env:DCM_FLOWUIVISION_EXE = 'E:\FlowverseTesting\Builds\DCM-QuickSort\Tooling\FlowUIVision.exe'
& $flowdev test dcm-quicksort ui-undo-rename
```

`FlowUIVision.exe` is built separately from the pinned `laxmanxp/flowverse` `dev` tree's `apps/flowuivision/cmd/agent` Go target, not from the QuickSort app source.

## Evidence and restrictions

Record exact git commit, tool SHA-256, individual registered selector, raw receipt, and separate GUI image review. The DCM-37 passing raw receipt on this host is `E:\FlowverseTesting\Builds\DCM-QuickSort\dcm-quicksort-ui-undo-rename-20261116-151836-bcd20ac1\receipt.json`, tested at `d5cd6df5618433d8b3806908232bb2ddef0160e6`. A documentation commit after that test must **not** be substituted into its historical receipt. Never test through an unregistered shell command, mutate the production candidate on the target, alter Office registrations, or mark the user acceptance as completed from a machine PASS.
