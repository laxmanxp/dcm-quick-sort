package main

// Explorer Send is deliberately a separate, opt-in installer component.
// This module only builds a validated packaging plan. It never copies payloads,
// registers COM, restarts Explorer, or changes the existing installer behavior.
import (
 "bytes"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
)

const explorerComponentName = "DCM QuickSort Explorer Send"

type explorerPayload struct {
 Name string `json:"name"`
 Length int64 `json:"length"`
 SHA256 string `json:"sha256"`
}
type explorerManifest struct {
 Schema int `json:"schema"`
 Component string `json:"component"`
 SourceCommit string `json:"source_commit"`
 EnabledByDefault bool `json:"enabled_by_default"`
 InstallationPerformed bool `json:"installation_performed"`
 RegistryModified bool `json:"registry_modified"`
 ExplorerRestarted bool `json:"explorer_restarted"`
 GUIAccepted bool `json:"gui_accepted"`
 InstallerQualified bool `json:"installer_qualified"`
 Passed bool `json:"passed"`
 Files []explorerPayload `json:"files"`
}
type explorerInstallPlan struct {
 SourceCommit string
 Payloads []explorerPayload
 ActivationAllowed bool
}

// planExplorerComponent validates a staged candidate for subsequent packaging.
// Activation is always blocked pending real Explorer GUI and installer acceptance.
func planExplorerComponent(packageDir string, requested bool) (explorerInstallPlan,error) {
 if !requested {return explorerInstallPlan{},nil}
 if strings.TrimSpace(packageDir)=="" {return explorerInstallPlan{},errors.New("Explorer package path required")}
 raw,err:=os.ReadFile(filepath.Join(packageDir,"manifest.json"))
 if err!=nil{return explorerInstallPlan{},fmt.Errorf("read Explorer manifest: %w",err)}
 var manifest explorerManifest
 raw=bytes.TrimPrefix(raw,[]byte{0xef,0xbb,0xbf})
 if err=json.Unmarshal(raw,&manifest);err!=nil{return explorerInstallPlan{},fmt.Errorf("decode Explorer manifest: %w",err)}
 if manifest.Schema!=1 || manifest.Component!=explorerComponentName || !manifest.Passed ||
    manifest.EnabledByDefault || manifest.InstallationPerformed || manifest.RegistryModified || manifest.ExplorerRestarted {
  return explorerInstallPlan{},errors.New("invalid Explorer component manifest or unsafe staging state")
 }
 if len(manifest.Files)!=2 {return explorerInstallPlan{},errors.New("expected exactly two Explorer payloads")}
 expected:=map[string]bool{"DCMQuickSortExplorerSend.dll":true,"DCMQuickSortExplorerSend.exe":true}
 seen:=map[string]bool{}
 for _,item:=range manifest.Files {
  if !expected[item.Name] || seen[item.Name] || item.Length<=0 || len(item.SHA256)!=64 {
   return explorerInstallPlan{},errors.New("invalid Explorer payload entry")
  }
  seen[item.Name]=true
  path:=filepath.Join(packageDir,"payload",item.Name)
  data,err:=os.ReadFile(path)
  if err!=nil{return explorerInstallPlan{},fmt.Errorf("read Explorer payload %s: %w",item.Name,err)}
  sum:=sha256.Sum256(data)
  if int64(len(data))!=item.Length || !strings.EqualFold(hex.EncodeToString(sum[:]),item.SHA256) {
   return explorerInstallPlan{},fmt.Errorf("Explorer payload integrity mismatch: %s",item.Name)
  }
 }
 // Neither a staging manifest nor the caller's request is sufficient to
 // authorize Explorer shell registration. That requires separate GUI acceptance.
 return explorerInstallPlan{SourceCommit:manifest.SourceCommit,Payloads:manifest.Files,ActivationAllowed:false},nil
}

 // explorerPreflightArgument recognizes one explicit verification-only CLI mode.
 // It never installs or activates an extension.
func explorerPreflightArgument(args []string)(string,bool,error){
 const prefix="--verify-explorer-package="
 requested:=false
 path:=""
 for _,arg:=range args {
  if strings.HasPrefix(strings.ToLower(arg),prefix) {
   if requested{return "",true,errors.New("Explorer verification option supplied twice")}
   requested=true
   path=strings.TrimSpace(arg[len(prefix):])
  } else if requested || len(args)>1 {
   // Once the explicit preflight mode is used, reject any second argument.
   // Ordinary single-argument installer invocations remain unchanged.
   if len(args)>1 {
    for _,a:=range args {if strings.HasPrefix(strings.ToLower(a),prefix){return "",true,errors.New("Explorer preflight cannot be combined with install or uninstall arguments")}}
   }
  }
 }
 if requested && path=="" {return "",true,errors.New("Explorer package path is empty")}
 return path,requested,nil
}
