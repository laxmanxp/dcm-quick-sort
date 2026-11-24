package main

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
)

const explorerBatchSchema = 1
const explorerBatchMaxFiles = 256
const explorerBatchMaxBytes = 1024 * 1024

// The Explorer helper writes this request as UTF-8 JSON. No files move until
// the GUI has shown its confirmation. AliasName resolves against the current
// destination favorites; paths in the JSON cannot silently choose a target.
type explorerBatchRequest struct {
 Schema int `json:"schema"`
 AliasName string `json:"alias_name"`
 Files []string `json:"files"`
}

func parseExplorerBatchRequest(data []byte) (explorerBatchRequest,error) {
 var req explorerBatchRequest
 if len(data)==0 || len(data)>explorerBatchMaxBytes{return req,errors.New("invalid Explorer request size")}
 if err:=json.Unmarshal(data,&req);err!=nil{return req,fmt.Errorf("invalid Explorer request: %w",err)}
 if req.Schema!=explorerBatchSchema{return req,errors.New("unsupported Explorer request version")}
 req.AliasName=strings.TrimSpace(req.AliasName)
 if req.AliasName=="" || len(req.Files)==0 || len(req.Files)>explorerBatchMaxFiles {
  return req,errors.New("Explorer batch must include one named destination and 1-256 files")
 }
 seen:=make(map[string]bool,len(req.Files))
 files:=make([]string,0,len(req.Files))
 for _,raw:=range req.Files {
  if strings.TrimSpace(raw)!=raw || !filepath.IsAbs(raw) {return req,fmt.Errorf("Explorer file must use an absolute path")}
  clean:=filepath.Clean(raw)
  if clean=="." || clean==string(filepath.Separator) {return req,errors.New("invalid Explorer file")}
  key:=strings.ToLower(clean)
  if seen[key]{continue}
  seen[key]=true
  files=append(files,clean)
 }
 req.Files=files
 return req,nil
}
func resolveExplorerBatchAlias(req explorerBatchRequest,aliases []DestinationAlias)(string,error){
 for _,alias:=range aliases {
  if strings.EqualFold(strings.TrimSpace(alias.Name),req.AliasName){
   return resolveExistingDestinationFolder(alias.Path)
  }
 }
 return "",fmt.Errorf("destination favorite %q was not found; no files moved",req.AliasName)
}
func validateExplorerBatchFiles(files []string)error{
 for _,path:=range files{
  st,err:=os.Lstat(path)
  if err!=nil{return fmt.Errorf("selected file unavailable: %s: %w",filepath.Base(path),err)}
  if st.Mode()&os.ModeSymlink!=0 || !st.Mode().IsRegular(){
   return fmt.Errorf("Explorer batch accepts ordinary files only: %s",filepath.Base(path))
  }
 }
 return nil
}

 // isTrustedExplorerRequestPath refuses request manifests outside the dedicated
 // per-user request directory. This must be evaluated before any file read.
func isTrustedExplorerRequestPath(requestPath, trustedDir string) bool {
 if requestPath=="" || trustedDir=="" || !filepath.IsAbs(requestPath) || !filepath.IsAbs(trustedDir) {
  return false
 }
 requestPath=filepath.Clean(requestPath)
 trustedDir=filepath.Clean(trustedDir)
 filename:=filepath.Base(requestPath)
 if filepath.Clean(filepath.Dir(requestPath))!=trustedDir || !strings.HasPrefix(filename,"batch-") || filepath.Ext(filename)!=".json" {
  return false
 }
 return len(filename)>len("batch-.json")
}
