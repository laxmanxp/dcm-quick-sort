# Smallest contract tests: blank, malformed, and valid event streams.
$ErrorActionPreference='Stop';Set-StrictMode -Version Latest
. (Join-Path $PSScriptRoot '../tests/native/dcm-pdf-inflight.ps1')
function New-Event([string]$kind,[long]$seq,[long]$gen=5){
 return [pscustomobject]@{schema=1;host_pid=99;sequence=$seq;event=$kind;generation=$gen;renderer_pid=123;input_path='synthetic.pdf';output_path='synthetic.png';failed=$false;state=[pscustomobject]@{active=$false;busy=$false;pending=$false;worker_generation=0;generation=6;path=''}}
}
function New-Valid {return @((New-Event 'worker-started' 1),(New-Event 'selection-stopped' 2),(New-Event 'worker-finished' 3),(New-Event 'result-discarded' 4))}
$checks=0
function Reject($rows,[long]$new=0){$caught=$false;try{$null=Assert-DcmPdfInFlightTrace $rows 5 $false $new}catch{if(-not $_.Exception.Message.StartsWith('INFLIGHT_TRACE:')){throw};$caught=$true};if(-not $caught){throw 'Invalid lifecycle falsely accepted'};$script:checks++}
Reject @()
Reject @((New-Event 'worker-started' 1))
$r=New-Valid;$r[1].sequence=3;$r[2].sequence=2;Reject @($r|Sort-Object sequence)
$r=New-Valid;$r[3].host_pid=101;Reject $r
$r=New-Valid;$r[3].state.active=$true;Reject $r
$r=New-Valid;$r+=New-Event 'result-applied' 5;Reject $r
$r=New-Valid;$r[2].output_path='other.png';Reject $r
$null=Assert-DcmPdfInFlightTrace (New-Valid) 5 $false;$checks++
$r=New-Valid;$r[2].failed=$true;$r[3].failed=$true;$null=Assert-DcmPdfInFlightTrace $r 5 $true;$checks++
$r=@((New-Event 'worker-started' 1),(New-Event 'selection-stopped' 2),(New-Event 'worker-started' 3 7),(New-Event 'worker-finished' 4),(New-Event 'result-discarded' 5),(New-Event 'worker-finished' 6 7),(New-Event 'result-applied' 7 7))
$r[4].state=[pscustomobject]@{active=$true;busy=$true;pending=$false;worker_generation=7;generation=7;path='synthetic.pdf'}
$null=Assert-DcmPdfInFlightTrace $r 5 $false 7;$checks++
$r[4].state.busy=$false;Reject $r 7
# Exercise the same observer against an owned, windowless control process.
$psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName='powershell.exe';$psi.Arguments='-NoProfile -NonInteractive -Command "Start-Sleep -Milliseconds 600; exit 7"';$psi.UseShellExecute=$false;$psi.CreateNoWindow=$true
$child=[Diagnostics.Process]::Start($psi);$observer=Get-Process -Id $child.Id
try{$null=Keep-DcmPdfProcessHandle $observer;if(-not $observer.WaitForExit(5000) -or $null -eq $observer.ExitCode -or $observer.ExitCode -ne 7){throw 'Retained observer exit-status regression'};$checks++}finally{$observer.Dispose();$child.Dispose()}
if($checks -ne 12){throw 'Incomplete in-flight contract tests'}
Write-Output 'PASS: 11 trace contracts and 1 owned process observer check (no product launched)'
