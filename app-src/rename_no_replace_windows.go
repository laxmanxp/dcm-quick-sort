//go:build windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var renameNoReplaceMoveFileExW = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

// renameFileNoReplace atomically refuses an occupied destination. Flags zero
// also refuse a cross-volume move, which must use the verified copy/retirement
// fallback instead of an opaque native copy-and-delete operation.
func renameFileNoReplace(source, target string) error {
	from, err := renameNoReplaceWindowsPath(source)
	if err != nil {
		return &os.LinkError{Op: "rename", Old: source, New: target, Err: err}
	}
	to, err := renameNoReplaceWindowsPath(target)
	if err != nil {
		return &os.LinkError{Op: "rename", Old: source, New: target, Err: err}
	}
	result, _, callErr := renameNoReplaceMoveFileExW.Call(uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(to)), 0)
	runtime.KeepAlive(from)
	runtime.KeepAlive(to)
	if result == 0 {
		if callErr == syscall.Errno(0) {
			callErr = syscall.EINVAL
		}
		return &os.LinkError{Op: "rename", Old: source, New: target, Err: callErr}
	}
	return nil
}

func renameNoReplaceWindowsPath(path string) (*uint16, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(absolute, `\\?\`) && !strings.HasPrefix(absolute, `\\.\`) {
		if strings.HasPrefix(absolute, `\\`) {
			absolute = `\\?\UNC\` + absolute[2:]
		} else {
			absolute = `\\?\` + absolute
		}
	}
	return syscall.UTF16PtrFromString(absolute)
}
