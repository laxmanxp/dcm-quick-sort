//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Keep the surviving original protected against writes and deletion for the
// whole restore. Create the output exclusively and keep its own handle locked,
// so a failed restore can retire only the file this operation actually created.
func restoreDuplicateOriginal(rec MoveRecord) (err error) {
	in, err := openGuardedRegularFile(rec.To, false)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := verifyDuplicateOriginalHandle(in, rec); err != nil {
		return err
	}
	if _, err := in.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(rec.From), 0o755); err != nil {
		return err
	}
	absolute, err := filepath.Abs(rec.From)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(absolute, `\\?\`) && !strings.HasPrefix(absolute, `\\.\`) {
		if strings.HasPrefix(absolute, `\\`) {
			absolute = `\\?\UNC\` + absolute[2:]
		} else {
			absolute = `\\?\` + absolute
		}
	}
	path, err := syscall.UTF16PtrFromString(absolute)
	if err != nil {
		return err
	}
	const deleteAccess = 0x00010000
	handle, err := syscall.CreateFile(path, syscall.GENERIC_READ|syscall.GENERIC_WRITE|deleteAccess, 0, nil, syscall.CREATE_NEW, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return fmt.Errorf("cannot create original path without overwriting an existing file: %w", err)
	}
	out := os.NewFile(uintptr(handle), rec.From)
	if out == nil {
		_ = syscall.CloseHandle(handle)
		return fmt.Errorf("cannot own restored file handle: %w", syscall.EINVAL)
	}
	complete := false
	defer func() {
		if !complete {
			if cleanupErr := deleteGuardedFile(out); cleanupErr != nil {
				err = fmt.Errorf("restore failed: %v; incomplete output retained at %s because cleanup failed: %w", err, rec.From, cleanupErr)
			}
		}
		if closeErr := out.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	if err = out.Sync(); err != nil {
		return err
	}
	if err = verifyDuplicateOriginalHandle(out, rec); err != nil {
		return err
	}
	complete = true
	return nil
}
