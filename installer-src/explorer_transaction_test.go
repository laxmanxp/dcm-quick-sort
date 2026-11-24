package main

import (
 "errors"
 "strings"
 "testing"
)

type fakeExplorerStore struct {
 current explorerRegistrationSnapshot
 writes int
 failOnWrite int
 concurrentOnWrite int
 injectForeignOnWrite int
 failReadAt int
 reads int
}
func (s *fakeExplorerStore) Snapshot()(explorerRegistrationSnapshot,error){
 s.reads++
 if s.failReadAt>0 && s.reads==s.failReadAt {return explorerRegistrationSnapshot{},errors.New("injected read failure")}
 return s.current,nil
}
func (s *fakeExplorerStore) CompareAndSwap(before,after explorerRegistrationSnapshot)error{
 s.writes++
 if s.concurrentOnWrite>0 && s.writes==s.concurrentOnWrite {
  s.current.HandlerCLSID="{00000000-0000-0000-0000-000000000000}"
  return errExplorerRegistryChanged
 }
 if s.injectForeignOnWrite>0 && s.writes==s.injectForeignOnWrite {
  s.current.ServerPath=`C:\OtherApp\DCMQuickSortExplorerSend.dll`
  return errExplorerRegistryChanged
 }
 if s.failOnWrite>0 && s.writes==s.failOnWrite {return errors.New("injected write failure")}
 if s.current!=before {return errExplorerRegistryChanged}
 s.current=after
 return nil
}
func TestExplorerTransactionLifecycle(t *testing.T){
 own:=explorerRegistrationSnapshot{HandlerCLSID:testExplorerCLSID,ServerPath:testExplorerDLL}
 cases:=[]struct{
  name,action string
  before,after explorerRegistrationSnapshot
  steps int
 }{
  {"fresh-install","install",explorerRegistrationSnapshot{},own,2},
  {"repeat-install","install",own,own,0},
  {"upgrade-partial-handler","upgrade",explorerRegistrationSnapshot{HandlerCLSID:testExplorerCLSID},own,1},
  {"upgrade-partial-server","upgrade",explorerRegistrationSnapshot{ServerPath:testExplorerDLL},own,1},
  {"uninstall-full","uninstall",own,explorerRegistrationSnapshot{},2},
  {"uninstall-partial-handler","uninstall",explorerRegistrationSnapshot{HandlerCLSID:testExplorerCLSID},explorerRegistrationSnapshot{},1},
  {"uninstall-absent","uninstall",explorerRegistrationSnapshot{},explorerRegistrationSnapshot{},0},
 }
 for _,tc:=range cases {
  t.Run(tc.name,func(t *testing.T){
   s:=&fakeExplorerStore{current:tc.before}
   result,err:=executeExplorerRegistrationTransaction(s,tc.action,testExplorerCLSID,testExplorerDLL)
   if err!=nil || !result.Committed || result.RolledBack || result.StepsApplied!=tc.steps || s.current!=tc.after {
    t.Fatalf("result=%+v err=%v current=%+v",result,err,s.current)
   }
  })
 }
}
func TestExplorerTransactionRollbackAndConflicts(t *testing.T){
 initial:=explorerRegistrationSnapshot{}
 full:=explorerRegistrationSnapshot{HandlerCLSID:testExplorerCLSID,ServerPath:testExplorerDLL}
 cases:=[]struct {
  name,action string
  before explorerRegistrationSnapshot
  failOnWrite,concurrentOnWrite,injectForeignOnWrite,failReadAt int
  wantRollback,wantConflict bool
  wantState explorerRegistrationSnapshot
 }{
  {"fail-first-step","install",initial,1,0,0,0,true,false,initial},
  {"fail-second-step","install",initial,2,0,0,0,true,false,initial},
  {"failed-read-after-first-step","install",initial,0,0,0,3,true,false,initial},
  {"uninstall-fail-second-step","uninstall",full,2,0,0,0,true,false,full},
  {"concurrent-first-step","install",initial,0,1,0,0,true,false,explorerRegistrationSnapshot{HandlerCLSID:"{00000000-0000-0000-0000-000000000000}"}},
  {"concurrent-second-step","install",initial,0,2,0,0,false,true,explorerRegistrationSnapshot{HandlerCLSID:"{00000000-0000-0000-0000-000000000000}",ServerPath:testExplorerDLL}},
  {"foreign-server-second-step","install",initial,0,0,2,0,false,true,explorerRegistrationSnapshot{ServerPath:`C:\OtherApp\DCMQuickSortExplorerSend.dll`}},
 }
 for _,tc:=range cases {
  t.Run(tc.name,func(t *testing.T){
   s:=&fakeExplorerStore{current:tc.before,failOnWrite:tc.failOnWrite,concurrentOnWrite:tc.concurrentOnWrite,injectForeignOnWrite:tc.injectForeignOnWrite,failReadAt:tc.failReadAt}
   result,err:=executeExplorerRegistrationTransaction(s,tc.action,testExplorerCLSID,testExplorerDLL)
   if err==nil || result.Committed || result.RolledBack!=tc.wantRollback || result.RollbackConflict!=tc.wantConflict || s.current!=tc.wantState {
    t.Fatalf("result=%+v err=%v current=%+v want=%+v",result,err,s.current,tc.wantState)
   }
  })
 }
}
func TestExplorerTransactionForeignOwnershipNeverWrites(t *testing.T){
 for _,action:=range []string{"install","upgrade","uninstall"} {
  s:=&fakeExplorerStore{current:explorerRegistrationSnapshot{ServerPath:`C:\Another\DCMQuickSortExplorerSend.dll`}}
  _,err:=executeExplorerRegistrationTransaction(s,action,testExplorerCLSID,testExplorerDLL)
  if err==nil || s.writes!=0 || !strings.Contains(err.Error(),"owned by another installation") {t.Fatalf("action=%s err=%v writes=%d",action,err,s.writes)}
 }
}
