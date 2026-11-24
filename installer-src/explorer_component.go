package main

// Explorer Send is deliberately a separate, opt-in installer component.
// This module only builds a validated packaging plan. It never copies payloads,
// registers COM, restarts Explorer, or changes the existing installer behavior.
import (
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
