// Test-only cross-process inspection. Only the recorded Sumatra PID embedded
// immediately inside QuickSortPreview is accepted; no unrelated windows touched.
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
public static class DcmSumatraProbe {
    delegate bool EnumProc(IntPtr h,IntPtr p);
    [DllImport("user32.dll")] static extern bool EnumChildWindows(IntPtr h,EnumProc f,IntPtr p);
    [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr h);
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint p);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h,StringBuilder b,int n);
    [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    [DllImport("user32.dll")] static extern bool GetClientRect(IntPtr h,out RECT r);
    [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h,out RECT r);
    [DllImport("user32.dll")] static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L,T,R,B; }
    public sealed class Embedded { public long Window; public int PID,Width,Height; public string Class; }
    static string Class(IntPtr h){var s=new StringBuilder(128);GetClassName(h,s,s.Capacity);return s.ToString();}
    static void OwnHost(IntPtr host,int pid){uint actual;GetWindowThreadProcessId(host,out actual);if(host==IntPtr.Zero||actual!=(uint)pid)throw new Exception("Host identity mismatch");}
    static IntPtr Panel(IntPtr host,int pid){OwnHost(host,pid);IntPtr found=IntPtr.Zero;EnumChildWindows(host,delegate(IntPtr h,IntPtr p){uint actual;GetWindowThreadProcessId(h,out actual);if(actual==(uint)pid&&Class(h)=="QuickSortPreview"){if(found!=IntPtr.Zero)throw new Exception("Multiple panels");found=h;}return true;},IntPtr.Zero);if(found==IntPtr.Zero)throw new Exception("No preview panel");return found;}
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] static extern IntPtr SendText(IntPtr h,uint m,IntPtr w,string text,uint flags,uint timeout,out IntPtr result);
    public static void SetEdit(IntPtr host,int pid,int id,string text){OwnHost(host,pid);IntPtr found=IntPtr.Zero;EnumChildWindows(host,delegate(IntPtr h,IntPtr p){uint actual;GetWindowThreadProcessId(h,out actual);if(actual==(uint)pid&&GetDlgCtrlID(h)==id&&Class(h)=="Edit")found=h;return true;},IntPtr.Zero);IntPtr r;if(found==IntPtr.Zero||SendText(found,12,IntPtr.Zero,text,2,2000,out r)==IntPtr.Zero)throw new Exception("Edit target/text failed");}
    [DllImport("user32.dll")] static extern int GetDlgCtrlID(IntPtr h);
    public static Embedded Find(IntPtr host,int hostPID){
        OwnHost(host,hostPID);
        IntPtr panel=Panel(host,hostPID);
        Embedded result=null;
        EnumChildWindows(panel,delegate(IntPtr h,IntPtr p){uint pid;GetWindowThreadProcessId(h,out pid);if(GetParent(h)==panel && pid!=(uint)hostPID && Class(h)=="SUMATRA_PDF_FRAME"){
            if(result!=null)throw new Exception("Multiple Sumatra frames");
            RECT r;GetClientRect(h,out r);result=new Embedded{Window=h.ToInt64(),PID=(int)pid,Width=r.R-r.L,Height=r.B-r.T,Class=Class(h)};
        }return true;},IntPtr.Zero);return result;
    }
    public static void Command(IntPtr host,int hostPID,int expectedPID,int command){
        if(command!=201 && command!=208 && command!=259 && command!=260 && command!=261 && command!=262 && command!=274)throw new Exception("Command not in test allowlist");
        var v=Find(host,hostPID);if(v==null||v.PID!=expectedPID)throw new Exception("Viewer identity mismatch");
        if(!PostMessage((IntPtr)v.Window,0x111,(IntPtr)command,IntPtr.Zero))throw new Exception("Owned command failed");
    }
    public static void CheckGeometry(IntPtr host,int hostPID,int expectedPID){
        var v=Find(host,hostPID);if(v==null||v.PID!=expectedPID)throw new Exception("Viewer absent");
        IntPtr panel=Panel(host,hostPID);
        IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));try{RECT p,c;GetWindowRect(panel,out p);GetWindowRect((IntPtr)v.Window,out c);if(c.L<p.L||c.T<p.T||c.R>p.R||c.B>p.B||c.R-c.L<200||c.B-c.T<100)throw new Exception("Embedded viewer escaped preview bounds");}finally{SetThreadDpiAwarenessContext(old);}
    }
}
