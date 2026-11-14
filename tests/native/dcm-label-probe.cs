// Owned direct-child layout audit. No foreign viewer memory, OCR or repaint.
using System;
using System.Collections.Generic;
using System.Drawing;
using System.Runtime.InteropServices;
using System.Text;
public static class DcmLabelProbe {
    delegate bool EnumProc(IntPtr h, IntPtr p);
    [DllImport("user32.dll")] static extern bool EnumChildWindows(IntPtr h, EnumProc f, IntPtr p);
    [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr h);
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint p);
    [DllImport("user32.dll")] static extern int GetDlgCtrlID(IntPtr h);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h,StringBuilder b,int n);
    [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h,out RECT r);
    [DllImport("user32.dll")] static extern bool GetClientRect(IntPtr h,out RECT r);
    [DllImport("user32.dll")] static extern bool ClientToScreen(IntPtr h,ref POINT p);
    [DllImport("user32.dll")] static extern IntPtr GetWindowDpiAwarenessContext(IntPtr h);
    [DllImport("user32.dll")] static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] static extern IntPtr Send(IntPtr h,uint m,IntPtr w,IntPtr l,uint flags,uint ms,out IntPtr r);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] static extern IntPtr Read(IntPtr h,uint m,IntPtr w,StringBuilder b,uint flags,uint ms,out IntPtr r);
    [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    [StructLayout(LayoutKind.Sequential)] public struct RECT {public int L,T,R,B;}
    [StructLayout(LayoutKind.Sequential)] struct POINT {public int X,Y;}
    public sealed class Item {public int ID; public long Window;public string Class,Text;public RECT Rect;public int TextWidth;}
    public sealed class Audit {public bool Passed;public int ClientWidth,ClientHeight;public Item[] Items;public string[] Errors;}
    static void Own(IntPtr h,int pid){uint p;GetWindowThreadProcessId(h,out p);if(h==IntPtr.Zero||p!=(uint)pid)throw new Exception("Label audit PID mismatch");}
    static string Text(IntPtr h){var b=new StringBuilder(512);IntPtr r;if(Read(h,13,(IntPtr)b.Capacity,b,2,1500,out r)==IntPtr.Zero)throw new Exception("Label text deadline");return b.ToString();}
    static string Class(IntPtr h){var b=new StringBuilder(128);GetClassName(h,b,b.Capacity);return b.ToString();}
    static bool Intersects(RECT a,RECT b){return a.L<b.R&&b.L<a.R&&a.T<b.B&&b.T<a.B;}
    // Matches the product's fixed Segoe UI 16px normal/semibold label contract.
    // A separate screenshot review remains necessary for actual rendered wording.
    static int Width(string text,bool bold){using(var bmp=new Bitmap(1,1))using(var g=Graphics.FromImage(bmp))using(var f=new Font("Segoe UI",16, bold?FontStyle.Bold:FontStyle.Regular,GraphicsUnit.Pixel)){return (int)Math.Ceiling(g.MeasureString(text,f,int.MaxValue,StringFormat.GenericTypographic).Width)+2;}}
    static Item ByID(List<Item> items,int id){return items.Find(delegate(Item i){return i.ID==id;});}
    static Item ByText(List<Item> items,string text){return items.Find(delegate(Item i){return i.Class=="Static"&&(text=="PROCESSED"?i.Text.StartsWith(text):i.Text==text);});}
    public static Audit Check(IntPtr host,int pid){
        Own(host,pid);IntPtr old=SetThreadDpiAwarenessContext(GetWindowDpiAwarenessContext(host));
        if(old==IntPtr.Zero)throw new Exception("Cannot align layout-audit DPI context");
        try{
            RECT client;if(!GetClientRect(host,out client))throw new Exception("Host client unavailable");var origin=new POINT();if(!ClientToScreen(host,ref origin))throw new Exception("Host coordinates unavailable");
            var items=new List<Item>();
            EnumChildWindows(host,delegate(IntPtr h,IntPtr ignored){if(GetParent(h)!=host||!IsWindowVisible(h))return true;Own(h,pid);RECT r;if(!GetWindowRect(h,out r))throw new Exception("Control geometry unavailable");r.L-=origin.X;r.R-=origin.X;r.T-=origin.Y;r.B-=origin.Y;items.Add(new Item{ID=GetDlgCtrlID(h),Window=h.ToInt64(),Class=Class(h),Text=Text(h),Rect=r});return true;},IntPtr.Zero);
            var errors=new List<string>();var inspected=new List<Item>();var filter=new List<Item>();
            string[] labels={"File Type:","Extension:","Age:","Size:","Sort By:","New Name:","Favorite:","Vault Location:","PROCESSED"};
            int[] pairs={1012,1013,1031,1032,1045,1005,1018,1003,0};
            for(int n=0;n<labels.Length;n++){
                Item label=ByText(items,labels[n]);if(label==null){errors.Add("Missing label "+labels[n]);continue;}
                label.TextWidth=Width(label.Text,n==8);inspected.Add(label);
                if(label.Rect.R-label.Rect.L<label.TextWidth)errors.Add("Clipped label "+label.Text+": width="+(label.Rect.R-label.Rect.L)+" needs="+label.TextWidth);
                if(label.Rect.L<0||label.Rect.T<0||label.Rect.R>client.R||label.Rect.B>client.B)errors.Add("Label outside client: "+label.Text);
                if(n<5)filter.Add(label);
                if(pairs[n]!=0){Item field=ByID(items,pairs[n]);if(field==null){errors.Add("Missing field "+pairs[n]);continue;}inspected.Add(field);if(n<5)filter.Add(field);
                    if(label.Rect.R>field.Rect.L||label.Rect.T>=field.Rect.B||label.Rect.B<=field.Rect.T)errors.Add("Label/field overlap or wrong row: "+label.Text);
                    if(n<5){field.TextWidth=Width(field.Text,false)+24;if(field.Rect.R-field.Rect.L<field.TextWidth)errors.Add("Clipped selected value "+field.Text);}
                }
            }
            Item clear=ByID(items,1033);if(clear==null)errors.Add("Clear Filters missing");else {filter.Add(clear);inspected.Add(clear);}
            for(int n=0;n<filter.Count;n++){Item a=filter[n];if(a.Rect.R>client.R||a.Rect.L<0)errors.Add("Filter outside client: "+a.Text);for(int j=n+1;j<filter.Count;j++)if(Intersects(a.Rect,filter[j].Rect))errors.Add("Filter overlap: "+a.Text+" / "+filter[j].Text);}
            var header=new List<Item>();Item title=ByText(items,"PROCESSED");if(title!=null)header.Add(title);
            foreach(int id in new int[]{1048,1008,1046,1017}){Item a=ByID(items,id);if(a==null)errors.Add("Missing processed control "+id);else {header.Add(a);inspected.Add(a);}}
            for(int i=0;i<header.Count;i++)for(int j=i+1;j<header.Count;j++)if(Intersects(header[i].Rect,header[j].Rect))errors.Add("Processed header overlap: "+header[i].Text+" / "+header[j].Text);
            Item panel=items.Find(delegate(Item a){return a.Class=="QuickSortPreview";});if(panel==null)errors.Add("Preview missing");else foreach(Item a in filter)if(a.Rect.B>panel.Rect.T)errors.Add("Filter extends into content: "+a.Text);
            return new Audit{Passed=errors.Count==0,ClientWidth=client.R,ClientHeight=client.B,Items=inspected.ToArray(),Errors=errors.ToArray()};
        }finally{SetThreadDpiAwarenessContext(old);}
    }
    public static void LongSort(IntPtr host,int pid){
        Own(host,pid);IntPtr combo=IntPtr.Zero;EnumChildWindows(host,delegate(IntPtr h,IntPtr p){if(GetParent(h)==host&&GetDlgCtrlID(h)==1045){Own(h,pid);combo=h;}return true;},IntPtr.Zero);
        IntPtr r;if(combo==IntPtr.Zero||Send(combo,0x014E,(IntPtr)3,IntPtr.Zero,2,1500,out r)==IntPtr.Zero||r.ToInt64()!=3)throw new Exception("Sort selection failed");
        if(!PostMessage(host,0x111,(IntPtr)(1045|(1<<16)),combo))throw new Exception("Sort notification failed");
    }
}
