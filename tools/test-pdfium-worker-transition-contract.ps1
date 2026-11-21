# Run with: powershell -NoProfile -ExecutionPolicy Bypass -File tools/test-pdfium-worker-transition-contract.ps1
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'pdfium-worker-transition-contract.ps1')
function Event([int]$sequence,[string]$name,[int]$rendererId=0,[int]$generation=0) {
 return [pscustomobject]@{sequence=$sequence;event=$name;renderer_pid=$rendererId;generation=$generation}
}
function MustPass([string]$name,[object[]]$events,[int]$expected) {
 $result=Assert-PDFiumWorkerTransitions -Events $events
 if($result.cancel_restarts -ne $expected){throw ('FAIL '+$name+': expected '+$expected+' got '+$result.cancel_restarts)}
 Write-Output ('PASS '+$name)
}
function MustFail([string]$name,[object[]]$events) {
 try {
  $null=Assert-PDFiumWorkerTransitions -Events $events
 } catch {
  if($_.Exception.Message -notmatch 'Unexpected PDFium worker restart'){throw}
  Write-Output ('PASS '+$name+' (rejected)')
  return
 }
 throw ('FAIL '+$name+': unexpected restart accepted')
}
$healthy=@((Event 1 'worker-started' 100 1),(Event 2 'worker-finished' 100 1),(Event 3 'worker-started' 100 2))
$valid=@((Event 1 'worker-started' 100 1),(Event 2 'worker-cancel-requested' 0 1),(Event 3 'worker-canceled' 100 1),(Event 4 'worker-started' 200 3))
MustPass 'healthy reuse' $healthy 0
MustPass 'canceled child replacement' $valid 1
MustFail 'unexplained restart' @((Event 1 'worker-started' 100 1),(Event 2 'worker-started' 200 2))
MustFail 'cancel request without completion' @((Event 1 'worker-started' 100 1),(Event 2 'worker-cancel-requested' 0 1),(Event 3 'worker-started' 200 3))
MustFail 'cancellation for wrong PID' @((Event 1 'worker-started' 100 1),(Event 2 'worker-cancel-requested' 0 1),(Event 3 'worker-canceled' 900 1),(Event 4 'worker-started' 200 3))
MustFail 'mismatched cancellation generation' @((Event 1 'worker-started' 100 1),(Event 2 'worker-cancel-requested' 0 1),(Event 3 'worker-canceled' 100 2),(Event 4 'worker-started' 200 3))
MustFail 'reversed cancellation order' @((Event 1 'worker-started' 100 1),(Event 2 'worker-canceled' 100 1),(Event 3 'worker-cancel-requested' 0 1),(Event 4 'worker-started' 200 3))
Write-Output 'PDFIUM_TRANSITION_CONTRACT_PASS'
