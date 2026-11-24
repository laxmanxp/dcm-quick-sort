package main

import (
 "errors"
 "fmt"
)

// explorerRegistryStore is deliberately abstract. This file has no production
// registry adapter: tests use an in-memory fake and do not touch HKCU or Explorer.
type explorerRegistryStore interface {
 Snapshot() (explorerRegistrationSnapshot, error)
 CompareAndSwap(before, after explorerRegistrationSnapshot) error
}

var errExplorerRegistryChanged = errors.New("Explorer registration changed concurrently")

// explorerTransactionResult distinguishes a clean commit, an ordinary rollback,
// and a rollback that was blocked by a concurrent modification.
type explorerTransactionResult struct {
 Committed bool
 RolledBack bool
 RollbackConflict bool
 StepsApplied int
}

// executeExplorerRegistrationTransaction validates the initial ownership,
// modifies one value at a time, rechecks ownership before each step, and
// compensates only values that still match this transaction's writes.
// A failure may need manual inspection if another process changed registry
// state concurrently; it is never safe to overwrite that other process.
func executeExplorerRegistrationTransaction(
 store explorerRegistryStore, action string, expectedCLSID, expectedDLL string,
) (explorerTransactionResult, error) {
 var result explorerTransactionResult
 original, err := store.Snapshot()
 if err != nil { return result,fmt.Errorf("snapshot Explorer registration: %w",err) }
 plan, err := decideExplorerLifecycle(action,original,expectedCLSID,expectedDLL)
 if err != nil { return result,err }
 if plan.Noop {result.Committed=true;return result,nil}

 // Register the server before advertising its handler. On removal, hide the
 // handler before removing the server. This avoids an orphan live menu entry.
 steps:=make([]explorerRegistrationSnapshot,0,2)
 current:=original
 if plan.RegisterServer {
  current.ServerPath=expectedDLL;steps=append(steps,current)
 }
 if plan.RegisterHandler {
  current.HandlerCLSID=expectedCLSID;steps=append(steps,current)
 }
 if plan.RemoveHandler {
  current.HandlerCLSID="";steps=append(steps,current)
 }
 if plan.RemoveServer {
  current.ServerPath="";steps=append(steps,current)
 }

 applied:=make([]explorerRegistrationSnapshot,0,len(steps)+1)
 applied=append(applied,original)
 for _,next:=range steps {
  before:=applied[len(applied)-1]
  // Fail closed if a previously valid pair has become foreign.
  live, readErr:=store.Snapshot()
  if readErr != nil {err=fmt.Errorf("resnapshot registration: %w",readErr);break}
  if live!=before {err=errExplorerRegistryChanged;break}
  if _,checkErr:=decideExplorerLifecycle(action,live,expectedCLSID,expectedDLL);checkErr!=nil {
   err=fmt.Errorf("registration ownership changed: %w",checkErr);break
  }
  if writeErr:=store.CompareAndSwap(before,next);writeErr!=nil {
   err=fmt.Errorf("apply registration step: %w",writeErr)
   // A registry adapter may fail after a write has reached Windows. Inspect
   // its post-error state before deciding whether a compensating write is safe.
   observed,inspectErr:=store.Snapshot()
   if inspectErr!=nil {
    result.RollbackConflict=true
    return result,errors.Join(err,fmt.Errorf("cannot inspect failed write: %w",inspectErr))
   }
   if observed==next {
    // This operation completed despite reporting an error. It must be undone.
    applied=append(applied,next)
    result.StepsApplied++
   } else if observed!=before {
    // Foreign/intermediate state: never overwrite another writer.
    result.RollbackConflict=true
    return result,errors.Join(err,errExplorerRegistryChanged)
   }
   break
  }
  applied=append(applied,next)
  result.StepsApplied++
 }
 if err==nil {result.Committed=true;return result,nil}

 // Roll back in reverse order, and never force a foreign or concurrent value.
 rollbackOK:=true
 for i:=len(applied)-1;i>0;i-- {
  if rollbackErr:=store.CompareAndSwap(applied[i],applied[i-1]);rollbackErr!=nil {
   // A rollback itself may have succeeded before returning an error.
   observed,inspectErr:=store.Snapshot()
   if inspectErr==nil && observed==applied[i-1] {
    continue
   }
   rollbackOK=false
   result.RollbackConflict=true
   err=errors.Join(err,fmt.Errorf("rollback step %d: %w",i,rollbackErr))
   if inspectErr!=nil {err=errors.Join(err,inspectErr)}
   break
  }
 }
 result.RolledBack=rollbackOK
 return result,err
}
