[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,[string]$Runtime=$env:DCM_SUMATRA_SOURCE_EXE)
# Measurement success is separate from the provisional 250ms queue-response budget.
$ErrorActionPreference='Stop';Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Windows x64 required'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt -or -not $Runtime){throw 'Workspace/build/runtime required'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-sumatra-startup-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Lab must be outside source'}
$null=New-Item -ItemType Directory -Path $lab;$utf8=New-Object Text.UTF8Encoding($false)
$measurements=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]';$viewers=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-SUMATRA-STARTUP-01';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;measurements=$measurements;captures=$shots;viewers=$viewers;responsiveness_budget_ms=250;cache_note='Fresh host/new viewer processes, OS cache not flushed; not a cold-boot benchmark.';scope='Measurement integrity and file safety; responsiveness_accepted is a separate result.'}
$app=$null;$monitor=$null;$hwnd=[IntPtr]::Zero;$head='';$manifest=@();$fixture=$null;$expected=@{};$runtimeExe='';$exe=''
function Hash([string]$p){return (Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Json([string]$p,$v){[IO.File]::WriteAllText($p,($v|ConvertTo-Json -Depth 30),$utf8)}
function Git([string[]]$a){$s=(& git.exe -C $repo @a|Out-String).Trim();if($LASTEXITCODE){throw 'Git query failed'};return $s}
function Source {
 if((Git @('rev-parse','HEAD')) -ne $head -or (Git @('status','--porcelain=v1','--untracked-files=all'))){throw 'Source changed'}
 foreach($f in $manifest){$p=[IO.Path]::GetFullPath((Join-Path $repo $f.path));if(-not $p.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $p) -ne $f.sha256){throw 'Manifest changed'}}
}
function Own {$app.Refresh();if($app.HasExited -or $app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $receipt.host_start){throw 'Host identity changed'};[DcmNative]::RequireOwned($hwnd,$app.Id)}
function Wait([string]$label,[scriptblock]$check){$until=[DateTime]::UtcNow.AddSeconds(20);do{Own;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected host dialog'};if(& $check){return};Start-Sleep -Milliseconds 25}while([DateTime]::UtcNow -lt $until);throw ('Deadline: '+$label)}
function PrivateProcesses {foreach($p in @(Get-Process -Name SumatraPDF -ErrorAction SilentlyContinue)){try{if($p.Path -eq $runtimeExe){$p}}catch{if(-not $p.HasExited){throw}}}}
function Files {
 $files=@(Get-ChildItem -LiteralPath $fixture.inbox -File);if($files.Count -ne $expected.Count){throw 'Fixture set changed'}
 foreach($f in $files){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Fixture bytes changed'}}
 if(@(Get-ChildItem -LiteralPath $fixture.destination -Force).Count){throw 'Destination changed'}
 $journal=Join-Path $fixture.config 'recovery\move-journal.jsonl';if((Test-Path $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Unexpected operation journal'}
}
function SelectFile([string]$file){$items=@([DcmNative]::ListItems($hwnd,$app.Id,1016));$index=[Array]::IndexOf($items,$file);if($index -lt 0){throw 'Fixture not listed'};[DcmUndoInput]::Select($hwnd,$app.Id,$index);return $index}
function TextReady {
 Wait 'text recovery' {([DcmUndoInput]::TryText($hwnd,$app.Id,1005)) -ceq '01-alpha' -and ($null -eq [DcmSumatraProbe]::Find($hwnd,$app.Id)) -and @((PrivateProcesses)).Count -eq 0}
 $p=[DcmTextProbe]::Preview($hwnd,$app.Id);if(-not $p -or $p.Text -cne $textExpected){throw 'Wrong text preview'}
 if(@(Get-ChildItem $lab -Directory -Filter 'dcm-sumatra-*').Count){throw 'Session profile retained'}
 $h=[IO.File]::Open((Join-Path $fixture.inbox '08-preview.pdf'),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()
}
Write-Output ('REPORT_ROOT='+$lab)
try {
 $build=Get-Content $BuildReceipt -Raw|ConvertFrom-Json;$head=Git @('rev-parse','HEAD')
 if(-not $build.passed -or $build.mode -ne 'build' -or -not $build.source_unchanged -or $build.commit -ne $head){throw 'Exact clean build required'}
 if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256 -or (Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'Build identity changed'}
 $manifest=Get-Content $build.source_hash_manifest.path -Raw|ConvertFrom-Json;Source
 $receipt['commit']=$head;$receipt['build_receipt']=$BuildReceipt;$receipt['app_sha256']=$build.app_artifact.sha256
 if((Hash $Runtime) -ne '719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0'){throw 'Pinned runtime required'}
 $runtimeDir=Join-Path $lab 'runtime';$null=New-Item -ItemType Directory $runtimeDir;$runtimeExe=Join-Path $runtimeDir 'SumatraPDF.exe';Copy-Item $Runtime $runtimeExe
 $policy="[Policies]`nInternetAccess = 0`nDiskAccess = 0`nSavePreferences = 0`nRegistryAccess = 0`nPrinterAccess = 0`nCopySelection = 0`nFullscreenAccess = 0`nLinkProtocols =`nSafeFileTypes =`n"
 $policyPath=Join-Path $runtimeDir 'sumatrapdfrestrict.ini';[IO.File]::WriteAllText($policyPath,$policy,$utf8)
 $receipt['runtime_sha256']=Hash $runtimeExe;$receipt['policy_sha256']=Hash $policyPath
 Add-Type -AssemblyName System.Drawing;Add-Type -AssemblyName System.Windows.Forms
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-sumatra-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','Accessibility.dll','System.Drawing.dll')
 foreach($name in @('dcm-text-probe','dcm-undo-input','dcm-startup-probe')){Add-Type -Path (Join-Path $repo ('tests/native/'+$name+'.cs')) -ReferencedAssemblies @('System.dll','System.Core.dll')}
 . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
 Get-ChildItem $fixture.inbox -File|ForEach-Object {$expected[$_.Name]=Hash $_.FullName};$receipt['initial_hashes']=$expected
 $textExpected=[IO.File]::ReadAllText((Join-Path $fixture.inbox '01-alpha.txt'))
 $exe=Join-Path $lab ('DCM-Startup-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item $build.app_artifact.path $exe
 $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
 $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming;$psi.EnvironmentVariables['TEMP']=$lab;$psi.EnvironmentVariables['TMP']=$lab
 $psi.EnvironmentVariables['DCM_STARTUP_TRACE']='1';$psi.EnvironmentVariables.Remove('DCM_LAYOUT_TRACE');$psi.EnvironmentVariables['DCM_PDF_PROVIDER']='sumatra';$psi.EnvironmentVariables['DCM_SUMATRA_EXE']=$runtimeExe;$psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
 $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'Host start failed'}
 $receipt['host_pid']=$app.Id;$receipt['host_start']=$app.StartTime.ToUniversalTime().ToString('o')
 $until=[DateTime]::UtcNow.AddSeconds(15);do{$app.Refresh();$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 50}while([DateTime]::UtcNow -lt $until)
 Wait 'four fixtures' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 4};$null=SelectFile '01-alpha.txt';TextReady
 $area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea;[DcmNative]::Restore($hwnd,$app.Id,$area.Left+8,$area.Top+8);Start-Sleep -Milliseconds 300
 for($i=1;$i -le 4;$i++) {
  $monitor=New-Object DcmStartupMonitor($hwnd,$app.Id);$until=[DateTime]::UtcNow.AddSeconds(3)
  while($monitor.SampleCount -lt 2 -and [DateTime]::UtcNow -lt $until){Start-Sleep -Milliseconds 10}
  if($monitor.SampleCount -lt 2){throw 'Queue probe not acknowledged'}
  $mark=$monitor.ElapsedMs;$clock=[Diagnostics.Stopwatch]::StartNew();$index=SelectFile '08-preview.pdf'
  while([DcmStartupMonitor]::Ready($hwnd,$app.Id) -lt $i -and $clock.ElapsedMilliseconds -lt 20000){Own;Start-Sleep -Milliseconds 10}
  $readyMs=$clock.Elapsed.TotalMilliseconds;if([DcmStartupMonitor]::Ready($hwnd,$app.Id) -ne $i){throw 'Document-ready observation deadline'}
  Start-Sleep -Milliseconds 100;$samples=@($monitor.Stop());$monitor.Dispose();$monitor=$null
  $timed=@($samples|Where-Object {$_.PostedMs+$_.DelayMs -ge $mark -and $_.PostedMs -le $mark+$readyMs})
  if(-not $timed.Count -or @($timed|Where-Object {-not $_.Acknowledged}).Count){throw 'Incomplete queue evidence'}
  $max=($timed|Measure-Object DelayMs -Maximum).Maximum
  $measurements.Add([ordered]@{sequence=$i;condition=$(if($i -eq 1){'first-in-fresh-host'}else{'repeat-in-same-host'});ready_status_ms=$readyMs;selection_at_ms=$mark;max_queue_ack_ms=$max;over_250ms=($max -gt 250);samples=$samples});Json (Join-Path $lab 'startup-measurements.json') $measurements
  if(([DcmUndoInput]::TryText($hwnd,$app.Id,1005)) -cne '08-preview' -or (@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)) -join ',') -ne [string]$index){throw 'Wrong selected target'}
  $v=[DcmSumatraProbe]::Find($hwnd,$app.Id);if(-not $v){throw 'No embedded viewer'};$p=Get-Process -Id $v.PID;$w=Get-CimInstance Win32_Process -Filter ('ProcessId='+$v.PID)
  if($p.Path -ne $runtimeExe -or $w.ParentProcessId -ne $app.Id -or -not $w.CommandLine.Contains((Join-Path $fixture.inbox '08-preview.pdf')) -or $w.CommandLine -notmatch '\-restrict' -or $w.CommandLine -notmatch '\-plugin'){throw 'Viewer identity mismatch'}
  $viewers.Add([ordered]@{pid=$p.Id;start=$p.StartTime.ToUniversalTime().ToString('o');path=$p.Path});[DcmSumatraProbe]::CheckGeometry($hwnd,$app.Id,$v.PID)
  Start-Sleep -Milliseconds 150
  foreach($crop in @($false,$true)){$name='startup-'+$i+$(if($crop){'-content'}else{''});$s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$crop);$shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false});if($crop -and ($s.TealSamples -lt 1000 -or $s.OrangeSamples -gt 100)){throw 'Wrong or blank PDF content'}}
  $null=SelectFile '01-alpha.txt';TextReady;Files
 }
 [DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000) -or $app.ExitCode -ne 0){throw 'Normal close failed'};$receipt['exit_code']=$app.ExitCode;$app.Dispose();$app=$null
 if(@((PrivateProcesses)).Count -or @(Get-ChildItem $lab -Directory -Filter 'dcm-sumatra-*').Count){throw 'Private viewer/profile survived'}
 foreach($v in $viewers){$p=Get-Process -Id $v.pid -ErrorAction SilentlyContinue;if($p -and $p.StartTime.ToUniversalTime().ToString('o') -eq $v.start){throw 'Recorded viewer survived'}}
 Files;Source
 if((Hash $runtimeExe) -ne $receipt.runtime_sha256 -or (Hash $policyPath) -ne $receipt.policy_sha256 -or (Hash $exe) -ne $receipt.app_sha256){throw 'Runtime/policy/app changed'}
 $trace=Join-Path $lab 'dcm-startup-trace.json';if(-not(Test-Path $trace)){throw 'Trace missing'}
 $receipt['startup_trace']=$trace;$receipt['startup_trace_sha256']=Hash $trace;$receipt['responsiveness_accepted']=(@($measurements|Where-Object {$_.over_250ms}).Count -eq 0)
 $receipt['source_unchanged']=$true;$receipt['fixtures_preserved']=$true;$receipt['journal_empty']=$true;$receipt.passed=$true
}catch{
 $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)
 if($app -and -not $app.HasExited){try{Own;$s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab 'failure.png'),$false);$shots.Add([ordered]@{name='failure';path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false})}catch{$receipt['failure_capture_error']=$_.Exception.Message}}
}
finally {
 if($monitor){$monitor.Dispose()}
 if($app){try{Own;[DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000)){throw 'Owned cleanup timeout'};$receipt['cleanup_exit']=$app.ExitCode;$app.Dispose()}catch{$receipt['cleanup_error']=$_.Exception.Message};$receipt.passed=$false}
 try{if($fixture){Files;$receipt['fixtures_preserved']=$true};if($head){Source;$receipt['source_unchanged']=$true}}catch{$receipt.passed=$false;$receipt['integrity_error']=$_.Exception.Message}
 $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt;Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
