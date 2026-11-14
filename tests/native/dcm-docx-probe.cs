// Read-only observation limited to the preview subtree of the exact test host.
// Does not relax the general adapter's same-PID input or screenshot guards.
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
using System.Windows.Automation;
using System.Threading;
public static class DcmDocxProbe {
 public sealed class Node {public long Handle,Parent;public int PID;public string Class,Name;public bool Visible;public int Left,Top,Right,Bottom;}
 [StructLayout(LayoutKind.Sequential)] struct R {public int Left,Top,Right,Bottom;}
 delegate bool EnumProc(IntPtr h,IntPtr p);
 [DllImport("user32.dll")] static extern bool EnumChildWindows(IntPtr h,EnumProc cb,IntPtr p);
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
 [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr h);
 [DllImport("user32.dll")] static extern bool IsChild(IntPtr root,IntPtr h);
 [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h,StringBuilder b,int n);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetWindowText(IntPtr h,StringBuilder b,int n);
 [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h,out R r);
 [DllImport("user32.dll")] static extern IntPtr GetWindowDpiAwarenessContext(IntPtr h);
 [DllImport("user32.dll")] static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
 static string Class(IntPtr h){var b=new StringBuilder(128);GetClassName(h,b,b.Capacity);return b.ToString();}
 static void Own(IntPtr h,int pid){uint p;GetWindowThreadProcessId(h,out p);if(h==IntPtr.Zero || p!=(uint)pid)throw new Exception("DOCX host identity mismatch");}
 public static IntPtr Pane(IntPtr host,int pid){Own(host,pid);IntPtr found=IntPtr.Zero;EnumChildWindows(host,delegate(IntPtr h,IntPtr p){if(Class(h)=="QuickSortPreview" && GetParent(h)==host){Own(h,pid);if(found!=IntPtr.Zero)throw new Exception("Duplicate preview");found=h;}return true;},IntPtr.Zero);if(found==IntPtr.Zero)throw new Exception("No owned preview");return found;}
 public static Node[] Snapshot(IntPtr host,int pid){IntPtr pane=Pane(host,pid);var result=new List<Node>();IntPtr previous=SetThreadDpiAwarenessContext(GetWindowDpiAwarenessContext(host));try{EnumChildWindows(pane,delegate(IntPtr h,IntPtr p){if(result.Count>=100 || !IsChild(pane,h))throw new Exception("Preview observation bounds");uint owner;GetWindowThreadProcessId(h,out owner);R r;if(!GetWindowRect(h,out r))throw new Exception("Preview rectangle unavailable");var b=new StringBuilder(1024);GetWindowText(h,b,b.Capacity);result.Add(new Node{Handle=h.ToInt64(),Parent=GetParent(h).ToInt64(),PID=(int)owner,Class=Class(h),Name=b.ToString(),Visible=IsWindowVisible(h),Left=r.Left,Top=r.Top,Right=r.Right,Bottom=r.Bottom});return true;},IntPtr.Zero);}finally{SetThreadDpiAwarenessContext(previous);}return result.ToArray();}
 static string ReadTextCore(IntPtr host,int pid){IntPtr pane=Pane(host,pid);var root=AutomationElement.FromHandle(pane);var all=root.FindAll(TreeScope.Descendants,Condition.TrueCondition);if(all.Count>200)throw new Exception("Too many DOCX accessibility nodes");var text=new StringBuilder();foreach(AutomationElement e in all){if(e.Current.IsOffscreen)continue;object value;if(e.TryGetCurrentPattern(TextPattern.Pattern,out value)){string s=((TextPattern)value).DocumentRange.GetText(20000);if(!String.IsNullOrEmpty(s)){text.AppendLine(s);}}else{string name=e.Current.Name;if(!String.IsNullOrEmpty(name))text.AppendLine(name);}if(text.Length>60000)throw new Exception("DOCX accessibility text bound");}return text.ToString();}
 public static string ReadAccessibleText(IntPtr host,int pid){string result=null;Exception error=null;var worker=new Thread(delegate(){try{result=ReadTextCore(host,pid);}catch(Exception e){error=e;}});worker.IsBackground=true;worker.SetApartmentState(ApartmentState.MTA);worker.Start();if(!worker.Join(5000))return "[Accessibility observation timed out; visual review required]";if(error!=null)return "[Accessibility observation unavailable: "+error.GetType().Name+"]";return result??"";}
 public static long[] HostStatics(IntPtr host,int pid){Own(host,pid);var found=new List<long>();EnumChildWindows(host,delegate(IntPtr h,IntPtr p){if(GetParent(h)==host && Class(h)=="Static" && IsWindowVisible(h)){Own(h,pid);found.Add(h.ToInt64());}return true;},IntPtr.Zero);return found.ToArray();}
}
