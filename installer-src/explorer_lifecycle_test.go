package main

import "testing"

const testExplorerCLSID="{8F2DA0C1-7D79-40AA-A030-19645D8C4A91}"
const testExplorerDLL=`C:\Users\fixture\AppData\Local\Programs\DCM Quick Sort\DCMQuickSortExplorerSend.dll`

func TestExplorerLifecycleOwnershipMatrix(t *testing.T) {
 tests:=[]struct{
  name,action string
  state explorerRegistrationSnapshot
  want explorerLifecycleDecision
  reject bool
 }{
  {"fresh-install","install",explorerRegistrationSnapshot{},explorerLifecycleDecision{RegisterHandler:true,RegisterServer:true},false},
  {"repeat-install","install",explorerRegistrationSnapshot{testExplorerCLSID,testExplorerDLL},explorerLifecycleDecision{Noop:true},false},
  {"upgrade-owned","upgrade",explorerRegistrationSnapshot{testExplorerCLSID,testExplorerDLL},explorerLifecycleDecision{Noop:true},false},
  {"repair-missing-handler","upgrade",explorerRegistrationSnapshot{"",testExplorerDLL},explorerLifecycleDecision{RegisterHandler:true},false},
  {"repair-missing-server","upgrade",explorerRegistrationSnapshot{testExplorerCLSID,""},explorerLifecycleDecision{RegisterServer:true},false},
  {"uninstall-absent","uninstall",explorerRegistrationSnapshot{},explorerLifecycleDecision{Noop:true},false},
  {"uninstall-owned","uninstall",explorerRegistrationSnapshot{testExplorerCLSID,testExplorerDLL},explorerLifecycleDecision{RemoveHandler:true,RemoveServer:true},false},
  {"uninstall-orphan-handler","uninstall",explorerRegistrationSnapshot{testExplorerCLSID,""},explorerLifecycleDecision{RemoveHandler:true},false},
  {"uninstall-orphan-server","uninstall",explorerRegistrationSnapshot{"",testExplorerDLL},explorerLifecycleDecision{RemoveServer:true},false},
  {"foreign-handler-install","install",explorerRegistrationSnapshot{"{00000000-0000-0000-0000-000000000000}",testExplorerDLL},explorerLifecycleDecision{},true},
  {"foreign-server-upgrade","upgrade",explorerRegistrationSnapshot{testExplorerCLSID,`C:\OtherApp\DCMQuickSortExplorerSend.dll`},explorerLifecycleDecision{},true},
  {"foreign-server-uninstall","uninstall",explorerRegistrationSnapshot{testExplorerCLSID,`C:\OtherApp\DCMQuickSortExplorerSend.dll`},explorerLifecycleDecision{},true},
  {"foreign-handler-uninstall","uninstall",explorerRegistrationSnapshot{"{00000000-0000-0000-0000-000000000000}",""},explorerLifecycleDecision{},true},
  {"foreign-both","uninstall",explorerRegistrationSnapshot{"{00000000-0000-0000-0000-000000000000}",`C:\OtherApp\DCMQuickSortExplorerSend.dll`},explorerLifecycleDecision{},true},
  {"unsupported","repair",explorerRegistrationSnapshot{},explorerLifecycleDecision{},true},
 }
 for _,tc:=range tests {
  t.Run(tc.name,func(t *testing.T){
   got,err:=decideExplorerLifecycle(tc.action,tc.state,testExplorerCLSID,testExplorerDLL)
   if (err!=nil)!=tc.reject {t.Fatalf("err=%v reject=%v",err,tc.reject)}
   if got!=tc.want {t.Fatalf("got %+v want %+v",got,tc.want)}
  })
 }
}
func TestExplorerLifecycleRequiresBoundedIdentity(t *testing.T) {
 for _,tc:=range []struct{clsid,path string}{
  {"",testExplorerDLL},
  {testExplorerCLSID,""},
  {testExplorerCLSID,"relative\\extension.dll"},
 }{
  if _,err:=decideExplorerLifecycle("install",explorerRegistrationSnapshot{},tc.clsid,tc.path);err==nil {
   t.Fatalf("accepted incomplete identity: %#v",tc)
  }
 }
}
