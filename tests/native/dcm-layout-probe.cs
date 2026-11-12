// Test-only non-repainting inspection of the exact host and its direct controls.
using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Threading;
public static class DcmLayoutProbe {
 [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L,T,R,B; }
 delegate bool EnumProc(IntPtr h,IntPtr p);
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
 [DllImport("user32.dll")] static extern bool GetClientRect(IntPtr h,out RECT r);
 [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h,out RECT r);
 [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr h);
 [DllImport("user32.dll")] static extern int GetDlgCtrlID(IntPtr h);
 [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
 [DllImport("user32.dll")] static extern bool GetUpdateRect(IntPtr h,out RECT r,bool erase);
 [DllImport("user32.dll")] static extern bool EnumChildWindows(IntPtr h,EnumProc f,IntPtr p);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern IntPtr GetProp(IntPtr h,string name);
 [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 [DllImport("user32.dll")] static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 static void Own(IntPtr h,int pid){uint actual;GetWindowThreadProcessId(h,out actual);if(h==IntPtr.Zero||actual!=(uint)pid)throw new Exception("Layout probe ownership mismatch");}
 public sealed class Child { public long Window;public int ID;public RECT Bounds,Update;public bool Pending; }
 public sealed class State { public long Timestamp,Sequence,Depth,Fence;public RECT Client;public bool HostPending;public Child[] Children; }
 public static State Observe(IntPtr host,int pid){
  Own(host,pid);IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
  try {
   var s=new State{Timestamp=Stopwatch.GetTimestamp(),Sequence=GetProp(host,"DCM.Layout.Sequence").ToInt64(),Depth=GetProp(host,"DCM.Layout.Depth").ToInt64(),Fence=GetProp(host,"DCM.Layout.Fence").ToInt64()};
   GetClientRect(host,out s.Client);RECT update;s.HostPending=GetUpdateRect(host,out update,false);
   var children=new List<Child>();
   EnumChildWindows(host,delegate(IntPtr h,IntPtr p){uint actual;GetWindowThreadProcessId(h,out actual);if(GetParent(h)!=host||actual!=(uint)pid||!IsWindowVisible(h))return true;
    var c=new Child{Window=h.ToInt64(),ID=GetDlgCtrlID(h)};GetWindowRect(h,out c.Bounds);c.Pending=GetUpdateRect(h,out c.Update,false);children.Add(c);return true;},IntPtr.Zero);
   s.Children=children.ToArray();return s;
  }finally{SetThreadDpiAwarenessContext(old);}
 }
 // A posted marker confirms earlier queued size requests have been serviced.
 // It neither drains WM_PAINT nor forces a repaint; pending regions are recorded.
 public static State Fence(IntPtr host,int pid,int token){
  Own(host,pid);if(!PostMessage(host,0x83A0,(IntPtr)token,IntPtr.Zero))throw new Exception("Could not post layout fence");
  var sw=Stopwatch.StartNew();while(sw.ElapsedMilliseconds<3000){Own(host,pid);if(GetProp(host,"DCM.Layout.Fence").ToInt64()==token)return Observe(host,pid);Thread.Sleep(10);}
  throw new Exception("Layout completion fence deadline");
 }
}
