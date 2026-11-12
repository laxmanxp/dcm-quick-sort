//go:build windows

package main

// Opt-in, bounded diagnostic recording. The UI thread only records numbers in
// memory; no file writes or forced repaint occur during layout/capture. Tests
// provide a private TEMP directory. Ordinary launches do not enable this.
import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
	"unsafe"
)

const wmLayoutTraceFence = 0x83A0
const layoutTraceLimit = 8192

var layoutTraceEnabled = os.Getenv("DCM_LAYOUT_TRACE") == "1"
var layoutTraceStart = time.Now()
var layoutTraceDepth, layoutTraceSequence, layoutTraceCount uint64
var layoutTraceEvents [layoutTraceLimit]layoutTraceEvent
var layoutTraceSetProp = user32.NewProc("SetPropW")
var layoutTraceThread = kernel32.NewProc("GetCurrentThreadId")

type layoutTraceEvent struct {
	Microseconds int64  `json:"us"`
	Event        string `json:"event"`
	Sequence     uint64 `json:"sequence"`
	Depth        uint64 `json:"depth"`
	Width        int32  `json:"width"`
	Height       int32  `json:"height"`
	Thread       uint32 `json:"thread"`
	Token        uint64 `json:"token,omitempty"`
}

func recordLayoutTrace(hwnd uintptr, event string, token uintptr) {
	if !layoutTraceEnabled {
		return
	}
	var rc RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	tid, _, _ := layoutTraceThread.Call()
	e := layoutTraceEvent{time.Since(layoutTraceStart).Microseconds(), event, layoutTraceSequence, layoutTraceDepth, rc.Right - rc.Left, rc.Bottom - rc.Top, uint32(tid), uint64(token)}
	layoutTraceEvents[layoutTraceCount%layoutTraceLimit] = e
	layoutTraceCount++
}
func beginLayoutTrace(hwnd uintptr) {
	if !layoutTraceEnabled {
		return
	}
	layoutTraceSequence++
	layoutTraceDepth++
	layoutTraceSetProp.Call(hwnd, uintptr(unsafe.Pointer(wstr("DCM.Layout.Sequence"))), uintptr(layoutTraceSequence))
	layoutTraceSetProp.Call(hwnd, uintptr(unsafe.Pointer(wstr("DCM.Layout.Depth"))), uintptr(layoutTraceDepth))
	recordLayoutTrace(hwnd, "layout-enter", 0)
}
func endLayoutTrace(hwnd uintptr) {
	if !layoutTraceEnabled {
		return
	}
	recordLayoutTrace(hwnd, "layout-exit", 0)
	layoutTraceDepth--
	layoutTraceSetProp.Call(hwnd, uintptr(unsafe.Pointer(wstr("DCM.Layout.Depth"))), uintptr(layoutTraceDepth))
}
func layoutTraceFence(hwnd, token uintptr) {
	if !layoutTraceEnabled {
		return
	}
	recordLayoutTrace(hwnd, "queue-fence", token)
	layoutTraceSetProp.Call(hwnd, uintptr(unsafe.Pointer(wstr("DCM.Layout.Fence"))), token)
}
func flushLayoutTrace() {
	if !layoutTraceEnabled {
		return
	}
	n := layoutTraceCount
	start := uint64(0)
	if n > layoutTraceLimit {
		start = n - layoutTraceLimit
	}
	events := make([]layoutTraceEvent, 0, n-start)
	for i := start; i < n; i++ {
		events = append(events, layoutTraceEvents[i%layoutTraceLimit])
	}
	b, err := json.MarshalIndent(struct {
		Schema int                `json:"schema"`
		Total  uint64             `json:"total"`
		Events []layoutTraceEvent `json:"events"`
	}{1, n, events}, "", "  ")
	if err == nil {
		_ = os.WriteFile(filepath.Join(os.TempDir(), "dcm-layout-trace.json"), b, 0600)
	}
}
