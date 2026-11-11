// Test-only bounded readback of the real read-only preview EDIT. The general
// adapter's 32-KiB text guard is retained; only this probe reads up to 600,000 chars.
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
public static class DcmTextProbe {
    public sealed class Item { public long Handle; public string Class,Text; public bool Visible; }
    private delegate bool EnumProc(IntPtr h,IntPtr p);
    [DllImport("user32.dll")] private static extern bool EnumChildWindows(IntPtr h,EnumProc cb,IntPtr p);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsChild(IntPtr root,IntPtr h);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr h,StringBuilder b,int n);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] private static extern IntPtr Send(IntPtr h,uint m,IntPtr w,IntPtr l,uint f,uint ms,out IntPtr r);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] private static extern IntPtr Read(IntPtr h,uint m,IntPtr w,StringBuilder b,uint f,uint ms,out IntPtr r);
    private static void Own(IntPtr h,int pid){uint actual;GetWindowThreadProcessId(h,out actual);if(h==IntPtr.Zero || actual!=(uint)pid)throw new Exception("Text probe ownership mismatch");}
    private static string Class(IntPtr h){var b=new StringBuilder(128);GetClassName(h,b,b.Capacity);return b.ToString();}
    private static string Text(IntPtr h){IntPtr n,r;if(Send(h,0x000E,IntPtr.Zero,IntPtr.Zero,2,2000,out n)==IntPtr.Zero || n.ToInt64()<0 || n.ToInt64()>600000)throw new Exception("Text probe size/deadline");var b=new StringBuilder((int)n+2);if(Read(h,0x000D,(IntPtr)b.Capacity,b,2,2000,out r)==IntPtr.Zero || r.ToInt64()!=n.ToInt64())throw new Exception("Text probe incomplete read");return b.ToString();}
    public static Item[] Controls(IntPtr root,int pid){Own(root,pid);var all=new List<Item>();EnumChildWindows(root,delegate(IntPtr h,IntPtr p){Own(h,pid);all.Add(new Item{Handle=h.ToInt64(),Class=Class(h),Text=Text(h),Visible=IsWindowVisible(h)});return true;},IntPtr.Zero);return all.ToArray();}
    public static Item Preview(IntPtr root,int pid){Own(root,pid);IntPtr pane=IntPtr.Zero;EnumChildWindows(root,delegate(IntPtr h,IntPtr p){if(Class(h)=="QuickSortPreview"){if(pane!=IntPtr.Zero)throw new Exception("Multiple preview panes");pane=h;}return true;},IntPtr.Zero);if(pane==IntPtr.Zero)throw new Exception("No preview pane");Item found=null;foreach(var c in Controls(pane,pid)){if(c.Visible && c.Class=="Edit"){if(found!=null)throw new Exception("Multiple visible preview edits");found=c;}}return found;}
    public static void ScrollEnd(IntPtr root,int pid,IntPtr edit){Own(root,pid);Own(edit,pid);if(!IsChild(root,edit) || Class(edit)!="Edit" || !IsWindowVisible(edit))throw new Exception("Not an owned visible text edit");IntPtr r;if(Send(edit,0x00B1,(IntPtr)(-1),(IntPtr)(-1),2,2000,out r)==IntPtr.Zero || Send(edit,0x00B7,IntPtr.Zero,IntPtr.Zero,2,2000,out r)==IntPtr.Zero)throw new Exception("Text scroll deadline");}
}
