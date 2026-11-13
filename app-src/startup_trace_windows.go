//go:build windows

package main

// Diagnostic only: bounded numeric events, no path/password logging, no file I/O
// in the UI handlers. Ordinary launches leave the probe disabled.
import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

const wmStartupProbe = 0x83A1

var startupProbeEnabled = os.Getenv("DCM_STARTUP_TRACE") == "1"
var startupProbeEpoch = time.Now()
var startupProbeSequence uint64
var startupProbeEvents = make([]startupProbeEvent, 0, 256)

type startupProbeEvent struct {
	Microseconds int64  `json:"us"`
	Sequence     uint64 `json:"sequence"`
	Stage        string `json:"stage"`
}

func beginStartupTrace() uint64 {
	if !startupProbeEnabled {
		return 0
	}
	startupProbeSequence++
	recordStartupTrace(startupProbeSequence, "start-enter")
	return startupProbeSequence
}
func recordStartupTrace(sequence uint64, stage string) {
	if !startupProbeEnabled || sequence == 0 || len(startupProbeEvents) >= 1024 {
		return
	}
	startupProbeEvents = append(startupProbeEvents, startupProbeEvent{time.Since(startupProbeEpoch).Microseconds(), sequence, stage})
}
func startupHeartbeat(hwnd, token uintptr) {
	if !startupProbeEnabled || token == 0 {
		return
	}
	layoutTraceSetProp.Call(hwnd, uintptr(unsafe.Pointer(wstr("DCM.Startup.Ack"))), token)
}
func flushStartupTrace() {
	if !startupProbeEnabled {
		return
	}
	value := struct {
		Schema int                 `json:"schema"`
		Events []startupProbeEvent `json:"events"`
	}{1, startupProbeEvents}
	if b, err := json.MarshalIndent(value, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(os.TempDir(), "dcm-startup-trace.json"), b, 0600)
	}
}

var startupReadySequence uint64

func recordStartupReady(status string) {
	if startupProbeEnabled && startupReadySequence != startupProbeSequence && strings.HasPrefix(status, "Sumatra PDF preview") {
		startupReadySequence = startupProbeSequence
		recordStartupTrace(startupProbeSequence, "document-ready-status")
		layoutTraceSetProp.Call(app.hwnd, uintptr(unsafe.Pointer(wstr("DCM.Startup.Ready"))), uintptr(startupProbeSequence))
	}
}
