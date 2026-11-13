//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// All coordinator fields are UI-thread-owned. Only one preparation/cleanup
// worker runs at once; a burst of selections replaces, rather than grows, the
// single queued request. Workers capture values and never access app or HWND UI.
type sumatraLaunchSpec struct {
	path, runtime string
	parent        uintptr
	traceID       uint64
}

var sumatraPending *preparedResource[*sumatraSession]
var sumatraQueued *sumatraLaunchSpec
var sumatraRetired *sumatraSession // retained on unconfirmed cleanup; fail closed
var sumatraClosing bool

func ensureSumatraTimer() bool {
	timer, _, _ := spSetTimer.Call(app.hwnd, sumatraTimerID, 100, 0)
	return timer != 0
}
func cancelSumatraPreparation() {
	sumatraQueued = nil
	if sumatraPending != nil {
		sumatraPending.Cancel()
	}
}
func sumatraPreparationBusy() bool {
	return sumatraPending != nil || sumatraQueued != nil || sumatraRetired != nil
}

func advanceSumatraPreparation() {
	if p := sumatraPending; p != nil {
		if p.decided {
			if out, done := p.PollDone(); done {
				sumatraPending = nil
				if out.err != nil {
					sumatraRetired = out.value
					sumatraQueued = nil
					sumatraClosing = false
					sumatraError(out.err)
					return
				}
			}
		} else if v, err, ready := p.Take(); ready {
			if err != nil {
				sumatraError(err)
			} else {
				// Take is the sole ownership transfer. The worker never resumes a process;
				// a cancelled/obsolete launch cannot create a visible plugin window.
				sumatraPending = nil
				sumatraView = v
				if r, _, e := spResumeThread.Call(v.thread); uint32(r) == 0xffffffff {
					_ = stopSumatraPreview()
					sumatraError(fmt.Errorf("resume viewer: %w", e))
					return
				}
				spClose.Call(v.thread)
				v.thread = 0
				v.deadline = time.Now().Add(15 * time.Second)
				recordStartupTrace(v.traceID, "process-resumed")
			}
		}
	}
	if sumatraPending == nil && sumatraRetired == nil && sumatraQueued != nil && !sumatraClosing {
		spec := *sumatraQueued
		sumatraQueued = nil
		sumatraPending = newPreparedResource(func(ctx context.Context) (*sumatraSession, error) { return prepareSumatraProcess(ctx, spec) }, disposePreparedSumatra)
	}
	if sumatraClosing && !sumatraPreparationBusy() && sumatraView == nil {
		spKillTimer.Call(app.hwnd, sumatraTimerID)
		procPostMessageW.Call(app.hwnd, WM_CLOSE, 0, 0)
	} else if !sumatraPreparationBusy() && sumatraView == nil {
		spKillTimer.Call(app.hwnd, sumatraTimerID)
	}
}

// A normal close keeps the message loop alive until a blocked preparation has
// returned and its worker confirms disposal. No synchronous UI wait on startup.
func closeSumatraPreview() bool {
	sumatraClosing = true
	if err := stopSumatraPreview(); err != nil {
		sumatraClosing = false
		sumatraError(err)
		return false
	}
	if sumatraPreparationBusy() {
		setText(app.statusLabel, "Closing PDF preview safely; waiting for owned startup cleanup…")
		return false
	}
	sumatraClosing = false
	return true
}

var (
	spInitAttributes   = kernel32.NewProc("InitializeProcThreadAttributeList")
	spUpdateAttribute  = kernel32.NewProc("UpdateProcThreadAttribute")
	spDeleteAttributes = kernel32.NewProc("DeleteProcThreadAttributeList")
	spIsProcessInJob   = kernel32.NewProc("IsProcessInJob")
)

type sumatraStartupInfoEx struct {
	Info       syscall.StartupInfo
	Attributes uintptr
}

// Background-only preparation. On error a partial session is returned so the
// same worker can dispose it. The child remains suspended until UI adoption.
func prepareSumatraProcess(ctx context.Context, spec sumatraLaunchSpec) (*sumatraSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	document, err := validateSumatraDocument(spec.path)
	if err != nil {
		return nil, err
	}
	recordStartupTrace(spec.traceID, "document-validated")
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	exe, err := validateSumatraRuntime(spec.runtime)
	if err != nil {
		return nil, err
	}
	recordStartupTrace(spec.traceID, "runtime-validated")
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	profile, err := os.MkdirTemp("", "dcm-sumatra-")
	if err != nil {
		return nil, err
	}
	v := &sumatraSession{profile: profile, path: document, traceID: spec.traceID}
	args, err := sumatraArguments(spec.parent, document, profile)
	if err != nil {
		return v, err
	}
	argv := append([]string{exe}, args...)
	for i := range argv {
		argv[i] = syscall.EscapeArg(argv[i])
	}
	command, err := syscall.UTF16FromString(strings.Join(argv, " "))
	if err != nil {
		return v, err
	}
	job, _, e := spCreateJob.Call(0, 0)
	v.job = job
	if job == 0 {
		return v, fmt.Errorf("create viewer job: %w", e)
	}
	limits := sumatraExtendedLimit{}
	limits.Basic.Flags = 0x2000
	if ok, _, e := spSetJob.Call(job, 9, uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits)); ok == 0 {
		return v, fmt.Errorf("contain viewer: %w", e)
	}
	// Creation-time job assignment closes the suspended-but-not-yet-assigned
	// gap, including cancellation/parent-exit races. No fallback to unowned start.
	var size uintptr
	spInitAttributes.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))
	if size == 0 || size > 65536 {
		return v, errors.New("viewer job attribute size unavailable")
	}
	words := make([]uintptr, (size+unsafe.Sizeof(uintptr(0))-1)/unsafe.Sizeof(uintptr(0)))
	attributes := uintptr(unsafe.Pointer(&words[0]))
	if ok, _, e := spInitAttributes.Call(attributes, 1, 0, uintptr(unsafe.Pointer(&size))); ok == 0 {
		return v, fmt.Errorf("initialize viewer job attributes: %w", e)
	}
	defer func() { spDeleteAttributes.Call(attributes); runtime.KeepAlive(words); runtime.KeepAlive(v) }()
	if ok, _, e := spUpdateAttribute.Call(attributes, 0, 0x0002000d, uintptr(unsafe.Pointer(&v.job)), unsafe.Sizeof(v.job), 0, 0); ok == 0 {
		return v, fmt.Errorf("set creation-time viewer job: %w", e)
	}
	si := sumatraStartupInfoEx{Attributes: attributes}
	si.Info.Cb = uint32(unsafe.Sizeof(si))
	si.Info.Flags = 1
	si.Info.ShowWindow = 0
	if err = ctx.Err(); err != nil {
		return v, err
	}
	recordStartupTrace(spec.traceID, "profile-job-prepared")
	var pi syscall.ProcessInformation
	ok, _, e := spCreateProcess.Call(uintptr(unsafe.Pointer(wstr(exe))), uintptr(unsafe.Pointer(&command[0])), 0, 0, 0, 0x00080000|4, 0, uintptr(unsafe.Pointer(wstr(filepath.Dir(exe)))), uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi)))
	runtime.KeepAlive(command)
	runtime.KeepAlive(words)
	runtime.KeepAlive(v)
	recordStartupTrace(spec.traceID, "create-process-return")
	if ok == 0 {
		return v, fmt.Errorf("start Sumatra in owned job: %w", e)
	}
	v.process = uintptr(pi.Process)
	v.thread = uintptr(pi.Thread)
	v.pid = pi.ProcessId
	var inJob int32
	if ok, _, e := spIsProcessInJob.Call(v.process, v.job, uintptr(unsafe.Pointer(&inJob))); ok == 0 || inJob == 0 {
		return v, fmt.Errorf("verify created viewer job: %w", e)
	}
	if err = ctx.Err(); err != nil {
		return v, err
	}
	return v, nil
}

// No UI work, no foreign-window operations. Keep handles/profile on failure so
// file operations/shutdown cannot pretend release succeeded. The job is owned.
func disposePreparedSumatra(v *sumatraSession) error {
	if v == nil {
		return nil
	}
	if v.process != 0 {
		r, _, _ := spWait.Call(v.process, 0)
		if r == 258 {
			if ok, _, e := spTerminateJob.Call(v.job, 0); ok == 0 {
				return fmt.Errorf("cancel owned startup: %w", e)
			}
			r, _, _ = spWait.Call(v.process, 1500)
		}
		if r != 0 {
			return errors.New("startup cleanup not confirmed; file operations remain blocked")
		}
		spClose.Call(v.process)
		v.process = 0
	}
	if v.thread != 0 {
		spClose.Call(v.thread)
		v.thread = 0
	}
	if v.job != 0 {
		spClose.Call(v.job)
		v.job = 0
	}
	if v.profile != "" {
		if err := os.RemoveAll(v.profile); err != nil {
			return fmt.Errorf("remove owned startup profile: %w", err)
		}
	}
	recordStartupTrace(v.traceID, "cancelled-preparation-disposed")
	return nil
}
