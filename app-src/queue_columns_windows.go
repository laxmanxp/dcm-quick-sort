//go:build windows

package main

import (
 "os"
 "path/filepath"
 "strings"
 "time"
 "unsafe"
)

// Queue rows remain native LISTBOX entries, including their original filenames
// and selection semantics. Drawing is strictly a presentation layer.
type queueRowDetail struct {
 Name string
 Ext string
 Created string
 Updated string
}

func queueDetailsForPaths(paths []string) []queueRowDetail {
 out:=make([]queueRowDetail,len(paths))
 for i,path:=range paths {
  name:=filepath.Base(path)
  extension:=strings.TrimPrefix(strings.ToUpper(filepath.Ext(path)),".")
  created:="—"
  updated:="—"
  if info,err:=os.Stat(path);err==nil {
   t:=fileCreatedTime(path,info)
   if !t.IsZero() {created=t.In(time.Local).Format("02 Jan 06")}
   if !info.ModTime().IsZero() {updated=info.ModTime().In(time.Local).Format("02 Jan 06")}
  }
  out[i]=queueRowDetail{Name:name,Ext:extension,Created:created,Updated:updated}
 }
 return out
}

func queueColumnWidths(width int32) (int32,int32,int32,int32) {
 if width<1 {return 0,0,0,0}
 ext:=width*16/100
 if ext<55 {ext=55}
 created:=width*23/100
 if created<79 {created=79}
 updated:=width*23/100
 if updated<79 {updated=79}
 name:=width-ext-created-updated
 if name<80 {
  name=80
  updated=width-name-ext-created
 }
 return name,ext,created,updated
}

func queueSortIsDescending(sortKey InboxSort,reverse bool) bool {
 return (sortKey==InboxSortDateCreated || sortKey==InboxSortDateUpdated || sortKey==InboxSortDateCreatedTypeName)!=reverse
}

func updateQueueHeaderCaptions() {
 active:=selectedInboxSort()
 captions:=[]struct{hwnd uintptr;label string;key InboxSort}{
  {app.queueNameHeader,"Name",InboxSortName},
  {app.queueExtHeader,"Extension",InboxSortFileType},
  {app.queueCreatedHeader,"Created",InboxSortDateCreated},
  {app.queueUpdatedHeader,"Updated",InboxSortDateUpdated},
 }
 for _,item:=range captions {
  if item.hwnd==0 {continue}
  caption:=item.label
  if active==item.key {
   if queueSortIsDescending(active,app.queueSortReverse){caption+=" ↓"}else{caption+=" ↑"}
  }
  setText(item.hwnd,caption)
 }
}

func activateQueueColumnSort(key InboxSort) {
 if app.filterSortCombo==0 {return}
 // Preserve the exact visible selection (including multi-select) across the
 // reload. The normal reload keeps the active filename but selects one row.
 selected:=selectedSourceFiles()
 current:=selectedInboxSort()
 if current==key {app.queueSortReverse=!app.queueSortReverse}else{app.queueSortReverse=false}
 index:=0
 for i,value:=range inboxSortOrder {if value==key {index=i;break}}
 procSendMessageW.Call(app.filterSortCombo,CB_SETCURSEL,uintptr(index),0)
 updateQueueHeaderCaptions()
 reloadFilesWithFocus(false)
 if len(selected)>0 {
  selectedPaths:=make(map[string]bool,len(selected))
  for _,path:=range selected {selectedPaths[strings.ToLower(filepath.Clean(path))]=true}
  app.updatingFileList=true
  procSendMessageW.Call(app.fileList,LB_SETSEL,0,^uintptr(0))
  for i,path:=range app.files {
   if selectedPaths[strings.ToLower(filepath.Clean(path))] {
    procSendMessageW.Call(app.fileList,LB_SETSEL,1,uintptr(i))
   }
  }
  procSendMessageW.Call(app.fileList,LB_SETCARETINDEX,uintptr(app.index),0)
  app.updatingFileList=false
  // Count and selected-file preview reflect the restored selection.
  showCurrentView(false,false)
 }
 procSetFocus.Call(app.fileList)
}

func drawQueueRow(dis *DRAWITEMSTRUCT) {
 if dis==nil || dis.HDC==0 || dis.ItemID==^uint32(0) {return}
 index:=int(dis.ItemID)
 if index<0 || index>=len(app.queueRowDetails) {return}
 row:=app.queueRowDetails[index]
 selected:=dis.ItemState&ODS_SELECTED!=0
 bg:=rgb(255,255,255)
 fg:=colorInk
 if selected {bg=rgb(217,232,252);fg=rgb(20,48,88)}
 if index%2==1 && !selected {bg=rgb(248,250,253)}
 brush,_,_:=procCreateSolidBrush.Call(bg)
 procFillRect.Call(dis.HDC,uintptr(unsafe.Pointer(&dis.RcItem)),brush)
 procDeleteObject.Call(brush)
 procSetBkMode.Call(dis.HDC,TRANSPARENT)
 procSetTextColor.Call(dis.HDC,fg)
 oldFont:=uintptr(0)
 if app.uiFont!=0 {oldFont,_,_=procSelectObject.Call(dis.HDC,app.uiFont)}
 width:=dis.RcItem.Right-dis.RcItem.Left
 nameW,extW,createdW,_:=queueColumnWidths(width)
 fields:=[]struct{txt string;left,right int32}{
  {row.Name,dis.RcItem.Left+5,dis.RcItem.Left+nameW-4},
  {row.Ext,dis.RcItem.Left+nameW+4,dis.RcItem.Left+nameW+extW-4},
  {row.Created,dis.RcItem.Left+nameW+extW+4,dis.RcItem.Left+nameW+extW+createdW-4},
  {row.Updated,dis.RcItem.Left+nameW+extW+createdW+4,dis.RcItem.Right-5},
 }
 for _,field:=range fields {
  if field.right<=field.left {continue}
  rect:=RECT{Left:field.left,Top:dis.RcItem.Top,Right:field.right,Bottom:dis.RcItem.Bottom}
  ptr:=wstr(field.txt)
  const flags=uintptr(0x00000004|0x00000020|0x00008000) // DT_VCENTER | DT_SINGLELINE | DT_END_ELLIPSIS
  procDrawTextW.Call(dis.HDC,uintptr(unsafe.Pointer(ptr)),^uintptr(0),uintptr(unsafe.Pointer(&rect)),flags)
 }
 if oldFont!=0 {procSelectObject.Call(dis.HDC,oldFont)}
 if dis.ItemState&ODS_FOCUS!=0 {procDrawFocusRect.Call(dis.HDC,uintptr(unsafe.Pointer(&dis.RcItem)))}
}
