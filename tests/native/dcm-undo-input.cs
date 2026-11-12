// Test-only asynchronous native list notification. No foreign viewer messages.
using System;
using System.Runtime.InteropServices;
public static class DcmUndoInput {
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll")] static extern IntPtr GetDlgItem(IntPtr h,int id);
    [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr h);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] static extern IntPtr Send(IntPtr h,uint m,IntPtr w,IntPtr l,uint flags,uint ms,out IntPtr result);
    [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    static long Message(IntPtr h,uint m,int w,int l){IntPtr r;if(Send(h,m,(IntPtr)w,(IntPtr)l,2,1500,out r)==IntPtr.Zero)throw new Exception("Owned list input deadline");return r.ToInt64();}
    public static void Select(IntPtr host,int pid,int index){
        uint actual;GetWindowThreadProcessId(host,out actual);
        if(host==IntPtr.Zero||actual!=(uint)pid)throw new Exception("Host identity mismatch");
        IntPtr list=GetDlgItem(host,1016);GetWindowThreadProcessId(list,out actual);
        if(list==IntPtr.Zero||actual!=(uint)pid||GetParent(list)!=host)throw new Exception("List identity mismatch");
        long count=Message(list,0x018B,0,0);if(index<0||index>=count||count>200)throw new Exception("Fixture selection bounds");
        if(Message(list,0x0185,0,-1)<0||Message(list,0x0185,1,index)<0)throw new Exception("List selection failed");
        Message(list,0x019E,index,0);
        // Do not make a cross-process synchronous send wait for renderer startup.
        // The harness still waits for matching selection, document identity and pixels.
        if(!PostMessage(host,0x0111,(IntPtr)(1016|(1<<16)),list))throw new Exception("Owned selection notification failed");
    }
}
