// Test-only input for DCM's fallback selector and the observed PowerPoint
// childClass surface embedded directly inside this exact owned preview pane.
// Native input is limited to posted Page Up/Down; no global input or COM calls.
// Saved preview pixels only corroborate visible ink/image colours; they do not
// establish semantic correctness, which requires the separate image review.
using System;
using System.Collections.Generic;
using System.Drawing;
using System.Drawing.Imaging;
using System.Diagnostics;
using System.IO;
using System.Runtime.InteropServices;
using System.Text;

public static class DcmPptxInput {
    private delegate bool EnumProc(IntPtr h, IntPtr p);
    [DllImport("user32.dll")] private static extern bool EnumChildWindows(IntPtr h, EnumProc callback, IntPtr p);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
    [DllImport("user32.dll")] private static extern IntPtr GetParent(IntPtr h);
    [DllImport("user32.dll")] private static extern IntPtr GetDlgItem(IntPtr h, int id);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsWindowEnabled(IntPtr h);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr h, StringBuilder text, int max);
    [DllImport("user32.dll", EntryPoint="SendMessageTimeoutW")] private static extern IntPtr Send(IntPtr h, uint m, IntPtr w, IntPtr l, uint flags, uint ms, out IntPtr result);
    [DllImport("user32.dll", EntryPoint="SendMessageTimeoutW", CharSet=CharSet.Unicode)] private static extern IntPtr Read(IntPtr h, uint m, IntPtr w, StringBuilder value, uint flags, uint ms, out IntPtr result);
    [DllImport("user32.dll",SetLastError=true)] private static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
    [DllImport("user32.dll")] private static extern bool IsWindow(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsIconic(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsChild(IntPtr parent,IntPtr h);
    [DllImport("user32.dll")] private static extern IntPtr GetAncestor(IntPtr h,uint flags);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetWindowText(IntPtr h,StringBuilder text,int max);
    [DllImport("user32.dll",EntryPoint="GetWindowLongPtrW")] private static extern IntPtr WindowStyle(IntPtr h,int index);
    [DllImport("user32.dll",EntryPoint="MapVirtualKeyW")] private static extern uint MapVirtualKey(uint key,uint type);
    [DllImport("user32.dll")] private static extern bool GetWindowRect(IntPtr h,out Rect r);
    [DllImport("user32.dll")] private static extern IntPtr SetThreadDpiAwarenessContext(IntPtr context);
    [StructLayout(LayoutKind.Sequential)] private struct Rect {public int Left,Top,Right,Bottom;}


    private static void Own(IntPtr h, int pid) {
        uint actual;
        GetWindowThreadProcessId(h, out actual);
        if(h==IntPtr.Zero || !IsWindow(h) || actual!=(uint)pid) throw new Exception("PPTX control ownership mismatch");
    }
    private static string Class(IntPtr h) {
        var text=new StringBuilder(128);
        GetClassName(h, text, text.Capacity);
        return text.ToString();
    }
    private static long Message(IntPtr h, uint m, int w) {
        IntPtr result;
        if(Send(h, m, (IntPtr)w, IntPtr.Zero, 2, 1500, out result)==IntPtr.Zero)
            throw new Exception("PPTX combo message deadline");
        return result.ToInt64();
    }
    private static IntPtr Pane(IntPtr host,int pid) {
        Own(host,pid);
        if(!IsWindowVisible(host) || IsIconic(host) || GetAncestor(host,2)!=host)
            throw new Exception("PPTX host is not a visible top-level test window");
        var panes=new List<IntPtr>();int count=0;bool truncated=false;
        EnumProc callback=delegate(IntPtr h,IntPtr p) {
            if(++count>200){truncated=true;return false;}
            if(GetParent(h)==host && Class(h)=="QuickSortPreview")panes.Add(h);
            return true;
        };
        EnumChildWindows(host,callback,IntPtr.Zero);GC.KeepAlive(callback);
        if(truncated || panes.Count!=1)throw new Exception("Expected one unambiguous direct owned preview pane");
        Own(panes[0],pid);
        if(!IsWindowVisible(panes[0]))throw new Exception("PPTX preview pane is hidden");
        return panes[0];
    }
    private static IntPtr Combo(IntPtr host,int pid) {
        IntPtr pane=Pane(host,pid),combo=GetDlgItem(pane,1025);
        Own(combo,pid);
        if(GetParent(combo)!=pane || !String.Equals(Class(combo),"ComboBox",StringComparison.OrdinalIgnoreCase)
            || !IsWindowVisible(combo) || !IsWindowEnabled(combo))
            throw new Exception("Owned PPTX slide selector is not available");
        return combo;
    }
    private static string[] Names(IntPtr combo) {
        long count=Message(combo, 0x0146, 0); // CB_GETCOUNT
        if(count!=2) throw new Exception("Expected exactly two fixture slides");
        var names=new string[(int)count];
        for(int i=0; i<count; i++) {
            long length=Message(combo, 0x0149, i); // CB_GETLBTEXTLEN
            if(length<1 || length>256) throw new Exception("Slide name outside fixture bounds");
            var name=new StringBuilder((int)length+2);
            IntPtr result;
            if(Read(combo, 0x0148, (IntPtr)i, name, 2, 1500, out result)==IntPtr.Zero || result.ToInt64()!=length)
                throw new Exception("Slide name readback failed");
            names[i]=name.ToString();
        }
        return names;
    }
    public static string[] SlideNames(IntPtr host, int pid) {
        return Names(Combo(host, pid));
    }
    public static string SelectedSlide(IntPtr host, int pid) {
        IntPtr combo=Combo(host, pid);
        string[] names=Names(combo);
        long selected=Message(combo, 0x0147, 0); // CB_GETCURSEL
        if(selected<0 || selected>=names.Length) throw new Exception("No valid slide selection");
        return names[(int)selected];
    }
    public static void SelectSlide(IntPtr host, int pid, string name) {
        IntPtr combo=Combo(host, pid);
        string[] names=Names(combo);
        int index=Array.IndexOf(names, name);
        if(index<0 || Array.LastIndexOf(names, name)!=index) throw new Exception("Missing or ambiguous slide name");
        if(Message(combo, 0x014E, index)!=index || Message(combo, 0x0147, 0)!=index)
            throw new Exception("Slide selection readback mismatch");
        IntPtr pane=GetParent(combo);
        Own(pane, pid);
        // The real CBN_SELCHANGE handler lives in previewWindowProc, not the
        // main window. Post its normal notification, then let the harness wait
        // for the selected name AND actual preview pixels to change.
        if(!PostMessage(pane, 0x0111, (IntPtr)(1025 | (1 << 16)), combo))
            throw new Exception("Could not post owned slide notification");
    }
    public sealed class NativeTarget {
        public long Host,Pane,Hwnd;
        public int HostPID,ProcessId;
        public uint ThreadId;
        public string Class,WindowTitle,ProcessPath,StartedUtc,FixtureName;
    }
    private static IntPtr NativeWindow(IntPtr host,int pid,string filename) {
        if(filename!="07-slides.pptx")throw new Exception("Only the pinned native PPTX fixture is allowed");
        IntPtr pane=Pane(host,pid);var found=new List<IntPtr>();int count=0;bool truncated=false;
        EnumProc callback=delegate(IntPtr h,IntPtr p) {
            if(++count>100){truncated=true;return false;}
            if(IsWindowVisible(h) && String.Equals(Class(h),"childClass",StringComparison.OrdinalIgnoreCase))found.Add(h);
            return true;
        };
        EnumChildWindows(pane,callback,IntPtr.Zero);GC.KeepAlive(callback);
        if(truncated || found.Count!=1)throw new Exception("Expected exactly one visible native PowerPoint childClass in the owned pane");
        IntPtr window=found[0];var title=new StringBuilder(512);GetWindowText(window,title,title.Capacity);
        if(GetParent(window)!=pane || !IsChild(pane,window) || GetAncestor(window,2)!=host || !IsWindowEnabled(window)
            || (WindowStyle(window,-16).ToInt64() & 0x40000000)==0
            || title.ToString()!="PowerPoint Slide Show  -  "+filename)
            throw new Exception("Native PPTX target does not match the observed fixture hierarchy/title");
        return window;
    }
    public static NativeTarget BindNativeSlide(IntPtr host,int pid) {
        IntPtr window=NativeWindow(host,pid,"07-slides.pptx");uint actual;uint thread=GetWindowThreadProcessId(window,out actual);
        if(actual==0 || actual==(uint)pid || thread==0)throw new Exception("Expected the observed out-of-process PowerPoint renderer");
        using(var process=Process.GetProcessById((int)actual)) {
            string path=process.MainModule.FileName;
            if(process.HasExited || !String.Equals(Path.GetFileName(path),"POWERPNT.EXE",StringComparison.OrdinalIgnoreCase))
                throw new Exception("Native slide is not rendered by the observed PowerPoint process");
            var target=new NativeTarget{Host=host.ToInt64(),HostPID=pid,Pane=GetParent(window).ToInt64(),Hwnd=window.ToInt64(),ProcessId=(int)actual,ThreadId=thread,
                Class=Class(window),WindowTitle="PowerPoint Slide Show  -  07-slides.pptx",ProcessPath=path,StartedUtc=process.StartTime.ToUniversalTime().ToString("o"),FixtureName="07-slides.pptx"};
            ValidateNativeSlide(host,pid,target);return target;
        }
    }
    public static void ValidateNativeSlide(IntPtr host,int pid,NativeTarget target) {
        if(target==null || target.Host!=host.ToInt64() || target.HostPID!=pid)throw new Exception("Native PPTX binding belongs to another host");
        IntPtr window=NativeWindow(host,pid,target.FixtureName);uint actual;uint thread=GetWindowThreadProcessId(window,out actual);
        if(window.ToInt64()!=target.Hwnd || GetParent(window).ToInt64()!=target.Pane || actual!=(uint)target.ProcessId || thread!=target.ThreadId)
            throw new Exception("Native PPTX HWND, pane, PID or thread changed");
        using(var process=Process.GetProcessById(target.ProcessId)) {
            if(process.HasExited || !String.Equals(process.MainModule.FileName,target.ProcessPath,StringComparison.OrdinalIgnoreCase)
                || process.StartTime.ToUniversalTime().ToString("o")!=target.StartedUtc)
                throw new Exception("Native PowerPoint process identity changed");
        }
    }
    public static void QueueNativePageKey(IntPtr host,int pid,NativeTarget target,uint key) {
        if(key!=0x21 && key!=0x22)throw new Exception("Only native Page Up and Page Down are allowed");
        uint scan=MapVirtualKey(key,0);if(scan==0)throw new Exception("Native navigation scan code unavailable");
        uint down=1|((scan&0xff)<<16)|0x01000000,up=down|0xc0000000;
        ValidateNativeSlide(host,pid,target);
        if(!PostMessage((IntPtr)target.Hwnd,0x0100,(IntPtr)key,(IntPtr)unchecked((int)down)))throw new Exception("Could not post scoped native Page key down");
        ValidateNativeSlide(host,pid,target);
        if(!PostMessage((IntPtr)target.Hwnd,0x0101,(IntPtr)key,(IntPtr)unchecked((int)up)))throw new Exception("Could not post scoped native Page key up");
        ValidateNativeSlide(host,pid,target);
    }
    public sealed class CropEvidence {public string Path;public int Left,Top,Width,Height;}
    public static CropEvidence CropFromFull(IntPtr host,int pid,NativeTarget target,string fullPath,int captureLeft,int captureTop,string output) {
        IntPtr pane=Pane(host,pid);if(target!=null)ValidateNativeSlide(host,pid,target);
        IntPtr previous=SetThreadDpiAwarenessContext((IntPtr)(-4));
        if(previous==IntPtr.Zero)throw new Exception("Physical screenshot crop context unavailable");
        try {
            Rect root,region;if(!GetWindowRect(host,out root) || !GetWindowRect(target==null?pane:(IntPtr)target.Hwnd,out region))throw new Exception("PPTX crop geometry unavailable");
            if(target==null) {
                Rect combo;if(!GetWindowRect(Combo(host,pid),out combo))throw new Exception("PPTX selector bounds unavailable");
                region.Top=Math.Max(region.Top,combo.Bottom+4);region.Left+=3;region.Right-=3;region.Bottom-=3;
            }
            using(var bitmap=new Bitmap(fullPath)) {
                if(root.Left!=captureLeft || root.Top!=captureTop || root.Right-root.Left!=bitmap.Width || root.Bottom-root.Top!=bitmap.Height)
                    throw new Exception("PPTX host geometry changed after source screenshot");
                if(region.Right-region.Left<100 || region.Bottom-region.Top<80 || region.Left<root.Left || region.Top<root.Top || region.Right>root.Right || region.Bottom>root.Bottom)
                    throw new Exception("PPTX surface escapes the full screenshot");
                var rectangle=new Rectangle(region.Left-captureLeft,region.Top-captureTop,region.Right-region.Left,region.Bottom-region.Top);
                using(var crop=bitmap.Clone(rectangle,PixelFormat.Format32bppArgb))crop.Save(output,ImageFormat.Png);
                if(target!=null)ValidateNativeSlide(host,pid,target);
                return new CropEvidence{Path=output,Left=region.Left,Top=region.Top,Width=rectangle.Width,Height=rectangle.Height};
            }
        } finally {if(SetThreadDpiAwarenessContext(previous)==IntPtr.Zero)throw new Exception("PPTX crop DPI context restoration failed");}
    }
    public sealed class PixelEvidence {
        public int Width, Height, SampledPixels, DarkPixels, LightPixels, TealPixels, OrangePixels;
    }
    public static PixelEvidence InspectPixels(string path) {
        using(var bitmap=new Bitmap(path)) {
            if(bitmap.Width<100 || bitmap.Height<80 || bitmap.Width>6000 || bitmap.Height>4000)
                throw new Exception("PPTX preview capture size outside bounds");
            var evidence=new PixelEvidence{Width=bitmap.Width,Height=bitmap.Height};
            // Skip capture borders. A same-frame physical crop excludes the
            // host-owned fallback selector. The counters are corroborative,
            // not OCR or a native slide index/readability assertion.
            for(int y=12;y<bitmap.Height-12;y+=2)for(int x=12;x<bitmap.Width-12;x+=2) {
                Color c=bitmap.GetPixel(x,y);evidence.SampledPixels++;
                if(c.R<100 && c.G<100 && c.B<100)evidence.DarkPixels++;
                if(c.R>230 && c.G>230 && c.B>230)evidence.LightPixels++;
                if(c.G>c.R+40 && c.B>c.R+40 && c.G>80)evidence.TealPixels++;
                if(c.R>c.G+50 && c.G>c.B+20 && c.R>150)evidence.OrangePixels++;
            }
            return evidence;
        }
    }
    public static int PixelDifference(string beforePath,string afterPath) {
        using(var before=new Bitmap(beforePath))using(var after=new Bitmap(afterPath)) {
            if(before.Size!=after.Size)throw new Exception("PPTX comparison capture geometry changed");
            int changed=0;
            for(int y=12;y<before.Height-12;y+=2)for(int x=12;x<before.Width-12;x+=2) {
                Color a=before.GetPixel(x,y),b=after.GetPixel(x,y);
                if(Math.Abs(a.R-b.R)>8 || Math.Abs(a.G-b.G)>8 || Math.Abs(a.B-b.B)>8)changed++;
            }
            return changed;
        }
    }
}
