// Test-only native input limited to a verified Quick Sort preview child.
using System;
using System.Runtime.InteropServices;
public static class DcmPdfWheelInput {
 [StructLayout(LayoutKind.Sequential)] struct RECT {public int L,T,R,B;}
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr window, out uint pid);
 [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr window);
 [DllImport("user32.dll")] static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] static extern bool SetForegroundWindow(IntPtr window);
 [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr window, out RECT rect);
 [DllImport("user32.dll")] static extern bool SetCursorPos(int x,int y);
 [DllImport("user32.dll")] static extern void keybd_event(byte key,byte scan,uint flags,UIntPtr extra);
 [DllImport("user32.dll")] static extern void mouse_event(uint flags,uint dx,uint dy,uint data,UIntPtr extra);
 const byte VK_CONTROL=0x11;
 const uint KEYUP=0x0002, WHEEL=0x0800;
 public static void Wheel(IntPtr root,IntPtr preview,int pid,int relativeX,int relativeY,int detents) {
   uint actual;
   GetWindowThreadProcessId(root,out actual);
   if(root==IntPtr.Zero||actual!=(uint)pid)throw new Exception("Host ownership mismatch");
   GetWindowThreadProcessId(preview,out actual);
   if(preview==IntPtr.Zero||actual!=(uint)pid||GetParent(preview)!=root)
      throw new Exception("Not a directly owned preview");
   if(detents==0||Math.Abs(detents)>4)throw new Exception("Wheel detents outside test bound");
   RECT rect;
   if(!GetWindowRect(preview,out rect)||rect.R-rect.L<150||rect.B-rect.T<150)
      throw new Exception("Preview geometry invalid");
   if(relativeX<20||relativeX>rect.R-rect.L-20||relativeY<45||relativeY>rect.B-rect.T-20)
      throw new Exception("Wheel pointer outside content");
   if(!SetForegroundWindow(root)||GetForegroundWindow()!=root)
      throw new Exception("Cannot guarantee owned foreground");
   if(!SetCursorPos(rect.L+relativeX,rect.T+relativeY))
      throw new Exception("Cannot place pointer on preview");
   keybd_event(VK_CONTROL,0,0,UIntPtr.Zero);
   try {
     System.Threading.Thread.Sleep(90);
     for(int i=0;i<Math.Abs(detents);i++){
       unchecked { mouse_event(WHEEL,0,0,(uint)(detents>0?120:-120),UIntPtr.Zero); }
       System.Threading.Thread.Sleep(35);
     }
   } finally { keybd_event(VK_CONTROL,0,KEYUP,UIntPtr.Zero); }
 }
}
