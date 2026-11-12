// DCM-owned adapter for native controls missing from FlowUIVision. Every action
// is bound to the disposable process, and messages use deadlines or PostMessage.
using System;
using System.Collections.Generic;
using System.Drawing;
using System.Drawing.Imaging;
using System.Runtime.InteropServices;
using System.Text;

public static class DcmNative {
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left,Top,Right,Bottom; }
    [StructLayout(LayoutKind.Sequential)] public struct POINT { public int X,Y; public POINT(int x,int y){X=x;Y=y;} }
    public sealed class WindowInfo { public long Handle; public string Class; public string Text; public int ID; public bool Visible; public bool Enabled; }
    public sealed class CaptureInfo { public string Path; public int Left,Top,Width,Height; public string Method="verified CopyFromScreen"; public int SampledPixels,TealSamples,OrangeSamples; }
    private delegate bool EnumProc(IntPtr h,IntPtr p);
    [DllImport("user32.dll")] private static extern bool EnumWindows(EnumProc cb,IntPtr p);
    [DllImport("user32.dll")] private static extern bool EnumChildWindows(IntPtr h,EnumProc cb,IntPtr p);
    [DllImport("user32.dll")] private static extern bool IsWindow(IntPtr h);
    [DllImport("user32.dll")] public static extern bool IsIconic(IntPtr h);
    [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] public static extern bool IsWindowEnabled(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsChild(IntPtr root,IntPtr h);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll")] private static extern int GetDlgCtrlID(IntPtr h);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr h,StringBuilder text,int max);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h,out RECT rect);
    [DllImport("user32.dll")] private static extern IntPtr GetWindow(IntPtr h,uint command);
    [DllImport("user32.dll")] private static extern IntPtr GetAncestor(IntPtr h,uint flags);
    [DllImport("user32.dll")] private static extern IntPtr WindowFromPoint(POINT p);
    [DllImport("user32.dll",EntryPoint="GetWindowLongPtrW")] private static extern IntPtr WindowLong(IntPtr h,int index);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] private static extern IntPtr Send(IntPtr h,uint m,IntPtr w,IntPtr l,uint flags,uint ms,out IntPtr result);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] private static extern IntPtr ReadText(IntPtr h,uint m,IntPtr w,StringBuilder text,uint flags,uint ms,out IntPtr result);
    [DllImport("user32.dll",SetLastError=true)] private static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    [DllImport("user32.dll")] private static extern bool ShowWindowAsync(IntPtr h,int command);
    [DllImport("user32.dll")] private static extern bool SetWindowPos(IntPtr h,IntPtr after,int x,int y,int w,int height,uint flags);
    [DllImport("user32.dll")] private static extern bool SetForegroundWindow(IntPtr h);
    [DllImport("user32.dll")] private static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);

    public static void RequireOwned(IntPtr h,int pid) {
        uint actual;
        if(h==IntPtr.Zero || !IsWindow(h)) throw new Exception("Owned window no longer exists");
        GetWindowThreadProcessId(h,out actual);
        if(actual!=(uint)pid) throw new Exception("Window PID mismatch");
    }
    public static string ClassName(IntPtr h) { var text=new StringBuilder(256); GetClassName(h,text,text.Capacity); return text.ToString(); }
    private static long Message(IntPtr h,uint m,long w,long l) {
        IntPtr result;
        if(Send(h,m,(IntPtr)w,(IntPtr)l,2,1500,out result)==IntPtr.Zero) throw new Exception("Native message deadline: "+m);
        return result.ToInt64();
    }
    public static string Text(IntPtr h,int pid) {
        RequireOwned(h,pid);
        long length=Message(h,0x000E,0,0);
        if(length<0 || length>32768) throw new Exception("Unexpected window text size");
        var text=new StringBuilder((int)length+2); IntPtr result;
        if(ReadText(h,0x000D,(IntPtr)text.Capacity,text,2,1500,out result)==IntPtr.Zero) throw new Exception("Window text deadline");
        return text.ToString();
    }
    public static IntPtr Control(IntPtr root,int pid,int id) {
        RequireOwned(root,pid); var found=new List<IntPtr>();
        EnumChildWindows(root,delegate(IntPtr h,IntPtr p){ if(GetDlgCtrlID(h)==id) found.Add(h); return true;},IntPtr.Zero);
        if(found.Count!=1) throw new Exception("Expected one owned control "+id+", found "+found.Count);
        RequireOwned(found[0],pid);
        if(!IsChild(root,found[0])) throw new Exception("Control escaped owned parent");
        return found[0];
    }
    public static IntPtr ChildByClass(IntPtr root,int pid,string className) {
        RequireOwned(root,pid); var found=new List<IntPtr>();
        EnumChildWindows(root,delegate(IntPtr h,IntPtr p){if(ClassName(h)==className) found.Add(h);return true;},IntPtr.Zero);
        if(found.Count!=1) throw new Exception("Expected one "+className+" preview child");
        RequireOwned(found[0],pid); return found[0];
    }
    public static WindowInfo[] Snapshot(IntPtr root,int pid) {
        RequireOwned(root,pid); var result=new List<WindowInfo>();
        EnumChildWindows(root,delegate(IntPtr h,IntPtr p){RequireOwned(h,pid);result.Add(new WindowInfo{Handle=h.ToInt64(),Class=ClassName(h),Text=Text(h,pid),ID=GetDlgCtrlID(h),Visible=IsWindowVisible(h),Enabled=IsWindowEnabled(h)});return true;},IntPtr.Zero);
        return result.ToArray();
    }
    public static WindowInfo[] Dialogs(IntPtr root,int pid) {
        RequireOwned(root,pid); var found=new List<WindowInfo>();
        EnumWindows(delegate(IntPtr h,IntPtr p){
            uint actual; GetWindowThreadProcessId(h,out actual);
            if(actual==(uint)pid && h!=root && IsWindowVisible(h) && ClassName(h)=="#32770") {
                IntPtr owner=GetWindow(h,4); bool ours=false;
                for(int i=0; owner!=IntPtr.Zero && i<16; i++){if(owner==root){ours=true;break;}owner=GetWindow(owner,4);}
                if(ours) found.Add(new WindowInfo{Handle=h.ToInt64(),Class=ClassName(h),Text=Text(h,pid),Visible=true,Enabled=IsWindowEnabled(h)});
            }
            return true;
        },IntPtr.Zero);
        return found.ToArray();
    }
    public static void ClickAsync(IntPtr root,int pid,int id) {
        IntPtr h=Control(root,pid,id);
        if(!String.Equals(ClassName(h),"Button",StringComparison.OrdinalIgnoreCase) || !IsWindowVisible(h) || !IsWindowEnabled(h)) throw new Exception("Owned button is not visible and enabled: "+id);
        if(!PostMessage(h,0x00F5,IntPtr.Zero,IntPtr.Zero)) throw new Exception("Could not post owned BM_CLICK");
    }
    public static string[] ListItems(IntPtr root,int pid,int id) {
        IntPtr h=Control(root,pid,id);
        if(!String.Equals(ClassName(h),"ListBox",StringComparison.OrdinalIgnoreCase)) throw new Exception("Not a native ListBox");
        long count=Message(h,0x018B,0,0);
        if(count<0 || count>200) throw new Exception("Unexpected fixture ListBox size");
        var items=new string[(int)count];
        for(int i=0;i<count;i++){
            long length=Message(h,0x018A,i,0);
            if(length<0 || length>32768) throw new Exception("Unexpected list text size");
            var b=new StringBuilder((int)length+2);IntPtr r;
            if(ReadText(h,0x0189,(IntPtr)i,b,2,1500,out r)==IntPtr.Zero || r.ToInt64()<0) throw new Exception("List text read failed");
            items[i]=b.ToString();
        }
        return items;
    }
    public static int[] SelectedIndexes(IntPtr root,int pid,int id) {
        IntPtr h=Control(root,pid,id);long count=Message(h,0x0190,0,0);
        if(count<0 || count>200) throw new Exception("Unexpected native selection size");
        if(count==0) return new int[0];
        IntPtr data=Marshal.AllocHGlobal((int)count*4);
        try { long got=Message(h,0x0191,count,data.ToInt64()); if(got!=count) throw new Exception("Selection changed during readback");var result=new int[(int)count];Marshal.Copy(data,result,0,(int)count);return result; }
        finally { Marshal.FreeHGlobal(data); }
    }
    public static int[] SelectFiles(IntPtr root,int pid,int id,string[] names) {
        IntPtr h=Control(root,pid,id);string[] items=ListItems(root,pid,id);
        if((WindowLong(h,-16).ToInt64() & 0x0800)==0) throw new Exception("ListBox does not have LBS_EXTENDEDSEL");
        if(names.Length<1 || names.Length>items.Length) throw new Exception("Invalid explicit file selection");
        var selected=new List<int>();
        foreach(string name in names){int index=Array.IndexOf(items,name);if(index<0 || selected.Contains(index))throw new Exception("Missing/duplicate fixture selection: "+name);selected.Add(index);}
        selected.Sort();
        if(Message(h,0x0185,0,-1)<0)throw new Exception("Clear selection failed");
        foreach(int index in selected){if(Message(h,0x0185,1,index)<0)throw new Exception("Select fixture failed");}
        Message(h,0x019E,selected[0],0);
        // Send the actual LBN_SELCHANGE notification through the app's normal handler.
        Message(root,0x0111,id|(1<<16),h.ToInt64());
        int[] actual=SelectedIndexes(root,pid,id);
        if(actual.Length!=selected.Count)throw new Exception("Selection readback count mismatch");
        for(int i=0;i<actual.Length;i++){if(actual[i]!=selected[i])throw new Exception("Selection readback mismatch");}
        return actual;
    }
    public static string SelectSingleItem(IntPtr root,int pid,int id,int index) {
        IntPtr h=Control(root,pid,id);string[] items=ListItems(root,pid,id);
        if(index<0 || index>=items.Length || (WindowLong(h,-16).ToInt64() & 0x0808)!=0)throw new Exception("Invalid single-selection ListBox target");
        if(Message(h,0x0186,index,0)!=index)throw new Exception("Single-selection readback mismatch");
        Message(root,0x0111,id|(1<<16),h.ToInt64());
        if(Message(h,0x0188,0,0)!=index)throw new Exception("Selection changed after notification");
        return items[index];
    }
    public static void Restore(IntPtr root,int pid,int x,int y) {
        RequireOwned(root,pid);
        if(!ShowWindowAsync(root,9) && !IsWindowVisible(root)) throw new Exception("Could not restore owned window");
        if(!SetWindowPos(root,(IntPtr)(-1),x,y,0,0,0x4041))throw new Exception("Could not place owned window");
        SetForegroundWindow(root);
    }
    public static void RemoveTopmost(IntPtr root,int pid) { RequireOwned(root,pid);SetWindowPos(root,(IntPtr)(-2),0,0,0,0,0x4013); }
    public static void CloseAsync(IntPtr root,int pid) { RequireOwned(root,pid);if(!PostMessage(root,0x0010,IntPtr.Zero,IntPtr.Zero))throw new Exception("Could not post owned WM_CLOSE"); }
    public static CaptureInfo Capture(IntPtr root,int pid,string path,bool previewOnly) {
        RequireOwned(root,pid);
        if(IsIconic(root) || !IsWindowVisible(root))throw new Exception("Refusing minimized/hidden capture");
        IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try {
            RECT r;if(!GetWindowRect(root,out r))throw new Exception("No owned window rectangle");
            int width=r.Right-r.Left,height=r.Bottom-r.Top;
            if(width<300 || height<150 || width>6000 || height>4000)throw new Exception("Invalid owned window rectangle");
            foreach(double fx in new double[]{0.1,0.5,0.9})foreach(double fy in new double[]{0.15,0.5,0.85}){
                int sampleX=r.Left+(int)(width*fx),sampleY=r.Top+(int)(height*fy);
                IntPtr at=WindowFromPoint(new POINT(sampleX,sampleY));
                IntPtr actualRoot=GetAncestor(at,2);
                if(actualRoot!=root){
                    // Diagnostics only: preserve the guard, never read another app's title/text.
                    uint actualPID;GetWindowThreadProcessId(actualRoot,out actualPID);
                    throw new Exception(String.Format("Refusing occluded/offscreen capture: point=({0},{1}); expected_root={2}; actual_root={3}; actual_pid={4}; class={5}; bounds=({6},{7},{8},{9})",sampleX,sampleY,root.ToInt64(),actualRoot.ToInt64(),actualPID,ClassName(actualRoot),r.Left,r.Top,r.Right,r.Bottom));
                }
            }
            if(previewOnly){IntPtr preview=ChildByClass(root,pid,"QuickSortPreview");if(!GetWindowRect(preview,out r))throw new Exception("No preview rectangle");r.Top+=40;r.Left+=3;r.Right-=3;r.Bottom-=3;width=r.Right-r.Left;height=r.Bottom-r.Top;}
            var result=new CaptureInfo{Path=path,Left=r.Left,Top=r.Top,Width=width,Height=height};
            using(var bitmap=new Bitmap(width,height))using(var graphics=Graphics.FromImage(bitmap)){
                graphics.CopyFromScreen(r.Left,r.Top,0,0,bitmap.Size);
                for(int y=0;y<height;y+=3)for(int x=0;x<width;x+=3){Color c=bitmap.GetPixel(x,y);result.SampledPixels++;if(c.G>c.R+40 && c.B>c.R+50 && c.G>80)result.TealSamples++;if(c.R>c.G+70 && c.G>c.B+30 && c.R>150)result.OrangeSamples++;}
                bitmap.Save(path,ImageFormat.Png);
            }
            RequireOwned(root,pid);if(IsIconic(root))throw new Exception("Window minimized during capture");return result;
        } finally {SetThreadDpiAwarenessContext(old);}
    }
}
