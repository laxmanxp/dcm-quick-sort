// DCM test-only popup inspection/input. No product hooks or host associations.
using System;
using System.Collections.Generic;
using System.Drawing;
using System.Drawing.Imaging;
using System.Runtime.InteropServices;
using System.Text;

public static class DcmMenuActions {
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left,Top,Right,Bottom; }
    [StructLayout(LayoutKind.Sequential)] public struct POINT { public int X,Y; public POINT(int x,int y){X=x;Y=y;} }
    [StructLayout(LayoutKind.Sequential)] public struct MENUBARINFO { public uint Size; public RECT Rect; public IntPtr Menu,Window; public uint Flags; }
    [StructLayout(LayoutKind.Sequential)] public struct MOUSEINPUT { public int X,Y; public uint Data,Flags,Time; public UIntPtr Extra; }
    [StructLayout(LayoutKind.Explicit,Size=40)] public struct INPUT { [FieldOffset(0)] public uint Type; [FieldOffset(8)] public MOUSEINPUT Mouse; }
    public sealed class Item { public int Position; public uint ID,State; public string Text; public bool Submenu; }
    public sealed class Shot { public string Path; public int Width,Height; }
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr h,StringBuilder text,int size);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] private static extern bool GetWindowRect(IntPtr h,out RECT rect);
    [DllImport("user32.dll")] private static extern bool GetMenuBarInfo(IntPtr h,int obj,int item,ref MENUBARINFO info);
    [DllImport("user32.dll")] private static extern int GetMenuItemCount(IntPtr menu);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetMenuString(IntPtr menu,uint position,StringBuilder text,int max,uint flags);
    [DllImport("user32.dll")] private static extern uint GetMenuItemID(IntPtr menu,int position);
    [DllImport("user32.dll")] private static extern uint GetMenuState(IntPtr menu,uint position,uint flags);
    [DllImport("user32.dll")] private static extern IntPtr GetSubMenu(IntPtr menu,int position);
    [DllImport("user32.dll")] private static extern bool GetMenuItemRect(IntPtr window,IntPtr menu,uint position,out RECT rect);
    [DllImport("user32.dll")] private static extern IntPtr WindowFromPoint(POINT p);
    [DllImport("user32.dll")] private static extern IntPtr GetAncestor(IntPtr h,uint flags);
    [DllImport("user32.dll")] private static extern bool SetCursorPos(int x,int y);
    [DllImport("user32.dll")] private static extern uint SendInput(uint count,INPUT[] inputs,int size);
    [DllImport("user32.dll")] private static extern short GetAsyncKeyState(int key);
    [DllImport("user32.dll")] private static extern IntPtr SetThreadDpiAwarenessContext(IntPtr value);
    private static IntPtr Require(IntPtr popup,int pid){
        uint actual;GetWindowThreadProcessId(popup,out actual);
        var text=new StringBuilder(128);GetClassName(popup,text,text.Capacity);
        if(actual!=(uint)pid || text.ToString()!="#32768" || !IsWindowVisible(popup))throw new Exception("Not the visible owned popup");
        var info=new MENUBARINFO();info.Size=(uint)Marshal.SizeOf(info);
        if(!GetMenuBarInfo(popup,-4,0,ref info)||info.Menu==IntPtr.Zero)throw new Exception("Cannot identify popup HMENU");
        return info.Menu;
    }
    public static Item[] Items(IntPtr popup,int pid){
        IntPtr menu=Require(popup,pid);int count=GetMenuItemCount(menu);
        if(count<1||count>100)throw new Exception("Unexpected popup item count");
        var result=new List<Item>();
        for(int i=0;i<count;i++){var text=new StringBuilder(1024);GetMenuString(menu,(uint)i,text,text.Capacity,0x400);result.Add(new Item{Position=i,ID=GetMenuItemID(menu,i),State=GetMenuState(menu,(uint)i,0x400),Text=text.ToString(),Submenu=GetSubMenu(menu,i)!=IntPtr.Zero});}
        return result.ToArray();
    }
    public static Shot Capture(IntPtr popup,int pid,string path){
        Require(popup,pid);IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try{
            RECT rect;if(!GetWindowRect(popup,out rect))throw new Exception("Popup rectangle unavailable");
            int width=rect.Right-rect.Left,height=rect.Bottom-rect.Top;
            if(width<40||height<20||width>2000||height>2000)throw new Exception("Invalid popup dimensions");
            foreach(double x in new double[]{0.15,0.5,0.85})foreach(double y in new double[]{0.1,0.5,0.9}){
                IntPtr at=WindowFromPoint(new POINT(rect.Left+(int)(width*x),rect.Top+(int)(height*y)));
                if(GetAncestor(at,2)!=popup)throw new Exception("Popup obscured or offscreen; refusing capture");
            }
            using(var bitmap=new Bitmap(width,height))using(var graphics=Graphics.FromImage(bitmap)){graphics.CopyFromScreen(rect.Left,rect.Top,0,0,bitmap.Size);bitmap.Save(path,ImageFormat.Png);}
            Require(popup,pid);return new Shot{Path=path,Width=width,Height=height};
        }finally{SetThreadDpiAwarenessContext(old);}
    }
    public static Item ClickOpen(IntPtr popup,int pid){
        IntPtr menu=Require(popup,pid);Item choice=null;
        foreach(var item in Items(popup,pid)){
            string label=item.Text.Replace("&","").Split('\t')[0].Trim();
            if(label=="Open"){if(choice!=null)throw new Exception("Ambiguous Open item");choice=item;}
        }
        if(choice==null || choice.Submenu || (choice.State & 3)!=0 || choice.ID==0xFFFFFFFF)throw new Exception("Exact enabled Open command unavailable; no command invoked");
        IntPtr old=SetThreadDpiAwarenessContext((IntPtr)(-4));
        try{
            RECT rect;if(!GetMenuItemRect(IntPtr.Zero,menu,(uint)choice.Position,out rect))throw new Exception("Cannot read Open geometry");
            int x=rect.Left+(rect.Right-rect.Left)/2,y=rect.Top+(rect.Bottom-rect.Top)/2;
            foreach(int key in new int[]{1,2,4,16,17,18,91,92})if((GetAsyncKeyState(key)&0x8000)!=0)throw new Exception("User key/button held; no input sent");
            if(GetAncestor(WindowFromPoint(new POINT(x,y)),2)!=popup){RECT window;GetWindowRect(popup,out window);IntPtr at=WindowFromPoint(new POINT(x,y));var cls=new StringBuilder(128);GetClassName(at,cls,cls.Capacity);throw new Exception(String.Format("Open target is obscured: item=[{0},{1},{2},{3}], popup=[{4},{5},{6},{7}], point=({8},{9}), hitClass={10}, popup={11}, hitRoot={12}",rect.Left,rect.Top,rect.Right,rect.Bottom,window.Left,window.Top,window.Right,window.Bottom,x,y,cls,popup,GetAncestor(at,2)));}
            Require(popup,pid);
            if(!SetCursorPos(x,y))throw new Exception("Cannot position pointer on owned Open item");
            if(GetAncestor(WindowFromPoint(new POINT(x,y)),2)!=popup)throw new Exception("Open target ownership changed");
            var input=new INPUT[2];input[0].Mouse.Flags=2;input[1].Mouse.Flags=4;
            if(SendInput(2,input,Marshal.SizeOf(typeof(INPUT)))!=2)throw new Exception("Open input was not fully delivered");
            return choice;
        }finally{SetThreadDpiAwarenessContext(old);}
    }
}
