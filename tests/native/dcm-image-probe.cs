// Test-only deterministic fixtures and pixel oracle. Never reads application memory.
using System;
using System.Collections.Generic;
using System.Drawing;
using System.Drawing.Imaging;
using System.IO;
using System.Runtime.InteropServices;
public static class DcmImageProbe {
    [StructLayout(LayoutKind.Sequential)] struct RECT { public int L,T,R,B; }
    [StructLayout(LayoutKind.Sequential)] struct POINT { public int X,Y; }
    public sealed class View { public int Width,Height,Left,Top,PhysicalWidth,PhysicalHeight; }
    public sealed class Result { public int Samples,ImageSamples,OutsideSamples; public int DrawWidth,DrawHeight,DrawX,DrawY; public View Viewport; }
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll")] static extern bool GetClientRect(IntPtr h,out RECT r);
    [DllImport("user32.dll")] static extern bool ClientToScreen(IntPtr h,ref POINT p);
    [DllImport("user32.dll")] static extern bool LogicalToPhysicalPointForPerMonitorDPI(IntPtr h,ref POINT p);
    [DllImport("user32.dll")] static extern IntPtr GetWindowDpiAwarenessContext(IntPtr h);
    [DllImport("user32.dll")] static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] static extern IntPtr Send(IntPtr h,uint m,IntPtr w,IntPtr l,uint flags,uint ms,out IntPtr r);
    static readonly Color[] Palette={Color.FromArgb(20,160,190),Color.FromArgb(230,140,30),Color.FromArgb(140,65,195),Color.FromArgb(50,170,70)};
    static void Own(IntPtr h,int pid){uint p;GetWindowThreadProcessId(h,out p);if(h==IntPtr.Zero || p!=(uint)pid)throw new Exception("Pixel probe ownership mismatch");}
    public static void Create(string path,int w,int h,int variant,bool transparent){
        if(w<100||h<100||w>1600||h>1600||variant<0||variant>3)throw new Exception("Fixture bounds");
        using(var b=new Bitmap(w,h,PixelFormat.Format32bppArgb))using(var g=Graphics.FromImage(b)){
            g.Clear(Color.Transparent);
            for(int q=0;q<4;q++){
                if(transparent && q==3)continue;
                using(var brush=new SolidBrush(Palette[(q+variant)%4]))g.FillRectangle(brush,(q%2)*w/2,(q/2)*h/2,w/2,h/2);
            }
            using(var font=new Font("Arial",12))using(var brush=new SolidBrush(Color.Black))
                g.DrawString("DCM-27 "+Path.GetExtension(path)+" "+w+"x"+h,font,brush,8,h/2-12);
            string ext=Path.GetExtension(path).ToLowerInvariant();
            ImageFormat f=ext==".bmp"?ImageFormat.Bmp:ext==".gif"?ImageFormat.Gif:(ext==".jpg"||ext==".jpeg")?ImageFormat.Jpeg:(ext==".tif"||ext==".tiff")?ImageFormat.Tiff:ImageFormat.Png;
            b.Save(path,f);
        }
        using(var check=Image.FromFile(path)){if(check.Width!=w||check.Height!=h)throw new Exception("Encoded fixture dimensions changed");}
    }
    public static View Geometry(IntPtr preview,int pid){
        Own(preview,pid);IntPtr old=SetThreadDpiAwarenessContext(GetWindowDpiAwarenessContext(preview));
        try{
            RECT r;if(!GetClientRect(preview,out r))throw new Exception("No preview client rect");
            var a=new POINT{X=0,Y=0};var b=new POINT{X=r.R,Y=r.B};
            if(!ClientToScreen(preview,ref a)||!ClientToScreen(preview,ref b)||!LogicalToPhysicalPointForPerMonitorDPI(preview,ref a)||!LogicalToPhysicalPointForPerMonitorDPI(preview,ref b))throw new Exception("DPI conversion failed");
            return new View{Width=r.R,Height=r.B,Left=a.X,Top=a.Y,PhysicalWidth=b.X-a.X,PhysicalHeight=b.Y-a.Y};
        }finally{SetThreadDpiAwarenessContext(old);}
    }
    static bool Near(Color a,Color b){return Math.Abs(a.R-b.R)<=35 && Math.Abs(a.G-b.G)<=35 && Math.Abs(a.B-b.B)<=35;}
    public static Result Verify(string capture,int captureLeft,int captureTop,View v,int w,int h,int variant,bool transparent,string mode,int percent,int panX,int panY){
        int aw=v.Width-8,ah=v.Height-8,dw,dh;
        if(mode=="fit"){double s=Math.Min((double)aw/w,(double)ah/h);dw=(int)(w*s);dh=(int)(h*s);}
        else if(mode=="width"){dw=aw;dh=(int)Math.Floor((double)h*aw/w+0.5);panY=Math.Max(0,(dh-ah)/2);}
        else {dw=(w*percent+50)/100;dh=(h*percent+50)/100;}
        panX=Math.Max(-Math.Max(0,(dw-aw)/2),Math.Min(Math.Max(0,(dw-aw)/2),panX));
        panY=Math.Max(-Math.Max(0,(dh-ah)/2),Math.Min(Math.Max(0,(dh-ah)/2),panY));
        int dx=(v.Width-dw)/2+panX,dy=(v.Height-dh)/2+panY;
        var result=new Result{DrawWidth=dw,DrawHeight=dh,DrawX=dx,DrawY=dy,Viewport=v};
        using(var image=new Bitmap(capture)){
            double[] positions={.035,.15,.28,.38,.62,.72,.85,.965};
            foreach(double fy in positions)foreach(double fx in positions){
                double lx=fx*v.Width,ly=fy*v.Height,sx=(lx-dx)/dw,sy=(ly-dy)/dh;
                bool inside=sx>=0&&sx<1&&sy>=0&&sy<1;
                // Avoid antialiased edges, the center label and quadrant seams.
                if(inside && (sy>.43 && sy<.57 || Math.Abs(sx-.5)<.025 || sx<.015 || sx>.985 || sy<.015 || sy>.985))continue;
                if(!inside && (Math.Abs(lx-dx)<5 || Math.Abs(lx-dx-dw)<5 || Math.Abs(ly-dy)<5 || Math.Abs(ly-dy-dh)<5))continue;
                int q=(sx>=.5?1:0)+(sy>=.5?2:0);Color expected=inside && !(transparent&&q==3)?Palette[(q+variant)%4]:Color.White;
                int x=v.Left-captureLeft+(int)(fx*v.PhysicalWidth),y=v.Top-captureTop+(int)(fy*v.PhysicalHeight);
                if(x<0||y<0||x>=image.Width||y>=image.Height)throw new Exception("Probe point escaped capture");
                Color actual=image.GetPixel(x,y);
                if(!Near(actual,expected))throw new Exception("DCM27_PIXEL_MISMATCH mode="+mode+" logical="+lx+","+ly+" expected="+expected+" actual="+actual+" draw="+dx+","+dy+","+dw+","+dh);
                result.Samples++;if(inside)result.ImageSamples++;else result.OutsideSamples++;
            }
        }
        if(result.ImageSamples<8 || result.Samples<20)throw new Exception("Insufficient independent pixel samples");
        return result;
    }
    // Exercises the owned preview's existing drag handler; not physical input certification.
    public static void Drag(IntPtr preview,int pid,int dx,int dy){
        Own(preview,pid);RECT r;if(!GetClientRect(preview,out r))throw new Exception("Drag geometry");
        int x=r.R/2,y=r.B/2;if(x+dx<5||x+dx>r.R-5||y+dy<5||y+dy>r.B-5)throw new Exception("Drag escaped preview");
        IntPtr res;
        if(Send(preview,0x0201,(IntPtr)1,(IntPtr)((y<<16)|x),2,1500,out res)==IntPtr.Zero)throw new Exception("Drag start timeout");
        try{if(Send(preview,0x0200,(IntPtr)1,(IntPtr)(((y+dy)<<16)|(x+dx)),2,1500,out res)==IntPtr.Zero)throw new Exception("Drag move timeout");}
        finally{Send(preview,0x0202,IntPtr.Zero,(IntPtr)(((y+dy)<<16)|(x+dx)),2,1500,out res);}
    }
}
