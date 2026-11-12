// Test-only input/readback for the exact owned embedded Sumatra canvas.
// Does not set scroll position directly, read process memory, or relax captures.
using System;
using System.Drawing;
using System.Runtime.InteropServices;
using System.Text;
public static class DcmScrollProbe {
    delegate bool EnumProc(IntPtr h,IntPtr p);
    [DllImport("user32.dll")] static extern bool EnumChildWindows(IntPtr h,EnumProc f,IntPtr p);
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint p);
    [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr h);
    [DllImport("user32.dll")] static extern IntPtr GetAncestor(IntPtr h,uint flag);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h,StringBuilder s,int n);
    [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll",EntryPoint="GetWindowLongPtrW")] static extern IntPtr Style(IntPtr h,int n);
    [DllImport("user32.dll")] static extern bool GetClientRect(IntPtr h,out RECT r);
    [DllImport("user32.dll")] static extern bool ClientToScreen(IntPtr h,ref POINT p);
    [DllImport("user32.dll")] static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
    [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    [DllImport("user32.dll")] static extern bool GetScrollInfo(IntPtr h,int bar,ref SCROLLINFO s);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] static extern IntPtr Read(IntPtr h,uint m,IntPtr w,StringBuilder b,uint flags,uint ms,out IntPtr r);
    [StructLayout(LayoutKind.Sequential)] struct RECT {public int L,T,R,B;}
    [StructLayout(LayoutKind.Sequential)] struct POINT {public int X,Y;}
    [StructLayout(LayoutKind.Sequential)] struct SCROLLINFO {public uint cbSize,fMask;public int nMin,nMax;public uint nPage;public int nPos,nTrackPos;}
    static string Class(IntPtr h){var b=new StringBuilder(100);GetClassName(h,b,b.Capacity);return b.ToString();}
    static void Own(IntPtr h,int pid){uint p;GetWindowThreadProcessId(h,out p);if(h==IntPtr.Zero||p!=(uint)pid)throw new Exception("Scroll target PID mismatch");}
    static IntPtr Canvas(IntPtr host,int hostPID,int viewerPID){
        Own(host,hostPID);IntPtr result=IntPtr.Zero;
        EnumChildWindows(host,delegate(IntPtr h,IntPtr unused){
            uint p;GetWindowThreadProcessId(h,out p);
            if(p!=(uint)viewerPID||Class(h)!="SUMATRA_PDF_CANVAS"||!IsWindowVisible(h))return true;
            IntPtr frame=GetParent(h),panel=GetParent(frame);
            Own(frame,viewerPID);Own(panel,hostPID);
            if(Class(frame)!="SUMATRA_PDF_FRAME"||Class(panel)!="QuickSortPreview"||GetAncestor(panel,2)!=host)throw new Exception("Scroll target escaped embedding");
            if(result!=IntPtr.Zero)throw new Exception("Ambiguous canvas");result=h;return true;
        },IntPtr.Zero);
        if(result==IntPtr.Zero)throw new Exception("No owned visible PDF canvas");return result;
    }
    public sealed class ScrollState {public long Canvas;public int Minimum,Maximum,Position,Limit,PageNumber,HorizontalPosition,HorizontalMinimum,HorizontalLimit;public uint Viewport;}
    public static ScrollState State(IntPtr host,int hostPID,int viewerPID){
        IntPtr canvas=Canvas(host,hostPID,viewerPID),frame=GetParent(canvas);var si=new SCROLLINFO{cbSize=(uint)Marshal.SizeOf(typeof(SCROLLINFO)),fMask=0x17};
        if(!GetScrollInfo(canvas,1,ref si))throw new Exception("Cannot read vertical scrollbar");
        int page=0;
        EnumChildWindows(frame,delegate(IntPtr h,IntPtr unused){uint p;GetWindowThreadProcessId(h,out p);
            if(p!=(uint)viewerPID||Class(h)!="Edit"||Class(GetParent(h))!="ToolbarWindow32"||(Style(h,-16).ToInt64()&0x2000)==0)return true;
            var b=new StringBuilder(32);IntPtr r;if(Read(h,13,(IntPtr)b.Capacity,b,2,300,out r)==IntPtr.Zero||!int.TryParse(b.ToString(),out page))throw new Exception("Page number unavailable");return true;
        },IntPtr.Zero);
        var hi=new SCROLLINFO{cbSize=(uint)Marshal.SizeOf(typeof(SCROLLINFO)),fMask=0x17};GetScrollInfo(canvas,0,ref hi);
        return new ScrollState{Canvas=canvas.ToInt64(),Minimum=si.nMin,Maximum=si.nMax,Position=si.nPos,Viewport=si.nPage,Limit=si.nMax-Math.Max((int)si.nPage-1,0),PageNumber=page,HorizontalPosition=hi.nPos,HorizontalMinimum=hi.nMin,HorizontalLimit=Math.Max(hi.nMin,hi.nMax-Math.Max((int)hi.nPage-1,0))};
    }
    public static void Scroll(IntPtr host,int hostPID,int viewerPID,int command){
        if(command!=0&&command!=1&&command!=2&&command!=3&&command!=6&&command!=7)throw new Exception("Scroll command outside allowlist");
        if(!PostMessage(Canvas(host,hostPID,viewerPID),0x115,(IntPtr)command,IntPtr.Zero))throw new Exception("Could not post owned scroll request");
    }
    public static void ScrollLeft(IntPtr host,int hostPID,int viewerPID){
        if(!PostMessage(Canvas(host,hostPID,viewerPID),0x114,(IntPtr)6,IntPtr.Zero))throw new Exception("Could not post owned horizontal scroll request");
    }
    public sealed class InkEvidence {public int MarkerPixels,MarkerX,MarkerY,MarkerWidth,MarkerHeight,CanvasWidth,CanvasHeight,TextInkPixels,TextInkRows;public bool MarkerFullyVisible;}
    // Region colours locate a known fixture text block; ink corroborates rendered
    // glyphs, not semantic text recognition. Exact wording is separately reviewed.
    public static InkEvidence Inspect(IntPtr host,int hostPID,int viewerPID,string capture,int left,int top,int red,int green,int blue){
        IntPtr canvas=Canvas(host,hostPID,viewerPID);IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try{
            RECT r;if(!GetClientRect(canvas,out r))throw new Exception("Canvas rectangle unavailable");var origin=new POINT();if(!ClientToScreen(canvas,ref origin))throw new Exception("Canvas origin unavailable");
            var e=new InkEvidence{CanvasWidth=r.R,CanvasHeight=r.B};int minX=r.R,minY=r.B,maxX=-1,maxY=-1;
            using(var image=new Bitmap(capture)){
                int ox=origin.X-left,oy=origin.Y-top;
                if(ox<0||oy<0||ox+r.R>image.Width||oy+r.B>image.Height)throw new Exception("Canvas escaped source screenshot");
                for(int y=2;y<r.B-2;y+=2)for(int x=2;x<r.R-2;x+=2){var c=image.GetPixel(ox+x,oy+y);if(Math.Abs(c.R-red)<=8&&Math.Abs(c.G-green)<=8&&Math.Abs(c.B-blue)<=8){e.MarkerPixels++;minX=Math.Min(minX,x);maxX=Math.Max(maxX,x);minY=Math.Min(minY,y);maxY=Math.Max(maxY,y);}}
                if(maxX<minX)return e;
                e.MarkerX=minX;e.MarkerY=minY;e.MarkerWidth=maxX-minX+2;e.MarkerHeight=maxY-minY+2;
                double ratio=(double)e.MarkerHeight/e.MarkerWidth;e.MarkerFullyVisible=minY>=10&&maxY<r.B-10&&ratio>2.9&&ratio<3.8;
                int x1=maxX+Math.Max(3,e.MarkerWidth/3),x2=Math.Min(r.R-3,minX+33*e.MarkerWidth),y1=Math.Max(1,minY-3),y2=Math.Min(r.B-2,maxY+3);
                for(int y=y1;y<y2;y++){int row=0;for(int x=x1;x<x2;x++){var c=image.GetPixel(ox+x,oy+y);if(c.R<100&&c.G<100&&c.B<100)row++;}e.TextInkPixels+=row;if(row>=5)e.TextInkRows++;}
            }
            return e;
        }finally{SetThreadDpiAwarenessContext(old);}
    }
}
