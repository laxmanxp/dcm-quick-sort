// Test-only host inspection. Reads only children owned by the exact QuickSort
// process, deliberately skipping the embedded foreign-process PDF tree.
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
public static class DcmLifecycleProbe {
    delegate bool EnumProc(IntPtr h, IntPtr p);
    [DllImport("user32.dll")] static extern bool EnumChildWindows(IntPtr h, EnumProc f, IntPtr p);
    [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h, StringBuilder b, int n);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] static extern IntPtr Send(IntPtr h,uint m,IntPtr w,IntPtr l,uint flags,uint ms,out IntPtr r);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] static extern IntPtr Read(IntPtr h,uint m,IntPtr w,StringBuilder b,uint flags,uint ms,out IntPtr r);
    static void Own(IntPtr h,int pid) {uint actual;GetWindowThreadProcessId(h,out actual);if(h==IntPtr.Zero||actual!=(uint)pid)throw new Exception("Lifecycle host identity mismatch");}
    static string Text(IntPtr h) {IntPtr n,r;if(Send(h,14,IntPtr.Zero,IntPtr.Zero,2,1500,out n)==IntPtr.Zero||n.ToInt64()<0||n.ToInt64()>32768)throw new Exception("Lifecycle text length/deadline");var b=new StringBuilder((int)n+2);if(Read(h,13,(IntPtr)b.Capacity,b,2,1500,out r)==IntPtr.Zero)throw new Exception("Lifecycle text read timeout");return b.ToString();}
    public static string[] Status(IntPtr host,int pid) {
        Own(host,pid);var texts=new List<string>();
        EnumChildWindows(host,delegate(IntPtr h,IntPtr unused){uint actual;GetWindowThreadProcessId(h,out actual);if(actual!=(uint)pid||!IsWindowVisible(h))return true;var b=new StringBuilder(128);GetClassName(h,b,b.Capacity);if(b.ToString()=="Static"){string s=Text(h);if(s.StartsWith("Loading PDF",StringComparison.Ordinal)||s.StartsWith("Sumatra",StringComparison.Ordinal)||s.StartsWith("Plain-text preview",StringComparison.Ordinal))texts.Add(s);}return true;},IntPtr.Zero);
        return texts.ToArray();
    }
}
