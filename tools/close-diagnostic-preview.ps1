param([Parameter(Mandatory=$true)][int]$TargetPid)
$ErrorActionPreference="Stop"
$process=Get-Process -Id $TargetPid -ErrorAction Stop
if ($process.ProcessName -notin @("dcm-viewport-resize-dev","dcm-word-dpi-experiment-dev","dcm-word-dpi-aware-test")) { throw "Refusing to close non-diagnostic process $TargetPid" }
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class DcmCloseDiagnostic {
 [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr hwnd, uint msg, IntPtr w, IntPtr l);
}
'@
$hwnd=$process.MainWindowHandle
if ($hwnd -eq [IntPtr]::Zero) {throw "Diagnostic window is unavailable"}
$sent=[DcmCloseDiagnostic]::PostMessage($hwnd,0x10,[IntPtr]::Zero,[IntPtr]::Zero)
Write-Output ("WM_CLOSE posted={0}, target PID={1}" -f $sent,$TargetPid)
