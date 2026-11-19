//go:build windows

package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

// Modification time drives the Age filter; Windows creation time drives the
// Date Created sort. Test fixtures must set both, not depend on execution order.
func setInboxFixtureCreatedTime(t *testing.T, path string, created time.Time) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	stamp := syscall.NsecToFiletime(created.UnixNano())
	setErr := syscall.SetFileTime(syscall.Handle(file.Fd()), &stamp, nil, nil)
	closeErr := file.Close()
	if setErr != nil {
		t.Fatalf("set fixture creation time: %v", setErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok || data == nil || data.CreationTime.Nanoseconds() != created.UnixNano() {
		t.Fatal("fixture creation time was not recorded as requested")
	}
}
