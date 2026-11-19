//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	ID_NTM_TEMPLATE_LIST = 1501 + iota
	ID_NTM_NAME
	ID_NTM_PATTERN
	ID_NTM_SAVE
	ID_NTM_NEW
	ID_NTM_DELETE
	ID_NTM_ACTIVE
	ID_NTM_VAR_LIST
	ID_NTM_VAR_NAME
	ID_NTM_VAR_VALUE
	ID_NTM_VAR_SAVE
	ID_NTM_VAR_DELETE
	ID_NTM_CLOSE
)

type namingTemplateManagerState struct {
	hwnd                                                                                            uintptr
	templateList, nameEdit, patternEdit, saveBtn, newBtn, deleteBtn, activeBtn                      uintptr
	variableList, variableNameEdit, variableValueEdit, variableSaveBtn, variableDeleteBtn, closeBtn uintptr
	statusLabel                                                                                     uintptr
	cfg                                                                                             NamingTemplateConfig
	workingVariables                                                                                []NamingVariable
	selectedTemplate, selectedVariable                                                              int
}

var namingTemplateManager = namingTemplateManagerState{selectedTemplate: -1, selectedVariable: -1}
var namingTemplateManagerWndProcCB = syscall.NewCallback(namingTemplateManagerWndProc)

func registerNamingTemplateManagerClass(hInst, cursor uintptr) bool {
	class := wstr("QuickSortNamingTemplates")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: namingTemplateManagerWndProcCB, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: class}
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	return r != 0
}

func namingTemplateManagerFocusOrder() []uintptr {
	return []uintptr{
		namingTemplateManager.templateList,
		namingTemplateManager.nameEdit,
		namingTemplateManager.patternEdit,
		namingTemplateManager.saveBtn,
		namingTemplateManager.newBtn,
		namingTemplateManager.deleteBtn,
		namingTemplateManager.activeBtn,
		namingTemplateManager.variableList,
		namingTemplateManager.variableNameEdit,
		namingTemplateManager.variableValueEdit,
		namingTemplateManager.variableSaveBtn,
		namingTemplateManager.variableDeleteBtn,
		namingTemplateManager.closeBtn,
	}
}

func focusWithinNamingTemplateManager(focus uintptr) bool {
	if namingTemplateManager.hwnd == 0 {
		return false
	}
	for _, hwnd := range namingTemplateManagerFocusOrder() {
		if focus == hwnd {
			return true
		}
	}
	return false
}

func moveNamingTemplateManagerFocus(reverse bool) bool {
	order := namingTemplateManagerFocusOrder()
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

func namingTemplateListText(t NamingTemplate, active string) string {
	prefix := "   "
	if strings.EqualFold(strings.TrimSpace(t.Name), strings.TrimSpace(active)) {
		prefix = "✓  "
	}
	return prefix + t.Name + "    →    " + t.Pattern
}

func populateNamingTemplateList(selection int) {
	procSendMessageW.Call(namingTemplateManager.templateList, LB_RESETCONTENT, 0, 0)
	for _, t := range namingTemplateManager.cfg.Templates {
		procSendMessageW.Call(namingTemplateManager.templateList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(namingTemplateListText(t, namingTemplateManager.cfg.Active)))))
	}
	namingTemplateManager.selectedTemplate = -1
	if selection >= 0 && selection < len(namingTemplateManager.cfg.Templates) {
		namingTemplateManager.selectedTemplate = selection
		procSendMessageW.Call(namingTemplateManager.templateList, LB_SETCURSEL, uintptr(selection), 0)
	}
	flag := uintptr(0)
	if namingTemplateManager.selectedTemplate >= 0 {
		flag = 1
	}
	procEnableWindow.Call(namingTemplateManager.deleteBtn, flag)
	procEnableWindow.Call(namingTemplateManager.activeBtn, flag)
}

func populateNamingVariableList(selection int) {
	procSendMessageW.Call(namingTemplateManager.variableList, LB_RESETCONTENT, 0, 0)
	for _, v := range namingTemplateManager.workingVariables {
		line := "{" + v.Name + "}    =    " + v.Value
		procSendMessageW.Call(namingTemplateManager.variableList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(line))))
	}
	namingTemplateManager.selectedVariable = -1
	if selection >= 0 && selection < len(namingTemplateManager.workingVariables) {
		namingTemplateManager.selectedVariable = selection
		procSendMessageW.Call(namingTemplateManager.variableList, LB_SETCURSEL, uintptr(selection), 0)
	}
	flag := uintptr(0)
	if namingTemplateManager.selectedVariable >= 0 {
		flag = 1
	}
	procEnableWindow.Call(namingTemplateManager.variableDeleteBtn, flag)
}

func clearNamingVariableForm() {
	namingTemplateManager.selectedVariable = -1
	setText(namingTemplateManager.variableNameEdit, "")
	setText(namingTemplateManager.variableValueEdit, "")
	procSendMessageW.Call(namingTemplateManager.variableList, LB_SETCURSEL, ^uintptr(0), 0)
	procEnableWindow.Call(namingTemplateManager.variableDeleteBtn, 0)
}

func clearNamingTemplateForm() {
	namingTemplateManager.selectedTemplate = -1
	namingTemplateManager.workingVariables = nil
	setText(namingTemplateManager.nameEdit, "")
	setText(namingTemplateManager.patternEdit, "{name}")
	populateNamingTemplateList(-1)
	populateNamingVariableList(-1)
	clearNamingVariableForm()
	setText(namingTemplateManager.statusLabel, "Create a template. Built-ins: {date}, {name}, {original}. Add your own variables below.")
	procSetFocus.Call(namingTemplateManager.nameEdit)
}

func loadNamingTemplateSelection(index int) {
	if index < 0 || index >= len(namingTemplateManager.cfg.Templates) {
		clearNamingTemplateForm()
		return
	}
	namingTemplateManager.selectedTemplate = index
	t := namingTemplateManager.cfg.Templates[index]
	namingTemplateManager.workingVariables = append([]NamingVariable(nil), t.Variables...)
	setText(namingTemplateManager.nameEdit, t.Name)
	setText(namingTemplateManager.patternEdit, t.Pattern)
	populateNamingVariableList(-1)
	procEnableWindow.Call(namingTemplateManager.deleteBtn, 1)
	procEnableWindow.Call(namingTemplateManager.activeBtn, 1)
	state := "Saved template"
	if strings.EqualFold(t.Name, namingTemplateManager.cfg.Active) {
		state = "Active template"
	}
	setText(namingTemplateManager.statusLabel, state+": "+t.Name+". Built-ins: {date}, {name}, {original}.")
}

func loadNamingVariableSelection(index int) {
	if index < 0 || index >= len(namingTemplateManager.workingVariables) {
		clearNamingVariableForm()
		return
	}
	namingTemplateManager.selectedVariable = index
	v := namingTemplateManager.workingVariables[index]
	setText(namingTemplateManager.variableNameEdit, v.Name)
	setText(namingTemplateManager.variableValueEdit, v.Value)
	procEnableWindow.Call(namingTemplateManager.variableDeleteBtn, 1)
}

func saveNamingTemplateFromManager() bool {
	t := NamingTemplate{
		Name:      getText(namingTemplateManager.nameEdit),
		Pattern:   getText(namingTemplateManager.patternEdit),
		Variables: append([]NamingVariable(nil), namingTemplateManager.workingVariables...),
	}
	base := namingTemplateManager.cfg
	if namingTemplateManager.selectedTemplate >= 0 && namingTemplateManager.selectedTemplate < len(base.Templates) {
		oldName := base.Templates[namingTemplateManager.selectedTemplate].Name
		if !strings.EqualFold(strings.TrimSpace(oldName), strings.TrimSpace(t.Name)) {
			wasActive := strings.EqualFold(strings.TrimSpace(base.Active), strings.TrimSpace(oldName))
			base = deleteNamingTemplate(base, namingTemplateManager.selectedTemplate)
			if wasActive {
				base.Active = strings.TrimSpace(t.Name)
			}
		}
	}
	cfg, index, err := upsertNamingTemplate(base, t)
	if err != nil {
		showError("Could not save naming template:\n" + err.Error())
		return false
	}
	if err := saveNamingTemplateConfig(namingTemplatesFilePath(), cfg); err != nil {
		showError("Could not save naming templates:\n" + err.Error())
		return false
	}
	namingTemplateManager.cfg = cfg
	populateNamingTemplateList(index)
	loadNamingTemplateSelection(index)
	status := "Saved naming template: " + cfg.Templates[index].Name
	if _, backupErr := backupCurrentConfiguration("naming-template-saved"); backupErr != nil {
		status += " · backup warning: " + backupErr.Error()
	}
	setText(namingTemplateManager.statusLabel, status)
	return true
}

func deleteNamingTemplateFromManager() {
	idx := namingTemplateManager.selectedTemplate
	if idx < 0 || idx >= len(namingTemplateManager.cfg.Templates) {
		return
	}
	name := namingTemplateManager.cfg.Templates[idx].Name
	cfg := deleteNamingTemplate(namingTemplateManager.cfg, idx)
	if err := saveNamingTemplateConfig(namingTemplatesFilePath(), cfg); err != nil {
		showError("Could not delete naming template:\n" + err.Error())
		return
	}
	namingTemplateManager.cfg = cfg
	selection := idx
	if selection >= len(cfg.Templates) {
		selection = len(cfg.Templates) - 1
	}
	populateNamingTemplateList(selection)
	loadNamingTemplateSelection(selection)
	status := "Deleted " + name + ". Active template: " + cfg.Active
	if _, backupErr := backupCurrentConfiguration("naming-template-removed"); backupErr != nil {
		status += " · backup warning: " + backupErr.Error()
	}
	setText(namingTemplateManager.statusLabel, status)
}

func setActiveNamingTemplateFromManager() {
	idx := namingTemplateManager.selectedTemplate
	if idx < 0 || idx >= len(namingTemplateManager.cfg.Templates) {
		return
	}
	namingTemplateManager.cfg.Active = namingTemplateManager.cfg.Templates[idx].Name
	if err := saveNamingTemplateConfig(namingTemplatesFilePath(), namingTemplateManager.cfg); err != nil {
		showError("Could not set active naming template:\n" + err.Error())
		return
	}
	populateNamingTemplateList(idx)
	status := "Active naming template: " + namingTemplateManager.cfg.Active + ". Choose Naming Template in the Rename Tool list to apply it."
	if _, backupErr := backupCurrentConfiguration("active-naming-template-changed"); backupErr != nil {
		status += " · backup warning: " + backupErr.Error()
	}
	setText(namingTemplateManager.statusLabel, status)
}

func saveNamingVariableFromManager() {
	v := NamingVariable{Name: getText(namingTemplateManager.variableNameEdit), Value: getText(namingTemplateManager.variableValueEdit)}
	base := append([]NamingVariable(nil), namingTemplateManager.workingVariables...)
	if namingTemplateManager.selectedVariable >= 0 && namingTemplateManager.selectedVariable < len(base) {
		base = deleteNamingVariable(base, namingTemplateManager.selectedVariable)
	}
	vars, index, err := upsertNamingVariable(base, v)
	if err != nil {
		showError("Could not add variable:\n" + err.Error())
		return
	}
	namingTemplateManager.workingVariables = vars
	populateNamingVariableList(index)
	loadNamingVariableSelection(index)
	setText(namingTemplateManager.statusLabel, fmt.Sprintf("Variable {%s} is ready. Save Template to persist it.", strings.TrimSpace(v.Name)))
}

func deleteNamingVariableFromManager() {
	idx := namingTemplateManager.selectedVariable
	if idx < 0 || idx >= len(namingTemplateManager.workingVariables) {
		return
	}
	name := namingTemplateManager.workingVariables[idx].Name
	namingTemplateManager.workingVariables = deleteNamingVariable(namingTemplateManager.workingVariables, idx)
	populateNamingVariableList(-1)
	clearNamingVariableForm()
	setText(namingTemplateManager.statusLabel, "Removed {"+name+"} from this template. Save Template to persist the change.")
}

func createNamingTemplateManagerUI(hwnd uintptr) {
	nameLabel := createControl("STATIC", "Template name:", WS_CHILD|WS_VISIBLE|SS_LEFT, 300, 22, 105, 22, hwnd, 0)
	namingTemplateManager.nameEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 410, 18, 430, 28, hwnd, ID_NTM_NAME)
	patternLabel := createControl("STATIC", "Pattern:", WS_CHILD|WS_VISIBLE|SS_LEFT, 300, 62, 105, 22, hwnd, 0)
	namingTemplateManager.patternEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 410, 58, 430, 28, hwnd, ID_NTM_PATTERN)
	builtinLabel := createControl("STATIC", "Built-in variables: {date}  {name}  {original}", WS_CHILD|WS_VISIBLE|SS_LEFT, 300, 96, 540, 22, hwnd, 0)

	listLabel := createControl("STATIC", "Saved templates", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 22, 240, 22, hwnd, 0)
	namingTemplateManager.templateList = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|WS_VSCROLL, 20, 48, 255, 250, hwnd, ID_NTM_TEMPLATE_LIST)

	namingTemplateManager.saveBtn = createControl("BUTTON", "&Save Template", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 300, 126, 125, 31, hwnd, ID_NTM_SAVE)
	namingTemplateManager.newBtn = createControl("BUTTON", "&New", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 435, 126, 80, 31, hwnd, ID_NTM_NEW)
	namingTemplateManager.deleteBtn = createControl("BUTTON", "&Delete", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 525, 126, 90, 31, hwnd, ID_NTM_DELETE)
	namingTemplateManager.activeBtn = createControl("BUTTON", "Set &Active", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 625, 126, 105, 31, hwnd, ID_NTM_ACTIVE)

	variablesTitle := createControl("STATIC", "Custom variables", WS_CHILD|WS_VISIBLE|SS_LEFT, 300, 184, 180, 22, hwnd, 0)
	namingTemplateManager.variableList = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|WS_VSCROLL, 300, 210, 540, 145, hwnd, ID_NTM_VAR_LIST)
	varNameLabel := createControl("STATIC", "Variable:", WS_CHILD|WS_VISIBLE|SS_LEFT, 300, 374, 65, 22, hwnd, 0)
	namingTemplateManager.variableNameEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 370, 370, 150, 28, hwnd, ID_NTM_VAR_NAME)
	varValueLabel := createControl("STATIC", "Value:", WS_CHILD|WS_VISIBLE|SS_LEFT, 535, 374, 50, 22, hwnd, 0)
	namingTemplateManager.variableValueEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 590, 370, 250, 28, hwnd, ID_NTM_VAR_VALUE)
	namingTemplateManager.variableSaveBtn = createControl("BUTTON", "Add / &Update Variable", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 300, 410, 175, 31, hwnd, ID_NTM_VAR_SAVE)
	namingTemplateManager.variableDeleteBtn = createControl("BUTTON", "Delete &Variable", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 485, 410, 135, 31, hwnd, ID_NTM_VAR_DELETE)
	namingTemplateManager.closeBtn = createControl("BUTTON", "Close", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 730, 410, 110, 31, hwnd, ID_NTM_CLOSE)
	namingTemplateManager.statusLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 466, 820, 48, hwnd, 0)

	for _, c := range []uintptr{nameLabel, patternLabel, builtinLabel, listLabel, variablesTitle, varNameLabel, varValueLabel, namingTemplateManager.templateList, namingTemplateManager.nameEdit, namingTemplateManager.patternEdit, namingTemplateManager.saveBtn, namingTemplateManager.newBtn, namingTemplateManager.deleteBtn, namingTemplateManager.activeBtn, namingTemplateManager.variableList, namingTemplateManager.variableNameEdit, namingTemplateManager.variableValueEdit, namingTemplateManager.variableSaveBtn, namingTemplateManager.variableDeleteBtn, namingTemplateManager.closeBtn, namingTemplateManager.statusLabel} {
		setControlFont(c, app.uiFont)
	}
	for _, c := range []uintptr{namingTemplateManager.templateList, namingTemplateManager.nameEdit, namingTemplateManager.patternEdit, namingTemplateManager.saveBtn, namingTemplateManager.newBtn, namingTemplateManager.deleteBtn, namingTemplateManager.activeBtn, namingTemplateManager.variableList, namingTemplateManager.variableNameEdit, namingTemplateManager.variableValueEdit, namingTemplateManager.variableSaveBtn, namingTemplateManager.variableDeleteBtn, namingTemplateManager.closeBtn} {
		setExplorerTheme(c)
	}
	procEnableWindow.Call(namingTemplateManager.deleteBtn, 0)
	procEnableWindow.Call(namingTemplateManager.activeBtn, 0)
	procEnableWindow.Call(namingTemplateManager.variableDeleteBtn, 0)

	selection := 0
	for i, t := range namingTemplateManager.cfg.Templates {
		if strings.EqualFold(t.Name, namingTemplateManager.cfg.Active) {
			selection = i
			break
		}
	}
	populateNamingTemplateList(selection)
	loadNamingTemplateSelection(selection)
}

func namingTemplateManagerWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		namingTemplateManager.hwnd = hwnd
		styleTitleBar(hwnd)
		createNamingTemplateManagerUI(hwnd)
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
			case ID_NTM_SAVE:
				saveNamingTemplateFromManager()
			case ID_NTM_NEW:
				clearNamingTemplateForm()
			case ID_NTM_DELETE:
				deleteNamingTemplateFromManager()
			case ID_NTM_ACTIVE:
				setActiveNamingTemplateFromManager()
			case ID_NTM_VAR_SAVE:
				saveNamingVariableFromManager()
			case ID_NTM_VAR_DELETE:
				deleteNamingVariableFromManager()
			case ID_NTM_CLOSE:
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			}
		}
		if notify == LBN_SELCHANGE {
			switch id {
			case ID_NTM_TEMPLATE_LIST:
				if idx, _, _ := procSendMessageW.Call(namingTemplateManager.templateList, LB_GETCURSEL, 0, 0); idx != LB_ERR {
					loadNamingTemplateSelection(int(idx))
				}
			case ID_NTM_VAR_LIST:
				if idx, _, _ := procSendMessageW.Call(namingTemplateManager.variableList, LB_GETCURSEL, 0, 0); idx != LB_ERR {
					loadNamingVariableSelection(int(idx))
				}
			}
		}
		return 0
	case WM_DESTROY:
		namingTemplateManager = namingTemplateManagerState{selectedTemplate: -1, selectedVariable: -1}
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func openNamingTemplateManager() {
	if namingTemplateManager.hwnd != 0 {
		procShowWindow.Call(namingTemplateManager.hwnd, SW_SHOW)
		procSetFocus.Call(namingTemplateManager.templateList)
		return
	}
	cfg, err := loadNamingTemplateConfig(namingTemplatesFilePath())
	if err != nil {
		showError("Could not open Naming Templates:\n" + err.Error())
		return
	}
	namingTemplateManager.cfg = cfg
	namingTemplateManager.selectedTemplate = -1
	namingTemplateManager.selectedVariable = -1
	hInst, _, _ := procGetModuleHandleW.Call(0)
	style := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE | WS_CLIPCHILDREN)
	hwnd, _, _ := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(wstr("QuickSortNamingTemplates"))),
		uintptr(unsafe.Pointer(wstr("Naming Templates — DCM - Quick Sort"))),
		style, CW_USEDEFAULT, CW_USEDEFAULT, 880, 570, app.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		showError("Could not open Naming Templates.")
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
}
