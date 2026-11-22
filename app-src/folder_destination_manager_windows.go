//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

const (
	ID_FDM_NAME = 1601 + iota
	ID_FDM_FOLDER
	ID_FDM_BROWSE
	ID_FDM_SAVE
	ID_FDM_DELETE
	ID_FDM_LIST
	ID_FDM_NEW
	ID_FDM_USE
	ID_FDM_CLOSE
)

type folderDestinationManagerState struct {
	hwnd, nameEdit, folderEdit, browseBtn, saveBtn, deleteBtn, newBtn, useBtn, closeBtn, list, statusLabel uintptr
	selected                                                                                               int
}

var folderDestinationManager = folderDestinationManagerState{selected: -1}
var folderDestinationManagerWndProcCB = syscall.NewCallback(folderDestinationManagerWndProc)

func registerFolderDestinationManagerClass(hInst, cursor uintptr) bool {
	class := wstr("QuickSortFolderDestinations")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: folderDestinationManagerWndProcCB, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: class}
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	return r != 0
}

func folderDestinationManagerFocusOrder() []uintptr {
	return []uintptr{folderDestinationManager.nameEdit, folderDestinationManager.folderEdit, folderDestinationManager.browseBtn, folderDestinationManager.saveBtn, folderDestinationManager.newBtn, folderDestinationManager.deleteBtn, folderDestinationManager.useBtn, folderDestinationManager.list, folderDestinationManager.closeBtn}
}

func focusWithinFolderDestinationManager(focus uintptr) bool {
	if folderDestinationManager.hwnd == 0 {
		return false
	}
	for _, hwnd := range folderDestinationManagerFocusOrder() {
		if focus == hwnd {
			return true
		}
	}
	return false
}

func moveFolderDestinationManagerFocus(reverse bool) bool {
	order := folderDestinationManagerFocusOrder()
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

func destinationFolderStatus(alias DestinationAlias) string { return alias.Name+" → "+alias.Path }

func folderDestinationManagerListText(alias DestinationAlias) string { return alias.Name + "    →    " + alias.Path }

func populateFolderDestinationManagerList(selection int) {
	if folderDestinationManager.list == 0 {
		return
	}
	procSendMessageW.Call(folderDestinationManager.list, LB_RESETCONTENT, 0, 0)
	for _, alias := range app.destAliases {
		procSendMessageW.Call(folderDestinationManager.list, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(folderDestinationManagerListText(alias)))))
	}
	folderDestinationManager.selected = -1
	if selection >= 0 && selection < len(app.destAliases) {
		folderDestinationManager.selected = selection
		procSendMessageW.Call(folderDestinationManager.list, LB_SETCURSEL, uintptr(selection), 0)
	}
	flag := uintptr(0)
	if folderDestinationManager.selected >= 0 {
		flag = 1
	}
	procEnableWindow.Call(folderDestinationManager.deleteBtn, flag)
	procEnableWindow.Call(folderDestinationManager.useBtn, flag)
}

func clearFolderDestinationManagerForm() {
	folderDestinationManager.selected = -1
	setText(folderDestinationManager.nameEdit, "")
	setText(folderDestinationManager.folderEdit, "")
	procSendMessageW.Call(folderDestinationManager.list, LB_SETCURSEL, ^uintptr(0), 0)
	procEnableWindow.Call(folderDestinationManager.deleteBtn, 0)
	procEnableWindow.Call(folderDestinationManager.useBtn, 0)
	setText(folderDestinationManager.statusLabel, "Enter a name, choose a folder, then Save Destination.")
	procSetFocus.Call(folderDestinationManager.nameEdit)
}

func loadFolderDestinationManagerSelection(index int) {
	if index < 0 || index >= len(app.destAliases) {
		clearFolderDestinationManagerForm()
		return
	}
	folderDestinationManager.selected = index
	alias := app.destAliases[index]
	setText(folderDestinationManager.nameEdit, alias.Name)
	setText(folderDestinationManager.folderEdit, alias.Path)
	procEnableWindow.Call(folderDestinationManager.deleteBtn, 1)
	procEnableWindow.Call(folderDestinationManager.useBtn, 1)
	setText(folderDestinationManager.statusLabel, folderDestinationManagerListText(alias))
}

func saveFolderDestinationManagerAlias() {
	alias := DestinationAlias{Name: strings.TrimSpace(getText(folderDestinationManager.nameEdit)), Path: strings.TrimSpace(getText(folderDestinationManager.folderEdit))}
	if err := validateDestinationAlias(alias); err != nil {
		setText(folderDestinationManager.statusLabel, err.Error())
		return
	}
	clean, folderErr := resolveExistingDestinationFolder(alias.Path)
	if folderErr != nil {
		setText(folderDestinationManager.statusLabel, folderErr.Error())
		return
	}
	alias.Path=clean
	
	aliases := append([]DestinationAlias(nil), app.destAliases...)
	selection := folderDestinationManager.selected
	var idx int
	var err error
	if selection >= 0 && selection < len(aliases) && strings.EqualFold(strings.TrimSpace(aliases[selection].Name), alias.Name) {
		aliases[selection] = alias
		idx = selection
	} else {
		if selection >= 0 && selection < len(aliases) {
			aliases = deleteDestinationAlias(aliases, selection)
		}
		aliases, idx, err = upsertDestinationAlias(aliases, alias)
	}
	if err != nil {
		setText(folderDestinationManager.statusLabel, err.Error())
		return
	}
	if err := writeDestinationAliases(aliases); err != nil {
		setText(folderDestinationManager.statusLabel, "Could not save destination: "+err.Error())
		return
	}
	app.destAliases = aliases
	populateDestinationAliases()
	populateFolderDestinationManagerList(idx)
	loadFolderDestinationManagerSelection(idx)
	backupNote := ""
	if _, e := backupCurrentConfiguration("destination-alias-saved"); e != nil {
		backupNote = " — backup warning: " + e.Error()
	}
	setText(folderDestinationManager.statusLabel, "Saved "+destinationFolderStatus(alias)+backupNote)
	setText(app.statusLabel, "Destination saved: "+destinationFolderStatus(alias)+backupNote)
}

func deleteFolderDestinationManagerAlias() {
	idx := folderDestinationManager.selected
	if idx < 0 || idx >= len(app.destAliases) {
		setText(folderDestinationManager.statusLabel, "Select a saved destination first.")
		return
	}
	alias := app.destAliases[idx]
	result, _, _ := procMessageBoxW.Call(folderDestinationManager.hwnd, uintptr(unsafe.Pointer(wstr("Remove destination alias '"+alias.Name+"'?\n\nThe folder and its files will not be deleted."))), uintptr(unsafe.Pointer(wstr("DCM - Quick Sort"))), 0x04|0x20)
	if result != 6 {
		return
	}
	aliases := deleteDestinationAlias(app.destAliases, idx)
	if err := writeDestinationAliases(aliases); err != nil {
		setText(folderDestinationManager.statusLabel, "Could not remove destination: "+err.Error())
		return
	}
	app.destAliases = aliases
	populateDestinationAliases()
	populateFolderDestinationManagerList(-1)
	clearFolderDestinationManagerForm()
	_, _ = backupCurrentConfiguration("destination-alias-removed")
	setText(app.statusLabel, "Destination shortcut removed: "+alias.Name)
}

func useSelectedFolderDestinationManagerAlias() {
	idx := folderDestinationManager.selected
	if idx < 0 || idx >= len(app.destAliases) {
		setText(folderDestinationManager.statusLabel, "Select a saved destination first.")
		return
	}
	alias := app.destAliases[idx]
	if activateDestinationFolder(alias.Path) {
		setText(folderDestinationManager.statusLabel, "Using "+destinationFolderStatus(alias))
	}
}

func createFolderDestinationManagerUI(hwnd uintptr) {
	folderDestinationManager.selected = -1
	createControl("STATIC", "Name:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 24, 70, 22, hwnd, 0)
	folderDestinationManager.nameEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 95, 20, 455, 28, hwnd, ID_FDM_NAME)
	createControl("STATIC", "Folder:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 64, 70, 22, hwnd, 0)
	folderDestinationManager.folderEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL|ES_READONLY, 95, 60, 455, 28, hwnd, ID_FDM_FOLDER)
	folderDestinationManager.browseBtn = createControl("BUTTON", "&Browse…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 565, 59, 120, 30, hwnd, ID_FDM_BROWSE)
	folderDestinationManager.saveBtn = createControl("BUTTON", "&Save Destination", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 95, 104, 135, 31, hwnd, ID_FDM_SAVE)
	folderDestinationManager.newBtn = createControl("BUTTON", "&New", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 240, 104, 70, 31, hwnd, ID_FDM_NEW)
	folderDestinationManager.deleteBtn = createControl("BUTTON", "&Delete Selected", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 320, 104, 135, 31, hwnd, ID_FDM_DELETE)
	folderDestinationManager.useBtn = createControl("BUTTON", "&Use Destination", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 465, 104, 110, 31, hwnd, ID_FDM_USE)
	folderDestinationManager.closeBtn = createControl("BUTTON", "Close", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 580, 104, 105, 31, hwnd, ID_FDM_CLOSE)
	folderDestinationManager.statusLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 148, 665, 24, hwnd, 0)
	folderDestinationManager.list = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|WS_VSCROLL, 20, 180, 665, 270, hwnd, ID_FDM_LIST)
	for _, c := range []uintptr{folderDestinationManager.nameEdit, folderDestinationManager.folderEdit, folderDestinationManager.browseBtn, folderDestinationManager.saveBtn, folderDestinationManager.newBtn, folderDestinationManager.deleteBtn, folderDestinationManager.useBtn, folderDestinationManager.closeBtn, folderDestinationManager.list} {
		setExplorerTheme(c)
	}
	populateFolderDestinationManagerList(-1)
	clearFolderDestinationManagerForm()
}

func folderDestinationManagerWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		folderDestinationManager.hwnd = hwnd
		createFolderDestinationManagerUI(hwnd)
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
			case ID_FDM_BROWSE:
				if p := browseFolder(hwnd, "Choose destination folder", strings.TrimSpace(getText(folderDestinationManager.folderEdit))); p != "" {
					setText(folderDestinationManager.folderEdit, p)
					setText(folderDestinationManager.statusLabel, "Folder selected: "+p)
				}
			case ID_FDM_SAVE:
				saveFolderDestinationManagerAlias()
			case ID_FDM_NEW:
				clearFolderDestinationManagerForm()
			case ID_FDM_DELETE:
				deleteFolderDestinationManagerAlias()
			case ID_FDM_USE:
				useSelectedFolderDestinationManagerAlias()
			case ID_FDM_CLOSE:
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			}
		}
		if notify == LBN_SELCHANGE && id == ID_FDM_LIST {
			if idx, _, _ := procSendMessageW.Call(folderDestinationManager.list, LB_GETCURSEL, 0, 0); idx != LB_ERR {
				loadFolderDestinationManagerSelection(int(idx))
			}
		}
		return 0
	case WM_DESTROY:
		folderDestinationManager = folderDestinationManagerState{selected: -1}
		reloadDestinationAliases()
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func openFolderDestinationManager() {
	if folderDestinationManager.hwnd != 0 {
		procShowWindow.Call(folderDestinationManager.hwnd, SW_SHOW)
		procSetFocus.Call(folderDestinationManager.nameEdit)
		return
	}
	reloadDestinationAliases()
	hInst, _, _ := procGetModuleHandleW.Call(0)
	style := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE | WS_CLIPCHILDREN)
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(wstr("QuickSortFolderDestinations"))), uintptr(unsafe.Pointer(wstr("Manage Destinations — DCM - Quick Sort"))), style, CW_USEDEFAULT, CW_USEDEFAULT, 725, 520, app.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		showError("Could not open Manage Destinations.")
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
}
