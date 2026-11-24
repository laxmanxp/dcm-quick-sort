//go:build windows

package main

import (
 "errors"
 "fmt"
 "strings"
 "syscall"
 "testing"
 "unsafe"
)

// This integration test uses its own random HKCU\\Software\\DCMQuickSortAcceptance
// subtree, NEVER Software\\Classes, Explorer's handler, or a production CLSID.
var sandboxAdvapi = syscall.NewLazyDLL("advapi32.dll")
var sandboxCreateKey = sandboxAdvapi.NewProc("RegCreateKeyExW")
var sandboxSetValue = sandboxAdvapi.NewProc("RegSetValueExW")
var sandboxDeleteTree = sandboxAdvapi.NewProc("RegDeleteTreeW")
const sandboxRegSZ = uintptr(1)
const sandboxKeyWrite = uintptr(0x20006)

type sandboxExplorerRegistry struct {
 handlerKey string
 serverKey string
 writes int
 failAfterWrite int
 failBeforeWrite int
}

func newSandboxExplorerRegistry(t *testing.T) *sandboxExplorerRegistry {
 t.Helper()
 // Create a unique run scope. t.TempDir has a random leaf not controlled by
 // external callers; it is used only for a unique registry suffix.
 suffix:=strings.ReplaceAll(strings.ReplaceAll(t.TempDir(),"\\","_"),":","_")
 parts:=strings.Split(suffix,"_")
 id:=parts[len(parts)-1]
 if len(id)<5 {t.Fatal("invalid unique test registry id")}
 base:=`Software\DCMQuickSortAcceptance\`+id
 store:=&sandboxExplorerRegistry{handlerKey:base+`\Handler`,serverKey:base+`\Server`}
 t.Cleanup(func(){
  // Delete ONLY this test's own random acceptance subtree.
  if err:=sandboxRemoveTree(base);err!=nil{t.Errorf("cleanup isolated registry tree: %v",err)}
 })
 return store
}
func sandboxWriteString(key,value string) error {
 path,err:=syscall.UTF16PtrFromString(key)
 if err!=nil{return err}
 var handle uintptr
 var disposition uint32
 rv,_,_:=sandboxCreateKey.Call(uintptr(syscall.HKEY_CURRENT_USER),uintptr(unsafe.Pointer(path)),0,0,0,sandboxKeyWrite,0,uintptr(unsafe.Pointer(&handle)),uintptr(unsafe.Pointer(&disposition)))
 if rv!=0{return syscall.Errno(rv)}
 defer syscall.RegCloseKey(syscall.Handle(handle))
 chars,err:=syscall.UTF16FromString(value)
 if err!=nil{return err}
 rv,_,_=sandboxSetValue.Call(handle,0,0,sandboxRegSZ,uintptr(unsafe.Pointer(&chars[0])),uintptr(len(chars)*2))
 if rv!=0{return syscall.Errno(rv)}
 return nil
}
func sandboxRemoveTree(key string)error{
 path,err:=syscall.UTF16PtrFromString(key)
 if err!=nil{return err}
 rv,_,_:=sandboxDeleteTree.Call(uintptr(syscall.HKEY_CURRENT_USER),uintptr(unsafe.Pointer(path)))
 if rv==0 || rv==uintptr(syscall.ERROR_FILE_NOT_FOUND) || rv==uintptr(syscall.ERROR_PATH_NOT_FOUND){return nil}
 return syscall.Errno(rv)
}
func (s *sandboxExplorerRegistry) Snapshot()(explorerRegistrationSnapshot,error){
 handler,err:=readExplorerDefaultValue(s.handlerKey)
 if err!=nil{return explorerRegistrationSnapshot{},err}
 server,err:=readExplorerDefaultValue(s.serverKey)
 if err!=nil{return explorerRegistrationSnapshot{},err}
 return explorerRegistrationSnapshot{HandlerCLSID:handler,ServerPath:server},nil
}
// This is an intentionally non-atomic check-then-write, for a temporary test
// subtree only. The production store never uses this implementation.
func (s *sandboxExplorerRegistry) CompareAndSwap(before,after explorerRegistrationSnapshot)error{
 actual,err:=s.Snapshot()
 if err!=nil{return err}
 if actual!=before{return errExplorerRegistryChanged}
 s.writes++
 if s.failBeforeWrite==s.writes{return errors.New("injected prewrite failure")}
 if before.ServerPath!=after.ServerPath{
  if after.ServerPath==""{err=sandboxRemoveTree(s.serverKey)}else{err=sandboxWriteString(s.serverKey,after.ServerPath)}
 }else if before.HandlerCLSID!=after.HandlerCLSID{
  if after.HandlerCLSID==""{err=sandboxRemoveTree(s.handlerKey)}else{err=sandboxWriteString(s.handlerKey,after.HandlerCLSID)}
 }else{return nil}
 if err!=nil{return err}
 if s.failAfterWrite==s.writes{return errors.New("injected postwrite failure")}
 return nil
}

func TestExplorerActualRegistrySandboxLifecycle(t *testing.T){
 scenarios:=[]struct{name,action string;initial explorerRegistrationSnapshot;final explorerRegistrationSnapshot}{
  {"install","install",explorerRegistrationSnapshot{},explorerRegistrationSnapshot{testExplorerCLSID,testExplorerDLL}},
  {"uninstall","uninstall",explorerRegistrationSnapshot{testExplorerCLSID,testExplorerDLL},explorerRegistrationSnapshot{}},
  {"partial-upgrade","upgrade",explorerRegistrationSnapshot{ServerPath:testExplorerDLL},explorerRegistrationSnapshot{testExplorerCLSID,testExplorerDLL}},
 }
 for _,tc:=range scenarios{
  t.Run(tc.name,func(t *testing.T){
   s:=newSandboxExplorerRegistry(t)
   if tc.initial.ServerPath!=""{if err:=sandboxWriteString(s.serverKey,tc.initial.ServerPath);err!=nil{t.Fatal(err)}}
   if tc.initial.HandlerCLSID!=""{if err:=sandboxWriteString(s.handlerKey,tc.initial.HandlerCLSID);err!=nil{t.Fatal(err)}}
   result,err:=executeExplorerRegistrationTransaction(s,tc.action,testExplorerCLSID,testExplorerDLL)
   if err!=nil || !result.Committed{t.Fatalf("transaction result=%+v err=%v",result,err)}
   actual,err:=s.Snapshot()
   if err!=nil || actual!=tc.final{t.Fatalf("actual=%+v want=%+v err=%v",actual,tc.final,err)}
  })
 }
}

func TestExplorerActualRegistrySandboxRecovery(t *testing.T){
 for _,tc:=range []struct{name string;failBefore,failAfter int}{
  {"first-write-failed",1,0},
  {"second-write-failed",2,0},
  {"first-write-committed-then-error",0,1},
  {"second-write-committed-then-error",0,2},
 } {
  t.Run(tc.name,func(t *testing.T){
   s:=newSandboxExplorerRegistry(t)
   s.failBeforeWrite=tc.failBefore
   s.failAfterWrite=tc.failAfter
   result,err:=executeExplorerRegistrationTransaction(s,"install",testExplorerCLSID,testExplorerDLL)
   if err==nil || !result.RolledBack || result.RollbackConflict || result.Committed {
    t.Fatalf("unexpected result=%+v err=%v",result,err)
   }
   actual,err:=s.Snapshot()
   if err!=nil || actual!=(explorerRegistrationSnapshot{}){t.Fatalf("failed rollback: %+v err=%v",actual,err)}
  })
 }
}

func TestExplorerActualRegistrySandboxForeignOwnership(t *testing.T){
 s:=newSandboxExplorerRegistry(t)
 foreign:=`C:\OtherProduct\DCMQuickSortExplorerSend.dll`
 if err:=sandboxWriteString(s.serverKey,foreign);err!=nil{t.Fatal(err)}
 result,err:=executeExplorerRegistrationTransaction(s,"install",testExplorerCLSID,testExplorerDLL)
 if err==nil || result.Committed || s.writes!=0{t.Fatalf("foreign registration not protected: result=%+v err=%v",result,err)}
 actual,err:=s.Snapshot()
 if err!=nil || actual.ServerPath!=foreign{t.Fatalf("foreign server altered: %+v err=%v",actual,err)}
}

func TestExplorerActualRegistrySandboxNoRealShellRegistration(t *testing.T) {
 live:=newWindowsExplorerRegistryStore()
 before,err:=live.Snapshot()
 if err!=nil{t.Fatal(err)}
 if err:=live.CompareAndSwap(before,before);!errors.Is(err,errExplorerRegistryWritesDisabled){t.Fatalf("production write gate failed: %v",err)}
 after,err:=live.Snapshot()
 if err!=nil || before!=after{t.Fatalf("live shell state changed: before=%+v after=%+v err=%v",before,after,err)}
 t.Log(fmt.Sprintf("live handler present=%t, server present=%t (read-only)",after.HandlerCLSID!="",after.ServerPath!=""))
}
