<#
.SYNOPSIS
Build and stage the QuickSort Explorer submenu as an opt-in installer candidate.
This is NOT an installer and never registers COM, copies to the live install
directory, restarts explorer.exe, or moves user files.
#>
[CmdletBinding()]
param(
 [Parameter(Mandatory=$true)][string]$OutputDir
)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
$source=(Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$destination=[IO.Path]::GetFullPath($OutputDir)
$sourcePrefix=$source.TrimEnd([IO.Path]::DirectorySeparatorChar)+[IO.Path]::DirectorySeparatorChar
if($destination -ieq $source -or $destination.StartsWith($sourcePrefix,[StringComparison]::OrdinalIgnoreCase)){
 throw 'Staging must be outside the source checkout.'
}
$live=[IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs\DCM Quick Sort'))
if($destination -ieq $live -or $destination.StartsWith($live+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){
 throw 'Staging cannot write into the live installed application.'
}
if(Test-Path -LiteralPath $destination){throw 'Use a fresh output directory to avoid overwriting prior evidence.'}
New-Item -Path $destination -ItemType Directory | Out-Null
$buildDir=Join-Path $destination 'payload'
$receiptPath=Join-Path $destination 'manifest.json'
$receipt=[ordered]@{
 schema=1
 component='DCM QuickSort Explorer Send'
 source_commit=(& git -C $source rev-parse HEAD | Out-String).Trim()
 generated_utc=[DateTime]::UtcNow.ToString('o')
 enabled_by_default=$false
 installation_performed=$false
 registry_modified=$false
 explorer_restarted=$false
 gui_accepted=$false
 installer_qualified=$false
 files=@()
 passed=$false
 error=$null
}
try {
 $null=& (Join-Path $PSScriptRoot 'build.ps1') -OutDir $buildDir
 if($LASTEXITCODE -ne 0){throw 'Explorer component build failed.'}
 foreach($name in @('DCMQuickSortExplorerSend.dll','DCMQuickSortExplorerSend.exe')){
  $path=Join-Path $buildDir $name
  if(-not (Test-Path -LiteralPath $path -PathType Leaf)){throw ('Missing payload: '+$name)}
  $item=Get-Item -LiteralPath $path
  if($item.Length -le 0){throw ('Empty payload: '+$name)}
  $receipt.files+= [ordered]@{
    name=$name
    length=$item.Length
    sha256=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
  }
 }
 $receipt.passed=$true
} catch {
 $receipt.error=$_.Exception.Message
} finally {
 $receipt | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $receiptPath -Encoding UTF8
 Write-Output ('MANIFEST='+$receiptPath)
 Write-Output ('STAGING_PASSED='+$receipt.passed)
 Write-Output 'No installer activation, COM registration, or user-file moves were performed.'
}
if(-not $receipt.passed){throw $receipt.error}
