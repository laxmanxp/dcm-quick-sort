# Pure contract tests: these MUST NOT access or alter the Windows registry.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
. (Join-Path $PSScriptRoot 'registration-ownership.ps1')
$clsid='{8F2DA0C1-7D79-40AA-A030-19645D8C4A91}'
$expected='C:\Users\fixture\AppData\Local\Programs\DCM Quick Sort\DCMQuickSortExplorerSend.dll'
$checks=0
function Assert-Case([bool]$Condition,[string]$Name) {
 if(-not $Condition){throw ('FAIL: '+$Name)}
 $script:checks++
 Write-Output ('PASS: '+$Name)
}
function Assert-Refused([string]$Handler,[string]$Server,[string]$Name) {
 $refused=$false
 try {
  $null=Get-QuickSortRegistrationDecision -HandlerValue $Handler -ServerValue $Server -ExpectedClsid $clsid -ExpectedDllPath $expected
 } catch { $refused=$true }
 Assert-Case $refused $Name
}
$d=Get-QuickSortRegistrationDecision -HandlerValue $null -ServerValue $null -ExpectedClsid $clsid -ExpectedDllPath $expected
Assert-Case (-not $d.HandlerOwned -and -not $d.ServerOwned) 'absent registration is a no-op'
$d=Get-QuickSortRegistrationDecision -HandlerValue $clsid -ServerValue $expected -ExpectedClsid $clsid -ExpectedDllPath $expected
Assert-Case ($d.HandlerOwned -and $d.ServerOwned) 'exact owned registration can be removed'
$d=Get-QuickSortRegistrationDecision -HandlerValue $clsid -ServerValue $null -ExpectedClsid $clsid -ExpectedDllPath $expected
Assert-Case ($d.HandlerOwned -and -not $d.ServerOwned) 'orphan owned handler is removable'
$d=Get-QuickSortRegistrationDecision -HandlerValue $null -ServerValue $expected -ExpectedClsid $clsid -ExpectedDllPath $expected
Assert-Case (-not $d.HandlerOwned -and $d.ServerOwned) 'orphan owned server is removable'
Assert-Refused '{00000000-0000-0000-0000-000000000000}' $expected 'foreign handler rejected'
Assert-Refused $clsid 'C:\OtherApp\DCMQuickSortExplorerSend.dll' 'same DLL filename at foreign path rejected'
Assert-Refused '{00000000-0000-0000-0000-000000000000}' 'C:\OtherApp\DCMQuickSortExplorerSend.dll' 'both foreign keys rejected atomically'
Assert-Refused $null 'C:\OtherApp\DCMQuickSortExplorerSend.dll' 'orphan foreign COM server rejected'
Assert-Refused '{00000000-0000-0000-0000-000000000000}' $null 'orphan foreign handler rejected'
Write-Output ('PASS: '+$checks+' pure registration ownership cases, no registry writes')
