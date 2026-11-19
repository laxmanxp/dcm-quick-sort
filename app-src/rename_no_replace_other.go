//go:build !windows

package main

import (
	"errors"
	"os"
)

func renameFileNoReplace(source, target string) error {
	return &os.LinkError{Op: "rename", Old: source, New: target, Err: errors.New("atomic no-replace rename requires the supported Windows runtime")}
}
