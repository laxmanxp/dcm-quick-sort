// Test-only cross-process inspection. Only the recorded Sumatra PID embedded
// immediately inside QuickSortPreview is accepted; no unrelated windows touched.
using System;
using System.Collections.Generic;
using System.Drawing;
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
        if(command!=201 && command!=208 && command!=259 && command!=260 && command!=261 && command!=262 && command!=274 && command!=275 && command!=276)throw new Exception("Command not in test allowlist");
        var v=Find(host,hostPID);if(v==null||v.PID!=expectedPID)throw new Exception("Viewer identity mismatch");
        if(!PostMessage((IntPtr)v.Window,0x111,(IntPtr)command,IntPtr.Zero))throw new Exception("Owned command failed");
    }
    [DllImport("user32.dll")] static extern bool IsIconic(IntPtr h);
    public static void CheckGeometry(IntPtr host,int hostPID,int expectedPID){
        var v=Find(host,hostPID);if(v==null||v.PID!=expectedPID)throw new Exception("Viewer absent");
        IntPtr panel=Panel(host,hostPID);
        IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));try{RECT p,c;GetWindowRect(panel,out p);GetWindowRect((IntPtr)v.Window,out c);if(c.L<p.L||c.T<p.T||c.R>p.R||c.B>p.B||c.R-c.L<200||c.B-c.T<100)throw new Exception(String.Format("Embedded viewer escaped preview bounds: host_minimized={0}; host_visible={1}; panel=({2},{3},{4},{5}); viewer=({6},{7},{8},{9}); viewer_pid={10}",IsIconic(host),IsWindowVisible(host),p.L,p.T,p.R,p.B,c.L,c.T,c.R,c.B,expectedPID));}finally{SetThreadDpiAwarenessContext(old);}
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
    [DllImport("oleacc.dll")] static extern int AccessibleChildren(Accessibility.IAccessible parent,int start,int count,[Out,MarshalAs(UnmanagedType.LPArray,SizeParamIndex=2)] object[] children,out int obtained);
    public sealed class AccessibleButton { public int Child,State,X,Y,Width,Height;public string Name; }
    public static AccessibleButton[] ToolbarButtons(IntPtr host,int hostPID,int expectedPID){
        var l=Layout(host,hostPID,expectedPID);RequireToolbar(l);
        object obj=null;
        // Establish the physical-coordinate context before creating the proxy.
        IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try{
            Guid iid=new Guid("618736e0-3c3d-11cf-810c-00aa00389b71");
            if(AccessibleObjectFromWindow((IntPtr)l.Window,0xfffffffc,ref iid,out obj)!=0||obj==null)throw new Exception("Toolbar accessibility unavailable");
            var a=(Accessibility.IAccessible)obj;var buttons=new List<AccessibleButton>();int count=a.accChildCount;
            if(count<1||count>64)throw new Exception("Toolbar accessibility bounds");
            var children=new object[count];int obtained;
            int hr=AccessibleChildren(a,0,count,children,out obtained);if(hr<0||obtained<1||obtained>count)throw new Exception("Toolbar child enumeration failed");
            try{
                foreach(object child in children){
                    if(!(child is int))continue;int id=(int)child;string name=a.get_accName(id)??"";if(name.Length==0)continue;
                    int state=Convert.ToInt32(a.get_accState(id));int x=0,y=0,w=0,h=0;
                    if((state&0x8000)==0)a.accLocation(out x,out y,out w,out h,id);
                    buttons.Add(new AccessibleButton{Child=id,State=state,Name=name,X=x,Y=y,Width=w,Height=h});
                }
            }finally{foreach(object child in children){if(child!=null&&Marshal.IsComObject(child))Marshal.ReleaseComObject(child);}}
            return buttons.ToArray();
        }finally{if(obj!=null&&Marshal.IsComObject(obj))Marshal.ReleaseComObject(obj);SetThreadDpiAwarenessContext(old);}
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

    [DllImport("user32.dll")] static extern bool SetWindowPos(IntPtr h,IntPtr after,int x,int y,int w,int height,uint flags);
    [DllImport("user32.dll")] static extern IntPtr GetWindowDpiAwarenessContext(IntPtr h);
    public static RECT HostBounds(IntPtr host,int pid){
        OwnHost(host,pid);IntPtr old=SetThreadDpiAwarenessContext(GetWindowDpiAwarenessContext(host));
        try{RECT r;if(!GetWindowRect(host,out r))throw new Exception("Host bounds missing");return r;}finally{SetThreadDpiAwarenessContext(old);}
    }
    public static void ResizeHost(IntPtr host,int pid,int width,int height){
        OwnHost(host,pid);if(width<1000||width>1500||height<650||height>900)throw new Exception("Test size outside bounds");
        IntPtr old=SetThreadDpiAwarenessContext(GetWindowDpiAwarenessContext(host));
        try{if(!SetWindowPos(host,IntPtr.Zero,0,0,width,height,0x4016))throw new Exception("Owned resize failed");}finally{SetThreadDpiAwarenessContext(old);}
    }

    // Read-only full-toolbar coverage: unlike Layout, this includes Aa and icons.
    public sealed class ButtonCoverage { public int Visible; public string[] Names,Clipped; public bool Passed; public RECT Toolbar,Frame; public AccessibleButton[] Buttons; }
    public static ButtonCoverage MeasureButtons(IntPtr host,int hostPID,int expectedPID){
        var v=Find(host,hostPID);if(v==null||v.PID!=expectedPID)throw new Exception("Viewer identity mismatch");
        var l=Layout(host,hostPID,expectedPID);RequireToolbar(l);
        var buttons=ToolbarButtons(host,hostPID,expectedPID);
        var names=new List<string>();var clipped=new List<string>();RECT t,f;
        IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try{
            if(!GetWindowRect((IntPtr)l.Window,out t)||!GetWindowRect((IntPtr)v.Window,out f))throw new Exception("Toolbar/frame bounds unavailable");
            int left=Math.Max(t.L,f.L),top=Math.Max(t.T,f.T),right=Math.Min(t.R,f.R),bottom=Math.Min(t.B,f.B);
            foreach(var b in buttons){
                if((b.State&0x8000)!=0)continue; // invisible, e.g. denied Print/Open
                names.Add(b.Name);
                if(b.Width<=0||b.Height<=0||b.X<left||b.Y<top||b.X+b.Width>right||b.Y+b.Height>bottom||(b.State&0x10000)!=0)clipped.Add(b.Name);
            }
        }finally{SetThreadDpiAwarenessContext(old);}
        if(names.Count<8)throw new Exception("Insufficient visible toolbar controls");
        return new ButtonCoverage{Visible=names.Count,Names=names.ToArray(),Clipped=clipped.ToArray(),Passed=clipped.Count==0,Toolbar=t,Frame=f,Buttons=buttons};
    }
    [StructLayout(LayoutKind.Sequential)] public struct POINT { public int X,Y; }
    [DllImport("user32.dll")] static extern bool ClientToScreen(IntPtr h,ref POINT p);
    public sealed class BackgroundCoverage { public int Samples,BadPixels,Masks;public RECT Client;public POINT[] FirstBad; public bool Passed; }
    // Inspect the captured host's bare client background only. Current visible
    // direct-child rectangles (including the whole foreign-viewer container)
    // are excluded. No repaint, input, memory reading, or process modification.
    public static BackgroundCoverage MeasureBackground(IntPtr host,int pid,string path,int captureLeft,int captureTop){
        OwnHost(host,pid);IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try{
            RECT c;if(!GetClientRect(host,out c))throw new Exception("Host client unavailable");
            var a=new POINT{X=0,Y=0};var b=new POINT{X=c.R,Y=c.B};
            if(!ClientToScreen(host,ref a)||!ClientToScreen(host,ref b))throw new Exception("Host coordinate conversion failed");
            c=new RECT{L=a.X,T=a.Y,R=b.X,B=b.Y};
            var masks=new List<RECT>();
            EnumChildWindows(host,delegate(IntPtr h,IntPtr ignored){
                if(GetParent(h)!=host||!IsWindowVisible(h))return true;
                OwnHost(h,pid);RECT r;if(!GetWindowRect(h,out r))throw new Exception("Control rectangle unavailable");
                if(r.R>r.L&&r.B>r.T)masks.Add(new RECT{L=r.L-4,T=r.T-4,R=r.R+4,B=r.B+4});return true;
            },IntPtr.Zero);
            if(masks.Count<20)throw new Exception("Insufficient host-control masks");
            var result=new BackgroundCoverage{Client=c,Masks=masks.Count};var first=new List<POINT>();
            using(var image=new Bitmap(path)){
                if(c.L<captureLeft||c.T<captureTop||c.R>captureLeft+image.Width||c.B>captureTop+image.Height)throw new Exception("Host client escaped capture");
                for(int y=c.T+4;y<c.B-4;y+=2)for(int x=c.L+4;x<c.R-4;x+=2){
                    bool covered=false;foreach(var r in masks){if(x>=r.L&&x<r.R&&y>=r.T&&y<r.B){covered=true;break;}}if(covered)continue;
                    Color actual=image.GetPixel(x-captureLeft,y-captureTop);result.Samples++;
                    // Pinned host brush colorWindowBG = rgb(245,247,250).
                    if(Math.Abs(actual.R-245)>3||Math.Abs(actual.G-247)>3||Math.Abs(actual.B-250)>3){
                        result.BadPixels++;if(first.Count<16)first.Add(new POINT{X=x-c.L,Y=y-c.T});
                    }
                }
            }
            if(result.Samples<1000)throw new Exception("Insufficient bare-background samples");
            result.FirstBad=first.ToArray();result.Passed=result.BadPixels==0;return result;
        }finally{SetThreadDpiAwarenessContext(old);}
    }

    [DllImport("user32.dll")] static extern bool RedrawWindow(IntPtr h,IntPtr rect,IntPtr region,uint flags);
    public static void PaintReference(IntPtr host,int pid){
        OwnHost(host,pid);
        // Diagnostic reference only, AFTER the unassisted screenshot. No
        // synchronous cross-process update and no document/control changes.
        if(!RedrawWindow(host,IntPtr.Zero,IntPtr.Zero,0x0085))throw new Exception("Reference repaint request failed");
    }
    public sealed class PaintDifference { public int Samples,ChangedPixels;public POINT[] FirstChanged;public bool Passed; }
    public static PaintDifference CompareHostPaint(IntPtr host,int pid,string raw,string reference,int captureLeft,int captureTop){
        OwnHost(host,pid);IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try{
            RECT c;if(!GetClientRect(host,out c))throw new Exception("Host client unavailable");
            var a=new POINT{X=0,Y=0};var b=new POINT{X=c.R,Y=c.B};
            if(!ClientToScreen(host,ref a)||!ClientToScreen(host,ref b))throw new Exception("Host coordinate conversion failed");
            var masks=new List<RECT>();
            EnumChildWindows(host,delegate(IntPtr h,IntPtr ignored){
                if(GetParent(h)!=host||!IsWindowVisible(h))return true;
                OwnHost(h,pid);string cls=Class(h);
                if(cls!="QuickSortPreview"&&cls!="Edit"&&cls!="ComboBox"&&cls!="ListBox"&&cls!="SysTreeView32")return true;
                RECT m;if(!GetWindowRect(h,out m))throw new Exception("Mask geometry unavailable");
                masks.Add(new RECT{L=m.L-4,T=m.T-4,R=m.R+4,B=m.B+4});return true;
            },IntPtr.Zero);
            var result=new PaintDifference();var first=new List<POINT>();
            using(var x=new Bitmap(raw))using(var y=new Bitmap(reference)){
                if(x.Size!=y.Size||a.X<captureLeft||a.Y<captureTop||b.X>captureLeft+x.Width||b.Y>captureTop+x.Height)throw new Exception("Reference capture geometry changed");
                for(int py=a.Y+4;py<b.Y-4;py+=2)for(int px=a.X+4;px<b.X-4;px+=2){
                    bool covered=false;foreach(var m in masks){if(px>=m.L&&px<m.R&&py>=m.T&&py<m.B){covered=true;break;}}if(covered)continue;
                    Color before=x.GetPixel(px-captureLeft,py-captureTop),after=y.GetPixel(px-captureLeft,py-captureTop);result.Samples++;
                    if(Math.Abs(before.R-after.R)>3||Math.Abs(before.G-after.G)>3||Math.Abs(before.B-after.B)>3){result.ChangedPixels++;if(first.Count<16)first.Add(new POINT{X=px-a.X,Y=py-a.Y});}
                }
            }
            if(result.Samples<1000)throw new Exception("Insufficient host repaint samples");
            result.FirstChanged=first.ToArray();result.Passed=result.ChangedPixels==0;return result;
        }finally{SetThreadDpiAwarenessContext(old);}
    }
}
