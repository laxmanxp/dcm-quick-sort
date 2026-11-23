param([Parameter(Mandatory=$true)][int]$TargetPid, [int]$WheelSteps=25)
$ErrorActionPreference='Stop'
$p=Get-CimInstance Win32_Process -Filter "ProcessId = $TargetPid"
if ($null -eq $p -or $p.ExecutablePath -ne 'C:\DCMQuickSortCollisionAcceptance\dcm-pdf-continuous-dev.exe' -or $p.CommandLine -notmatch 'sumatra-scroll.pdf') {throw "Refusing non-fixture process"}
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using System.Text;
public static class PDFModeAcceptance {
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern IntPtr FindWindowEx(IntPtr p,IntPtr after,string cl,string title);
 [DllImport("user32.dll")] public static extern IntPtr GetDlgItem(IntPtr p,int id);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr h,StringBuilder b,int count);
 [DllImport("user32.dll")] public static extern IntPtr SendMessage(IntPtr h,uint msg,IntPtr w,IntPtr l);
}
'@
$main=(Get-Process -Id $TargetPid).MainWindowHandle
if ($main -eq [IntPtr]::Zero) {throw 'Main window not ready'}
$pane=[PDFModeAcceptance]::FindWindowEx($main,[IntPtr]::Zero,'QuickSortPreview',$null)
if ($pane -eq [IntPtr]::Zero){throw 'Preview pane missing'}
$mode=[PDFModeAcceptance]::GetDlgItem($pane,1079)
$page=[PDFModeAcceptance]::GetDlgItem($pane,1073)
if ($mode -eq [IntPtr]::Zero -or $page -eq [IntPtr]::Zero){throw 'PDF view mode control not found'}
function ReadLabel([IntPtr]$h){$b=New-Object System.Text.StringBuilder 128;[void][PDFModeAcceptance]::GetWindowText($h,$b,128);return $b.ToString()}
Write-Output ("Before: mode={0}; page={1}" -f (ReadLabel $mode),(ReadLabel $page))
[void][PDFModeAcceptance]::SendMessage($mode,0x00F5,[IntPtr]::Zero,[IntPtr]::Zero)
Start-Sleep -Milliseconds 300
Write-Output ("After toggle: mode={0}; page={1}" -f (ReadLabel $mode),(ReadLabel $page))
for($i=0;$i -lt $WheelSteps;$i++){
 [void][PDFModeAcceptance]::SendMessage($pane,0x020A,[IntPtr]::new([long]0xFF880000),[IntPtr]::Zero)
 Start-Sleep -Milliseconds 80
}
Write-Output ("After wheel: mode={0}; page={1}" -f (ReadLabel $mode),(ReadLabel $page))
