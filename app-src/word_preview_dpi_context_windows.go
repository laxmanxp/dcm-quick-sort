//go:build windows

package main

import (
 "runtime"
)

var wordDPIGetThreadContext = user32.NewProc("GetThreadDpiAwarenessContext")
var wordDPISetThreadContext = user32.NewProc("SetThreadDpiAwarenessContext")

// Isolated opt-in diagnostic only. Keep a single OS thread across the call,
// then restore its prior DPI awareness before returning to the message loop.
// No foreign Word HWND is resized, moved, destroyed, or subclassed.
func invokeWordCOMWithTestDPIContext(mode string, invoke func() uintptr) uintptr {
 if mode != "aware" { return invoke() }
 runtime.LockOSThread()
 defer runtime.UnlockOSThread()
 if wordDPIGetThreadContext.Find()!=nil || wordDPISetThreadContext.Find()!=nil { return invoke() }
 original,_,_:=wordDPIGetThreadContext.Call()
 if original==0 { return invoke() }
 const pmAwareV2 = ^uintptr(3)
 previous,_,_:=wordDPISetThreadContext.Call(pmAwareV2)
 if previous==0 { return invoke() }
 result:=invoke()
 // Restore immediately, including when COM reports failure.
 wordDPISetThreadContext.Call(original)
 return result
}
