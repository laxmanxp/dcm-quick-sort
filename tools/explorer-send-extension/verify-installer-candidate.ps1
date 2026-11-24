# Verify a staged Explorer submenu package before any installer integration.
# Read-only: no registry changes, installs, Explorer restarts, or file moves.
[CmdletBinding()]
param(
 [Parameter(Mandatory=$true)][string]$PackageDir,
 [switch]$RequestActivation
)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
function Assert-Package {
 param([string]$Directory,[bool]$Activate)
 $dir=[IO.Path]::GetFullPath($Directory)
 $manifestPath=Join-Path $dir 'manifest.json'
 if(-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)){throw 'Missing manifest.json'}
 $manifest=Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
 if($manifest.schema -ne 1 -or $manifest.component -cne 'DCM QuickSort Explorer Send' -or $manifest.passed -ne $true) {
  throw 'Invalid or unsuccessful staging manifest'
 }
 if($manifest.enabled_by_default -ne $false -or $manifest.installation_performed -ne $false -or
    $manifest.registry_modified -ne $false -or $manifest.explorer_restarted -ne $false) {
  throw 'Staging manifest violates the default-off, no-side-effects contract'
 }
 $expected=@('DCMQuickSortExplorerSend.dll','DCMQuickSortExplorerSend.exe')
 if(@($manifest.files).Count -ne $expected.Count){throw 'Unexpected staged file count'}
 $seen=@{}
 foreach($entry in @($manifest.files)){
  $name=[string]$entry.name
  if($expected -cnotcontains $name -or $seen.ContainsKey($name)){throw 'Unexpected or duplicate file in manifest'}
  $seen[$name]=$true
  $path=Join-Path $dir ('payload\'+$name)
  if(-not (Test-Path -LiteralPath $path -PathType Leaf)){throw ('Missing staged component: '+$name)}
  $item=Get-Item -LiteralPath $path
  if($item.Length -ne [int64]$entry.length -or $item.Length -le 0){throw ('Size mismatch: '+$name)}
  $digest=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
  if($digest -cne ([string]$entry.sha256).ToLowerInvariant()){throw ('Hash mismatch: '+$name)}
 }
 if($Activate){
  if($manifest.gui_accepted -ne $true -or $manifest.installer_qualified -ne $true){
   throw 'Activation refused: real Explorer GUI and installer acceptance are not complete'
  }
  throw 'Activation refused: this preflight never performs an installation'
 }
 return [pscustomobject]@{Passed=$true;Components=2;ActivationAllowed=$false;SourceCommit=$manifest.source_commit}
}
Assert-Package -Directory $PackageDir -Activate ([bool]$RequestActivation)
