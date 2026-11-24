# Destructive tests use only a disposable copy of the staged package.
[CmdletBinding()]
param([Parameter(Mandatory=$true)][string]$PackageDir)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
$verifier=Join-Path $PSScriptRoot 'verify-installer-candidate.ps1'
$root=Join-Path $env:TEMP ('dcm-explorer-preflight-test-'+[Guid]::NewGuid().ToString('N'))
$cases=0
function Expect-Reject([scriptblock]$Action,[string]$Label){
 $rejected=$false
 try { $null=& $Action } catch { $rejected=$true }
 if(-not $rejected){throw ('FAIL: '+$Label)}
 $script:cases++
 Write-Output ('PASS: '+$Label)
}
try {
 $valid=& $verifier -PackageDir $PackageDir
 if(-not $valid.Passed -or $valid.Components -ne 2 -or $valid.ActivationAllowed){throw 'Valid package preflight failed'}
 $cases++;Write-Output 'PASS: valid package checked without activation'
 Expect-Reject { & $verifier -PackageDir $PackageDir -RequestActivation } 'activation request refused without Explorer GUI qualification'
 New-Item -ItemType Directory -Path $root | Out-Null
 Copy-Item -LiteralPath (Join-Path $PackageDir 'manifest.json') -Destination (Join-Path $root 'manifest.json')
 Copy-Item -LiteralPath (Join-Path $PackageDir 'payload') -Destination (Join-Path $root 'payload') -Recurse
 $dll=Join-Path $root 'payload\DCMQuickSortExplorerSend.dll'
 $stream=[IO.File]::Open($dll,[IO.FileMode]::Open,[IO.FileAccess]::ReadWrite,[IO.FileShare]::None)
 try {
  $stream.Position=0
  $first=$stream.ReadByte()
  $stream.Position=0
  $stream.WriteByte([byte]($first -bxor 1))
 } finally { $stream.Dispose() }
 Expect-Reject { & $verifier -PackageDir $root } 'one-byte DLL corruption rejected'
 Copy-Item -LiteralPath (Join-Path $PackageDir 'payload\DCMQuickSortExplorerSend.dll') -Destination $dll -Force
 Remove-Item -LiteralPath (Join-Path $root 'payload\DCMQuickSortExplorerSend.exe')
 Expect-Reject { & $verifier -PackageDir $root } 'missing helper rejected'
 Copy-Item -LiteralPath (Join-Path $PackageDir 'payload\DCMQuickSortExplorerSend.exe') -Destination (Join-Path $root 'payload\DCMQuickSortExplorerSend.exe')
 $manifest=Get-Content -LiteralPath (Join-Path $root 'manifest.json') -Raw | ConvertFrom-Json
 $manifest.enabled_by_default=$true
 $manifest | ConvertTo-Json -Depth 7 | Set-Content -LiteralPath (Join-Path $root 'manifest.json')
 Expect-Reject { & $verifier -PackageDir $root } 'unexpected default-on manifest rejected'
 Write-Output ('PASS: '+$cases+' package preflight cases')
} finally {
 if(Test-Path -LiteralPath $root){Remove-Item -LiteralPath $root -Recurse -Force}
}
