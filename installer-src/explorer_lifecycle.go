package main

import (
 "errors"
 "path/filepath"
 "strings"
)

// explorerRegistrationSnapshot represents the two registry values that identify
// the QuickSort Send shell extension. Empty means the value is absent.
type explorerRegistrationSnapshot struct {
 HandlerCLSID string
 ServerPath string
}

// explorerLifecycleDecision is a pure, side-effect-free plan. No code here
// writes to the Windows registry or changes an installed application.
type explorerLifecycleDecision struct {
 RegisterHandler bool
 RegisterServer bool
 RemoveHandler bool
 RemoveServer bool
 Noop bool
}

// decideExplorerLifecycle fails closed if either registry value belongs to
// another COM registration. Only the exact canonical installed path is owned.
func decideExplorerLifecycle(action string, state explorerRegistrationSnapshot, expectedCLSID, expectedDLL string) (explorerLifecycleDecision,error) {
 var plan explorerLifecycleDecision
 if expectedCLSID=="" || strings.TrimSpace(expectedDLL)=="" || !filepath.IsAbs(expectedDLL) {
  return plan,errors.New("expected CLSID and absolute DLL path are required")
 }
 if action!="install" && action!="upgrade" && action!="uninstall" {
  return plan,errors.New("unsupported Explorer lifecycle action")
 }
 handler:=strings.TrimSpace(state.HandlerCLSID)
 server:=strings.TrimSpace(state.ServerPath)
 if handler!="" && !strings.EqualFold(handler,expectedCLSID) {
  return plan,errors.New("Explorer context handler is owned by another COM class")
 }
 if server!="" {
  if !filepath.IsAbs(server) || !strings.EqualFold(filepath.Clean(server),filepath.Clean(expectedDLL)) {
   return plan,errors.New("COM server is owned by another installation")
  }
 }
 ownedHandler:=handler!=""
 ownedServer:=server!=""
 switch action {
 case "install","upgrade":
  plan.RegisterHandler=!ownedHandler
  plan.RegisterServer=!ownedServer
 case "uninstall":
  plan.RemoveHandler=ownedHandler
  plan.RemoveServer=ownedServer
 }
 plan.Noop=!(plan.RegisterHandler||plan.RegisterServer||plan.RemoveHandler||plan.RemoveServer)
 return plan,nil
}
