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
	spShowWindowAsync  = user32.NewProc("ShowWindowAsync")
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
			layoutUI(app.hwnd)
			resizeSumatraPreview()
			app.fallback = ""
			setText(app.statusLabel, "Sumatra PDF preview · restricted · use viewer navigation/zoom")
			updatePreviewZoomControls()
			procInvalidateRect.Call(app.previewHwnd, 0, 1)
		} else if time.Now().After(v.deadline) {
			_ = stopSumatraPreview()
			sumatraError(errors.New("Sumatra did not attach within the startup deadline"))
		}
	}
	if sumatraView == v && v.child != 0 {
		repairSumatraToolbar(v)
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
		spShowWindowAsync.Call(v.child, 0)
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
	updatePreviewZoomControls()
	layoutUI(app.hwnd)
	// Only the exact session directory created by MkdirTemp belongs to this viewer.
	_ = os.RemoveAll(v.profile)
	return nil
}

// The pinned 3.6.1 toolbar lays out page fields relative to a hidden Print button.
// Repair only clipped direct Edit/Static children of our own viewer's toolbar.
// This is a host-side geometry compatibility fix, not a runtime/policy change.
type sumatraToolbarQuery struct {
	viewer  *sumatraSession
	toolbar uintptr
}

var spGetClass = user32.NewProc("GetClassNameW")
var spScreenToClient = user32.NewProc("ScreenToClient")
var spGetWindowDPIContext = user32.NewProc("GetWindowDpiAwarenessContext")
var spSetThreadDPIContext = user32.NewProc("SetThreadDpiAwarenessContext")

func sumatraClass(hwnd uintptr) string {
	var name [128]uint16
	n, _, _ := spGetClass.Call(hwnd, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(name[:])
}
func sumatraHasPID(hwnd uintptr, pid uint32) bool {
	var actual uint32
	spGetPID.Call(hwnd, uintptr(unsafe.Pointer(&actual)))
	return actual == pid
}

var sumatraFindToolbar = syscall.NewCallback(func(hwnd, context uintptr) uintptr {
	q := (*sumatraToolbarQuery)(unsafe.Pointer(context))
	if sumatraHasPID(hwnd, q.viewer.pid) && sumatraClass(hwnd) == "ToolbarWindow32" {
		q.toolbar = hwnd
		return 0
	}
	return 1
})
var sumatraRepairToolbarChild = syscall.NewCallback(func(hwnd, context uintptr) uintptr {
	q := (*sumatraToolbarQuery)(unsafe.Pointer(context))
	parent, _, _ := procGetParent.Call(hwnd)
	if parent != q.toolbar || !sumatraHasPID(hwnd, q.viewer.pid) {
		return 1
	}
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	class := sumatraClass(hwnd)
	if visible == 0 || (class != "Edit" && class != "Static") {
		return 1
	}
	var toolbar, child RECT
	if ok, _, _ := procGetClientRect.Call(q.toolbar, uintptr(unsafe.Pointer(&toolbar))); ok == 0 {
		return 1
	}
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&child))); ok == 0 {
		return 1
	}
	point := POINT{X: child.Left, Y: child.Top}
	if ok, _, _ := spScreenToClient.Call(q.toolbar, uintptr(unsafe.Pointer(&point))); ok == 0 {
		return 1
	}
	width, height := child.Right-child.Left, child.Bottom-child.Top
	if width <= 0 || point.X < 0 || width > toolbar.Right || point.X > toolbar.Right-width {
		return 1
	}
	y, move := sumatraToolbarChildY(point.Y, height, toolbar.Bottom)
	if move && sumatraView == q.viewer && sumatraOwnsWindow(q.viewer, q.viewer.child) {
		// Async and no-size/no-activate/no-z-order: only this child's y changes.
		spSetWindowPos.Call(hwnd, 0, uintptr(point.X), uintptr(y), 0, 0, 0x4000|0x0001|0x0010|0x0004)
	}
	return 1
})

func repairSumatraToolbar(v *sumatraSession) {
	if v == nil || !sumatraOwnsWindow(v, v.child) {
		return
	}
	q := &sumatraToolbarQuery{viewer: v}
	spEnumChildren.Call(v.child, sumatraFindToolbar, uintptr(unsafe.Pointer(q)))
	if q.toolbar != 0 {
		context, _, _ := spGetWindowDPIContext.Call(q.toolbar)
		previous, _, _ := spSetThreadDPIContext.Call(context)
		if previous != 0 {
			defer spSetThreadDPIContext.Call(previous)
			spEnumChildren.Call(q.toolbar, sumatraRepairToolbarChild, uintptr(unsafe.Pointer(q)))
		}
	}
	runtime.KeepAlive(q)
}

func releaseSumatraForOperation() bool {
	if err := stopSumatraPreview(); err != nil {
		sumatraError(err)
		return false
	}
	return true
}
