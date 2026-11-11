//go:build windows

package main

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

const qsWMContextMenu = 0x007B

type qsContextMenu struct{ Vtbl *qsContextMenuVtbl }
type qsContextMenuVtbl struct {
	QueryInterface, AddRef, Release                   uintptr
	QueryContextMenu, InvokeCommand, GetCommandString uintptr
	HandleMenuMsg, HandleMenuMsg2                     uintptr
}
type qsInvokeCommand struct {
	Size, Mask                                          uint32
	Window, Verb, Parameters, Directory                 uintptr
	Show                                                int32
	HotKey                                              uint32
	Icon, Title, VerbW, ParametersW, DirectoryW, TitleW uintptr
	Point                                               POINT
}

var (
	qsTrackPopupMenu = user32.NewProc("TrackPopupMenuEx")
	qsScreenToClient = user32.NewProc("ScreenToClient")
	qsClientToScreen = user32.NewProc("ClientToScreen")
	qsMenu2, qsMenu3 *qsContextMenu // UI thread only, while TrackPopupMenuEx runs.
	qsContextActive  bool
)

func init() {
	// registerClasses already consumes mainWndProc. Install this narrow wrapper
	// before main starts; all unrelated messages retain the existing handler.
	mainWndProc = syscall.NewCallback(qsContextWindowProc)
}

func qsContextWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if result, handled := qsForwardMenuMessage(msg, wParam, lParam); handled {
		return result
	}
	if msg == qsWMContextMenu && wParam == app.fileList && app.fileList != 0 {
		if app.backgroundMoveBusy || qsContextActive {
			return 0
		}
		point := POINT{X: int32(int16(uint16(lParam))), Y: int32(int16(uint16(lParam >> 16)))}
		keyboard := point.X == -1 && point.Y == -1
		hit, inside := -1, false
		if keyboard {
			if fileListSelectionCount() == 0 {
				return 0
			}
			hit = app.index
			if hit >= 0 && hit < len(app.files) {
				var rect RECT
				ok, _, _ := procSendMessageW.Call(app.fileList, 0x0198, uintptr(hit), uintptr(unsafe.Pointer(&rect)))
				if ok == LB_ERR {
					return 0
				}
				point = POINT{X: rect.Left + 8, Y: rect.Bottom}
				if ok, _, _ := qsClientToScreen.Call(app.fileList, uintptr(unsafe.Pointer(&point))); ok == 0 {
					return 0
				}
				inside = true
			}
		} else {
			client := point
			if ok, _, _ := qsScreenToClient.Call(app.fileList, uintptr(unsafe.Pointer(&client))); ok == 0 {
				return 0
			}
			packed := uintptr(uint32(uint16(client.X)) | uint32(uint16(client.Y))<<16)
			value, _, _ := procSendMessageW.Call(app.fileList, 0x01A9, 0, packed) // LB_ITEMFROMPOINT
			hit = int(uint16(value))
			if uint16(value>>16) == 0 && hit < len(app.files) {
				var rect RECT
				ok, _, _ := procSendMessageW.Call(app.fileList, 0x0198, uintptr(hit), uintptr(unsafe.Pointer(&rect)))
				inside = ok != LB_ERR && qsContextContains(rect, client)
			}
		}
		index := qsContextIndex(len(app.files), hit, inside)
		if index < 0 {
			return 0
		}
		// This is a single-file shell menu. Never operate on an old selection
		// when the user right-clicks a different row, including in a filtered list.
		app.updatingFileList = true
		procSendMessageW.Call(app.fileList, LB_SETSEL, 0, ^uintptr(0))
		procSendMessageW.Call(app.fileList, LB_SETSEL, 1, uintptr(index))
		procSendMessageW.Call(app.fileList, LB_SETCARETINDEX, uintptr(index), 0)
		app.updatingFileList = false
		app.index = index
		showCurrent(false)
		target := currentFile()
		if target != "" {
			if err := qsShowShellMenu(hwnd, target, point); err != nil {
				setText(app.statusLabel, "Cannot open the Windows file menu: "+err.Error())
			}
		}
		return 0
	}
	return mainWindowProc(hwnd, msg, wParam, lParam)
}

func qsContextIndex(count, hit int, inside bool) int {
	if !inside || hit < 0 || hit >= count {
		return -1
	}
	return hit
}
func qsContextContains(rect RECT, point POINT) bool {
	return point.X >= rect.Left && point.X < rect.Right && point.Y >= rect.Top && point.Y < rect.Bottom
}
func qsContextOffset(command uintptr) (uintptr, bool) {
	const first, last = 1, 0x7FFF
	if command < first || command > last {
		return 0, false
	}
	return command - first, true
}
func qsReleaseMenu(menu *qsContextMenu) {
	if menu != nil {
		syscall.SyscallN(menu.Vtbl.Release, uintptr(unsafe.Pointer(menu)))
	}
}
func qsMenuInterface(menu *qsContextMenu, id string) *qsContextMenu {
	var iid GUID
	if !guidFromString(id, &iid) {
		return nil
	}
	var result *qsContextMenu
	hr, _, _ := syscall.SyscallN(menu.Vtbl.QueryInterface, uintptr(unsafe.Pointer(menu)), uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&result)))
	if hresultFailed(hr) {
		return nil
	}
	return result
}
func qsForwardMenuMessage(msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	switch msg {
	case 0x0117, 0x0120: // WM_INITMENUPOPUP, WM_MENUCHAR
	case 0x002B, 0x002C: // WM_DRAWITEM, WM_MEASUREITEM: only menus, not controls.
		if wParam != 0 {
			return 0, false
		}
	default:
		return 0, false
	}
	if qsMenu3 != nil {
		var result uintptr
		hr, _, _ := syscall.SyscallN(qsMenu3.Vtbl.HandleMenuMsg2, uintptr(unsafe.Pointer(qsMenu3)), uintptr(msg), wParam, lParam, uintptr(unsafe.Pointer(&result)))
		if hr == 0 {
			return result, true
		}
	} else if qsMenu2 != nil && msg != 0x0120 {
		hr, _, _ := syscall.SyscallN(qsMenu2.Vtbl.HandleMenuMsg, uintptr(unsafe.Pointer(qsMenu2)), uintptr(msg), wParam, lParam)
		if hr == 0 {
			if msg == 0x002B || msg == 0x002C {
				return 1, true
			}
			return 0, true
		}
	}
	return 0, false
}

func qsShowShellMenu(hwnd uintptr, path string, point POINT) error {
	qsContextActive = true
	defer func() { qsContextActive = false }()
	item := shellItemFromPath(path)
	if item == nil {
		return fmt.Errorf("cannot identify the selected file")
	}
	defer releaseShellItem(item)
	var bhid, iid GUID
	if !guidFromString("{3981E225-F559-11D3-8E3A-00C04F6837D5}", &bhid) ||
		!guidFromString("{000214E4-0000-0000-C000-000000000046}", &iid) {
		return fmt.Errorf("invalid Shell interface identifiers")
	}
	var menu *qsContextMenu
	hr, _, _ := syscall.SyscallN(item.Vtbl.BindToHandler, uintptr(unsafe.Pointer(item)), 0, uintptr(unsafe.Pointer(&bhid)), uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&menu)))
	if hresultFailed(hr) || menu == nil {
		return fmt.Errorf("Shell menu unavailable (0x%08x)", uint32(hr))
	}
	defer qsReleaseMenu(menu)
	native, _, _ := procCreatePopupMenu.Call()
	if native == 0 {
		return fmt.Errorf("cannot create popup")
	}
	defer procDestroyMenu.Call(native)
	hr, _, _ = syscall.SyscallN(menu.Vtbl.QueryContextMenu, uintptr(unsafe.Pointer(menu)), native, 0, 1, 0x7FFF, 0)
	if hresultFailed(hr) {
		return fmt.Errorf("cannot populate Shell menu (0x%08x)", uint32(hr))
	}
	qsMenu3 = qsMenuInterface(menu, "{BCFCE0A0-EC17-11D0-8D10-00A0C90F2719}")
	if qsMenu3 == nil {
		qsMenu2 = qsMenuInterface(menu, "{000214F4-0000-0000-C000-000000000046}")
	}
	defer func() { qsReleaseMenu(qsMenu3); qsReleaseMenu(qsMenu2); qsMenu3 = nil; qsMenu2 = nil }()
	procSetForegroundWindow.Call(hwnd)
	// TPM_RETURNCMD | TPM_RIGHTBUTTON: zero means cancellation, not command zero.
	command, _, _ := qsTrackPopupMenu.Call(native, 0x0102, uintptr(point.X), uintptr(point.Y), hwnd, 0)
	offset, selected := qsContextOffset(command)
	if !selected {
		return nil
	}
	invoke := qsInvokeCommand{Mask: 0x00004000 | 0x20000000, Window: hwnd, Verb: offset, VerbW: offset, Show: SW_SHOWNORMAL, Point: point}
	invoke.Size = uint32(unsafe.Sizeof(invoke))
	// Keep normal Windows safety/association handling; never bypass zone checks.
	hr, _, _ = syscall.SyscallN(menu.Vtbl.InvokeCommand, uintptr(unsafe.Pointer(menu)), uintptr(unsafe.Pointer(&invoke)))
	runtime.KeepAlive(menu)
	runtime.KeepAlive(invoke)
	if hresultFailed(hr) {
		return fmt.Errorf("Windows command failed (0x%08x)", uint32(hr))
	}
	// External Shell operations do not become DCM journal/Undo transactions.
	reloadFilesWithFocus(false)
	return nil
}
