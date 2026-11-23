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
if(-not $ApprovedIsolatedAccount) {
 Write-Output 'Not registered. Supply -Register or -Unregister and -ApprovedIsolatedAccount in a disposable Windows test profile.'
 return
}
if($Register){
 foreach($entry in @(@($DllPath,$expectedName),@($HelperPath,'DCMQuickSortExplorerSend.exe'),@($AppPath,'DCM-QuickSort.exe'))){
  $p=[IO.Path]::GetFullPath($entry[0])
  if(-not [IO.Path]::IsPathRooted($p) -or -not (Test-Path -LiteralPath $p -PathType Leaf) -or [IO.Path]::GetFileName($p) -ine $entry[1]){
   throw "Required installed artifact is missing or not named correctly: $p"
  }
 }
 $appDir=[IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($AppPath))
 if([IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($HelperPath)) -ine $appDir -or
    [IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($DllPath)) -ine $appDir) {
   throw 'The DLL, helper and app must be in the same installed QuickSort directory.'
 }
 $existing=(Get-ItemProperty -LiteralPath "$classKey\InprocServer32" -ErrorAction SilentlyContinue).'(default)'
 if($existing -and $existing -ine ([IO.Path]::GetFullPath($DllPath))){throw 'CLSID is already registered to a different path'}
 $existingHandler=(Get-ItemProperty -LiteralPath $handler -ErrorAction SilentlyContinue).'(default)'
 if($existingHandler -and $existingHandler -ine $clsid){throw 'ContextMenuHandlers key belongs to a different COM class'}
 New-Item -Path "$classKey\InprocServer32" -Force | Out-Null
 Set-Item -Path "$classKey\InprocServer32" -Value ([IO.Path]::GetFullPath($DllPath))
 New-ItemProperty -LiteralPath "$classKey\InprocServer32" -Name 'ThreadingModel' -PropertyType String -Value 'Apartment' -Force | Out-Null
 New-Item -Path $handler -Force | Out-Null
 Set-Item -Path $handler -Value $clsid
 Write-Output 'Registered DCM QuickSort Send submenu for the current isolated Windows user.'
 Write-Output 'On Windows 11, check Explorer > Show more options.'
} elseif($Unregister){
 $value=(Get-ItemProperty -LiteralPath $handler -ErrorAction SilentlyContinue).'(default)'
 if($value -and $value -ine $clsid){throw "Refusing to alter unrelated Explorer handler: $value"}
 if($value){Remove-Item -LiteralPath $handler -Recurse -Force}
 $registered=(Get-ItemProperty -LiteralPath "$classKey\InprocServer32" -ErrorAction SilentlyContinue).'(default)'
 if($registered -and [IO.Path]::GetFileName($registered) -ine $expectedName){
  throw 'Refusing to unregister a different COM server'
 }
 if($registered){Remove-Item -LiteralPath $classKey -Recurse -Force}
 Write-Output 'Unregistered the current isolated user QuickSort Send submenu.'
} else {
 Write-Output "QuickSort Send CLSID: $clsid"
 Write-Output "Handler: $handler"
 Write-Output 'No registry changes performed.'
}
