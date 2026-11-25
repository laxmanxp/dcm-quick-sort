//go:build windows

package main

import (
 "crypto/sha256"
 "encoding/hex"
 _ "embed"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "syscall"
 "unsafe"
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
type explorerRegistryLocations struct{
 handler string
 server string
}
var productionExplorerRegistryLocations=explorerRegistryLocations{explorerSendHandlerKey,explorerSendServerKey}

func snapshotExplorerLocations(keys explorerRegistryLocations)(explorerRegistrationSnapshot,error){
 handler,err:=readExplorerDefaultValue(keys.handler)
 if err!=nil{return explorerRegistrationSnapshot{},err}
 server,err:=readExplorerDefaultValue(keys.server)
 if err!=nil{return explorerRegistrationSnapshot{},err}
 return explorerRegistrationSnapshot{HandlerCLSID:handler,ServerPath:server},nil
}
func verifyExplorerRegistryOwner(expectedDLL string) error {
 return verifyExplorerRegistryOwnerAt(expectedDLL,productionExplorerRegistryLocations)
}
func verifyExplorerRegistryOwnerAt(expectedDLL string,keys explorerRegistryLocations) error {
 state,err:=snapshotExplorerLocations(keys)
 if err!=nil{return err}
 _,err=decideExplorerLifecycle("upgrade",state,explorerSendCLSID,expectedDLL)
 return err
}

func writeEmbeddedExplorerFile(path string, payload []byte) error {
 if len(payload)==0{return errors.New("empty embedded Explorer component")}
 if info,err:=os.Lstat(path);err==nil {
  if !info.Mode().IsRegular(){return fmt.Errorf("refusing non-file Explorer payload: %s",path)}
  existing,readErr:=os.ReadFile(path)
  if readErr!=nil{return readErr}
  if sha256.Sum256(existing)==sha256.Sum256(payload){return nil}
  return fmt.Errorf("Explorer component already exists with different bytes: %s; close the old Explorer extension before upgrading",path)
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
 return installExplorerSendComponentAt(installDir,productionExplorerRegistryLocations)
}
func installExplorerSendComponentAt(installDir string,keys explorerRegistryLocations) error {
 dll,helper:=explorerInstalledPaths(installDir)
 if err:=verifyExplorerRegistryOwnerAt(dll,keys);err!=nil{return fmt.Errorf("Explorer registration ownership: %w",err)}
 // A pre-existing component file must never be silently destroyed on upgrade.
 if err:=writeEmbeddedExplorerFile(dll,explorerDLLPayload);err!=nil{return err}
 if err:=writeEmbeddedExplorerFile(helper,explorerHelperPayload);err!=nil{return err}
 state,err:=snapshotExplorerLocations(keys)
 if err!=nil{return err}
 // Best-effort rollback of only newly introduced owned entries. Windows
 // registry APIs do not make these separate operations atomic.
 registered:=false
 defer func(){
  if registered{return}
  current,readErr:=snapshotExplorerLocations(keys)
  if readErr!=nil{return}
  if state.HandlerCLSID=="" && strings.EqualFold(current.HandlerCLSID,explorerSendCLSID) {
   deleteRegistryTree(keys.handler)
  }
  if state.ServerPath=="" && strings.EqualFold(filepath.Clean(current.ServerPath),filepath.Clean(dll)) {
   deleteRegistryTree(filepath.Dir(keys.server))
  }
 }()
 // Register server before publishing the Explorer handler.
 if err:=regSetString(keys.server,"",dll);err!=nil{return err}
 if err:=regSetString(keys.server,"ThreadingModel",explorerSendThreadingModel);err!=nil{return err}
 if err:=regSetString(keys.handler,"",explorerSendCLSID);err!=nil{return err}
 registered=true
 return nil
}

// removeOwnedExplorerRegistryTree reports failures instead of treating failed
// COM unregistration as a successful uninstall.
func removeOwnedExplorerRegistryTree(path string) error {
 ptr,err:=syscall.UTF16PtrFromString(path)
 if err!=nil{return err}
 code,_,_:=procRegDeleteTreeW.Call(HKEY_CURRENT_USER,uintptr(unsafe.Pointer(ptr)))
 if code!=0 && code!=uintptr(syscall.ERROR_FILE_NOT_FOUND) && code!=uintptr(syscall.ERROR_PATH_NOT_FOUND) {
  return fmt.Errorf("remove Explorer registration %s: Windows error %d",path,code)
 }
 return nil
}

func uninstallExplorerSendComponent(installDir string) error {
 return uninstallExplorerSendComponentAt(installDir,productionExplorerRegistryLocations)
}
func uninstallExplorerSendComponentAt(installDir string,keys explorerRegistryLocations) error {
 dll,helper:=explorerInstalledPaths(installDir)
 state,err:=snapshotExplorerLocations(keys)
 if err!=nil{return err}
 if _,err=decideExplorerLifecycle("uninstall",state,explorerSendCLSID,dll);err!=nil {
  return fmt.Errorf("refusing to remove foreign Explorer registration: %w",err)
 }
 if strings.EqualFold(strings.TrimSpace(state.HandlerCLSID),explorerSendCLSID) {
  if err:=removeOwnedExplorerRegistryTree(keys.handler);err!=nil{return err}
 }
 if state.ServerPath!="" && strings.EqualFold(filepath.Clean(state.ServerPath),filepath.Clean(dll)) {
  // Remove the class's InprocServer32 parent only when we own its exact path.
  if err:=removeOwnedExplorerRegistryTree(filepath.Dir(keys.server));err!=nil{return err}
 }
 // A loaded DLL may be locked by Explorer. Report this instead of claiming the
 // uninstall succeeded; no Explorer process is forcibly restarted.
 for _,path:=range []string{helper,dll} {
  if err:=os.Remove(path);err!=nil && !os.IsNotExist(err) {return fmt.Errorf("remove Explorer component %s: %w",filepath.Base(path),err)}
 }
 return nil
}

func verifyEmbeddedExplorerRelease() (string,error) {
 items:=[]struct{name string;data []byte}{
  {"DCM-QuickSort.exe",appPayload},
  {"DCMQuickSortExplorerSend.dll",explorerDLLPayload},
  {"DCMQuickSortExplorerSend.exe",explorerHelperPayload},
 }
 var result strings.Builder
 for _,item:=range items {
  if len(item.data)<512 || item.data[0]!='M' || item.data[1]!='Z' {
   return "",fmt.Errorf("invalid or missing embedded executable: %s",item.name)
  }
  hash:=sha256.Sum256(item.data)
  fmt.Fprintf(&result,"%s: %d bytes SHA256 %s\n",item.name,len(item.data),hex.EncodeToString(hash[:]))
 }
 return result.String(),nil
}
