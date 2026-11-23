package main

import (
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
)

const maxFiles=256
type request struct {
 Schema int `json:"schema"`
 AliasName string `json:"alias_name"`
 Files []string `json:"files"`
}
func prepare(alias string,files []string)(request,error){
 alias=strings.TrimSpace(alias)
 if alias==""||len(files)==0||len(files)>maxFiles{return request{},errors.New("provide a saved destination name and 1-256 files")}
 result:=request{Schema:1,AliasName:alias}
 seen:=map[string]bool{}
 for _,v:=range files{
  if strings.TrimSpace(v)!=v || !filepath.IsAbs(v){return request{},fmt.Errorf("file must have an absolute path: %q",v)}
  clean:=filepath.Clean(v)
  key:=strings.ToLower(clean)
  if !seen[key]{result.Files=append(result.Files,clean);seen[key]=true}
 }
 return result,nil
}
func requestDir()(string,error){
 base:=strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
 if base=="" {
  var err error
  base,err=os.UserConfigDir()
  if err!=nil{return "",err}
 }
 return filepath.Join(base,"DCM Quick Sort","explorer-requests"),nil
}
func writeRequest(req request)(string,error){
 dir,err:=requestDir()
 if err!=nil{return "",err}
 if err:=os.MkdirAll(dir,0700);err!=nil{return "",err}
 file,err:=os.CreateTemp(dir,"batch-*.json")
 if err!=nil{return "",err}
 name:=file.Name()
 data,err:=json.Marshal(req)
 if err==nil {_,err=file.Write(data)}
 closeErr:=file.Close()
 if err==nil {err=closeErr}
 if err!=nil {_=os.Remove(name);return "",err}
 return name,nil
}
func main(){
 app:=flag.String("app","","absolute path of DCM QuickSort.exe")
 alias:=flag.String("alias","","saved destination favorite name")
 flag.Parse()
 if !filepath.IsAbs(*app){fmt.Fprintln(os.Stderr,"absolute -app path required");os.Exit(2)}
 req,err:=prepare(*alias,flag.Args())
 if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(2)}
 manifest,err:=writeRequest(req)
 if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
 cmd:=exec.Command(*app,"--explorer-send-request",manifest)
 if err:=cmd.Start();err!=nil{_ = os.Remove(manifest);fmt.Fprintln(os.Stderr,err);os.Exit(1)}
 // The GUI owns the request once launched; it always asks for confirmation.
}
