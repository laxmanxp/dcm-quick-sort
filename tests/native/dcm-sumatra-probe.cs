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

    [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll",EntryPoint="GetWindowLongPtrW")] static extern IntPtr GetStyle(IntPtr h,int index);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] static extern IntPtr SendValue(IntPtr h,uint m,IntPtr w,IntPtr l,uint flags,uint ms,out IntPtr result);
    public sealed class ToolbarItem { public long Window; public string Class; public int X,Y,Width,Height; public bool Numeric,Inside; }
    public sealed class ToolbarLayout { public long Window; public int Width,Height; public ToolbarItem[] Items; public bool PageFieldFound,AllInside; }
    public static ToolbarLayout Layout(IntPtr host,int hostPID,int expectedPID){
        var v=Find(host,hostPID);if(v==null||v.PID!=expectedPID)throw new Exception("Viewer identity mismatch");
        IntPtr toolbar=IntPtr.Zero;
        EnumChildWindows((IntPtr)v.Window,delegate(IntPtr h,IntPtr ignored){uint p;GetWindowThreadProcessId(h,out p);if(p==(uint)expectedPID&&Class(h)=="ToolbarWindow32"&&IsWindowVisible(h)){if(toolbar!=IntPtr.Zero)throw new Exception("Multiple viewer toolbars");toolbar=h;}return true;},IntPtr.Zero);
        if(toolbar==IntPtr.Zero)throw new Exception("Viewer toolbar absent");
        IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try{
            RECT t;if(!GetWindowRect(toolbar,out t))throw new Exception("Toolbar bounds unavailable");
            var list=new List<ToolbarItem>();var result=new ToolbarLayout{Window=toolbar.ToInt64(),Width=t.R-t.L,Height=t.B-t.T,AllInside=true};
            EnumChildWindows(toolbar,delegate(IntPtr h,IntPtr ignored){
                uint p;GetWindowThreadProcessId(h,out p);string c=Class(h);
                if(p!=(uint)expectedPID||GetParent(h)!=toolbar||!IsWindowVisible(h)||(c!="Edit"&&c!="Static"))return true;
                RECT b;if(!GetWindowRect(h,out b))throw new Exception("Child bounds unavailable");if(b.R<=b.L||b.B<=b.T)return true;
                bool numeric=c=="Edit"&&(GetStyle(h,-16).ToInt64()&0x2000)!=0;
                bool inside=b.T>=t.T&&b.B<=t.B&&b.L>=t.L&&b.R<=t.R;
                list.Add(new ToolbarItem{Window=h.ToInt64(),Class=c,X=b.L-t.L,Y=b.T-t.T,Width=b.R-b.L,Height=b.B-b.T,Numeric=numeric,Inside=inside});
                result.PageFieldFound|=numeric;result.AllInside&=inside;return true;
            },IntPtr.Zero);result.Items=list.ToArray();return result;
        }finally{SetThreadDpiAwarenessContext(old);}
    }
    public static void RequireToolbar(ToolbarLayout layout){
        if(!layout.PageFieldFound)throw new Exception("SUMATRA_PAGE_FIELD_MISSING");
        if(!layout.AllInside)throw new Exception("SUMATRA_TOOLBAR_CLIPPED: visible edit/label is outside toolbar bounds");
    }

    [DllImport("oleacc.dll")] static extern int AccessibleObjectFromWindow(IntPtr hwnd,uint id,ref Guid iid,[MarshalAs(UnmanagedType.Interface)] out object obj);
    public sealed class AccessibleButton { public int Child,State,X,Y,Width,Height;public string Name; }
    public static AccessibleButton[] ToolbarButtons(IntPtr host,int hostPID,int expectedPID){
        var l=Layout(host,hostPID,expectedPID);RequireToolbar(l);
        object obj;Guid iid=new Guid("618736e0-3c3d-11cf-810c-00aa00389b71");
        if(AccessibleObjectFromWindow((IntPtr)l.Window,0xfffffffc,ref iid,out obj)!=0||obj==null)throw new Exception("Toolbar accessibility unavailable");
        IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try{
            var a=(Accessibility.IAccessible)obj;var buttons=new List<AccessibleButton>();int count=a.accChildCount;
            if(count<1||count>64)throw new Exception("Toolbar accessibility bounds");
            for(int i=1;i<=count;i++){int x,y,w,h;a.accLocation(out x,out y,out w,out h,i);buttons.Add(new AccessibleButton{Child=i,State=Convert.ToInt32(a.get_accState(i)),Name=a.get_accName(i)??"",X=x,Y=y,Width=w,Height=h});}
            return buttons.ToArray();
        }finally{SetThreadDpiAwarenessContext(old);Marshal.ReleaseComObject(obj);}
    }
    public static void InvokeToolbar(IntPtr host,int hostPID,int expectedPID,string prefix){
        if(prefix!="Next Page"&&prefix!="Previous Page"&&prefix!="Zoom In"&&prefix!="Zoom Out"&&prefix!="Fit a Single Page"&&prefix!="Fit Width and Show Pages Continuously")throw new Exception("Toolbar action not allowed");
        AccessibleButton found=null;foreach(var b in ToolbarButtons(host,hostPID,expectedPID)){if(b.Name.StartsWith(prefix,StringComparison.Ordinal)){if(found!=null)throw new Exception("Ambiguous toolbar action");found=b;}}
        if(found==null||(found.State&0x8001)!=0||found.Width<=0||found.Height<=0)throw new Exception("Toolbar action not available: "+prefix);
        var l=Layout(host,hostPID,expectedPID);RequireToolbar(l);object obj;Guid iid=new Guid("618736e0-3c3d-11cf-810c-00aa00389b71");
        if(AccessibleObjectFromWindow((IntPtr)l.Window,0xfffffffc,ref iid,out obj)!=0)throw new Exception("Toolbar accessibility changed");
        try{var a=(Accessibility.IAccessible)obj;if(!(a.get_accName(found.Child)??"").StartsWith(prefix,StringComparison.Ordinal))throw new Exception("Toolbar target changed");a.accDoDefaultAction(found.Child);}finally{Marshal.ReleaseComObject(obj);}
    }
    public static void SetPage(IntPtr host,int hostPID,int expectedPID,int page){
        if(page<1||page>2)throw new Exception("Fixture page outside bounds");var l=Layout(host,hostPID,expectedPID);RequireToolbar(l);
        foreach(var item in l.Items){if(!item.Numeric)continue;IntPtr r;
            if(SendText((IntPtr)item.Window,12,IntPtr.Zero,page.ToString(),2,2000,out r)==IntPtr.Zero)throw new Exception("Page text timeout");
            if(!PostMessage((IntPtr)item.Window,0x102,(IntPtr)13,IntPtr.Zero))throw new Exception("Page Enter failed");return;
        }throw new Exception("No numeric page input");
    }
}
