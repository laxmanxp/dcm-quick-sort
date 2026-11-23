param([Parameter(Mandatory=$true)][int]$TargetPid)
$ErrorActionPreference='Stop'
$p=Get-CimInstance Win32_Process -Filter "ProcessId = $TargetPid"
if ($null -eq $p -or $p.ExecutablePath -ne 'C:\DCMQuickSortCollisionAcceptance\dcm-html-rect-experiment.exe' -or $p.CommandLine -notmatch 'TraceHTMLPhysicalSetRect') { throw "Refusing non-fixture PID $TargetPid" }
Add-Type -TypeDefinition @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class DcmHtmlRepairProbe {
 [StructLayout(LayoutKind.Sequential)] public struct RECT {public int Left,Top,Right,Bottom;}
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern IntPtr FindWindowEx(IntPtr parent, IntPtr childAfter, string className, string windowName);
 [DllImport("user32.dll")] public static extern bool GetClientRect(IntPtr h, out RECT r);
 [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
 [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr insert, int x, int y, int w, int height, uint flags);
 [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
 [DllImport("user32.dll")] public static extern IntPtr GetParent(IntPtr h);
}
'@
$main=(Get-Process -Id $TargetPid).MainWindowHandle
if ($main -eq [IntPtr]::Zero){throw 'Main HWND missing'}
$pane=[DcmHtmlRepairProbe]::FindWindowEx($main,[IntPtr]::Zero,'QuickSortPreview',$null)
if ($pane -eq [IntPtr]::Zero){throw 'Pane missing'}
$extra=[DcmHtmlRepairProbe]::FindWindowEx($main,$pane,'QuickSortPreview',$null)
if ($extra -ne [IntPtr]::Zero){throw 'Multiple panes'}
$outer=[DcmHtmlRepairProbe]::FindWindowEx($pane,[IntPtr]::Zero,'##MimeEdit_Server',$null)
if ($outer -eq [IntPtr]::Zero){throw 'HTML host missing'}
$other=[DcmHtmlRepairProbe]::FindWindowEx($pane,$outer,'##MimeEdit_Server',$null)
if ($other -ne [IntPtr]::Zero){throw 'Multiple HTML hosts'}
$inner=[DcmHtmlRepairProbe]::FindWindowEx($outer,[IntPtr]::Zero,'Internet Explorer_Server',$null)
if ($inner -eq [IntPtr]::Zero){throw 'HTML inner missing'}
[uint32]$outerPid=0
[uint32]$innerPid=0
[void][DcmHtmlRepairProbe]::GetWindowThreadProcessId($outer,[ref]$outerPid)
[void][DcmHtmlRepairProbe]::GetWindowThreadProcessId($inner,[ref]$innerPid)
if ($outerPid -ne $innerPid -or $outerPid -eq $TargetPid){throw "Unexpected cross-process hierarchy"}
$rc=New-Object DcmHtmlRepairProbe+RECT
$outerRc=New-Object DcmHtmlRepairProbe+RECT
[void][DcmHtmlRepairProbe]::GetClientRect($pane,[ref]$rc)
[void][DcmHtmlRepairProbe]::GetWindowRect($outer,[ref]$outerRc)
if ($rc.Right -le 0 -or $rc.Bottom -le 0){throw 'Invalid pane dimensions'}
Write-Output ("Before: pane client {0}x{1}; HTML outer {2}x{3}; child PID {4}" -f $rc.Right,$rc.Bottom,($outerRc.Right-$outerRc.Left),($outerRc.Bottom-$outerRc.Top),$outerPid)
# SWP_NOZORDER | SWP_NOACTIVATE | SWP_NOOWNERZORDER | SWP_ASYNCWINDOWPOS
$ok=[DcmHtmlRepairProbe]::SetWindowPos($outer,[IntPtr]::Zero,0,0,$rc.Right,$rc.Bottom,0x4214)
Write-Output ("SetWindowPos outer accepted: {0}" -f $ok)
