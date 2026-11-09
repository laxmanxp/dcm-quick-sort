[CmdletBinding()]
param(
    [ValidateSet('baseline','regressions','installer-unit','installer-regressions','installer-integration','build','package','format')]
    [string]$Mode = 'baseline',
    [string]$Run = '^Test(D[1-6]|Release)',
    [string]$ExpectedBranch = 'release/v1.0.71'
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
if ($env:OS -ne 'Windows_NT') { throw 'DCM qualification must run natively on Windows through FlowDev.' }
if (-not [Environment]::Is64BitProcess) { throw 'Use 64-bit PowerShell for the Windows amd64 candidate.' }
if (-not $env:FLOWDEV_WORKSPACE_ROOT) { throw 'FLOWDEV_WORKSPACE_ROOT must be set by the selected FlowDev profile.' }

$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$workspace = [IO.Path]::GetFullPath($env:FLOWDEV_WORKSPACE_ROOT)
$report = Join-Path $workspace ('dcm-quicksort-' + $Mode + '-' + [DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss') + '-' + [Guid]::NewGuid().ToString('N').Substring(0,8))
if ($report.StartsWith($repo + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Reports must be outside the candidate checkout.' }
New-Item -ItemType Directory -Path $report | Out-Null
$source = Join-Path $report 'source'
$artifacts = Join-Path $report 'artifacts'
New-Item -ItemType Directory -Path $source,$artifacts | Out-Null
$goCache = Join-Path $workspace 'dcm-quicksort-go-cache'
New-Item -ItemType Directory -Path $goCache -Force | Out-Null
$utf8 = New-Object Text.UTF8Encoding($false)
$commands = New-Object 'Collections.Generic.List[object]'
$testRuns = New-Object 'Collections.Generic.List[object]'
$receipt = [ordered]@{
    schema = 1; app = 'dcm-quicksort'; mode = $Mode; started_utc = [DateTime]::UtcNow.ToString('o')
    report = $report; repository = $repo; passed = $false
    installer_execution = 'setup EXE not executed; engine install/uninstall only in explicitly reported private-HKCU stage'
    commands = $commands; test_runs = $testRuns
}
Write-Output ('REPORT_ROOT=' + $report)

function Write-Json([string]$Path, $Value) {
    [IO.File]::WriteAllText($Path, ($Value | ConvertTo-Json -Depth 30), $utf8)
}
function Write-Checkpoint([string]$Phase, $Details) {
    $checkpoint = [ordered]@{schema=1;app='dcm-quicksort';mode=$Mode;runner_pid=$PID;updated_utc=[DateTime]::UtcNow.ToString('o');phase=$Phase;details=$Details}
    $temporary = Join-Path $report 'active-step.tmp'
    Write-Json $temporary $checkpoint
    Move-Item -LiteralPath $temporary -Destination (Join-Path $report 'active-step.json') -Force
}
function Quote-Argument([string]$Value) {
    # Windows CommandLineToArgvW quoting, including terminal backslashes.
    return '"' + (($Value -replace '(\\*)"', '$1$1\"') -replace '(\\+)$', '$1$1') + '"'
}
function Invoke-OwnedCommand([string]$Name, [string]$File, [string[]]$Arguments, [string]$Directory, [int]$Seconds = 600) {
    Write-Checkpoint 'command-starting' ([ordered]@{name=$Name;file=$File;arguments=$Arguments;directory=$Directory;deadline_seconds=$Seconds})
    $outPath = Join-Path $report ($Name + '.stdout.log')
    $errPath = Join-Path $report ($Name + '.stderr.log')
    $psi = New-Object Diagnostics.ProcessStartInfo
    $psi.FileName = $File
    $psi.Arguments = (($Arguments | ForEach-Object { Quote-Argument $_ }) -join ' ')
    $psi.WorkingDirectory = $Directory
    $psi.UseShellExecute = $false
    $psi.CreateNoWindow = $true
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.EnvironmentVariables['GOWORK'] = 'off'
    $psi.EnvironmentVariables['GOOS'] = 'windows'
    $psi.EnvironmentVariables['GOARCH'] = 'amd64'
    $psi.EnvironmentVariables['GOFLAGS'] = ''
    $psi.EnvironmentVariables['CGO_ENABLED'] = '0'
    $psi.EnvironmentVariables['GOCACHE'] = $goCache
    $psi.EnvironmentVariables['LOCALAPPDATA'] = Join-Path $report 'profile\local'
    $psi.EnvironmentVariables['APPDATA'] = Join-Path $report 'profile\roaming'
    $psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR'] = Join-Path $report 'configuration-backups'
    $psi.EnvironmentVariables['DCM_INSTALLER_EVIDENCE_ROOT'] = Join-Path $report 'installer-engine'
    $proc = New-Object Diagnostics.Process
    $proc.StartInfo = $psi
    $entry = [ordered]@{name=$Name; file=$File; arguments=$Arguments; directory=$Directory; started_utc=[DateTime]::UtcNow.ToString('o'); stdout=$outPath; stderr=$errPath; timed_out=$false}
    $commands.Add($entry)
    try {
        if (-not $proc.Start()) { throw "Could not start $Name" }
        $entry['pid'] = $proc.Id
        Write-Checkpoint 'command-running' ([ordered]@{name=$Name;pid=$proc.Id;started_utc=$entry.started_utc;deadline_seconds=$Seconds;stdout=$outPath;stderr=$errPath})
        $outTask = $proc.StandardOutput.ReadToEndAsync()
        $errTask = $proc.StandardError.ReadToEndAsync()
        if (-not $proc.WaitForExit($Seconds * 1000)) {
            $entry.timed_out = $true
            & "$env:SystemRoot\System32\taskkill.exe" /PID $proc.Id /T /F | Out-Null
            if (-not $proc.WaitForExit(10000)) { throw "Owned command $Name could not be drained after deadline." }
        }
        $stdout = $outTask.GetAwaiter().GetResult()
        $stderr = $errTask.GetAwaiter().GetResult()
        [IO.File]::WriteAllText($outPath, $stdout, $utf8)
        [IO.File]::WriteAllText($errPath, $stderr, $utf8)
        $entry['exit_code'] = $proc.ExitCode
        $entry['finished_utc'] = [DateTime]::UtcNow.ToString('o')
        Write-Checkpoint 'command-finished' ([ordered]@{name=$Name;pid=$proc.Id;exit_code=$proc.ExitCode;timed_out=$entry.timed_out})
        return [pscustomobject]@{Stdout=$stdout; Stderr=$stderr; ExitCode=$proc.ExitCode; TimedOut=$entry.timed_out}
    } finally { $proc.Dispose() }
}
function Require-Success($Result, [string]$Name) {
    if ($Result.TimedOut -or $Result.ExitCode -ne 0) { throw "$Name failed (exit=$($Result.ExitCode), deadline=$($Result.TimedOut)); inspect the preserved command logs." }
}
function Git-Value([string]$Name, [string[]]$Arguments) {
    $result = Invoke-OwnedCommand $Name 'git.exe' (@('-C',$repo) + $Arguments) $repo 60
    Require-Success $result $Name
    return $result.Stdout.Trim()
}
function Source-Manifest([string[]]$Paths, [string]$Root, [string]$Phase='source-hashing') {
    $index = 0
    return @($Paths | ForEach-Object {
        $relative = $_
        $index++
        if ($index -eq 1 -or $index % 8 -eq 0 -or $index -eq $Paths.Count) { Write-Checkpoint $Phase ([ordered]@{root=$Root;file=$relative;index=$index;total=$Paths.Count}) }
        $full = [IO.Path]::GetFullPath((Join-Path $Root $relative))
        if (-not $full.StartsWith($Root + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw "Tracked path escapes source root: $relative" }
        if (-not [IO.File]::Exists($full)) { throw "Tracked file is missing: $relative" }
        [pscustomobject]@{path=$relative; size=(Get-Item -LiteralPath $full).Length; sha256=(Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash.ToLowerInvariant()}
    })
}
function Run-Tests([string]$Name, [string]$Directory, [string]$Pattern = '') {
    $arguments = @('test','-json','-count=1','-timeout=180s')
    if ($Pattern) { $arguments += @('-run',$Pattern) }
    $arguments += './...'
    $result = Invoke-OwnedCommand $Name 'go.exe' $arguments $Directory 300
    $events = @($result.Stdout -split '\r?\n' | Where-Object { $_.Trim().StartsWith('{') } | ForEach-Object { $_ | ConvertFrom-Json })
    $finished = @($events | Where-Object { $_.PSObject.Properties['Test'] -and $_.Test -notmatch '/' -and $_.Action -in @('pass','fail','skip') })
    $failures = @($finished | Where-Object { $_.Action -eq 'fail' } | ForEach-Object { $_.Test })
    $skips = @($events | Where-Object { $_.PSObject.Properties['Test'] -and $_.Action -eq 'skip' } | ForEach-Object { $_.Test })
    $summary = [ordered]@{name=$Name; pattern=$Pattern; count=$finished.Count; passed_count=@($finished | Where-Object { $_.Action -eq 'pass' }).Count; failures=$failures; skipped=$skips; exit_code=$result.ExitCode; timed_out=$result.TimedOut}
    $testRuns.Add($summary)
    Write-Json (Join-Path $report ($Name + '.summary.json')) $summary
    Require-Success $result $Name
    if ($finished.Count -eq 0) { throw "$Name ran no tests; a green process exit is insufficient." }
    if ($failures.Count -gt 0) { throw "$Name contains failed test events." }
    if ($Pattern -and $skips.Count -gt 0) { throw "$Name skipped targeted regression coverage; inspect the explicit skip events." }
}
function Build-App {
    $appExe = Join-Path $artifacts 'DCM-QuickSort.exe'
    $result = Invoke-OwnedCommand 'build-app' 'go.exe' @('build','-trimpath','-buildvcs=false','-ldflags=-H=windowsgui','-o',$appExe,'.') (Join-Path $source 'app-src')
    Require-Success $result 'build-app'
    if (-not (Test-Path -LiteralPath $appExe -PathType Leaf)) { throw 'App build did not produce the expected executable.' }
    $receipt['app_artifact'] = [ordered]@{path=$appExe; size=(Get-Item -LiteralPath $appExe).Length; sha256=(Get-FileHash -LiteralPath $appExe -Algorithm SHA256).Hash.ToLowerInvariant()}
    $payloadDir = Join-Path $source 'installer-src\payload'
    New-Item -ItemType Directory -Path $payloadDir -Force | Out-Null
    $payload = Join-Path $payloadDir 'DCM-QuickSort.exe'
    Copy-Item -LiteralPath $appExe -Destination $payload
    if ((Get-FileHash -LiteralPath $payload -Algorithm SHA256).Hash.ToLowerInvariant() -ne $receipt.app_artifact.sha256) { throw 'Installer payload staging changed the app bytes.' }
    $receipt['staged_payload'] = [ordered]@{path=$payload; sha256=$receipt.app_artifact.sha256}
}

$tracked = @()
$beforeManifestJson = ''
try {
    Write-Checkpoint 'identity-checks' ([ordered]@{repository=$repo;expected_branch=$ExpectedBranch})
    $receipt['commit'] = Git-Value 'git-head' @('rev-parse','HEAD')
    $receipt['branch'] = Git-Value 'git-branch' @('symbolic-ref','--quiet','--short','HEAD')
    $receipt['origin'] = Git-Value 'git-origin' @('remote','get-url','origin')
    if ($receipt.branch -ne $ExpectedBranch) { throw "Expected $ExpectedBranch, found $($receipt.branch)" }
    if ($receipt.origin -ne 'https://github.com/laxmanxp/dcm-quick-sort.git') { throw 'Unexpected DCM origin URL.' }
    if (Git-Value 'git-status-before' @('status','--porcelain=v1','--untracked-files=all')) { throw 'Candidate checkout is dirty; commit tests before native RED/GREEN.' }
    $ls = Invoke-OwnedCommand 'git-tracked' 'git.exe' @('-C',$repo,'ls-files','-z') $repo 60
    Require-Success $ls 'git-tracked'
    $tracked = @($ls.Stdout.Split([char]0) | Where-Object { $_ -ne '' } | Sort-Object)
    if ($tracked.Count -eq 0) { throw 'No tracked source files.' }
    $before = Source-Manifest $tracked $repo 'hashing-checkout'
    $beforeManifestJson = $before | ConvertTo-Json -Depth 5 -Compress
    Write-Json (Join-Path $report 'source-hashes.json') $before
    $receipt['source_hash_manifest'] = [ordered]@{path=(Join-Path $report 'source-hashes.json'); sha256=(Get-FileHash -LiteralPath (Join-Path $report 'source-hashes.json') -Algorithm SHA256).Hash.ToLowerInvariant(); files=$tracked.Count}
    $copyIndex = 0
    foreach ($relative in $tracked) {
        $copyIndex++
        if ($copyIndex -eq 1 -or $copyIndex % 8 -eq 0 -or $copyIndex -eq $tracked.Count) { Write-Checkpoint 'copying-source' ([ordered]@{file=$relative;index=$copyIndex;total=$tracked.Count;destination=$source}) }
        $destination = Join-Path $source $relative
        New-Item -ItemType Directory -Path (Split-Path -Parent $destination) -Force | Out-Null
        Copy-Item -LiteralPath (Join-Path $repo $relative) -Destination $destination
    }
    $copyJson = (Source-Manifest $tracked $source 'hashing-staged-copy') | ConvertTo-Json -Depth 5 -Compress
    if ($copyJson -cne $beforeManifestJson) { throw 'Staged source differs from the committed checkout bytes.' }
    $go = Invoke-OwnedCommand 'go-version' 'go.exe' @('version') $source 30
    Require-Success $go 'go-version'
    $receipt['go_version'] = $go.Stdout.Trim()
    $goEnv = Invoke-OwnedCommand 'go-environment' 'go.exe' @('env','-json','GOHOSTOS','GOHOSTARCH','GOOS','GOARCH','GOVERSION','CGO_ENABLED','GOWORK') $source 30
    Require-Success $goEnv 'go-environment'
    $receipt['go_environment'] = $goEnv.Stdout | ConvertFrom-Json
    if ($receipt.go_environment.GOHOSTOS -ne 'windows' -or $receipt.go_environment.GOHOSTARCH -ne 'amd64') { throw 'Cross-compilation is not native qualification.' }
    $receipt['go_cache'] = $goCache

    if ($Mode -eq 'format') {
        $goFiles = @($tracked | Where-Object { $_ -like '*.go' } | ForEach-Object { Join-Path $source $_ })
        $format = Invoke-OwnedCommand 'gofmt-check' 'gofmt.exe' (@('-l') + $goFiles) $source 60
        Require-Success $format 'gofmt-check'
        $differences = @($format.Stdout -split '\r?\n' | Where-Object { $_.Trim() })
        $receipt['format_differences'] = $differences
        if ($differences.Count -gt 0) {
            $formatted = Invoke-OwnedCommand 'gofmt-staged-only' 'gofmt.exe' (@('-w') + $differences) $source 60
            Require-Success $formatted 'gofmt-staged-only'
            Write-Json (Join-Path $report 'formatted-files.json') @($differences | ForEach-Object { [ordered]@{path=$_;sha256=(Get-FileHash -LiteralPath $_ -Algorithm SHA256).Hash.ToLowerInvariant();content=[IO.File]::ReadAllText($_)} })
            throw 'gofmt differences found; formatted copies are preserved in the report source tree and formatted-files.json. The checkout was not edited.'
        }
    }

    if ($Mode -in @('baseline','package')) { Run-Tests 'app-unit' (Join-Path $source 'app-src') }
    if ($Mode -eq 'regressions') { Run-Tests 'app-regressions' (Join-Path $source 'app-src') $Run }
    if ($Mode -in @('build','installer-unit','installer-regressions','installer-integration','package')) { Build-App }
    if ($Mode -in @('installer-unit','package')) { Run-Tests 'installer-unit' (Join-Path $source 'installer-src') }
    if ($Mode -eq 'installer-regressions') { Run-Tests 'installer-regressions' (Join-Path $source 'installer-src') $Run }
    if ($Mode -in @('installer-integration','package')) {
        Copy-Item -LiteralPath (Join-Path $source 'tests\native\installer_isolated_windows_test.go') -Destination (Join-Path $source 'installer-src\installer_isolated_windows_test.go')
        Run-Tests 'installer-engine' (Join-Path $source 'installer-src') '^TestNativeInstallerIsolatedRegistryLifecycle$'
        $enginePath = Join-Path $report 'installer-engine\engine-receipt.json'
        # Registry default values have an empty-string name. Keep those raw keys
        # intact; Windows PowerShell 5.1 cannot represent them as PSObject members.
        Add-Type -AssemblyName System.Web.Extensions
        $engineReader = New-Object System.Web.Script.Serialization.JavaScriptSerializer
        $engineReader.MaxJsonLength = 16 * 1024 * 1024
        $engine = $engineReader.DeserializeObject([IO.File]::ReadAllText($enginePath))
        if ($engine -isnot [Collections.IDictionary]) { throw 'Installer engine receipt must be a JSON object.' }
        if ($engine.passed -ne $true -or $engine.live_registry_unchanged -ne $true -or $engine.registry_override_reset -ne $true -or $engine.private_registry_removed -ne $true) { throw 'Installer engine did not prove successful private-registry cleanup and unchanged live keys.' }
        if ($engine.fresh_install.installed_app_sha256 -ne $receipt.app_artifact.sha256) { throw 'Installer engine installed different app bytes.' }
        $receipt['installer_engine'] = [ordered]@{path=$enginePath;sha256=(Get-FileHash -LiteralPath $enginePath -Algorithm SHA256).Hash.ToLowerInvariant();scope=$engine.scope;limitations=$engine.limitations}
    }
    if ($Mode -eq 'package') {
        $installerExe = Join-Path $artifacts 'DCM-QuickSort-Setup.exe'
        $built = Invoke-OwnedCommand 'build-installer' 'go.exe' @('build','-trimpath','-buildvcs=false','-ldflags=-H=windowsgui','-o',$installerExe,'.') (Join-Path $source 'installer-src')
        Require-Success $built 'build-installer'
        $verified = Invoke-OwnedCommand 'verify-payload' 'go.exe' @('run',(Join-Path $source 'tools\verify-payload\main.go'),'-app',$receipt.app_artifact.path,'-installer',$installerExe) $source 120
        Require-Success $verified 'verify-payload'
        $receipt['payload_identity'] = $verified.Stdout | ConvertFrom-Json
        Write-Json (Join-Path $artifacts 'payload-identity.json') $receipt.payload_identity
        $receipt['installer_artifact'] = [ordered]@{path=$installerExe; size=(Get-Item -LiteralPath $installerExe).Length; sha256=(Get-FileHash -LiteralPath $installerExe -Algorithm SHA256).Hash.ToLowerInvariant()}
        [IO.File]::WriteAllText((Join-Path $artifacts 'SHA256SUMS.txt'), ($receipt.app_artifact.sha256 + '  DCM-QuickSort.exe' + "`n" + $receipt.installer_artifact.sha256 + '  DCM-QuickSort-Setup.exe' + "`n"), $utf8)
    }
    $receipt.passed = $true
} catch {
    $receipt['error'] = $_.Exception.Message
    Write-Output ('FAIL=' + $_.Exception.Message)
} finally {
    try {
        if ($beforeManifestJson) {
            $afterJson = (Source-Manifest $tracked $repo 'verifying-unchanged-source') | ConvertTo-Json -Depth 5 -Compress
            $receipt['source_unchanged'] = ($afterJson -ceq $beforeManifestJson)
            if (-not $receipt.source_unchanged) { throw 'Tracked source changed during validation.' }
            if ((Git-Value 'git-head-after' @('rev-parse','HEAD')) -ne $receipt.commit) { throw 'Candidate HEAD changed during validation.' }
            if (Git-Value 'git-status-after' @('status','--porcelain=v1','--untracked-files=all')) { throw 'Candidate checkout became dirty during validation.' }
        }
    } catch { $receipt.passed=$false; $receipt['integrity_error']=$_.Exception.Message }
    $receipt['finished_utc'] = [DateTime]::UtcNow.ToString('o')
    Write-Json (Join-Path $report 'receipt.json') $receipt
    Write-Checkpoint 'finished' ([ordered]@{passed=$receipt.passed;receipt=(Join-Path $report 'receipt.json')})
    Write-Output ('RECEIPT=' + (Join-Path $report 'receipt.json'))
}
if (-not $receipt.passed) { exit 1 }
exit 0
