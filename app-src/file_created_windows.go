//go:build windows

package main

import (
	"os"
	"syscall"
	"time"
)

func fileCreatedTime(_ string, info os.FileInfo) time.Time {
	if info == nil {
		return time.Time{}
	}
	if data, ok := info.Sys().(*syscall.Win32FileAttributeData); ok && data != nil {
		ns := data.CreationTime.Nanoseconds()
		if ns > 0 {
			return time.Unix(0, ns).Local()
		}
	}
	return info.ModTime()
}
