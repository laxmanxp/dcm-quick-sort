. (Join-Path $PSScriptRoot 'common.ps1')
Add-Type @'
using System;using System.Runtime.InteropServices;
public static class QSCrossCheck{
 [StructLayout(LayoutKind.Sequential)]public struct POINT{public int X,Y;}
 [DllImport("user32.dll")]public static extern bool ShowWindow(IntPtr h,int command);
 [DllImport("user32.dll")]public static extern bool SetWindowPos(IntPtr h,IntPtr a,int x,int y,int w,int height,uint flags);
 [DllImport("user32.dll")]public static extern IntPtr WindowFromPoint(POINT p);
 [DllImport("user32.dll")]public static extern IntPtr GetAncestor(IntPtr h,uint flags);
}
'@
function Save-VerifiedScreen([string]$Name){
 $old=[QSEvidence]::SetThreadDpiAwarenessContext([IntPtr](-4))
 try{$r=New-Object QSEvidence+RECT;[void][QSEvidence]::GetWindowRect($hwnd,[ref]$r)
  foreach($fx in @(0.1,0.5,0.9)){foreach($fy in @(0.15,0.5,0.85)){$pt=New-Object QSCrossCheck+POINT;$pt.X=[int]($r.L+($r.R-$r.L)*$fx);$pt.Y=[int]($r.T+($r.B-$r.T)*$fy);if([QSCrossCheck]::GetAncestor([QSCrossCheck]::WindowFromPoint($pt),2) -ne $hwnd){throw 'Refusing occluded screen capture'}}}
  $b=New-Object Drawing.Bitmap(($r.R-$r.L),($r.B-$r.T));$g=[Drawing.Graphics]::FromImage($b)
  try{$g.CopyFromScreen($r.L,$r.T,0,0,$b.Size);$b.Save((Join-Path $report ($Name+'.png')),[Drawing.Imaging.ImageFormat]::Png)}finally{$g.Dispose();$b.Dispose()}
 }finally{[void][QSEvidence]::SetThreadDpiAwarenessContext($old)}
}
$topmost=(([QSEvidence]::Style($hwnd,-20).ToInt64() -band 8) -ne 0)
try{
 [void][QSCrossCheck]::ShowWindow($hwnd,9);[void][QSCrossCheck]::SetWindowPos($hwnd,[IntPtr](-1),0,0,1280,820,0x0050)
 [void](Run-Case 'pdf-100-delayed' @'
click clear
type name "08-print"
wait_ready type=textbox target=newname text="08-print" timeout=5s
delay 1200ms
click zoomreset
wait_ready type=control target=pdfstatus text~="100%" timeout=10s
delay 2500ms
inspect main
expect pdfstatus text~="100%"
'@ $null)
 Save-VerifiedScreen 'pdf-100-screen'
 [void](Run-Case 'pdf-width-delayed' @'
click width
wait_ready type=control target=pdfstatus text~="Fit Width" timeout=10s
delay 1000ms
inspect main
expect pdfstatus text~="Fit Width"
'@ $null)
 Save-VerifiedScreen 'pdf-width-screen'
 $source=Join-Path $Lab 'inbox\01-alpha.txt';$target=Join-Path $Lab 'vault\01-alpha.txt'
 if(-not(Test-Path $source)-or(Test-Path $target)){throw 'Unexpected disposable move fixture'}
 $hash=(Get-FileHash $source).Hash;$dest=Join-Path $Lab 'vault'
 $body="click clear`ntype name `"01-alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ntype destination `"$dest`"`ndelay 300ms`nclick move`ndelay 1800ms`ninspect main"
 $moved=Run-Case 'move-restored-window' $body {if((Test-Path $source)-or(-not(Test-Path $target))){throw 'Fixture was not moved to isolated vault'};if((Get-FileHash $target).Hash -ne $hash){throw 'Move content mismatch'}}
 if($moved){[void](Run-Case 'undo-move-restored-window' "click undo`ndelay 1000ms`ninspect main" {if((-not(Test-Path $source))-or(Test-Path $target)){throw 'Undo move failed'};if((Get-FileHash $source).Hash -ne $hash){throw 'Restored content mismatch'}})}
}finally{if(-not $topmost){[void][QSCrossCheck]::SetWindowPos($hwnd,[IntPtr](-2),0,0,0,0,0x0013)}}
Write-Output ('REPORT='+$report)
