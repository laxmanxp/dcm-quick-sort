//go:build !windows

package main

import (
	"os"
	"time"
)

func fileCreatedTime(_ string, info os.FileInfo) time.Time {
	if info == nil {
		return time.Time{}
	}
	// Portable fallback for development/tests. DCM's production target is
	// Windows, where fileCreatedTime reads the filesystem creation timestamp.
	return info.ModTime()
}
