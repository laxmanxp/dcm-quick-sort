//go:build !windows

package main

import "errors"

func sendFlowverseServiceWake(payload []byte) error {
	return errors.New("Flowverse named-pipe wake is only available on Windows")
}
