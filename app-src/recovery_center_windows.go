//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	ID_RC_LIST           = 1201
	ID_RC_MARK_REVIEWED  = 1202
	ID_RC_OPEN_SOURCE    = 1203
	ID_RC_OPEN_DEST      = 1204
	ID_RC_OPEN_BACKUP    = 1205
	ID_RC_CLEANUP        = 1206
	ID_RC_COMPACT        = 1207
	ID_RC_REFRESH        = 1208
	ID_RC_CLOSE          = 1209
	ID_RC_RESUME         = 1210
	ID_RC_RESTART        = 1211
	ID_RC_REMOVE_PARTIAL = 1212
	ID_RC_FINISH         = 1213

	WM_RECOVERY_ACTION_DONE     = 0x8005 // WM_APP + 5
	WM_RECOVERY_ACTION_PROGRESS = 0x8006 // WM_APP + 6
)

type recoveryCenterWindowState struct {
	hwnd, list, detail, statusLabel                                                    uintptr
	markReviewedBtn, openSourceBtn, openDestBtn, openBackupBtn, cleanupBtn, compactBtn uintptr
	resumeBtn, restartBtn, removePartialBtn, finishBtn, refreshBtn, closeBtn           uintptr
	items                                                                              []RecoveryCenterItem
	selected                                                                           int
	actionBusy                                                                         bool
	activeAction                                                                       RecoveryMoveAction
}

var recoveryCenter recoveryCenterWindowState
var recoveryCenterWndProcCB = syscall.NewCallback(recoveryCenterWndProc)
var recoveryMoveActionResults = make(chan InterruptedMoveRecoveryResult, 1)
var recoveryMoveActionProgress = make(chan BackgroundMoveProgress, 16)

func registerRecoveryCenterClass(hInst, cursor uintptr) bool {
	class := wstr("QuickSortRecoveryCenter")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: recoveryCenterWndProcCB, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: class}
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	return r != 0
}

func openRecoveryCenter() {
	if recoveryCenter.hwnd != 0 {
		procShowWindow.Call(recoveryCenter.hwnd, SW_SHOW)
		refreshRecoveryCenterItems(-1)
		return
	}
	hInst, _, _ := procGetModuleHandleW.Call(0)
	style := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE | WS_CLIPCHILDREN)
	hwnd, _, _ := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(wstr("QuickSortRecoveryCenter"))),
		uintptr(unsafe.Pointer(wstr("Recovery Center — DCM - Quick Sort"))),
		style, CW_USEDEFAULT, CW_USEDEFAULT, 960, 700, app.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		showError("Could not open Recovery Center.")
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
}

func recoveryCenterWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		recoveryCenter.hwnd = hwnd
		recoveryCenter.selected = -1
		styleTitleBar(hwnd)
		createRecoveryCenterUI(hwnd)
		refreshRecoveryCenterItems(-1)
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
		if notify == LBN_SELCHANGE && id == ID_RC_LIST {
			showSelectedRecoveryCenterItem()
			return 0
		}
		if notify == BN_CLICKED {
			switch id {
			case ID_RC_MARK_REVIEWED:
				markSelectedRecoveryCenterReviewed()
			case ID_RC_RESUME:
				beginSelectedRecoveryMoveAction(RecoveryMoveResume)
			case ID_RC_RESTART:
				beginSelectedRecoveryMoveAction(RecoveryMoveRestart)
			case ID_RC_REMOVE_PARTIAL:
				beginSelectedRecoveryMoveAction(RecoveryMoveRemovePartial)
			case ID_RC_FINISH:
				beginSelectedRecoveryMoveAction(RecoveryMoveFinish)
			case ID_RC_OPEN_SOURCE:
				openSelectedRecoveryFolder("source")
			case ID_RC_OPEN_DEST:
				openSelectedRecoveryFolder("destination")
			case ID_RC_OPEN_BACKUP:
				openSelectedRecoveryFolder("backup")
			case ID_RC_CLEANUP:
				runRecoveryCleanup()
			case ID_RC_COMPACT:
				runRecoveryCompaction()
			case ID_RC_REFRESH:
				refreshRecoveryCenterItems(recoveryCenter.selected)
			case ID_RC_CLOSE:
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			}
		}
		return 0
	case WM_RECOVERY_ACTION_DONE:
		finishRecoveryMoveAction()
		return 0
	case WM_RECOVERY_ACTION_PROGRESS:
		finishRecoveryMoveProgress()
		return 0
	case WM_CLOSE:
		if recoveryCenter.actionBusy {
			procMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(wstr("A recovery operation is still running. Recovery Center will remain open until it finishes safely."))), uintptr(unsafe.Pointer(wstr("Recovery Operation — DCM - Quick Sort"))), 0x40)
			return 0
		}
		procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return 0
	case WM_DESTROY:
		recoveryCenter = recoveryCenterWindowState{selected: -1}
		refreshRecoveryCenterBadge()
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func createRecoveryCenterUI(hwnd uintptr) {
	title := createControl("STATIC", "Recovery / Attention", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 18, 300, 24, hwnd, 0)
	setControlFont(title, app.headingFont)
	help := createControl("STATIC", "Review unresolved operations. Mark Reviewed only acknowledges an item; it never moves or deletes source/destination files.", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 47, 880, 36, hwnd, 0)
	setControlFont(help, app.smallFont)

	recoveryCenter.list = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|WS_VSCROLL, 20, 88, 310, 360, hwnd, ID_RC_LIST)
	recoveryCenter.detail = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 345, 88, 555, 360, hwnd, 0)
	setExplorerTheme(recoveryCenter.list)

	recoveryCenter.resumeBtn = createControl("BUTTON", "&Resume Copy", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 20, 463, 130, 31, hwnd, ID_RC_RESUME)
	recoveryCenter.restartBtn = createControl("BUTTON", "Restart &Move", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 160, 463, 130, 31, hwnd, ID_RC_RESTART)
	recoveryCenter.removePartialBtn = createControl("BUTTON", "Remove &Partial", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 300, 463, 140, 31, hwnd, ID_RC_REMOVE_PARTIAL)
	recoveryCenter.finishBtn = createControl("BUTTON", "&Finish Move", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 450, 463, 130, 31, hwnd, ID_RC_FINISH)
	recoveryCenter.markReviewedBtn = createControl("BUTTON", "Mark Reviewed", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 590, 463, 130, 31, hwnd, ID_RC_MARK_REVIEWED)

	recoveryCenter.openSourceBtn = createControl("BUTTON", "Open &Source Folder", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 20, 507, 150, 31, hwnd, ID_RC_OPEN_SOURCE)
	recoveryCenter.openDestBtn = createControl("BUTTON", "Open &Destination", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 180, 507, 150, 31, hwnd, ID_RC_OPEN_DEST)
	recoveryCenter.openBackupBtn = createControl("BUTTON", "Open Recovery &Copy", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 340, 507, 160, 31, hwnd, ID_RC_OPEN_BACKUP)

	recoveryCenter.cleanupBtn = createControl("BUTTON", "Cleanup Reviewed (30d)", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 20, 551, 175, 31, hwnd, ID_RC_CLEANUP)
	recoveryCenter.compactBtn = createControl("BUTTON", "Compact &Journal", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 205, 551, 145, 31, hwnd, ID_RC_COMPACT)
	recoveryCenter.refreshBtn = createControl("BUTTON", "Re&fresh", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 360, 551, 100, 31, hwnd, ID_RC_REFRESH)
	recoveryCenter.closeBtn = createControl("BUTTON", "Close", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 800, 551, 100, 31, hwnd, ID_RC_CLOSE)

	recoveryCenter.statusLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 595, 900, 48, hwnd, 0)
	setControlFont(recoveryCenter.statusLabel, app.smallFont)

	for _, h := range []uintptr{recoveryCenter.resumeBtn, recoveryCenter.restartBtn, recoveryCenter.removePartialBtn, recoveryCenter.finishBtn, recoveryCenter.markReviewedBtn, recoveryCenter.openSourceBtn, recoveryCenter.openDestBtn, recoveryCenter.openBackupBtn, recoveryCenter.cleanupBtn, recoveryCenter.compactBtn, recoveryCenter.refreshBtn, recoveryCenter.closeBtn} {
		setExplorerTheme(h)
	}
}

func recoveryCenterLabel(item RecoveryCenterItem) string {
	name := filepath.Base(item.To)
	if strings.TrimSpace(name) == "" || name == "." {
		name = filepath.Base(item.From)
	}
	state := strings.ToUpper(string(item.State))
	kind := string(item.Kind)
	if kind == "" {
		kind = "move"
	}
	return state + " · " + kind + " · " + name
}

func refreshRecoveryCenterItems(preferred int) {
	if recoveryCenter.hwnd == 0 || recoveryCenter.list == 0 {
		return
	}
	items, err := loadRecoveryCenterItems(moveJournalFilePath())
	if err != nil {
		setText(recoveryCenter.statusLabel, "Could not read durable journal: "+err.Error())
		return
	}
	recoveryCenter.items = items
	procSendMessageW.Call(recoveryCenter.list, LB_RESETCONTENT, 0, 0)
	for _, item := range items {
		procSendMessageW.Call(recoveryCenter.list, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(recoveryCenterLabel(item)))))
	}
	if len(items) == 0 {
		recoveryCenter.selected = -1
		setText(recoveryCenter.detail, "No recovery or attention items.\r\n\r\nCompleted undoable operations remain in the main Processed list.")
		setText(recoveryCenter.statusLabel, "Recovery Center is clear.")
		setRecoveryCenterActionState(nil)
		refreshRecoveryCenterBadge()
		return
	}
	if preferred < 0 || preferred >= len(items) {
		preferred = 0
	}
	recoveryCenter.selected = preferred
	procSendMessageW.Call(recoveryCenter.list, LB_SETCURSEL, uintptr(preferred), 0)
	showSelectedRecoveryCenterItem()
	refreshRecoveryCenterBadge()
}

func selectedRecoveryCenterItem() *RecoveryCenterItem {
	if recoveryCenter.list == 0 {
		return nil
	}
	idx, _, _ := procSendMessageW.Call(recoveryCenter.list, LB_GETCURSEL, 0, 0)
	if idx == LB_ERR || int(idx) < 0 || int(idx) >= len(recoveryCenter.items) {
		return nil
	}
	recoveryCenter.selected = int(idx)
	return &recoveryCenter.items[int(idx)]
}

func showSelectedRecoveryCenterItem() {
	item := selectedRecoveryCenterItem()
	if item == nil {
		setRecoveryCenterActionState(nil)
		return
	}
	var b strings.Builder
	b.WriteString("State: ")
	b.WriteString(string(item.State))
	b.WriteString("\r\nOperation: ")
	b.WriteString(string(item.Kind))
	if item.Stage != "" {
		b.WriteString("\r\nStage: ")
		b.WriteString(string(item.Stage))
	}
	b.WriteString("\r\nUpdated: ")
	b.WriteString(item.UpdatedAt.Local().Format("2006-01-02 15:04:05"))
	b.WriteString("\r\n\r\nSource:\r\n")
	b.WriteString(item.From)
	b.WriteString("\r\n\r\nDestination:\r\n")
	b.WriteString(item.To)
	if strings.TrimSpace(item.Backup) != "" {
		b.WriteString("\r\n\r\nRecovery copy:\r\n")
		b.WriteString(item.Backup)
	}
	if strings.TrimSpace(item.Detail) != "" {
		b.WriteString("\r\n\r\nDetail:\r\n")
		b.WriteString(item.Detail)
	}
	if item.Kind == MoveKindMove && (item.State == MoveJournalAttention || item.State == MoveJournalFailed) {
		availability := interruptedMoveRecoveryAvailability(item.From, item.To)
		if availability.CanAttemptPartial {
			b.WriteString(fmt.Sprintf("\r\n\r\nPower-failure recovery candidate: destination is smaller than source (%d / %d bytes). Resume/Restart/Remove will verify every existing destination byte against the source before changing anything.", availability.TargetSize, availability.SourceSize))
		} else if availability.CanAttemptFinish {
			b.WriteString("\r\n\r\nPower-failure recovery candidate: source and destination have equal size. Finish Move will require a full SHA-256 match before deleting the source.")
		}
	}
	if item.Reviewed {
		b.WriteString("\r\n\r\nReviewed items are retained in the journal. Cleanup only removes 30-day-old recovery copies that are provably redundant with the destination.")
	}
	setText(recoveryCenter.detail, b.String())
	setRecoveryCenterActionState(item)
	setText(recoveryCenter.statusLabel, "Selected "+recoveryCenterLabel(*item))
}

func setRecoveryCenterActionState(item *RecoveryCenterItem) {
	enable := func(hwnd uintptr, yes bool) {
		v := uintptr(0)
		if yes {
			v = 1
		}
		procEnableWindow.Call(hwnd, v)
	}
	if item == nil {
		enable(recoveryCenter.resumeBtn, false)
		enable(recoveryCenter.restartBtn, false)
		enable(recoveryCenter.removePartialBtn, false)
		enable(recoveryCenter.finishBtn, false)
		enable(recoveryCenter.markReviewedBtn, false)
		enable(recoveryCenter.openSourceBtn, false)
		enable(recoveryCenter.openDestBtn, false)
		enable(recoveryCenter.openBackupBtn, false)
		return
	}
	availability := interruptedMoveRecoveryAvailability(item.From, item.To)
	moveRecoverable := item.Kind == MoveKindMove && (item.State == MoveJournalAttention || item.State == MoveJournalFailed) && !recoveryCenter.actionBusy
	enable(recoveryCenter.resumeBtn, moveRecoverable && availability.CanAttemptPartial)
	enable(recoveryCenter.restartBtn, moveRecoverable && availability.CanAttemptPartial)
	enable(recoveryCenter.removePartialBtn, moveRecoverable && availability.CanAttemptPartial)
	enable(recoveryCenter.finishBtn, moveRecoverable && availability.CanAttemptFinish)
	enable(recoveryCenter.markReviewedBtn, item.CanMarkReviewed && !recoveryCenter.actionBusy)
	enable(recoveryCenter.openSourceBtn, existingParentDirectory(item.From) != "" && !recoveryCenter.actionBusy)
	enable(recoveryCenter.openDestBtn, existingParentDirectory(item.To) != "" && !recoveryCenter.actionBusy)
	enable(recoveryCenter.openBackupBtn, strings.TrimSpace(item.Backup) != "" && existingParentDirectory(item.Backup) != "" && !recoveryCenter.actionBusy)
}

func existingParentDirectory(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		return path
	}
	parent := filepath.Dir(path)
	if st, err := os.Stat(parent); err == nil && st.IsDir() {
		return parent
	}
	return ""
}

func openSelectedRecoveryFolder(which string) {
	item := selectedRecoveryCenterItem()
	if item == nil {
		return
	}
	path := item.From
	switch which {
	case "destination":
		path = item.To
	case "backup":
		path = item.Backup
	}
	folder := existingParentDirectory(path)
	if folder == "" {
		setText(recoveryCenter.statusLabel, "That folder is no longer available.")
		return
	}
	shellOpen(folder)
	setText(recoveryCenter.statusLabel, "Opened: "+folder)
}

func markSelectedRecoveryCenterReviewed() {
	item := selectedRecoveryCenterItem()
	if item == nil || !item.CanMarkReviewed {
		return
	}
	if err := markRecoveryEntryReviewed(moveJournalFilePath(), item.ID, time.Now()); err != nil {
		setText(recoveryCenter.statusLabel, "Could not mark reviewed: "+err.Error())
		return
	}
	setText(recoveryCenter.statusLabel, "Marked reviewed. No files were changed.")
	refreshRecoveryCenterItems(recoveryCenter.selected)
}

func runRecoveryCleanup() {
	summary, err := cleanupResolvedRecoveryArtifacts(moveJournalFilePath(), replacementRecoveryDirectory(), defaultResolvedRecoveryRetention, time.Now())
	if err != nil {
		setText(recoveryCenter.statusLabel, "Cleanup failed safely: "+err.Error())
		return
	}
	setText(recoveryCenter.statusLabel, fmt.Sprintf("Cleanup: removed %d redundant recovery copy/copies · skipped %d · already missing %d", summary.Removed, summary.Skipped, summary.Missing))
	refreshRecoveryCenterItems(recoveryCenter.selected)
}

func runRecoveryCompaction() {
	if app.backgroundMoveBusy {
		setText(recoveryCenter.statusLabel, "Journal compaction waits until the active file operation finishes.")
		return
	}
	summary, err := compactMoveJournal(moveJournalFilePath(), 2)
	if err != nil {
		setText(recoveryCenter.statusLabel, "Journal compaction failed safely: "+err.Error())
		return
	}
	if !summary.Compacted {
		setText(recoveryCenter.statusLabel, fmt.Sprintf("Journal already compact: %d event(s).", summary.BeforeEvents))
		return
	}
	setText(recoveryCenter.statusLabel, fmt.Sprintf("Journal compacted: %d events → %d current states.", summary.BeforeEvents, summary.AfterEvents))
	refreshRecoveryCenterItems(recoveryCenter.selected)
}

func refreshRecoveryCenterBadge() {
	if app.recoveryBtn == 0 {
		return
	}
	items, err := loadRecoveryCenterItems(moveJournalFilePath())
	if err != nil {
		setText(app.recoveryBtn, "Recovery !")
		return
	}
	pending := 0
	for _, item := range items {
		if item.State == MoveJournalAttention || item.State == MoveJournalFailed {
			pending++
		}
	}
	if pending > 0 {
		setText(app.recoveryBtn, fmt.Sprintf("Recovery (%d)", pending))
	} else {
		setText(app.recoveryBtn, "Recovery")
	}
}

func setRecoveryCenterBusy(busy bool) {
	recoveryCenter.actionBusy = busy
	flag := uintptr(1)
	if busy {
		flag = 0
	}
	for _, hwnd := range []uintptr{recoveryCenter.list, recoveryCenter.resumeBtn, recoveryCenter.restartBtn, recoveryCenter.removePartialBtn, recoveryCenter.finishBtn, recoveryCenter.markReviewedBtn, recoveryCenter.openSourceBtn, recoveryCenter.openDestBtn, recoveryCenter.openBackupBtn, recoveryCenter.cleanupBtn, recoveryCenter.compactBtn, recoveryCenter.refreshBtn} {
		if hwnd != 0 {
			procEnableWindow.Call(hwnd, flag)
		}
	}
	if !busy {
		setRecoveryCenterActionState(selectedRecoveryCenterItem())
	}
}

func confirmRecoveryMoveAction(action RecoveryMoveAction, item RecoveryCenterItem) bool {
	var text string
	switch action {
	case RecoveryMoveRestart:
		text = "Restart Move will first verify that the existing destination is a partial byte-for-byte prefix of the source. Only then will DCM remove that partial destination and copy the source again from the beginning.\n\nThe source will not be deleted until the new destination is complete and verified. Continue?"
	case RecoveryMoveRemovePartial:
		text = "Remove Partial will first verify that the destination is a partial byte-for-byte prefix of the source. Only that verified partial destination will be removed.\n\nThe source file will remain untouched. Continue?"
	case RecoveryMoveFinish:
		text = "Finish Move will calculate SHA-256 for both files. DCM will remove the source only if the destination is already an exact complete match. Continue?"
	default:
		return true
	}
	text += "\n\nSource: " + item.From + "\nDestination: " + item.To
	r, _, _ := procMessageBoxW.Call(recoveryCenter.hwnd, uintptr(unsafe.Pointer(wstr(text))), uintptr(unsafe.Pointer(wstr("Power-Failure Recovery — DCM - Quick Sort"))), 0x34) // MB_YESNO | MB_ICONWARNING
	return r == 6                                                                                                                                                               // IDYES
}

func beginSelectedRecoveryMoveAction(action RecoveryMoveAction) {
	item := selectedRecoveryCenterItem()
	if item == nil {
		return
	}
	if recoveryCenter.actionBusy || app.backgroundMoveBusy {
		setText(recoveryCenter.statusLabel, "Another file operation is already running.")
		return
	}
	if item.Kind != MoveKindMove || (item.State != MoveJournalAttention && item.State != MoveJournalFailed) {
		setText(recoveryCenter.statusLabel, "This recovery action applies only to an interrupted ordinary move.")
		return
	}
	availability := interruptedMoveRecoveryAvailability(item.From, item.To)
	if (action == RecoveryMoveResume || action == RecoveryMoveRestart || action == RecoveryMoveRemovePartial) && !availability.CanAttemptPartial {
		setText(recoveryCenter.statusLabel, "Destination is not a smaller recovery candidate. Refresh and review the paths.")
		return
	}
	if action == RecoveryMoveFinish && !availability.CanAttemptFinish {
		setText(recoveryCenter.statusLabel, "Finish Move requires source and destination to have the same size before SHA-256 verification.")
		return
	}
	if !confirmRecoveryMoveAction(action, *item) {
		setText(recoveryCenter.statusLabel, "Recovery action cancelled; no files were changed.")
		return
	}

	recoveryCenter.activeAction = action
	setBackgroundMoveBusy(true)
	setRecoveryCenterBusy(true)
	setText(recoveryCenter.statusLabel, "Validating interrupted move before "+string(action)+"… Source remains protected.")
	journalID := item.ID
	hwnd := recoveryCenter.hwnd
	progressFn := func(copied, total int64) {
		progress := BackgroundMoveProgress{Source: item.From, Target: item.To, Copied: copied, Total: total}
		select {
		case recoveryMoveActionProgress <- progress:
			procPostMessageW.Call(hwnd, WM_RECOVERY_ACTION_PROGRESS, 0, 0)
		default:
		}
	}
	go func() {
		result := executeInterruptedMoveRecovery(moveJournalFilePath(), journalID, action, progressFn)
		recoveryMoveActionResults <- result
		procPostMessageW.Call(hwnd, WM_RECOVERY_ACTION_DONE, 0, 0)
	}()
}

func finishRecoveryMoveProgress() {
	var latest BackgroundMoveProgress
	found := false
	for {
		select {
		case latest = <-recoveryMoveActionProgress:
			found = true
		default:
			if !found || !recoveryCenter.actionBusy {
				return
			}
			if latest.Total <= 0 {
				return
			}
			pct := float64(latest.Copied) * 100 / float64(latest.Total)
			copiedMB := float64(latest.Copied) / (1024 * 1024)
			totalMB := float64(latest.Total) / (1024 * 1024)
			msg := fmt.Sprintf("Recovery %s: %.1f%% · %.1f / %.1f MB", recoveryCenter.activeAction, pct, copiedMB, totalMB)
			setText(recoveryCenter.statusLabel, msg)
			setText(app.statusLabel, msg)
			return
		}
	}
}

func finishRecoveryMoveAction() {
	var result InterruptedMoveRecoveryResult
	select {
	case result = <-recoveryMoveActionResults:
	default:
		return
	}
	action := recoveryCenter.activeAction
	recoveryCenter.activeAction = ""
	setRecoveryCenterBusy(false)
	setBackgroundMoveBusy(false)

	if result.Err != nil {
		msg := "Recovery action stopped safely: " + result.Err.Error()
		setText(recoveryCenter.statusLabel, msg)
		setText(app.statusLabel, msg)
		refreshRecoveryCenterItems(recoveryCenter.selected)
		showError(msg + "\n\nDCM did not delete the source unless a complete destination had already been verified.")
		return
	}

	if action != RecoveryMoveRemovePartial {
		recordProcessed(result.Record)
		reloadFiles()
	}
	refreshRecoveryCenterItems(-1)
	refreshRecoveryCenterBadge()
	var msg string
	switch action {
	case RecoveryMoveResume:
		msg = "Interrupted copy resumed, verified, and completed. Source was removed only after verification."
	case RecoveryMoveRestart:
		msg = "Interrupted move restarted from zero, verified, and completed safely."
	case RecoveryMoveRemovePartial:
		msg = "Verified partial destination removed. Source remains unchanged; interrupted move is resolved."
	case RecoveryMoveFinish:
		msg = "Destination was already an exact SHA-256 match. Move finalized without recopying the file."
	default:
		msg = "Recovery operation completed safely."
	}
	setText(recoveryCenter.statusLabel, msg)
	setText(app.statusLabel, msg)
}
