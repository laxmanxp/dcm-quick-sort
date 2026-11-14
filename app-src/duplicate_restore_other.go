//go:build !windows

package main

import "errors"

func restoreDuplicateOriginal(rec MoveRecord) error {
	return errors.New("protected duplicate restoration is supported only on Windows")
}
