[CmdletBinding()]
param(
  [Parameter(Mandatory=$true)][string]$CandidateExe,
  [Parameter(Mandatory=$true)][string]$WorkerExe,
  [string]$LabRoot='D:\FlowverseReportsRepo',
  [switch]$Persistent,
  [switch]$DevDefault,
  [switch]$WindowsRollback,
  [switch]$VerifyRotation,
  [switch]$VerifyNarrowToolbar,
  [switch]$VerifyCodeText,
  [switch]$VerifyMoveGating,
  [switch]$VerifyActualFilename,
  [switch]$VerifyTwoPane,
  [switch]$VerifySplitter,
  [switch]$VerifyNativeSplitterOnly,
  [switch]$VerifyColumns,
  [switch]$VerifyNativeReport,
  [switch]$VerifyPDFPicker,
  [string]$SumatraRuntime='',
  [switch]$LegacyQueue,
  [switch]$NativeReportDiagnostic,
  [switch]$NativeReportMouse,
  [switch]$VerifyNativeMulti,
  [switch]$VerifyRefreshSelection,
  [switch]$VerifyPreviewCoalescing,
  [switch]$VerifyNativeRangeKeyboard,
  [switch]$SkipPointerWheel,
  [switch]$SkipScreenshots,
  [ValidateRange(0,600)][int]$KeyboardSoakSteps=0,
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
if($DevDefault){
 # DevDefault intentionally clears explicit worker configuration. Require
 # the exact sidecar layout that production's automatic discovery expects.
 $sidecar=Join-Path ([IO.Path]::GetDirectoryName($exe)) 'flowpdf-preview-worker-persistent.exe'
 if(-not (Test-Path -LiteralPath $sidecar -PathType Leaf)){
  throw 'DevDefault requires a sidecar PDFium worker beside the candidate EXE'
 }
 if((Get-FileHash -LiteralPath $sidecar -Algorithm SHA256).Hash -ne (Get-FileHash -LiteralPath $worker -Algorithm SHA256).Hash){
  throw 'DevDefault worker sidecar does not match the qualified worker'
 }
}
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
 if($VerifyNativeReport){
  Step 'native-operation-select-request' @{name=$name;index=$index}
  $nativeList=[DcmTwoPaneGeometry]::FindWindowEx([IntPtr]$hwnd,[IntPtr]::Zero,'SysListView32',$null)
  if($nativeList -eq [IntPtr]::Zero -or -not [DcmNativeReportProbe]::NavigateToRow($nativeList,$index)){throw ('Native ListView cannot navigate to '+$name)}
  Wait ('native focused row '+$index) {[DcmTwoPaneGeometry]::SendMessage($nativeList,0x100C,[IntPtr](-1),[IntPtr]1).ToInt64() -eq $index}
 }else{
  [DcmUndoInput]::Select($hwnd,$app.Id,$index)
 }
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
 [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)] public static extern IntPtr SendMessageTimeout(IntPtr hwnd,uint message,IntPtr wparam,IntPtr lparam,uint flags,uint timeout,out IntPtr result);
 [DllImport("user32.dll", SetLastError=true)] public static extern bool PostMessage(IntPtr hwnd,uint message,IntPtr wparam,IntPtr lparam);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern IntPtr FindWindowEx(IntPtr parent,IntPtr after,string className,string caption);
}
'@
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public struct DcmSplitterPoint {
 public int X,Y;
 [DllImport("user32.dll")] public static extern bool ScreenToClient(IntPtr hwnd,ref DcmSplitterPoint point);
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
public static class DcmPDFPickerProbe {
 [DllImport("user32.dll")] public static extern IntPtr GetMenu(IntPtr hwnd);
 [DllImport("user32.dll")] public static extern IntPtr GetSubMenu(IntPtr menu,int position);
 [DllImport("user32.dll")] public static extern int GetMenuItemCount(IntPtr menu);
 [DllImport("user32.dll")] public static extern uint GetMenuState(IntPtr menu,uint item,uint flags);
 [DllImport("user32.dll",SetLastError=true)] public static extern bool PostMessage(IntPtr hwnd,uint msg,IntPtr wparam,IntPtr lparam);
 public static bool IsChecked(IntPtr menu,uint id) {
  if(menu==IntPtr.Zero)return false;
  uint state=GetMenuState(menu,id,0);
  if(state!=0xffffffff)return (state&8)!=0;
  int count=GetMenuItemCount(menu);
  for(int i=0;i<count;i++)if(IsChecked(GetSubMenu(menu,i),id))return true;
  return false;
 }
}
public static class DcmNativeReportProbe {
 [DllImport("user32.dll", CharSet=CharSet.Unicode, EntryPoint="SendMessageW")]
 public static extern IntPtr SendMessage(IntPtr hWnd,uint msg,IntPtr wParam,IntPtr lParam);
 [DllImport("user32.dll", SetLastError=true)] public static extern bool PostMessage(IntPtr hwnd,uint msg,IntPtr wParam,IntPtr lParam);
 public static bool NavigateToRow(IntPtr hwnd,int row) {
  if(row<0||row>600)return false;
  // Keyboard interaction uses native ListView selection and scrolling,
  // independent of test display DPI, header height and old scroll offsets.
  if(!PostMessage(hwnd,0x0100,new IntPtr(0x24),IntPtr.Zero))return false;
  if(!PostMessage(hwnd,0x0101,new IntPtr(0x24),IntPtr.Zero))return false;
  for(int i=0;i<row;i++){
   if(!PostMessage(hwnd,0x0100,new IntPtr(0x28),IntPtr.Zero))return false;
   if(!PostMessage(hwnd,0x0101,new IntPtr(0x28),IntPtr.Zero))return false;
  }
  return true;
 }
 public static bool ClickRow(IntPtr hwnd,int row) {
  // Fixture rows are visible; do not send synchronous mouse/scroll messages
  // across processes while the Go UI thread is busy with previews.
  if(row<0||row>16)return false;
  int x=65;
  int y=45+row*19; // center of report-view rows at tested Win32 DPI
  IntPtr point=new IntPtr(((y&0xffff)<<16)|(x&0xffff));
  return PostMessage(hwnd,0x0201,new IntPtr(1),point)
      && PostMessage(hwnd,0x0202,IntPtr.Zero,point);
 }
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
function QueueIsVisible {return (Is-Visible $(if($LegacyQueue){1016}else{1071}))}
function Verify-TwoPaneTab([bool]$history){
 Owned
 if((QueueIsVisible) -eq $history){throw 'Queue list visibility disagrees with active tab'}
 if((Is-Visible 1017) -ne $history -or (Is-Visible 1046) -ne $history){throw 'History list/search visibility disagrees with active tab'}
 if(-not(Is-Visible 1008) -or -not(Is-Visible 1048)){throw 'Undo controls disappeared when switching tabs'}
 }
function Verify-TwoPaneGeometry([string]$label){
 $queue=Rect $(if($LegacyQueue){1016}else{1071})
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
function PageLabel {
 $page=TextControl 1073
 $snapshot=@([DcmNative]::Snapshot($hwnd,$app.Id))
 $label=@($snapshot | Where-Object {$_.Visible -and $_.Text -match '^/ [0-9]+'} | Select-Object -First 1)
 if(-not $label.Count){return ''}
 $details=$label[0].Text
 $parts=[regex]::Match($details,'^/\s*(\d+)(?:\s+(\d+)°)?')
 if(-not $parts.Success){return ''}
 $v='Page '+$page+' of '+$parts.Groups[1].Value
 if($parts.Groups[2].Success){$v+=' '+[char]0x00B7+' '+$parts.Groups[2].Value+[char]0x00B0}
 return $v
}
function RotatedLabel([int]$page,[int]$angle){
 return ('Page '+$page+' of 2 '+[char]0x00B7+' '+$angle+[char]0x00B0)
}
function PreviewSnapshot {
 return @([DcmNative]::Snapshot($hwnd,$app.Id) | Where-Object {$_.Visible -and ($_.Text -like 'DCM PDF Preview*' -or $_.Text -like 'Page * of *')})
}
$lastPDFReadyGeneration=0
function PageReady([string]$want){
 Wait ('rendered '+$want) {
  if((PageLabel) -cne $want){return $false}
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
 # Avoid redirected stderr in GUI runs: a child renderer can inherit the pipe.
 # Native diagnostics use isolated event traces instead.
 foreach($kv in @{
  'FILEWORKSPACE_RUNTIME_DIR'='C:\Flowverse\runtime';
  'LOCALAPPDATA'=$fixture.local;'APPDATA'=$fixture.roaming;'TEMP'=$lab;'TMP'=$lab;
  'DCM_PDF_TRACE'=$trace;'DCM_QUICKSORT_BACKUP_DIR'=(Join-Path $lab 'backups')
 }.GetEnumerator()){$psi.EnvironmentVariables[$kv.Key]=$kv.Value}
 foreach($key in @('DCM_PDF_PROVIDER','DCM_FLOWPDF_WORKER','DCM_FLOWPDF_PERSISTENT')){$psi.EnvironmentVariables.Remove($key)}
 if($SumatraRuntime){
  if(-not(Test-Path -LiteralPath $SumatraRuntime -PathType Leaf)){throw 'Approved Sumatra runtime missing'}
  if((Get-FileHash -LiteralPath $SumatraRuntime -Algorithm SHA256).Hash.ToLowerInvariant() -ne '719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0'){throw 'Sumatra runtime failed approved pin'}
  $psi.EnvironmentVariables['DCM_SUMATRA_EXE']=[IO.Path]::GetFullPath($SumatraRuntime)
 }
 if($LegacyQueue){$psi.EnvironmentVariables['DCM_NATIVE_QUEUE_LISTVIEW']='0'}else{$psi.EnvironmentVariables.Remove('DCM_NATIVE_QUEUE_LISTVIEW')}
 if($VerifyNativeReport -and $LegacyQueue){throw 'Cannot verify native report with legacy rollback enabled'}
 if($NativeReportDiagnostic -or $VerifyPreviewCoalescing){$psi.EnvironmentVariables['DCM_NATIVE_QUEUE_TRACE']=(Join-Path $lab 'native-report-events.log')}
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
 if($KeyboardSoakSteps -gt 0){Write-Output 'KBD_STAGE=fixture-loaded'}
 if($VerifyNativeReport){
  $native=[DcmTwoPaneGeometry]::FindWindowEx([IntPtr]$hwnd,[IntPtr]::Zero,'SysListView32',$null)
  if($native -eq [IntPtr]::Zero -or -not [DcmTwoPaneGeometry]::IsWindowVisible($native)){throw 'Native ListView report is not visible'}
  $header=[DcmTwoPaneGeometry]::FindWindowEx($native,[IntPtr]::Zero,'SysHeader32',$null)
  if($header -eq [IntPtr]::Zero){throw 'Native ListView header is missing'}
  $columnCount=[DcmTwoPaneGeometry]::SendMessage($header,0x1200,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
  $itemCount=[DcmTwoPaneGeometry]::SendMessage($native,0x1004,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
  if($columnCount -ne 4 -or $itemCount -ne 4){throw ('Native ListView columns/items mismatch '+$columnCount+'/'+$itemCount)}
  Step 'native-report-four-columns' @{class='SysListView32';header_class='SysHeader32';columns=$columnCount;items=$itemCount;visible=$true}
  if($VerifyNativeSplitterOnly){
   # Isolated native-table + splitter qualification before any queue
   # selection actions (which are separately investigated).
   $queueRect=[DcmTwoPaneGeometry+RECT]::new()
   $preview=[DcmNative]::ChildByClass($hwnd,$app.Id,'QuickSortPreview')
   $previewRect=[DcmTwoPaneGeometry+RECT]::new()
   if(-not [DcmTwoPaneGeometry]::GetWindowRect($native,[ref]$queueRect) -or
      -not [DcmTwoPaneGeometry]::GetWindowRect($preview,[ref]$previewRect)){throw 'Native splitter geometry missing'}
   $widthBefore=$queueRect.Right-$queueRect.Left
   $previewBefore=$previewRect.Right-$previewRect.Left
   $start=[DcmSplitterPoint]::new()
   $start.X=[int](($queueRect.Right+$previewRect.Left)/2)
   $start.Y=[int](($queueRect.Top+$queueRect.Bottom)/2)
   if(-not [DcmSplitterPoint]::ScreenToClient([IntPtr]$hwnd,[ref]$start)){throw 'Native splitter coordinate conversion failed'}
   $pack={param([int]$x,[int]$y);return [IntPtr]((($y -band 0xffff) -shl 16) -bor ($x -band 0xffff))}
   [void][DcmTwoPaneGeometry]::SendMessage([IntPtr]$hwnd,0x0201,[IntPtr]1,(& $pack $start.X $start.Y))
   foreach($offset in @(25,50,75,110)){
    [void][DcmTwoPaneGeometry]::SendMessage([IntPtr]$hwnd,0x0200,[IntPtr]1,(& $pack ($start.X+$offset) $start.Y))
   }
   # No resize during guide-only drag: the native ListView must remain stable.
   $during=[DcmTwoPaneGeometry+RECT]::new()
   [void][DcmTwoPaneGeometry]::GetWindowRect($native,[ref]$during)
   if($during.Right-$during.Left -ne $widthBefore){throw 'Native ListView resized during guide movement'}
   [void][DcmTwoPaneGeometry]::SendMessage([IntPtr]$hwnd,0x0202,[IntPtr]::Zero,(& $pack ($start.X+110) $start.Y))
   Start-Sleep -Milliseconds 400
   $after=[DcmTwoPaneGeometry+RECT]::new()
   $afterPreview=[DcmTwoPaneGeometry+RECT]::new()
   [void][DcmTwoPaneGeometry]::GetWindowRect($native,[ref]$after)
   [void][DcmTwoPaneGeometry]::GetWindowRect($preview,[ref]$afterPreview)
   $widthAfter=$after.Right-$after.Left
   $previewAfter=$afterPreview.Right-$afterPreview.Left
   if($widthAfter-$widthBefore -lt 70 -or $previewBefore-$previewAfter -lt 70){throw 'Native splitter release failed to resize both panes'}
   $afterHeader=[DcmTwoPaneGeometry]::FindWindowEx($native,[IntPtr]::Zero,'SysHeader32',$null)
   $cols=[DcmTwoPaneGeometry]::SendMessage($afterHeader,0x1200,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
   if($cols -ne 4){throw 'Native header lost columns after splitter drag'}
   Step 'native-splitter-guide-and-release' @{list_before=$widthBefore;list_after=$widthAfter;preview_before=$previewBefore;preview_after=$previewAfter;native_columns=$cols;stable_while_dragging=$true}
   if(-not $SkipScreenshots){Capture 'native-splitter-after-drag' 'teal'}
   $receipt.passed=$true
   return
  }
  if($VerifyPDFPicker){
   List-Selected '08-preview.pdf'
   PageReady 'Page 1 of 2'
   $mainMenu=[DcmPDFPickerProbe]::GetMenu([IntPtr]$hwnd)
   if($mainMenu -eq [IntPtr]::Zero){throw 'PDF previewer menu missing'}
   foreach($choice in @(2012,2011,2010)){
    if(-not [DcmPDFPickerProbe]::PostMessage([IntPtr]$hwnd,0x0111,[IntPtr]$choice,[IntPtr]::Zero)){throw ('Could not select PDF backend '+$choice)}
    $expected=$choice
    Wait ('PDF menu choice '+$choice) {[DcmPDFPickerProbe]::IsChecked($mainMenu,[uint32]$expected)} 12
    PageReady 'Page 1 of 2'
    Step 'pdf-preview-selector-switch' @{command=$choice;checked=$true;page_rendered=$true}
   }
   if($SumatraRuntime){
    if(-not [DcmPDFPickerProbe]::PostMessage([IntPtr]$hwnd,0x0111,[IntPtr]2013,[IntPtr]::Zero)){throw 'Cannot select Sumatra provider'}
    Wait 'Sumatra option checked' {[DcmPDFPickerProbe]::IsChecked($mainMenu,[uint32]2013)} 12
    Wait 'Sumatra embedded child attached' {$null -ne [DcmSumatraProbe]::Find($hwnd,$app.Id)} 30
    $viewer=[DcmSumatraProbe]::Find($hwnd,$app.Id)
    if(-not $viewer -or $viewer.PID -le 0){throw 'Sumatra owned child unavailable'}
    Wait 'Sumatra viewer settled inside preview pane' {
     try {[DcmSumatraProbe]::CheckGeometry($hwnd,$app.Id,$viewer.PID);return $true}
     catch{return $false}
    } 20
    Step 'pdf-preview-selector-sumatra-embedded' @{viewer_pid=$viewer.PID;class=$viewer.Class;width=$viewer.Width;height=$viewer.Height;checked=$true}
    if(-not [DcmPDFPickerProbe]::PostMessage([IntPtr]$hwnd,0x0111,[IntPtr]2010,[IntPtr]::Zero)){throw 'Cannot restore Auto provider after Sumatra'}
    Wait 'Auto restored after Sumatra' {[DcmPDFPickerProbe]::IsChecked($mainMenu,[uint32]2010) -and $null -eq [DcmSumatraProbe]::Find($hwnd,$app.Id)} 25
    PageReady 'Page 1 of 2'
    if(Get-Process -Id $viewer.PID -ErrorAction SilentlyContinue){throw 'Sumatra child survived viewer switch'}
    Step 'pdf-preview-selector-sumatra-released' @{viewer_pid=$viewer.PID;owned_child_exited=$true;pdfium_auto_restored=$true}
   }
   Step 'pdf-preview-selector' @{windows=$true;pdfium=$true;auto=$true;sumatra=([bool]$SumatraRuntime);active_file='08-preview.pdf'}
  }
  if($KeyboardSoakSteps -gt 0){
   Write-Output 'KBD_STAGE=starting-mixed-queue'
   if(-not $NativeReportMouse){throw 'Keyboard soak requires native mouse-driven selection'}
   # Add 36 authentic disposable mixed-document fixtures after the basic
   # four-row checks and remove them before ordinary rename/move tests.
   $kbAdded=New-Object 'System.Collections.Generic.List[string]'
   $kbSources=@(
    @{src=(Join-Path $fixture.inbox '01-alpha.txt');ext='txt'},
    @{src=(Join-Path $repo 'tests/fixtures/dcm33-legacy.source.docx');ext='docx'},
    @{src=(Join-Path $repo 'tests/fixtures/dcm33-legacy-xls.source.xlsx');ext='xlsx'},
    @{src=$pdf;ext='pdf'}
   )
   foreach($kind in $kbSources){
    if(-not(Test-Path -LiteralPath $kind.src)){throw ('Mixed keyboard fixture missing: '+$kind.src)}
    for($i=0;$i -lt 9;$i++){
     $target=Join-Path $fixture.inbox ('kbdstress-'+$kind.ext+'-'+$i.ToString('D2')+'.'+$kind.ext)
     Copy-Item -LiteralPath $kind.src -Destination $target
     $kbAdded.Add($target)
    }
   }
   [DcmNative]::ClickAsync($hwnd,$app.Id,1011)
   Wait 'Mixed keyboard queue loaded' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 40}
   if([DcmTwoPaneGeometry]::SendMessage($native,0x1004,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64() -ne 40){throw 'Native mixed queue did not load forty rows'}
   Step 'keyboard-mixed-queue-created' @{rows=40;extra_documents=36;types=@('TXT','DOCX','XLSX','PDF')}
   Write-Output 'KBD_STAGE=mixed-queue-ready'
   $beforeHost=$app.PrivateMemorySize64
   $beforeHandles=$app.HandleCount
   # Home then traverse the 40-row queue down and back, repeatedly.
   [void][DcmTwoPaneGeometry]::PostMessage($native,0x0100,[IntPtr]0x24,[IntPtr]::Zero)
   [void][DcmTwoPaneGeometry]::PostMessage($native,0x0101,[IntPtr]0x24,[IntPtr]::Zero)
   Wait 'Keyboard Home reaches first row' {[DcmTwoPaneGeometry]::SendMessage($native,0x100C,[IntPtr](-1),[IntPtr]1).ToInt64() -eq 0}
   $cursor=0
   $direction=1
   $keyboardStart=[DateTime]::UtcNow
   for($k=0;$k -lt $KeyboardSoakSteps;$k++){
    if($cursor -ge 39){$direction=-1}
    if($cursor -le 0){$direction=1}
    $key=if($direction -gt 0){0x28}else{0x26}
    if(-not [DcmTwoPaneGeometry]::PostMessage($native,0x0100,[IntPtr]$key,[IntPtr]::Zero)){throw 'Keyboard press not posted'}
    if(-not [DcmTwoPaneGeometry]::PostMessage($native,0x0101,[IntPtr]$key,[IntPtr]::Zero)){throw 'Keyboard release not posted'}
    $cursor+=$direction
    if(($k+1)%30 -eq 0){
     # Keep this responsive without holding the dispatcher synchronously.
     Start-Sleep -Milliseconds 75
     Owned
    }
   }
   $expectedCursor=$cursor
   Wait 'Keyboard soak ends at expected focused row' {[DcmTwoPaneGeometry]::SendMessage($native,0x100C,[IntPtr](-1),[IntPtr]1).ToInt64() -eq $expectedCursor} 30
   $finalFile=@([DcmNative]::ListItems($hwnd,$app.Id,1016))[$expectedCursor]
   Wait 'Keyboard soak filename and preview selection match' {(TextControl 1005) -ceq [IO.Path]::GetFileNameWithoutExtension($finalFile)}
   Start-Sleep -Milliseconds 350
   $app.Refresh()
   $hostGrowth=[long]$app.PrivateMemorySize64-[long]$beforeHost
   $handleGrowth=[int]$app.HandleCount-[int]$beforeHandles
   if($hostGrowth -gt 48MB -or $handleGrowth -gt 64){throw ('Keyboard soak resource growth: '+$hostGrowth+' bytes, '+$handleGrowth+' handles')}
   Step 'native-keyboard-mixed-endurance' @{steps=$KeyboardSoakSteps;rows=40;focused_index=$expectedCursor;host_private_growth_bytes=$hostGrowth;host_handle_growth=$handleGrowth;elapsed_ms=([DateTime]::UtcNow-$keyboardStart).TotalMilliseconds;responsive=$true}
   Write-Output 'KBD_STAGE=keys-verified'
   [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1030,'01-alpha')
   Wait 'Keyboard soak cleanup filters to original file' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 1}
   Write-Output 'KBD_STAGE=cleanup-filter-ready'
   List-Selected '01-alpha.txt'
   foreach($tempFile in $kbAdded){Remove-Item -LiteralPath $tempFile -Force}
   [DcmNative]::ClickAsync($hwnd,$app.Id,1033)
   Wait 'Mixed keyboard fixture cleanup' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 4}
   Step 'keyboard-mixed-fixtures-cleaned' @{removed=36;original_rows=4}
   Write-Output 'KBD_STAGE=cleanup-complete'
  }

  if($VerifyPreviewCoalescing){
   $nativeRows=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
   if($nativeRows.Count -lt 4){throw 'Rapid-navigation preview test needs four queue rows'}
   $tracePath=(Join-Path $lab 'native-report-events.log')
   Start-Sleep -Milliseconds 250
   if(Test-Path $tracePath){[IO.File]::WriteAllText($tracePath,'')}
   foreach($row in @(1,2,3)){
    if(-not [DcmNativeReportProbe]::ClickRow($native,$row)){throw ('Cannot click native row '+$row)}
   }
   $expectedStem=[IO.Path]::GetFileNameWithoutExtension($nativeRows[3])
   Wait 'Rapid native navigation updates filename' {(TextControl 1005) -ceq $expectedStem}
   Wait 'Rapid native navigation settles preview' {(Test-Path $tracePath) -and @((Get-Content $tracePath) | Where-Object {$_ -like 'preview-settled *'}).Count -ge 1}
   Start-Sleep -Milliseconds 350
   $settled=@((Get-Content $tracePath) | Where-Object {$_ -like 'preview-settled *'})
   if($settled.Count -ne 1){throw ('Expected one preview for three quick selections; observed '+$settled.Count)}
   if(-not $settled[0].EndsWith($nativeRows[3])){throw ('Settled wrong file: '+$settled[0])}
   Step 'native-queue-preview-coalesced' @{rapid_row_changes=3;settled_previews=1;final_file=$nativeRows[3]}
  }
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
  if($VerifyNativeMulti){
   # Use real ListView mouse messages, keeping the standard app/adapter path.
   # The original row is selected. Ctrl-click the adjacent row and verify
   # BOTH native selections are visible through the legacy queue adapter.
   $nextRowPoint=[IntPtr]((88 -shl 16) -bor 65)
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0201,[IntPtr]9,$nextRowPoint)
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0202,[IntPtr]8,$nextRowPoint)
   $legacySelected=@()
   Wait 'Native Ctrl selection propagated into queue' {
    $script:legacySelected=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016))
    return $script:legacySelected.Count -eq 2
   }
   $nativeSelectedCount=[DcmTwoPaneGeometry]::SendMessage($native,0x1032,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
   if($nativeSelectedCount -ne 2){throw ('Native Ctrl click selection count mismatch: '+$nativeSelectedCount)}
   Step 'native-report-multiselection' @{native_count=$nativeSelectedCount;legacy_indices=$legacySelected;preserved=$true}
  }
  if($VerifyRefreshSelection){
   if(-not $VerifyNativeMulti){throw 'VerifyRefreshSelection requires VerifyNativeMulti'}
   # Reload preserves the same two file identities, not merely their old
   # row numbers. The native and backing list selections must still match.
   $before=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016))
   [DcmNative]::ClickAsync($hwnd,$app.Id,1011)
   Wait 'Reload preserved two selected files' {@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)).Count -eq 2}
   $after=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016))
   if(($before -join ',') -cne ($after -join ',')){throw 'Reload changed selected file identities'}
   $nativeCount=[DcmTwoPaneGeometry]::SendMessage($native,0x1032,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
   if($nativeCount -ne 2){throw 'Reload did not restore the native multi-selection'}
   Step 'queue-refresh-reload-preserved' @{selected=$after;native_count=$nativeCount}

   [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1030,'01-alpha')
   Wait 'Name filter narrowed list' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq '01-alpha.txt'}
   Wait 'Name filter retained visible selection' {@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)).Count -eq 1 -and (TextControl 1005) -ceq '01-alpha'}
   $nativeCount=[DcmTwoPaneGeometry]::SendMessage($native,0x1032,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
   if($nativeCount -ne 1){throw 'Name filter did not retain native selected file'}
   Step 'queue-refresh-filter-intersection' @{selected_file='01-alpha.txt';native_count=$nativeCount}

   [DcmNative]::ClickAsync($hwnd,$app.Id,1033)
   Wait 'Clear Filters restores four rows' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 4}
   Wait 'Clear Filters preserves surviving selection' {(TextControl 1005) -ceq '01-alpha' -and @([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)).Count -eq 1}
   Step 'queue-refresh-clear-preserved' @{selected_file='01-alpha.txt'}

   [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1030,'not-a-real-file')
   Wait 'Filter hides all files' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 0 -and (TextControl 1005) -ceq ''}
   if([DcmTwoPaneGeometry]::SendMessage($native,0x1004,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64() -ne 0){throw 'Native queue retained stale rows when filter empty'}
   [DcmNative]::ClickAsync($hwnd,$app.Id,1033)
   Wait 'Clear impossible filter restores queue' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 4 -and @([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)).Count -eq 1}
   Step 'queue-refresh-hidden-all-safe' @{empty_queue_cleared=$true;fallback_selected=$true}

   SwitchSource $fixture.destination 0
   if([DcmTwoPaneGeometry]::SendMessage($native,0x1004,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64() -ne 0){throw 'Source switch retained old-folder native rows'}
   SwitchSource $fixture.inbox 4
   Wait 'Source return selected only a current-folder file' {@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)).Count -eq 1}
   Step 'queue-refresh-source-switch-safe' @{old_folder_selection_cleared=$true;restored_folder_count=4}
  }
  if($VerifyNativeRangeKeyboard){
   # Select first row, then Shift-click third row. This tests range
   # selection through real ListView mouse paths and the legacy adapter.
   $first=[IntPtr]((45 -shl 16) -bor 65)
   $third=[IntPtr]((88 -shl 16) -bor 65)
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0201,[IntPtr]1,$first)
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0202,[IntPtr]::Zero,$first)
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0201,[IntPtr]5,$third)
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0202,[IntPtr]4,$third)
   Wait 'Native Shift range propagated to QuickSort' {@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)).Count -eq 3}
   $range=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016))
   $nativeRange=[DcmTwoPaneGeometry]::SendMessage($native,0x1032,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
   if($nativeRange -ne 3 -or (($range -join ',') -cne '0,1,2')){throw ('Native Shift range wrong: '+($range -join ',')+'; native='+$nativeRange)}
   Step 'native-report-shift-range' @{native_count=$nativeRange;legacy_indices=$range}
   # Navigate by native keyboard: arrow down should focus a new row and
   # preserve the one-row selected filename through the adapter.
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0100,[IntPtr]0x28,[IntPtr]::Zero)
   [void][DcmTwoPaneGeometry]::SendMessage($native,0x0101,[IntPtr]0x28,[IntPtr]::Zero)
   $focusIndex=[DcmTwoPaneGeometry]::SendMessage($native,0x100C,[IntPtr](-1),[IntPtr]1).ToInt64()
   Step 'native-report-keyboard-arrow' @{focused_row=$focusIndex;processed=$true}
  }
  # Clicking the native SysHeader32 (not a surrogate button) should sort the
  # same underlying file array used by Rename/Move and all other operations.
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0201,[IntPtr]1,[IntPtr]0x000C000A)
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0202,[IntPtr]::Zero,[IntPtr]0x000C000A)
  $nameOrder=@('01-alpha.txt','02-bravo.txt','03-charlie.txt','08-preview.pdf')
  Wait 'Native ListView Name header sorts ascending' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($nameOrder -join '|')}
  Step 'native-report-header-sort' @{name_ascending=$true;native_header_clicked=$true}
  $headerRect=New-Object DcmTwoPaneGeometry+RECT
  if(-not [DcmTwoPaneGeometry]::GetWindowRect($header,[ref]$headerRect)){throw 'Native report header geometry unavailable'}
  $updatedX=[int](($headerRect.Right-$headerRect.Left)*0.9)
  $updatedPoint=[IntPtr]((12 -shl 16) -bor $updatedX)
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0201,[IntPtr]1,$updatedPoint)
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0202,[IntPtr]::Zero,$updatedPoint)
  Wait 'Native Updated header selected' {(TextControl 1045) -eq 'Date Updated'}
  $updatedFirst=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0201,[IntPtr]1,$updatedPoint)
  [void][DcmTwoPaneGeometry]::SendMessage($header,0x0202,[IntPtr]::Zero,$updatedPoint)
  [array]::Reverse($updatedFirst)
  Wait 'Native Updated header reverses order' {(@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($updatedFirst -join '|')}
  Step 'native-report-updated-header-sort' @{updated_ascending_and_descending=$true}
  $legacyNow=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016))
  $nativeNow=@()
  $scan=[IntPtr](-1)
  while($true){
   $next=[DcmTwoPaneGeometry]::SendMessage($native,0x100C,$scan,[IntPtr]2).ToInt64()
   if($next -lt 0){break}
   $nativeNow+=([int]$next)
   $scan=[IntPtr]$next
  }
  if(($legacyNow -join ',') -cne ($nativeNow -join ',')){throw ('Native/legacy selected rows diverged after header sort: '+($nativeNow -join ',')+' / '+($legacyNow -join ','))}
  if($nativeNow.Count -lt 1){throw 'Native selection lost after header sorting'}
  $visibleNames=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
  $selectedStems=@($nativeNow | ForEach-Object {[IO.Path]::GetFileNameWithoutExtension($visibleNames[$_])})
  $activeStem=(TextControl 1005)
  if($selectedStems -cnotcontains $activeStem){throw ('QuickSort active name is absent from sorted selection: '+$activeStem+'; selected='+($selectedStems -join ','))}
  Step 'native-report-selection-after-sort' @{selected_indices=$nativeNow;selected_names=$selectedStems;active_name=$activeStem;preserved=$true}

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
  Wait 'History visible at 800' {(Is-Visible 1017) -and -not (QueueIsVisible)}
  Verify-TwoPaneTab $true
  Select-NativeQueueTab $false
  Wait 'Queue visible at 800' {(QueueIsVisible) -and -not (Is-Visible 1017)}
  Verify-TwoPaneTab $false
  Step 'two-pane-compact-tab-toggle' @{width_requested=800;queue_restored=$true;history_available=$true}
  if(-not [DcmTwoPaneGeometry]::MoveWindow([IntPtr]$hwnd,$area.Left+10,$area.Top+10,[Math]::Min(1350,$area.Width-20),[Math]::Min(820,$area.Height-20),$true)){throw 'Unable to restore owned QuickSort window size'}
  Start-Sleep -Milliseconds 300
  Verify-TwoPaneGeometry 'wide'
  Select-NativeQueueTab $true
  Wait 'History tab visible' {(Is-Visible 1017) -and -not (QueueIsVisible)}
  Verify-TwoPaneTab $true
  Select-NativeQueueTab $false
  Wait 'Queue tab visible' {(QueueIsVisible) -and -not (Is-Visible 1017)}
  Verify-TwoPaneTab $false
  Step 'two-pane-queue-history-toggle' @{queue_visible=$true;history_visible_when_selected=$true;undo_always_visible=$true}
 }
 List-Selected '08-preview.pdf'
 if($VerifyActualFilename){
  Assert-ActualFile '08-preview.pdf' 'PDF'
  Assert-ActualFilenameLayout
 }
 PageReady 'Page 1 of 2'
 if($VerifyMoveGating){
  $move=[DcmNative]::Control($hwnd,$app.Id,1010)
  if([DcmNative]::IsWindowEnabled($move)){throw 'Move to Vault enabled without configured Vault'}
  Step 'move-gating-unconfigured-vault' @{selected_file='08-preview.pdf';button_enabled=$false}
  # Fixture config is isolated from the user's actual Vault settings.
  # Recheck disabled state after moving between two selected files.
  List-Selected '03-charlie.txt'
  if([DcmNative]::IsWindowEnabled($move)){throw 'Move to Vault enabled after file selection without Vault'}
  Step 'move-gating-selected-without-vault' @{selected_file='03-charlie.txt';button_enabled=$false}
  List-Selected '08-preview.pdf'
  PageReady 'Page 1 of 2'
 }
 if($VerifySplitter){
  # Exercise the exact app-owned splitter through native Win32 mouse messages.
  # No desktop-wide input and no changes to the user's current files/windows.
  $queueHwnd=[DcmNative]::Control($hwnd,$app.Id,1016)
  $pane=[DcmNative]::ChildByClass($hwnd,$app.Id,'QuickSortPreview')
  $beforeQueue=[DcmTwoPaneGeometry+RECT]::new()
  $beforePane=[DcmTwoPaneGeometry+RECT]::new()
  $windowRect=[DcmTwoPaneGeometry+RECT]::new()
  if(-not [DcmTwoPaneGeometry]::GetWindowRect([IntPtr]$queueHwnd,[ref]$beforeQueue) -or
     -not [DcmTwoPaneGeometry]::GetWindowRect([IntPtr]$pane,[ref]$beforePane) -or
     -not [DcmTwoPaneGeometry]::GetWindowRect([IntPtr]$hwnd,[ref]$windowRect)){
   throw 'Splitter baseline rectangles unavailable'
  }
  $screenStartX=[int](($beforeQueue.Right+$beforePane.Left)/2)
  # Convert screen to client using the actual window borders through ScreenToClient.
  $start=[DcmSplitterPoint]::new()
  $start.X=$screenStartX;$start.Y=[int](($beforeQueue.Top+$beforeQueue.Bottom)/2)
  if(-not [DcmSplitterPoint]::ScreenToClient([IntPtr]$hwnd,[ref]$start)){throw 'Splitter ScreenToClient failed'}
  $startX=$start.X;$y=$start.Y
  $pack={param([int]$x,[int]$vertical);return [IntPtr]((($vertical -band 0xffff) -shl 16) -bor ($x -band 0xffff))}
  [void][DcmTwoPaneGeometry]::SendMessage([IntPtr]$hwnd,0x0201,[IntPtr]1,(& $pack $startX $y))
  [void][DcmTwoPaneGeometry]::SendMessage([IntPtr]$hwnd,0x0200,[IntPtr]1,(& $pack ($startX+110) $y))
  [void][DcmTwoPaneGeometry]::SendMessage([IntPtr]$hwnd,0x0202,[IntPtr]::Zero,(& $pack ($startX+110) $y))
  Start-Sleep -Milliseconds 400
  $afterQueue=[DcmTwoPaneGeometry+RECT]::new()
  $afterPane=[DcmTwoPaneGeometry+RECT]::new()
  if(-not [DcmTwoPaneGeometry]::GetWindowRect([IntPtr]$queueHwnd,[ref]$afterQueue) -or
     -not [DcmTwoPaneGeometry]::GetWindowRect([IntPtr]$pane,[ref]$afterPane)){
   throw 'Splitter drag rectangles unavailable'
  }
  $queueBefore=$beforeQueue.Right-$beforeQueue.Left
  $queueAfter=$afterQueue.Right-$afterQueue.Left
  $previewBefore=$beforePane.Right-$beforePane.Left
  $previewAfter=$afterPane.Right-$afterPane.Left
  if($queueAfter-$queueBefore -lt 70 -or $previewBefore-$previewAfter -lt 70) {
   throw ('Splitter drag failed to resize both panes: queue '+$queueBefore+' -> '+$queueAfter+' preview '+$previewBefore+' -> '+$previewAfter)
  }
  if((PageLabel) -cne 'Page 1 of 2'){throw 'PDF page state changed during splitter drag'}
  Step 'queue-preview-splitter-drag' @{queue_before=$queueBefore;queue_after=$queueAfter;preview_before=$previewBefore;preview_after=$previewAfter;page_preserved=$true}
  if(-not $SkipScreenshots){Capture 'splitter-after-drag' 'teal'}
 }
 if($VerifyPDFPicker -and $LegacyQueue){
  # Isolate the backend menu from the native report selection adapter.
  $mainMenu=[DcmPDFPickerProbe]::GetMenu([IntPtr]$hwnd)
  if($mainMenu -eq [IntPtr]::Zero){throw 'PDF previewer menu missing'}
  foreach($choice in @(2012,2011,2010)){
   if(-not [DcmPDFPickerProbe]::PostMessage([IntPtr]$hwnd,0x0111,[IntPtr]$choice,[IntPtr]::Zero)){
    throw ('Could not select PDF backend '+$choice)
   }
   $expected=$choice
   Wait ('PDF menu choice '+$choice) {[DcmPDFPickerProbe]::IsChecked($mainMenu,[uint32]$expected)} 12
   PageReady 'Page 1 of 2'
   Step 'pdf-preview-selector-switch' @{command=$choice;checked=$true;page_rendered=$true;queue='legacy'}
  }
  Step 'pdf-preview-selector' @{windows=$true;pdfium=$true;auto=$true;sumatra=$false;active_file='08-preview.pdf';queue='legacy'}
 }
 if($VerifyNarrowToolbar){
  $area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
  foreach($requestedWidth in @(1350,920,800)){
   $targetWidth=[Math]::Min($requestedWidth,$area.Width-20)
   if(-not [DcmTwoPaneGeometry]::MoveWindow([IntPtr]$hwnd,$area.Left+10,$area.Top+10,$targetWidth,[Math]::Min(760,$area.Height-20),$true)){
    throw ('Unable to resize QuickSort for toolbar width '+$targetWidth)
   }
   Start-Sleep -Milliseconds 350
   Owned
   $preview=[DcmNative]::ChildByClass($hwnd,$app.Id,'QuickSortPreview')
   $bounds=[DcmTwoPaneGeometry+RECT]::new()
   if(-not [DcmTwoPaneGeometry]::GetWindowRect($preview,[ref]$bounds)){throw 'Preview pane geometry unavailable'}
   $buttons=@()
   foreach($id in @(1063,1062)){
    if(-not (Is-Visible $id)){throw ('Rotation button hidden at window width '+$targetWidth+' (ID '+$id+')')}
    $r=Rect $id
    if($r.Left -lt $bounds.Left -or $r.Right -gt $bounds.Right -or $r.Top -lt $bounds.Top -or $r.Bottom -gt $bounds.Bottom){
     throw ('Rotation button outside preview at width '+$targetWidth+' (ID '+$id+')')
    }
    $buttons+=@($r)
   }
   if($buttons[0].Right -gt $buttons[1].Left){throw ('Rotation buttons overlap at window width '+$targetWidth)}
   Step 'narrow-toolbar-rotation-visible' @{window_width=$targetWidth;preview_width=($bounds.Right-$bounds.Left);left_id=1063;right_id=1062;both_visible=$true;no_overlap=$true}
   # Capture the actual resized geometry. Generic Capture restores the window
   # first and would silently invalidate this width-specific evidence.
   if(-not $SkipScreenshots){
    $shot=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ('narrow-toolbar-'+$targetWidth+'.png')),$false)
    $receipt.captures+=@([ordered]@{name=('narrow-toolbar-'+$targetWidth);path=$shot.Path;width=$shot.Width;height=$shot.Height;visual_reviewed=$false})
   }
  }
  if(-not [DcmTwoPaneGeometry]::MoveWindow([IntPtr]$hwnd,$area.Left+10,$area.Top+10,[Math]::Min(1350,$area.Width-20),[Math]::Min(820,$area.Height-20),$true)){
   throw 'Unable to restore QuickSort window after narrow-toolbar verification'
  }
  Start-Sleep -Milliseconds 350
  if((PageLabel) -cne 'Page 1 of 2'){throw 'PDF page state changed after resizing'}
 }
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
 [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1073,'2')
 [DcmNative]::ClickAsync($hwnd,$app.Id,1074)
 PageReady 'Page 2 of 2'
 Step 'custom-pdf-page-jump' @{entered=2;rendered=2}
 [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1073,'1')
 [DcmNative]::ClickAsync($hwnd,$app.Id,1074)
 PageReady 'Page 1 of 2'
 Step 'custom-pdf-page-return' @{entered=1;rendered=1}
 foreach($control in @(
  @{id=1075;name='fit-width'},
  @{id=1076;name='fit-page'},
  @{id=1078;name='zoom-in'},
  @{id=1077;name='zoom-out'}
 )){
  $handle=[DcmNative]::Control($hwnd,$app.Id,$control.id)
  if($handle -eq [IntPtr]::Zero -or -not [DcmNative]::IsWindowEnabled($handle)){throw ('Custom PDF toolbar control unavailable: '+$control.name)}
  [DcmNative]::ClickAsync($hwnd,$app.Id,$control.id)
  PageReady 'Page 1 of 2'
  Step ('custom-pdf-toolbar-'+$control.name) @{control_id=$control.id;rendered=$true}
 }
 if($VerifyRotation) {
  $originalHash=Hash $pdf
  foreach($angle in @(90,180,270)) {
   $previous=@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}).Count
   [DcmNative]::ClickAsync($hwnd,$app.Id,1062)
   $expected=RotatedLabel 1 $angle
   Wait ('PDF clockwise rotation '+$angle) {
    if((TextControl 1073) -cne '1'){return $false}
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
     $pageOK=(TextControl 1073) -ceq '2'
     $applied=@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}|ForEach-Object {$_|ConvertFrom-Json})
     return ($pageOK -and $applied[-1].rotation -eq 90 -and $applied[-1].generation -gt $nextFromGeneration)
    } 50
    Capture 'rotate-90-next-page' 'orange'
    $returnFromGeneration=(@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}|ForEach-Object {$_|ConvertFrom-Json})[-1]).generation
    [DcmNative]::ClickAsync($hwnd,$app.Id,1060)
    Wait 'rotated previous-page navigation' {
     $pageOK=(TextControl 1073) -ceq '1'
     $applied=@(Get-Content -LiteralPath $trace|Where-Object {$_ -match '"event":"result-applied"'}|ForEach-Object {$_|ConvertFrom-Json})
     return ($pageOK -and $applied[-1].rotation -eq 90 -and $applied[-1].generation -gt $returnFromGeneration)
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
    if((TextControl 1073) -cne '1'){return $false}
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
  Wait 'Queue restored after history' {(QueueIsVisible) -and -not (Is-Visible 1017)}
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
 if($VerifyCodeText){
  $samples=@(
   @{name='15-sample_test.go';body="package example`nfunc TestSourceFile() {}`n";marker='TestSourceFile'},
   @{name='16-test-config.json';body='{"fixture":"read-as-text-check"}';marker='read-as-text-check'},
   @{name='17-sample_test.py';body="def test_preview():`n    return 'python-preview-marker'`n";marker='python-preview-marker'}
  )
  foreach($sample in $samples){
   [IO.File]::WriteAllText((Join-Path $fixture.inbox $sample.name),$sample.body,[Text.UTF8Encoding]::new($false))
  }
  [DcmNative]::ClickAsync($hwnd,$app.Id,1011)
  Wait 'code samples listed' { $items=@([DcmNative]::ListItems($hwnd,$app.Id,1016));($items -contains '15-sample_test.go') -and ($items -contains '16-test-config.json') -and ($items -contains '17-sample_test.py') }
  foreach($sample in $samples){
   List-Selected $sample.name
   $check=[DcmNative]::Control($hwnd,$app.Id,1014)
   if(-not [DcmNative]::IsWindowEnabled($check)){throw ('Read as Text disabled for '+$sample.name)}
   $state=[DcmTwoPaneGeometry]::SendMessage([IntPtr]$check,0x00F0,[IntPtr]::Zero,[IntPtr]::Zero).ToInt64()
   if($state -ne 1){[DcmNative]::ClickAsync($hwnd,$app.Id,1014)}
   $pane=[DcmNative]::ChildByClass($hwnd,$app.Id,'QuickSortPreview')
   $edit=[DcmTwoPaneGeometry]::FindWindowEx([IntPtr]$pane,[IntPtr]::Zero,'Edit',$null)
   if($edit -eq [IntPtr]::Zero){throw 'Read as Text preview edit missing'}
   $marker=$sample.marker
   Wait ('text preview '+$sample.name) {([DcmNative]::Text($edit,$app.Id)).Contains($marker)}
   Step 'read-as-text-code-file' @{file=$sample.name;marker=$marker;enabled=$true;content_visible=$true}
  }
  List-Selected '08-preview.pdf'
  if([DcmNative]::IsWindowEnabled([DcmNative]::Control($hwnd,$app.Id,1014))){throw 'Read as Text enabled for PDF'}
  PageReady 'Page 1 of 2'
  Step 'read-as-text-pdf-rejection' @{enabled=$false;pdf_preview_restored=$true}
 }
 if($Persistent -or $DevDefault) {
  $events=@(Get-Content -LiteralPath $trace | ForEach-Object {$_|ConvertFrom-Json})
  $starts=@($events | Where-Object {$_.event -eq 'worker-started' -and $_.renderer_pid -gt 0})
  $unique=@($starts | Select-Object -ExpandProperty renderer_pid -Unique)
  if($starts.Count -lt 6){throw ('Too few persistent PDFium render requests: '+$starts.Count)}
  . (Join-Path $PSScriptRoot 'pdfium-worker-transition-contract.ps1')
  if($VerifyPDFPicker){
   # Renderer changes are intentional here. Require working final PDFium
   # rendering and no one-shot fallback; ordinary runs retain strict worker
   # cancellation/reuse acceptance unchanged.
   $restarts=0
   Step 'pdf-preview-selector-intentional-worker-transition' @{mode='backend comparison';last_worker=$starts[-1].renderer_pid;worker_start_events=$starts.Count}
  }else{
   $contract=Assert-PDFiumWorkerTransitions -Events $events
   $restarts=[int]$contract.cancel_restarts
  }
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
