//go:build windows

package main

import ("unsafe"; "syscall")

// SysTabControl32 is a native Windows common control. The existing Queue and
// History child windows remain independent; the tab changes only visibility.
const (
 nativeTabControlID = 1070
 tcmInsertItemW = 0x133E
 tcmGetCurSel = 0x130B
 tcmSetCurSel = 0x130C
 tcifText = 0x0001
 tcnSelChange = -551 // TCN_FIRST (-550) - 1
 iccTabClasses = 0x00000008
)

type nativeCommonControlsInit struct {
 Size uint32
 Classes uint32
}

type nativeTabItem struct {
 Mask uint32
 State uint32
 StateMask uint32
 Text *uint16
 TextMax int32
 Image int32
 Param uintptr
}

type nativeNotifyHeader struct {
 From uintptr
 ID uintptr
 Code int32
}

func createNativeQueueHistoryTabs(parent uintptr) uintptr {
 commonControls := syscall.NewLazyDLL("comctl32.dll").NewProc("InitCommonControlsEx")
 init := nativeCommonControlsInit{Size:uint32(unsafe.Sizeof(nativeCommonControlsInit{})),Classes:iccTabClasses}
 commonControls.Call(uintptr(unsafe.Pointer(&init)))
 hwnd := createControl("SysTabControl32","",WS_CHILD|WS_VISIBLE|WS_TABSTOP,20,116,350,27,parent,nativeTabControlID)
 if hwnd==0 {return 0}
 for _,title := range []string{"Queue","History"} {
  label:=wstr(title)
  item:=nativeTabItem{Mask:tcifText,Text:label}
  procSendMessageW.Call(hwnd,tcmInsertItemW,uintptr(lenTitleIndex(title)),uintptr(unsafe.Pointer(&item)))
 }
 procSendMessageW.Call(hwnd,tcmSetCurSel,0,0)
 return hwnd
}

// Both native tab labels have stable indices; avoid deriving from window text.
func lenTitleIndex(title string) int {
 if title=="History" {return 1}
 return 0
}

func syncNativeTabSelection() {
 if app.queueTabControl==0 {return}
 index:=uintptr(0)
 if app.showHistoryTab {index=1}
 selected,_,_:=procSendMessageW.Call(app.queueTabControl,tcmGetCurSel,0,0)
 if selected!=index {procSendMessageW.Call(app.queueTabControl,tcmSetCurSel,index,0)}
}

func handleQueueNativeTabNotification(lParam uintptr) bool {
 hdr:=(*nativeNotifyHeader)(unsafe.Pointer(lParam))
 if hdr.From!=app.queueTabControl || hdr.Code!=tcnSelChange {return false}
 selected,_,_:=procSendMessageW.Call(app.queueTabControl,tcmGetCurSel,0,0)
 if selected>1 {return true}
 app.showHistoryTab=selected==1
 layoutUI(app.hwnd)
 if app.showHistoryTab {procSetFocus.Call(app.processedList)} else {procSetFocus.Call(app.fileList)}
 return true
}
