param([Parameter(Mandatory=$true)][int]$TargetPid, [switch]$Nudge)
$ErrorActionPreference='Stop'
$p=Get-CimInstance Win32_Process -Filter "ProcessId = $TargetPid"
if ($null -eq $p -or $p.ExecutablePath -ne 'C:\DCMQuickSortCollisionAcceptance\dcm-docx-lifecycle-test.exe' -or $p.CommandLine -notmatch 'TraceWordLifecycle') {throw "Refusing non-fixture PID $TargetPid"}
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class DcmWordProbe {
 [StructLayout(LayoutKind.Sequential)] public struct RECT {public int Left,Top,Right,Bottom;}
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern IntPtr FindWindowEx(IntPtr parent, IntPtr childAfter, string className, string windowName);
 [DllImport("user32.dll")] public static extern bool GetClientRect(IntPtr h, out RECT r);
 [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
 [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr insert, int x, int y, int w, int height, uint flags);
 [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
 [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
}
'@
$main=(Get-Process -Id $TargetPid).MainWindowHandle
if ($main -eq [IntPtr]::Zero) {throw 'No main window'}
$pane=[DcmWordProbe]::FindWindowEx($main,[IntPtr]::Zero,'QuickSortPreview',$null)
if ($pane -eq [IntPtr]::Zero -or [DcmWordProbe]::FindWindowEx($main,$pane,'QuickSortPreview',$null) -ne [IntPtr]::Zero) {throw 'Ambiguous preview pane'}
$outer=[DcmWordProbe]::FindWindowEx($pane,[IntPtr]::Zero,'_WwB',$null)
if ($outer -eq [IntPtr]::Zero -or [DcmWordProbe]::FindWindowEx($pane,$outer,'_WwB',$null) -ne [IntPtr]::Zero) {throw 'Ambiguous Word outer'}
[uint32]$outerPid=0
[void][DcmWordProbe]::GetWindowThreadProcessId($outer,[ref]$outerPid)
if ($outerPid -eq $TargetPid -or $outerPid -eq 0) {throw 'Unexpected Word process ownership'}
$rc=New-Object DcmWordProbe+RECT
$or=New-Object DcmWordProbe+RECT
[void][DcmWordProbe]::GetClientRect($pane,[ref]$rc)
[void][DcmWordProbe]::GetWindowRect($outer,[ref]$or)
if ($rc.Right -lt 200 -or $rc.Bottom -lt 100){throw 'Invalid dimensions'}
Write-Output ("Pane={0}x{1}, Word outer={2}x{3}, Word pid={4}" -f $rc.Right,$rc.Bottom,($or.Right-$or.Left),($or.Bottom-$or.Top),$outerPid)
$flags=0x4214
if ($Nudge) {
 $ok=[DcmWordProbe]::SetWindowPos($outer,[IntPtr]::Zero,0,0,($rc.Right-1),($rc.Bottom-1),$flags)
 Write-Output ("Word outer 1px nudge accepted: {0}" -f $ok)
 Start-Sleep -Milliseconds 150
}
$result=[DcmWordProbe]::SetWindowPos($outer,[IntPtr]::Zero,0,0,$rc.Right,$rc.Bottom,$flags)
Write-Output ("Word outer exact-size request accepted: {0}" -f $result)
