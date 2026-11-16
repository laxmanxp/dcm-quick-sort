// flowdev-test is the isolated DCM QuickSort FlowDev profile adapter.
// It does not modify the monorepo FlowDev registry or application source.
package main

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
)

const appName = "dcm-quicksort"
type registry struct { Apps map[string]struct{Tests map[string][]string `json:"tests"`} `json:"apps"` }
type repoRow struct { app, directory, origin, branch string }

func parseRepoRows(b []byte) ([]repoRow,error) {
 var rows []repoRow
 for _,raw := range strings.Split(string(b),string([]byte{10})) {
  line:=strings.TrimSpace(raw)
  if line=="" || strings.HasPrefix(line,"#") {continue}
  parts:=strings.Split(line,"|")
  if len(parts)!=4 {return nil,fmt.Errorf("invalid repos.list row")}
  rows=append(rows,repoRow{strings.TrimSpace(parts[0]),strings.TrimSpace(parts[1]),strings.TrimSpace(parts[2]),strings.TrimSpace(parts[3])})
 }
 return rows,nil
}
func run(cwd string, name string, args ...string) (string,error) {
 cmd:=exec.Command(name,args...)
 cmd.Dir=cwd
 cmd.Env=append(os.Environ(),"GIT_TERMINAL_PROMPT=0")
 out,err:=cmd.CombinedOutput()
 return strings.TrimSpace(string(out)),err
}
func git(repo string,args ...string)(string,error){return run(repo,"git",args...)}
func profileRoot() (string,error){
 cwd,err:=os.Getwd();if err!=nil{return "",err}
 for p:=cwd;;p=filepath.Dir(p){
  if _,err:=os.Stat(filepath.Join(p,"config","checks.json"));err==nil {
   if _,err:=os.Stat(filepath.Join(p,"config","repos.list"));err==nil{return p,nil}
  }
  if filepath.Dir(p)==p {break}
 }
 return "",errors.New("run within the registered tools/flowdev-dcm profile")
}
func registration(root string)(repoRow,string,error){
 b,err:=os.ReadFile(filepath.Join(root,"config","repos.list"));if err!=nil{return repoRow{},"",err}
 rows,err:=parseRepoRows(b);if err!=nil{return repoRow{},"",err}
 ws:=os.Getenv("FLOWDEV_WORKSPACE_ROOT")
 if ws=="" {ws=filepath.Clean(filepath.Join(root,"..","..",".."))}
 abs,err:=filepath.Abs(ws);if err!=nil{return repoRow{},"",err}
 for _,r:=range rows{
  if r.app!=appName {continue}
  if r.directory=="" || filepath.Base(r.directory)!=r.directory || r.directory=="." || r.directory==".." {return repoRow{},"",errors.New("unsafe repository directory")}
  if r.branch=="" || r.origin==""{return repoRow{},"",errors.New("incomplete repository registration")}
  return r,filepath.Join(abs,r.directory),nil
 }
 return repoRow{},"",errors.New("app is not registered")
}
func identity(path string, spec repoRow, clean bool) (string,error){
 abs,err:=filepath.Abs(path);if err!=nil{return "",err}
 top,err:=git(path,"rev-parse","--show-toplevel");if err!=nil{return "",fmt.Errorf("not a repository: %s",err)}
 topAbs,err:=filepath.Abs(top);if err!=nil{return "",err}
 if !strings.EqualFold(filepath.Clean(abs),filepath.Clean(topAbs)){return "",errors.New("candidate is not repository root")}
 gotOrigin,err:=git(path,"remote","get-url","origin");if err!=nil{return "",err}
 if !strings.EqualFold(strings.TrimSuffix(gotOrigin,".git"),strings.TrimSuffix(spec.origin,".git")){return "",fmt.Errorf("unexpected origin: %s",gotOrigin)}
 gotBranch,err:=git(path,"symbolic-ref","--short","HEAD");if err!=nil{return "",err}
 if gotBranch!=spec.branch{return "",fmt.Errorf("unexpected branch: %s",gotBranch)}
 status,err:=git(path,"status","--porcelain","--untracked-files=all");if err!=nil{return "",err}
 if clean && status!=""{return "",errors.New("candidate has uncommitted or untracked files")}
 return git(path,"rev-parse","HEAD")
}
func update(path string, spec repoRow)error{
 if _,err:=os.Stat(path);os.IsNotExist(err){return errors.New("candidate clone absent: initialize the registered checkout first")}
 if _,err:=identity(path,spec,true);err!=nil{return err}
 if out,err:=git(path,"fetch","--no-tags","origin",spec.branch);err!=nil{return fmt.Errorf("git fetch: %v: %s",err,out)}
 upstream:="origin/"+spec.branch
 if out,err:=git(path,"merge-base","--is-ancestor","HEAD",upstream);err!=nil{return fmt.Errorf("cannot fast-forward: %v %s",err,out)}
 if out,err:=git(path,"merge","--ff-only",upstream);err!=nil{return fmt.Errorf("fast-forward failed: %v %s",err,out)}
 sha,err:=identity(path,spec,true);if err!=nil{return err}
 fmt.Println("[PASS] update",appName,sha)
 return nil
}
func test(path,profile,selector string,spec repoRow)error{
 sha,err:=identity(path,spec,true);if err!=nil{return err}
 b,err:=os.ReadFile(filepath.Join(profile,"config","checks.json"));if err!=nil{return err}
 var cfg registry
 if err:=json.Unmarshal(b,&cfg);err!=nil{return err}
 app,ok:=cfg.Apps[appName];if !ok{return errors.New("missing registered application")}
 argv,ok:=app.Tests[selector]
 if !ok || len(argv)==0{return fmt.Errorf("unregistered test selector: %s",selector)}
 if strings.ContainsAny(argv[0],string([]byte{47,92,58})) {return errors.New("test executable must be a registered command name")}
 // Avoid arbitrary shell execution: launch the registered executable directly.
 cmd:=exec.Command(argv[0],argv[1:]...)
 cmd.Dir=path;cmd.Stdin=os.Stdin;cmd.Stdout=os.Stdout;cmd.Stderr=os.Stderr
 cmd.Env=append(os.Environ(),"FLOWDEV_SOURCE_COMMIT="+sha)
 fmt.Println("[RUN] test",appName,selector,"commit="+sha)
 if err:=cmd.Run();err!=nil{return fmt.Errorf("registered test failed: %w",err)}
 after,err:=identity(path,spec,true);if err!=nil{return err}
 if after!=sha{return errors.New("source commit changed during test")}
 fmt.Println("[PASS] test",appName,selector,"commit="+sha)
 return nil
}
func main(){
 if len(os.Args)<3 {fmt.Fprintln(os.Stderr,"usage: flowdev-test update dcm-quicksort | flowdev-test test dcm-quicksort <registered-selector>");os.Exit(2)}
 profile,err:=profileRoot();if err!=nil{fatal(err)}
 spec,path,err:=registration(profile);if err!=nil{fatal(err)}
 if os.Args[2]!=appName{fatal(errors.New("wrong app"))}
 switch os.Args[1]{
 case "update":
  if len(os.Args)!=3 {fatal(errors.New("update takes exactly one app"))}
  err=update(path,spec)
 case "test":
  if len(os.Args)!=4{fatal(errors.New("test requires exactly one selector"))}
  err=test(path,profile,os.Args[3],spec)
 default: err=errors.New("unsupported command")
 }
 if err!=nil{fatal(err)}
}
func fatal(err error){fmt.Fprintln(os.Stderr,"[FAIL]",err);os.Exit(1)}
