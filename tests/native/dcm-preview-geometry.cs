// Test-only geometry observation of one verified DCM host and its preview tree.
// No window input, layout, redraw, paint, queue marker or Office action occurs.
// Only this observer thread's DPI context changes, and Observe restores it.
using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Text;

public static class DcmPreviewGeometry {
    private const int MaxPreviewDescendants=100;
    private delegate bool EnumProc(IntPtr h, IntPtr p);

    [StructLayout(LayoutKind.Sequential)]
    public struct Rect { public int Left, Top, Right, Bottom; }

    public sealed class WindowObservation {
        public string Role, Class, UnavailableReason;
        public long Handle, Parent, ObservedAtTicks, WindowDpiContext;
        public int PID, WindowDpiAwareness;
        public uint WindowDpi;
        public bool Available, Visible, WindowRectAvailable, ClientRectAvailable;
        public Rect WindowRect, ClientRect;
    }
    public sealed class ContextObservation {
        public string CallerContextLabel, StartedUtc, FinishedUtc;
        public long RequestedContext, AppliedThreadContext, StartedTicks, FinishedTicks;
        public int AppliedThreadAwareness;
        public bool HostMinimized, HostMaximized;
        public WindowObservation[] Windows;
    }
    public sealed class Observation {
        public int Schema=1, HostPID, PreviewDescendantLimit=MaxPreviewDescendants;
        public long Host, PreviewPane, OriginalThreadContext, HostWindowContext;
        public long StartedTicks, FinishedTicks, StopwatchFrequency=Stopwatch.Frequency;
        public string StartedUtc, FinishedUtc;
        public bool AtomicSnapshot=false;
        public string CoordinateNote="WindowRect is GetWindowRect under each named caller DPI context. ClientRect is the raw GetClientRect reply for that window; foreign-window units are not normalized or inferred. Sequential reads can observe layout changes between contexts.";
        public ContextObservation[] Contexts;
    }

    [DllImport("user32.dll")] private static extern bool IsWindow(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsIconic(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsZoomed(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsChild(IntPtr parent, IntPtr h);
    [DllImport("user32.dll")] private static extern IntPtr GetParent(IntPtr h);
    [DllImport("user32.dll")] private static extern IntPtr GetWindow(IntPtr h, uint command);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr h, StringBuilder name, int max);
    [DllImport("user32.dll")] private static extern bool GetWindowRect(IntPtr h, out Rect rect);
    [DllImport("user32.dll")] private static extern bool GetClientRect(IntPtr h, out Rect rect);
    [DllImport("user32.dll")] private static extern bool EnumChildWindows(IntPtr parent, EnumProc callback, IntPtr p);
    [DllImport("user32.dll")] private static extern uint GetDpiForWindow(IntPtr h);
    [DllImport("user32.dll")] private static extern IntPtr GetWindowDpiAwarenessContext(IntPtr h);
    [DllImport("user32.dll")] private static extern IntPtr GetThreadDpiAwarenessContext();
    [DllImport("user32.dll")] private static extern int GetAwarenessFromDpiAwarenessContext(IntPtr context);
    [DllImport("user32.dll")] private static extern bool AreDpiAwarenessContextsEqual(IntPtr first, IntPtr second);
    [DllImport("user32.dll")] private static extern IntPtr SetThreadDpiAwarenessContext(IntPtr context);

    private static string Class(IntPtr h) {
        var name=new StringBuilder(128);
        return GetClassName(h, name, name.Capacity)>0 ? name.ToString() : "";
    }
    private static void Own(IntPtr h, int pid) {
        uint actual;
        GetWindowThreadProcessId(h, out actual);
        if(h==IntPtr.Zero || !IsWindow(h) || actual!=(uint)pid)
            throw new Exception("Preview geometry host/pane ownership mismatch");
    }
    private static void OwnPane(IntPtr host, IntPtr pane, int pid) {
        Own(host, pid);
        Own(pane, pid);
        if(GetParent(pane)!=host || Class(pane)!="QuickSortPreview")
            throw new Exception("Preview geometry pane identity changed");
    }
    private static IntPtr Pane(IntPtr host, int pid) {
        Own(host, pid);
        IntPtr found=IntPtr.Zero;
        int visited=0;
        // Inspect only direct host children when resolving the pane. The native
        // renderer's larger subtree is enumerated separately under its own bound.
        for(IntPtr h=GetWindow(host, 5); h!=IntPtr.Zero; h=GetWindow(h, 2)) {
            if(++visited>128) throw new Exception("Host direct-child observation bound exceeded");
            if(GetParent(h)==host && Class(h)=="QuickSortPreview") {
                Own(h, pid);
                if(found!=IntPtr.Zero) throw new Exception("Multiple owned preview panes");
                found=h;
            }
        }
        if(found==IntPtr.Zero) throw new Exception("No owned preview pane");
        OwnPane(host, found, pid);
        return found;
    }
    private static WindowObservation ReadWindow(IntPtr h, string role, IntPtr pane) {
        var result=new WindowObservation{Role=role, Handle=h.ToInt64(), ObservedAtTicks=Stopwatch.GetTimestamp(), WindowDpiAwareness=-1};
        uint pid;
        GetWindowThreadProcessId(h, out pid);
        result.PID=(int)pid;
        if(pid==0 || !IsWindow(h) || (role=="preview_descendant" && !IsChild(pane, h))) {
            result.UnavailableReason="window disappeared or left preview subtree before read";
            return result;
        }
        result.Parent=GetParent(h).ToInt64();
        result.Class=Class(h);
        result.Visible=IsWindowVisible(h);
        IntPtr context=GetWindowDpiAwarenessContext(h);
        result.WindowDpiContext=context.ToInt64();
        result.WindowDpiAwareness=GetAwarenessFromDpiAwarenessContext(context);
        result.WindowDpi=GetDpiForWindow(h);
        result.WindowRectAvailable=GetWindowRect(h, out result.WindowRect);
        result.ClientRectAvailable=GetClientRect(h, out result.ClientRect);
        uint after;
        GetWindowThreadProcessId(h, out after);
        result.Available=IsWindow(h) && after==pid && (role!="preview_descendant" || IsChild(pane, h));
        if(!result.Available) result.UnavailableReason="window changed or left preview subtree during read";
        return result;
    }
    private static ContextObservation ReadContext(IntPtr host, IntPtr pane, int pid, string label, IntPtr context) {
        if(context==IntPtr.Zero || SetThreadDpiAwarenessContext(context)==IntPtr.Zero)
            throw new Exception("Could not apply observer DPI context: "+label);
        IntPtr applied=GetThreadDpiAwarenessContext();
        if(!AreDpiAwarenessContextsEqual(applied, context))
            throw new Exception("Observer DPI context did not match request: "+label);
        OwnPane(host, pane, pid);
        var result=new ContextObservation{
            CallerContextLabel=label, RequestedContext=context.ToInt64(), AppliedThreadContext=applied.ToInt64(),
            AppliedThreadAwareness=GetAwarenessFromDpiAwarenessContext(applied),
            StartedUtc=DateTime.UtcNow.ToString("o"), StartedTicks=Stopwatch.GetTimestamp(),
            HostMinimized=IsIconic(host), HostMaximized=IsZoomed(host)
        };
        var descendants=new List<IntPtr>();
        string enumerationError=null;
        EnumChildWindows(pane, delegate(IntPtr h, IntPtr p) {
            if(descendants.Count>=MaxPreviewDescendants) { enumerationError="Preview descendant observation bound exceeded"; return false; }
            if(!IsChild(pane, h)) { enumerationError="Enumerated window escaped preview subtree"; return false; }
            descendants.Add(h);
            return true;
        }, IntPtr.Zero);
        if(enumerationError!=null) throw new Exception(enumerationError);
        var windows=new List<WindowObservation>();
        windows.Add(ReadWindow(host, "host", pane));
        windows.Add(ReadWindow(pane, "owned_preview_pane", pane));
        foreach(IntPtr h in descendants) windows.Add(ReadWindow(h, "preview_descendant", pane));
        OwnPane(host, pane, pid);
        result.Windows=windows.ToArray();
        result.FinishedTicks=Stopwatch.GetTimestamp();
        result.FinishedUtc=DateTime.UtcNow.ToString("o");
        return result;
    }
    public static Observation Observe(IntPtr host, int pid) {
        Own(host, pid);
        IntPtr pane=Pane(host, pid);
        IntPtr original=GetThreadDpiAwarenessContext();
        IntPtr hostContext=GetWindowDpiAwarenessContext(host);
        if(original==IntPtr.Zero || hostContext==IntPtr.Zero)
            throw new Exception("Observer or host DPI context unavailable");
        var result=new Observation{
            Host=host.ToInt64(), HostPID=pid, PreviewPane=pane.ToInt64(),
            OriginalThreadContext=original.ToInt64(), HostWindowContext=hostContext.ToInt64(),
            StartedUtc=DateTime.UtcNow.ToString("o"), StartedTicks=Stopwatch.GetTimestamp()
        };
        try {
            result.Contexts=new ContextObservation[]{
                ReadContext(host, pane, pid, "host_window_dpi_context", hostContext),
                ReadContext(host, pane, pid, "per_monitor_aware_v2_requested_minus4", (IntPtr)(-4))
            };
        } finally {
            if(SetThreadDpiAwarenessContext(original)==IntPtr.Zero ||
                !AreDpiAwarenessContextsEqual(GetThreadDpiAwarenessContext(), original))
                throw new Exception("Could not restore observer thread DPI context");
        }
        OwnPane(host, pane, pid);
        result.FinishedTicks=Stopwatch.GetTimestamp();
        result.FinishedUtc=DateTime.UtcNow.ToString("o");
        return result;
    }
}
