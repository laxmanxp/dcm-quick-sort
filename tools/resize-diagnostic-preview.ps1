param(
 [Parameter(Mandatory=$true)][int]$TargetPid,
 [ValidateRange(900,1800)][int]$Width=1100,
 [ValidateRange(680,1100)][int]$Height=760
)
$ErrorActionPreference="Stop"
$p=Get-Process -Id $TargetPid -ErrorAction Stop
$approvedWord = $p.ProcessName -eq "dcm-word-dpi-aware-test"
$approvedHTML = $p.ProcessName -eq "dcm-html-host-repair-dev" -and
 (Get-CimInstance Win32_Process -Filter "ProcessId = $TargetPid").CommandLine -match "TraceHTMLHostRepair"
$approvedDOCX = $p.ProcessName -eq "dcm-docx-lifecycle-test" -and
 (Get-CimInstance Win32_Process -Filter "ProcessId = $TargetPid").CommandLine -match "TraceWordLifecycle"
if (-not ($approvedWord -or $approvedHTML -or $approvedDOCX)) {throw "Refusing to resize non-diagnostic process"}
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class DcmResize {
 [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h,IntPtr insert,int x,int y,int w,int height,uint flags);
}
'@
if ($p.MainWindowHandle -eq [IntPtr]::Zero) {throw "No main diagnostic window"}
$ok=[DcmResize]::SetWindowPos($p.MainWindowHandle,[IntPtr]::Zero,80,60,$Width,$Height,0x214)
Write-Output ("Diagnostic PID={0}; resize request={1}x{2}; accepted={3}" -f $TargetPid,$Width,$Height,$ok)
