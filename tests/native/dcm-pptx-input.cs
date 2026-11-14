// Test-only input for DCM's own fallback slide COMBOBOX. No messages are
// sent to an installed PowerPoint/foreign preview handler or a global Office window.
// Saved preview pixels only corroborate visible ink/image colours; they do not
// establish semantic correctness, which requires the separate image review.
using System;
using System.Collections.Generic;
using System.Drawing;
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
    [DllImport("user32.dll")] private static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);

    private static void Own(IntPtr h, int pid) {
        uint actual;
        GetWindowThreadProcessId(h, out actual);
        if(h==IntPtr.Zero || actual!=(uint)pid) throw new Exception("PPTX control ownership mismatch");
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
    private static IntPtr Combo(IntPtr host, int pid) {
        Own(host, pid);
        var panes=new List<IntPtr>();
        EnumChildWindows(host, delegate(IntPtr h, IntPtr p) {
            if(GetParent(h)==host && Class(h)=="QuickSortPreview") panes.Add(h);
            return true;
        }, IntPtr.Zero);
        if(panes.Count!=1) throw new Exception("Expected one direct owned preview pane");
        Own(panes[0], pid);
        IntPtr combo=GetDlgItem(panes[0], 1025);
        Own(combo, pid);
        if(GetParent(combo)!=panes[0] || !String.Equals(Class(combo), "ComboBox", StringComparison.OrdinalIgnoreCase)
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
    public sealed class PixelEvidence {
        public int Width, Height, SampledPixels, DarkPixels, LightPixels, TealPixels, OrangePixels;
    }
    public static PixelEvidence InspectPixels(string path) {
        using(var bitmap=new Bitmap(path)) {
            if(bitmap.Width<100 || bitmap.Height<80 || bitmap.Width>6000 || bitmap.Height>4000)
                throw new Exception("PPTX preview capture size outside bounds");
            var evidence=new PixelEvidence{Width=bitmap.Width,Height=bitmap.Height};
            // Skip capture borders. Capture's preview-only mode already excludes
            // the top host-owned fallback combo. The counters are corroborative,
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
