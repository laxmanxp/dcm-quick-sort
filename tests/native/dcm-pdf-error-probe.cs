// Test-only inspection/input for the exact private viewer PID. Password values are never logged.
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
public static class DcmPdfErrorProbe {
    delegate bool EnumProc(IntPtr h,IntPtr p);
    [DllImport("user32.dll")] static extern bool EnumWindows(EnumProc p,IntPtr data);
    [DllImport("user32.dll")] static extern bool EnumChildWindows(IntPtr root,EnumProc p,IntPtr data);
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint p);
    [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] static extern bool IsWindowEnabled(IntPtr h);
    [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr h);
    [DllImport("user32.dll")] static extern int GetDlgCtrlID(IntPtr h);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h,StringBuilder b,int n);
    [DllImport("user32.dll",EntryPoint="GetWindowLongPtrW")] static extern IntPtr GetStyle(IntPtr h,int index);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] static extern IntPtr Read(IntPtr h,uint m,IntPtr w,StringBuilder b,uint f,uint ms,out IntPtr r);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW",CharSet=CharSet.Unicode)] static extern IntPtr SetText(IntPtr h,uint m,IntPtr w,string text,uint f,uint ms,out IntPtr r);
    [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    public sealed class Item {public long Window,Parent;public int ID;public string Class,Text;public bool Visible,Enabled,Password;}
    static string Class(IntPtr h){var b=new StringBuilder(128);GetClassName(h,b,b.Capacity);return b.ToString();}
    static bool Own(IntPtr h,int pid){uint actual;GetWindowThreadProcessId(h,out actual);return h!=IntPtr.Zero&&actual==(uint)pid;}
    static Item Describe(IntPtr h,int pid){
        if(!Own(h,pid))throw new Exception("Foreign error-dialog control");
        string c=Class(h);bool password=c=="Edit"&&(GetStyle(h,-16).ToInt64()&0x20)!=0;string text="";
        if(!password){IntPtr result;var b=new StringBuilder(1024);if(Read(h,13,(IntPtr)b.Capacity,b,2,100,out result)!=IntPtr.Zero)text=b.ToString();else text="<text timeout>";}
        return new Item{Window=h.ToInt64(),Parent=GetParent(h).ToInt64(),ID=GetDlgCtrlID(h),Class=c,Text=text,Visible=IsWindowVisible(h),Enabled=IsWindowEnabled(h),Password=password};
    }
    public static Item[] Windows(int pid){
        if(pid<=0)throw new Exception("Invalid owned PID");var roots=new List<IntPtr>();
        EnumWindows(delegate(IntPtr h,IntPtr data){if(Own(h,pid))roots.Add(h);return true;},IntPtr.Zero);
        var items=new List<Item>();foreach(var root in roots){items.Add(Describe(root,pid));EnumChildWindows(root,delegate(IntPtr h,IntPtr data){if(Own(h,pid))items.Add(Describe(h,pid));return true;},IntPtr.Zero);}return items.ToArray();
    }
    public static long PasswordDialog(int pid){
        var items=Windows(pid);long found=0;
        foreach(var root in items){if(root.Class!="#32770"||!root.Visible)continue;foreach(var c in items){if(c.Parent==root.Window&&c.Visible&&c.Password){if(found!=0&&found!=root.Window)throw new Exception("Ambiguous password prompt");found=root.Window;}}}return found;
    }
    public static void Password(long dialog,int pid,string value,bool cancel){
        IntPtr root=(IntPtr)dialog;if(!Own(root,pid)||Class(root)!="#32770"||!IsWindowVisible(root))throw new Exception("Owned password prompt gone");
        IntPtr edit=IntPtr.Zero,button=IntPtr.Zero;int count=0;
        foreach(var i in Windows(pid)){if(i.Parent!=dialog||!i.Visible||!i.Enabled)continue;if(i.Password){edit=(IntPtr)i.Window;count++;}if(i.Class=="Button"&&i.ID==(cancel?2:1))button=(IntPtr)i.Window;}
        if(count!=1||button==IntPtr.Zero)throw new Exception("Password control contract differs");
        if(!cancel){IntPtr result;if(value==null||value.Length>128||SetText(edit,12,IntPtr.Zero,value,2,1000,out result)==IntPtr.Zero)throw new Exception("Password entry failed");}
        if(!PostMessage(button,0xF5,IntPtr.Zero,IntPtr.Zero))throw new Exception("Password submit/cancel failed");
    }
}
