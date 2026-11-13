//go:build windows

package main

// Read-only, opt-in observations of the existing native preview calls. Records
// stay in memory until flushLayoutTrace; no DPI context, window or input changes
// are made here. Only identities, numeric results and rectangles are recorded.
import (
	"fmt"
	"time"
)

const nativePreviewTraceLimit = 128

type nativePreviewTraceSession struct {
	ID    uint64
	CLSID string
}

type nativePreviewTraceDPI struct {
	Thread                 uint32 `json:"thread"`
	ThreadContextAvailable bool   `json:"thread_context_available"`
	ThreadContext          uint64 `json:"thread_context"`
	ThreadAwareness        int32  `json:"thread_awareness"`
	PaneContextAvailable   bool   `json:"pane_context_available"`
	PaneContext            uint64 `json:"pane_context"`
	PaneAwareness          int32  `json:"pane_awareness"`
}

type nativePreviewTraceEvent struct {
	Call              uint64                `json:"call"`
	Session           uint64                `json:"session"`
	CLSID             string                `json:"clsid"`
	Method            string                `json:"method"`
	StartedUS         int64                 `json:"started_us"`
	FinishedUS        int64                 `json:"finished_us"`
	Completed         bool                  `json:"completed"`
	Pane              uint64                `json:"pane"`
	Handler           uint64                `json:"handler"`
	ActivationContext uint32                `json:"activation_context,omitempty"`
	HasRectArgument   bool                  `json:"has_rect_argument"`
	RectArgument      *RECT                 `json:"rect_argument,omitempty"`
	RectResult        *RECT                 `json:"rect_result,omitempty"`
	BOOLResult        *bool                 `json:"bool_result,omitempty"`
	HRESULT           *int32                `json:"hresult,omitempty"`
	HRESULTHex        string                `json:"hresult_hex,omitempty"`
	Before            nativePreviewTraceDPI `json:"before"`
	After             nativePreviewTraceDPI `json:"after"`
}

var nativePreviewTraceSessionCount, nativePreviewTraceCallCount uint64
var nativePreviewTraceEvents [nativePreviewTraceLimit]nativePreviewTraceEvent
var nativePreviewTraceActiveSession nativePreviewTraceSession
var nativePreviewTraceActiveHandler uintptr
var nativePreviewTraceGetThreadDPI = user32.NewProc("GetThreadDpiAwarenessContext")
var nativePreviewTraceGetWindowDPI = user32.NewProc("GetWindowDpiAwarenessContext")
var nativePreviewTraceGetAwareness = user32.NewProc("GetAwarenessFromDpiAwarenessContext")

func newNativePreviewTraceSession(clsid GUID) nativePreviewTraceSession {
	if !layoutTraceEnabled {
		return nativePreviewTraceSession{}
	}
	nativePreviewTraceSessionCount++
	return nativePreviewTraceSession{
		ID: nativePreviewTraceSessionCount,
		CLSID: fmt.Sprintf("{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
			clsid.Data1, clsid.Data2, clsid.Data3, clsid.Data4[0], clsid.Data4[1],
			clsid.Data4[2], clsid.Data4[3], clsid.Data4[4], clsid.Data4[5], clsid.Data4[6], clsid.Data4[7]),
	}
}

func setNativePreviewTraceActive(handler uintptr, session nativePreviewTraceSession) {
	if layoutTraceEnabled {
		nativePreviewTraceActiveHandler, nativePreviewTraceActiveSession = handler, session
	}
}

func nativePreviewTraceSessionFor(handler uintptr) nativePreviewTraceSession {
	if layoutTraceEnabled && handler == nativePreviewTraceActiveHandler {
		return nativePreviewTraceActiveSession
	}
	return nativePreviewTraceSession{}
}

func observeNativePreviewTraceDPI(pane uintptr) nativePreviewTraceDPI {
	d := nativePreviewTraceDPI{ThreadAwareness: -1, PaneAwareness: -1}
	thread, _, _ := layoutTraceThread.Call()
	d.Thread = uint32(thread)
	if nativePreviewTraceGetThreadDPI.Find() == nil {
		context, _, _ := nativePreviewTraceGetThreadDPI.Call()
		d.ThreadContext, d.ThreadContextAvailable = uint64(context), context != 0
	}
	if nativePreviewTraceGetWindowDPI.Find() == nil {
		context, _, _ := nativePreviewTraceGetWindowDPI.Call(pane)
		d.PaneContext, d.PaneContextAvailable = uint64(context), context != 0
	}
	if nativePreviewTraceGetAwareness.Find() == nil {
		if d.ThreadContextAvailable {
			awareness, _, _ := nativePreviewTraceGetAwareness.Call(uintptr(d.ThreadContext))
			d.ThreadAwareness = int32(awareness)
		}
		if d.PaneContextAvailable {
			awareness, _, _ := nativePreviewTraceGetAwareness.Call(uintptr(d.PaneContext))
			d.PaneAwareness = int32(awareness)
		}
	}
	return d
}

func beginNativePreviewTrace(session nativePreviewTraceSession, method string, pane, handler, activationContext uintptr, rect *RECT) *nativePreviewTraceEvent {
	if !layoutTraceEnabled || session.ID == 0 {
		return nil
	}
	nativePreviewTraceCallCount++
	// Keep the first calls, including initial activation, rather than evicting
	// them on a later resize. The exported total makes any truncation explicit.
	if nativePreviewTraceCallCount > nativePreviewTraceLimit {
		return nil
	}
	e := &nativePreviewTraceEvents[nativePreviewTraceCallCount-1]
	*e = nativePreviewTraceEvent{
		Call: nativePreviewTraceCallCount, Session: session.ID, CLSID: session.CLSID,
		Method: method, StartedUS: time.Since(layoutTraceStart).Microseconds(),
		Pane: uint64(pane), Handler: uint64(handler), ActivationContext: uint32(activationContext),
		HasRectArgument: rect != nil, Before: observeNativePreviewTraceDPI(pane),
	}
	if rect != nil {
		copy := *rect
		e.RectArgument = &copy
	}
	return e
}

func endNativePreviewTraceHRESULT(e *nativePreviewTraceEvent, hr, handler uintptr) {
	if e == nil {
		return
	}
	result := int32(hr)
	e.HRESULT, e.HRESULTHex, e.Handler = &result, fmt.Sprintf("0x%08X", uint32(hr)), uint64(handler)
	e.After = observeNativePreviewTraceDPI(uintptr(e.Pane))
	e.FinishedUS, e.Completed = time.Since(layoutTraceStart).Microseconds(), true
}

func endNativePreviewTraceClientRect(e *nativePreviewTraceEvent, ok uintptr, rect RECT) {
	if e == nil {
		return
	}
	result := ok != 0
	e.BOOLResult, e.RectResult = &result, &rect
	e.After = observeNativePreviewTraceDPI(uintptr(e.Pane))
	e.FinishedUS, e.Completed = time.Since(layoutTraceStart).Microseconds(), true
}

func nativePreviewTraceSnapshot() (uint64, []nativePreviewTraceEvent) {
	n := nativePreviewTraceCallCount
	if n > nativePreviewTraceLimit {
		n = nativePreviewTraceLimit
	}
	events := make([]nativePreviewTraceEvent, n)
	copy(events, nativePreviewTraceEvents[:n])
	return nativePreviewTraceCallCount, events
}
