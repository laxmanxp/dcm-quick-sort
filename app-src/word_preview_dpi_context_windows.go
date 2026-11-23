//go:build windows

package main

import "runtime"

var wordDPIGetThreadContext = user32.NewProc("GetThreadDpiAwarenessContext")
var wordDPISetThreadContext = user32.NewProc("SetThreadDpiAwarenessContext")

// Word and HTML native handlers interpret their viewports correctly when their COM sizing
// calls run in a temporarily per-monitor-aware V2 context. Do not scale the
// rectangle or directly resize any renderer-owned HWND. Always restore the
// original caller context before returning to QuickSort's message loop.
func invokeNativePreviewCOMWithDPIContext(mode string, invoke func() uintptr) uintptr {
 if mode != "aware" && mode != "lifecycle" { return invoke() }
 runtime.LockOSThread()
 defer runtime.UnlockOSThread()
 if wordDPIGetThreadContext.Find()!=nil || wordDPISetThreadContext.Find()!=nil { return invoke() }
 original,_,_:=wordDPIGetThreadContext.Call()
 if original==0 { return invoke() }
 const pmAwareV2 = ^uintptr(3)
 previous,_,_:=wordDPISetThreadContext.Call(pmAwareV2)
 if previous==0 { return invoke() }
 defer wordDPISetThreadContext.Call(original)
 return invoke()
}
