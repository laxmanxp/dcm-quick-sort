[CmdletBinding()]
param(
  [Parameter(Mandatory=$true)][string]$CandidateExe,
  [Parameter(Mandatory=$true)][string]$WorkerExe,
  [string]$LabRoot='D:\FlowverseReportsRepo',
  [switch]$Persistent,
  [switch]$DevDefault,
  [switch]$WindowsRollback
)
# Test-only. Source is GitHub; Lenovo is a disposable acceptance environment.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($DevDefault -and ($Persistent -or $WindowsRollback)){throw 'Use one provider-selection scenario per test'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$exe=[IO.Path]::GetFullPath($CandidateExe)
$worker=[IO.Path]::GetFullPath($WorkerExe)
if(-not (Test-Path $exe -PathType Leaf) -or -not (Test-Path $worker -PathType Leaf)){throw 'Candidate/worker binaries missing'}
if(-not (Test-Path 'C:\Flowverse\runtime\pdfium\bin\pdfium.dll')){throw 'Managed PDFium runtime missing'}
$lab=Join-Path $LabRoot ('dcm-pdfium-gui-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,7))
$null=New-Item -ItemType Directory -Path $lab
$receipt=[ordered]@{schema=1;feature='DCM-SHARED-PDFIUM-GUI';started_utc=[DateTime]::UtcNow.ToString('o');lab=$lab;commit=(& git -C $repo rev-parse HEAD).Trim();candidate=$exe;worker=$worker;passed=$false;steps=@();captures=@();error=$null;normal_exit=$false;visual_reviewed=$false}
$receipt['persistent']=([bool]($Persistent -or $DevDefault))
$receipt['dev_default']=([bool]$DevDefault)
$receipt['windows_rollback']=([bool]$WindowsRollback)
$receipt.candidate_sha256=(Get-FileHash $exe -Algorithm SHA256).Hash
$receipt.worker_sha256=(Get-FileHash $worker -Algorithm SHA256).Hash
$trace=Join-Path $lab 'pdf-trace.jsonl'
$app=$null
$hwnd=[IntPtr]::Zero
$failure=$null
$serviceChildPID=0
function Step([string]$name,$data){$receipt.steps+=@([ordered]@{name=$name;utc=[DateTime]::UtcNow.ToString('o');result=$data})}
function Owned {
 if($null -eq $app){throw 'No owned app'}
 $app.Refresh()
 if($app.HasExited){throw 'Owned app exited unexpectedly'}
 [DcmNative]::RequireOwned($hwnd,$app.Id)
 if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected owned dialog'}
}
function Wait([string]$name,[scriptblock]$condition,[int]$seconds=25){
 $end=[DateTime]::UtcNow.AddSeconds($seconds)
 do{Owned;if(& $condition){return};Start-Sleep -Milliseconds 120}while([DateTime]::UtcNow -lt $end)
 throw ('Deadline waiting for '+$name)
}
function List-Selected([string]$name){
 $items=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
 $index=[Array]::IndexOf($items,$name)
 if($index -lt 0){throw ('Fixture absent: '+$name)}
 [DcmUndoInput]::Select($hwnd,$app.Id,$index)
 $stem=[IO.Path]::GetFileNameWithoutExtension($name)
 Wait ('selected '+$name) {([DcmUndoInput]::TryText($hwnd,$app.Id,1005)) -ceq $stem}
}
function TextControl([int]$id){return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,$id),$app.Id)}
function Hash([string]$path){return (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()}
function Exclusive([string]$path){
 $f=[IO.File]::Open($path,[IO.FileMode]::Open,[IO.FileAccess]::ReadWrite,[IO.FileShare]::None)
 $f.Dispose()
}
function Journal([string]$from,[string]$to,[string]$state,[int]$count) {
 $records=@{}
 if(Test-Path $journal){
  foreach($line in [IO.File]::ReadAllLines($journal)){
   if($line.Trim()){$row=$line|ConvertFrom-Json;$records[$row.id]=$row}
  }
 }
 if($records.Count -ne $count){throw ('Expected '+$count+' journal entries, got '+$records.Count)}
 $matches=@($records.Values|Where-Object {$_.record.from -eq $from -and $_.record.to -eq $to})
 if($matches.Count -ne 1 -or $matches[0].state -ne $state){throw ('Journal state mismatch '+$state)}
 return [string]$matches[0].id
}
function SelectUndoRecord([string]$name) {
 Wait 'Undo button ready' {[DcmNative]::IsWindowEnabled([DcmNative]::Control($hwnd,$app.Id,1008))}
 $items=@([DcmNative]::ListItems($hwnd,$app.Id,1017))
 $matches=@()
 for($i=0;$i -lt $items.Count;$i++){if($items[$i].Contains($name)){$matches+=,$i}}
 if($matches.Count -ne 1){throw ('Undo history row not unique for '+$name)}
 $null=[DcmNative]::SelectSingleItem($hwnd,$app.Id,1017,$matches[0])
}
function SwitchSource([string]$path,[int]$count){
 [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1001,$path)
 [DcmNative]::ClickAsync($hwnd,$app.Id,1011)
 Wait 'source switch' {(TextControl 1001) -eq $path -and @([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $count}
}
function VerifyPDF([string]$name,[string]$expectedPath,[string]$expectedHash) {
 List-Selected $name
 PageReady 'Page 1 of 2'
 if((Hash $expectedPath) -ne $expectedHash){throw 'PDF bytes changed'}
}
function Label {return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,1061),$app.Id)}
function PageLabel {return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,1060),$app.Id)}
function PreviewSnapshot {
 return @([DcmNative]::Snapshot($hwnd,$app.Id) | Where-Object {$_.Visible -and ($_.Text -like 'DCM PDF Preview*' -or $_.Text -like 'Page * of *')})
}
$lastPDFReadyGeneration=0
function PageReady([string]$want){
 Wait ('rendered '+$want) {
  $snap=PreviewSnapshot
  if(@($snap | Where-Object {$_.Text -eq $want}).Count -eq 0){return $false}
  if(-not(Test-Path -LiteralPath $trace)){return $false}
  $applied=@(Get-Content -LiteralPath $trace | ForEach-Object {$_|ConvertFrom-Json} |
    Where-Object {$_.event -eq 'result-applied' -and $_.state.active})
  if(-not $applied.Count){return $false}
  $latest=@($applied | Sort-Object generation)[-1]
  if([uint64]$latest.generation -le [uint64]$script:lastPDFReadyGeneration){return $false}
  $script:lastPDFReadyGeneration=[uint64]$latest.generation
  return $true
 } 45
}
function TealSpan([string]$path){
 $bmp=[Drawing.Bitmap]::FromFile($path)
 try{
  $min=999999;$max=-1;$count=0
  for($y=0;$y -lt $bmp.Height;$y+=3){
   for($x=0;$x -lt $bmp.Width;$x+=3){
    $c=$bmp.GetPixel($x,$y)
    if($c.R -lt 35 -and $c.G -gt 125 -and $c.G -lt 155 -and $c.B -gt 145 -and $c.B -lt 187){
     $min=[Math]::Min($min,$x);$max=[Math]::Max($max,$x);$count++
    }
   }
  }
  if($count -lt 400 -or $max-$min -lt 100){throw 'PDF content teal horizontal span insufficient'}
  return [pscustomobject]@{min=$min;max=$max;count=$count}
 }finally{$bmp.Dispose()}
}
function Capture([string]$name,[string]$rgb){
 Owned
 $area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
 [DcmNative]::Restore($hwnd,$app.Id,$area.Left+8,$area.Top+8)
 Start-Sleep -Milliseconds 450
 $shot=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false)
 $pixels=0
 $bmp=[Drawing.Bitmap]::FromFile($shot.Path)
 try {
  for($y=0;$y -lt $bmp.Height;$y+=4){
   for($x=0;$x -lt $bmp.Width;$x+=4){
    $c=$bmp.GetPixel($x,$y)
    if($rgb -eq 'teal' -and $c.G -gt 95 -and $c.B -gt 125 -and $c.R -lt 85){$pixels++}
    if($rgb -eq 'orange' -and $c.R -gt 140 -and $c.G -gt 30 -and $c.G -lt 160 -and $c.B -lt 85){$pixels++}
   }
  }
 }finally{$bmp.Dispose()}
 $receipt.captures+=@([ordered]@{name=$name;path=$shot.Path;left=$shot.Left;top=$shot.Top;width=$shot.Width;height=$shot.Height;sha256=(Get-FileHash $shot.Path -Algorithm SHA256).Hash;colour=$rgb;sampled_pixels=$pixels;visual_reviewed=$false})
 if($pixels -lt 250){throw ('Expected '+$rgb+' PDF content pixels missing: '+$pixels)}
 Step ('visible-content-'+$name) @{colour=$rgb;sampled_pixels=$pixels}
}
try{
 Add-Type -AssemblyName System.Drawing
 Add-Type -AssemblyName System.Windows.Forms
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-sumatra-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','Accessibility.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-pdfium-wheel-input.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
 . (Join-Path $repo 'tests/native/new-fixtures.ps1')
 $fixture=New-DcmFixtures $lab
 $pdf=Join-Path $fixture.inbox '08-preview.pdf'
 $sha=(Get-FileHash $pdf -Algorithm SHA256).Hash
 $journal=Join-Path $fixture.config 'recovery\move-journal.jsonl'
 $txtHashes=@{}
 Get-ChildItem -LiteralPath $fixture.inbox -Filter '*.txt' | ForEach-Object {$txtHashes[$_.Name]=Hash $_.FullName}
 $psi=New-Object Diagnostics.ProcessStartInfo
 $psi.FileName=$exe
 $psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"'
 $psi.WorkingDirectory=$lab
 $psi.UseShellExecute=$false
 foreach($kv in @{
  'FILEWORKSPACE_RUNTIME_DIR'='C:\Flowverse\runtime';
  'LOCALAPPDATA'=$fixture.local;'APPDATA'=$fixture.roaming;'TEMP'=$lab;'TMP'=$lab;
  'DCM_PDF_TRACE'=$trace;'DCM_QUICKSORT_BACKUP_DIR'=(Join-Path $lab 'backups')
 }.GetEnumerator()){$psi.EnvironmentVariables[$kv.Key]=$kv.Value}
 foreach($key in @('DCM_PDF_PROVIDER','DCM_FLOWPDF_WORKER','DCM_FLOWPDF_PERSISTENT')){$psi.EnvironmentVariables.Remove($key)}
 if($DevDefault) {
  # A genuine automatic-default acceptance must not supply PDFium-specific
  # configuration or the managed runtime location to the child.
  $psi.EnvironmentVariables.Remove('FILEWORKSPACE_RUNTIME_DIR')
  $psi.EnvironmentVariables.Remove('FILEWORKSPACE_PDFIUM')
 }
 if($WindowsRollback){
  $psi.EnvironmentVariables['DCM_PDF_PROVIDER']='windows'
 }elseif(-not $DevDefault){
  $psi.EnvironmentVariables['DCM_PDF_PROVIDER']='pdfium'
  $psi.EnvironmentVariables['DCM_FLOWPDF_WORKER']=$worker
  $psi.EnvironmentVariables['DCM_FLOWPDF_PERSISTENT']=$(if($Persistent){'1'}else{'0'})
 }
 $app=New-Object Diagnostics.Process
 $app.StartInfo=$psi
 if(-not $app.Start()){throw 'Failed to start test candidate'}
 $receipt['host_pid']=$app.Id
 $receipt['host_started_utc']=$app.StartTime.ToUniversalTime().ToString('o')
 $deadline=[DateTime]::UtcNow.AddSeconds(20)
 do{$app.Refresh();if($app.HasExited){throw 'Candidate exited at startup'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 120}while([DateTime]::UtcNow -lt $deadline)
 if($hwnd -eq [IntPtr]::Zero){throw 'Candidate HWND missing'}
 Wait 'fixture loaded' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 4}
 List-Selected '08-preview.pdf'
 PageReady 'Page 1 of 2'
 Step 'page-one-controls' (PreviewSnapshot)
 Capture '01-page-one' 'teal'
 [DcmNative]::ClickAsync($hwnd,$app.Id,1061)
 PageReady 'Page 2 of 2'
 Step 'page-two-controls' (PreviewSnapshot)
 Capture '02-page-two' 'orange'
 [DcmNative]::ClickAsync($hwnd,$app.Id,1060)
 PageReady 'Page 1 of 2'
 Capture '03-page-one-return' 'teal'
 # Native toolbar zoom; wheel mouse-location acceptance is a separate gate.
 [DcmNative]::ClickAsync($hwnd,$app.Id,1022)
 Wait 'zoom render success' {
  $events=if(Test-Path $trace){@(Get-Content $trace | Where-Object {$_ -match 'result-applied'})}else{@()}
  $events.Count -ge 4
 } 40
 Capture '04-after-zoom-in' 'teal'
 Step 'toolbar-zoom-success' 'PDFium result applied after native Zoom In control'
 $pane=[DcmNative]::ChildByClass($hwnd,$app.Id,'QuickSortPreview')
 $rect=[DcmNative+RECT]::new()
 if(-not [DcmNative]::GetWindowRect($pane,[ref]$rect)){throw 'Owned preview rectangle unavailable'}
 $width=$rect.Right-$rect.Left
 $height=$rect.Bottom-$rect.Top
 $beforeEvents=@(Get-Content -LiteralPath $trace | Where-Object {$_ -match 'result-applied'}).Count
 [DcmPdfWheelInput]::Wheel($hwnd,$pane,$app.Id,[int]($width*0.4),[int]($height*0.5),2)
 Wait 'Ctrl-wheel PDFium completed' {
  $now=@(Get-Content -LiteralPath $trace | Where-Object {$_ -match 'result-applied'}).Count
  $now -gt $beforeEvents
 } 45
 Start-Sleep -Milliseconds 650
 Capture '05-after-ctrl-wheel' 'teal'
 $beforeShot=@($receipt.captures | Where-Object {$_.name -eq '04-after-zoom-in'})[0]
 $afterShot=@($receipt.captures | Where-Object {$_.name -eq '05-after-ctrl-wheel'})[0]
 $beforeSpan=TealSpan $beforeShot.path
 $afterSpan=TealSpan $afterShot.path
 $mouseScreenX=$rect.Left+[int]($width*0.4)
 $beforeFX=($mouseScreenX-$beforeShot.left-$beforeSpan.min)/($beforeSpan.max-$beforeSpan.min)
 $afterFX=($mouseScreenX-$afterShot.left-$afterSpan.min)/($afterSpan.max-$afterSpan.min)
 $drift=[Math]::Abs($beforeFX-$afterFX)
 Step 'cursor-anchor-image-diagnostic' @{before_fraction=$beforeFX;after_fraction=$afterFX;drift=$drift;before_span=$beforeSpan;after_span=$afterSpan;qualifies=($drift -lt 0.04);note='Horizontal colour-panel edge metric; not a full 2-axis viewport anchor certification'}
 $afterEvents=@(Get-Content -LiteralPath $trace | Where-Object {$_ -match 'result-applied'}).Count
 Step 'ctrl-wheel-zoom-routed' @{before_render_count=$beforeEvents;after_render_count=$afterEvents;pointer_fraction_x=0.4;pointer_fraction_y=0.5;detents=2;cursor_anchoring_verified=$false}
 List-Selected '01-alpha.txt'
 $file=[IO.File]::Open($pdf,[IO.FileMode]::Open,[IO.FileAccess]::ReadWrite,[IO.FileShare]::None)
 $file.Dispose()
 if((Get-FileHash $pdf -Algorithm SHA256).Hash -ne $sha){throw 'PDF fixture changed'}
 Step 'pdf-exclusive-release' 'exclusive read/write lock succeeded after PDF deselection'
 List-Selected '08-preview.pdf'
 PageReady 'Page 1 of 2'
 Capture '06-reopened-pdf' 'teal'
 Step 'pdf-reopen' 'same PDF reopened after exclusive file access'
 # Operate only on the disposable PDF fixture, never on a user file.
 [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1003,'Accepted')
 [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1005,'undo-preview')
 [DcmNative]::ClickAsync($hwnd,$app.Id,1006)
 $renamed=Join-Path $fixture.inbox 'undo-preview.pdf'
 Wait 'PDF rename completed' {(Test-Path -LiteralPath $renamed) -and -not(Test-Path -LiteralPath $pdf)}
 if((Hash $renamed) -ne $sha.ToLowerInvariant()){throw 'Rename altered PDF content'}
 Wait 'Rename journal completed' {try {$null=Journal $pdf $renamed 'completed' 1;return $true} catch {return $false}}
 $renameID=Journal $pdf $renamed 'completed' 1
 Step 'rename-pdf' @{source=$pdf;target=$renamed;bytes_preserved=$true;journal_id=$renameID}
 Wait 'renamed selection' {(TextControl 1005) -eq 'undo-preview'}
 PageReady 'Page 1 of 2'
 Capture '07-renamed-pdf-visible' 'teal'
 SelectUndoRecord 'undo-preview.pdf'
 [DcmNative]::ClickAsync($hwnd,$app.Id,1008)
 Wait 'Undo Rename completed' {(Test-Path -LiteralPath $pdf) -and -not(Test-Path -LiteralPath $renamed)}
 Wait 'Undo Rename journal' {try {$null=Journal $pdf $renamed 'undone' 1;return $true} catch {return $false}}
 if((Hash $pdf) -ne $sha.ToLowerInvariant() -or (Journal $pdf $renamed 'undone' 1) -ne $renameID){throw 'Undo Rename content/journal mismatch'}
 Step 'undo-rename-pdf' @{original_restored=$true;bytes_preserved=$true}
 VerifyPDF '08-preview.pdf' $pdf ($sha.ToLowerInvariant())
 Capture '08-restored-after-rename-undo' 'teal'
 [DcmNative]::ClickAsync($hwnd,$app.Id,1041)
 $moved=Join-Path $fixture.destination '08-preview.pdf'
 Wait 'PDF move completed' {(Test-Path -LiteralPath $moved) -and -not(Test-Path -LiteralPath $pdf)}
 if((Hash $moved) -ne $sha.ToLowerInvariant()){throw 'Move altered PDF content'}
 Wait 'Move journal completed' {try {$null=Journal $pdf $moved 'completed' 2;return $true} catch {return $false}}
 $moveID=Journal $pdf $moved 'completed' 2
 Step 'move-pdf' @{destination=$moved;bytes_preserved=$true;journal_id=$moveID}
 SwitchSource $fixture.destination 1
 VerifyPDF '08-preview.pdf' $moved ($sha.ToLowerInvariant())
 Capture '09-moved-pdf-visible' 'teal'
 SelectUndoRecord '08-preview.pdf'
 [DcmNative]::ClickAsync($hwnd,$app.Id,1008)
 Wait 'Undo Move completed' {(Test-Path -LiteralPath $pdf) -and -not(Test-Path -LiteralPath $moved)}
 Wait 'Undo Move journal' {try {$null=Journal $pdf $moved 'undone' 2;return $true} catch {return $false}}
 if((Hash $pdf) -ne $sha.ToLowerInvariant() -or (Journal $pdf $moved 'undone' 2) -ne $moveID){throw 'Undo Move content/journal mismatch'}
 Wait 'destination source emptied after undo' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 0 -and (TextControl 1005) -ceq ''}
 Exclusive $pdf
 Step 'undo-move-pdf' @{source_restored=$true;destination_empty=$true;bytes_preserved=$true}
 SwitchSource $fixture.inbox 4
 VerifyPDF '08-preview.pdf' $pdf ($sha.ToLowerInvariant())
 Capture '10-final-pdf-reopen' 'teal'
 if(@([DcmNative]::ListItems($hwnd,$app.Id,1017)).Count){throw 'Undone transactions remain in Processed UI'}
 if(@(Get-ChildItem -LiteralPath $fixture.destination -Force).Count){throw 'Destination contains unexpected files'}
 foreach($key in $txtHashes.Keys) {
  if((Hash (Join-Path $fixture.inbox $key)) -ne $txtHashes[$key]){throw 'Unselected TXT content changed'}
 }
 $null=Journal $pdf $renamed 'undone' 2
 $null=Journal $pdf $moved 'undone' 2
 Step 'final-file-integrity' @{original_pdf_restored=$true;unselected_txt_intact=$true;destination_empty=$true;undone_journal_records=2}
 # Failure/recovery is exercised in the live application, not only the worker.
 # Use the versioned synthetic corrupt/encrypted fixtures from this checkout.
 $damaged=Join-Path $fixture.inbox '11-damaged.pdf'
 $protected=Join-Path $fixture.inbox '12-protected.pdf'
 Copy-Item -LiteralPath (Join-Path $repo 'tests/fixtures/dcm34-damaged.pdf') -Destination $damaged
 $definition=Get-Content -LiteralPath (Join-Path $repo 'tests/fixtures/sumatra-password.json') -Raw | ConvertFrom-Json
 $protectedBytes=[Convert]::FromBase64String($definition.Base64)
 [IO.File]::WriteAllBytes($protected,$protectedBytes)
 if((Hash $protected) -ne $definition.SHA256.ToLowerInvariant()){throw 'Protected PDF fixture checksum mismatch'}
 $badHash=Hash $damaged
 $lockedHash=Hash $protected
 [DcmNative]::ClickAsync($hwnd,$app.Id,1011)
 Wait 'failure fixtures listed' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 6}
 foreach($case in @(
   @{name='11-damaged.pdf';path=$damaged;hash=$badHash},
   @{name='12-protected.pdf';path=$protected;hash=$lockedHash}
 )) {
  List-Selected $case.name
  $selected=$case.path
  Wait ('PDFium refusal for '+$case.name) {
   if(-not(Test-Path $trace)){return $false}
   $matches=@(Get-Content -LiteralPath $trace | ForEach-Object {$_|ConvertFrom-Json} | Where-Object {$_.event -eq 'result-unavailable' -and $_.input_path -eq $selected})
   $matches.Count -gt 0
  } 40
  if((Hash $selected) -ne $case.hash){throw ('Failed PDF modified: '+$case.name)}
  Exclusive $selected
  Step ('refused-'+$case.name) @{worker_refusal_observed=$true;bytes_preserved=$true;exclusive_release=$true}
  VerifyPDF '08-preview.pdf' $pdf ($sha.ToLowerInvariant())
  Capture ('11-recovered-after-'+$case.name.Replace('.pdf','')) 'teal'
  Step ('recovered-after-'+$case.name) @{valid_pdf_rendered_again=$true}
 }
 if(@([DcmNative]::ListItems($hwnd,$app.Id,1017)).Count){throw 'Processed history changed during failed-PDF recovery'}
 if($Persistent -or $DevDefault) {
  $events=@(Get-Content -LiteralPath $trace | ForEach-Object {$_|ConvertFrom-Json})
  $starts=@($events | Where-Object {$_.event -eq 'worker-started' -and $_.renderer_pid -gt 0})
  $unique=@($starts | Select-Object -ExpandProperty renderer_pid -Unique)
  if($starts.Count -lt 6 -or $unique.Count -ne 1){throw ('Persistent worker PID was not reused: renders='+$starts.Count+' PIDs='+($unique -join ','))}
  $fallbacks=@($events | Where-Object {$_.event -match 'persistent-transport-fallback|persistent-fallback-started'})
  if($fallbacks.Count){throw ('Unexpected one-shot fallback during healthy GUI: '+$fallbacks.Count)}
  $serviceChildPID=[int]$unique[0]
  Step 'persistent-process-reuse' @{worker_pid=$serviceChildPID;render_requests=$starts.Count;distinct_pids=1;one_shot_fallbacks=0}
 }
 $receipt.passed=$true
}catch{
 $failure=$_.Exception.Message
 $receipt.error=$failure
 Write-Output ('FAIL='+$failure)
}finally{
 if($null -ne $app){
  try{
   $app.Refresh()
   if(-not $app.HasExited -and $hwnd -ne [IntPtr]::Zero){
    [DcmNative]::CloseAsync($hwnd,$app.Id)
    if(-not $app.WaitForExit(20000)){throw 'Owned close timeout'}
   }
   $receipt.normal_exit=($app.HasExited -and $app.ExitCode -eq 0)
   if(($Persistent -or $DevDefault) -and $serviceChildPID -gt 0 -and $receipt.normal_exit) {
    $until=[DateTime]::UtcNow.AddSeconds(5)
    do {
     $child=Get-Process -Id $serviceChildPID -ErrorAction SilentlyContinue
     if($null -eq $child){break}
     Start-Sleep -Milliseconds 100
    }while([DateTime]::UtcNow -lt $until)
    if($null -ne $child){throw ('Persistent PDFium worker survived owner shutdown: '+$serviceChildPID)}
    Step 'persistent-owned-worker-exited' @{worker_pid=$serviceChildPID;terminated_after_host_close=$true}
   }
  }catch{
   $receipt.passed=$false
   $receipt['close_error']=$_.Exception.Message
  }
  $app.Dispose()
 }
 if(-not $receipt.normal_exit){$receipt.passed=$false}
 $receipt['completed_utc']=[DateTime]::UtcNow.ToString('o')
 [IO.File]::WriteAllText((Join-Path $lab 'receipt.json'),($receipt|ConvertTo-Json -Depth 20))
 Write-Output ('REPORT='+$lab)
 Write-Output ('PASSED='+$receipt.passed)
 Write-Output ('NORMAL_EXIT='+$receipt.normal_exit)
}
if(-not $receipt.passed){exit 1}
