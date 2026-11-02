//go:build windows

package main

import "os"

func sendFlowverseServiceWake(payload []byte) error {
	f, err := os.OpenFile(fileFlowServicePipeName, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(payload)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
