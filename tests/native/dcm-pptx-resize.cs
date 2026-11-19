// Strict read-only geometry and host-window maximize controls for the
// existing two-slide native PPTX acceptance harness. Never resize PowerPoint
// childClass, reparent foreign HWNDs, or send global keyboard input.
using System;
using System.Runtime.InteropServices;

public static class DcmPptxResize {
    [StructLayout(LayoutKind.Sequential)]
    public struct Rect {
        public int Left, Top, Right, Bottom;
        public int Width { get { return Right - Left; } }
        public int Height { get { return Bottom - Top; } }
    }
    [DllImport("user32.dll",EntryPoint="GetWindowLongPtrW")]
    public static extern IntPtr GetWindowLongPtr(IntPtr h,int index);
    [DllImport("user32.dll")]
    public static extern bool IsZoomed(IntPtr h);
    [DllImport("user32.dll")]
    public static extern bool ShowWindow(IntPtr h,int command);
    [DllImport("user32.dll")]
    private static extern bool GetWindowRect(IntPtr h,out Rect rect);
    [DllImport("user32.dll")]
    private static extern IntPtr SetThreadDpiAwarenessContext(IntPtr context);

    // GetWindowRect on a DPI-unaware caller returns virtualized coordinates.
    // Inspect owned-pane and observed PowerPoint child in one physical context.
    // This is a measurement only; no foreign window is manipulated.
    public static Rect PhysicalRect(IntPtr h) {
        if(h==IntPtr.Zero)throw new Exception("Missing window to measure");
        IntPtr original=SetThreadDpiAwarenessContext((IntPtr)(-4));
        if(original==IntPtr.Zero)throw new Exception("Physical DPI context unavailable");
        try {
            Rect rect;
            if(!GetWindowRect(h,out rect) || rect.Width<=0 || rect.Height<=0)
                throw new Exception("Window physical bounds unavailable");
            return rect;
        } finally {
            if(SetThreadDpiAwarenessContext(original)==IntPtr.Zero)
                throw new Exception("Could not restore caller DPI context");
        }
    }
}
