// Isolated Windows acceptance: locate only the owned QuickSort Destination Manager.
using System;
using System.Runtime.InteropServices;
public static class DcmDestinationManagerProbe {
 [DllImport("user32.dll",CharSet=CharSet.Unicode,EntryPoint="FindWindowW")]
 static extern IntPtr FindWindow(string className,string windowName);
 [DllImport("user32.dll")] static extern IntPtr GetWindow(IntPtr hwnd,uint command);
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr hwnd,out uint pid);
 public static IntPtr FindOwned(IntPtr root,int pid) {
  var hwnd=FindWindow("QuickSortFolderDestinations",null);
  if(hwnd==IntPtr.Zero)return IntPtr.Zero;
  uint owner;GetWindowThreadProcessId(hwnd,out owner);
  if(owner!=(uint)pid || GetWindow(hwnd,4)!=root)throw new Exception("Destination Manager does not belong to disposable QuickSort");
  return hwnd;
 }
}
