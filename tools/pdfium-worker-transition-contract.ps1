# Source-controlled, offline-testable contract for the PDFium GUI trace.
# An in-flight obsolete render may terminate its owned worker deliberately.
# Every observed PID transition must have a matching request and completion,
# with the same generation and old worker PID, before the new worker starts.
function Assert-PDFiumWorkerTransitions {
 [CmdletBinding()]
 param([Parameter(Mandatory=$true)][object[]]$Events)
 $starts=@($Events | Where-Object {$_.event -eq 'worker-started' -and [int]$_.renderer_pid -gt 0} | Sort-Object {[long]$_.sequence})
 if(-not $starts.Count){throw 'No PDFium persistent worker starts recorded'}
 $previous=$starts[0]
 $restarts=0
 for($i=1;$i -lt $starts.Count;$i++){
  $current=$starts[$i]
  if([int]$current.renderer_pid -eq [int]$previous.renderer_pid) {
   $previous=$current
   continue
  }
  $prior=[int]$previous.renderer_pid
  $next=[int]$current.renderer_pid
  $boundary=[long]$current.sequence
  $requested=@($Events | Where-Object {
   $_.event -eq 'worker-cancel-requested' -and
   [long]$_.sequence -gt [long]$previous.sequence -and
   [long]$_.sequence -lt $boundary
  })
  $canceled=@($Events | Where-Object {
   $_.event -eq 'worker-canceled' -and [int]$_.renderer_pid -eq $prior -and
   [long]$_.sequence -gt [long]$previous.sequence -and
   [long]$_.sequence -lt $boundary
  })
  if($requested.Count -ne 1 -or $canceled.Count -ne 1 -or
     [long]$requested[0].generation -ne [long]$canceled[0].generation -or
     [long]$requested[0].sequence -ge [long]$canceled[0].sequence) {
   throw ('Unexpected PDFium worker restart: '+$prior+' -> '+$next+' without a matching ordered cancellation')
  }
  $restarts++
  $previous=$current
 }
 return @{cancel_restarts=$restarts;render_requests=$starts.Count;final_pid=[int]$starts[-1].renderer_pid}
}
