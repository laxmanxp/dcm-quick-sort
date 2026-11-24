# Pure ownership decisions for the QuickSort per-user Explorer COM registration.
# No Windows registry reads or writes take place in this file.
function Get-QuickSortRegistrationDecision {
 [CmdletBinding()]
 param(
  [AllowNull()][string]$HandlerValue,
  [AllowNull()][string]$ServerValue,
  [Parameter(Mandatory=$true)][string]$ExpectedClsid,
  [Parameter(Mandatory=$true)][string]$ExpectedDllPath
 )
 $expected=[IO.Path]::GetFullPath($ExpectedDllPath)
 if($HandlerValue -and $HandlerValue -ine $ExpectedClsid) {
  throw 'Explorer handler key is owned by another COM class; refusing changes.'
 }
 if($ServerValue) {
  $actual=[IO.Path]::GetFullPath($ServerValue)
  if($actual -ine $expected) {
   throw 'COM server does not match the exact expected QuickSort installation path; refusing changes.'
  }
 }
 [pscustomobject]@{
  HandlerOwned = [bool]($HandlerValue -and $HandlerValue -ieq $ExpectedClsid)
  ServerOwned = [bool]($ServerValue -and ([IO.Path]::GetFullPath($ServerValue)) -ieq $expected)
 }
}
