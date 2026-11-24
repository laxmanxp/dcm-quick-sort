//go:build windows

package main

import (
 "errors"
 "testing"
)

func TestWindowsExplorerRegistryAdapterMutationGate(t *testing.T){
 store:=newWindowsExplorerRegistryStore()
 before:=explorerRegistrationSnapshot{}
 after:=explorerRegistrationSnapshot{HandlerCLSID:testExplorerCLSID,ServerPath:testExplorerDLL}
 if err:=store.CompareAndSwap(before,after);!errors.Is(err,errExplorerRegistryWritesDisabled){
  t.Fatalf("unexpected write gate: %v",err)
 }
 if err:=store.CompareAndSwap(before,before);!errors.Is(err,errExplorerRegistryWritesDisabled){
  t.Fatalf("even no-op write must stay disabled: %v",err)
 }
}

func TestWindowsExplorerRegistryAdapterReadOnlySnapshot(t *testing.T) {
 store:=newWindowsExplorerRegistryStore()
 before,err:=store.Snapshot()
 if err!=nil{t.Fatalf("read HKCU Explorer keys: %v",err)}
 after,err:=store.Snapshot()
 if err!=nil{t.Fatalf("reread HKCU Explorer keys: %v",err)}
 if before!=after{t.Fatalf("Explorer registrations changed during read: before=%+v after=%+v",before,after)}
 t.Logf("read-only HKCU snapshot: handler-present=%v server-present=%v",before.HandlerCLSID!="",before.ServerPath!="")
}
