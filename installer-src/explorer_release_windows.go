//go:build windows

package main

import (
 "crypto/sha256"
 _ "embed"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
)

// These are compiled into the real installer, not loaded from an arbitrary
// directory at installation time. The release build must stage both payloads.
//go:embed payload/DCMQuickSortExplorerSend.dll
var explorerDLLPayload []byte

//go:embed payload/DCMQuickSortExplorerSend.exe
var explorerHelperPayload []byte

const explorerSendCLSID = "{8F2DA0C1-7D79-40AA-A030-19645D8C4A91}"
const explorerSendThreadingModel = "Apartment"

func explorerInstalledPaths(installDir string) (string,string) {
 return filepath.Join(installDir,"DCMQuickSortExplorerSend.dll"),filepath.Join(installDir,"DCMQuickSortExplorerSend.exe")
}

// Refuse to overwrite any COM class not pointing to this particular install.
// The live registry operations are not atomic: never claim they are CAS.
func verifyExplorerRegistryOwner(expectedDLL string) error {
 state,err:=newWindowsExplorerRegistryStore().Snapshot()
 if err!=nil{return err}
 _,err=decideExplorerLifecycle("upgrade",state,explorerSendCLSID,expectedDLL)
 return err
}

func writeEmbeddedExplorerFile(path string, payload []byte) error {
 if len(payload)==0{return errors.New("empty embedded Explorer component")}
 if info,err:=os.Lstat(path);err==nil {
  if !info.Mode().IsRegular(){return fmt.Errorf("refusing non-file Explorer payload: %s",path)}
 }else if !os.IsNotExist(err){return err}
 tmp:=path+".new"
 if err:=os.WriteFile(tmp,payload,0600);err!=nil{return err}
 got,err:=os.ReadFile(tmp)
 if err!=nil{_ = os.Remove(tmp);return err}
 if sha256.Sum256(got)!=sha256.Sum256(payload){_ = os.Remove(tmp);return errors.New("Explorer payload integrity mismatch")}
 // Windows does not permit renaming over a loaded DLL. Fail safely instead of
 // tearing down a running Explorer session.
 if err:=os.Rename(tmp,path);err!=nil {
  _=os.Remove(tmp)
  return fmt.Errorf("activate Explorer component %s: %w",filepath.Base(path),err)
 }
 return nil
}

func installExplorerSendComponent(installDir string) error {
 dll,helper:=explorerInstalledPaths(installDir)
 if err:=verifyExplorerRegistryOwner(dll);err!=nil{return fmt.Errorf("Explorer registration ownership: %w",err)}
 // A pre-existing component file must never be silently destroyed on upgrade.
 if err:=writeEmbeddedExplorerFile(dll,explorerDLLPayload);err!=nil{return err}
 if err:=writeEmbeddedExplorerFile(helper,explorerHelperPayload);err!=nil{return err}
 state,err:=newWindowsExplorerRegistryStore().Snapshot()
 if err!=nil{return err}
 // Register server before publishing the Explorer handler.
 if err:=regSetString(explorerSendServerKey,"",dll);err!=nil{return err}
 if err:=regSetString(explorerSendServerKey,"ThreadingModel",explorerSendThreadingModel);err!=nil{return err}
 if err:=regSetString(explorerSendHandlerKey,"",explorerSendCLSID);err!=nil{return err}
 _=state // The earlier ownership snapshot prevents knowingly taking foreign keys.
 return nil
}

func uninstallExplorerSendComponent(installDir string) error {
 dll,helper:=explorerInstalledPaths(installDir)
 state,err:=newWindowsExplorerRegistryStore().Snapshot()
 if err!=nil{return err}
 if _,err=decideExplorerLifecycle("uninstall",state,explorerSendCLSID,dll);err!=nil {
  return fmt.Errorf("refusing to remove foreign Explorer registration: %w",err)
 }
 if strings.EqualFold(strings.TrimSpace(state.HandlerCLSID),explorerSendCLSID) {
  deleteRegistryTree(explorerSendHandlerKey)
 }
 if state.ServerPath!="" && strings.EqualFold(filepath.Clean(state.ServerPath),filepath.Clean(dll)) {
  // Remove the class's InprocServer32 parent only when we own its exact path.
  deleteRegistryTree(filepath.Dir(explorerSendServerKey))
 }
 // A loaded DLL may be locked by Explorer. Report this instead of claiming the
 // uninstall succeeded; no Explorer process is forcibly restarted.
 for _,path:=range []string{helper,dll} {
  if err:=os.Remove(path);err!=nil && !os.IsNotExist(err) {return fmt.Errorf("remove Explorer component %s: %w",filepath.Base(path),err)}
 }
 return nil
}
