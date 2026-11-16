# Windows FlowDev checkpoint persistence. This initial contract is intentionally
# the existing Move-Item implementation; the contention test should reject it.
function Write-FlowDevCheckpoint {
    [CmdletBinding()]
    param([Parameter(Mandatory=$true)][string]$Path,
          [Parameter(Mandatory=$true)][string]$Json)
    $temp = $Path + '.tmp'
    [IO.File]::WriteAllText($temp, $Json, (New-Object Text.UTF8Encoding($false)))
    Move-Item -LiteralPath $temp -Destination $Path -Force -ErrorAction Stop
}
