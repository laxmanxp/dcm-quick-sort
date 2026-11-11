//go:build windows

package main

// Reuses the prototype's separate-process -plugin/owned-job design, not Sumatra
// source code. A pinned unmodified runtime remains an opt-in external dependency.
import (
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

const sumatraTimerID = 0x5355

var (
	spCreateProcess    = kernel32.NewProc("CreateProcessW")
	spCreateJob        = kernel32.NewProc("CreateJobObjectW")
	spSetJob           = kernel32.NewProc("SetInformationJobObject")
	spAssignJob        = kernel32.NewProc("AssignProcessToJobObject")
	spTerminateJob     = kernel32.NewProc("TerminateJobObject")
	spResumeThread     = kernel32.NewProc("ResumeThread")
	spWait             = kernel32.NewProc("WaitForSingleObject")
	spTerminateProcess = kernel32.NewProc("TerminateProcess")
	spClose            = kernel32.NewProc("CloseHandle")
	spEnumChildren     = user32.NewProc("EnumChildWindows")
	spGetPID           = user32.NewProc("GetWindowThreadProcessId")
	spSetTimer         = user32.NewProc("SetTimer")
	spKillTimer        = user32.NewProc("KillTimer")
	spSetWindowPos     = user32.NewProc("SetWindowPos")
)

type sumatraBasicLimit struct {
	ProcessTime, JobTime           int64
	Flags                          uint32
	MinWorkingSet, MaxWorkingSet   uintptr
	ActiveProcesses                uint32
	Affinity                       uintptr
	PriorityClass, SchedulingClass uint32
}
type sumatraExtendedLimit struct {
	Basic                                                      sumatraBasicLimit
	IO                                                         [6]uint64
	ProcessMemory, JobMemory, PeakProcessMemory, PeakJobMemory uintptr
}
type sumatraSession struct {
	process, job, child uintptr
	pid                 uint32
	profile, path       string
	deadline            time.Time
}

var sumatraView *sumatraSession

func sumatraOwnsWindow(v *sumatraSession, hwnd uintptr) bool {
	if v == nil || hwnd == 0 {
		return false
	}
	var pid uint32
	spGetPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	parent, _, _ := procGetParent.Call(hwnd)
	return pid == v.pid && parent == app.previewHwnd
}

var sumatraFindChild = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
	if sumatraOwnsWindow(sumatraView, hwnd) {
		sumatraView.child = hwnd
		return 0
	}
	return 1
})

func sumatraError(err error) {
	app.fallback = "PDF preview unavailable.\n\n" + err.Error() + "\n\nSelect another file or Reload to retry."
	setText(app.statusLabel, "Sumatra preview unavailable: "+err.Error())
	updatePreviewZoomControls()
	procInvalidateRect.Call(app.previewHwnd, 0, 1)
}

// Startup returns immediately after a suspended process is assigned to its own
// kill-on-close job. A UI timer observes attachment instead of waiting on rendering.
func startSumatraPreview(path string) bool {
	if !useSumatraProvider() {
		return false
	}
	if err := stopSumatraPreview(); err != nil {
		sumatraError(err)
		return true
	}
	document, err := validateSumatraDocument(path)
	if err != nil {
		sumatraError(err)
		return true
	}
	exe, err := validateSumatraRuntime(os.Getenv("DCM_SUMATRA_EXE"))
	if err != nil {
		sumatraError(err)
		return true
	}
	profile, err := os.MkdirTemp("", "dcm-sumatra-")
	if err != nil {
		sumatraError(err)
		return true
	}
	args, err := sumatraArguments(app.previewHwnd, document, profile)
	if err != nil {
		os.RemoveAll(profile)
		sumatraError(err)
		return true
	}
	argv := append([]string{exe}, args...)
	for i := range argv {
		argv[i] = syscall.EscapeArg(argv[i])
	}
	command, err := syscall.UTF16FromString(strings.Join(argv, " "))
	if err != nil {
		os.RemoveAll(profile)
		sumatraError(err)
		return true
	}
	job, _, callErr := spCreateJob.Call(0, 0)
	if job == 0 {
		os.RemoveAll(profile)
		sumatraError(fmt.Errorf("create viewer job: %w", callErr))
		return true
	}
	limits := sumatraExtendedLimit{}
	limits.Basic.Flags = 0x2000
	if ok, _, e := spSetJob.Call(job, 9, uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits)); ok == 0 {
		spClose.Call(job)
		os.RemoveAll(profile)
		sumatraError(fmt.Errorf("contain viewer: %w", e))
		return true
	}
	si := syscall.StartupInfo{}
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = 1
	si.ShowWindow = 0
	var pi syscall.ProcessInformation
	ok, _, e := spCreateProcess.Call(uintptr(unsafe.Pointer(wstr(exe))), uintptr(unsafe.Pointer(&command[0])), 0, 0, 0, 4, 0, uintptr(unsafe.Pointer(wstr(filepath.Dir(exe)))), uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi)))
	runtime.KeepAlive(command)
	if ok == 0 {
		spClose.Call(job)
		os.RemoveAll(profile)
		sumatraError(fmt.Errorf("start Sumatra: %w", e))
		return true
	}
	defer spClose.Call(uintptr(pi.Thread))
	sumatraView = &sumatraSession{process: uintptr(pi.Process), job: job, pid: pi.ProcessId, profile: profile, path: document, deadline: time.Now().Add(15 * time.Second)}
	if ok, _, e = spAssignJob.Call(job, uintptr(pi.Process)); ok == 0 {
		spTerminateProcess.Call(uintptr(pi.Process), 1)
		_ = stopSumatraPreview()
		sumatraError(fmt.Errorf("assign viewer job: %w", e))
		return true
	}
	if r, _, e := spResumeThread.Call(uintptr(pi.Thread)); uint32(r) == 0xffffffff {
		_ = stopSumatraPreview()
		sumatraError(fmt.Errorf("resume viewer: %w", e))
		return true
	}
	app.fallback = "Loading PDF in Sumatra…"
	setText(app.statusLabel, "Loading PDF in restricted Sumatra preview…")
	if timer, _, _ := spSetTimer.Call(app.hwnd, sumatraTimerID, 100, 0); timer == 0 {
		_ = stopSumatraPreview()
		sumatraError(errors.New("could not start viewer readiness timer"))
	}
	procInvalidateRect.Call(app.previewHwnd, 0, 1)
	return true
}

func pollSumatraPreview() {
	v := sumatraView
	if v == nil {
		return
	}
	if r, _, _ := spWait.Call(v.process, 0); r != 258 {
		_ = stopSumatraPreview()
		sumatraError(errors.New("the owned Sumatra viewer exited; Reload to retry"))
		return
	}
	if v.child == 0 {
		spEnumChildren.Call(app.previewHwnd, sumatraFindChild, 0)
		if v.child != 0 {
			resizeSumatraPreview()
			app.fallback = ""
			setText(app.statusLabel, "Sumatra PDF preview · restricted · use viewer navigation/zoom")
			procInvalidateRect.Call(app.previewHwnd, 0, 1)
		} else if time.Now().After(v.deadline) {
			_ = stopSumatraPreview()
			sumatraError(errors.New("Sumatra did not attach within the startup deadline"))
		}
	}
}
func resizeSumatraPreview() {
	v := sumatraView
	if v == nil || !sumatraOwnsWindow(v, v.child) {
		return
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	w, h := rc.Right-rc.Left, rc.Bottom-rc.Top
	if w < 1 || h < 1 {
		return
	}
	// Async cross-thread positioning avoids blocking QuickSort on the PDF renderer.
	spSetWindowPos.Call(v.child, 0, 0, 0, uintptr(w), uintptr(h), 0x4000|0x0040|0x0010|0x0004)
}

func stopSumatraPreview() error {
	v := sumatraView
	if v == nil {
		return nil
	}
	spKillTimer.Call(app.hwnd, sumatraTimerID)
	if sumatraOwnsWindow(v, v.child) {
		procShowWindow.Call(v.child, 0)
		procPostMessageW.Call(v.child, WM_CLOSE, 0, 0)
	}
	r, _, _ := spWait.Call(v.process, 200)
	if r == 258 {
		if ok, _, e := spTerminateJob.Call(v.job, 0); ok == 0 {
			return fmt.Errorf("could not release owned viewer job: %w", e)
		}
		r, _, _ = spWait.Call(v.process, 1500)
	}
	if r != 0 {
		return errors.New("viewer release not confirmed; file operations remain blocked")
	}
	spClose.Call(v.process)
	spClose.Call(v.job)
	sumatraView = nil
	// Only the exact session directory created by MkdirTemp belongs to this viewer.
	_ = os.RemoveAll(v.profile)
	return nil
}
func releaseSumatraForOperation() bool {
	if err := stopSumatraPreview(); err != nil {
		sumatraError(err)
		return false
	}
	return true
}
