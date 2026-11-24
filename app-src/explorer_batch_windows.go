//go:build windows

package main

import (
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "unsafe"
)

// Explorer's helper passes a request file, rather than dozens of separately
// interpreted command-line arguments. The GUI consumes it only after UI and
// recovery initialization. A malicious or stale request cannot move files
// without the explicit Windows confirmation.
func consumeExplorerBatchStartup() {
 requestPath:=explorerBatchRequestPath
 explorerBatchRequestPath=""
 if requestPath=="" {return}
 if app.batchMoveActive || app.backgroundMoveBusy {showError("QuickSort is already processing files.");return}
 // Only launcher-owned request manifests are removed. Arbitrary paths
 // supplied through a command line must never be deleted by QuickSort.
 trustedDir:=filepath.Join(settingsDirectory(),"explorer-requests")
 raw,err:=readTrustedExplorerRequest(requestPath,trustedDir)
 if err!=nil {showError("Cannot read Explorer batch request: "+err.Error());return}
 _=os.Remove(requestPath)
 req,err:=parseExplorerBatchRequest(raw)
 if err!=nil {showError(err.Error());return}
 aliases,err:=loadDestinationAliases("")
 if err!=nil {showError("Unable to load saved destinations: "+err.Error());return}
 dest,err:=resolveExplorerBatchAlias(req,aliases)
 if err!=nil {showError(err.Error());return}
 if err=validateExplorerBatchFiles(req.Files);err!=nil {showError(err.Error());return}
 plan,err:=buildBatchMovePlan(req.Files,dest)
 if err!=nil {showError(err.Error());return}
 if len(plan)==0{return}
 var sample strings.Builder
 for i,item:=range plan {
  if i>=8 {fmt.Fprintf(&sample,"\n… and %d more",len(plan)-i);break}
  fmt.Fprintf(&sample,"\n%s",filepath.Base(item.Source))
 }
 msg:=fmt.Sprintf("Move %d selected file(s) from Windows Explorer to:\n\n%s\n\nSelected files:%s\n\nThis will move, not copy. Conflicts are handled one file at a time; successful moves are recorded in History and can be undone subject to safety checks.\n\nContinue?",len(plan),dest,sample.String())
 answer,_,_:=procMessageBoxW.Call(app.hwnd,uintptr(unsafe.Pointer(wstr(msg))),uintptr(unsafe.Pointer(wstr("Explorer Batch Move — DCM QuickSort"))),0x04|0x20)
 if int(answer)!=6 {setText(app.statusLabel,"Explorer batch cancelled — no files moved.");return}
 startApprovedBatchMove(plan,dest)
}
