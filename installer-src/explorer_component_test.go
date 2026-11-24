package main

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
)

func makeExplorerPackage(t *testing.T) string {
 t.Helper()
 dir:=t.TempDir()
 payloadDir:=filepath.Join(dir,"payload")
 if err:=os.Mkdir(payloadDir,0700);err!=nil{t.Fatal(err)}
 names:=[]string{"DCMQuickSortExplorerSend.dll","DCMQuickSortExplorerSend.exe"}
 m:=explorerManifest{Schema:1,Component:explorerComponentName,SourceCommit:"test-commit",Passed:true}
 for _,name:=range names {
  data:=[]byte("fixture payload "+name)
  if err:=os.WriteFile(filepath.Join(payloadDir,name),data,0600);err!=nil{t.Fatal(err)}
  hash:=sha256.Sum256(data)
  m.Files=append(m.Files,explorerPayload{Name:name,Length:int64(len(data)),SHA256:hex.EncodeToString(hash[:])})
 }
 writeExplorerManifest(t,dir,m)
 return dir
}
func writeExplorerManifest(t *testing.T,dir string,m explorerManifest){
 t.Helper()
 bytes,err:=json.Marshal(m)
 if err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"manifest.json"),bytes,0600);err!=nil{t.Fatal(err)}
}
func readExplorerManifest(t *testing.T,dir string) explorerManifest{
 t.Helper()
 var m explorerManifest
 b,err:=os.ReadFile(filepath.Join(dir,"manifest.json"))
 if err!=nil{t.Fatal(err)}
 if err=json.Unmarshal(b,&m);err!=nil{t.Fatal(err)}
 return m
}
func TestExplorerComponentDefaultOff(t *testing.T){
 plan,err:=planExplorerComponent("",false)
 if err!=nil || plan.ActivationAllowed || len(plan.Payloads)!=0{t.Fatalf("default-off plan=%+v err=%v",plan,err)}
}
func TestExplorerComponentRequiresPackage(t *testing.T){
 if _,err:=planExplorerComponent("",true);err==nil{t.Fatal("accepted empty package")}
}
func TestExplorerComponentValidatedButNotActivated(t *testing.T){
 dir:=makeExplorerPackage(t)
 plan,err:=planExplorerComponent(dir,true)
 if err!=nil{t.Fatal(err)}
 if plan.ActivationAllowed || len(plan.Payloads)!=2 || plan.SourceCommit!="test-commit"{
  t.Fatalf("unexpected plan: %+v",plan)
 }
}
func TestExplorerComponentTamperedDLLRejected(t *testing.T){
 dir:=makeExplorerPackage(t)
 if err:=os.WriteFile(filepath.Join(dir,"payload","DCMQuickSortExplorerSend.dll"),[]byte("tampered"),0600);err!=nil{t.Fatal(err)}
 if _,err:=planExplorerComponent(dir,true);err==nil{t.Fatal("accepted tampered DLL")}
}
func TestExplorerComponentMissingHelperRejected(t *testing.T){
 dir:=makeExplorerPackage(t)
 if err:=os.Remove(filepath.Join(dir,"payload","DCMQuickSortExplorerSend.exe"));err!=nil{t.Fatal(err)}
 if _,err:=planExplorerComponent(dir,true);err==nil{t.Fatal("accepted missing helper")}
}
func TestExplorerComponentUnsafeFlagsRejected(t *testing.T){
 for _,tc:=range []struct{name string;mutate func(*explorerManifest)}{
  {"default-on",func(m *explorerManifest){m.EnabledByDefault=true}},
  {"registered",func(m *explorerManifest){m.RegistryModified=true}},
  {"installed",func(m *explorerManifest){m.InstallationPerformed=true}},
  {"restarted",func(m *explorerManifest){m.ExplorerRestarted=true}},
  {"failed",func(m *explorerManifest){m.Passed=false}},
 }{
  t.Run(tc.name,func(t *testing.T){
   dir:=makeExplorerPackage(t)
   m:=readExplorerManifest(t,dir)
   tc.mutate(&m);writeExplorerManifest(t,dir,m)
   if _,err:=planExplorerComponent(dir,true);err==nil{t.Fatal("accepted unsafe manifest")}
  })
 }
}
func TestExplorerComponentDuplicateAndUnknownFilesRejected(t *testing.T){
 for _,name:=range []string{"duplicate","unknown","short-hash"}{
  t.Run(name,func(t *testing.T){
   dir:=makeExplorerPackage(t)
   m:=readExplorerManifest(t,dir)
   switch name {
   case "duplicate":m.Files[1]=m.Files[0]
   case "unknown":m.Files[1].Name="unrelated.exe"
   case "short-hash":m.Files[1].SHA256="abc"
   }
   writeExplorerManifest(t,dir,m)
   if _,err:=planExplorerComponent(dir,true);err==nil{t.Fatal("accepted malformed file list")}
  })
 }
}

func TestExplorerPreflightCLI(t *testing.T){
 cases:=[]struct{args []string;requested bool;valid bool}{
  {[]string{},false,true},
  {[]string{"/S"},false,true},
  {[]string{"--verify-explorer-package=C:\\fixture"},true,true},
  {[]string{"--verify-explorer-package="},true,false},
  {[]string{"--verify-explorer-package=C:\\fixture","/S"},true,false},
  {[]string{"/uninstall","--verify-explorer-package=C:\\fixture"},true,false},
  {[]string{"--verify-explorer-package=C:\\one","--verify-explorer-package=C:\\two"},true,false},
 }
 for _,tc:=range cases{
  _,requested,err:=explorerPreflightArgument(tc.args)
  if requested!=tc.requested||(err==nil)!=tc.valid{t.Errorf("args=%v requested=%v err=%v",tc.args,requested,err)}
 }
}
