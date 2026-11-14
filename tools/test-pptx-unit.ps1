[CmdletBinding()]
param()
# DCM-32: supply the pinned real PPTX to the unchanged integration assertion.
# Keep the native runner's raw receipt intact; add a separate fixture/event check.
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
if ($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess) { throw 'Native Windows x64 required.' }
if (-not $env:FLOWDEV_WORKSPACE_ROOT) { throw 'FLOWDEV_WORKSPACE_ROOT is required.' }
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$fixtureRelative = 'tests/fixtures/dcm32-slides.pptx'
$fixture = Join-Path $repo $fixtureRelative
$fixtureManifestRelative = 'tests/fixtures/dcm32-slides.manifest.json'
$fixtureManifest = Join-Path $repo $fixtureManifestRelative
$expectedHash = 'af42fff55297109195b95039d2ac39c5d53b1fae23b59e7b40a7184d679b93f1'
$pattern = '^(Test(PPTX|RenderPPTX)|TestRealGeneratedPPTXIntegration$|Test(OpenXML|InspectOpenXML|ResolveOpenXML|ReadOpenXML|ReadZipEntryLimited|OfficeFoundation))'
$nativeReceiptPath = ''
$previousFixture = [Environment]::GetEnvironmentVariable('DCM_REAL_PPTX', 'Process')
$verification = [ordered]@{
    schema = 1; app = 'dcm-quicksort'; feature = 'DCM-32'; selector = 'pptx-unit'
    started_utc = [DateTime]::UtcNow.ToString('o'); passed = $false
    fixture = $fixtureRelative; fixture_sha256 = $expectedHash
    test = 'TestRealGeneratedPPTXIntegration'; scope = 'PPTX parser and fixture; no visible UI acceptance'
}
try {
    if (-not (Test-Path -LiteralPath $fixture -PathType Leaf)) { throw 'DCM32_PPTX_FIXTURE_MISSING: tracked tests/fixtures/dcm32-slides.pptx is required.' }
    if (-not (Test-Path -LiteralPath $fixtureManifest -PathType Leaf)) { throw 'DCM32_PPTX_MANIFEST_MISSING: tracked fixture manifest is required.' }
    try { $definition = Get-Content -LiteralPath $fixtureManifest -Raw -Encoding UTF8 | ConvertFrom-Json } catch { throw 'DCM32_PPTX_MANIFEST_INVALID: fixture manifest must contain valid JSON.' }
    foreach ($field in @('schema_version', 'feature', 'file', 'slide_count', 'sha256', 'bytes')) {
        if ($null -eq $definition -or -not $definition.PSObject.Properties[$field]) { throw ('DCM32_PPTX_MANIFEST_INVALID: missing fixture identity field ' + $field + '.') }
    }
    if ($definition.schema_version -ne 1 -or $definition.feature -cne 'DCM-32' -or $definition.file -cne 'dcm32-slides.pptx' -or $definition.slide_count -ne 2 -or $definition.sha256 -cne $expectedHash -or $definition.bytes -ne (Get-Item -LiteralPath $fixture).Length) { throw 'DCM32_PPTX_MANIFEST_IDENTITY: manifest does not identify the pinned two-slide fixture.' }
    $manifestHash = (Get-FileHash -LiteralPath $fixtureManifest -Algorithm SHA256).Hash.ToLowerInvariant()
    $verification['fixture_manifest'] = $fixtureManifestRelative
    $verification['fixture_manifest_sha256'] = $manifestHash
    $verification['fixture_bytes'] = $definition.bytes
    if ((Get-FileHash -LiteralPath $fixture -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expectedHash) { throw 'DCM32_PPTX_FIXTURE_IDENTITY: fixture bytes do not match the pinned presentation.' }
    $env:DCM_REAL_PPTX = $fixture
    & powershell.exe -NoProfile -NonInteractive -File (Join-Path $PSScriptRoot 'run-native.ps1') -Mode regressions -Run $pattern | ForEach-Object {
        $line = [string]$_
        Write-Output $line
        if ($line.StartsWith('RECEIPT=', [StringComparison]::Ordinal)) { $script:nativeReceiptPath = $line.Substring(8).Trim() }
    }
    if ($LASTEXITCODE -ne 0) { throw 'DCM32_PPTX_NATIVE_FAILED: inspect the native runner receipt and preserved logs.' }
    if (-not $nativeReceiptPath -or -not (Test-Path -LiteralPath $nativeReceiptPath -PathType Leaf)) { throw 'DCM32_PPTX_RECEIPT_MISSING: native runner did not return its exact receipt.' }
    $native = Get-Content -LiteralPath $nativeReceiptPath -Raw -Encoding UTF8 | ConvertFrom-Json
    $verification['native_receipt'] = $nativeReceiptPath
    $verification['native_receipt_sha256'] = (Get-FileHash -LiteralPath $nativeReceiptPath -Algorithm SHA256).Hash.ToLowerInvariant()
    $verification['commit'] = $native.commit
    if (-not $native.passed -or $native.mode -ne 'regressions' -or -not $native.source_unchanged) { throw 'DCM32_PPTX_NATIVE_RECEIPT: exact native regression and source-integrity checks must pass.' }
    $runs = @($native.test_runs | Where-Object { $_.name -eq 'app-regressions' })
    if ($runs.Count -ne 1 -or $runs[0].pattern -cne $pattern -or $runs[0].passed_count -lt 1 -or @($runs[0].failures).Count -ne 0 -or @($runs[0].skipped).Count -ne 0) { throw 'DCM32_PPTX_TARGETED_COVERAGE: required targeted tests failed, were skipped or did not execute.' }
    if ((Get-FileHash -LiteralPath $native.source_hash_manifest.path -Algorithm SHA256).Hash.ToLowerInvariant() -ne $native.source_hash_manifest.sha256) { throw 'DCM32_PPTX_SOURCE_MANIFEST: native source manifest identity changed.' }
    # Windows PowerShell 5.1 emits a parsed JSON array as one pipeline object.
    # Assign it directly so later filtering enumerates its actual source entries.
    $manifest = Get-Content -LiteralPath $native.source_hash_manifest.path -Raw -Encoding UTF8 | ConvertFrom-Json
    $fixtureEntries = @($manifest | Where-Object { $_.path -ceq $fixtureRelative })
    if ($fixtureEntries.Count -ne 1 -or $fixtureEntries[0].sha256 -cne $expectedHash) { throw 'DCM32_PPTX_TRACKED_FIXTURE: pinned fixture must belong to the native source manifest.' }
    $manifestEntries = @($manifest | Where-Object { $_.path -ceq $fixtureManifestRelative })
    if ($manifestEntries.Count -ne 1 -or $manifestEntries[0].sha256 -cne $manifestHash) { throw 'DCM32_PPTX_TRACKED_MANIFEST: fixture manifest must match the native source snapshot.' }
    $log = Join-Path (Split-Path -Parent $nativeReceiptPath) 'app-regressions.stdout.log'
    $verification['real_test_log'] = $log
    $verification['real_test_log_sha256'] = (Get-FileHash -LiteralPath $log -Algorithm SHA256).Hash.ToLowerInvariant()
    $events = @(Get-Content -LiteralPath $log -Encoding UTF8 | Where-Object { $_.Trim().StartsWith('{') } | ForEach-Object { $_ | ConvertFrom-Json })
    $real = @($events | Where-Object { $_.PSObject.Properties['Test'] -and $_.Test -ceq 'TestRealGeneratedPPTXIntegration' })
    $runCount = @($real | Where-Object { $_.Action -eq 'run' }).Count
    $passCount = @($real | Where-Object { $_.Action -eq 'pass' }).Count
    $rejected = @($real | Where-Object { $_.Action -in @('fail', 'skip') }).Count
    $verification['real_test_events'] = [ordered]@{run=$runCount; pass=$passCount; fail_or_skip=$rejected}
    if ($runCount -ne 1 -or $passCount -ne 1 -or $rejected -ne 0) { throw 'DCM32_PPTX_REAL_TEST: unchanged external-fixture test must execute and pass exactly once, without skips.' }
    if ((Get-FileHash -LiteralPath $fixture -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expectedHash) { throw 'DCM32_PPTX_FIXTURE_CHANGED: original fixture changed during testing.' }
    if ((Get-FileHash -LiteralPath $fixtureManifest -Algorithm SHA256).Hash.ToLowerInvariant() -ne $manifestHash) { throw 'DCM32_PPTX_MANIFEST_CHANGED: fixture manifest changed during testing.' }
    $verification['native_tests_passed'] = $runs[0].passed_count
    $verification['fixture_preserved'] = $true
    $verification.passed = $true
} catch {
    $verification['error'] = $_.Exception.Message
    Write-Output ('FAIL=' + $_.Exception.Message)
} finally {
    [Environment]::SetEnvironmentVariable('DCM_REAL_PPTX', $previousFixture, 'Process')
    $verification['finished_utc'] = [DateTime]::UtcNow.ToString('o')
    if ($nativeReceiptPath -and (Test-Path -LiteralPath $nativeReceiptPath -PathType Leaf)) {
        $verificationPath = Join-Path (Split-Path -Parent $nativeReceiptPath) 'pptx-fixture-verification.json'
        [IO.File]::WriteAllText($verificationPath, ($verification | ConvertTo-Json -Depth 10), (New-Object Text.UTF8Encoding($false)))
        Write-Output ('PPTX_VERIFICATION=' + $verificationPath)
    }
}
if (-not $verification.passed) { exit 1 }
exit 0
