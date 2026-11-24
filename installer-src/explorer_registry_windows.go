//go:build windows

package main

import (
 "encoding/binary"
 "errors"
 "fmt"
 "syscall"
)

// The production-facing registry adapter is intentionally read-only until
// disposable Windows GUI qualification is complete. Opening it cannot enable
// the shell extension or change an existing user's Explorer registration.
var errExplorerRegistryWritesDisabled = errors.New("Explorer registry mutations are disabled until isolated GUI acceptance")

const (
 explorerSendHandlerKey = `Software\Classes\*\shellex\ContextMenuHandlers\DCMQuickSortSend`
 explorerSendServerKey = `Software\Classes\CLSID\{8F2DA0C1-7D79-40AA-A030-19645D8C4A91}\InprocServer32`
)

type windowsExplorerRegistryStore struct{}

func newWindowsExplorerRegistryStore() explorerRegistryStore {
 return windowsExplorerRegistryStore{}
}

func (windowsExplorerRegistryStore) Snapshot() (explorerRegistrationSnapshot,error) {
 handler,err:=readExplorerDefaultValue(explorerSendHandlerKey)
 if err!=nil{return explorerRegistrationSnapshot{},fmt.Errorf("read Explorer handler: %w",err)}
 server,err:=readExplorerDefaultValue(explorerSendServerKey)
 if err!=nil{return explorerRegistrationSnapshot{},fmt.Errorf("read COM server: %w",err)}
 return explorerRegistrationSnapshot{HandlerCLSID:handler,ServerPath:server},nil
}

func (windowsExplorerRegistryStore) CompareAndSwap(_, _ explorerRegistrationSnapshot) error {
 // No atomic compare-and-swap primitive is provided by the Win32 registry
 // functions used here. Do not silently emulate CAS with vulnerable independent
 // reads and writes or allow a test mode to register in the normal profile.
 return errExplorerRegistryWritesDisabled
}

// readExplorerDefaultValue reads only the default REG_SZ of one fixed HKCU key.
// A missing key/value means absent; malformed data or access errors fail closed.
func readExplorerDefaultValue(path string) (string,error) {
 name,err:=syscall.UTF16PtrFromString(path)
 if err!=nil{return "",err}
 var handle syscall.Handle
 err=syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER,name,0,syscall.KEY_QUERY_VALUE,&handle)
 if errors.Is(err,syscall.ERROR_FILE_NOT_FOUND)||errors.Is(err,syscall.ERROR_PATH_NOT_FOUND){return "",nil}
 if err!=nil{return "",err}
 defer syscall.RegCloseKey(handle)
 var kind,size uint32
 err=syscall.RegQueryValueEx(handle,nil,nil,&kind,nil,&size)
 if errors.Is(err,syscall.ERROR_FILE_NOT_FOUND){return "",nil}
 if err!=nil{return "",err}
 if kind!=syscall.REG_SZ || size<2 || size>65536 || size%2!=0 {
  return "",fmt.Errorf("invalid default registry value type or length: kind=%d size=%d",kind,size)
 }
 buf:=make([]byte,size)
 err=syscall.RegQueryValueEx(handle,nil,nil,&kind,&buf[0],&size)
 if err!=nil{return "",err}
 if kind!=syscall.REG_SZ || size<2 || size>uint32(len(buf)) || size%2!=0 {
  return "",errors.New("registry default value changed during read")
 }
 words:=make([]uint16,int(size)/2)
 for i:=range words{words[i]=binary.LittleEndian.Uint16(buf[i*2:])}
 if words[len(words)-1]!=0{return "",errors.New("registry default value is not null terminated")}
 return syscall.UTF16ToString(words),nil
}
