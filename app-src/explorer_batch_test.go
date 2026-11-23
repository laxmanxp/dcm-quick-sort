package main

import (
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
)
func TestExplorerBatchRequest(t *testing.T){
 home:=t.TempDir()
 a:=filepath.Join(home,"one file.pdf")
 b:=filepath.Join(home,"two.png")
 data,_:=json.Marshal(explorerBatchRequest{Schema:1,AliasName:" Accounts ",Files:[]string{a,b,a}})
 req,err:=parseExplorerBatchRequest(data)
 if err!=nil{t.Fatal(err)}
 if req.AliasName!="Accounts" || len(req.Files)!=2{t.Fatalf("bad request: %+v",req)}
 dest:=filepath.Join(home,"Accounts")
 if err:=os.Mkdir(dest,0700);err!=nil{t.Fatal(err)}
 got,err:=resolveExplorerBatchAlias(req,[]DestinationAlias{{Name:"accounts",Path:dest}})
 if err!=nil || got!=dest{t.Fatalf("destination: %q %v",got,err)}
}
func TestExplorerBatchRejectsBadInputs(t *testing.T) {
 dir:=t.TempDir()
 file:=filepath.Join(dir,"file.txt")
 for _,req:=range []explorerBatchRequest{
  {Schema:0,AliasName:"Accounts",Files:[]string{file}},
  {Schema:1,AliasName:"",Files:[]string{file}},
  {Schema:1,AliasName:"Accounts"},
  {Schema:1,AliasName:"Accounts",Files:[]string{"relative.txt"}},
 }{
  data,_:=json.Marshal(req)
  if _,err:=parseExplorerBatchRequest(data);err==nil{t.Fatalf("accepted %+v",req)}
 }
}
func TestExplorerBatchValidatesFilesWithoutMoving(t *testing.T){
 root:=t.TempDir()
 file:=filepath.Join(root,"a.txt")
 if err:=os.WriteFile(file,[]byte("unchanged"),0600);err!=nil{t.Fatal(err)}
 if err:=validateExplorerBatchFiles([]string{file});err!=nil{t.Fatal(err)}
 if err:=validateExplorerBatchFiles([]string{root});err==nil{t.Fatal("directory accepted")}
 data,err:=os.ReadFile(file)
 if err!=nil||string(data)!="unchanged"{t.Fatal("fixture changed")}
}
