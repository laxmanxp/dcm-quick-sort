[CmdletBinding()]
param(
  [Parameter(Mandatory=$true)][string]$CandidateExe,
  [Parameter(Mandatory=$true)][string]$WorkerExe,
  [string]$LabRoot='D:\FlowverseReportsRepo',
  [switch]$Persistent,
  [switch]$DevDefault,
  [switch]$WindowsRollback,
  [switch]$VerifyRotation,
  [switch]$VerifyActualFilename,
  [switch]$VerifyTwoPane,
  [switch]$VerifyColumns,
  [switch]$VerifyNativeReport,
  [switch]$NativeReportDiagnostic,
  [switch]$NativeReportMouse,
  [switch]$VerifyNativeContext,
  [switch]$SkipPointerWheel,
  [switch]$SkipScreenshots,
  [ValidateRange(0,160)][int]$StressSwitches=0
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
# Two-pane checks inspect only controls owned by the disposable test process.
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class DcmTwoPaneGeometry {
 [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left,Top,Right,Bottom; }
 [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr hwnd);
 [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hwnd,out RECT rect);
 [DllImport("user32.dll")] public static extern bool MoveWindow(IntPtr hwnd,int x,int y,int width,int height,bool repaint);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern IntPtr SendMessage(IntPtr hwnd,uint message,IntPtr wparam,IntPtr lparam);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern IntPtr FindWindowEx(IntPtr parent,IntPtr after,string className,string caption);
}
'@
function Is-Visible([int]$id){
 return [DcmTwoPaneGeometry]::IsWindowVisible([IntPtr][DcmNative]::Control($hwnd,$app.Id,$id))
}
function Rect([int]$id){
 $r=New-Object DcmTwoPaneGeometry+RECT
 if(-not [DcmTwoPaneGeometry]::GetWindowRect([IntPtr][DcmNative]::Control($hwnd,$app.Id,$id),[ref]$r)){throw ('Missing geometry for control '+$id)}
 return $r
}
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class DcmNativeReportProbe {
 [DllImport("user32.dll", CharSet=CharSet.Unicode, EntryPoint="SendMessageW")]
 public static extern IntPtr SendMessage(IntPtr hWnd,uint msg,IntPtr wParam,IntPtr lParam);
 public static void SelectRow(IntPtr hwnd,int row) {
  IntPtr payload=Marshal.AllocHGlobal(88);
  try {
   for(int i=0;i<88;i++) Marshal.WriteByte(payload,i,0);
   Marshal.WriteInt32(payload,12,0); // clear prior selection/focus
   Marshal.WriteInt32(payload,16,3); // LVIS_SELECTED | LVIS_FOCUSED mask
   SendMessage(hwnd,0x102B,new IntPtr(-1),payload);
   Marshal.WriteInt32(payload,12,3); // select/focus one test row
   SendMessage(hwnd,0x102B,new IntPtr(row),payload);
  } finally {Marshal.FreeHGlobal(payload);}
 }
}
'@
function Select-NativeQueueTab([bool]$history) {
 $tab=[DcmTwoPaneGeometry]::FindWindowEx([IntPtr]$hwnd,[IntPtr]::Zero,'SysTabControl32',$null)
 if($tab -eq [IntPtr]::Zero){throw 'Native Win32 SysTabControl32 not found'}
 $x=$(if($history){125}else{35})
 $lparam=[IntPtr](($x -band 0xffff) -bor (12 -shl 16))
 # Send actual tab mouse messages, exercising TCN_SELCHANGE and focus.
 [void][DcmTwoPaneGeometry]::SendMessage($tab,0x0201,[IntPtr]1,$lparam)
 [void][DcmTwoPaneGeometry]::SendMessage($tab,0x0202,[IntPtr]::Zero,$lparam)
}
function Verify-TwoPaneTab([bool]$history){
 Owned
 if((Is-Visible 1016) -eq $history){throw 'Queue list visibility disagrees with active tab'}
 if((Is-Visible 1017) -ne $history -or (Is-Visible 1046) -ne $history){throw 'History list/search visibility disagrees with active tab'}
 if(-not(Is-Visible 1008) -or -not(Is-Visible 1048)){throw 'Undo controls disappeared when switching tabs'}
 }
function Verify-TwoPaneGeometry([string]$label){
 $queue=Rect 1016
 $previewHwnd=[DcmTwoPaneGeometry]::FindWindowEx([IntPtr]$hwnd,[IntPtr]::Zero,'QuickSortPreview',$null)
 if($previewHwnd -eq [IntPtr]::Zero){throw 'Native preview child HWND missing'}
 $preview=New-Object DcmTwoPaneGeometry+RECT
 if(-not [DcmTwoPaneGeometry]::GetWindowRect($previewHwnd,[ref]$preview)){throw 'Preview geometry missing'}
 $main=New-Object DcmTwoPaneGeometry+RECT
 if(-not [DcmTwoPaneGeometry]::GetWindowRect([IntPtr]$hwnd,[ref]$main)){throw 'Main window geometry missing'}
 $outerWidth=$main.Right-$main.Left
 $qWidth=$queue.Right-$queue.Left
 $pWidth=$preview.Right-$preview.Left
 if($qWidth -lt 230 -or $qWidth -gt 470){throw ('Queue pane width out of bounds: '+$qWidth)}
 if($pWidth -lt 420 -or $queue.Right -ge $preview.Left){throw ('Two-pane geometry overlaps or preview too narrow: '+$pWidth)}
 Step ('two-pane-geometry-'+$label) @{outer_width=$outerWidth;queue_width=$qWidth;preview_width=$pWidth;no_overlap=$true}
}
function Assert-ActualFile([string]$filename,[string]$extension) {
 # Wait's own $name parameter must not shadow the expected filename.
 $expectedName=$filename
 $expectedType='Type: '+$extension
 Wait ('actual filename '+$expectedName) {
  (TextControl 1049) -ceq $expectedName -and (TextControl 1050) -ceq $expectedType
 } 20
 Step ('actual-filename-'+$extension) @{filename=(TextControl 1049);type=(TextControl 1050);
   rename_stem=(TextControl 1005);extension_visible=$true;selection=$expectedName}
}
function Assert-ActualFilenameLayout {
 $file=[DcmNative]::Control($hwnd,$app.Id,1049)
 $kind=[DcmNative]::Control($hwnd,$app.Id,1050)
 $rename=[DcmNative]::Control($hwnd,$app.Id,1005)
 $fileRect=[DcmNative+RECT]::new()
 $typeRect=[DcmNative+RECT]::new()
 $renameRect=[DcmNative+RECT]::new()
 if(-not [DcmNative]::GetWindowRect($file,[ref]$fileRect) -or
    -not [DcmNative]::GetWindowRect($kind,[ref]$typeRect) -or
    -not [DcmNative]::GetWindowRect($rename,[ref]$renameRect)){
  throw 'Actual filename layout rectangles unavailable'
 }
 if($fileRect.Right + 4 -gt $typeRect.Left){throw 'Filename overlaps file-type label'}
 if($fileRect.Bottom + 3 -gt $renameRect.Top){throw 'Actual filename overlaps editable New Name row'}
 if($fileRect.Right-$fileRect.Left -lt 180){throw 'Actual filename field is too narrow'}
 $style=[DcmFilenameStyle]::GetWindowLongPtr($file,-16).ToInt64()
 if(($style -band 0x800) -eq 0){throw 'Actual filename edit is unexpectedly writable'}
 $snapshot=@([DcmNative]::Snapshot($hwnd,$app.Id))
 if(@($snapshot|Where-Object {$_.Text -eq 'Actual File:' -and $_.Visible}).Count -ne 1){throw 'Actual File label not visible'}
 Step 'actual-filename-layout' @{readonly=$true;no_overlap=$true;file_width=$fileRect.Right-$fileRect.Left;
   file_left=$fileRect.Left;file_right=$fileRect.Right;type_left=$typeRect.Left;rename_top=$renameRect.Top}
}
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
function RotatedLabel([int]$page,[int]$angle){
 return ('Page '+$page+' of 2 '+[char]0x00B7+' '+$angle+[char]0x00B0)
}
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
 if($SkipScreenshots){
  Step ('native-capture-skipped-'+$name) 'Desktop has another foreground window; filename text/layout checks do not require screenshots.'
  return
 }
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
 Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;public static class DcmFilenameStyle{[DllImport("user32.dll",EntryPoint="GetWindowLongPtrW")]public static extern IntPtr GetWindowLongPtr(IntPtr h,int index);}'
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
 if($VerifyNativeReport){$psi.RedirectStandardError=$true}
 foreach($kv in @{
  'FILEWORKSPACE_RUNTIME_DIR'='C:\Flowverse\runtime';
  'LOCALAPPDATA'=$fixture.local;'APPDATA'=$fixture.roaming;'TEMP'=$lab;'TMP'=$lab;
  'DCM_PDF_TRACE'=$trace;'DCM_QUICKSORT_BACKUP_DIR'=(Join-Path $lab 'backups')
 }.GetEnumerator()){$psi.EnvironmentVariables[$kv.Key]=$kv.Value}
 foreach($key in @('DCM_PDF_PROVIDER','DCM_FLOWPDF_WORKER','DCM_FLOWPDF_PERSISTENT')){$psi.EnvironmentVariables.Remove($key)}
 if($VerifyNativeReport){$psi.EnvironmentVariables['DCM_NATIVE_QUEUE_LISTVIEW']='1'}
 if($NativeReportDiagnostic -or $VerifyNativeContext){$psi.EnvironmentVariables['DCM_NATIVE_QUEUE_TRACE']=(Join-Path $lab 'native-report-events.log')}
 if($VerifyNativeContext){$psi.EnvironmentVariables['DCM_NATIVE_QUEUE_CONTEXT_PROBE']='1'}
 if($NativeReportDiagnostic){$psi.EnvironmentVariables['DCM_NATIVE_QUEUE_DIAG_NO_SYNC']='1'}
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
 if($VerifyNativeReport){
  $native=[DcmTwoPaneGeometry]::FindWindowEx([IntPtr]$hwnd,[IntPtr]::Zero,'SysListView32',$null)
  if($native -eq [IntPtr]::Zero -or -not [DcmTwoPaneGeometry]::IsWindowVisible($native)){throw 'Native ListView report is not visible'}
  $header=[DcmTwoPaneGeometry]::FindWindowEx($native,[IntPtr]::Zero,'SysHeader32',$null)
  if($header -eq [IntPtr]::Zero){throw 'Native ListView header is missing'}
  $columnCount=[DcmTwoPaneGeometry]::SendMessage($header,0x1200,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
  $itemCount=[DcmTwoPaneGeometry]::SendMessage($native,0x1004,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
  if($columnCount -ne 4 -or $itemCount -ne 4){throw ('Native ListView columns/items mismatch '+$columnCount+'/'+$itemCount)}
  Step 'native-report-four-columns' @{class='SysListView32';header_class='SysHeader32';columns=$columnCount;items=$itemCount;visible=$true}
  Step 'native-report-before-select' @{diagnostic=$NativeReportDiagnostic}
  if($NativeReportDiagnostic -or $NativeReportMouse){
   # A physical-style row click is distinct from LVM_SETITEMSTATE.
   $point=[IntPtr]((69 -shl 16) -bor 65)
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0201,[IntPtr]1,$point)
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0202,[IntPtr]::Zero,$point)
   Step 'native-report-after-mouse-click' @{diagnostic=$true}
   Start-Sleep -Milliseconds 850
   $app.Refresh()
   if($app.HasExited){throw 'Native report exited after physical-style row click'}
  }else{
   [DcmNativeReportProbe]::SelectRow($native,2)
  }
  $selectedNative=[DcmTwoPaneGeometry]::SendMessage($native,0x100C,[IntPtr](-1),[IntPtr]2).ToInt64()
  Step 'native-report-after-select' @{diagnostic=$NativeReportDiagnostic;selected_index=$selectedNative;current_name=(TextControl 1005)}
  $actualIndex=[int]$selectedNative
  $actualRows=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
  if($actualIndex -lt 0 -or $actualIndex -ge $actualRows.Count){throw ('Native report selection invalid index: '+$actualIndex)}
  $expectedStem=[IO.Path]::GetFileNameWithoutExtension($actualRows[$actualIndex])
  Wait 'Native report selection mirrored into QuickSort workflow' { (TextControl 1005) -ceq $expectedStem }
  Step 'native-report-selection-adapter' @{selected_name=(TextControl 1005);selected_index=$actualIndex;preview_selection_updated=$true}
  # Clicking the native SysHeader32 (not a surrogate button) should sort the
  # same underlying file array used by Rename/Move and all other operations.
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0201,[IntPtr]1,[IntPtr]0x000C000A)
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0202,[IntPtr]::Zero,[IntPtr]0x000C000A)
  $nameOrder=@('01-alpha.txt','02-bravo.txt','03-charlie.txt','08-preview.pdf')
  Wait 'Native ListView Name header sorts ascending' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($nameOrder -join '|')}
  Step 'native-report-header-sort' @{name_ascending=$true;native_header_clicked=$true}
  $updatedPoint=[IntPtr]((12 -shl 16) -bor 380)
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0201,[IntPtr]1,$updatedPoint)
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0202,[IntPtr]::Zero,$updatedPoint)
  Wait 'Native Updated header selected' {(TextControl 1045) -eq 'Date Updated'}
  $updatedFirst=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0201,[IntPtr]1,$updatedPoint)
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0202,[IntPtr]::Zero,$updatedPoint)
  [array]::Reverse($updatedFirst)
  Wait 'Native Updated header reverses order' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($updatedFirst -join '|')}
  Step 'native-report-updated-header-sort' @{updated_ascending_and_descending=$true}
  if($VerifyNativeContext){
   $rc=New-Object DcmTwoPaneGeometry+RECT
   if(-not [DcmTwoPaneGeometry]::GetWindowRect($native,[ref]$rc)){throw 'Cannot locate native table for context test'}
   $cx=$rc.Left+65; $cy=$rc.Top+69
   $packed=[IntPtr]((($cy -band 0xffff) -shl 16) -bor ($cx -band 0xffff))
   [void][DcmTwoPaneGeometry]::SendMessage([IntPtr]$hwnd,0x007B,$native,$packed)
   Wait 'Native context record captured' {(Test-Path (Join-Path $lab 'native-report-events.log')) -and ((Get-Content (Join-Path $lab 'native-report-events.log') -Tail 3) -match 'context-selected').Count -gt 0}
   $contextLine=(Get-Content (Join-Path $lab 'native-report-events.log') | Where-Object {$_ -like 'context-selected*'} | Select-Object -Last 1)
   Step 'native-report-context-hit' @{event=$contextLine;native_control=$true}
  }
 }
 if($VerifyColumns){
  foreach($id in @(1066,1067,1068,1069)){if(-not(Is-Visible $id)){throw ('Queue header missing: '+$id)}}
  $expected=@('01-alpha.txt','02-bravo.txt','03-charlie.txt','08-preview.pdf')
  [DcmNative]::ClickAsync($hwnd,$app.Id,1066)
  Wait 'Name ascending by header' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($expected -join '|')}
  $queueControl=[IntPtr][DcmNative]::Control($hwnd,$app.Id,1016)
  [void][DcmTwoPaneGeometry]::SendMessage($queueControl,0x185,[IntPtr]::Zero,[IntPtr](-1))
  [void][DcmTwoPaneGeometry]::SendMessage($queueControl,0x185,[IntPtr]1,[IntPtr]0)
  [void][DcmTwoPaneGeometry]::SendMessage($queueControl,0x185,[IntPtr]1,[IntPtr]3)
  $selectedBefore=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)|ForEach-Object {$expected[$_] }|Sort-Object)
  if($selectedBefore.Count -ne 2){throw 'Multi-select fixture setup failed'}
  [DcmNative]::ClickAsync($hwnd,$app.Id,1066)
  [array]::Reverse($expected)
  Wait 'Name descending by second click' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($expected -join '|')}
  $selectedAfter=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)|ForEach-Object {$expected[$_]}|Sort-Object)
  if(($selectedBefore -join '|') -cne ($selectedAfter -join '|')){throw ('Multi-selection not preserved by header sort: before='+($selectedBefore -join ',')+' after='+($selectedAfter -join ','))}
  Step 'queue-column-multiselection-preserved' @{selected_files=$selectedAfter;count=$selectedAfter.Count}
  [DcmNative]::ClickAsync($hwnd,$app.Id,1067)
  $extOrder=@('08-preview.pdf','01-alpha.txt','02-bravo.txt','03-charlie.txt')
  Wait 'Extension ascending by header' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($extOrder -join '|')}
  [DcmNative]::ClickAsync($hwnd,$app.Id,1067)
  [array]::Reverse($extOrder)
  Wait 'Extension descending by second click' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($extOrder -join '|')}
  [DcmNative]::ClickAsync($hwnd,$app.Id,1068)
  $createdOrder=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
  [DcmNative]::ClickAsync($hwnd,$app.Id,1068)
  [array]::Reverse($createdOrder)
  Wait 'Created reverse on second click' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($createdOrder -join '|')}
  [DcmNative]::ClickAsync($hwnd,$app.Id,1068)
  [DcmNative]::ClickAsync($hwnd,$app.Id,1069)
  $updatedOrder=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
  [DcmNative]::ClickAsync($hwnd,$app.Id,1069)
  [array]::Reverse($updatedOrder)
  Wait 'Updated reverse on second click' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($updatedOrder -join '|')}
  [DcmNative]::ClickAsync($hwnd,$app.Id,1069)
  Step 'queue-column-header-sort' @{name_ascending=$true;name_descending=$true;extension_ascending=$true;extension_descending=$true;created_toggle=$true;updated_toggle=$true}
 }
 if($VerifyTwoPane){
  Verify-TwoPaneTab $false
  Verify-TwoPaneGeometry 'normal'
  $area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
  if(-not [DcmTwoPaneGeometry]::MoveWindow([IntPtr]$hwnd,$area.Left+10,$area.Top+10,920,740,$true)){throw 'Unable to resize owned QuickSort window'}
  Start-Sleep -Milliseconds 300
  Verify-TwoPaneGeometry 'narrow-920'
  if(-not [DcmTwoPaneGeometry]::MoveWindow([IntPtr]$hwnd,$area.Left+10,$area.Top+10,800,720,$true)){throw 'Unable to resize owned QuickSort window to 800'}
  Start-Sleep -Milliseconds 300
  Verify-TwoPaneGeometry 'minimum-clamped-800-request'
  Select-NativeQueueTab $true
  Wait 'History visible at 800' {(Is-Visible 1017) -and -not (Is-Visible 1016)}
  Verify-TwoPaneTab $true
  Select-NativeQueueTab $false
  Wait 'Queue visible at 800' {(Is-Visible 1016) -and -not (Is-Visible 1017)}
  Verify-TwoPaneTab $false
  Step 'two-pane-compact-tab-toggle' @{width_requested=800;queue_restored=$true;history_available=$true}
  if(-not [DcmTwoPaneGeometry]::MoveWindow([IntPtr]$hwnd,$area.Left+10,$area.Top+10,[Math]::Min(1350,$area.Width-20),[Math]::Min(820,$area.Height-20),$true)){throw 'Unable to restore owned QuickSort window size'}
  Start-Sleep -Milliseconds 300
  Verify-TwoPaneGeometry 'wide'
  Select-NativeQueueTab $true
  Wait 'History tab visible' {(Is-Visible 1017) -and -not (Is-Visible 1016)}
  Verify-TwoPaneTab $true
  Select-NativeQueueTab $false
  Wait 'Queue tab visible' {(Is-Visible 1016) -and -not (Is-Visible 1017)}
  Verify-TwoPaneTab $false
  Step 'two-pane-queue-history-toggle' @{queue_visible=$true;history_visible_when_selected=$true;undo_always_visible=$true}
 }
 List-Selected '08-preview.pdf'
 if($VerifyActualFilename){
  Assert-ActualFile '08-preview.pdf' 'PDF'
  Assert-ActualFilenameLayout
 }
 PageReady 'Page 1 of 2'
 if($VerifyColumns){Capture 'queue-three-columns' 'teal'}
 Step 'page-one-controls' (PreviewSnapshot)
 Capture '01-page-one' 'teal'
 [DcmNative]::ClickAsync($hwnd,$app.Id,1061)
 PageReady 'Page 2 of 2'
 Step 'page-two-controls' (PreviewSnapshot)
 Capture '02-page-two' 'orange'
 [DcmNative]::ClickAsync($hwnd,$app.Id,1060)
 PageReady 'Page 1 of 2'
 Capture '03-page-one-return' 'teal'
 if($VerifyRotation) {
  $originalHash=Hash $pdf
  foreach($angle in @(90,180,270)) {
   $previous=@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}).Count
   [DcmNative]::ClickAsync($hwnd,$app.Id,1062)
   $expected=RotatedLabel 1 $angle
   Wait ('PDF clockwise rotation '+$angle) {
    $label=@(PreviewSnapshot|Where-Object {$_.Text -eq $expected})
    if($label.Count -ne 1){return $false}
    $applied=@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}|ForEach-Object {$_|ConvertFrom-Json})
    if($applied.Count -le $previous){return $false}
    $last=$applied[-1]
    return ($last.rotation -eq $angle -and $last.input_path -eq $pdf)
   } 50
   Capture ('rotate-'+$angle) 'teal'
   Step ('rotation-'+$angle) @{angle=$angle;page=1;source_hash_intact=((Hash $pdf) -eq $originalHash)}
   if($angle -eq 90) {
    $nextFromGeneration=(@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}|ForEach-Object {$_|ConvertFrom-Json})[-1]).generation
    [DcmNative]::ClickAsync($hwnd,$app.Id,1061)
    Wait 'rotated page-two navigation' {
     $labels=@(PreviewSnapshot|Where-Object {$_.Text -eq (RotatedLabel 2 90)})
     $applied=@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}|ForEach-Object {$_|ConvertFrom-Json})
     return ($labels.Count -eq 1 -and $applied[-1].rotation -eq 90 -and $applied[-1].generation -gt $nextFromGeneration)
    } 50
    Capture 'rotate-90-next-page' 'orange'
    $returnFromGeneration=(@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}|ForEach-Object {$_|ConvertFrom-Json})[-1]).generation
    [DcmNative]::ClickAsync($hwnd,$app.Id,1060)
    Wait 'rotated previous-page navigation' {
     $labels=@(PreviewSnapshot|Where-Object {$_.Text -eq (RotatedLabel 1 90)})
     $applied=@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}|ForEach-Object {$_|ConvertFrom-Json})
     return ($labels.Count -eq 1 -and $applied[-1].rotation -eq 90 -and $applied[-1].generation -gt $returnFromGeneration)
    } 50
    Capture 'rotate-90-previous-page' 'teal'
   }
  }
  [DcmNative]::ClickAsync($hwnd,$app.Id,1062)
  PageReady 'Page 1 of 2'
  Capture 'rotate-reset-0' 'teal'
  # Each owner-drawn icon must be a separate accessible, enabled native button.
  foreach($button in @(@{id=1063;label='Rotate left 90 degrees'},@{id=1062;label='Rotate right 90 degrees'})){
   $control=[DcmNative]::Control($hwnd,$app.Id,$button.id)
   if(-not [DcmNative]::IsWindowEnabled($control)){throw ('Rotation button disabled: '+$button.id)}
   $snapshot=@([DcmNative]::Snapshot($hwnd,$app.Id)|Where-Object {$_.Text -eq $button.label -and $_.Visible})
   if($snapshot.Count -ne 1){throw ('Missing accessible rotation icon: '+$button.label)}
  }
  Step 'two-rotation-icon-controls' @{rotate_left_id=1063;rotate_right_id=1062;both_visible=$true;both_enabled=$true}
  foreach($angle in @(270,180,90,0)) {
   $previous=@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}).Count
   [DcmNative]::ClickAsync($hwnd,$app.Id,1063)
   $expected=if($angle -eq 0){'Page 1 of 2'}else{RotatedLabel 1 $angle}
   Wait ('PDF counterclockwise rotation '+$angle) {
    $label=@(PreviewSnapshot|Where-Object {$_.Text -eq $expected})
    if($label.Count -ne 1){return $false}
    $applied=@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}|ForEach-Object {$_|ConvertFrom-Json})
    if($applied.Count -le $previous){return $false}
    $last=$applied[-1]
    # Trace JSON omits rotation when the angle returns to zero.
    $renderRotation=if($last.PSObject.Properties['rotation']){[int]$last.rotation}else{0}
    return ($renderRotation -eq $angle -and $last.input_path -eq $pdf)
   } 50
   Capture ('rotate-left-'+$angle) 'teal'
   Step ('counterclockwise-rotation-'+$angle) @{angle=$angle;page=1;source_hash_intact=((Hash $pdf) -eq $originalHash)}
  }
  if((Hash $pdf) -ne $originalHash){throw 'Preview rotation modified input PDF bytes'}
  Step 'preview-rotation-cycle' @{clockwise_angles=@(0,90,180,270,0);counterclockwise_angles=@(0,270,180,90,0);page_navigation_preserved=$true;source_file_untouched=$true}
 }
 # Native toolbar zoom; wheel mouse-location acceptance is a separate gate.
 [DcmNative]::ClickAsync($hwnd,$app.Id,1022)
 Wait 'zoom render success' {
  $events=if(Test-Path $trace){@(Get-Content $trace | Where-Object {$_ -match 'result-applied'})}else{@()}
  $events.Count -ge 4
 } 40
 Capture '04-after-zoom-in' 'teal'
 Step 'toolbar-zoom-success' 'PDFium result applied after native Zoom In control'
 if(-not $SkipPointerWheel){
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
 }else{
  Step 'pointer-wheel-not-in-scope' 'Separate cursor-wheel acceptance previously qualified; this run checks filename identity and file lifecycle.'
 }
 if($StressSwitches -gt 0){
  if(-not($Persistent -or $DevDefault)){throw 'Persistent GUI soak requires the PDFium provider'}
  $soakSamples=@()
  $originalPdfHash=Hash $pdf
  function Median-Soak([long[]]$numbers){
   if($numbers.Count -lt 1){return 0L}
   $sorted=@($numbers|Sort-Object)
   return [long]$sorted[[int][Math]::Floor(($sorted.Count-1)/2)]
  }
  for($soakIndex=0;$soakIndex -lt $StressSwitches;$soakIndex++){
   List-Selected '01-alpha.txt'
   List-Selected '08-preview.pdf'
   PageReady 'Page 1 of 2'
   $events=@(Get-Content -LiteralPath $trace|ForEach-Object {$_|ConvertFrom-Json})
   $starts=@($events|Where-Object {$_.event -eq 'worker-started' -and $_.renderer_pid -gt 0})
   if(-not $starts.Count){throw 'No owned persistent PDFium worker during GUI soak'}
   $workerPid=[int]$starts[-1].renderer_pid
   $process=Get-Process -Id $workerPid -ErrorAction Stop
   $app.Refresh()
   $soakSamples+=@([ordered]@{cycle=$soakIndex;worker_pid=$workerPid;
    host_private_bytes=[long]$app.PrivateMemorySize64;host_handles=[int]$app.HandleCount;
    worker_private_bytes=[long]$process.PrivateMemorySize64;worker_handles=[int]$process.HandleCount})
   if(($soakIndex+1)%20 -eq 0){Write-Output ('GUI_STRESS_SWITCHES='+($soakIndex+1)+'/'+$StressSwitches+' host_MiB='+[Math]::Round($app.PrivateMemorySize64/1MB,1)+' worker_MiB='+[Math]::Round($process.PrivateMemorySize64/1MB,1))}
  }
  if((Hash $pdf) -ne $originalPdfHash){throw 'PDF contents changed during switch endurance'}
  $unique=@($soakSamples|ForEach-Object {$_.worker_pid}|Sort-Object -Unique)
  if($unique.Count -ne 1){throw ('Unexpected worker restart during healthy switching: '+($unique -join ','))}
  $events=@(Get-Content -LiteralPath $trace|ForEach-Object {$_|ConvertFrom-Json})
  if(@($events|Where-Object {$_.event -match 'persistent-transport-fallback|persistent-fallback-started'}).Count){
   throw 'GUI switch endurance triggered unplanned one-shot fallback'
  }
  $first=@($soakSamples|Select-Object -Skip 8 -First 12)
  $last=@($soakSamples|Select-Object -Last 12)
  $hostDelta=(Median-Soak ([long[]]@($last|ForEach-Object {$_.host_private_bytes})))-(Median-Soak ([long[]]@($first|ForEach-Object {$_.host_private_bytes})))
  $workerDelta=(Median-Soak ([long[]]@($last|ForEach-Object {$_.worker_private_bytes})))-(Median-Soak ([long[]]@($first|ForEach-Object {$_.worker_private_bytes})))
  $hostHandleDelta=(Median-Soak ([long[]]@($last|ForEach-Object {$_.host_handles})))-(Median-Soak ([long[]]@($first|ForEach-Object {$_.host_handles})))
  $workerHandleDelta=(Median-Soak ([long[]]@($last|ForEach-Object {$_.worker_handles})))-(Median-Soak ([long[]]@($first|ForEach-Object {$_.worker_handles})))
  $receipt['stress_samples']=$soakSamples
  Step 'gui-pdf-switch-endurance' @{cycles=$StressSwitches;worker_pid=$unique[0];
   host_private_delta_bytes=$hostDelta;worker_private_delta_bytes=$workerDelta;
   host_handle_delta=$hostHandleDelta;worker_handle_delta=$workerHandleDelta;
   source_hash_intact=$true;one_shot_fallbacks=0;peak_memory_is_sampled=$true}
  if($hostDelta -gt 48MB -or $workerDelta -gt 32MB -or $hostHandleDelta -gt 48 -or $workerHandleDelta -gt 32){
   throw ('Potential GUI/worker sustained resource growth: hostMiB='+[Math]::Round($hostDelta/1MB,1)+' workerMiB='+[Math]::Round($workerDelta/1MB,1)+' hostHandles='+$hostHandleDelta+' workerHandles='+$workerHandleDelta)
  }
 }
 List-Selected '01-alpha.txt'
 if($VerifyActualFilename){Assert-ActualFile '01-alpha.txt' 'TXT'}
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
 if($VerifyTwoPane){
  Select-NativeQueueTab $true
  Wait 'History after rename' {(Is-Visible 1017) -and @([DcmNative]::ListItems($hwnd,$app.Id,1017)).Count -gt 0}
  Verify-TwoPaneTab $true
  [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1046,'not-a-match-123456')
  Wait 'History filtered to none' {@([DcmNative]::ListItems($hwnd,$app.Id,1017)).Count -eq 0}
  [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1046,'')
  Wait 'History search cleared' {@([DcmNative]::ListItems($hwnd,$app.Id,1017)).Count -gt 0}
  Step 'two-pane-history-search' @{history_records_restored=$true;search_filters=$true}
  Capture 'two-pane-history-after-rename' 'teal'
  Select-NativeQueueTab $false
  Wait 'Queue restored after history' {(Is-Visible 1016) -and -not (Is-Visible 1017)}
 }

 Wait 'renamed selection' {(TextControl 1005) -eq 'undo-preview'}
 if($VerifyActualFilename){Assert-ActualFile 'undo-preview.pdf' 'PDF'}
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
 if($VerifyActualFilename){
  Wait 'empty source actual filename cleared' {(TextControl 1049) -eq 'No file selected' -and (TextControl 1050) -eq ''}
  Step 'actual-filename-cleared' @{actual_name=(TextControl 1049);type=(TextControl 1050)}
 }
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
 if($VerifyActualFilename){
  # Use real OpenXML samples, never arbitrary renamed empty files.
  $docx=Join-Path $fixture.inbox '13-Letter.DOCX'
  $xlsx=Join-Path $fixture.inbox '14-Budget.xlsx'
  Copy-Item -LiteralPath (Join-Path $repo 'tests/fixtures/dcm33-legacy.source.docx') -Destination $docx
  Copy-Item -LiteralPath (Join-Path $repo 'tests/fixtures/dcm33-legacy-xls.source.xlsx') -Destination $xlsx
  [DcmNative]::ClickAsync($hwnd,$app.Id,1011)
  Wait 'Office file fixtures listed' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 8}
  List-Selected '13-Letter.DOCX'
  Assert-ActualFile '13-Letter.DOCX' 'DOCX'
  List-Selected '14-Budget.xlsx'
  Assert-ActualFile '14-Budget.xlsx' 'XLSX'
  List-Selected '08-preview.pdf'
  Assert-ActualFile '08-preview.pdf' 'PDF'
  Step 'actual-filename-office-switching' @{docx=$true;xlsx=$true;pdf=$true;read_only_field=$true}
 }
 if($Persistent -or $DevDefault) {
  $events=@(Get-Content -LiteralPath $trace | ForEach-Object {$_|ConvertFrom-Json})
  $starts=@($events | Where-Object {$_.event -eq 'worker-started' -and $_.renderer_pid -gt 0})
  $unique=@($starts | Select-Object -ExpandProperty renderer_pid -Unique)
  if($starts.Count -lt 6){throw ('Too few persistent PDFium render requests: '+$starts.Count)}
  . (Join-Path $PSScriptRoot 'pdfium-worker-transition-contract.ps1')
  $contract=Assert-PDFiumWorkerTransitions -Events $events
  $restarts=[int]$contract.cancel_restarts
  $fallbacks=@($events | Where-Object {$_.event -match 'persistent-transport-fallback|persistent-fallback-started'})
  if($fallbacks.Count){throw ('Unexpected one-shot fallback during healthy GUI: '+$fallbacks.Count)}
  # Only the final owned worker may still be running at normal shutdown.
  $serviceChildPID=[int]$starts[-1].renderer_pid
  Step 'persistent-process-reuse' @{worker_pid=$serviceChildPID;render_requests=$starts.Count;distinct_pids=$unique.Count;cancel_restarts=$restarts;one_shot_fallbacks=0}
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
   if($VerifyNativeReport -and $app.HasExited){
    $stderrText=$app.StandardError.ReadToEnd()
    if($stderrText){[IO.File]::WriteAllText((Join-Path $lab 'native-report-stderr.txt'),$stderrText); Step 'native-report-process-stderr' @{length=$stderrText.Length;head=$stderrText.Substring(0,[Math]::Min(500,$stderrText.Length))}}
   }
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
