[CmdletBinding()]
param()
# Deterministic Windows contention test for runner checkpoint persistence.
# Test content is isolated outside the repository. No app/Office process launched.
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
if ($env:OS -ne 'Windows_NT') { throw 'Windows is required for checkpoint contention qualification' }
if (-not $env:FLOWDEV_WORKSPACE_ROOT) { throw 'FlowDev workspace root required' }
. (Join-Path $PSScriptRoot 'flowdev-checkpoint.ps1')
$root = Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-checkpoint-unit-' + [DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss') + '-' + [Guid]::NewGuid().ToString('N').Substring(0,8))
New-Item -ItemType Directory -Path $root -ErrorAction Stop | Out-Null
$target = Join-Path $root 'active-step.json'
$signal = Join-Path $root 'held.signal'
$receiptPath = Join-Path $root 'receipt.json'
$receipt = [ordered]@{schema=1;selector='checkpoint-unit';passed=$false;attempts=0;locked_write_retried=$false;target=$target;report=$root;source_unchanged=$true}
$job = $null
try {
    Write-FlowDevCheckpoint -Path $target -Json '{"step":0}'
    for ($i=1; $i -le 30; $i++) {
        Write-FlowDevCheckpoint -Path $target -Json ('{"step":'+$i+'}')
        $actual = Get-Content -LiteralPath $target -Raw -ErrorAction Stop | ConvertFrom-Json
        if ($actual.step -ne $i) { throw ('Checkpoint overwrite failed at '+$i) }
        $receipt.attempts++
    }
    $job = Start-Job -ScriptBlock {
        param($targetPath,$readyPath)
        $handle = New-Object IO.FileStream($targetPath, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::Read)
        try {
            [IO.File]::WriteAllText($readyPath, 'locked')
            Start-Sleep -Milliseconds 1600
        } finally { $handle.Dispose() }
    } -ArgumentList @($target,$signal)
    $deadline = [DateTime]::UtcNow.AddSeconds(12)
    while (-not (Test-Path -LiteralPath $signal)) {
        if ([DateTime]::UtcNow -gt $deadline -or $job.State -in @('Failed','Stopped','Completed')) { throw 'Locker failed to establish exclusive checkpoint hold' }
        Start-Sleep -Milliseconds 30
    }
    $started = [DateTime]::UtcNow
    Write-FlowDevCheckpoint -Path $target -Json '{"step":31}'
    $elapsed = ([DateTime]::UtcNow - $started).TotalMilliseconds
    $actual = Get-Content -LiteralPath $target -Raw -ErrorAction Stop | ConvertFrom-Json
    if ($actual.step -ne 31) { throw 'Checkpoint did not recover after lock release' }
    if ($elapsed -lt 200) { throw ('Lock test did not exercise retry: elapsed_ms='+$elapsed) }
    $receipt.locked_write_retried=$true
    $receipt.lock_elapsed_ms=[math]::Round($elapsed)
    for ($i=32; $i -le 40; $i++) {
        Write-FlowDevCheckpoint -Path $target -Json ('{"step":'+$i+'}')
        $actual = Get-Content -LiteralPath $target -Raw | ConvertFrom-Json
        if ($actual.step -ne $i) { throw ('Post-lock update invalid: '+$i) }
        $receipt.attempts++
    }
    if (@(Get-ChildItem $root -Filter 'active-step*.tmp' -File -ErrorAction SilentlyContinue).Count -ne 0) {
        throw 'Stale checkpoint temp files left behind'
    }
    $receipt.passed=$true
} catch {
    $receipt.error=$_.Exception.Message
    Write-Output ('CHECKPOINT_FAIL='+$receipt.error)
} finally {
    if ($job) {
        Wait-Job -Job $job -Timeout 8 | Out-Null
        if ($job.State -eq 'Running') { Stop-Job -Job $job -ErrorAction SilentlyContinue }
        $jobResult = @(Receive-Job -Job $job -ErrorAction SilentlyContinue)
        if ($jobResult.Count) { $receipt.locker_output=$jobResult }
        Remove-Job -Job $job -Force -ErrorAction SilentlyContinue
    }
    $receipt.finished_utc=[DateTime]::UtcNow.ToString('o')
    [IO.File]::WriteAllText($receiptPath, ($receipt | ConvertTo-Json -Depth 6), (New-Object Text.UTF8Encoding($false)))
    Write-Output ('CHECKPOINT_RECEIPT='+$receiptPath)
}
if (-not $receipt.passed) { exit 1 }
Write-Output ('CHECKPOINT_PASS=40 writes including held-target retry')
exit 0
