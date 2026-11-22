// Test-only native Ctrl/Shift input for QuickSort's owned ListView.
// Unlike posting WM_LBUTTONDOWN with MK_CONTROL/MK_SHIFT, keybd_event sets
// real Windows modifier state that common-controls queries via GetKeyState.
using System;
using System.Runtime.InteropServices;
using System.Threading;
public static class DcmNativeModifierInput {
 [StructLayout(LayoutKind.Sequential)] struct RECT {public int Left,Top,Right,Bottom;}
 [StructLayout(LayoutKind.Sequential)] struct POINT {public int X,Y;}
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr hwnd,out uint pid);
 [DllImport("user32.dll")] static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] static extern bool SetForegroundWindow(IntPtr hwnd);
 [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr hwnd,out RECT rect);
 [DllImport("user32.dll")] static extern bool GetCursorPos(out POINT point);
 [DllImport("user32.dll")] static extern bool SetCursorPos(int x,int y);
 [DllImport("user32.dll")] static extern void keybd_event(byte key,byte scan,uint flags,UIntPtr extra);
 [DllImport("user32.dll")] static extern void mouse_event(uint flags,uint x,uint y,uint data,UIntPtr extra);
 [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr hwnd);
 const uint KEYUP=2, LEFTDOWN=2, LEFTUP=4;
 const byte VK_CONTROL=0x11, VK_SHIFT=0x10;
 public static void ModifierClick(IntPtr root,IntPtr list,int pid,int row,bool shift){
  Click(root,list,pid,row,shift?VK_SHIFT:VK_CONTROL);
 }
 public static void PlainClick(IntPtr root,IntPtr list,int pid,int row){
  Click(root,list,pid,row,0);
 }
 static void Click(IntPtr root,IntPtr list,int pid,int row,byte key){
  uint id;
  if(root==IntPtr.Zero||list==IntPtr.Zero||GetWindowThreadProcessId(root,out id)==0||id!=(uint)pid)
   throw new Exception("QuickSort owner invalid");
  if(GetWindowThreadProcessId(list,out id)==0||id!=(uint)pid||GetParent(list)!=root)
   throw new Exception("ListView not directly owned by test application");
  if(row<0||row>3)throw new Exception("Modifier test row out of range");
  RECT rect;
  if(!GetWindowRect(list,out rect)||rect.Right-rect.Left<240||rect.Bottom-rect.Top<120)
   throw new Exception("Invalid native ListView geometry");
  int x=rect.Left+65, y=rect.Top+45+row*19;
  if(x>=rect.Right-12||y>=rect.Bottom-12)throw new Exception("Modifier target off ListView");
  var before=GetForegroundWindow();
  POINT cursor;
  bool haveCursor=GetCursorPos(out cursor);
  bool held=false;
  try {
   if(!SetForegroundWindow(root)||GetForegroundWindow()!=root)
    throw new Exception("Cannot guarantee foreground belongs to disposable QuickSort test");
   if(!SetCursorPos(x,y))throw new Exception("Cannot position pointer inside native ListView");
   if(key!=0){
    keybd_event(key,0,0,UIntPtr.Zero);
    held=true;
   }
   Thread.Sleep(90);
   if(GetForegroundWindow()!=root)throw new Exception("Foreground changed during modifier test");
   mouse_event(LEFTDOWN,0,0,0,UIntPtr.Zero);
   Thread.Sleep(35);
   mouse_event(LEFTUP,0,0,0,UIntPtr.Zero);
   Thread.Sleep(100);
  } finally {
   if(held)keybd_event(key,0,KEYUP,UIntPtr.Zero);
   if(haveCursor)SetCursorPos(cursor.X,cursor.Y);
   if(before!=IntPtr.Zero&&before!=root)SetForegroundWindow(before);
  }
 }
}
