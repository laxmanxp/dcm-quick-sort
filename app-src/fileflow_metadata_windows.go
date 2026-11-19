//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

type metadataPromptKind uint8

const (
	metadataPromptTag metadataPromptKind = iota + 1
	metadataPromptTodo
	metadataPromptNote
)

const (
	ID_FFM_VALUE = 1601 + iota
	ID_FFM_ADD
	ID_FFM_CANCEL
)

type fileFlowMetadataPromptState struct {
	hwnd, valueEdit, addBtn, cancelBtn, statusLabel uintptr
	kind                                            metadataPromptKind
}

var fileFlowMetadataPrompt fileFlowMetadataPromptState
var fileFlowMetadataPromptWndProcCB = syscall.NewCallback(fileFlowMetadataPromptWndProc)

func registerFileFlowMetadataPromptClass(hInst, cursor uintptr) bool {
	class := wstr("QuickSortFileFlowMetadata")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: fileFlowMetadataPromptWndProcCB, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: class}
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	return r != 0
}

func metadataPromptTitle(kind metadataPromptKind) (string, string, string) {
	switch kind {
	case metadataPromptTodo:
		return "Add Todo — DCM - Quick Sort", "Todo:", "Stage Todo"
	case metadataPromptNote:
		return "Add Note — DCM - Quick Sort", "Note:", "Stage Note"
	default:
		return "Add Tag — DCM - Quick Sort", "Tag:", "Stage Tag"
	}
}

func createFileFlowMetadataPromptUI(hwnd uintptr) {
	_, label, button := metadataPromptTitle(fileFlowMetadataPrompt.kind)
	createControl("STATIC", label, WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 24, 70, 22, hwnd, 0)
	style := uint32(WS_CHILD | WS_VISIBLE | WS_BORDER | WS_TABSTOP | ES_AUTOHSCROLL)
	height := int32(30)
	if fileFlowMetadataPrompt.kind == metadataPromptNote {
		style = uint32(WS_CHILD | WS_VISIBLE | WS_BORDER | WS_TABSTOP | ES_MULTILINE | ES_AUTOVSCROLL | WS_VSCROLL)
		height = 92
	}
	fileFlowMetadataPrompt.valueEdit = createControl("EDIT", "", style, 90, 20, 390, height, hwnd, ID_FFM_VALUE)
	by := int32(64)
	if fileFlowMetadataPrompt.kind == metadataPromptNote {
		by = 126
	}
	fileFlowMetadataPrompt.addBtn = createControl("BUTTON", button, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 250, by, 110, 31, hwnd, ID_FFM_ADD)
	fileFlowMetadataPrompt.cancelBtn = createControl("BUTTON", "Cancel", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 370, by, 110, 31, hwnd, ID_FFM_CANCEL)
	fileFlowMetadataPrompt.statusLabel = createControl("STATIC", "Staged metadata is saved locally until Update + Move succeeds.", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, by+42, 460, 36, hwnd, 0)
	for _, c := range []uintptr{fileFlowMetadataPrompt.valueEdit, fileFlowMetadataPrompt.addBtn, fileFlowMetadataPrompt.cancelBtn} {
		setExplorerTheme(c)
	}
	procSetFocus.Call(fileFlowMetadataPrompt.valueEdit)
}

func stageFileFlowMetadataValue(kind metadataPromptKind, value string) error {
	source := currentFile()
	if source == "" {
		return fmt.Errorf("no source file is selected")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("enter a value first")
	}
	draft, _, err := loadFileFlowDraft(fileFlowDraftStorePath(), source)
	if err != nil {
		return err
	}
	switch kind {
	case metadataPromptTodo:
		draft.AddTodo(value)
	case metadataPromptNote:
		draft.AddNote(value)
	default:
		draft.AddTag(value)
	}
	if err := saveFileFlowDraft(fileFlowDraftStorePath(), source, draft); err != nil {
		return err
	}
	refreshFileFlowMetadataControls()
	setText(app.statusLabel, fmt.Sprintf("FileFlow metadata staged: %d tag(s), %d todo(s), %d note(s). Use Update + Move to commit it with this file.", len(draft.Tags), len(draft.Todos), len(draft.Notes)))
	return nil
}

func fileFlowMetadataPromptWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		fileFlowMetadataPrompt.hwnd = hwnd
		createFileFlowMetadataPromptUI(hwnd)
		showKeyboardCues(hwnd)
		return 0
	case WM_CTLCOLORSTATIC:
		procSetBkMode.Call(wParam, TRANSPARENT)
		procSetTextColor.Call(wParam, colorInk)
		return app.brushWindow
	case WM_CTLCOLOREDIT:
		procSetTextColor.Call(wParam, colorInk)
		procSetBkColor.Call(wParam, colorPanelBG)
		return app.brushPanel
	case WM_COMMAND:
		if hiword(wParam) == BN_CLICKED {
			switch int(loword(wParam)) {
			case ID_FFM_ADD:
				if err := stageFileFlowMetadataValue(fileFlowMetadataPrompt.kind, getText(fileFlowMetadataPrompt.valueEdit)); err != nil {
					setText(fileFlowMetadataPrompt.statusLabel, err.Error())
					return 0
				}
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			case ID_FFM_CANCEL:
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			}
		}
		return 0
	case WM_DESTROY:
		fileFlowMetadataPrompt = fileFlowMetadataPromptState{}
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func openFileFlowMetadataPrompt(kind metadataPromptKind) {
	if currentFile() == "" {
		setText(app.statusLabel, "Select a source file before adding FileFlow metadata.")
		return
	}
	if fileFlowMetadataPrompt.hwnd != 0 {
		procPostMessageW.Call(fileFlowMetadataPrompt.hwnd, WM_CLOSE, 0, 0)
	}
	fileFlowMetadataPrompt.kind = kind
	title, _, _ := metadataPromptTitle(kind)
	hInst, _, _ := procGetModuleHandleW.Call(0)
	height := int32(205)
	if kind == metadataPromptNote {
		height = 270
	}
	style := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE | WS_CLIPCHILDREN)
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(wstr("QuickSortFileFlowMetadata"))), uintptr(unsafe.Pointer(wstr(title))), style, CW_USEDEFAULT, CW_USEDEFAULT, 520, uintptr(height), app.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		showError("Could not open FileFlow metadata editor.")
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
}
