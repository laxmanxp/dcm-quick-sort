param([string]$Lab='C:\FlowverseReports\dcm-flowuivision-20261103')
$ErrorActionPreference='Stop'
$owned=Get-Content (Join-Path $Lab 'owned.json') -Raw|ConvertFrom-Json
$app=Get-Process -Id $owned.pid -ErrorAction Stop
if($app.Path -ne $owned.exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $owned.start){throw 'Not the owned disposable QuickSort process'}
$app.Refresh();$hwnd=$app.MainWindowHandle
if($app.MainWindowTitle -notlike '*1.0.69*'){throw 'Unexpected QuickSort version'}
$report=Join-Path $Lab ('features-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss'))
New-Item -ItemType Directory $report|Out-Null
$env:FLOWVERSE_REPO=$Lab
$vision=Join-Path $Lab 'FlowUIVision.exe'
Add-Type -AssemblyName System.Drawing
Add-Type @'
using System;using System.Text;using System.Runtime.InteropServices;
public static class QSEvidence {
 [StructLayout(LayoutKind.Sequential)] public struct RECT{public int L,T,R,B;}
 [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h,out RECT r);
 [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr h,IntPtr dc,uint flags);
 [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
 [DllImport("user32.dll")] public static extern IntPtr GetDlgItem(IntPtr h,int id);
 [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 [DllImport("user32.dll",EntryPoint="GetWindowLongPtrW")] public static extern IntPtr Style(IntPtr h,int index);
 [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] static extern IntPtr Send(IntPtr h,uint m,IntPtr w,IntPtr l,uint f,uint ms,out IntPtr r);
 [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] static extern IntPtr SendText(IntPtr h,uint m,IntPtr w,StringBuilder b,uint f,uint ms,out IntPtr r);
 public static string[] ListItems(IntPtr h){IntPtr r;if(Send(h,0x018B,IntPtr.Zero,IntPtr.Zero,2,1000,out r)==IntPtr.Zero)throw new Exception("List query timeout");int n=r.ToInt32();if(n<0||n>100)throw new Exception("Unexpected disposable list size");var a=new string[n];for(int i=0;i<n;i++){var b=new StringBuilder(4096);if(SendText(h,0x0189,(IntPtr)i,b,2,1000,out r)==IntPtr.Zero)throw new Exception("List text timeout");a[i]=b.ToString();}return a;}
}
'@
function Save-Shot([string]$Name){
 [uint32]$owner=0;[void][QSEvidence]::GetWindowThreadProcessId($hwnd,[ref]$owner)
 if($owner -ne $app.Id){throw 'Screenshot PID mismatch'}
 $old=[QSEvidence]::SetThreadDpiAwarenessContext([IntPtr](-4))
 try{$r=New-Object QSEvidence+RECT;[void][QSEvidence]::GetWindowRect($hwnd,[ref]$r)
  $b=New-Object Drawing.Bitmap(($r.R-$r.L),($r.B-$r.T));$g=[Drawing.Graphics]::FromImage($b);$dc=$g.GetHdc()
  try{$ok=[QSEvidence]::PrintWindow($hwnd,$dc,2)}finally{$g.ReleaseHdc($dc);$g.Dispose()}
  try{if(-not $ok){throw 'PrintWindow failed'};$b.Save((Join-Path $report ($Name+'.png')),[Drawing.Imaging.ImageFormat]::Png)}finally{$b.Dispose()}
 }finally{[void][QSEvidence]::SetThreadDpiAwarenessContext($old)}
}
$header=@'
target sut type=desktop app=dcm-quicksort
surface main target=sut process="DCM-QS-VisionTest.exe" title~="1.0.69"
control path surface=main id=1001 role=textbox
control name surface=main id=1030 role=textbox
control clear surface=main id=1033 role=button
control newname surface=main id=1005 role=textbox
control tool surface=main id=1027 role=combobox
control apply surface=main id=1028 role=button
control rename surface=main id=1006 role=button
control undo surface=main id=1008 role=button
control destination surface=main id=1003 role=textbox
control move surface=main id=1041 role=button
control fit surface=main id=1023 role=button
control width surface=main id=1047 role=button
control zoomreset surface=main id=1021 role=button
control zoomin surface=main id=1022 role=button
control zoomout surface=main id=1020 role=button
control nextpage surface=main id=1061 role=button
control prevpage surface=main id=1060 role=button
control status surface=main role=text text~="FILES"
control pdfstatus surface=main role=text text~="DCM PDF Preview"
wait main timeout=5s
inspect main
expect path text~="dcm-flowuivision-20261103"
'@
$script:results=@()
function Run-Case([string]$Name,[string]$Body,[scriptblock]$Oracle){
 $fvs=Join-Path $report ($Name+'.fvs')
 [IO.File]::WriteAllText($fvs,($header+"`n"+$Body+"`n"),[Text.UTF8Encoding]::new($false))
 $job=Start-Process $vision -ArgumentList @('run-script',$fvs) -PassThru -RedirectStandardOutput (Join-Path $report ($Name+'.json')) -RedirectStandardError (Join-Path $report ($Name+'.stderr'))
 if(-not $job.WaitForExit(20000)){Stop-Process -Id $job.Id -Force;throw "FlowUIVision deadline: $Name; stop before issuing more inputs"}
 $raw=Get-Content (Join-Path $report ($Name+'.json')) -Raw
 $r=$null;if($raw){$r=$raw|ConvertFrom-Json}
 $result=@{name=$Name;passed=($null -ne $r -and $r.Passed -eq $true);steps=@($r.Steps).Count;visual='not reviewed'}
 if(-not $result.passed){$result.error=(@($r.Steps|Where-Object {-not $_.Passed})|Select-Object -Last 1).Error;if(-not $result.error){$result.error=Get-Content (Join-Path $report ($Name+'.stderr')) -Raw}}
 if($result.passed -and $Oracle){try{& $Oracle;$result.filesystem_verified=$true}catch{$result.passed=$false;$result.error=$_.Exception.Message}}
 Save-Shot $Name
 $result.items=[QSEvidence]::ListItems([QSEvidence]::GetDlgItem($hwnd,1016))
 $script:results+=$result
 Write-Host ($result|ConvertTo-Json -Compress)
 $script:results|ConvertTo-Json -Depth 6|Set-Content (Join-Path $report 'summary.json') -Encoding UTF8
 return [bool]$result.passed
}
foreach($f in @(@('text','01-alpha'),@('png','04-image'),@('docx','05-letter'),@('xlsx','06-orders'),@('pptx','07-slides'),@('pdf','08-print'))){
 $body="click clear`ntype name `"$($f[1])`"`nwait_ready type=textbox target=newname text=`"$($f[1])`" timeout=5s`ndelay 1800ms`ninspect main`nexpect status text~=`"of 1`""
 [void](Run-Case ('preview-'+$f[0]) $body $null)
}
[void](Run-Case 'pdf-zoom' @'
click zoomreset
click zoomin
delay 400ms
inspect main
expect pdfstatus text~="110%"
click zoomout
delay 400ms
inspect main
expect pdfstatus text~="100%"
click width
delay 400ms
inspect main
expect pdfstatus text~="Fit Width"
'@ $null)
[void](Run-Case 'name-clean-spacing' @'
click clear
type name "03"
wait_ready type=textbox target=newname text~="03" timeout=5s
type newname "03   cleanup   test"
select tool "Clean spacing"
click apply
inspect main
expect newname text="03 cleanup test"
'@ $null)
$inbox=Join-Path $Lab 'inbox';$old=Join-Path $inbox '01-alpha.txt';$new=Join-Path $inbox '01-renamed-alpha.txt'
if(-not(Test-Path $old)-or(Test-Path $new)){throw 'Unexpected rename fixture state'}
$hash=(Get-FileHash $old).Hash
$renamed=Run-Case 'rename' @'
click clear
type name "01-"
wait_ready type=textbox target=newname text="01-alpha" timeout=5s
type newname "01-renamed-alpha"
click rename
delay 600ms
inspect main
expect undo enabled
'@ {if((Test-Path $old)-or(-not(Test-Path $new))){throw 'Rename paths incorrect'};if((Get-FileHash $new).Hash -ne $hash){throw 'Rename changed content'}}
if($renamed){[void](Run-Case 'undo-rename' @'
click undo
delay 600ms
inspect main
'@ {if((-not(Test-Path $old))-or(Test-Path $new)){throw 'Undo did not restore original filename'};if((Get-FileHash $old).Hash -ne $hash){throw 'Undo changed content'}})}
$dest=[QSEvidence]::GetDlgItem($hwnd,1003)
@{destination_readonly=(([QSEvidence]::Style($dest,-16).ToInt64() -band 0x800) -ne 0);report=$report}|ConvertTo-Json|Set-Content (Join-Path $report 'capabilities.json')
Write-Output ('REPORT='+$report)
