// Test-only input for DCM's own fallback worksheet COMBOBOX. No messages are
// sent to an installed Excel/foreign preview handler or a global Office window.
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;

public static class DcmXlsxInput {
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
        if(h==IntPtr.Zero || actual!=(uint)pid) throw new Exception("XLSX control ownership mismatch");
    }
    private static string Class(IntPtr h) {
        var text=new StringBuilder(128);
        GetClassName(h, text, text.Capacity);
        return text.ToString();
    }
    private static long Message(IntPtr h, uint m, int w) {
        IntPtr result;
        if(Send(h, m, (IntPtr)w, IntPtr.Zero, 2, 1500, out result)==IntPtr.Zero)
            throw new Exception("XLSX combo message deadline");
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
        IntPtr combo=GetDlgItem(panes[0], 1024);
        Own(combo, pid);
        if(GetParent(combo)!=panes[0] || !String.Equals(Class(combo), "ComboBox", StringComparison.OrdinalIgnoreCase)
            || !IsWindowVisible(combo) || !IsWindowEnabled(combo))
            throw new Exception("Owned XLSX sheet selector is not available");
        return combo;
    }
    private static string[] Names(IntPtr combo) {
        long count=Message(combo, 0x0146, 0); // CB_GETCOUNT
        if(count<1 || count>32) throw new Exception("Unexpected fixture worksheet count");
        var names=new string[(int)count];
        for(int i=0; i<count; i++) {
            long length=Message(combo, 0x0149, i); // CB_GETLBTEXTLEN
            if(length<1 || length>256) throw new Exception("Worksheet name outside fixture bounds");
            var name=new StringBuilder((int)length+2);
            IntPtr result;
            if(Read(combo, 0x0148, (IntPtr)i, name, 2, 1500, out result)==IntPtr.Zero || result.ToInt64()!=length)
                throw new Exception("Worksheet name readback failed");
            names[i]=name.ToString();
        }
        return names;
    }
    public static string[] SheetNames(IntPtr host, int pid) {
        return Names(Combo(host, pid));
    }
    public static string SelectedSheet(IntPtr host, int pid) {
        IntPtr combo=Combo(host, pid);
        string[] names=Names(combo);
        long selected=Message(combo, 0x0147, 0); // CB_GETCURSEL
        if(selected<0 || selected>=names.Length) throw new Exception("No valid worksheet selection");
        return names[(int)selected];
    }
    public static void SelectSheet(IntPtr host, int pid, string name) {
        IntPtr combo=Combo(host, pid);
        string[] names=Names(combo);
        int index=Array.IndexOf(names, name);
        if(index<0 || Array.LastIndexOf(names, name)!=index) throw new Exception("Missing or ambiguous worksheet name");
        if(Message(combo, 0x014E, index)!=index || Message(combo, 0x0147, 0)!=index)
            throw new Exception("Worksheet selection readback mismatch");
        IntPtr pane=GetParent(combo);
        Own(pane, pid);
        // The real CBN_SELCHANGE handler lives in previewWindowProc, not the
        // main window. Post its normal notification, then let the harness wait
        // for the selected name AND actual grid content to change.
        if(!PostMessage(pane, 0x0111, (IntPtr)(1024 | (1 << 16)), combo))
            throw new Exception("Could not post owned worksheet notification");
    }
}
