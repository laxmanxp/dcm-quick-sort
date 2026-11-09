//go:build windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

const guardedFileDeleteAccess = uint32(0x00010000)

var sourceRetirementSetFileInformationByHandle = syscall.NewLazyDLL("kernel32.dll").NewProc("SetFileInformationByHandle")

// openGuardedRegularFile denies other write/delete handles until Close. Opening
// the final reparse point itself lets us refuse links without deleting a link's
// target when the caller later requests deletion by handle.
func openGuardedRegularFile(path string, forDelete bool) (*os.File, error) {
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
	name, err := syscall.UTF16PtrFromString(absolute)
	if err != nil {
		return nil, err
	}
	access := uint32(syscall.GENERIC_READ)
	if forDelete {
		access |= guardedFileDeleteAccess
	}
	handle, err := syscall.CreateFile(name, access, syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, &os.PathError{Op: "guard", Path: path, Err: err}
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = syscall.CloseHandle(handle)
		return nil, &os.PathError{Op: "guard", Path: path, Err: syscall.EINVAL}
	}
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(handle, &info); err != nil {
		file.Close()
		return nil, &os.PathError{Op: "guard", Path: path, Err: err}
	}
	if info.FileAttributes&(syscall.FILE_ATTRIBUTE_REPARSE_POINT|syscall.FILE_ATTRIBUTE_DIRECTORY) != 0 {
		file.Close()
		return nil, &os.PathError{Op: "guard", Path: path, Err: errors.New("source retirement requires a regular file without a reparse point")}
	}
	stat, err := file.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		file.Close()
		if err == nil {
			err = errors.New("guarded path is not a regular file")
		}
		return nil, &os.PathError{Op: "guard", Path: path, Err: err}
	}
	return file, nil
}

// deleteGuardedFile marks this exact handle for deletion. It deliberately does
// not close the handle; callers keep other required guards until Close returns.
func deleteGuardedFile(file *os.File) error {
	if file == nil {
		return syscall.EINVAL
	}
	disposition := struct{ DeleteFile byte }{DeleteFile: 1}
	result, _, callErr := sourceRetirementSetFileInformationByHandle.Call(file.Fd(), 4, uintptr(unsafe.Pointer(&disposition)), unsafe.Sizeof(disposition))
	runtime.KeepAlive(file)
	if result == 0 {
		if callErr == syscall.Errno(0) {
			callErr = syscall.EINVAL
		}
		return &os.PathError{Op: "retire", Path: file.Name(), Err: callErr}
	}
	return nil
}
