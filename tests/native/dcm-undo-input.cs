// Test-only asynchronous native list notification. No foreign viewer messages.
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class DcmUndoInput {
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll")] static extern IntPtr GetDlgItem(IntPtr h,int id);
    [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr h);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] static extern IntPtr Send(IntPtr h,uint m,IntPtr w,IntPtr l,uint flags,uint ms,out IntPtr result);
    [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    static long Message(IntPtr h,uint m,int w,int l){IntPtr r;if(Send(h,m,(IntPtr)w,(IntPtr)l,2,1500,out r)==IntPtr.Zero)throw new Exception("Owned list input deadline");return r.ToInt64();}
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] static extern IntPtr Read(IntPtr h,uint m,IntPtr w,StringBuilder value,uint flags,uint ms,out IntPtr result);
    public static string TryText(IntPtr host,int pid,int id){
        if(id!=1005)throw new Exception("Only selected filename readiness may be polled");
        uint actual;GetWindowThreadProcessId(host,out actual);
        if(host==IntPtr.Zero||actual!=(uint)pid)throw new Exception("Host identity mismatch");
        IntPtr edit=GetDlgItem(host,id);GetWindowThreadProcessId(edit,out actual);
        if(edit==IntPtr.Zero||actual!=(uint)pid||GetParent(edit)!=host)throw new Exception("Filename control identity mismatch");
        IntPtr n,r;
        if(Send(edit,14,IntPtr.Zero,IntPtr.Zero,2,50,out n)==IntPtr.Zero)return null;
        if(n.ToInt64()<0||n.ToInt64()>32768)throw new Exception("Filename length outside bounds");
        var text=new StringBuilder((int)n+2);
        if(Read(edit,13,(IntPtr)text.Capacity,text,2,50,out r)==IntPtr.Zero)return null;
        return text.ToString();
    }
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
