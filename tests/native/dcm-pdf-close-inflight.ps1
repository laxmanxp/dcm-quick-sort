# DCM-34: real default Windows.Data.Pdf worker, normal window-close while active.
# Dot-sourced only by tools/test-ui-pdf-failure.ps1 -CloseInFlight.
# No worker delay, suspension, injected completion or forced success cleanup.
function Read-DcmPdfClosedTrace([long]$HostPID) {
 if (-not (Test-Path -LiteralPath $tracePath)) { throw 'Close trace file missing' }
 $fs = [IO.File]::Open($tracePath,[IO.FileMode]::Open,[IO.FileAccess]::Read,([IO.FileShare]::ReadWrite -bor [IO.FileShare]::Delete))
 $reader = New-Object IO.StreamReader($fs,[Text.Encoding]::UTF8)
 try { $data = $reader.ReadToEnd() } finally { $reader.Dispose() }
 if (-not $data.EndsWith("`n",[StringComparison]::Ordinal)) { throw 'Close trace final record incomplete' }
 $items = @()
 foreach ($line in $data.Split("`n")) {
  if (-not $line.Trim()) { continue }
  $item = $line | ConvertFrom-Json
  if ($item.schema -ne 1 -or $item.host_pid -ne $HostPID) { throw 'Unexpected close trace source identity' }
  $items += ,$item
 }
 $last = [long]0
 foreach ($item in $items) { if ([long]$item.sequence -le $last) { throw 'Close trace events out of order' }; $last = [long]$item.sequence }
 return $items
}
function Invoke-DcmPdfCloseInFlight {
 $prior = @(Read-DcmPdfTrace)
 $after = if ($prior.Count) { [long]$prior[-1].sequence } else { 0 }
 SelectFile '08-preview.pdf'
 $started = Wait-DcmPdfStarted '08-preview.pdf' $after
 $renderer = Get-Process -Id $started.renderer_pid -ErrorAction Stop
 try {
  $null = Keep-DcmPdfProcessHandle $renderer
  $renderer.Refresh()
  if ($renderer.HasExited -or [IO.Path]::GetFileName($renderer.Path) -cnotin @('powershell.exe','pwsh.exe') -or
      $renderer.StartTime.ToUniversalTime() -lt $app.StartTime.ToUniversalTime()) { throw 'Renderer not alive for close-during-render claim' }
  $captions = @(HostCaptions)
  if (-not @($captions | Where-Object { $_.StartsWith('Rendering PDF page ',[StringComparison]::Ordinal) }).Count) {
   throw 'Actual PDF rendering footer not visible before close'
  }
  $observation = [ordered]@{
   renderer_pid = $renderer.Id
   renderer_path = $renderer.Path
   renderer_started_utc = $renderer.StartTime.ToUniversalTime().ToString('o')
   host_pid = $owned.pid
   worker_generation = [long]$started.generation
   output_path = $started.output_path
   renderer_alive_before_close = $true
   close_requested_utc = [DateTime]::UtcNow.ToString('o')
   host_captions_at_close = $captions
   synthetic_delay_or_suspension_used = $false
   forced_worker_cancellation_used = $false
  }
  # Real WM_CLOSE on our uniquely identified disposable host. Never send to
  # a foreign HWND, alter Office providers or terminate another process.
  Own
  [DcmNative]::CloseAsync($hwnd,$app.Id)
  $owned['close_requested_utc'] = [DateTime]::UtcNow.ToString('o')
  $observation['close_posted_utc'] = $owned.close_requested_utc
  StopApp
  $observation['host_exit_code'] = $owned.exit_code
  $exited = $renderer.WaitForExit(10000)
  $observation['renderer_exited_after_close'] = $exited
  if (-not $exited) { throw 'Owned renderer still running after normal host shutdown' }
  $observation['renderer_exit_code'] = $renderer.ExitCode
  if ($renderer.ExitCode -ne 0) { throw 'Valid PDF renderer did not complete successfully' }
  $events = @(Read-DcmPdfClosedTrace ([long]$owned.pid))
  $filtered = @($events | Where-Object { $_.generation -eq $started.generation })
  $startEvents = @($filtered | Where-Object { $_.event -ceq 'worker-started' })
  $stopEvents = @($filtered | Where-Object { $_.event -ceq 'selection-stopped' })
  $finishEvents = @($filtered | Where-Object { $_.event -ceq 'worker-finished' })
  $discardEvents = @($filtered | Where-Object { $_.event -ceq 'result-discarded' })
  $observation['events'] = $filtered
  $observation['trace_event_names'] = @($filtered | ForEach-Object { $_.event })
  $observed = $startEvents.Count -eq 1 -and $stopEvents.Count -eq 1 -and
       $finishEvents.Count -eq 1 -and $discardEvents.Count -eq 1
  if (-not $observed) {
   Json (Join-Path $lab 'close-inflight.events.json') $observation
   throw 'Close did not consume completed renderer result before host teardown'
  }
  if (-not ($startEvents[0].sequence -lt $stopEvents[0].sequence -and
        $stopEvents[0].sequence -lt $finishEvents[0].sequence -and
        $finishEvents[0].sequence -lt $discardEvents[0].sequence)) {
   throw 'Close trace lacks ordered start > stop > finish > discard lifecycle'
  }
  if ($finishEvents[0].failed -or $discardEvents[0].failed -or
      $finishEvents[0].renderer_pid -ne $started.renderer_pid) { throw 'Close renderer trace has invalid outcome or identity' }
  if (@($filtered | Where-Object { $_.event -cin @('result-applied','result-unavailable') }).Count) {
   throw 'Closing host applied an obsolete preview'
  }
  if (Test-Path -LiteralPath $started.output_path) { throw 'Completed renderer cache PNG left after closing' }
  $cache = Join-Path $fixture.local 'DCM Quick Sort\preview-cache'
  $left = @(Get-ChildItem -LiteralPath $cache -Filter 'pdf-page-*.png' -File -ErrorAction SilentlyContinue)
  if ($left.Count) { throw 'Owned PDF preview cache contains leftover temporary images' }
  ExclusivePDFs
  $observation['discarded_before_host_exit'] = $true
  $observation['output_removed'] = $true
  $observation['renderer_naturally_exited'] = $true
  $observation['source_fixture_exclusively_readable'] = $true
  Json (Join-Path $lab 'close-inflight.events.json') $observation
  Record 'inflight-valid-PDF-close-drains-renderer' ([ordered]@{
   observation = (Join-Path $lab 'close-inflight.events.json')
   renderer_pid = $started.renderer_pid
   worker_generation = $started.generation
   renderer_naturally_exited = $true
   obsolete_completion_discarded = $true
   cache_removed = $true
   host_exit_code = $owned.exit_code
  })
 } finally { $renderer.Dispose() }
}
