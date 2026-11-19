//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

const (
	ID_SM_NAME = 1201 + iota
	ID_SM_FOLDER
	ID_SM_BROWSE
	ID_SM_SAVE
	ID_SM_DELETE
	ID_SM_LIST
	ID_SM_NEW
	ID_SM_USE
	ID_SM_CLOSE
)

type sourceManagerState struct {
	hwnd, nameEdit, folderEdit, browseBtn, saveBtn, deleteBtn, newBtn, useBtn, closeBtn, list, statusLabel uintptr
	selected                                                                                               int
}

var sourceManager = sourceManagerState{selected: -1}
var sourceManagerWndProcCB = syscall.NewCallback(sourceManagerWndProc)

func registerSourceManagerClass(hInst, cursor uintptr) bool {
	class := wstr("QuickSortSources")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: sourceManagerWndProcCB, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: class}
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	return r != 0
}

func sourceManagerFocusOrder() []uintptr {
	return []uintptr{sourceManager.nameEdit, sourceManager.folderEdit, sourceManager.browseBtn, sourceManager.saveBtn, sourceManager.newBtn, sourceManager.deleteBtn, sourceManager.useBtn, sourceManager.list, sourceManager.closeBtn}
}

func focusWithinSourceManager(focus uintptr) bool {
	if sourceManager.hwnd == 0 {
		return false
	}
	for _, hwnd := range sourceManagerFocusOrder() {
		if focus == hwnd {
			return true
		}
	}
	return false
}

func moveSourceManagerFocus(reverse bool) bool {
	order := sourceManagerFocusOrder()
	focus, _, _ := procGetFocus.Call()
	current := -1
	enabled := make([]bool, len(order))
	for i, hwnd := range order {
		enabled[i] = focusableControl(hwnd)
		if hwnd == focus {
			current = i
		}
	}
	next := nextEnabledIndex(current, enabled, reverse)
	if next < 0 {
		return false
	}
	procSetFocus.Call(order[next])
	return true
}

func sourceManagerListText(alias SourceAlias) string { return alias.Name + "    →    " + alias.Path }

func populateSourceManagerList(selection int) {
	if sourceManager.list == 0 {
		return
	}
	procSendMessageW.Call(sourceManager.list, LB_RESETCONTENT, 0, 0)
	for _, alias := range app.sourceAliases {
		procSendMessageW.Call(sourceManager.list, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(sourceManagerListText(alias)))))
	}
	sourceManager.selected = -1
	if selection >= 0 && selection < len(app.sourceAliases) {
		sourceManager.selected = selection
		procSendMessageW.Call(sourceManager.list, LB_SETCURSEL, uintptr(selection), 0)
	}
	flag := uintptr(0)
	if sourceManager.selected >= 0 {
		flag = 1
	}
	procEnableWindow.Call(sourceManager.deleteBtn, flag)
	procEnableWindow.Call(sourceManager.useBtn, flag)
}

func clearSourceManagerForm() {
	sourceManager.selected = -1
	setText(sourceManager.nameEdit, "")
	setText(sourceManager.folderEdit, "")
	procSendMessageW.Call(sourceManager.list, LB_SETCURSEL, ^uintptr(0), 0)
	procEnableWindow.Call(sourceManager.deleteBtn, 0)
	procEnableWindow.Call(sourceManager.useBtn, 0)
	setText(sourceManager.statusLabel, "Enter a name, choose a folder, then Save Source.")
	procSetFocus.Call(sourceManager.nameEdit)
}

func loadSourceManagerSelection(index int) {
	if index < 0 || index >= len(app.sourceAliases) {
		clearSourceManagerForm()
		return
	}
	sourceManager.selected = index
	alias := app.sourceAliases[index]
	setText(sourceManager.nameEdit, alias.Name)
	setText(sourceManager.folderEdit, alias.Path)
	procEnableWindow.Call(sourceManager.deleteBtn, 1)
	procEnableWindow.Call(sourceManager.useBtn, 1)
	setText(sourceManager.statusLabel, sourceManagerListText(alias))
}

func saveSourceManagerAlias() {
	alias := SourceAlias{Name: strings.TrimSpace(getText(sourceManager.nameEdit)), Path: strings.TrimSpace(getText(sourceManager.folderEdit))}
	if err := validateSourceAlias(alias); err != nil {
		setText(sourceManager.statusLabel, err.Error())
		return
	}
	aliases := append([]SourceAlias(nil), app.sourceAliases...)
	selection := sourceManager.selected
	var idx int
	var err error
	if selection >= 0 && selection < len(aliases) && strings.EqualFold(strings.TrimSpace(aliases[selection].Name), alias.Name) {
		aliases[selection] = alias
		idx = selection
	} else {
		if selection >= 0 && selection < len(aliases) {
			aliases = deleteSourceAlias(aliases, selection)
		}
		aliases, idx, err = upsertSourceAlias(aliases, alias)
	}
	if err != nil {
		setText(sourceManager.statusLabel, err.Error())
		return
	}
	if err := writeSourceAliases(aliases); err != nil {
		setText(sourceManager.statusLabel, "Could not save source: "+err.Error())
		return
	}
	app.sourceAliases = aliases
	populateSourceAliases()
	populateSourceManagerList(idx)
	loadSourceManagerSelection(idx)
	backupNote := ""
	if _, e := backupCurrentConfiguration("source-alias-saved"); e != nil {
		backupNote = " — backup warning: " + e.Error()
	}
	setText(sourceManager.statusLabel, "Saved "+sourceStatus(alias)+backupNote)
	setText(app.statusLabel, "Source saved: "+sourceStatus(alias)+backupNote)
}

func deleteSourceManagerAlias() {
	idx := sourceManager.selected
	if idx < 0 || idx >= len(app.sourceAliases) {
		setText(sourceManager.statusLabel, "Select a saved source first.")
		return
	}
	alias := app.sourceAliases[idx]
	result, _, _ := procMessageBoxW.Call(sourceManager.hwnd, uintptr(unsafe.Pointer(wstr("Remove source alias '"+alias.Name+"'?\n\nThe folder and its files will not be deleted."))), uintptr(unsafe.Pointer(wstr("DCM - Quick Sort"))), 0x04|0x20)
	if result != 6 {
		return
	}
	aliases := deleteSourceAlias(app.sourceAliases, idx)
	if err := writeSourceAliases(aliases); err != nil {
		setText(sourceManager.statusLabel, "Could not remove source: "+err.Error())
		return
	}
	app.sourceAliases = aliases
	populateSourceAliases()
	populateSourceManagerList(-1)
	clearSourceManagerForm()
	_, _ = backupCurrentConfiguration("source-alias-removed")
	setText(app.statusLabel, "Source alias removed: "+alias.Name)
}

func useSelectedSourceManagerAlias() {
	idx := sourceManager.selected
	if idx < 0 || idx >= len(app.sourceAliases) {
		setText(sourceManager.statusLabel, "Select a saved source first.")
		return
	}
	alias := app.sourceAliases[idx]
	if activateSourcePath(alias.Path) {
		setText(sourceManager.statusLabel, "Using "+sourceStatus(alias))
	}
}

func createSourceManagerUI(hwnd uintptr) {
	sourceManager.selected = -1
	createControl("STATIC", "Name:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 24, 70, 22, hwnd, 0)
	sourceManager.nameEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 95, 20, 455, 28, hwnd, ID_SM_NAME)
	createControl("STATIC", "Folder:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 64, 70, 22, hwnd, 0)
	sourceManager.folderEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL|ES_READONLY, 95, 60, 455, 28, hwnd, ID_SM_FOLDER)
	sourceManager.browseBtn = createControl("BUTTON", "&Browse…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 565, 59, 120, 30, hwnd, ID_SM_BROWSE)
	sourceManager.saveBtn = createControl("BUTTON", "&Save Source", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 95, 104, 115, 31, hwnd, ID_SM_SAVE)
	sourceManager.newBtn = createControl("BUTTON", "&New", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 220, 104, 90, 31, hwnd, ID_SM_NEW)
	sourceManager.deleteBtn = createControl("BUTTON", "&Delete Selected", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 320, 104, 135, 31, hwnd, ID_SM_DELETE)
	sourceManager.useBtn = createControl("BUTTON", "&Use Source", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 465, 104, 105, 31, hwnd, ID_SM_USE)
	sourceManager.closeBtn = createControl("BUTTON", "Close", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 580, 104, 105, 31, hwnd, ID_SM_CLOSE)
	sourceManager.statusLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 148, 665, 24, hwnd, 0)
	sourceManager.list = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|WS_VSCROLL, 20, 180, 665, 270, hwnd, ID_SM_LIST)
	for _, c := range []uintptr{sourceManager.nameEdit, sourceManager.folderEdit, sourceManager.browseBtn, sourceManager.saveBtn, sourceManager.newBtn, sourceManager.deleteBtn, sourceManager.useBtn, sourceManager.closeBtn, sourceManager.list} {
		setExplorerTheme(c)
	}
	populateSourceManagerList(-1)
	clearSourceManagerForm()
}

func sourceManagerWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		sourceManager.hwnd = hwnd
		createSourceManagerUI(hwnd)
		showKeyboardCues(hwnd)
		return 0
	case WM_CTLCOLORSTATIC:
		procSetBkMode.Call(wParam, TRANSPARENT)
		procSetTextColor.Call(wParam, colorInk)
		return app.brushWindow
	case WM_CTLCOLOREDIT, WM_CTLCOLORLISTBOX:
		procSetTextColor.Call(wParam, colorInk)
		procSetBkColor.Call(wParam, colorPanelBG)
		return app.brushPanel
	case WM_COMMAND:
		id := int(loword(wParam))
		notify := hiword(wParam)
		if notify == BN_CLICKED {
			switch id {
			case ID_SM_BROWSE:
				if p := browseFolder(hwnd, "Choose source folder", strings.TrimSpace(getText(sourceManager.folderEdit))); p != "" {
					setText(sourceManager.folderEdit, p)
					setText(sourceManager.statusLabel, "Folder selected: "+p)
				}
			case ID_SM_SAVE:
				saveSourceManagerAlias()
			case ID_SM_NEW:
				clearSourceManagerForm()
			case ID_SM_DELETE:
				deleteSourceManagerAlias()
			case ID_SM_USE:
				useSelectedSourceManagerAlias()
			case ID_SM_CLOSE:
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			}
		}
		if notify == LBN_SELCHANGE && id == ID_SM_LIST {
			if idx, _, _ := procSendMessageW.Call(sourceManager.list, LB_GETCURSEL, 0, 0); idx != LB_ERR {
				loadSourceManagerSelection(int(idx))
			}
		}
		return 0
	case WM_DESTROY:
		sourceManager = sourceManagerState{selected: -1}
		reloadSourceAliases()
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func openSourceManager() {
	if sourceManager.hwnd != 0 {
		procShowWindow.Call(sourceManager.hwnd, SW_SHOW)
		procSetFocus.Call(sourceManager.nameEdit)
		return
	}
	reloadSourceAliases()
	hInst, _, _ := procGetModuleHandleW.Call(0)
	style := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE | WS_CLIPCHILDREN)
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(wstr("QuickSortSources"))), uintptr(unsafe.Pointer(wstr("Manage Sources — DCM - Quick Sort"))), style, CW_USEDEFAULT, CW_USEDEFAULT, 725, 520, app.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		showError("Could not open Manage Sources.")
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
}
