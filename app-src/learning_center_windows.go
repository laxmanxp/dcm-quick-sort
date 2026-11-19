//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	ID_LC_LIST    = 1301
	ID_LC_PAUSE   = 1302
	ID_LC_RESET   = 1303
	ID_LC_REFRESH = 1304
	ID_LC_CLOSE   = 1305
)

type learningCenterWindowState struct {
	hwnd, list, detail, statusLabel          uintptr
	pauseBtn, resetBtn, refreshBtn, closeBtn uintptr
	items                                    []LearningPatternItem
	state                                    LearningCenterState
	selected                                 int
}

var learningCenter learningCenterWindowState
var learningCenterWndProcCB = syscall.NewCallback(learningCenterWndProc)

func registerLearningCenterClass(hInst, cursor uintptr) bool {
	class := wstr("QuickSortLearningCenter")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: learningCenterWndProcCB, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: class}
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	return r != 0
}

func openLearningCenter() {
	if learningCenter.hwnd != 0 {
		procShowWindow.Call(learningCenter.hwnd, SW_SHOW)
		refreshLearningCenterItems(learningCenter.selected)
		return
	}
	hInst, _, _ := procGetModuleHandleW.Call(0)
	style := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE | WS_CLIPCHILDREN)
	hwnd, _, _ := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(wstr("QuickSortLearningCenter"))),
		uintptr(unsafe.Pointer(wstr("Learning Center — DCM - Quick Sort"))),
		style, CW_USEDEFAULT, CW_USEDEFAULT, 1060, 710, app.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		showError("Could not open Learning Center.")
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
}

func learningCenterWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		learningCenter.hwnd = hwnd
		learningCenter.selected = -1
		styleTitleBar(hwnd)
		createLearningCenterUI(hwnd)
		refreshLearningCenterItems(-1)
		return 0
	case WM_CTLCOLORSTATIC:
		procSetBkMode.Call(wParam, TRANSPARENT)
		procSetTextColor.Call(wParam, colorInk)
		return app.brushWindow
	case WM_CTLCOLOREDIT, WM_CTLCOLORLISTBOX:
		procSetTextColor.Call(wParam, colorInk)
		procSetBkColor.Call(wParam, colorPanelBG)
		return app.brushPanel
	case WM_CTLCOLORBTN:
		procSetTextColor.Call(wParam, colorInk)
		procSetBkMode.Call(wParam, TRANSPARENT)
		return app.brushWindow
	case WM_COMMAND:
		id := int(loword(wParam))
		notify := hiword(wParam)
		if notify == LBN_SELCHANGE && id == ID_LC_LIST {
			showSelectedLearningPattern()
			return 0
		}
		if notify == BN_CLICKED {
			switch id {
			case ID_LC_PAUSE:
				toggleLearningPaused()
			case ID_LC_RESET:
				resetLearningFromCenter()
			case ID_LC_REFRESH:
				refreshLearningCenterItems(learningCenter.selected)
			case ID_LC_CLOSE:
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			}
		}
		return 0
	case WM_DESTROY:
		learningCenter = learningCenterWindowState{selected: -1}
		refreshLearningCenterBadge()
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func createLearningCenterUI(hwnd uintptr) {
	title := createControl("STATIC", "Local learning — observation only", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 18, 520, 24, hwnd, 0)
	setControlFont(title, app.headingFont)
	intro := createControl("STATIC", "DCM learns aggregated filename features, destinations and rename styles from completed moves. No file content is stored. Go 14 may offer an explainable suggestion, but nothing is applied without your explicit acceptance.", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 48, 1000, 42, hwnd, 0)
	setControlFont(intro, app.smallFont)

	learningCenter.list = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|WS_VSCROLL|WS_HSCROLL, 20, 98, 550, 480, hwnd, ID_LC_LIST)
	learningCenter.detail = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 585, 98, 445, 480, hwnd, 0)
	setExplorerTheme(learningCenter.list)
	setExplorerTheme(learningCenter.detail)

	learningCenter.pauseBtn = createControl("BUTTON", "Pause Learning", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 20, 594, 135, 30, hwnd, ID_LC_PAUSE)
	learningCenter.resetBtn = createControl("BUTTON", "Reset Learning…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 165, 594, 135, 30, hwnd, ID_LC_RESET)
	learningCenter.refreshBtn = createControl("BUTTON", "Refresh", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 310, 594, 105, 30, hwnd, ID_LC_REFRESH)
	learningCenter.closeBtn = createControl("BUTTON", "Close", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 925, 594, 105, 30, hwnd, ID_LC_CLOSE)
	for _, h := range []uintptr{learningCenter.pauseBtn, learningCenter.resetBtn, learningCenter.refreshBtn, learningCenter.closeBtn} {
		setExplorerTheme(h)
	}
	learningCenter.statusLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 642, 1010, 28, hwnd, 0)
	setControlFont(learningCenter.statusLabel, app.smallFont)
}

func learningPatternLabel(item LearningPatternItem) string {
	kind := "DEST"
	if item.Kind == LearningPatternRename {
		kind = "RENAME"
	}
	return fmt.Sprintf("%-6s %3.0f%%  %d/%d   %s  →  %s", kind, item.Confidence*100, item.Count, item.Total, item.Feature, item.Choice)
}

func refreshLearningCenterItems(preferred int) {
	if learningCenter.hwnd == 0 {
		return
	}
	items, state, err := loadLearningCenterItems(learningFilePath())
	if err != nil {
		learningCenter.items = nil
		learningCenter.state = LearningCenterState{}
		procSendMessageW.Call(learningCenter.list, LB_RESETCONTENT, 0, 0)
		setText(learningCenter.detail, "Learning data could not be read.\r\n\r\n"+err.Error()+"\r\n\r\nDCM will not overwrite a corrupt learning file automatically. Reset Learning is available only after you decide to discard it manually.")
		setText(learningCenter.statusLabel, "Learning unavailable: "+err.Error())
		procEnableWindow.Call(learningCenter.pauseBtn, 0)
		procEnableWindow.Call(learningCenter.resetBtn, 0)
		refreshLearningCenterBadge()
		return
	}
	learningCenter.items = items
	learningCenter.state = state
	procEnableWindow.Call(learningCenter.pauseBtn, 1)
	procEnableWindow.Call(learningCenter.resetBtn, 1)
	procSendMessageW.Call(learningCenter.list, LB_RESETCONTENT, 0, 0)
	limit := len(items)
	if limit > 500 {
		limit = 500
	}
	for _, item := range items[:limit] {
		procSendMessageW.Call(learningCenter.list, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(learningPatternLabel(item)))))
	}
	if state.Paused {
		setText(learningCenter.pauseBtn, "Resume Learning")
	} else {
		setText(learningCenter.pauseBtn, "Pause Learning")
	}
	status := fmt.Sprintf("%d completed move observation(s) · %d learned pattern rows", state.TotalObservations, len(items))
	if state.Paused {
		status += " · learning is paused"
	}
	if len(items) > limit {
		status += fmt.Sprintf(" · showing top %d", limit)
	}
	setText(learningCenter.statusLabel, status)
	if limit == 0 {
		learningCenter.selected = -1
		setText(learningCenter.detail, "No learning observations yet.\r\n\r\nSort files normally. After a successful Move or Replace, DCM records only aggregate filename features, the chosen destination and detected rename style. Exact-duplicate cleanup is not used as training data.")
	} else {
		if preferred < 0 || preferred >= limit {
			preferred = 0
		}
		learningCenter.selected = preferred
		procSendMessageW.Call(learningCenter.list, LB_SETCURSEL, uintptr(preferred), 0)
		showSelectedLearningPattern()
	}
	refreshLearningCenterBadge()
}

func showSelectedLearningPattern() {
	if learningCenter.list == 0 {
		return
	}
	idx, _, _ := procSendMessageW.Call(learningCenter.list, LB_GETCURSEL, 0, 0)
	if idx == LB_ERR || int(idx) < 0 || int(idx) >= len(learningCenter.items) {
		return
	}
	learningCenter.selected = int(idx)
	item := learningCenter.items[int(idx)]
	kind := "Destination"
	if item.Kind == LearningPatternRename {
		kind = "Rename style"
	}
	detail := fmt.Sprintf("Pattern type: %s\r\n\r\nFeature: %s\r\n\r\nLearned choice: %s\r\n\r\nObserved: %d of %d matching completed moves\r\nConfidence: %.1f%%\r\n\r\nThis is local observation data. Go 14 may surface a suggestion from qualifying evidence, but applying it still requires explicit acceptance and does not move a file.", kind, item.Feature, item.Choice, item.Count, item.Total, item.Confidence*100)
	setText(learningCenter.detail, detail)
}

func toggleLearningPaused() {
	paused := !learningCenter.state.Paused
	if err := setLearningPaused(learningFilePath(), paused, time.Now()); err != nil {
		procMessageBoxW.Call(learningCenter.hwnd, uintptr(unsafe.Pointer(wstr("Could not update learning state:\n"+err.Error()))), uintptr(unsafe.Pointer(wstr("Learning Center — DCM - Quick Sort"))), 0x10)
		return
	}
	refreshLearningCenterItems(learningCenter.selected)
}

func resetLearningFromCenter() {
	result, _, _ := procMessageBoxW.Call(learningCenter.hwnd, uintptr(unsafe.Pointer(wstr("Reset all locally learned destination and rename patterns?\n\nThis does not affect your files, destinations, move journal, or recovery data."))), uintptr(unsafe.Pointer(wstr("Reset Learning — DCM - Quick Sort"))), 0x04|0x30)
	if int(result) != 6 {
		return
	} // IDYES
	if err := resetLearningModel(learningFilePath(), time.Now()); err != nil {
		procMessageBoxW.Call(learningCenter.hwnd, uintptr(unsafe.Pointer(wstr("Could not reset learning data:\n"+err.Error()))), uintptr(unsafe.Pointer(wstr("Learning Center — DCM - Quick Sort"))), 0x10)
		return
	}
	refreshLearningCenterItems(-1)
	refreshSuggestionForCurrent()
}

func refreshLearningCenterBadge() {
	if app.learningBtn == 0 {
		return
	}
	model, err := loadLearningModel(learningFilePath())
	if err != nil {
		setText(app.learningBtn, "Learning !")
		return
	}
	if model.Paused {
		setText(app.learningBtn, "Learning paused")
	} else if model.TotalObservations > 0 {
		setText(app.learningBtn, "Learning")
	} else {
		setText(app.learningBtn, "Learning")
	}
}

func learningStatusAfterObservation(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "learning update failed"
	}
	return text
}
