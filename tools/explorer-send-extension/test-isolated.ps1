# DCM QuickSort Explorer COM acceptance without Windows shell registration.
# Exercises a real compiled COM DLL with a disposable data object and receiver.
# Does NOT install software, write registry keys, restart Explorer, or move files.
[CmdletBinding()]
param([string]$ReportDir = (Join-Path $env:TEMP ('dcm-explorer-isolated-' + [guid]::NewGuid().ToString('N'))))
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$root = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$stage = Join-Path $ReportDir 'build'
New-Item -Path $stage -ItemType Directory -Force | Out-Null
$receipt = [ordered]@{
  schema = 1
  suite = 'dcm-explorer-com-isolated'
  started_utc = [DateTime]::UtcNow.ToString('o')
  source_commit = (& git -C $root rev-parse HEAD | Out-String).Trim()
  passed = $false
  real_explorer_gui_tested = $false
  windows_shell_registered = $false
  actual_file_move_tested = $false
  output = @()
  error = $null
}
try {
  $build = Join-Path $PSScriptRoot 'build.ps1'
  $buildOutput = @(& $build -OutDir $stage 2>&1 | ForEach-Object { $_.ToString() })
  if ($LASTEXITCODE -ne 0) { throw 'Extension build failed' }
  $smoke = Join-Path $stage 'explorer-smoke.exe'
  $receiver = Join-Path $stage 'fixture-receiver.exe'
  $source = Join-Path $PSScriptRoot 'smoke.cpp'
  & clang++ -std=c++17 -O2 -static -municode -o $smoke $source -lole32 -lshell32 -luuid -ladvapi32
  if ($LASTEXITCODE -ne 0) { throw 'Unicode C++ smoke harness build failed' }
  & go build -o $receiver (Join-Path $PSScriptRoot 'receiver_fixture.go')
  if ($LASTEXITCODE -ne 0) { throw 'Harmless receiver fixture build failed' }
  $output = @(& $smoke (Join-Path $stage 'DCMQuickSortExplorerSend.dll') (Join-Path $stage 'DCMQuickSortExplorerSend.exe') $receiver 2>&1 | ForEach-Object { $_.ToString() })
  $receipt.output = $output
  if ($LASTEXITCODE -ne 0) { throw ('COM smoke harness failed with exit code ' + $LASTEXITCODE) }
  $required = @(
    'PASS: 2 selected files, 2 live destination favorites, one dynamic submenu',
    'PASS: submenu command launched one helper handoff carrying both files; fixture source bytes unchanged',
    'PASS: folder selection is excluded',
    'PASS: unloaded DLL and removed disposable fixture'
  )
  foreach ($line in $required) {
    if ($output -notcontains $line) { throw ('Expected verification not observed: ' + $line) }
  }
  $receipt.dll_sha256 = (Get-FileHash (Join-Path $stage 'DCMQuickSortExplorerSend.dll') -Algorithm SHA256).Hash.ToLowerInvariant()
  $receipt.helper_sha256 = (Get-FileHash (Join-Path $stage 'DCMQuickSortExplorerSend.exe') -Algorithm SHA256).Hash.ToLowerInvariant()
  $receipt.passed = $true
  $output | ForEach-Object { Write-Output $_ }
} catch {
  $receipt.error = $_.Exception.Message
  Write-Error -Message $receipt.error -ErrorAction Continue
} finally {
  $receipt.finished_utc = [DateTime]::UtcNow.ToString('o')
  $receipt | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $ReportDir 'receipt.json') -Encoding UTF8
  Write-Output ('RECEIPT=' + (Join-Path $ReportDir 'receipt.json'))
}
if (-not $receipt.passed) { exit 1 }
