[CmdletBinding()]
param(
  [Parameter(Mandatory=$true)][string]$CandidateExe,
  [Parameter(Mandatory=$true)][string]$WorkerExe,
  [string]$LabRoot='D:\FlowverseReportsRepo'
)
# Test-only. Source is GitHub; Lenovo is a disposable acceptance environment.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$exe=[IO.Path]::GetFullPath($CandidateExe)
$worker=[IO.Path]::GetFullPath($WorkerExe)
if(-not (Test-Path $exe -PathType Leaf) -or -not (Test-Path $worker -PathType Leaf)){throw 'Candidate/worker binaries missing'}
if(-not (Test-Path 'C:\Flowverse\runtime\pdfium\bin\pdfium.dll')){throw 'Managed PDFium runtime missing'}
$lab=Join-Path $LabRoot ('dcm-pdfium-gui-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,7))
$null=New-Item -ItemType Directory -Path $lab
$receipt=[ordered]@{schema=1;feature='DCM-SHARED-PDFIUM-GUI';started_utc=[DateTime]::UtcNow.ToString('o');lab=$lab;commit=(& git -C $repo rev-parse HEAD).Trim();candidate=$exe;worker=$worker;passed=$false;steps=@();captures=@();error=$null;normal_exit=$false;visual_reviewed=$false}
$receipt.candidate_sha256=(Get-FileHash $exe -Algorithm SHA256).Hash
$receipt.worker_sha256=(Get-FileHash $worker -Algorithm SHA256).Hash
$trace=Join-Path $lab 'pdf-trace.jsonl'
$app=$null
$hwnd=[IntPtr]::Zero
$failure=$null
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
function Label {return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,1061),$app.Id)}
function PageLabel {return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,1060),$app.Id)}
function PreviewSnapshot {
 return @([DcmNative]::Snapshot($hwnd,$app.Id) | Where-Object {$_.Visible -and ($_.Text -like 'DCM PDF Preview*' -or $_.Text -like 'Page * of *')})
}
function PageReady([string]$want){
 Wait ('rendered '+$want) {
  $snap=PreviewSnapshot
  @($snap | Where-Object {$_.Text -eq $want}).Count -gt 0
 } 35
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
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-pdfium-wheel-input.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
 . (Join-Path $repo 'tests/native/new-fixtures.ps1')
 $fixture=New-DcmFixtures $lab
 $pdf=Join-Path $fixture.inbox '08-preview.pdf'
 $sha=(Get-FileHash $pdf -Algorithm SHA256).Hash
 $psi=New-Object Diagnostics.ProcessStartInfo
 $psi.FileName=$exe
 $psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"'
 $psi.WorkingDirectory=$lab
 $psi.UseShellExecute=$false
 foreach($kv in @{
  'DCM_PDF_PROVIDER'='pdfium';'DCM_FLOWPDF_WORKER'=$worker;'FILEWORKSPACE_RUNTIME_DIR'='C:\Flowverse\runtime';
  'LOCALAPPDATA'=$fixture.local;'APPDATA'=$fixture.roaming;'TEMP'=$lab;'TMP'=$lab;
  'DCM_PDF_TRACE'=$trace;'DCM_QUICKSORT_BACKUP_DIR'=(Join-Path $lab 'backups')
 }.GetEnumerator()){$psi.EnvironmentVariables[$kv.Key]=$kv.Value}
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
