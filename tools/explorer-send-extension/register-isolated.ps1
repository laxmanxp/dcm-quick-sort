<#
.SYNOPSIS
Explicit, per-user registration of the DCM QuickSort classic Explorer submenu.
Does not perform an install by default and never restarts explorer.exe.

This script is for approved isolated GUI acceptance only until a production
installer has been qualified. It does not modify the existing "Sort with DCM"
shell command, user documents, or any other shell extension registration.
#>
[CmdletBinding(DefaultParameterSetName='Inspect')]
param(
 [Parameter(ParameterSetName='Register',Mandatory=$true)][switch]$Register,
 [Parameter(ParameterSetName='Unregister',Mandatory=$true)][switch]$Unregister,
 [Parameter(ParameterSetName='Register',Mandatory=$true)]
 [Parameter(ParameterSetName='Unregister',Mandatory=$true)]
 [switch]$ApprovedIsolatedAccount,
 [Parameter(ParameterSetName='Register',Mandatory=$true)][string]$DllPath,
 [Parameter(ParameterSetName='Register',Mandatory=$true)][string]$HelperPath,
 [Parameter(ParameterSetName='Register',Mandatory=$true)][string]$AppPath
)
$ErrorActionPreference='Stop'
$clsid='{8F2DA0C1-7D79-40AA-A030-19645D8C4A91}'
$classes='HKCU:\Software\Classes'
$classKey="$classes\CLSID\$clsid"
$handler="$classes\*\shellex\ContextMenuHandlers\DCMQuickSortSend"
$expectedName='DCMQuickSortExplorerSend.dll'
. (Join-Path $PSScriptRoot 'registration-ownership.ps1')
if(-not $ApprovedIsolatedAccount) {
 Write-Output 'Not registered. Supply -Register or -Unregister and -ApprovedIsolatedAccount in a disposable Windows test profile.'
 return
}
function Get-DefaultValue([string]$Path) {
 $key=Get-Item -LiteralPath $Path -ErrorAction SilentlyContinue
 if($null -eq $key){return $null}
 return $key.GetValue('')
}
if($Register){
 foreach($entry in @(@($DllPath,$expectedName),@($HelperPath,'DCMQuickSortExplorerSend.exe'),@($AppPath,'DCM-QuickSort.exe'))){
  $p=[IO.Path]::GetFullPath($entry[0])
  if(-not [IO.Path]::IsPathRooted($p) -or -not (Test-Path -LiteralPath $p -PathType Leaf) -or [IO.Path]::GetFileName($p) -ine $entry[1]){
   throw "Required installed artifact is missing or not named correctly: $p"
  }
 }
 $appDir=[IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($AppPath))
 $expectedInstall=[IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs\DCM Quick Sort'))
 if($appDir -ine $expectedInstall){throw 'This DLL only resolves helper and app from the standard per-user installation directory.'}
 if([IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($HelperPath)) -ine $appDir -or
    [IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($DllPath)) -ine $appDir) {
   throw 'The DLL, helper and app must be in the same installed QuickSort directory.'
 }
 $existing=Get-DefaultValue "$classKey\InprocServer32"
 $existingHandler=Get-DefaultValue $handler
 $null=Get-QuickSortRegistrationDecision -HandlerValue $existingHandler -ServerValue $existing -ExpectedClsid $clsid -ExpectedDllPath $DllPath
 New-Item -Path "$classKey\InprocServer32" -Force | Out-Null
 Set-Item -Path "$classKey\InprocServer32" -Value ([IO.Path]::GetFullPath($DllPath))
 New-ItemProperty -LiteralPath "$classKey\InprocServer32" -Name 'ThreadingModel' -PropertyType String -Value 'Apartment' -Force | Out-Null
 New-Item -Path $handler -Force | Out-Null
 Set-Item -Path $handler -Value $clsid
 Write-Output 'Registered DCM QuickSort Send submenu for the current isolated Windows user.'
 Write-Output 'On Windows 11, check Explorer > Show more options.'
} elseif($Unregister){
 $expectedInstall=[IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs\DCM Quick Sort'))
 $expectedDll=Join-Path $expectedInstall $expectedName
 $value=Get-DefaultValue $handler
 $registered=Get-DefaultValue "$classKey\InprocServer32"
 # Validate both keys before any mutation, and require an exact path match.
 $decision=Get-QuickSortRegistrationDecision -HandlerValue $value -ServerValue $registered -ExpectedClsid $clsid -ExpectedDllPath $expectedDll
 if($decision.HandlerOwned){Remove-Item -LiteralPath $handler -Recurse -Force}
 if($decision.ServerOwned){Remove-Item -LiteralPath $classKey -Recurse -Force}
 Write-Output 'Unregistered only owned QuickSort Explorer keys for the current isolated user.'
} else {
 Write-Output "QuickSort Send CLSID: $clsid"
 Write-Output "Handler: $handler"
 Write-Output 'No registry changes performed.'
}
