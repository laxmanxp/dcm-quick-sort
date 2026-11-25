//go:build windows

package main

import (
 "crypto/rand"
 "encoding/hex"
 "errors"
 "os"
 "path/filepath"
 "syscall"
 "testing"
 "unsafe"
)

// Tests the actual release install/uninstall functions with RegOverridePredefKey.
// All HKCU access in this Go process is redirected into a random acceptance
// subtree. No actual Explorer/Classes registration is written.
func withOverriddenExplorerHKCU(t *testing.T, run func()) {
 t.Helper()
 nonce:=make([]byte,16)
 if _,err:=rand.Read(nonce);err!=nil{t.Fatal(err)}
 keyPath:=`Software\DCMQuickSortAcceptance\Release-`+hex.EncodeToString(nonce)
 name,err:=syscall.UTF16PtrFromString(keyPath)
 if err!=nil{t.Fatal(err)}
 var handle uintptr
 var disposition uint32
 create:=syscall.NewLazyDLL("advapi32.dll").NewProc("RegCreateKeyExW")
 code,_,_:=create.Call(uintptr(syscall.HKEY_CURRENT_USER),uintptr(unsafe.Pointer(name)),0,0,0,0xf003f,0,uintptr(unsafe.Pointer(&handle)),uintptr(unsafe.Pointer(&disposition)))
 if code!=0{t.Fatalf("create isolated HKCU root: %v",syscall.Errno(code))}
 override:=syscall.NewLazyDLL("advapi32.dll").NewProc("RegOverridePredefKey")
 code,_,_ = override.Call(uintptr(syscall.HKEY_CURRENT_USER),handle)
 if code!=0 {
  syscall.RegCloseKey(syscall.Handle(handle))
  _=sandboxRemoveTree(keyPath)
  t.Fatalf("redirect HKCU refused: %v",syscall.Errno(code))
 }
 defer func(){
  restoreCode,_,_:=override.Call(uintptr(syscall.HKEY_CURRENT_USER),0)
  syscall.RegCloseKey(syscall.Handle(handle))
  if restoreCode!=0{t.Errorf("restore real HKCU failed: %v",syscall.Errno(restoreCode));return}
  if err:=sandboxRemoveTree(keyPath);err!=nil{t.Errorf("remove disposable registry tree: %v",err)}
 }()
 run()
}

func TestReleaseExplorerRealRegistrationRedirectedHKCU(t *testing.T){
 // This test is deliberately not parallel. HKCU overrides are process-wide.
 before,err:=newWindowsExplorerRegistryStore().Snapshot()
 if err!=nil{t.Fatal(err)}
 withOverriddenExplorerHKCU(t,func(){
  dir:=t.TempDir()
  dll,helper:=explorerInstalledPaths(dir)
  if err:=installExplorerSendComponent(dir);err!=nil{t.Fatalf("real Explorer registration failed: %v",err)}
  state,err:=newWindowsExplorerRegistryStore().Snapshot()
  if err!=nil || state.HandlerCLSID!=explorerSendCLSID || state.ServerPath!=dll{
   t.Fatalf("wrong registered values: %+v, err=%v",state,err)
  }
  for _,path:=range []string{dll,helper}{
   info,err:=os.Stat(path)
   if err!=nil || !info.Mode().IsRegular() || info.Size()==0 {t.Fatalf("missing embedded file %s: %v",path,err)}
  }
  model,err:=readExplorerStringValue(explorerSendServerKey,"ThreadingModel")
  if err!=nil || model!=explorerSendThreadingModel {t.Fatalf("ThreadingModel=%q, err=%v",model,err)}
  if err:=installExplorerSendComponent(dir);err!=nil{t.Fatalf("idempotent reinstall failed: %v",err)}
  if err:=uninstallExplorerSendComponent(dir);err!=nil{t.Fatalf("actual uninstall failed: %v",err)}
  state,err=newWindowsExplorerRegistryStore().Snapshot()
  if err!=nil || state!=(explorerRegistrationSnapshot{}) {t.Fatalf("registration remains: %+v, err=%v",state,err)}
  for _,path:=range []string{dll,helper}{
   if _,err:=os.Lstat(path);!errors.Is(err,os.ErrNotExist){t.Fatalf("uninstaller left file %s: %v",path,err)}
  }
 })
 after,err:=newWindowsExplorerRegistryStore().Snapshot()
 if err!=nil || before!=after{t.Fatalf("normal HKCU affected: before=%+v after=%+v err=%v",before,after,err)}
}

func readExplorerStringValue(path, name string)(string,error){
 keyName,err:=syscall.UTF16PtrFromString(path)
 if err!=nil{return "",err}
 valueName,err:=syscall.UTF16PtrFromString(name)
 if err!=nil{return "",err}
 var key syscall.Handle
 if err:=syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER,keyName,0,syscall.KEY_QUERY_VALUE,&key);err!=nil{return "",err}
 defer syscall.RegCloseKey(key)
 var kind,size uint32
 if err:=syscall.RegQueryValueEx(key,valueName,nil,&kind,nil,&size);err!=nil{return "",err}
 if kind!=syscall.REG_SZ || size<2 || size>4096 {return "",errors.New("invalid registry string")}
 data:=make([]byte,size)
 if err:=syscall.RegQueryValueEx(key,valueName,nil,&kind,&data[0],&size);err!=nil{return "",err}
 words:=make([]uint16,int(size)/2)
 for i:=range words{words[i]=uint16(data[2*i])|uint16(data[2*i+1])<<8}
 return syscall.UTF16ToString(words),nil
}

func TestReleaseExplorerRejectsForeignServerRedirectedHKCU(t *testing.T){
 withOverriddenExplorerHKCU(t,func(){
  dir:=t.TempDir()
  if err:=regSetString(explorerSendServerKey,"",filepath.Join(dir,"OtherProduct.dll"));err!=nil{t.Fatal(err)}
  if err:=installExplorerSendComponent(dir);err==nil{t.Fatal("overwrote foreign COM server")}
  state,err:=newWindowsExplorerRegistryStore().Snapshot()
  if err!=nil || state.HandlerCLSID!="" || state.ServerPath!=filepath.Join(dir,"OtherProduct.dll") {
   t.Fatalf("foreign registration changed: %+v err=%v",state,err)
  }
 })
}
