// Test-only real input for an identified disposable ListBox. Never sets selection
// with LB_SETSEL: Ctrl/Shift must travel through Windows' input and notification path.
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
using System.Threading;

public static class DcmSelectionInput {
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left,Top,Right,Bottom; }
    [StructLayout(LayoutKind.Sequential)] public struct POINT { public int X,Y; }
    [StructLayout(LayoutKind.Sequential)] public struct MOUSEINPUT { public int X,Y; public uint Data,Flags,Time; public UIntPtr Extra; }
    [StructLayout(LayoutKind.Sequential)] public struct KEYBDINPUT { public ushort Key,Scan; public uint Flags,Time; public UIntPtr Extra; }
    [StructLayout(LayoutKind.Explicit,Size=40)] public struct INPUT {
        [FieldOffset(0)] public uint Type;
        [FieldOffset(8)] public MOUSEINPUT Mouse;
        [FieldOffset(8)] public KEYBDINPUT Keyboard;
    }
    public sealed class Receipt { public int Index,X,Y,Modifier; public string Name,Method="SendInput modifier+mouse, owned foreground and physical hit-test"; }
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll")] private static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")] private static extern IntPtr GetAncestor(IntPtr h,uint flags);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] private static extern bool IsWindowEnabled(IntPtr h);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr h,StringBuilder text,int size);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] private static extern IntPtr Send(IntPtr h,uint m,IntPtr w,IntPtr l,uint flags,uint ms,out IntPtr result);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] private static extern IntPtr GetRect(IntPtr h,uint m,IntPtr w,ref RECT rect,uint flags,uint ms,out IntPtr result);
    [DllImport("user32.dll")] private static extern bool ClientToScreen(IntPtr h,ref POINT p);
    [DllImport("user32.dll")] private static extern bool GetClientRect(IntPtr h,out RECT r);
    [DllImport("user32.dll")] private static extern bool LogicalToPhysicalPointForPerMonitorDPI(IntPtr h,ref POINT p);
    [DllImport("user32.dll")] private static extern IntPtr GetWindowDpiAwarenessContext(IntPtr h);
    [DllImport("user32.dll")] private static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
    [DllImport("user32.dll")] private static extern IntPtr WindowFromPoint(POINT p);
    [DllImport("user32.dll")] private static extern bool SetCursorPos(int x,int y);
    [DllImport("user32.dll")] private static extern short GetAsyncKeyState(int key);
    [DllImport("user32.dll",SetLastError=true)] private static extern uint SendInput(uint count,INPUT[] inputs,int size);
    private static void Require(IntPtr root,IntPtr list,int pid){
        uint actual; GetWindowThreadProcessId(root,out actual);
        if(actual!=(uint)pid || !IsWindowVisible(root) || !IsWindowEnabled(root) || GetForegroundWindow()!=root)throw new Exception("Input refused: owned foreground window unavailable");
        GetWindowThreadProcessId(list,out actual);
        var cls=new StringBuilder(128);GetClassName(list,cls,cls.Capacity);
        if(actual!=(uint)pid || cls.ToString()!="ListBox" || GetAncestor(list,2)!=root || !IsWindowVisible(list) || !IsWindowEnabled(list))throw new Exception("Input refused: target is not the owned enabled ListBox");
        foreach(int key in new int[]{1,2,4,5,6,16,17,18,91,92})if((GetAsyncKeyState(key)&0x8000)!=0)throw new Exception("User key/button held; no input sent");
    }
    private static INPUT Key(int key,bool up){var x=new INPUT();x.Type=1;x.Keyboard.Key=(ushort)key;x.Keyboard.Flags=up?2u:0u;return x;}
    private static INPUT Mouse(bool up){var x=new INPUT();x.Mouse.Flags=up?4u:2u;return x;}
    private static void Deliver(INPUT[] input){
        uint sent=SendInput((uint)input.Length,input,Marshal.SizeOf(typeof(INPUT)));
        if(sent!=(uint)input.Length){
            // Release only down-events actually inserted by this failed sequence.
            var held=new HashSet<int>();bool mouse=false;
            for(int i=0;i<sent;i++){var x=input[i];if(x.Type==1){if((x.Keyboard.Flags&2)!=0)held.Remove(x.Keyboard.Key);else held.Add(x.Keyboard.Key);}else{if((x.Mouse.Flags&2)!=0)mouse=true;if((x.Mouse.Flags&4)!=0)mouse=false;}}
            var releases=new List<INPUT>();if(mouse)releases.Add(Mouse(true));foreach(int key in held)releases.Add(Key(key,true));
            if(releases.Count>0)SendInput((uint)releases.Count,releases.ToArray(),Marshal.SizeOf(typeof(INPUT)));
            throw new Exception("SendInput did not deliver the entire owned sequence: "+sent+"/"+input.Length);
        }
        Thread.Sleep(180);
        foreach(int key in new int[]{1,16,17})if((GetAsyncKeyState(key)&0x8000)!=0)throw new Exception("Input sequence did not finish with modifiers/buttons released");
    }
    public static Receipt ClickRow(IntPtr root,IntPtr list,int pid,int index,string name,int modifier){
        if(IntPtr.Size!=8 || Marshal.SizeOf(typeof(INPUT))!=40)throw new Exception("Native x64 input layout required");
        if(modifier!=0 && modifier!=16 && modifier!=17)throw new Exception("Unsupported selection modifier");
        Require(root,list,pid);IntPtr old=SetThreadDpiAwarenessContext(GetWindowDpiAwarenessContext(list));
        try{
            IntPtr result;var rect=new RECT();RECT client;
            if(GetRect(list,0x0198,(IntPtr)index,ref rect,2,1500,out result)==IntPtr.Zero || result.ToInt64()<0 || !GetClientRect(list,out client))throw new Exception("Cannot identify row geometry");
            var p=new POINT{X=rect.Left+Math.Min(50,Math.Max(5,(rect.Right-rect.Left)/2)),Y=(rect.Top+rect.Bottom)/2};
            if(p.X<client.Left || p.X>=client.Right || p.Y<client.Top || p.Y>=client.Bottom)throw new Exception("Target row is outside the visible client area");
            long packed=((long)(ushort)p.Y<<16)|(ushort)p.X;
            if(Send(list,0x01A9,IntPtr.Zero,(IntPtr)packed,2,1500,out result)==IntPtr.Zero || (result.ToInt64()&0xFFFF)!=index || ((result.ToInt64()>>16)&0xFFFF)!=0)throw new Exception("Independent ListBox hit-test did not identify target row");
            if(!ClientToScreen(list,ref p) || !LogicalToPhysicalPointForPerMonitorDPI(list,ref p))throw new Exception("Cannot convert row coordinates");
            SetThreadDpiAwarenessContext((IntPtr)(-4));
            Require(root,list,pid);
            if(WindowFromPoint(p)!=list)throw new Exception("Physical row point is obscured or belongs to another control");
            if(!SetCursorPos(p.X,p.Y) || WindowFromPoint(p)!=list)throw new Exception("Cannot safely position pointer");
            var input=new List<INPUT>();if(modifier!=0)input.Add(Key(modifier,false));input.Add(Mouse(false));input.Add(Mouse(true));if(modifier!=0)input.Add(Key(modifier,true));
            Deliver(input.ToArray());
            return new Receipt{Index=index,Name=name,Modifier=modifier,X=p.X,Y=p.Y};
        }finally{SetThreadDpiAwarenessContext(old);}
    }
}
