param([Parameter(Mandatory=$true)][int]$TargetPid)
$ErrorActionPreference='Stop'
$exe='C:\DCMQuickSortCollisionAcceptance\dcm-image-buffered-scroll-dev.exe'
$file='C:\DCMQuickSortCollisionAcceptance\TraceImagePreview\wide-test.png'
$p=Get-CimInstance Win32_Process -Filter "ProcessId = $TargetPid"
if($null -eq $p -or $p.ExecutablePath -ne $exe -or $p.CommandLine -notmatch 'wide-test.png'){throw 'Refusing non-disposable image process'}
$before=(Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class DcmImageWheelProbe {
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern IntPtr FindWindowEx(IntPtr p,IntPtr after,string cl,string title);
 [DllImport("user32.dll")] public static extern IntPtr GetDlgItem(IntPtr p,int id);
 [DllImport("user32.dll")] public static extern IntPtr SendMessage(IntPtr h,uint msg,IntPtr w,IntPtr l);
}
'@
$main=(Get-Process -Id $TargetPid).MainWindowHandle
if ($main -eq [IntPtr]::Zero){throw 'Main window not ready'}
$pane=[DcmImageWheelProbe]::FindWindowEx($main,[IntPtr]::Zero,'QuickSortPreview',$null)
if ($pane -eq [IntPtr]::Zero){throw 'Image preview host missing'}
$zoom=[DcmImageWheelProbe]::GetDlgItem($pane,1085)
if ($zoom -eq [IntPtr]::Zero){throw 'Image zoom control missing'}
for($i=0;$i -lt 7;$i++){[void][DcmImageWheelProbe]::SendMessage($zoom,0x00F5,[IntPtr]::Zero,[IntPtr]::Zero)}
$brush=[DcmImageWheelProbe]::SendMessage($pane,0x0014,[IntPtr]::Zero,[IntPtr]::Zero)
Write-Output ('WM_ERASEBKGND handled: '+($brush -eq [IntPtr]::new(1)))
if ($brush -ne [IntPtr]::new(1)){throw 'Image background erasure was not suppressed'}
for($i=0;$i -lt 35;$i++){
 [void][DcmImageWheelProbe]::SendMessage($pane,0x020A,[IntPtr]::new([long]0xFF880000),[IntPtr]::Zero)
}
if((Get-Process -Id $TargetPid -ErrorAction SilentlyContinue) -eq $null){throw 'App exited during wheel sequence'}
$after=(Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash
Write-Output ('Wheel steps accepted: 35')
Write-Output ('Source image unchanged: '+($before -eq $after))
if($before -ne $after){throw 'Source image changed'}
