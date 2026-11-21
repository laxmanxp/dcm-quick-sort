//go:build windows

package main

import "testing"

func TestQueueColumnWidths(t *testing.T) {
 for _,width:=range []int32{260,379,442,460} {
  name,ext,created,updated:=queueColumnWidths(width)
  if name+ext+created+updated!=width {
   t.Fatalf("column widths did not fill pane: width=%d sum=%d",width,name+ext+created+updated)
  }
  if name<80 || ext<55 || created<50 || updated<50 {t.Fatalf("unusable queue columns at %d: %d %d %d %d",width,name,ext,created,updated)}
 }
}

func TestQueueColumnSortDirections(t *testing.T) {
 cases:=[]struct{key InboxSort;reverse,descending bool}{
  {InboxSortName,false,false},
  {InboxSortName,true,true},
  {InboxSortFileType,false,false},
  {InboxSortFileType,true,true},
  {InboxSortDateCreated,false,true},
  {InboxSortDateCreated,true,false},
  {InboxSortDateUpdated,false,true},
  {InboxSortDateUpdated,true,false},
  {InboxSortDateCreatedTypeName,false,true},
  {InboxSortDateCreatedTypeName,true,false},
 }
 for _,test:=range cases {
  if got:=queueSortIsDescending(test.key,test.reverse);got!=test.descending {
   t.Errorf("sort %s reverse=%v descending=%v, want %v",test.key,test.reverse,got,test.descending)
  }
 }
}
