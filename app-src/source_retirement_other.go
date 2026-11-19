//go:build !windows

package main

import (
	"errors"
	"os"
)

func openGuardedRegularFile(path string, forDelete bool) (*os.File, error) {
	return nil, errors.New("guarded file operations require the supported Windows runtime")
}

func deleteGuardedFile(file *os.File) error {
	return errors.New("verified handle retirement requires the supported Windows runtime")
}
