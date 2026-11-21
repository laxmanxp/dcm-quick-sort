package main

import (
 "reflect"
 "testing"
)

func TestQueueSelectionAfterRefresh(t *testing.T) {
 a:=`C:\Inbox\a.pdf`
 b:=`C:\Inbox\b.docx`
 c:=`C:\Inbox\c.xlsx`
 d:=`D:\Other\d.pdf`
 for _,tc:=range []struct{
  name string
  visible,selected []string
  active string
  wantSelected []int
  wantActive int
 }{
  {"multi-selection survives sorting",[]string{c,a,b},[]string{a,c},c,[]int{0,1},0},
  {"filter leaves active selected",[]string{b,c},[]string{a,c},c,[]int{1},1},
  {"filter removes active but preserves other selection",[]string{b,c},[]string{a,c},a,[]int{1},1},
  {"filter hides all selected",[]string{b},[]string{a,c},a,[]int{0},0},
  {"source folder changes",[]string{d},[]string{a,b},a,[]int{0},0},
  {"reload retains active without explicit selection",[]string{b,a},nil,a,[]int{1},1},
  {"empty filtered queue",nil,[]string{a},a,nil,-1},
  {"Windows case insensitive paths",[]string{`c:\inbox\A.PDF`,b},[]string{a},a,[]int{0},0},
 } {
  t.Run(tc.name,func(t *testing.T){
   got,active:=queueSelectionAfterRefresh(tc.visible,tc.selected,tc.active)
   if !reflect.DeepEqual(got,tc.wantSelected)||active!=tc.wantActive {
    t.Fatalf("got selected %v active %d, want %v active %d",got,active,tc.wantSelected,tc.wantActive)
   }
  })
 }
}
