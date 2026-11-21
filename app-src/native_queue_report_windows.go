//go:build windows

package main

import (
 "os"
 "strings"
 "fmt"
 "unsafe"
)

// Native report-view queue backed by the established listbox selection adapter.
// The native control is not an Explorer-style file manager.
const (
 nativeQueueReportID = 1071
 wmNativeQueueSelection = 0x8020
 lvmFirst = 0x1000
 lvmInsertColumnW = lvmFirst+97
 lvmInsertItemW = lvmFirst+77
 lvmSetItemTextW = lvmFirst+116
 lvmDeleteAllItems = lvmFirst+9
 lvmSetExtendedStyle = lvmFirst+54
 lvmSetItemState = lvmFirst+43
 lvmGetNextItem = lvmFirst+12
 lvmGetItemCount = lvmFirst+4
 lvifText = 0x0001
 lvcfFmt = 0x0001
 lvcfWidth = 0x0002
 lvcfText = 0x0004
 lvcfSubitem = 0x0008
 lvisFocused = 0x0001
 lvisSelected = 0x0002
 lvniSelected = 0x0002
 lvnFirst = -100
 lvnItemChanged = lvnFirst-1
 lvnColumnClick = lvnFirst-8
)

type nativeLVColumn struct {
 Mask uint32
 Format int32
 Width int32
 Text *uint16
 TextMax int32
 SubItem int32
 Image int32
 Order int32
 MinWidth int32
 DefaultWidth int32
 IdealWidth int32
}
type nativeLVItem struct {
 Mask uint32
 Item int32
 SubItem int32
 State uint32
 StateMask uint32
 Text *uint16
 TextMax int32
 Image int32
 Param uintptr
 Indent int32
 GroupID int32
 Columns uint32
 ColumnIndices *uint32
 ColumnFormats *int32
 Group int32
}
type nativeLVNotify struct {
 Header nativeNotifyHeader
 Item int32
 SubItem int32
 NewState uint32
 OldState uint32
 Changed uint32
 Action POINT
 Param uintptr
}

// Development-only trace, enabled only by an explicit test environment variable.
func nativeQueueTrace(phase string) {
 path:=os.Getenv("DCM_NATIVE_QUEUE_TRACE")
 if path=="" {return}
 f,err:=os.OpenFile(path,os.O_APPEND|os.O_CREATE|os.O_WRONLY,0600)
 if err!=nil {return}
 defer f.Close()
 fmt.Fprintln(f,phase)
}

func nativeQueueEnabled() bool {
 return strings.TrimSpace(os.Getenv("DCM_NATIVE_QUEUE_LISTVIEW"))=="1"
}

func createNativeQueueReport(parent uintptr) uintptr {
 if !nativeQueueEnabled(){return 0}
 hwnd:=createControl("SysListView32","",WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|0x0001|0x0008,20,148,350,362,parent,nativeQueueReportID)
 if hwnd==0{return 0}
 procSendMessageW.Call(hwnd,lvmSetExtendedStyle,0,0x00000020|0x00010000) // full row, double-buffer
 for i,name:=range []string{"Name","Extension","Created","Updated"}{
  label:=wstr(name)
  col:=nativeLVColumn{Mask:lvcfText|lvcfWidth|lvcfSubitem,Text:label,Width:100,SubItem:int32(i)}
  procSendMessageW.Call(hwnd,lvmInsertColumnW,uintptr(i),uintptr(unsafe.Pointer(&col)))
 }
 return hwnd
}

func nativeQueueResizeColumns(width int32) {
 if app.nativeFileList==0 {return}
 dims:=[4]int32{}
 dims[0],dims[1],dims[2],dims[3]=queueColumnWidths(width)
 for i,value:=range dims{
  procSendMessageW.Call(app.nativeFileList,lvmFirst+30,uintptr(i),uintptr(value)) // LVM_SETCOLUMNWIDTH
 }
}

func nativeQueueRefresh() {
 if app.nativeFileList==0 {return}
 app.nativeQueueSyncing=true
 procSendMessageW.Call(app.nativeFileList,lvmDeleteAllItems,0,0)
 for i,row:=range app.queueRowDetails {
  text:=wstr(row.Name)
  item:=nativeLVItem{Mask:lvifText,Item:int32(i),SubItem:0,Text:text}
  procSendMessageW.Call(app.nativeFileList,lvmInsertItemW,0,uintptr(unsafe.Pointer(&item)))
  for column,value:=range []string{row.Ext,row.Created,row.Updated} {
   valueUTF16:=wstr(value)
   item:=nativeLVItem{Item:int32(i),SubItem:int32(column+1),Text:valueUTF16}
   procSendMessageW.Call(app.nativeFileList,lvmSetItemTextW,uintptr(i),uintptr(unsafe.Pointer(&item)))
  }
 }
 app.nativeQueueSyncing=false
 nativeQueueSelectionFromLegacy()
}

func nativeQueueSelectionFromLegacy() {
 if app.nativeFileList==0 || app.nativeQueueSyncing {return}
 app.nativeQueueSyncing=true
 defer func(){app.nativeQueueSyncing=false}()
 // Clear and restore exact selection, including Ctrl/Shift multi-select.
 clear:=nativeLVItem{State:0,StateMask:lvisSelected|lvisFocused}
 procSendMessageW.Call(app.nativeFileList,lvmSetItemState,^uintptr(0),uintptr(unsafe.Pointer(&clear)))
 for _,idx:=range selectedSourceIndexes() {
  state:=nativeLVItem{State:lvisSelected,StateMask:lvisSelected}
  if int(idx)==app.index {state.State|=lvisFocused;state.StateMask|=lvisFocused}
  procSendMessageW.Call(app.nativeFileList,lvmSetItemState,uintptr(idx),uintptr(unsafe.Pointer(&state)))
 }
}

func nativeQueueSelectionToLegacy() {
 if os.Getenv("DCM_NATIVE_QUEUE_DIAG_NO_SYNC")=="1" {return}
 if app.nativeFileList==0 || app.nativeQueueSyncing || app.updatingFileList{return}
 // Coalesce redundant notifications before changing the legacy adapter or
 // restarting the active PDFium preview. ListView emits multiple state
 // notifications for a single ordinary mouse/keyboard selection.
 nativeSelected:=make([]int32,0)
 for cursor:=int32(-1);;{
  next,_,_:=procSendMessageW.Call(app.nativeFileList,lvmGetNextItem,uintptr(cursor),lvniSelected)
  if next==^uintptr(0) || next>=uintptr(len(app.files)){break}
  cursor=int32(next)
  nativeSelected=append(nativeSelected,cursor)
 }
 previous:=selectedSourceIndexes()
 unchanged:=len(previous)==len(nativeSelected)
 if unchanged {
  for i,value:=range nativeSelected {if previous[i]!=value {unchanged=false;break}}
 }
 selectedIndex:=-1
 if len(nativeSelected)>0 {selectedIndex=int(nativeSelected[len(nativeSelected)-1])}
 if unchanged && (selectedIndex<0 || selectedIndex==app.index) {return}
 app.nativeQueueSyncing=true
 defer func(){app.nativeQueueSyncing=false}()
 app.updatingFileList=true
 defer func(){app.updatingFileList=false}()
 procSendMessageW.Call(app.fileList,LB_SETSEL,0,^uintptr(0))
 for _,idx:=range nativeSelected {
  procSendMessageW.Call(app.fileList,LB_SETSEL,1,uintptr(idx))
 }
 if selectedIndex>=0 {
  app.index=selectedIndex
  procSendMessageW.Call(app.fileList,LB_SETCARETINDEX,uintptr(app.index),0)
  nativeQueueTrace(fmt.Sprintf("sync-selected index=%d count=%d",selectedIndex,len(nativeSelected)))
  showCurrentView(false,false)
 }
}

func nativeQueueNotify(lParam uintptr) bool {
 if app.nativeFileList==0 || lParam==0{return false}
 hdr:=(*nativeNotifyHeader)(unsafe.Pointer(lParam))
 if hdr.From!=app.nativeFileList{return false}
 nativeQueueTrace(fmt.Sprintf("notify code=%d syncing=%v",hdr.Code,app.nativeQueueSyncing))
 if hdr.Code==lvnColumnClick {
  note:=(*nativeLVNotify)(unsafe.Pointer(lParam))
  keys:=[]InboxSort{InboxSortName,InboxSortFileType,InboxSortDateCreated,InboxSortDateUpdated}
  if note.SubItem>=0 && int(note.SubItem)<len(keys){activateQueueColumnSort(keys[note.SubItem])}
  return true
 }
 if hdr.Code==lvnItemChanged {
  // Never mutate a ListView (or start its preview) inside its own synchronous
  // LVN_ITEMCHANGED dispatch: defer until control notification unwinds.
  if !app.nativeQueueSyncing {procPostMessageW.Call(app.hwnd,wmNativeQueueSelection,0,0)}
  return true
 }
 return false
}
