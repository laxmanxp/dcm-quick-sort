//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	ID_SC_APPLY_DEST = 1401
	ID_SC_APPLY_NAME = 1402
	ID_SC_APPLY_BOTH = 1403
	ID_SC_DISMISS    = 1404
	ID_SC_CLOSE      = 1405
)

type suggestionCenterWindowState struct {
	hwnd, detail, statusLabel                uintptr
	applyDestBtn, applyNameBtn, applyBothBtn uintptr
	dismissBtn, closeBtn                     uintptr
	source                                   string
	suggestion                               LearnedSuggestion
}

var suggestionCenter suggestionCenterWindowState
var suggestionCenterWndProcCB = syscall.NewCallback(suggestionCenterWndProc)

func registerSuggestionCenterClass(hInst, cursor uintptr) bool {
	class := wstr("QuickSortSuggestionCenter")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: suggestionCenterWndProcCB, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: class}
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	return r != 0
}

func openSuggestionCenter() {
	if suggestionCenter.hwnd != 0 {
		procShowWindow.Call(suggestionCenter.hwnd, SW_SHOW)
		refreshSuggestionCenter()
		return
	}
	if !suggestionAvailable(app.currentSuggestion) {
		setText(app.statusLabel, "No learned suggestion has enough evidence for this file yet.")
		return
	}
	hInst, _, _ := procGetModuleHandleW.Call(0)
	style := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE | WS_CLIPCHILDREN)
	hwnd, _, _ := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(wstr("QuickSortSuggestionCenter"))),
		uintptr(unsafe.Pointer(wstr("Suggestion — DCM - Quick Sort"))),
		style, CW_USEDEFAULT, CW_USEDEFAULT, 820, 650, app.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		showError("Could not open Suggestion Center.")
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
}

func suggestionCenterWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		suggestionCenter.hwnd = hwnd
		styleTitleBar(hwnd)
		createSuggestionCenterUI(hwnd)
		refreshSuggestionCenter()
		return 0
	case WM_CTLCOLORSTATIC:
		procSetBkMode.Call(wParam, TRANSPARENT)
		procSetTextColor.Call(wParam, colorInk)
		return app.brushWindow
	case WM_CTLCOLOREDIT:
		procSetTextColor.Call(wParam, colorInk)
		procSetBkColor.Call(wParam, colorPanelBG)
		return app.brushPanel
	case WM_CTLCOLORBTN:
		procSetTextColor.Call(wParam, colorInk)
		procSetBkMode.Call(wParam, TRANSPARENT)
		return app.brushWindow
	case WM_COMMAND:
		if hiword(wParam) == BN_CLICKED {
			switch int(loword(wParam)) {
			case ID_SC_APPLY_DEST:
				applySuggestionFromCenter(false, true)
			case ID_SC_APPLY_NAME:
				applySuggestionFromCenter(true, false)
			case ID_SC_APPLY_BOTH:
				applySuggestionFromCenter(true, true)
			case ID_SC_DISMISS:
				app.suggestionDismissedFor = suggestionCenter.source
				refreshSuggestionForCurrent()
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			case ID_SC_CLOSE:
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			}
		}
		return 0
	case WM_DESTROY:
		suggestionCenter = suggestionCenterWindowState{}
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func createSuggestionCenterUI(hwnd uintptr) {
	title := createControl("STATIC", "Learned suggestion — explicit acceptance only", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 18, 700, 24, hwnd, 0)
	setControlFont(title, app.headingFont)
	intro := createControl("STATIC", "DCM combines only local learning evidence. Applying a suggestion edits the Name/Destination fields; it never moves a file by itself.", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 48, 760, 38, hwnd, 0)
	setControlFont(intro, app.smallFont)
	suggestionCenter.detail = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 20, 94, 760, 410, hwnd, 0)
	setExplorerTheme(suggestionCenter.detail)

	suggestionCenter.applyDestBtn = createControl("BUTTON", "Apply Destination", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 20, 522, 145, 31, hwnd, ID_SC_APPLY_DEST)
	suggestionCenter.applyNameBtn = createControl("BUTTON", "Apply Name", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 175, 522, 125, 31, hwnd, ID_SC_APPLY_NAME)
	suggestionCenter.applyBothBtn = createControl("BUTTON", "Apply Both", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 310, 522, 125, 31, hwnd, ID_SC_APPLY_BOTH)
	suggestionCenter.dismissBtn = createControl("BUTTON", "Dismiss for this file", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 445, 522, 155, 31, hwnd, ID_SC_DISMISS)
	suggestionCenter.closeBtn = createControl("BUTTON", "Close", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 675, 522, 105, 31, hwnd, ID_SC_CLOSE)
	for _, h := range []uintptr{suggestionCenter.applyDestBtn, suggestionCenter.applyNameBtn, suggestionCenter.applyBothBtn, suggestionCenter.dismissBtn, suggestionCenter.closeBtn} {
		setExplorerTheme(h)
	}
	suggestionCenter.statusLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 570, 760, 35, hwnd, 0)
	setControlFont(suggestionCenter.statusLabel, app.smallFont)
}

func suggestionEvidenceText(items []SuggestionEvidence) string {
	if len(items) == 0 {
		return "  no qualifying evidence"
	}
	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "  • %s: %d/%d (%.0f%%)\r\n", item.Feature, item.Count, item.Total, item.Confidence*100)
	}
	return strings.TrimRight(b.String(), "\r\n")
}

func refreshSuggestionCenter() {
	if suggestionCenter.hwnd == 0 {
		return
	}
	source := currentFile()
	suggestionCenter.source = source
	if source == "" {
		suggestionCenter.suggestion = LearnedSuggestion{}
		setText(suggestionCenter.detail, "No file is currently selected.")
		setSuggestionCenterApplyAvailability(false, false)
		return
	}
	suggestion, err := loadLearnedSuggestion(learningFilePath(), source)
	if err != nil {
		suggestionCenter.suggestion = LearnedSuggestion{}
		setText(suggestionCenter.detail, "Learning data could not be read:\r\n\r\n"+err.Error())
		setSuggestionCenterApplyAvailability(false, false)
		return
	}
	suggestionCenter.suggestion = suggestion
	if !suggestionAvailable(suggestion) {
		setText(suggestionCenter.detail, "No learned pattern currently clears the minimum evidence threshold for this file. DCM stays silent instead of guessing.")
		setSuggestionCenterApplyAvailability(false, false)
		return
	}
	destText := "None"
	if suggestion.Destination != "" {
		destText = fmt.Sprintf("%s\r\nConfidence: %.1f%% · strongest support: %d\r\nReasons:\r\n%s", suggestion.Destination, suggestion.DestinationConfidence*100, suggestion.DestinationSupport, suggestionEvidenceText(suggestion.DestinationReasons))
	}
	renameText := "None"
	if suggestion.RenameStyle != "" {
		renameText = fmt.Sprintf("%s\r\nConfidence: %.1f%% · strongest support: %d\r\nReasons:\r\n%s", suggestion.RenameStyle, suggestion.RenameConfidence*100, suggestion.RenameSupport, suggestionEvidenceText(suggestion.RenameReasons))
	}
	detail := fmt.Sprintf("File:\r\n%s\r\n\r\nDESTINATION SUGGESTION\r\n%s\r\n\r\nRENAME SUGGESTION\r\n%s\r\n\r\nApplying here only edits the proposal fields in the main window. The actual filesystem change still requires Rename + Move + Next.", source, destText, renameText)
	setText(suggestionCenter.detail, detail)
	setSuggestionCenterApplyAvailability(suggestion.RenameStyle != "", suggestion.Destination != "")
	setText(suggestionCenter.statusLabel, "Suggestion is derived from local aggregate history only. No file content or cloud service is used.")
}

func setSuggestionCenterApplyAvailability(nameOK, destOK bool) {
	boolFlag := func(v bool) uintptr {
		if v {
			return 1
		}
		return 0
	}
	procEnableWindow.Call(suggestionCenter.applyNameBtn, boolFlag(nameOK))
	procEnableWindow.Call(suggestionCenter.applyDestBtn, boolFlag(destOK))
	procEnableWindow.Call(suggestionCenter.applyBothBtn, boolFlag(nameOK || destOK))
}

func applySuggestionFromCenter(applyName, applyDestination bool) {
	if currentFile() == "" || filepathCleanEqual(currentFile(), suggestionCenter.source) == false {
		showError("The selected file changed. Reopen Suggestion for the current file before applying anything.")
		return
	}
	suggestion := suggestionCenter.suggestion
	if applyDestination && suggestion.Destination != "" {
		st, err := os.Stat(suggestion.Destination)
		if err != nil || !st.IsDir() {
			showError("The learned destination is no longer an existing folder:\n" + suggestion.Destination + "\n\nDCM did not change the Destination field.")
			return
		}
	}
	if applyName && suggestion.RenameStyle != "" {
		st, err := os.Stat(currentFile())
		if err != nil {
			showError("Could not read the selected file before applying the rename suggestion:\n" + err.Error())
			return
		}
		proposed, err := proposedNameForRenameStyle(getText(app.nameEdit), currentOriginalStem(), suggestion.RenameStyle, st.ModTime())
		if err != nil {
			showError("Could not apply the rename suggestion:\n" + err.Error())
			return
		}
		setText(app.nameEdit, proposed)
	}
	if applyDestination && suggestion.Destination != "" {
		setDestinationText(suggestion.Destination)
	}
	setText(app.statusLabel, "Learned suggestion applied to editable fields only. Review it, then use Rename, Move, or Next as appropriate.")
	setText(suggestionCenter.statusLabel, "Applied to proposal fields only; no file was moved.")
}

func filepathCleanEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
