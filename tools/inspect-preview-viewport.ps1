param([int]$TargetPid = 0, [switch]$OnlyPreview)
$ErrorActionPreference = 'Stop'
Add-Type -TypeDefinition @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class DcmWindows {
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L,T,R,B; }
  public delegate bool WindowCallback(IntPtr hWnd, IntPtr param);
  [DllImport("user32.dll")] public static extern bool EnumWindows(WindowCallback callback, IntPtr param);
  [DllImport("user32.dll")] public static extern bool EnumChildWindows(IntPtr hwnd, WindowCallback callback, IntPtr param);
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint pid);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetClassName(IntPtr hwnd, StringBuilder name, int length);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr hwnd, StringBuilder name, int length);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hwnd, out RECT rect);
  [DllImport("user32.dll")] public static extern bool GetClientRect(IntPtr hwnd, out RECT rect);
  [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr hwnd);
  [DllImport("user32.dll")] public static extern uint GetDpiForWindow(IntPtr hwnd);
  [DllImport("user32.dll")] public static extern IntPtr GetWindowDpiAwarenessContext(IntPtr hwnd);
  [DllImport("user32.dll")] public static extern int GetAwarenessFromDpiAwarenessContext(IntPtr dpiContext);
  [DllImport("user32.dll")] public static extern IntPtr GetParent(IntPtr hwnd);
  [DllImport("user32.dll", EntryPoint="GetWindowLongPtrW")] public static extern IntPtr GetWindowLongPtr(IntPtr hwnd, int index);
}
'@
if ($TargetPid -eq 0) {
  $match = Get-CimInstance Win32_Process -Filter "Name = 'dcm-viewport-resize-dev.exe'" | Select-Object -First 1
  if ($null -eq $match) { throw 'Development QuickSort process was not found' }
  $TargetPid = [int]$match.ProcessId
}
$roots = New-Object System.Collections.Generic.List[System.IntPtr]
$rootCallback = [DcmWindows+WindowCallback]{
  param([IntPtr]$hwnd,[IntPtr]$param)
  [uint32]$windowPid = 0
  [void][DcmWindows]::GetWindowThreadProcessId($hwnd,[ref]$windowPid)
  if ($windowPid -eq $TargetPid) { $roots.Add($hwnd) }
  return $true
}
[void][DcmWindows]::EnumWindows($rootCallback,[IntPtr]::Zero)
function Describe([IntPtr]$h,[string]$kind) {
  $cl = New-Object System.Text.StringBuilder 256
  $tt = New-Object System.Text.StringBuilder 256
  [void][DcmWindows]::GetClassName($h,$cl,256)
  [void][DcmWindows]::GetWindowText($h,$tt,256)
  $rect = New-Object DcmWindows+RECT
  $client = New-Object DcmWindows+RECT
  [void][DcmWindows]::GetWindowRect($h,[ref]$rect)
  [void][DcmWindows]::GetClientRect($h,[ref]$client)
  $style = [DcmWindows]::GetWindowLongPtr($h,-16).ToInt64()
  [uint32]$windowOwnerPid = 0
  [void][DcmWindows]::GetWindowThreadProcessId($h,[ref]$windowOwnerPid)
  $parent = [DcmWindows]::GetParent($h)
  [pscustomobject]@{ Kind=$kind; Hwnd=$h.ToInt64(); Parent=$parent.ToInt64(); WindowPid=$windowOwnerPid; Dpi=[DcmWindows]::GetDpiForWindow($h); Awareness=[DcmWindows]::GetAwarenessFromDpiAwarenessContext([DcmWindows]::GetWindowDpiAwarenessContext($h)); Class=$cl.ToString(); Title=$tt.ToString(); Visible=[DcmWindows]::IsWindowVisible($h); Left=$rect.L; Top=$rect.T; Width=($rect.R-$rect.L); Height=($rect.B-$rect.T); ClientWidth=($client.R-$client.L); ClientHeight=($client.B-$client.T); Style=('0x{0:X}' -f $style)}
}
Write-Output "Process $TargetPid, roots $($roots.Count)"
foreach ($root in $roots) {
  $children = New-Object System.Collections.Generic.List[System.IntPtr]
  $callback = [DcmWindows+WindowCallback]{
    param([IntPtr]$hwnd,[IntPtr]$param)
    $children.Add($hwnd)
    return $true
  }
  [void][DcmWindows]::EnumChildWindows($root,$callback,[IntPtr]::Zero)
  $rootClass = New-Object System.Text.StringBuilder 256
  [void][DcmWindows]::GetClassName($root,$rootClass,256)
  if ($rootClass.ToString() -ne 'QuickSortMain') { continue }
  Describe $root 'TOP'
  Write-Output ("Descendant count: {0}" -f $children.Count)
  foreach ($h in $children) {
    $child = Describe $h 'CHILD'
    if ($OnlyPreview) {
      if ($child.Class -match 'QuickSortPreview|^_Ww|NUIScrollbar') { $child }
    } elseif ($child.Visible -or $child.Class -match 'QuickSortPreview') { $child }
  }
}
