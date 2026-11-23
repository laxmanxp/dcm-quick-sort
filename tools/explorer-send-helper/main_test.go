package main

import (
 "path/filepath"
 "testing"
)
func TestPrepareExplorerSend(t *testing.T){
 root:=t.TempDir()
 a:=filepath.Join(root,"a file.txt")
 b:=filepath.Join(root,"b.txt")
 req,err:=prepare(" Accounts ",[]string{a,b,a})
 if err!=nil{t.Fatal(err)}
 if req.Schema!=1||req.AliasName!="Accounts"||len(req.Files)!=2{t.Fatalf("bad request: %+v",req)}
}
func TestPrepareRejectsRelativeAndEmpty(t *testing.T){
 for _,tc:=range []struct{alias string;files []string}{
  {"",[]string{filepath.Join(t.TempDir(),"a")}},
  {"Accounts",nil},
  {"Accounts",[]string{"relative.txt"}},
 }{
  if _,err:=prepare(tc.alias,tc.files);err==nil{t.Fatalf("accepted %+v",tc)}
 }
}
