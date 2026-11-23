//go:build windows

// Test-only QuickSort stand-in. It only copies the Explorer request JSON into
// a disposable receipt; it has no file-operation implementation.
package main

import (
 "os"
 "path/filepath"
)
func main(){
 if len(os.Args)!=3 || os.Args[1]!="--explorer-send-request"{os.Exit(2)}
 data,err:=os.ReadFile(os.Args[2])
 if err!=nil{os.Exit(3)}
 root:=os.Getenv("LOCALAPPDATA")
 if root==""{os.Exit(4)}
 if err=os.WriteFile(filepath.Join(root,"shell-lab-receipt.json"),data,0600);err!=nil{os.Exit(5)}
}
