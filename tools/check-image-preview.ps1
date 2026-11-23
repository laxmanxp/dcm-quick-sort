param([Parameter(Mandatory=$true)][int]$TargetPid)
$ErrorActionPreference='Stop'
$exe='C:\DCMQuickSortCollisionAcceptance\dcm-image-zoom-rotate-dev.exe'
$file='C:\DCMQuickSortCollisionAcceptance\TraceImagePreview\wide-test.png'
$p=Get-CimInstance Win32_Process -Filter "ProcessId = $TargetPid"
if($null -eq $p -or $p.ExecutablePath -ne $exe -or $p.CommandLine -notmatch 'wide-test.png'){throw 'Refusing non-disposable image process'}
$before=(Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class ImageViewSmoke {
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern IntPtr FindWindowEx(IntPtr p,IntPtr after,string cl,string title);
 [DllImport("user32.dll")] public static extern IntPtr GetDlgItem(IntPtr p,int id);
 [DllImport("user32.dll")] public static extern IntPtr SendMessage(IntPtr h,uint msg,IntPtr w,IntPtr l);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr h,System.Text.StringBuilder b,int max);
}
'@
$main=(Get-Process -Id $TargetPid).MainWindowHandle
if ($main -eq [IntPtr]::Zero){throw 'Image preview main window missing'}
$pane=[ImageViewSmoke]::FindWindowEx($main,[IntPtr]::Zero,'QuickSortPreview',$null)
if ($pane -eq [IntPtr]::Zero){throw 'Image preview pane missing'}
$ids=@(1080,1081,1082,1083,1084,1085,1086)
foreach($id in $ids){if([ImageViewSmoke]::GetDlgItem($pane,$id) -eq [IntPtr]::Zero){throw "Missing image toolbar control $id"}}
Write-Output 'All 7 image toolbar controls present'
foreach($id in @(1081,1080,1083,1085,1084,1082,1086)){
 $btn=[ImageViewSmoke]::GetDlgItem($pane,$id)
 [void][ImageViewSmoke]::SendMessage($btn,0x00F5,[IntPtr]::Zero,[IntPtr]::Zero)
 Start-Sleep -Milliseconds 100
 if ($id -eq 1081) {
  $static=[IntPtr]::Zero
  $found=$false
  while($true) {
   $static=[ImageViewSmoke]::FindWindowEx($main,$static,'Static',$null)
   if($static -eq [IntPtr]::Zero) {break}
   $text=New-Object System.Text.StringBuilder 256
   [void][ImageViewSmoke]::GetWindowText($static,$text,256)
   if($text.ToString() -match 'Image preview rotated') {
    Write-Output ('Rotation feedback: '+$text.ToString())
    $found=$true
    break
   }
  }
  if(-not $found){throw 'Rotation command did not update image preview feedback'}
 }
 Write-Output ("Image toolbar command accepted: "+$id)
}
$after=(Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash
Write-Output ("Source image unchanged: "+($before -eq $after))
if ($before -ne $after){throw 'Image file changed during preview'}
