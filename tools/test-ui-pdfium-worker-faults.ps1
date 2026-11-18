[CmdletBinding()]
param(
 [Parameter(Mandatory=$true)][string]$CandidateExe,
 [Parameter(Mandatory=$true)][string]$WorkerExe,
 [ValidateSet('cancel','crash','timeout')][string]$Scenario='cancel',
 [string]$LabRoot='D:\FlowverseReportsRepo'
)
# Only disposable native-GUI acceptance fixtures and the child this test starts.
# The worker MUST be built with the pdfium_acceptance build tag; never run
# this script against production or user documents.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$exe=[IO.Path]::GetFullPath($CandidateExe)
$worker=[IO.Path]::GetFullPath($WorkerExe)
foreach($path in @($exe,$worker,'C:\Flowverse\runtime\pdfium\bin\pdfium.dll')){
 if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw ('Missing acceptance prerequisite: '+$path)}
}
$lab=Join-Path $LabRoot ('dcm-pdfium-fault-'+$Scenario+'-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,6))
$null=New-Item -ItemType Directory -Path $lab
$trace=Join-Path $lab 'pdf-trace.jsonl'
$marker=Join-Path $lab 'injected-once.marker'
$receipt=[ordered]@{schema=1;scenario=$Scenario;started_utc=[DateTime]::UtcNow.ToString('o');lab=$lab;
  commit=(& git -C $repo rev-parse HEAD).Trim();candidate=$exe;worker=$worker;
  candidate_sha256=(Get-FileHash $exe -Algorithm SHA256).Hash;
  worker_sha256=(Get-FileHash $worker -Algorithm SHA256).Hash;
  passed=$false;normal_exit=$false;steps=@();captures=@();error=$null}
$app=$null;$hwnd=[IntPtr]::Zero
function Step([string]$name,$value){
 $receipt.steps+=@([ordered]@{name=$name;utc=[DateTime]::UtcNow.ToString('o');result=$value})
}
function Owned {
 if($null -eq $app){throw 'No owned Quick Sort process'}
 $app.Refresh()
 if($app.HasExited){throw 'Quick Sort exited unexpectedly'}
 [DcmNative]::RequireOwned($hwnd,$app.Id)
 if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected owned dialog'}
}
function Wait([string]$name,[scriptblock]$condition,[int]$seconds=15){
 $end=[DateTime]::UtcNow.AddSeconds($seconds)
 do{
  Owned
  if(& $condition){return}
  Start-Sleep -Milliseconds 75
 }while([DateTime]::UtcNow -lt $end)
 throw ('Deadline waiting for '+$name)
}
function Events {
 if(-not(Test-Path -LiteralPath $trace)){return @()}
 return @(Get-Content -LiteralPath $trace | Where-Object {$_.Trim()} | ForEach-Object {$_|ConvertFrom-Json})
}
function Selected([string]$name){
 $items=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
 $index=[Array]::IndexOf($items,$name)
 if($index -lt 0){throw ('Fixture unavailable: '+$name)}
 [DcmUndoInput]::Select($hwnd,$app.Id,$index)
 $stem=[IO.Path]::GetFileNameWithoutExtension($name)
 Wait ('selection '+$name) {([DcmUndoInput]::TryText($hwnd,$app.Id,1005)) -ceq $stem}
}
function AcceptedRender([string]$path,[int]$seconds=15){
 Wait ('applied preview '+$path) {
  @((Events)|Where-Object {$_.event -eq 'result-applied' -and $_.input_path -eq $path}).Count -gt 0
 } $seconds
 $page=@([DcmNative]::Snapshot($hwnd,$app.Id)|Where-Object {$_.Visible -and $_.Text -eq 'Page 1 of 2'})
 if($page.Count -lt 1){throw 'Expected visible Page 1 of 2 label after applied PDF result'}
 $area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
 [DcmNative]::Restore($hwnd,$app.Id,$area.Left+8,$area.Top+8)
 Start-Sleep -Milliseconds 200
 $capture=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ('accepted-'+$Scenario+'.png')),$false)
 $bmp=[Drawing.Bitmap]::FromFile($capture.Path)
 $pixels=0
 try{
  for($y=0;$y -lt $bmp.Height;$y+=5){
   for($x=0;$x -lt $bmp.Width;$x+=5){
    $col=$bmp.GetPixel($x,$y)
    if($col.G -gt 95 -and $col.B -gt 125 -and $col.R -lt 85){$pixels++}
   }
  }
 }finally{$bmp.Dispose()}
 if($pixels -lt 125){throw ('Valid PDF page content missing: '+$pixels)}
 $receipt.captures+=@([ordered]@{path=$capture.Path;sha256=(Get-FileHash $capture.Path -Algorithm SHA256).Hash;teal_sampled_pixels=$pixels;visual_reviewed=$false})
 Step 'valid-preview-applied' @{path=$path;teal_sampled_pixels=$pixels}
}
try{
 Add-Type -AssemblyName System.Drawing
 Add-Type -AssemblyName System.Windows.Forms
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
 . (Join-Path $repo 'tests/native/new-fixtures.ps1')
 $fixture=New-DcmFixtures $lab
 $old=Join-Path $fixture.inbox '08-preview.pdf'
 $new=Join-Path $fixture.inbox '09-followup.pdf'
 Copy-Item -LiteralPath $old -Destination $new
 $oldHash=(Get-FileHash $old -Algorithm SHA256).Hash
 $newHash=(Get-FileHash $new -Algorithm SHA256).Hash
 $psi=New-Object Diagnostics.ProcessStartInfo
 $psi.FileName=$exe
 $psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"'
 $psi.WorkingDirectory=$lab
 $psi.UseShellExecute=$false
 foreach($kv in @{
  'LOCALAPPDATA'=$fixture.local;'APPDATA'=$fixture.roaming;
  'TEMP'=$lab;'TMP'=$lab;
  'FILEWORKSPACE_RUNTIME_DIR'='C:\Flowverse\runtime';
  'DCM_PDF_PROVIDER'='pdfium';'DCM_FLOWPDF_WORKER'=$worker;
  'DCM_FLOWPDF_PERSISTENT'='1';'DCM_PDF_TRACE'=$trace;
  'DCM_QUICKSORT_BACKUP_DIR'=(Join-Path $lab 'backups');
  'FLOWPDF_ACCEPTANCE_TRIGGER_NAME'='08-preview.pdf';
  'FLOWPDF_ACCEPTANCE_MARKER'=$marker;
  'FLOWPDF_ACCEPTANCE_ACTION'=$(switch($Scenario){'cancel'{'delay'};'crash'{'crash'};'timeout'{'delay_every'}});
  'FLOWPDF_ACCEPTANCE_DELAY_MS'=$(if($Scenario -eq 'timeout'){'22000'}else{'8000'})
 }.GetEnumerator()){$psi.EnvironmentVariables[$kv.Key]=$kv.Value}
 $app=New-Object Diagnostics.Process
 $app.StartInfo=$psi
 if(-not $app.Start()){throw 'Cannot start owned Quick Sort candidate'}
 $receipt.host_pid=$app.Id
 $receipt.host_started_utc=$app.StartTime.ToUniversalTime().ToString('o')
 $until=[DateTime]::UtcNow.AddSeconds(20)
 do{
  $app.Refresh()
  if($app.HasExited){throw 'Candidate exited during startup'}
  $hwnd=$app.MainWindowHandle
  if($hwnd -ne [IntPtr]::Zero){break}
  Start-Sleep -Milliseconds 90
 }while([DateTime]::UtcNow -lt $until)
 if($hwnd -eq [IntPtr]::Zero){throw 'No owned native GUI window'}
 Wait 'five disposable fixtures' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 5}
 Selected '08-preview.pdf'
 Wait 'fault marker from test-tagged worker' {Test-Path -LiteralPath $marker} 12
 $markerUTC=[DateTime]::UtcNow
 Step 'fault-injected' @{scenario=$Scenario;marker=$marker}
 if($Scenario -eq 'cancel'){
  $changedAt=[DateTime]::UtcNow
  Selected '09-followup.pdf'
  AcceptedRender $new 12
  $applied=@((Events)|Where-Object {$_.event -eq 'result-applied' -and $_.input_path -eq $new}|
    Sort-Object sequence|Select-Object -Last 1)[0]
  $elapsed=([DateTimeOffset]::Parse($applied.utc).UtcDateTime-$changedAt).TotalMilliseconds
  Wait 'obsolete selection canceled' {
   @((Events)|Where-Object {$_.event -eq 'worker-canceled' -and $_.input_path -eq $old}).Count -ge 1
  } 8
  Wait 'obsolete result discarded' {
   @((Events)|Where-Object {$_.event -eq 'result-discarded' -and $_.input_path -eq $old}).Count -ge 1
  } 8
  $events=@(Events)
  if(@($events|Where-Object {$_.event -match 'persistent-transport-fallback|persistent-fallback-started'}).Count){
   throw 'Canceled stale render triggered an unintended one-shot fallback'
  }
  if($elapsed -gt 4000){throw ('Replacement page took too long after stale cancellation: '+$elapsed+'ms')}
  Step 'cancel-switch-accepted' @{replacement_ms=$elapsed;max_ms=4000;cancel_event=$true;obsolete_discarded=$true;one_shot_fallbacks=0}
  Selected '08-preview.pdf'
  AcceptedRender $old 12
  Step 'first-file-reopen' @{old_source_rendered_after_cancel=$true}
 }elseif($Scenario -eq 'crash'){
  AcceptedRender $old 16
  $events=@(Events)
  $starts=@($events|Where-Object {$_.event -eq 'worker-started' -and $_.input_path -eq $old})
  $pids=@($starts|Select-Object -ExpandProperty renderer_pid -Unique)
  if($pids.Count -lt 2){throw ('Expected owned worker restart after crash, saw '+($pids -join ','))}
  if(@($events|Where-Object {$_.event -match 'persistent-transport-fallback|persistent-fallback-started'}).Count){
   throw 'Unexpected one-shot fallback: retry should recover from injected single crash'
  }
  Step 'crash-restart-accepted' @{worker_start_pids=$pids;one_shot_fallbacks=0}
  Selected '09-followup.pdf'
  AcceptedRender $new 12
 }else{
  # Both persistent attempts exceed the supervised 15-second deadline.
  # The standalone one-shot renderer must still serve the valid PDF.
  AcceptedRender $old 48
  $events=@(Events)
  $fallback=@($events|Where-Object {$_.event -eq 'persistent-fallback-started' -and $_.input_path -eq $old})
  $starts=@($events|Where-Object {$_.event -eq 'worker-started' -and $_.input_path -eq $old})
  if($fallback.Count -ne 1 -or $starts.Count -lt 2){
   throw ('Expected bounded timeout and one-shot recovery; starts='+$starts.Count+' fallback='+$fallback.Count)
  }
  Step 'timeout-fallback-accepted' @{persistent_attempts=$starts.Count;one_shot_fallbacks=$fallback.Count}
  Selected '09-followup.pdf'
  AcceptedRender $new 12
 }
 if((Get-FileHash $old -Algorithm SHA256).Hash -ne $oldHash){throw 'First fixture bytes changed'}
 if((Get-FileHash $new -Algorithm SHA256).Hash -ne $newHash){throw 'Second fixture bytes changed'}
 Step 'source-integrity' @{both_pdf_hashes_preserved=$true}
 $receipt.passed=$true
}catch{
 $receipt.error=$_.Exception.Message
 Write-Output ('FAIL='+$receipt.error)
}finally{
 if($null -ne $app){
  try{
   $app.Refresh()
   if(-not $app.HasExited -and $hwnd -ne [IntPtr]::Zero){
    [DcmNative]::CloseAsync($hwnd,$app.Id)
    if(-not $app.WaitForExit(20000)){throw 'Owned GUI close did not complete'}
   }
   $receipt.normal_exit=($app.HasExited -and $app.ExitCode -eq 0)
   if(-not $receipt.normal_exit){$receipt.passed=$false}
  }catch{
   $receipt.passed=$false
   $receipt['close_error']=$_.Exception.Message
  }
  $app.Dispose()
 }
 if(Test-Path $trace){
  $events=@(Events)
  $pids=@($events|Where-Object {$_.event -eq 'worker-started' -and $_.renderer_pid -gt 0}|Select-Object -ExpandProperty renderer_pid -Unique)
  foreach($workerPID in $pids){
   $child=Get-Process -Id $workerPID -ErrorAction SilentlyContinue
   if($null -ne $child){
    $receipt.passed=$false
    $receipt['orphan_worker_pid']=$workerPID
   }
  }
  Step 'worker-process-cleanup' @{worker_pids=$pids;all_workers_exited=$receipt.passed}
 }
 $receipt['completed_utc']=[DateTime]::UtcNow.ToString('o')
 [IO.File]::WriteAllText((Join-Path $lab 'receipt.json'),($receipt|ConvertTo-Json -Depth 20))
 Write-Output ('REPORT='+$lab)
 Write-Output ('PASSED='+$receipt.passed)
 Write-Output ('NORMAL_EXIT='+$receipt.normal_exit)
}
if(-not $receipt.passed){exit 1}
