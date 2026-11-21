package main

import (
 "path/filepath"
 "strings"
)

// queueSelectionAfterRefresh maps selected file identities, not row indexes,
// onto the newly sorted/filtered queue. Hidden, moved or deleted files are
// intentionally excluded from the visible selection.
func queueSelectionAfterRefresh(visible []string, selected []string, active string) ([]int, int) {
 if len(visible)==0 {return nil,-1}
 selectedKeys:=make(map[string]bool,len(selected))
 for _,path:=range selected {if strings.TrimSpace(path)!="" {selectedKeys[queueFileKey(path)]=true}}
 indexes:=make([]int,0,len(selected))
 activeIndex:=-1
 activeKey:=""
 if strings.TrimSpace(active)!="" {activeKey=queueFileKey(active)}
 for i,path:=range visible {
  key:=queueFileKey(path)
  if key==activeKey {activeIndex=i}
  if selectedKeys[key] {indexes=append(indexes,i)}
 }
 if len(indexes)>0 {
  for _,idx:=range indexes {if idx==activeIndex{return indexes,activeIndex}}
  // Preserve a surviving selected file as the active preview when the old
  // active file is no longer visible, instead of previewing an unselected row.
  return indexes,indexes[0]
 }
 if activeIndex>=0 {return []int{activeIndex},activeIndex}
 return []int{0},0
}

func queueFileKey(path string) string {
 return strings.ToLower(filepath.Clean(path))
}
