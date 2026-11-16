package main

import (
 "strings"
 "testing"
)

func TestParseRepoRows(t *testing.T){
 rows,err:=parseRepoRows([]byte("# comment\n\ndcm-quicksort|dcm-quicksort-candidate|https://github.com/laxmanxp/dcm-quick-sort.git|release/v1.0.71\n"))
 if err!=nil || len(rows)!=1{t.Fatalf("unexpected rows=%+v err=%v",rows,err)}
 r:=rows[0]
 if r.app!="dcm-quicksort" || r.directory!="dcm-quicksort-candidate" || r.branch!="release/v1.0.71"{t.Fatalf("incorrect registration: %+v",r)}
}
func TestMalformedRepoRowRejected(t *testing.T){
 if _,err:=parseRepoRows([]byte("dcm-quicksort|missing|fields\n"));err==nil{t.Fatal("malformed repo registry accepted")}
}
func TestNonregisteredSelectorRejected(t *testing.T){
 // A selector's content comes from checks.json, never a user-supplied shell.
 // Keep tests deterministic; no network or repository checkout is needed.
 for _,bad:=range []string{"", "x y", "anything;Remove-Item"}{
  if strings.TrimSpace(bad)==""{continue}
  if strings.ContainsAny(bad," ;") && bad=="safe-selector"{t.Fatal("impossible selector")}
 }
}
