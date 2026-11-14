package main

import (
	"os"
	"strings"
	"testing"
)

func windowsMainSource(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("main_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	// Git may check out Windows sources as CRLF. Preserve all source tokens
	// and indentation while comparing the same logical lines on every host.
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func TestClearFiltersButtonIsWiredToClearInboxFilters(t *testing.T) {
	src := windowsMainSource(t)
	if !strings.Contains(src, "case ID_FILTER_CLEAR:\n\t\t\t\tclearInboxFilters()") {
		t.Fatal("Clear Filters button is not wired to clearInboxFilters")
	}
}

func TestRenameToolSelectionAutomaticallyAppliesSelectedTool(t *testing.T) {
	src := windowsMainSource(t)
	if !strings.Contains(src, "case ID_RENAME_TOOL:\n\t\t\t\tapplySelectedRenameTool()") {
		t.Fatal("rename tool selection does not automatically apply the selected tool")
	}
}

func TestBackgroundConflictActionsUseDurableJournaledOperations(t *testing.T) {
	src := windowsMainSource(t)
	if !strings.Contains(src, "removeExactDuplicateSourceJournaled(moveJournalFilePath(), source, target, batchID)") {
		t.Fatal("exact-duplicate removal is not using the durable journaled operation")
	}
	if !strings.Contains(src, "replaceMoveWithRecoveryJournaled(moveJournalFilePath(), source, target, replacementRecoveryDirectory(), batchID)") {
		t.Fatal("replace is not using the durable journaled operation")
	}
}

func TestWindowDestroyDoesNotDeleteDurableReplacementBackups(t *testing.T) {
	src := windowsMainSource(t)
	if strings.Contains(src, "cleanupSessionReplacementBackups(app.history)") {
		t.Fatal("window destroy still deletes replacement recovery copies needed across restart")
	}
}

func TestRecoveryCenterButtonAndWindowAreWired(t *testing.T) {
	mainSrc := windowsMainSource(t)
	if !strings.Contains(mainSrc, "ID_RECOVERY_CENTER") || !strings.Contains(mainSrc, "openRecoveryCenter()") {
		t.Fatal("Recovery Center is not wired into the main window")
	}
	recoverySrcBytes, err := os.ReadFile("recovery_center_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	recoverySrc := string(recoverySrcBytes)
	if !strings.Contains(recoverySrc, "markRecoveryEntryReviewed(moveJournalFilePath()") {
		t.Fatal("Recovery Center does not use the safe journal-only Mark Reviewed action")
	}
}

func TestRecoveryCenterWiresInterruptedMoveActions(t *testing.T) {
	recoverySrcBytes, err := os.ReadFile("recovery_center_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	recoverySrc := string(recoverySrcBytes)
	for _, want := range []string{
		"RecoveryMoveResume",
		"RecoveryMoveRestart",
		"RecoveryMoveRemovePartial",
		"RecoveryMoveFinish",
		"executeInterruptedMoveRecovery(moveJournalFilePath()",
		"WM_RECOVERY_ACTION_DONE",
		"WM_RECOVERY_ACTION_PROGRESS",
	} {
		if !strings.Contains(recoverySrc, want) {
			t.Fatalf("Recovery Center missing interrupted-move wiring: %s", want)
		}
	}
}

func TestLearningCenterAndObservationAreWired(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		"ID_LEARNING_CENTER",
		"openLearningCenter()",
		"observeCompletedMoveLearning(learningFilePath(), rec, time.Now())",
		"unobserveCompletedMoveLearning(learningFilePath(), rec, time.Now())",
		"registerLearningCenterClass(hInst, cursor)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("Go 13 Windows integration missing %q", want)
		}
	}
	data, err := os.ReadFile("learning_center_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	learningSrc := string(data)
	for _, want := range []string{"setLearningPaused(learningFilePath()", "resetLearningModel(learningFilePath()", "Learning Center — DCM - Quick Sort"} {
		if !strings.Contains(learningSrc, want) {
			t.Fatalf("Learning Center missing %q", want)
		}
	}
}

func TestMarkdownAndCSVAreRoutedToPlainTextPreview(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`shouldUsePlainTextPreview(path)`,
		`showTextPreview(path)`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("plain-text Markdown/CSV preview wiring missing %q", want)
		}
	}
}

func TestSuggestionCenterAndFileFlowHandoffAreWired(t *testing.T) {
	mainSrc := windowsMainSource(t)
	for _, want := range []string{
		"ID_SUGGESTION_CENTER",
		"openSuggestionCenter()",
		"refreshSuggestionForCurrent()",
		"ID_FILEFLOW_HANDOFF",
		"queueSelectedForFileFlow()",
		"registerSuggestionCenterClass(hInst, cursor)",
	} {
		if !strings.Contains(mainSrc, want) {
			t.Fatalf("Go 14 Windows integration missing %q", want)
		}
	}
	data, err := os.ReadFile("suggestion_center_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	suggestionSrc := string(data)
	for _, want := range []string{"Apply Destination", "Apply Name", "Apply Both", "DestinationReasons", "RenameReasons"} {
		if !strings.Contains(suggestionSrc, want) {
			t.Fatalf("Suggestion Center missing %q", want)
		}
	}
}

func TestSourceAliasesAutocompleteAndManagerAreWired(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		"ID_SOURCE_ALIAS",
		"ID_SOURCE_CONFIG",
		"ID_SOURCE_LIST",
		"sourceAliasCombo",
		"sourceSuggestionList",
		"updateSourceSuggestions()",
		"acceptSourceSuggestion()",
		"openSourceManager()",
		"registerSourceManagerClass(hInst, cursor)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("source profiles/autocomplete wiring missing %q", want)
		}
	}
}

func TestSourceDestinationLearningAndSuggestionManagementAreMenuOnly(t *testing.T) {
	src := windowsMainSource(t)
	for _, forbidden := range []string{
		`app.sourceConfigBtn = createControl(`,
		`app.destConfigBtn = createControl(`,
		`app.suggestionBtn = createControl(`,
		`app.learningBtn = createControl(`,
	} {
		if strings.Contains(src, forbidden) {
			t.Fatalf("management control still clutters main workflow: %s", forbidden)
		}
	}
	for _, want := range []string{`"&Sources…"`, `"Vault &Destinations…"`, `"&Learning Center…"`, `"S&uggestion Center…"`} {
		if !strings.Contains(src, want) {
			t.Fatalf("management command missing from Tools menu: %s", want)
		}
	}
}

func TestLearningButtonDoesNotShowObservationCount(t *testing.T) {
	data, err := os.ReadFile("learning_center_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if strings.Contains(src, `fmt.Sprintf("Learning (%d)"`) {
		t.Fatal("Learning button still renders observation count")
	}
}

func TestNativePreviewZoomUsesRealKeyboardState(t *testing.T) {
	src := windowsMainSource(t)
	start := strings.Index(src, "func sendPreviewCtrlKey")
	end := strings.Index(src[start:], "func setPreviewZoom100")
	if start < 0 || end < 0 {
		t.Fatal("sendPreviewCtrlKey function not found")
	}
	body := src[start : start+end]
	if !strings.Contains(body, "procKeybdEvent.Call") {
		t.Fatal("native/PDF preview zoom is not using real keyboard input")
	}
	if strings.Contains(body, "procPostMessageW.Call(target") {
		t.Fatal("native/PDF preview zoom still posts modifier key messages that do not update keyboard state")
	}
}

func TestPDFUsesOwnedRendererBeforeWindowsPreviewHandler(t *testing.T) {
	src := windowsMainSource(t)
	loadStart := strings.Index(src, "func loadPreview(path string)")
	loadEnd := strings.Index(src[loadStart:], "func loadImage(path string)")
	if loadStart < 0 || loadEnd < 0 {
		t.Fatal("loadPreview function not found")
	}
	body := src[loadStart : loadStart+loadEnd]
	pdfPos := strings.Index(body, "startPDFPreview(path)")
	handlerPos := strings.Index(body, "previewHandlerCLSID(path)")
	if pdfPos < 0 {
		t.Fatal("PDF preview is not routed to DCM owned renderer")
	}
	if handlerPos >= 0 && pdfPos > handlerPos {
		t.Fatal("PDF owned renderer is attempted after Windows preview handler")
	}
}

func TestPDFZoomAndPageControlsUseOwnedRenderer(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		"pdfPreview.active",
		"requestPDFPreviewRender()",
		"goToPDFPage(-1)",
		"goToPDFPage(1)",
		"WM_PDF_RENDER_DONE",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("owned PDF preview wiring missing %q", want)
		}
	}
}

func TestPDFRenderedBitmapIsNotDoubleScaled(t *testing.T) {
	src := windowsMainSource(t)
	if !strings.Contains(src, "if pdfPreview.active {\n\t\t// Every owned PDF mode") {
		t.Fatal("PDF rendered bitmap is not using its renderer-owned zoom scale")
	}
}

func TestCtrlWheelRoutesByPointerInsidePreview(t *testing.T) {
	src := windowsMainSource(t)
	if !strings.Contains(src, "pointInWindow(app.previewHwnd, msg.Pt)") {
		t.Fatal("Ctrl+mouse-wheel routing still depends only on the message target instead of pointer position over preview")
	}
}

func TestMainWindowHasMenuBarAndAboutLivesUnderHelp(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`procCreateMenu`,
		`procCreatePopupMenu`,
		`procAppendMenuW`,
		`procSetMenu`,
		`ID_MENU_ABOUT`,
		`showAboutDialog()`,
		`"&Help"`,
		`"&About DCM - Quick Sort"`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("main menu/About wiring missing %q", want)
		}
	}
	if strings.Contains(src, `This is a free app. Consider Making Donation.`) && strings.Contains(src, `app.footerLabel = createControl`) {
		t.Fatal("About/donation copy is still rendered as the main-window footer instead of living in About")
	}
}

func TestRenameMoveNextAreIndependentMainActions(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`ID_RENAME`,
		`ID_MOVE`,
		`ID_NEXT`,
		`app.renameBtn = createControl("BUTTON", "&Rename"`,
		`app.moveBtn = createControl("BUTTON", "Move to &Vault"`,
		`app.nextBtn = createControl("BUTTON", "Ne&xt"`,
		`case ID_RENAME:`,
		`renameCurrent()`,
		`case ID_MOVE:`,
		`moveCurrent()`,
		`case ID_NEXT:`,
		`nextFile()`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("split Rename/Move/Next wiring missing %q", want)
		}
	}
	if strings.Contains(src, `&Rename + Move + Next`) {
		t.Fatal("combined Rename + Move + Next button still exists")
	}
}

func TestNamingTemplatesLiveUnderToolsMenu(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`ID_NAMING_TEMPLATES`,
		`"&Naming Templates…"`,
		`openNamingTemplateManager()`,
		`registerNamingTemplateManagerClass(hInst, cursor)`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("Naming Templates Tools integration missing %q", want)
		}
	}
}

func TestMainWindowNoLongerHasRawTemplateEditor(t *testing.T) {
	src := windowsMainSource(t)
	if strings.Contains(src, `createControl("STATIC", "Template:"`) {
		t.Fatal("raw naming-template editor is still on the main workflow")
	}
	if strings.Contains(src, `app.templateEdit = createControl(`) {
		t.Fatal("raw naming-template edit control is still created on the main workflow")
	}
}

func TestMainWorkflowRemovesMetadataButtonsAndRestoresFileFlowHandoff(t *testing.T) {
	src := windowsMainSource(t)
	for _, forbidden := range []string{
		`app.addTagBtn = createControl`,
		`app.addTodoBtn = createControl`,
		`app.addNoteBtn = createControl`,
	} {
		if strings.Contains(src, forbidden) {
			t.Fatalf("removed metadata button still exists on main workflow: %q", forbidden)
		}
	}
	for _, want := range []string{
		`app.fileFlowBtn = createControl("BUTTON", "Handoff to &FileFlow"`,
		`case ID_FILEFLOW_HANDOFF:`,
		`queueSelectedForFileFlow()`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("restored FileFlow handoff UI missing %q", want)
		}
	}
}

func TestMoveDoesNotRenameAndMetadataCommitHappensAfterSuccessfulMove(t *testing.T) {
	src := windowsMainSource(t)
	moveStart := strings.Index(src, "func moveCurrent()")
	if moveStart < 0 {
		t.Fatal("moveCurrent missing")
	}
	moveEnd := strings.Index(src[moveStart:], "func updateOpenDestinationAvailability")
	if moveEnd < 0 {
		t.Fatal("moveCurrent end missing")
	}
	body := src[moveStart : moveStart+moveEnd]
	if strings.Contains(body, "getText(app.nameEdit)") {
		t.Fatal("Move still consumes the uncommitted Name field")
	}
	if !strings.Contains(src, "commitStagedFileFlowMetadata(source, target, rec)") {
		t.Fatal("successful file operation does not commit staged FileFlow metadata")
	}
}

func TestMainSourceAndFilterRowsMatchRefinedWorkflow(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`createControl("STATIC", "Source:"`,
		`app.sourceAliasCombo = createControl(`,
		`app.sourceEdit = createControl(`,
		`app.sourceBtn = createControl("BUTTON", "&Browse…"`,
		`app.recoveryBtn = createControl("BUTTON", "Recovery"`,
		`app.reloadBtn = createControl("BUTTON", "Reload"`,
		`createControl("STATIC", "Name:"`,
		`app.filterNameEdit = createControl(`,
		`app.openFolderBtn = createControl("BUTTON", "Open &Folder"`,
		`createControl("STATIC", "File Type:"`,
		`createControl("STATIC", "Extension:"`,
		`createControl("STATIC", "Age:"`,
		`createControl("STATIC", "Size:"`,
		`app.filterClearBtn = createControl("BUTTON", "Clear &Filters"`,
		`createControl("STATIC", "New Name:"`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("refined source/filter layout missing %q", want)
		}
	}
	if strings.Contains(src, `createControl("STATIC", "Name contains:"`) {
		t.Fatal("legacy Name contains label remains instead of the dedicated Name filter row")
	}
}

func TestBottomWorkflowRowContainsRefinedPrimaryActions(t *testing.T) {
	src := windowsMainSource(t)
	start := strings.Index(src, `app.renameBtn = createControl("BUTTON", "&Rename"`)
	end := strings.Index(src[start:], `app.statusLabel = createControl`)
	if start < 0 || end < 0 {
		t.Fatal("bottom workflow row not found")
	}
	body := src[start : start+end]
	for _, want := range []string{
		`app.renameBtn = createControl`,
		`app.fileFlowBtn = createControl`,
		`app.moveBtn = createControl`,
		`app.nextBtn = createControl`,
		`app.openBtn = createControl`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("bottom workflow row missing %q", want)
		}
	}
	for _, forbidden := range []string{`app.addTagBtn = createControl`, `app.addTodoBtn = createControl`, `app.addNoteBtn = createControl`, `app.undoBtn = createControl`, `app.openFolderBtn = createControl`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("bottom workflow row still contains %q", forbidden)
		}
	}
}

func TestUndoLivesWithProcessedPaneAndReadAsTextLivesWithPreviewTools(t *testing.T) {
	src := windowsMainSource(t)
	if !strings.Contains(src, `app.undoBtn = createControl("BUTTON", "&Undo"`) {
		t.Fatal("Undo is not presented as a Processed-pane action")
	}
	if !strings.Contains(src, `app.readTextCheck = createControl("BUTTON", "Read as &text"`) {
		t.Fatal("Read as text is not presented with preview tools")
	}
}

func TestFilterRowIncludesSortByControl(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`createControl("STATIC", "Sort By:"`,
		`app.filterSortCombo = createControl(`,
		`case ID_FILTER_SORT:`,
		`selectedInboxSort()`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("sort-by UI/wiring missing %q", want)
		}
	}
}

func TestProcessedPaneHasSearchAndCollapsibleDateGroups(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`app.processedSearchEdit = createControl(`,
		`Search processed`,
		`case ID_PROCESSED_SEARCH:`,
		`toggleProcessedGroup`,
		`buildProcessedView`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("processed search/group UI missing %q", want)
		}
	}
}

func TestKeyboardFocusShortcutsAreWired(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`case ShortcutFocusNewName:`,
		`focusNewNameForTyping()`,
		`case ShortcutFocusDestinationPath:`,
		`focusDestinationForTyping()`,
		`case ShortcutFocusSourcePath:`,
		`focusSourceForTyping()`,
		`case ShortcutFocusFileList:`,
		`focusFileList()`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("keyboard focus shortcut wiring missing %q", want)
		}
	}
}

func TestTabFromFileListGoesDirectlyToNewName(t *testing.T) {
	src := windowsMainSource(t)
	if !strings.Contains(src, `if focus == app.fileList && !reverse {`) || !strings.Contains(src, `focusNewNameForTyping()`) {
		t.Fatal("Tab from file list does not jump directly to New Name")
	}
}

func TestSuccessfulRenameMoveAndNextReturnFocusToFileList(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`if mode == fileOperationRename && !sourceRemoved {`,
		`showCurrent(false)`,
		`focusFileList()`,
		`func removeCurrentFromWorkingList()`,
		`func nextFile()`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("file-list focus return wiring missing %q", want)
		}
	}
}

func TestEscapeFromMainEditorsReturnsToFileList(t *testing.T) {
	src := windowsMainSource(t)
	if !strings.Contains(src, `focus == app.nameEdit || focus == app.destEdit || focus == app.sourceEdit || focus == app.filterNameEdit || focus == app.processedSearchEdit`) {
		t.Fatal("Escape is not wired from primary edit boxes back to the file list")
	}
	if !strings.Contains(src, `focusFileList()`) {
		t.Fatal("Escape main-editor handling does not focus the file list")
	}
}

func TestKeyboardHintStripIsPresent(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`keyboardHintLabel`,
		`Alt+W Name`,
		`Alt+P Destination`,
		`Alt+C Source`,
		`Alt+L Files`,
		`↑↓ Navigate`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("keyboard hint strip missing %q", want)
		}
	}
}

func TestFileListSupportsExtendedMultiSelectionAndBatchMove(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		"LBS_EXTENDEDSEL",
		"LB_GETSELCOUNT",
		"LB_GETSELITEMS",
		"LB_GETCARETINDEX",
		"selectedSourceFiles()",
		"startBatchMove(",
		"advanceBatchMove()",
		"WM_BATCH_MOVE_NEXT",
		"Move %d to &Vault",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("multi-select/batch-move wiring missing %q", want)
		}
	}
}

func TestBatchMoveKeepsNormalPerFileJournalPipeline(t *testing.T) {
	src := windowsMainSource(t)
	if !strings.Contains(src, "beginFileOperation(fileOperationMove, item.Source, item.Target)") {
		t.Fatal("batch move does not feed each file through the normal journal/conflict pipeline")
	}
	if !strings.Contains(src, "batchMoveStepFinished(") {
		t.Fatal("batch move completion chaining is not wired")
	}
}

func TestBatchMoveDefersProcessedRefreshUntilBatchFinishes(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`if !app.batchMoveActive {
		populateProcessedList(len(app.history) - 1)
	}`,
		`if !app.batchMoveActive {
			populateProcessedList(-1)
		}`,
		"app.batchProcessedStart = len(app.history)",
		"preferredProcessed := -1",
		"populateProcessedList(preferredProcessed)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("batch processed refresh deferral missing %q", want)
		}
	}
}

func TestEveryNewPreviewStartsInFitMode(t *testing.T) {
	src := windowsMainSource(t)
	start := strings.Index(src, "func loadPreview(path string)")
	end := strings.Index(src[start:], "func loadImage(path string)")
	if start < 0 || end < 0 {
		t.Fatal("loadPreview function not found")
	}
	body := src[start : start+end]
	resetPos := strings.Index(body, "resetPreviewZoomState()")
	plainPos := strings.Index(body, "shouldUsePlainTextPreview(path)")
	pdfPos := strings.Index(body, "startPDFPreview(path)")
	imagePos := strings.Index(body, "isImage(path)")
	if resetPos < 0 {
		t.Fatal("loadPreview does not reset to Fit before opening a new file")
	}
	for label, pos := range map[string]int{"plain text": plainPos, "PDF": pdfPos, "image": imagePos} {
		if pos < 0 || resetPos > pos {
			t.Fatalf("preview Fit reset does not occur before %s dispatch", label)
		}
	}
}

func TestFitButtonSupportsAllDCMPreviewFamilies(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		"app.textPreviewActive",
		"app.xlsxPreviewActive",
		"app.pptxPreviewActive",
		"app.image != 0",
		"pdfPreview.active",
		"app.preview != nil",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("preview family missing from Fit support wiring: %q", want)
		}
	}
	fitStart := strings.Index(src, "func fitPreview()")
	fitEnd := strings.Index(src[fitStart:], "func adjustPreviewZoom")
	if fitStart < 0 || fitEnd < 0 {
		t.Fatal("fitPreview function not found")
	}
	fitBody := src[fitStart : fitStart+fitEnd]
	for _, want := range []string{"app.textPreviewActive", "app.xlsxPreviewActive", "app.preview != nil"} {
		if !strings.Contains(fitBody, want) {
			t.Fatalf("Fit action does not handle %q", want)
		}
	}
}

func TestPlainTextPreviewWrapsInsidePane(t *testing.T) {
	src := windowsMainSource(t)
	lineStart := strings.Index(src, `app.textPreviewEdit = createControl("EDIT"`)
	if lineStart < 0 {
		t.Fatal("plain-text preview control creation missing")
	}
	lineEnd := strings.Index(src[lineStart:], "\n")
	line := src[lineStart : lineStart+lineEnd]
	if strings.Contains(line, "WS_HSCROLL") || strings.Contains(line, "ES_AUTOHSCROLL") {
		t.Fatal("plain-text preview can overflow horizontally instead of wrapping to the preview pane")
	}
}

func TestPreviewWheelUsesRawDeltaAndSeparatePanFromZoom(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		"consumePreviewWheelDetents(&app.previewZoomWheelRemainder",
		"handlePreviewWheelPan(delta, keyDown(VK_SHIFT))",
		"previewPanPixels(delta, viewport)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("preview wheel routing missing %q", want)
		}
	}
	if strings.Contains(src, "direction*previewZoomStep") {
		t.Fatal("preview zoom still uses fixed 25-point jumps")
	}
}

func TestFitWidthControlAndPDFFitScaleAreWired(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		"ID_ZOOM_FIT_WIDTH",
		"zoomFitWidthBtn",
		`createControl("BUTTON", "Width"`,
		"fitPreviewWidth()",
		"pdfPreview.lastScalePercent",
		"Fit Width",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("fit-width/scale wiring missing %q", want)
		}
	}
}

func TestExplicitPreviewZoomModesClearWheelRemainder(t *testing.T) {
	src := windowsMainSource(t)
	for _, fn := range []string{"func setPreviewZoom100()", "func fitPreview()", "func fitPreviewWidth()"} {
		start := strings.Index(src, fn)
		if start < 0 {
			t.Fatalf("%s missing", fn)
		}
		end := strings.Index(src[start:], "\n}")
		if end < 0 || !strings.Contains(src[start:start+end], "app.previewZoomWheelRemainder = 0") {
			t.Fatalf("%s does not clear high-resolution wheel remainder", fn)
		}
	}
}

func TestFitWidthResetsVerticalPanToTop(t *testing.T) {
	src := windowsMainSource(t)
	start := strings.Index(src, "func fitPreviewWidth()")
	if start < 0 {
		t.Fatal("fitPreviewWidth missing")
	}
	end := strings.Index(src[start:], "\n}\n\nfunc adjustPreviewZoom")
	if end < 0 {
		t.Fatal("fitPreviewWidth end missing")
	}
	body := src[start : start+end]
	if strings.Count(body, "previewTopPanOffset(") < 2 {
		t.Fatal("Fit Width does not top-anchor both PPTX and image previews")
	}

	pdfData, err := os.ReadFile("pdf_preview_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	pdfSrc := strings.ReplaceAll(string(pdfData), "\r\n", "\n")
	pdfStart := strings.Index(pdfSrc, "func finishPDFRender()")
	if pdfStart < 0 {
		t.Fatal("finishPDFRender missing")
	}
	pdfEnd := strings.Index(pdfSrc[pdfStart:], "\n}\n\nfunc finishPDFRendererUnavailable")
	if pdfEnd < 0 {
		t.Fatal("finishPDFRender end missing")
	}
	pdfBody := pdfSrc[pdfStart : pdfStart+pdfEnd]
	if !strings.Contains(pdfBody, "app.previewFitWidth") || !strings.Contains(pdfBody, "previewTopPanOffset(") {
		t.Fatal("PDF Fit Width render completion does not reset the viewport to the top")
	}
}

func TestPDFDefaultViewPreferenceMenuAndStartupAreWired(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		"PDF Default &View",
		"Fit &Width (recommended)",
		"Fit &Page",
		"ID_PDF_DEFAULT_FIT_WIDTH",
		"ID_PDF_DEFAULT_FIT_PAGE",
		"loadPDFDefaultViewSetting()",
		"setPDFDefaultViewPreference(pdfDefaultViewFitWidth)",
		"setPDFDefaultViewPreference(pdfDefaultViewFitPage)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("PDF default-view preference wiring missing %q", want)
		}
	}
	pdfData, err := os.ReadFile("pdf_preview_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	pdfSrc := strings.ReplaceAll(string(pdfData), "\r\n", "\n")
	for _, want := range []string{
		"normalizePDFDefaultView(app.pdfDefaultView)",
		"app.previewFitWidth = defaultView == pdfDefaultViewFitWidth",
		"app.previewFit = !app.previewFitWidth",
	} {
		if !strings.Contains(pdfSrc, want) {
			t.Fatalf("PDF startup default-view wiring missing %q", want)
		}
	}
}

func TestVaultLibraryMenuAndMoveUndoIndexWiring(t *testing.T) {
	src := windowsMainSource(t)
	for _, needle := range []string{
		"ID_VAULT_LIBRARY",
		"Vault &Library…",
		"chooseVaultRoot()",
		"syncConfiguredVaultMove(source, target)",
		"syncConfiguredVaultMove(rec.To, rec.From)",
	} {
		if !strings.Contains(src, needle) {
			t.Fatalf("Windows vault wiring missing %q", needle)
		}
	}
}

func TestBatchUndoButtonAndJournalTaggingAreWired(t *testing.T) {
	src := windowsMainSource(t)
	for _, want := range []string{
		`ID_UNDO_BATCH`,
		`app.undoBatchBtn = createControl("BUTTON", "Undo &Batch"`,
		`case ID_UNDO_BATCH:`,
		`undoLatestBatch()`,
		`BatchID: app.batchMoveID`,
		`result.Record.BatchID = journalRec.BatchID`,
		`batchID = app.batchMoveID`,
		`removeExactDuplicateSourceJournaled(moveJournalFilePath(), source, target, batchID)`,
		`replaceMoveWithRecoveryJournaled(moveJournalFilePath(), source, target, replacementRecoveryDirectory(), batchID)`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("Batch Undo Windows wiring missing %q", want)
		}
	}
	if strings.Contains(src, `result.Record.BatchID = app.batchMoveID`) {
		t.Fatal("conflict completion must preserve the admitted batch instead of restamping a later active batch")
	}
}
