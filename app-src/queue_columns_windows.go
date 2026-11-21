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
}

func queueDetailsForPaths(paths []string) []queueRowDetail {
 out:=make([]queueRowDetail,len(paths))
 for i,path:=range paths {
  name:=filepath.Base(path)
  extension:=strings.TrimPrefix(strings.ToUpper(filepath.Ext(path)),".")
  created:="—"
  if info,err:=os.Stat(path);err==nil {
   t:=fileCreatedTime(path,info)
   if !t.IsZero() {created=t.In(time.Local).Format("02 Jan 2006")}
  }
  out[i]=queueRowDetail{Name:name,Ext:extension,Created:created}
 }
 return out
}

func queueColumnWidths(width int32) (int32,int32,int32) {
 if width<1 {return 0,0,0}
 ext:=width*18/100
 if ext<55 {ext=55}
 created:=width*30/100
 if created<105 {created=105}
 name:=width-ext-created
 if name<80 {
  name=80
  created=width-name-ext
 }
 return name,ext,created
}

func queueSortIsDescending(sortKey InboxSort,reverse bool) bool {
 return (sortKey==InboxSortDateCreated || sortKey==InboxSortDateCreatedTypeName)!=reverse
}

func updateQueueHeaderCaptions() {
 active:=selectedInboxSort()
 captions:=[]struct{hwnd uintptr;label string;key InboxSort}{
  {app.queueNameHeader,"Name",InboxSortName},
  {app.queueExtHeader,"Extension",InboxSortFileType},
  {app.queueCreatedHeader,"Created",InboxSortDateCreated},
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
 current:=selectedInboxSort()
 if current==key {app.queueSortReverse=!app.queueSortReverse}else{app.queueSortReverse=false}
 index:=0
 for i,value:=range inboxSortOrder {if value==key {index=i;break}}
 procSendMessageW.Call(app.filterSortCombo,CB_SETCURSEL,uintptr(index),0)
 updateQueueHeaderCaptions()
 reloadFilesWithFocus(false)
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
 nameW,extW,_:=queueColumnWidths(width)
 fields:=[]struct{txt string;left,right int32}{
  {row.Name,dis.RcItem.Left+5,dis.RcItem.Left+nameW-4},
  {row.Ext,dis.RcItem.Left+nameW+4,dis.RcItem.Left+nameW+extW-4},
  {row.Created,dis.RcItem.Left+nameW+extW+4,dis.RcItem.Right-5},
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
