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
)

func withIsolatedExplorerKeys(t *testing.T, run func(explorerRegistryLocations)){
 t.Helper()
 nonce:=make([]byte,16)
 if _,err:=rand.Read(nonce);err!=nil{t.Fatal(err)}
 base:=`Software\DCMQuickSortAcceptance\Release-`+hex.EncodeToString(nonce)
 keys:=explorerRegistryLocations{handler:base+`\Handler`,server:base+`\CLSID\InprocServer32`}
 defer func(){if err:=sandboxRemoveTree(base);err!=nil{t.Errorf("clean acceptance keys: %v",err)}}()
 run(keys)
}

func TestReleaseExplorerRealRegistrationIsolatedKeys(t *testing.T){
 before,err:=newWindowsExplorerRegistryStore().Snapshot()
 if err!=nil{t.Fatal(err)}
 withIsolatedExplorerKeys(t,func(keys explorerRegistryLocations){
  dir:=t.TempDir()
  dll,helper:=explorerInstalledPaths(dir)
  if err:=installExplorerSendComponentAt(dir,keys);err!=nil{t.Fatalf("real registry install failed: %v",err)}
  state,err:=snapshotExplorerLocations(keys)
  if err!=nil || state.HandlerCLSID!=explorerSendCLSID || state.ServerPath!=dll{
   t.Fatalf("wrong registered values: %+v err=%v",state,err)
  }
  for _,path:=range []string{dll,helper}{
   info,err:=os.Stat(path)
   if err!=nil || !info.Mode().IsRegular() || info.Size()==0 {t.Fatalf("missing embedded payload %s: %v",path,err)}
  }
  threading,err:=readExplorerStringValue(keys.server,"ThreadingModel")
  if err!=nil || threading!=explorerSendThreadingModel{t.Fatalf("ThreadingModel=%q err=%v",threading,err)}
  if err:=installExplorerSendComponentAt(dir,keys);err!=nil{t.Fatalf("idempotent install failed: %v",err)}
  if err:=uninstallExplorerSendComponentAt(dir,keys);err!=nil{t.Fatalf("real registry uninstall failed: %v",err)}
  state,err=snapshotExplorerLocations(keys)
  if err!=nil || state!=(explorerRegistrationSnapshot{}) {t.Fatalf("registered keys remain: %+v err=%v",state,err)}
  for _,path:=range []string{dll,helper}{
   if _,err:=os.Lstat(path);!errors.Is(err,os.ErrNotExist){t.Fatalf("installed component still exists %s: %v",path,err)}
  }
 })
 after,err:=newWindowsExplorerRegistryStore().Snapshot()
 if err!=nil || before!=after{t.Fatalf("live Explorer changed: before=%+v after=%+v err=%v",before,after,err)}
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

func TestReleaseExplorerRejectsForeignServerIsolatedKeys(t *testing.T){
 withIsolatedExplorerKeys(t,func(keys explorerRegistryLocations){
  dir:=t.TempDir()
  foreign:=filepath.Join(dir,"OtherProduct.dll")
  if err:=regSetString(keys.server,"",foreign);err!=nil{t.Fatal(err)}
  if err:=installExplorerSendComponentAt(dir,keys);err==nil{t.Fatal("accepted foreign COM server")}
  state,err:=snapshotExplorerLocations(keys)
  if err!=nil || state.HandlerCLSID!="" || state.ServerPath!=foreign{t.Fatalf("foreign registration modified: %+v err=%v",state,err)}
 })
}

func TestReleaseExplorerPreflightRejectsForeignWithoutFiles(t *testing.T){
 withIsolatedExplorerKeys(t,func(keys explorerRegistryLocations){
  root:=t.TempDir()
  foreign:=filepath.Join(root,"foreign.dll")
  if err:=regSetString(keys.server,"",foreign);err!=nil{t.Fatal(err)}
  if err:=preflightExplorerSendInstallAt(root,keys);err==nil{t.Fatal("accepted foreign COM owner")}
  dll,helper:=explorerInstalledPaths(root)
  for _,path:=range []string{dll,helper}{
   if _,err:=os.Lstat(path);!errors.Is(err,os.ErrNotExist){t.Fatalf("modified files during preflight: %s",path)}
  }
 })
}
func TestReleaseExplorerPreflightRejectsModifiedPayload(t *testing.T){
 withIsolatedExplorerKeys(t,func(keys explorerRegistryLocations){
  root:=t.TempDir()
  dll,helper:=explorerInstalledPaths(root)
  if err:=os.WriteFile(dll,[]byte("foreign DLL"),0600);err!=nil{t.Fatal(err)}
  if err:=preflightExplorerSendInstallAt(root,keys);err==nil{t.Fatal("accepted foreign DLL bytes")}
  if err:=os.Remove(dll);err!=nil{t.Fatal(err)}
  if err:=os.WriteFile(helper,[]byte("foreign helper"),0600);err!=nil{t.Fatal(err)}
  if err:=preflightExplorerSendInstallAt(root,keys);err==nil{t.Fatal("accepted foreign helper bytes")}
  if err:=os.Remove(helper);err!=nil{t.Fatal(err)}
  if err:=preflightExplorerSendInstallAt(root,keys);err!=nil{t.Fatalf("fresh install preflight rejected: %v",err)}
 })
}
func TestReleaseExplorerPreflightAcceptsOwnedInstalledPayload(t *testing.T){
 withIsolatedExplorerKeys(t,func(keys explorerRegistryLocations){
  root:=t.TempDir()
  if err:=installExplorerSendComponentAt(root,keys);err!=nil{t.Fatal(err)}
  if err:=preflightExplorerSendInstallAt(root,keys);err!=nil{t.Fatalf("existing owned payload rejected: %v",err)}
  if err:=uninstallExplorerSendComponentAt(root,keys);err!=nil{t.Fatal(err)}
 })
}
