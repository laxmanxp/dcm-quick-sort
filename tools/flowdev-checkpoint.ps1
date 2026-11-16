# Replace FlowDev checkpoints without exposing a partially written JSON file.
# Windows may briefly deny renaming/replacing a checkpoint opened by a reader,
# anti-virus scanner or desktop monitor. Retry bounded IO/share violations.
function Write-FlowDevCheckpoint {
    [CmdletBinding()]
    param([Parameter(Mandatory=$true)][string]$Path,
          [Parameter(Mandatory=$true)][string]$Json)

    $destination = [IO.Path]::GetFullPath($Path)
    $directory = [IO.Path]::GetDirectoryName($destination)
    if (-not [IO.Directory]::Exists($directory)) {
        throw ('FLOWDEV_CHECKPOINT_PARENT_MISSING: ' + $directory)
    }
    $base = [IO.Path]::GetFileName($destination)
    $nonce = [Guid]::NewGuid().ToString('N')
    $temporary = [IO.Path]::Combine($directory, '.' + $base + '.' + $nonce + '.tmp')
    $backup = [IO.Path]::Combine($directory, '.' + $base + '.' + $nonce + '.bak')
    $utf8 = New-Object Text.UTF8Encoding($false)
    $lastFailure = $null
    try {
        [IO.File]::WriteAllText($temporary, $Json, $utf8)
        for ($attempt=1; $attempt -le 50; $attempt++) {
            try {
                # File.Replace is atomic on the same Windows volume. Supply a
                # real backup pathname; PowerShell 5.1/.NET rejects null here.
                if (-not [IO.File]::Exists($temporary)) {
                    [IO.File]::WriteAllText($temporary, $Json, $utf8)
                }
                if ([IO.File]::Exists($destination)) {
                    [IO.File]::Replace($temporary, $destination, $backup, $true)
                } else {
                    [IO.File]::Move($temporary, $destination)
                }
                return $destination
            } catch {
                $problem = $_.Exception
                $inner = $problem.InnerException
                $transient = ($problem -is [IO.IOException]) -or ($problem -is [UnauthorizedAccessException]) -or
                             ($inner -is [IO.IOException]) -or ($inner -is [UnauthorizedAccessException])
                if (-not $transient) { throw }
                $lastFailure = $problem.Message
                if ($attempt -eq 50) { break }
                Start-Sleep -Milliseconds 100
            }
        }
        throw ('FLOWDEV_CHECKPOINT_LOCKED: bounded atomic replacement exhausted after 50 attempts: ' + $lastFailure)
    } finally {
        # Only this invocation's uniquely named sidecars are removed. Never
        # delete the previous valid checkpoint if a replacement fails.
        foreach ($sidecar in @($temporary, $backup)) {
            if ([IO.File]::Exists($sidecar)) {
                [IO.File]::Delete($sidecar)
            }
        }
    }
}
