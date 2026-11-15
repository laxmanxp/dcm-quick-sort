# Pure event contract used by the real-app harness and the tooling gate.
# No timing sleeps, process suspension, injected completions or fake renderer.
function Assert-DcmPdfInFlightTrace($Events,[long]$Generation,[bool]$Failed,[long]$NewGeneration=0){
 $rows=@($Events);if(-not $rows.Count){throw 'INFLIGHT_TRACE: empty evidence'}
 $last=[long]0;$hostId=[long]$rows[0].host_pid
 foreach($e in $rows){if($e.schema -ne 1 -or $e.host_pid -ne $hostId -or $e.sequence -le $last){throw 'INFLIGHT_TRACE: invalid stream identity/order'};$last=[long]$e.sequence}
 $old=@($rows|Where-Object {$_.generation -eq $Generation})
 $start=@($old|Where-Object {$_.event -ceq 'worker-started'});$finish=@($old|Where-Object {$_.event -ceq 'worker-finished'});$discard=@($old|Where-Object {$_.event -ceq 'result-discarded'});$stop=@($old|Where-Object {$_.event -ceq 'selection-stopped'})
 if($start.Count -ne 1 -or $finish.Count -ne 1 -or $discard.Count -ne 1 -or $stop.Count -ne 1){throw 'INFLIGHT_TRACE: missing/duplicate lifecycle event'}
 $s=$start[0];$f=$finish[0];$d=$discard[0];$x=$stop[0]
 if($s.renderer_pid -le 0 -or $f.renderer_pid -ne $s.renderer_pid -or $f.input_path -cne $s.input_path -or $d.input_path -cne $s.input_path -or $f.output_path -cne $s.output_path -or $d.output_path -cne $s.output_path){throw 'INFLIGHT_TRACE: request or child identity changed'}
 if(-not ($s.sequence -lt $x.sequence -and $x.sequence -lt $f.sequence -and $f.sequence -lt $d.sequence)){throw 'INFLIGHT_TRACE: renderer was not observed spanning the selection change'}
 if($f.failed -ne $Failed -or $d.failed -ne $Failed){throw 'INFLIGHT_TRACE: unexpected renderer outcome'}
 if(@($old|Where-Object {$_.event -cin @('result-applied','result-unavailable')}).Count){throw 'INFLIGHT_TRACE: obsolete result reached presenter'}
 if($NewGeneration -gt 0){
  $n=@($rows|Where-Object {$_.generation -eq $NewGeneration -and $_.event -ceq 'worker-started'});$a=@($rows|Where-Object {$_.generation -eq $NewGeneration -and $_.event -ceq 'result-applied'})
  if($n.Count -ne 1 -or $a.Count -ne 1 -or $n[0].sequence -ge $f.sequence -or $a[0].sequence -le $d.sequence -or $n[0].input_path -cne $s.input_path){throw 'INFLIGHT_TRACE: same-file replacement overlap not proved'}
  if(-not $d.state.active -or -not $d.state.busy -or $d.state.pending -or $d.state.worker_generation -ne $NewGeneration -or $d.state.generation -ne $NewGeneration -or $d.state.path -cne $s.input_path){throw 'INFLIGHT_TRACE: stale completion altered newer worker'}
 }else{
  if($d.state.active -or $d.state.busy -or $d.state.pending -or $d.state.worker_generation -ne 0 -or $d.state.path -cne ''){throw 'INFLIGHT_TRACE: stale completion altered stopped PDF state'}
 }
 return [pscustomobject]@{generation=$Generation;renderer_pid=$s.renderer_pid;input_path=$s.input_path;output_path=$s.output_path;started_sequence=$s.sequence;stopped_sequence=$x.sequence;finished_sequence=$f.sequence;discarded_sequence=$d.sequence;replacement_generation=$NewGeneration;failed=$Failed;passed=$true}
}

function Read-DcmPdfTrace {
 if(-not(Test-Path -LiteralPath $tracePath)){return @()}
 $stream=[IO.File]::Open($tracePath,[IO.FileMode]::Open,[IO.FileAccess]::Read,([IO.FileShare]::ReadWrite -bor [IO.FileShare]::Delete))
 $reader=New-Object IO.StreamReader($stream,[Text.Encoding]::UTF8)
 try{$text=$reader.ReadToEnd()}finally{$reader.Dispose()}
 $end=$text.LastIndexOf("`n");if($end -lt 0){return @()}
 $rows=@();foreach($line in $text.Substring(0,$end).Split("`n")){if($line.Trim()){$e=$line|ConvertFrom-Json;if($e.host_pid -ne $app.Id){throw 'Trace host does not match owned app'};$rows+=,$e}}
 return $rows
}
function Wait-DcmPdfStarted([string]$Name,[long]$After){
 $until=[DateTime]::UtcNow.AddSeconds(5)
 do{OwnProcess;$rows=@(Read-DcmPdfTrace);$starts=@($rows|Where-Object {$_.event -ceq 'worker-started' -and $_.sequence -gt $After -and $_.input_path -ceq (Join-Path $fixture.inbox $Name)})
  if($starts.Count -gt 1){throw 'Multiple unexpected render starts'}
  if($starts.Count -eq 1){return $starts[0]};Start-Sleep -Milliseconds 10
 }while([DateTime]::UtcNow -lt $until)
 throw 'Real renderer process start not observed'
}
function Invoke-DcmPdfInFlightSwitch([string]$Name,[string]$Tag,[bool]$Failed,[switch]$Reopen){
 $prior=@(Read-DcmPdfTrace);$after=if($prior.Count){[long]$prior[-1].sequence}else{0}
 SelectFile $Name;$start=Wait-DcmPdfStarted $Name $after
 $p=Get-Process -Id $start.renderer_pid -ErrorAction Stop
 try{
  $p.Refresh();if($p.HasExited -or [IO.Path]::GetFileName($p.Path) -cnotin @('powershell.exe','pwsh.exe') -or $p.StartTime.ToUniversalTime() -lt $app.StartTime.ToUniversalTime()){throw 'Renderer no longer live before switch'}
  # Read the owned footer before switching, not merely a sleep-based estimate.
  $before=HostCaptions;if(-not @($before|Where-Object {$_.StartsWith('Rendering PDF page ',[StringComparison]::Ordinal)}).Count){throw 'Expected actual rendering footer before switch'}
  $observed=[ordered]@{pid=$p.Id;path=$p.Path;start=$p.StartTime.ToUniversalTime().ToString('o');observed_running_utc=[DateTime]::UtcNow.ToString('o');host_captions=$before;request=$start}
  SelectFile '01-alpha.txt';$p.Refresh();$observed['alive_after_text_switch']=-not $p.HasExited
  if(-not $observed.alive_after_text_switch){throw 'Renderer already exited at switch; no in-flight claim'}
  $new=$null
  if($Reopen){SelectFile $Name;$new=Wait-DcmPdfStarted $Name $start.sequence;$p.Refresh();if($p.HasExited){throw 'Old renderer exited before reopened request overlap'}}
  $until=[DateTime]::UtcNow.AddSeconds(25);$ready=$false
  do{NoDialog;$rows=@(Read-DcmPdfTrace);$discard=@($rows|Where-Object {$_.generation -eq $start.generation -and $_.event -ceq 'result-discarded'});$ready=$discard.Count -eq 1
   if($new){$ready=$ready -and @($rows|Where-Object {$_.generation -eq $new.generation -and $_.event -ceq 'result-applied'}).Count -eq 1}
   if($ready){break};Start-Sleep -Milliseconds 20
  }while([DateTime]::UtcNow -lt $until)
  if(-not $ready){throw 'Late completion was not consumed within deadline'}
  $newGen=if($new){[long]$new.generation}else{0};$proof=Assert-DcmPdfInFlightTrace $rows $start.generation $Failed $newGen
  if(-not $p.WaitForExit(1000) -or $p.ExitCode -ne $(if($Failed){1}else{0})){throw 'Renderer did not exit with expected status'}
  if(Test-Path -LiteralPath $start.output_path){throw 'Obsolete output remains after discarded result'}
  $observed['exit_code']=$p.ExitCode;$observed['proof']=$proof;$observed['events']=$rows;$observed['renderer_delay_or_suspension_used']=$false
  Json (Join-Path $lab ($Tag+'.events.json')) $observed
  Record $Tag ([ordered]@{proof=$proof;observation=(Join-Path $lab ($Tag+'.events.json'));real_process_observed=$true;owned_selection_input=$true;obsolete_output_removed=$true;no_synthetic_completion=$true})
 }finally{$p.Dispose()}
}
