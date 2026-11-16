package main

import "testing"

func TestParseRepoRows(t *testing.T) {
 rows,err:=parseRepoRows([]byte("# comment\n\ndcm-quicksort|dcm-quicksort-candidate|https://github.com/laxmanxp/dcm-quick-sort.git|release/v1.0.71\n"))
 if err!=nil || len(rows)!=1 {t.Fatalf("unexpected rows=%+v err=%v",rows,err)}
 r:=rows[0]
 if r.app!="dcm-quicksort" || r.directory!="dcm-quicksort-candidate" || r.branch!="release/v1.0.71" {t.Fatalf("incorrect registration: %+v",r)}
}
func TestMalformedRepoRowRejected(t *testing.T) {
 if _,err:=parseRepoRows([]byte("dcm-quicksort|missing|fields\n"));err==nil {t.Fatal("malformed repo registry accepted")}
}
func TestCommentsAndBlankLinesIgnored(t *testing.T) {
 rows,err:=parseRepoRows([]byte(" \n# registry metadata\n\n"))
 if err!=nil || len(rows)!=0 {t.Fatalf("unexpected rows: %+v err=%v",rows,err)}
}
