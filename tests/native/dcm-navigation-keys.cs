using System;
using System.Runtime.InteropServices;

// Owned queued navigation messages. No global SendInput or direct selection change.
public static class DcmNavigationKeys {
    [StructLayout(LayoutKind.Sequential)] private struct RECT {public int Left,Top,Right,Bottom;}
    [StructLayout(LayoutKind.Sequential)] private struct GUIINFO {
        public int cbSize; public uint flags;
        public IntPtr active,focus,capture,menuOwner,moveSize,caret; public RECT caretRect;
    }
    [DllImport("user32.dll",SetLastError=true)] private static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll",SetLastError=true)] private static extern bool GetGUIThreadInfo(uint tid,ref GUIINFO info);
    [DllImport("user32.dll")] private static extern short GetAsyncKeyState(int key);
    [DllImport("user32.dll")] private static extern uint MapVirtualKey(uint key,uint map);
    private static void IdleKeys() {
        foreach(int key in new int[]{1,2,4,16,17,18,91,92,35,36,38,40})
            if((GetAsyncKeyState(key)&0x8000)!=0)throw new Exception("User key/button held; queued navigation refused.");
    }
    public static bool ListFocused(IntPtr root,int pid) {
        IntPtr list=DcmNative.Control(root,pid,1016);uint actual;
        uint tid=GetWindowThreadProcessId(root,out actual);
        if(actual!=(uint)pid)throw new Exception("Navigation PID changed.");
        var info=new GUIINFO();info.cbSize=Marshal.SizeOf(typeof(GUIINFO));
        if(!GetGUIThreadInfo(tid,ref info))throw new Exception("Cannot verify list focus.");
        return info.focus==list;
    }
    private static void Queue(IntPtr h,int key,bool system) {
        long bits=1L|((long)(MapVirtualKey((uint)key,0)&255)<<16);
        if(key==35||key==36||key==38||key==40)bits|=1L<<24;
        if(system)bits|=1L<<29;
        if(!PostMessage(h,system?0x0104u:0x0100u,(IntPtr)key,(IntPtr)bits))throw new Exception("Key-down queue failed.");
        if(!PostMessage(h,system?0x0105u:0x0101u,(IntPtr)key,(IntPtr)(bits|0xC0000000L)))throw new Exception("Key-up queue failed.");
    }
    public static void FocusList(IntPtr root,int pid) {
        DcmNative.RequireOwned(root,pid);IdleKeys();
        if(!ListFocused(root,pid))Queue(root,0x4C,true); // Existing Alt+L app mnemonic.
    }
    public static void Press(IntPtr root,int pid,int key) {
        if(key!=35&&key!=36&&key!=38&&key!=40)throw new Exception("Unsupported navigation key.");
        IdleKeys();if(!ListFocused(root,pid))throw new Exception("File list is not focused.");
        var list=DcmNative.Control(root,pid,1016);
        if(!DcmNative.IsWindowEnabled(list)||!DcmNative.IsWindowVisible(list))throw new Exception("File list not usable.");
        Queue(list,key,false);
    }
}
