//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	WM_CREATE                   = 0x0001
	WM_DESTROY                  = 0x0002
	WM_SIZE                     = 0x0005
	WM_SETFOCUS                 = 0x0007
	WM_CLOSE                    = 0x0010
	WM_PAINT                    = 0x000F
	WM_ERASEBKGND               = 0x0014
	WM_DRAWITEM                 = 0x002B
	WM_COMMAND                  = 0x0111
	WM_HSCROLL                  = 0x0114
	WM_VSCROLL                  = 0x0115
	WM_CTLCOLOREDIT             = 0x0133
	WM_CTLCOLORBTN              = 0x0135
	WM_CTLCOLORSTATIC           = 0x0138
	WM_CTLCOLORLISTBOX          = 0x0134
	WM_KEYDOWN                  = 0x0100
	WM_KEYUP                    = 0x0101
	WM_SYSKEYDOWN               = 0x0104
	WM_MOUSEMOVE                = 0x0200
	WM_LBUTTONDOWN              = 0x0201
	WM_LBUTTONUP                = 0x0202
	WM_MOUSEWHEEL               = 0x020A
	WM_CHANGEUISTATE            = 0x0127
	WM_SETFONT                  = 0x0030
	WM_GETFONT                  = 0x0031
	WM_BACKGROUND_MOVE_DONE     = 0x8001 // WM_APP + 1
	WM_BACKGROUND_CONFLICT_DONE = 0x8002 // WM_APP + 2
	WM_BACKGROUND_ACTION_DONE   = 0x8003 // WM_APP + 3
	WM_BACKGROUND_MOVE_PROGRESS = 0x8004 // WM_APP + 4
	WM_BATCH_MOVE_NEXT          = 0x8005 // WM_APP + 5
	EM_SETLIMITTEXT             = 0x00C5
	EM_SETSEL                   = 0x00B1
	EM_SETCUEBANNER             = 0x1501

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_THICKFRAME       = 0x00040000
	WS_MAXIMIZEBOX      = 0x00010000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_CLIPCHILDREN     = 0x02000000
	WS_TABSTOP          = 0x00010000
	WS_BORDER           = 0x00800000
	ES_AUTOHSCROLL      = 0x0080
	ES_READONLY         = 0x0800
	BS_PUSHBUTTON       = 0x00000000
	BS_AUTOCHECKBOX     = 0x00000003
	BS_OWNERDRAW        = 0x0000000B
	SS_LEFT             = 0x00000000
	CBS_DROPDOWN        = 0x0002
	CBS_DROPDOWNLIST    = 0x0003
	CBS_AUTOHSCROLL     = 0x0040
	LBS_NOTIFY          = 0x0001
	LBS_EXTENDEDSEL     = 0x0800
	WS_VSCROLL          = 0x00200000
	WS_HSCROLL          = 0x00100000
	ES_MULTILINE        = 0x0004
	ES_AUTOVSCROLL      = 0x0040

	CW_USEDEFAULT = 0x80000000
	SW_SHOW       = 5
	SW_SHOWNORMAL = 1

	COLOR_WINDOW     = 5
	IDC_ARROW        = 32512
	DEFAULT_GUI_FONT = 17
	WHITE_BRUSH      = 0
	TRANSPARENT      = 1
	OPAQUE           = 2
	FW_NORMAL        = 400
	FW_SEMIBOLD      = 600
	PS_SOLID         = 0
	ODS_SELECTED     = 0x0001
	ODS_DISABLED     = 0x0004
	ODS_FOCUS        = 0x0010

	MF_STRING    = 0x0000
	MF_CHECKED   = 0x0008
	MF_POPUP     = 0x0010
	MF_SEPARATOR = 0x0800

	BN_CLICKED    = 0
	BST_CHECKED   = 1
	BM_GETCHECK   = 0x00F0
	BM_SETCHECK   = 0x00F1
	BM_CLICK      = 0x00F5
	CBN_SELCHANGE = 1
	EN_CHANGE     = 0x0300
	LBN_SELCHANGE = 1

	CB_ADDSTRING    = 0x0143
	CB_RESETCONTENT = 0x014B
	CB_GETCURSEL    = 0x0147
	CB_SETCURSEL    = 0x014E
	CB_SHOWDROPDOWN = 0x014F

	LB_ADDSTRING     = 0x0180
	LB_RESETCONTENT  = 0x0184
	LB_SETSEL        = 0x0185
	LB_SETCURSEL     = 0x0186
	LB_GETSEL        = 0x0187
	LB_GETCURSEL     = 0x0188
	LB_GETTEXT       = 0x0189
	LB_GETTEXTLEN    = 0x018A
	LB_GETSELCOUNT   = 0x0190
	LB_GETSELITEMS   = 0x0191
	LB_SETCARETINDEX = 0x019E
	LB_GETCARETINDEX = 0x019F
	LB_ERR           = ^uintptr(0)

	VK_TAB       = 0x09
	VK_RETURN    = 0x0D
	VK_SHIFT     = 0x10
	VK_ESCAPE    = 0x1B
	VK_UP        = 0x26
	VK_DOWN      = 0x28
	VK_CONTROL   = 0x11
	VK_MENU      = 0x12
	VK_A         = 0x41
	VK_Z         = 0x5A
	VK_0         = 0x30
	VK_ADD       = 0x6B
	VK_SUBTRACT  = 0x6D
	VK_OEM_PLUS  = 0xBB
	VK_OEM_MINUS = 0xBD
	VK_F5        = 0x74
	VK_OEM_5     = 0xDC // backslash on US/Windows keyboards

	MK_LBUTTON   = 0x0001
	SB_LINEUP    = 0
	SB_LINEDOWN  = 1
	SB_LINELEFT  = 0
	SB_LINERIGHT = 1

	KEYEVENTF_KEYUP = 0x0002

	BIF_RETURNONLYFSDIRS = 0x0001
	BIF_NEWDIALOGSTYLE   = 0x0040

	FOS_PICKFOLDERS     = 0x00000020
	FOS_FORCEFILESYSTEM = 0x00000040
	FOS_PATHMUSTEXIST   = 0x00000800
	FOS_FILEMUSTEXIST   = 0x00001000
	SIGDN_FILESYSPATH   = 0x80058000

	UIS_CLEAR      = 2
	UISF_HIDEACCEL = 0x0002

	COINIT_APARTMENTTHREADED = 0x2
	CLSCTX_INPROC_SERVER     = 0x1
	CLSCTX_LOCAL_SERVER      = 0x4

	S_OK          = 0x00000000
	S_FALSE       = 0x00000001
	E_NOINTERFACE = 0x80004002
	E_POINTER     = 0x80004003

	STGM_READ             = 0x00000000
	STGM_SHARE_DENY_NONE  = 0x00000040
	FILE_ATTRIBUTE_NORMAL = 0x00000080
	GMEM_MOVEABLE         = 0x0002

	RRF_RT_REG_SZ = 0x00000002

	ID_SOURCE_EDIT       = 1001
	ID_SOURCE_BTN        = 1002
	ID_DEST_EDIT         = 1003
	ID_DEST_BTN          = 1004
	ID_NAME_EDIT         = 1005
	ID_RENAME            = 1006
	ID_NEXT              = 1007
	ID_UNDO              = 1008
	ID_OPEN              = 1009
	ID_OPEN_FOLDER       = 1010
	ID_RELOAD            = 1011
	ID_CATEGORY          = 1012
	ID_EXTENSION         = 1013
	ID_READ_TEXT         = 1014
	ID_DEST_LIST         = 1015
	ID_FILE_LIST         = 1016
	ID_PROCESSED_LIST    = 1017
	ID_DEST_ALIAS        = 1018
	ID_DEST_CONFIG       = 1019
	ID_ZOOM_OUT          = 1020
	ID_ZOOM_100          = 1021
	ID_ZOOM_IN           = 1022
	ID_ZOOM_FIT          = 1023
	ID_XLSX_SHEET        = 1024
	ID_PPTX_SLIDE        = 1025
	ID_OPEN_DEST         = 1026
	ID_RENAME_TOOL       = 1027
	ID_APPLY_RENAME      = 1028
	ID_TEMPLATE_EDIT     = 1029
	ID_FILTER_NAME       = 1030
	ID_FILTER_AGE        = 1031
	ID_FILTER_SIZE       = 1032
	ID_FILTER_CLEAR      = 1033
	ID_RECOVERY_CENTER   = 1034
	ID_LEARNING_CENTER   = 1035
	ID_SUGGESTION_CENTER = 1036
	ID_FILEFLOW_HANDOFF  = 1037
	ID_SOURCE_ALIAS      = 1038
	ID_SOURCE_CONFIG     = 1039
	ID_SOURCE_LIST       = 1040
	ID_MOVE              = 1041
	ID_ADD_TAG           = 1042
	ID_ADD_TODO          = 1043
	ID_ADD_NOTE          = 1044
	ID_FILTER_SORT       = 1045
	ID_PROCESSED_SEARCH  = 1046
	ID_ZOOM_FIT_WIDTH    = 1047
	ID_UNDO_BATCH        = 1048

	ID_MENU_EXIT             = 2001
	ID_MENU_ABOUT            = 2002
	ID_NAMING_TEMPLATES      = 2003
	ID_PDF_DEFAULT_FIT_WIDTH = 2004
	ID_PDF_DEFAULT_FIT_PAGE  = 2005
	ID_VAULT_LIBRARY         = 2006

	ID_DM_VAULT_ROOT      = 1101
	ID_DM_CHANGE_VAULT    = 1102
	ID_DM_SEARCH          = 1103
	ID_DM_LOCATION_LIST   = 1104
	ID_DM_LOGICAL         = 1105
	ID_DM_PHYSICAL        = 1106
	ID_DM_CHILD_NAME      = 1107
	ID_DM_CREATE_CHILD    = 1108
	ID_DM_FAVORITE_NAME   = 1109
	ID_DM_SAVE_FAVORITE   = 1110
	ID_DM_FAVORITES_LIST  = 1111
	ID_DM_DELETE_FAVORITE = 1112
	ID_DM_OPEN_LOCATION   = 1113
	ID_DM_REBUILD_INDEX   = 1114
	ID_DM_USE_LOCATION    = 1115
	ID_DM_CLOSE           = 1116
	ID_DM_EXPORT_CONFIG   = 1117
	ID_DM_IMPORT_CONFIG   = 1118
	ID_DM_RESTORE_LATEST  = 1119
	ID_DM_BACKUP_FOLDER   = 1120
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	gdiplus  = syscall.NewLazyDLL("gdiplus.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	shlwapi  = syscall.NewLazyDLL("shlwapi.dll")
	uxtheme  = syscall.NewLazyDLL("uxtheme.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procPostMessageW         = user32.NewProc("PostMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procSetWindowTextW       = user32.NewProc("SetWindowTextW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procSendMessageW         = user32.NewProc("SendMessageW")
	procMoveWindow           = user32.NewProc("MoveWindow")
	procRedrawWindow         = user32.NewProc("RedrawWindow")
	procGetClientRect        = user32.NewProc("GetClientRect")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procBeginPaint           = user32.NewProc("BeginPaint")
	procEndPaint             = user32.NewProc("EndPaint")
	procFillRect             = user32.NewProc("FillRect")
	procDrawTextW            = user32.NewProc("DrawTextW")
	procInvalidateRect       = user32.NewProc("InvalidateRect")
	procSetFocus             = user32.NewProc("SetFocus")
	procSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	procGetFocus             = user32.NewProc("GetFocus")
	procGetParent            = user32.NewProc("GetParent")
	procLoadCursorW          = user32.NewProc("LoadCursorW")
	procMessageBoxW          = user32.NewProc("MessageBoxW")
	procGetKeyState          = user32.NewProc("GetKeyState")
	procKeybdEvent           = user32.NewProc("keybd_event")
	procEnableWindow         = user32.NewProc("EnableWindow")
	procIsWindowEnabled      = user32.NewProc("IsWindowEnabled")
	procIsWindowVisible      = user32.NewProc("IsWindowVisible")
	procSetCapture           = user32.NewProc("SetCapture")
	procReleaseCapture       = user32.NewProc("ReleaseCapture")
	procCreateMenu           = user32.NewProc("CreateMenu")
	procCreatePopupMenu      = user32.NewProc("CreatePopupMenu")
	procAppendMenuW          = user32.NewProc("AppendMenuW")
	procCheckMenuItem        = user32.NewProc("CheckMenuItem")
	procSetMenu              = user32.NewProc("SetMenu")
	procDrawMenuBar          = user32.NewProc("DrawMenuBar")
	procDestroyMenu          = user32.NewProc("DestroyMenu")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
	procGetStockObject   = gdi32.NewProc("GetStockObject")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procSetBkColor       = gdi32.NewProc("SetBkColor")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procCreatePen        = gdi32.NewProc("CreatePen")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procRoundRect        = gdi32.NewProc("RoundRect")
	procRectangle        = gdi32.NewProc("Rectangle")
	procDrawFocusRect    = user32.NewProc("DrawFocusRect")

	procSHBrowseForFolderW          = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW        = shell32.NewProc("SHGetPathFromIDListW")
	procShellExecuteW               = shell32.NewProc("ShellExecuteW")
	procSHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")

	procGdiplusStartup      = gdiplus.NewProc("GdiplusStartup")
	procGdiplusShutdown     = gdiplus.NewProc("GdiplusShutdown")
	procGdipLoadImageStream = gdiplus.NewProc("GdipLoadImageFromStream")
	procGdipDisposeImage    = gdiplus.NewProc("GdipDisposeImage")
	procGdipGetImageWidth   = gdiplus.NewProc("GdipGetImageWidth")
	procGdipGetImageHeight  = gdiplus.NewProc("GdipGetImageHeight")
	procGdipCreateFromHDC   = gdiplus.NewProc("GdipCreateFromHDC")
	procGdipDeleteGraphics  = gdiplus.NewProc("GdipDeleteGraphics")
	procGdipDrawImageRectI  = gdiplus.NewProc("GdipDrawImageRectI")

	procCoInitializeEx        = ole32.NewProc("CoInitializeEx")
	procCoUninitialize        = ole32.NewProc("CoUninitialize")
	procCoCreateInstance      = ole32.NewProc("CoCreateInstance")
	procCLSIDFromString       = ole32.NewProc("CLSIDFromString")
	procCoTaskMemFree         = ole32.NewProc("CoTaskMemFree")
	procCreateStreamOnHGlobal = ole32.NewProc("CreateStreamOnHGlobal")

	procRegGetValueW           = advapi32.NewProc("RegGetValueW")
	procSHCreateStreamOnFileEx = shlwapi.NewProc("SHCreateStreamOnFileEx")
	procIUnknownSetSite        = shlwapi.NewProc("IUnknown_SetSite")
	procSetWindowTheme         = uxtheme.NewProc("SetWindowTheme")
	procDwmSetWindowAttribute  = dwmapi.NewProc("DwmSetWindowAttribute")
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }

func pointInWindow(hwnd uintptr, pt POINT) bool {
	if hwnd == 0 {
		return false
	}
	var rc RECT
	ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	if ok == 0 {
		return false
	}
	return pt.X >= rc.Left && pt.X < rc.Right && pt.Y >= rc.Top && pt.Y < rc.Bottom
}

type MSG struct {
	Hwnd     uintptr
	Message  uint32
	_        uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}
type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}
type PAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}
type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	HDC        uintptr
	RcItem     RECT
	ItemData   uintptr
}
type COMDLG_FILTERSPEC struct {
	PszName *uint16
	PszSpec *uint16
}

type BROWSEINFO struct {
	HwndOwner      uintptr
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}
type GdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type iPreviewHandler struct{ Vtbl *iPreviewHandlerVtbl }
type iPreviewHandlerVtbl struct {
	QueryInterface       uintptr
	AddRef               uintptr
	Release              uintptr
	SetWindow            uintptr
	SetRect              uintptr
	DoPreview            uintptr
	Unload               uintptr
	SetFocus             uintptr
	QueryFocus           uintptr
	TranslateAccelerator uintptr
}
type iInitializeWithFile struct{ Vtbl *iInitializeWithFileVtbl }
type iInitializeWithFileVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Initialize     uintptr
}
type iInitializeWithStream struct{ Vtbl *iInitializeWithStreamVtbl }
type iInitializeWithStreamVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Initialize     uintptr
}
type iInitializeWithItem struct{ Vtbl *iInitializeWithItemVtbl }
type iInitializeWithItemVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Initialize     uintptr
}
type iUnknown struct{ Vtbl *iUnknownVtbl }
type iUnknownVtbl struct{ QueryInterface, AddRef, Release uintptr }

type iFileOpenDialog struct{ Vtbl *iFileOpenDialogVtbl }
type iFileOpenDialogVtbl struct {
	QueryInterface, AddRef, Release uintptr
	Show                            uintptr
	SetFileTypes                    uintptr
	SetFileTypeIndex                uintptr
	GetFileTypeIndex                uintptr
	Advise                          uintptr
	Unadvise                        uintptr
	SetOptions                      uintptr
	GetOptions                      uintptr
	SetDefaultFolder                uintptr
	SetFolder                       uintptr
	GetFolder                       uintptr
	GetCurrentSelection             uintptr
	SetFileName                     uintptr
	GetFileName                     uintptr
	SetTitle                        uintptr
	SetOkButtonLabel                uintptr
	SetFileNameLabel                uintptr
	GetResult                       uintptr
	AddPlace                        uintptr
	SetDefaultExtension             uintptr
	Close                           uintptr
	SetClientGuid                   uintptr
	ClearClientData                 uintptr
	SetFilter                       uintptr
	GetResults                      uintptr
	GetSelectedItems                uintptr
}

type iShellItem struct{ Vtbl *iShellItemVtbl }
type iShellItemVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	BindToHandler  uintptr
	GetParent      uintptr
	GetDisplayName uintptr
	GetAttributes  uintptr
	Compare        uintptr
}

type previewHandlerFrameInfo struct {
	Haccel        uintptr
	CAccelEntries uint32
}

type previewHostSiteVtbl struct {
	QueryInterface       uintptr
	AddRef               uintptr
	Release              uintptr
	GetWindowContext     uintptr
	TranslateAccelerator uintptr
}

type previewHostSite struct {
	Vtbl *previewHostSiteVtbl
	refs uint32
}

type destinationManagerState struct {
	hwnd                                                                    uintptr
	vaultRootEdit, changeVaultBtn, searchEdit, locationList                 uintptr
	logicalEdit, physicalEdit, childNameEdit, createChildBtn                uintptr
	favoriteNameEdit, saveFavoriteBtn, favoritesList, deleteFavoriteBtn     uintptr
	openLocationBtn, rebuildIndexBtn, useLocationBtn, closeBtn, statusLabel uintptr
	exportConfigBtn, importConfigBtn, restoreLatestBtn, backupFolderBtn     uintptr
	locations                                                               []VaultAddressEntry
	selectedLocation, selectedFavorite                                      int
}

type pptxLoadedImage struct {
	shapeIndex int
	image      uintptr
	stream     *iUnknown
}

type fileOperationMode uint8

const (
	fileOperationNone fileOperationMode = iota
	fileOperationRename
	fileOperationMove
)

type appState struct {
	hwnd                                                                                                                                               uintptr
	mainMenu, pdfDefaultViewMenu                                                                                                                       uintptr
	sourceEdit, destEdit, nameEdit, categoryCombo, extensionCombo, readTextCheck                                                                       uintptr
	sourceAliasCombo, sourceSuggestionList, sourceConfigBtn                                                                                            uintptr
	filterNameEdit, filterAgeCombo, filterSizeCombo, filterSortLabel, filterSortCombo, filterClearBtn                                                  uintptr
	renameToolCombo, renameToolBtn                                                                                                                     uintptr
	destAliasCombo, destSuggestionList, fileList, processedList, processedSearchEdit                                                                   uintptr
	currentLabel, countLabel, processedTitleLabel, statusLabel, keyboardHintLabel                                                                      uintptr
	nameLabel, renameToolLabel, destinationLabel, pathLabel                                                                                            uintptr
	sourceBtn, destBtn, destOpenBtn, destConfigBtn, renameBtn, fileFlowBtn, moveBtn, nextBtn, undoBtn, undoBatchBtn, openBtn, openFolderBtn, reloadBtn uintptr
	recoveryBtn, learningBtn, suggestionBtn                                                                                                            uintptr
	zoomOutBtn, zoom100Btn, zoomInBtn, zoomFitBtn, zoomFitWidthBtn                                                                                     uintptr
	previewHwnd, textPreviewEdit, xlsxSheetCombo, xlsxGridEdit, pptxSlideCombo                                                                         uintptr
	uiFont, headingFont, smallFont, textPreviewFont, xlsxGridFont                                                                                      uintptr
	brushWindow, brushPanel, brushStatus, brushFooter                                                                                                  uintptr
	sourceDir, destDir                                                                                                                                 string
	pdfDefaultView                                                                                                                                     string
	startupPreferredFile                                                                                                                               string
	files                                                                                                                                              []string
	index                                                                                                                                              int
	history                                                                                                                                            []MoveRecord
	processedRows                                                                                                                                      []processedViewRow
	processedCollapsed                                                                                                                                 map[string]bool
	gpToken                                                                                                                                            uintptr
	image                                                                                                                                              uintptr
	imageStream                                                                                                                                        *iUnknown
	preview                                                                                                                                            *iPreviewHandler
	previewActive                                                                                                                                      bool
	fallback                                                                                                                                           string
	updatingSuggestions, updatingFileList, updatingProcessedList                                                                                       bool
	updatingInboxFilters                                                                                                                               bool
	destSuggestions                                                                                                                                    []string
	destAliases                                                                                                                                        []DestinationAlias
	destSuggestionsVisible                                                                                                                             bool
	updatingDestinationAlias                                                                                                                           bool
	sourceSuggestions                                                                                                                                  []string
	sourceAliases                                                                                                                                      []SourceAlias
	sourceSuggestionsVisible                                                                                                                           bool
	updatingSourceAlias, updatingSourceSuggestions                                                                                                     bool
	previewFit                                                                                                                                         bool
	previewFitWidth                                                                                                                                    bool
	previewZoom                                                                                                                                        int
	previewZoomWheelRemainder                                                                                                                          int
	previewPanX, previewPanY                                                                                                                           int32
	previewDragging                                                                                                                                    bool
	textPreviewActive                                                                                                                                  bool
	textPreviewKind                                                                                                                                    string
	xlsxPreviewActive                                                                                                                                  bool
	xlsxWorkbook                                                                                                                                       XLSXWorkbookPreview
	xlsxSheetIndex                                                                                                                                     int
	pptxPreviewActive                                                                                                                                  bool
	pptxDeck                                                                                                                                           PPTXPresentationPreview
	pptxSlideIndex                                                                                                                                     int
	pptxImages                                                                                                                                         []pptxLoadedImage
	previewDragX, previewDragY, previewPanStartX, previewPanStartY                                                                                     int32
	backgroundMoveBusy                                                                                                                                 bool
	pendingFileOperation                                                                                                                               fileOperationMode
	activeMoveJournal                                                                                                                                  MoveRecord
	batchMoveActive                                                                                                                                    bool
	batchMovePlan                                                                                                                                      []batchMovePlanItem
	batchMoveTotal, batchMoveCompleted, batchMoveIssues                                                                                                int
	batchProcessedStart                                                                                                                                int
	batchMoveID                                                                                                                                        string
	currentSuggestion                                                                                                                                  LearnedSuggestion
	suggestionDismissedFor                                                                                                                             string
}

var app appState
var backgroundMoveResults = make(chan BackgroundMoveResult, 1)
var backgroundConflictResults = make(chan BackgroundConflictResult, 1)
var backgroundConflictActionResults = make(chan BackgroundConflictActionResult, 1)
var backgroundMoveProgress = make(chan BackgroundMoveProgress, 16)
var destManager destinationManagerState
var mainWndProc = syscall.NewCallback(mainWindowProc)
var destinationManagerWndProcCB = syscall.NewCallback(destinationManagerWndProc)
var previewWndProc = syscall.NewCallback(previewWindowProc)
var previewHostQueryInterfaceCB = syscall.NewCallback(previewHostQueryInterface)
var previewHostAddRefCB = syscall.NewCallback(previewHostAddRef)
var previewHostReleaseCB = syscall.NewCallback(previewHostRelease)
var previewHostGetWindowContextCB = syscall.NewCallback(previewHostGetWindowContext)
var previewHostTranslateAcceleratorCB = syscall.NewCallback(previewHostTranslateAccelerator)
var previewSiteVtbl = previewHostSiteVtbl{
	QueryInterface:       previewHostQueryInterfaceCB,
	AddRef:               previewHostAddRefCB,
	Release:              previewHostReleaseCB,
	GetWindowContext:     previewHostGetWindowContextCB,
	TranslateAccelerator: previewHostTranslateAcceleratorCB,
}
var previewSite = previewHostSite{Vtbl: &previewSiteVtbl, refs: 1}
var iidIUnknown = GUID{Data1: 0x00000000, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
var iidPreviewHandlerFrame = GUID{Data1: 0xFEC87AAF, Data2: 0x35F9, Data3: 0x447A, Data4: [8]byte{0xAD, 0xB7, 0x20, 0x23, 0x44, 0x91, 0x40, 0x1A}}

func wstr(s string) *uint16               { p, _ := syscall.UTF16PtrFromString(s); return p }
func loword(v uintptr) uint16             { return uint16(v & 0xffff) }
func hiword(v uintptr) uint16             { return uint16((v >> 16) & 0xffff) }
func signedLoword(v uintptr) int32        { return int32(int16(loword(v))) }
func signedHiword(v uintptr) int32        { return int32(int16(hiword(v))) }
func makeWParam(low, high uint16) uintptr { return uintptr(low) | uintptr(high)<<16 }
func hresultFailed(v uintptr) bool        { return int32(v) < 0 }

func previewHostQueryInterface(this, riidPtr, ppvPtr uintptr) uintptr {
	if riidPtr == 0 || ppvPtr == 0 {
		return E_POINTER
	}
	*(*uintptr)(unsafe.Pointer(ppvPtr)) = 0
	riid := (*GUID)(unsafe.Pointer(riidPtr))
	if *riid != iidIUnknown && *riid != iidPreviewHandlerFrame {
		return E_NOINTERFACE
	}
	*(*uintptr)(unsafe.Pointer(ppvPtr)) = this
	previewHostAddRef(this)
	return S_OK
}

func previewHostAddRef(this uintptr) uintptr {
	if this == 0 {
		return 0
	}
	site := (*previewHostSite)(unsafe.Pointer(this))
	return uintptr(atomic.AddUint32(&site.refs, 1))
}

func previewHostRelease(this uintptr) uintptr {
	if this == 0 {
		return 0
	}
	site := (*previewHostSite)(unsafe.Pointer(this))
	for {
		old := atomic.LoadUint32(&site.refs)
		if old <= 1 {
			return 1
		}
		if atomic.CompareAndSwapUint32(&site.refs, old, old-1) {
			return uintptr(old - 1)
		}
	}
}

func previewHostGetWindowContext(this, infoPtr uintptr) uintptr {
	if infoPtr == 0 {
		return E_POINTER
	}
	info := (*previewHandlerFrameInfo)(unsafe.Pointer(infoPtr))
	info.Haccel = 0
	info.CAccelEntries = 0
	return S_OK
}

func previewHostTranslateAccelerator(this, msgPtr uintptr) uintptr {
	return S_FALSE
}

func rgb(r, g, b byte) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }

var (
	colorInk       = rgb(23, 32, 51)
	colorMuted     = rgb(98, 112, 133)
	colorAccent    = rgb(24, 89, 183)
	colorAccentHot = rgb(14, 63, 136)
	colorWindowBG  = rgb(245, 247, 250)
	colorPanelBG   = rgb(255, 255, 255)
	colorStatusBG  = rgb(234, 242, 255)
	colorFooterBG  = rgb(247, 248, 250)
	colorBorder    = rgb(216, 222, 231)
)

func createFontFace(height int32, weight int32, face string) uintptr {
	font, _, _ := procCreateFontW.Call(
		uintptr(int64(height)), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1, 0, 0, 5, 0, uintptr(unsafe.Pointer(wstr(face))),
	)
	return font
}

func createUIFont(height int32, weight int32) uintptr {
	return createFontFace(height, weight, "Segoe UI")
}

func initVisualResources() {
	if app.uiFont == 0 {
		app.uiFont = createUIFont(-16, FW_NORMAL)
		app.headingFont = createUIFont(-16, FW_SEMIBOLD)
		app.smallFont = createUIFont(-13, FW_NORMAL)
		app.brushWindow, _, _ = procCreateSolidBrush.Call(colorWindowBG)
		app.brushPanel, _, _ = procCreateSolidBrush.Call(colorPanelBG)
		app.brushStatus, _, _ = procCreateSolidBrush.Call(colorStatusBG)
		app.brushFooter, _, _ = procCreateSolidBrush.Call(colorFooterBG)
	}
}

func freeVisualResources() {
	for _, obj := range []uintptr{app.uiFont, app.headingFont, app.smallFont, app.brushWindow, app.brushPanel, app.brushStatus, app.brushFooter} {
		if obj != 0 {
			procDeleteObject.Call(obj)
		}
	}
	app.uiFont, app.headingFont, app.smallFont = 0, 0, 0
	app.brushWindow, app.brushPanel, app.brushStatus, app.brushFooter = 0, 0, 0, 0
}

func setControlFont(hwnd, font uintptr) {
	if hwnd != 0 && font != 0 {
		procSendMessageW.Call(hwnd, WM_SETFONT, font, 1)
	}
}

func setExplorerTheme(hwnd uintptr) {
	if hwnd != 0 {
		procSetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(wstr("Explorer"))), 0)
	}
}

func styleTitleBar(hwnd uintptr) {
	// Windows 11 supports caption/text colors through DWM. Older Windows simply
	// ignores these calls and retains the normal system title bar.
	caption := uint32(colorAccent)
	white := uint32(rgb(255, 255, 255))
	const DWMWA_CAPTION_COLOR = 35
	const DWMWA_TEXT_COLOR = 36
	procDwmSetWindowAttribute.Call(hwnd, DWMWA_CAPTION_COLOR, uintptr(unsafe.Pointer(&caption)), unsafe.Sizeof(caption))
	procDwmSetWindowAttribute.Call(hwnd, DWMWA_TEXT_COLOR, uintptr(unsafe.Pointer(&white)), unsafe.Sizeof(white))
}

func appendMenuString(menu uintptr, id uintptr, label string) bool {
	r, _, _ := procAppendMenuW.Call(menu, MF_STRING, id, uintptr(unsafe.Pointer(wstr(label))))
	return r != 0
}

func appendPopupMenu(menu, popup uintptr, label string) bool {
	r, _, _ := procAppendMenuW.Call(menu, MF_POPUP, popup, uintptr(unsafe.Pointer(wstr(label))))
	return r != 0
}

func createMainMenu(hwnd uintptr) uintptr {
	menu, _, _ := procCreateMenu.Call()
	fileMenu, _, _ := procCreatePopupMenu.Call()
	toolsMenu, _, _ := procCreatePopupMenu.Call()
	pdfViewMenu, _, _ := procCreatePopupMenu.Call()
	helpMenu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 || fileMenu == 0 || toolsMenu == 0 || pdfViewMenu == 0 || helpMenu == 0 {
		for _, handle := range []uintptr{fileMenu, toolsMenu, pdfViewMenu, helpMenu, menu} {
			if handle != 0 {
				procDestroyMenu.Call(handle)
			}
		}
		return 0
	}

	appendMenuString(fileMenu, ID_NEXT, "Ne&xt\tAlt+X")
	appendMenuString(fileMenu, ID_OPEN, "&Open\tAlt+O")
	appendMenuString(fileMenu, ID_OPEN_FOLDER, "Open &Folder\tAlt+F")
	procAppendMenuW.Call(fileMenu, MF_SEPARATOR, 0, 0)
	appendMenuString(fileMenu, ID_MENU_EXIT, "E&xit")

	appendMenuString(toolsMenu, ID_SOURCE_CONFIG, "&Sources…")
	appendMenuString(toolsMenu, ID_DEST_CONFIG, "Vault &Destinations…")
	appendMenuString(toolsMenu, ID_VAULT_LIBRARY, "Vault &Library…")
	appendMenuString(toolsMenu, ID_NAMING_TEMPLATES, "&Naming Templates…")
	widthFlags := uintptr(MF_STRING)
	pageFlags := uintptr(MF_STRING)
	if normalizePDFDefaultView(app.pdfDefaultView) == pdfDefaultViewFitWidth {
		widthFlags |= MF_CHECKED
	} else {
		pageFlags |= MF_CHECKED
	}
	procAppendMenuW.Call(pdfViewMenu, widthFlags, ID_PDF_DEFAULT_FIT_WIDTH, uintptr(unsafe.Pointer(wstr("Fit &Width (recommended)"))))
	procAppendMenuW.Call(pdfViewMenu, pageFlags, ID_PDF_DEFAULT_FIT_PAGE, uintptr(unsafe.Pointer(wstr("Fit &Page"))))
	appendPopupMenu(toolsMenu, pdfViewMenu, "PDF Default &View")
	procAppendMenuW.Call(toolsMenu, MF_SEPARATOR, 0, 0)
	appendMenuString(toolsMenu, ID_RECOVERY_CENTER, "&Recovery Center…")
	appendMenuString(toolsMenu, ID_LEARNING_CENTER, "&Learning Center…")
	appendMenuString(toolsMenu, ID_SUGGESTION_CENTER, "S&uggestion Center…")
	appendMenuString(toolsMenu, ID_FILEFLOW_HANDOFF, "&FileFlow Handoff…")

	appendMenuString(helpMenu, ID_MENU_ABOUT, "&About DCM - Quick Sort")

	if !appendPopupMenu(menu, fileMenu, "&File") || !appendPopupMenu(menu, toolsMenu, "&Tools") || !appendPopupMenu(menu, helpMenu, "&Help") {
		procDestroyMenu.Call(menu)
		return 0
	}
	if r, _, _ := procSetMenu.Call(hwnd, menu); r == 0 {
		procDestroyMenu.Call(menu)
		return 0
	}
	app.pdfDefaultViewMenu = pdfViewMenu
	procDrawMenuBar.Call(hwnd)
	return menu
}

func updatePDFDefaultViewMenuChecks() {
	if app.pdfDefaultViewMenu == 0 {
		return
	}
	widthFlags := uintptr(0)
	pageFlags := uintptr(0)
	if normalizePDFDefaultView(app.pdfDefaultView) == pdfDefaultViewFitWidth {
		widthFlags = MF_CHECKED
	} else {
		pageFlags = MF_CHECKED
	}
	procCheckMenuItem.Call(app.pdfDefaultViewMenu, ID_PDF_DEFAULT_FIT_WIDTH, widthFlags)
	procCheckMenuItem.Call(app.pdfDefaultViewMenu, ID_PDF_DEFAULT_FIT_PAGE, pageFlags)
	procDrawMenuBar.Call(app.hwnd)
}

func setPDFDefaultViewPreference(view string) {
	view = normalizePDFDefaultView(view)
	if err := savePDFDefaultViewSetting(view, app.sourceDir); err != nil {
		showError("Could not save the PDF default-view preference:\n" + err.Error())
		return
	}
	app.pdfDefaultView = view
	updatePDFDefaultViewMenuChecks()
	if pdfPreview.active {
		if view == pdfDefaultViewFitWidth {
			fitPreviewWidth()
		} else {
			fitPreview()
		}
		return
	}
	if view == pdfDefaultViewFitWidth {
		setText(app.statusLabel, "PDF default view: Fit Width · top anchored")
	} else {
		setText(app.statusLabel, "PDF default view: Fit Page")
	}
}

func showAboutDialog() {
	message := "DCM - Quick Sort\r\n" +
		"Downloaded Content Management\r\n" +
		"Version " + currentAppVersion + "\r\n\r\n" +
		"Preview, rename, move and organize downloaded files with durable recovery and Undo.\r\n\r\n" +
		"This is a free app. Consider Making Donation.\r\n" +
		"UPI: 8763320974@upi\r\n" +
		"Cyber Planet, Sector-18, Raurkela"
	procMessageBoxW.Call(app.hwnd, uintptr(unsafe.Pointer(wstr(message))), uintptr(unsafe.Pointer(wstr("About DCM - Quick Sort"))), 0x40)
}

func drawPrimaryButton(dis *DRAWITEMSTRUCT) {
	if dis == nil || dis.HDC == 0 {
		return
	}
	rc := dis.RcItem
	fill := colorAccent
	if dis.ItemState&ODS_SELECTED != 0 {
		fill = colorAccentHot
	}
	if dis.ItemState&ODS_DISABLED != 0 {
		fill = rgb(176, 186, 199)
	}
	brush, _, _ := procCreateSolidBrush.Call(fill)
	pen, _, _ := procCreatePen.Call(PS_SOLID, 1, fill)
	oldBrush, _, _ := procSelectObject.Call(dis.HDC, brush)
	oldPen, _, _ := procSelectObject.Call(dis.HDC, pen)
	procRoundRect.Call(dis.HDC, uintptr(rc.Left), uintptr(rc.Top), uintptr(rc.Right), uintptr(rc.Bottom), 10, 10)
	procSelectObject.Call(dis.HDC, oldBrush)
	procSelectObject.Call(dis.HDC, oldPen)
	procDeleteObject.Call(brush)
	procDeleteObject.Call(pen)
	procSetBkMode.Call(dis.HDC, TRANSPARENT)
	procSetTextColor.Call(dis.HDC, rgb(255, 255, 255))
	font := app.headingFont
	oldFont := uintptr(0)
	if font != 0 {
		oldFont, _, _ = procSelectObject.Call(dis.HDC, font)
	}
	labelText := strings.TrimSpace(getText(dis.HwndItem))
	if labelText == "" {
		labelText = "Move to &Vault"
	}
	label := wstr(labelText)
	const flags = 0x00000001 | 0x00000004 | 0x00000020 // DT_CENTER|DT_VCENTER|DT_SINGLELINE
	procDrawTextW.Call(dis.HDC, uintptr(unsafe.Pointer(label)), ^uintptr(0), uintptr(unsafe.Pointer(&rc)), flags)
	if oldFont != 0 {
		procSelectObject.Call(dis.HDC, oldFont)
	}
	if dis.ItemState&ODS_FOCUS != 0 {
		focus := RECT{rc.Left + 4, rc.Top + 4, rc.Right - 4, rc.Bottom - 4}
		procDrawFocusRect.Call(dis.HDC, uintptr(unsafe.Pointer(&focus)))
	}
}

func createControl(class, text string, style uint32, x, y, w, h int32, parent uintptr, id int) uintptr {
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(wstr(class))), uintptr(unsafe.Pointer(wstr(text))), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, uintptr(id), 0, 0)
	if hwnd != 0 {
		font := app.uiFont
		if font == 0 {
			font, _, _ = procGetStockObject.Call(DEFAULT_GUI_FONT)
		}
		procSendMessageW.Call(hwnd, WM_SETFONT, font, 1)
	}
	return hwnd
}

func setText(hwnd uintptr, s string) { procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(wstr(s)))) }
func getText(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func comboAddItem(hwnd uintptr, text string) {
	procSendMessageW.Call(hwnd, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(text))))
}

func populateCategoryFilter() {
	procSendMessageW.Call(app.categoryCombo, CB_RESETCONTENT, 0, 0)
	for _, category := range categoryOrder {
		comboAddItem(app.categoryCombo, string(category))
	}
	procSendMessageW.Call(app.categoryCombo, CB_SETCURSEL, 0, 0)
}

func selectedCategory() FileCategory {
	value := FileCategory(strings.TrimSpace(getText(app.categoryCombo)))
	for _, category := range categoryOrder {
		if value == category {
			return category
		}
	}
	return CategoryAll
}

func populateExtensionFilter(category FileCategory) {
	procSendMessageW.Call(app.extensionCombo, CB_RESETCONTENT, 0, 0)
	comboAddItem(app.extensionCombo, "All extensions")
	for _, ext := range extensionsForCategory(category) {
		comboAddItem(app.extensionCombo, ext)
	}
	procSendMessageW.Call(app.extensionCombo, CB_SETCURSEL, 0, 0)
}

func selectedExtension() string {
	value := strings.TrimSpace(getText(app.extensionCombo))
	if value == "" {
		return "All extensions"
	}
	return value
}

func populateInboxFilters() {
	if app.filterAgeCombo != 0 {
		procSendMessageW.Call(app.filterAgeCombo, CB_RESETCONTENT, 0, 0)
		for _, value := range inboxAgeOrder {
			comboAddItem(app.filterAgeCombo, string(value))
		}
		procSendMessageW.Call(app.filterAgeCombo, CB_SETCURSEL, 0, 0)
	}
	if app.filterSizeCombo != 0 {
		procSendMessageW.Call(app.filterSizeCombo, CB_RESETCONTENT, 0, 0)
		for _, value := range inboxSizeOrder {
			comboAddItem(app.filterSizeCombo, string(value))
		}
		procSendMessageW.Call(app.filterSizeCombo, CB_SETCURSEL, 0, 0)
	}
	if app.filterSortCombo != 0 {
		procSendMessageW.Call(app.filterSortCombo, CB_RESETCONTENT, 0, 0)
		for _, value := range inboxSortOrder {
			comboAddItem(app.filterSortCombo, string(value))
		}
		procSendMessageW.Call(app.filterSortCombo, CB_SETCURSEL, 2, 0)
	}
}

func selectedInboxAge() InboxAgeFilter {
	value := InboxAgeFilter(strings.TrimSpace(getText(app.filterAgeCombo)))
	for _, candidate := range inboxAgeOrder {
		if value == candidate {
			return candidate
		}
	}
	return InboxAgeAll
}

func selectedInboxSize() InboxSizeFilter {
	value := InboxSizeFilter(strings.TrimSpace(getText(app.filterSizeCombo)))
	for _, candidate := range inboxSizeOrder {
		if value == candidate {
			return candidate
		}
	}
	return InboxSizeAll
}

func selectedInboxSort() InboxSort {
	value := InboxSort(strings.TrimSpace(getText(app.filterSortCombo)))
	for _, candidate := range inboxSortOrder {
		if value == candidate {
			return candidate
		}
	}
	return InboxSortDateCreated
}

func selectedInboxFilter() InboxFilter {
	return InboxFilter{
		Category:     selectedCategory(),
		Extension:    selectedExtension(),
		NameContains: strings.TrimSpace(getText(app.filterNameEdit)),
		Age:          selectedInboxAge(),
		Size:         selectedInboxSize(),
		Sort:         selectedInboxSort(),
	}
}

func clearInboxFilters() {
	app.updatingInboxFilters = true
	setText(app.filterNameEdit, "")
	procSendMessageW.Call(app.categoryCombo, CB_SETCURSEL, 0, 0)
	populateExtensionFilter(CategoryAll)
	procSendMessageW.Call(app.filterAgeCombo, CB_SETCURSEL, 0, 0)
	procSendMessageW.Call(app.filterSizeCombo, CB_SETCURSEL, 0, 0)
	app.updatingInboxFilters = false
	reloadFilesWithFocus(false)
}

func populateRenameTools() {
	if app.renameToolCombo == 0 {
		return
	}
	procSendMessageW.Call(app.renameToolCombo, CB_RESETCONTENT, 0, 0)
	for _, tool := range renameToolOptions {
		comboAddItem(app.renameToolCombo, tool)
	}
	procSendMessageW.Call(app.renameToolCombo, CB_SETCURSEL, 0, 0)
}

func populateDestinationAliases() {
	if app.destAliasCombo == 0 {
		return
	}
	app.updatingDestinationAlias = true
	procSendMessageW.Call(app.destAliasCombo, CB_RESETCONTENT, 0, 0)
	comboAddItem(app.destAliasCombo, "Vault location")
	for _, alias := range app.destAliases {
		comboAddItem(app.destAliasCombo, alias.Name)
	}
	app.updatingDestinationAlias = false
	syncDestinationAliasSelection(strings.TrimSpace(getText(app.destEdit)))
}

func syncDestinationAliasSelection(path string) {
	if app.destAliasCombo == 0 || app.updatingDestinationAlias {
		return
	}
	idx := destinationAliasIndexForPath(app.destAliases, path)
	selection := uintptr(0)
	if idx >= 0 {
		selection = uintptr(idx + 1)
	}
	app.updatingDestinationAlias = true
	procSendMessageW.Call(app.destAliasCombo, CB_SETCURSEL, selection, 0)
	app.updatingDestinationAlias = false
}

func selectedDestinationAlias() (DestinationAlias, bool) {
	if app.destAliasCombo == 0 {
		return DestinationAlias{}, false
	}
	idx, _, _ := procSendMessageW.Call(app.destAliasCombo, CB_GETCURSEL, 0, 0)
	if idx == ^uintptr(0) || idx == 0 {
		return DestinationAlias{}, false
	}
	ai := int(idx) - 1
	if ai < 0 || ai >= len(app.destAliases) {
		return DestinationAlias{}, false
	}
	return app.destAliases[ai], true
}

func reloadDestinationAliases() {
	raw, err := loadDestinationAliases("")
	if err != nil {
		setText(app.statusLabel, "Could not load Vault destination favorites: "+err.Error())
		return
	}
	app.destAliases = normalizeVaultDestinationFavorites(loadVaultRootSetting(), raw)
	populateDestinationAliases()
	// Keep an external latest snapshot current without creating duplicate history entries.
	_, _ = backupCurrentConfiguration("configuration-loaded")
}

func destinationManagerFavoriteText(alias DestinationAlias) string {
	return alias.Name + "    →    " + alias.Path
}

func destinationManagerLocationText(entry VaultAddressEntry) string {
	depth := strings.Count(entry.Address, "/")
	name := entry.Address
	if pos := strings.LastIndex(name, "/"); pos >= 0 {
		name = name[pos+1:]
	}
	return strings.Repeat("    ", depth) + "▸ " + name
}

func populateDestinationManagerFavorites(selection int) {
	if destManager.favoritesList == 0 {
		return
	}
	procSendMessageW.Call(destManager.favoritesList, LB_RESETCONTENT, 0, 0)
	for _, alias := range app.destAliases {
		procSendMessageW.Call(destManager.favoritesList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(destinationManagerFavoriteText(alias)))))
	}
	destManager.selectedFavorite = -1
	if selection >= 0 && selection < len(app.destAliases) {
		destManager.selectedFavorite = selection
		procSendMessageW.Call(destManager.favoritesList, LB_SETCURSEL, uintptr(selection), 0)
	} else {
		procSendMessageW.Call(destManager.favoritesList, LB_SETCURSEL, ^uintptr(0), 0)
	}
	flag := uintptr(0)
	if destManager.selectedFavorite >= 0 {
		flag = 1
	}
	procEnableWindow.Call(destManager.deleteFavoriteBtn, flag)
}

func clearDestinationManagerLocation() {
	destManager.selectedLocation = -1
	setText(destManager.logicalEdit, "")
	setText(destManager.physicalEdit, "")
	procEnableWindow.Call(destManager.openLocationBtn, 0)
	procEnableWindow.Call(destManager.useLocationBtn, 0)
	procEnableWindow.Call(destManager.createChildBtn, 0)
}

func selectDestinationManagerVaultRoot() {
	root := loadVaultRootSetting()
	destManager.selectedLocation = -1
	setText(destManager.logicalEdit, "")
	setText(destManager.physicalEdit, root)
	flag := uintptr(0)
	if root != "" {
		flag = 1
	}
	procEnableWindow.Call(destManager.openLocationBtn, flag)
	procEnableWindow.Call(destManager.useLocationBtn, 0)
	procEnableWindow.Call(destManager.createChildBtn, flag)
	if destManager.locationList != 0 {
		procSendMessageW.Call(destManager.locationList, LB_SETCURSEL, ^uintptr(0), 0)
	}
}

func setDestinationManagerLocation(address string) bool {
	root := loadVaultRootSetting()
	physical, logical, err := resolveExistingVaultDestination(root, address)
	if err != nil {
		clearDestinationManagerLocation()
		if destManager.statusLabel != 0 {
			setText(destManager.statusLabel, err.Error())
		}
		return false
	}
	setText(destManager.logicalEdit, logical)
	setText(destManager.physicalEdit, physical)
	procEnableWindow.Call(destManager.openLocationBtn, 1)
	procEnableWindow.Call(destManager.useLocationBtn, 1)
	procEnableWindow.Call(destManager.createChildBtn, 1)
	destManager.selectedLocation = -1
	for i, entry := range destManager.locations {
		if strings.EqualFold(entry.Address, logical) {
			destManager.selectedLocation = i
			procSendMessageW.Call(destManager.locationList, LB_SETCURSEL, uintptr(i), 0)
			break
		}
	}
	setText(destManager.statusLabel, "Selected Vault location: "+logical)
	return true
}

func populateDestinationManagerLocations(preferred string) {
	if destManager.locationList == 0 {
		return
	}
	procSendMessageW.Call(destManager.locationList, LB_RESETCONTENT, 0, 0)
	destManager.locations = nil
	destManager.selectedLocation = -1
	root := loadVaultRootSetting()
	setText(destManager.vaultRootEdit, root)
	if root == "" {
		clearDestinationManagerLocation()
		setText(destManager.statusLabel, "Configure a Vault Library first. Every Quick Sort destination must live inside it.")
		return
	}
	idx, err := loadVaultDestinationIndex(root)
	if err != nil {
		clearDestinationManagerLocation()
		setText(destManager.statusLabel, "Could not read Vault index: "+err.Error())
		return
	}
	filter := strings.TrimSpace(getText(destManager.searchEdit))
	destManager.locations = vaultDestinationFolderSuggestions(idx, filter, 5000)
	for _, entry := range destManager.locations {
		procSendMessageW.Call(destManager.locationList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(destinationManagerLocationText(entry)))))
	}
	preferred = strings.TrimSpace(preferred)
	if preferred != "" && setDestinationManagerLocation(preferred) {
		return
	}
	selectDestinationManagerVaultRoot()
	if filter == "" {
		if len(destManager.locations) == 0 {
			setText(destManager.statusLabel, "Vault is empty · create the first top-level folder on the right.")
			return
		}
		setText(destManager.statusLabel, fmt.Sprintf("Vault ready: %d folder locations · select one or type to filter.", len(destManager.locations)))
	} else {
		setText(destManager.statusLabel, fmt.Sprintf("%d Vault location(s) match %q.", len(destManager.locations), filter))
	}
}

func refreshDestinationManagerVault(preferred string) {
	reloadDestinationAliases()
	populateDestinationManagerFavorites(-1)
	populateDestinationManagerLocations(preferred)
}

func loadDestinationManagerFavoriteSelection(index int) {
	if index < 0 || index >= len(app.destAliases) {
		destManager.selectedFavorite = -1
		procEnableWindow.Call(destManager.deleteFavoriteBtn, 0)
		return
	}
	destManager.selectedFavorite = index
	alias := app.destAliases[index]
	setText(destManager.favoriteNameEdit, alias.Name)
	procEnableWindow.Call(destManager.deleteFavoriteBtn, 1)
	if setDestinationManagerLocation(alias.Path) {
		setText(destManager.statusLabel, "Favorite: "+alias.Name+" → "+alias.Path)
	}
}

func saveDestinationManagerFavorite() {
	name := strings.TrimSpace(getText(destManager.favoriteNameEdit))
	logical := strings.TrimSpace(getText(destManager.logicalEdit))
	if _, logical, err := resolveExistingVaultDestination(loadVaultRootSetting(), logical); err != nil {
		setText(destManager.statusLabel, "Select an existing Vault location before saving a favorite.")
		return
	} else {
		alias := DestinationAlias{Name: name, Path: logical}
		if err := validateDestinationAlias(alias); err != nil {
			setText(destManager.statusLabel, err.Error())
			return
		}
		raw, readErr := loadDestinationAliases("")
		if readErr != nil {
			setText(destManager.statusLabel, "Could not read destination favorites: "+readErr.Error())
			return
		}
		raw, _, err = upsertDestinationAlias(raw, alias)
		if err != nil {
			setText(destManager.statusLabel, err.Error())
			return
		}
		if err := writeDestinationAliases(raw); err != nil {
			setText(destManager.statusLabel, "Could not save favorite: "+err.Error())
			return
		}
		reloadDestinationAliases()
		favoriteIndex := -1
		for i, item := range app.destAliases {
			if strings.EqualFold(item.Name, alias.Name) {
				favoriteIndex = i
				break
			}
		}
		populateDestinationManagerFavorites(favoriteIndex)
		backupNote := ""
		if _, backupErr := backupCurrentConfiguration("vault-destination-favorite-saved"); backupErr != nil {
			backupNote = " · backup warning: " + backupErr.Error()
		}
		setText(destManager.statusLabel, "Saved favorite "+alias.Name+" → "+alias.Path+backupNote)
		setText(app.statusLabel, "Vault destination favorite saved: "+alias.Name+" → "+alias.Path+backupNote)
	}
}

func deleteDestinationManagerFavorite() {
	idx := destManager.selectedFavorite
	if idx < 0 || idx >= len(app.destAliases) {
		setText(destManager.statusLabel, "Select a favorite first.")
		return
	}
	alias := app.destAliases[idx]
	result, _, _ := procMessageBoxW.Call(destManager.hwnd,
		uintptr(unsafe.Pointer(wstr("Remove favorite '"+alias.Name+"'?\n\nThe Vault folder and its files will not be changed."))),
		uintptr(unsafe.Pointer(wstr("DCM - Quick Sort"))), 0x04|0x20)
	if result != 6 {
		return
	}
	raw, err := loadDestinationAliases("")
	if err != nil {
		setText(destManager.statusLabel, "Could not read favorites: "+err.Error())
		return
	}
	filtered := make([]DestinationAlias, 0, len(raw))
	for _, item := range raw {
		if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(alias.Name)) {
			continue
		}
		filtered = append(filtered, item)
	}
	if err := writeDestinationAliases(filtered); err != nil {
		setText(destManager.statusLabel, "Could not remove favorite: "+err.Error())
		return
	}
	reloadDestinationAliases()
	populateDestinationManagerFavorites(-1)
	setText(destManager.favoriteNameEdit, "")
	_, _ = backupCurrentConfiguration("vault-destination-favorite-removed")
	setText(destManager.statusLabel, "Removed favorite '"+alias.Name+"'. The Vault location was not changed.")
}

func chooseDestinationManagerVault() {
	initial := loadVaultRootSetting()
	if initial == "" {
		initial = app.sourceDir
	}
	root := browseFolder(destManager.hwnd, "Choose Flowverse Vault / Library", initial)
	if root == "" {
		return
	}
	if err := saveVaultRootSetting(root, app.sourceDir); err != nil {
		setText(destManager.statusLabel, "Could not save Vault Library: "+err.Error())
		return
	}
	setText(destManager.statusLabel, "Rebuilding Vault address index…")
	if err := rebuildVaultAddressIndex(root); err != nil {
		setText(destManager.statusLabel, "Vault saved, but rebuild failed: "+err.Error())
		return
	}
	current := strings.TrimSpace(getText(app.destEdit))
	if _, _, err := resolveExistingVaultDestination(root, current); err != nil {
		setDestinationText("")
	}
	setText(destManager.searchEdit, "")
	refreshDestinationManagerVault(current)
	setText(app.statusLabel, "Vault Library configured: "+root)
}

func rebuildDestinationManagerIndex() {
	root := loadVaultRootSetting()
	if root == "" {
		setText(destManager.statusLabel, "Configure a Vault Library first.")
		return
	}
	preferred := strings.TrimSpace(getText(destManager.logicalEdit))
	setText(destManager.statusLabel, "Rebuilding Vault address index…")
	if err := rebuildVaultAddressIndex(root); err != nil {
		setText(destManager.statusLabel, "Could not rebuild Vault index: "+err.Error())
		return
	}
	populateDestinationManagerLocations(preferred)
	setText(destManager.statusLabel, fmt.Sprintf("Vault index rebuilt: %d folder locations.", len(destManager.locations)))
}

func createDestinationManagerChild() {
	parent := strings.TrimSpace(getText(destManager.logicalEdit))
	child := strings.TrimSpace(getText(destManager.childNameEdit))
	_, logical, err := createVaultChildFolder(loadVaultRootSetting(), parent, child)
	if err != nil {
		setText(destManager.statusLabel, "Could not create Vault folder: "+err.Error())
		return
	}
	setText(destManager.childNameEdit, "")
	setText(destManager.searchEdit, "")
	populateDestinationManagerLocations(logical)
	setText(destManager.statusLabel, "Created Vault location: "+logical)
}

func openDestinationManagerLocation() {
	logicalText := strings.TrimSpace(getText(destManager.logicalEdit))
	if logicalText == "" {
		root := loadVaultRootSetting()
		if root == "" {
			setText(destManager.statusLabel, "Configure a Vault Library first.")
			return
		}
		shellOpen(root)
		setText(destManager.statusLabel, "Opened Vault root.")
		return
	}
	physical, logical, err := resolveExistingVaultDestination(loadVaultRootSetting(), logicalText)
	if err != nil {
		setText(destManager.statusLabel, err.Error())
		return
	}
	shellOpen(physical)
	setText(destManager.statusLabel, "Opened Vault location: "+logical)
}

func useDestinationManagerLocation() {
	_, logical, err := resolveExistingVaultDestination(loadVaultRootSetting(), getText(destManager.logicalEdit))
	if err != nil {
		setText(destManager.statusLabel, err.Error())
		return
	}
	setDestinationText(logical)
	setText(app.statusLabel, "Vault destination: "+logical)
	procPostMessageW.Call(destManager.hwnd, WM_CLOSE, 0, 0)
}

func exportDestinationManagerConfiguration() {
	result, err := exportCurrentConfiguration()
	if err != nil {
		setText(destManager.statusLabel, "Could not export configuration: "+err.Error())
		return
	}
	_, _ = backupCurrentConfiguration("manual-export")
	setText(destManager.statusLabel, "Exported configuration: "+result.Path)
	setText(app.statusLabel, "Configuration exported: "+result.Path)
	if dir := filepath.Dir(result.Path); dir != "" {
		shellOpen(dir)
	}
}

func configurationImportSummary(pkg configurationImportPackage) string {
	parts := make([]string, 0, 3)
	if pkg.Settings != nil {
		parts = append(parts, "Default source: "+pkg.Settings.DefaultSource)
	}
	if pkg.HasDestinations {
		parts = append(parts, fmt.Sprintf("Destination aliases: %d", len(pkg.Destinations)))
	}
	if pkg.HasSources {
		parts = append(parts, fmt.Sprintf("Source aliases: %d", len(pkg.Sources)))
	}
	if len(pkg.Learning) > 0 {
		parts = append(parts, "Local learning model: included")
	}
	if pkg.BackupInfo != nil && strings.TrimSpace(pkg.BackupInfo.CreatedAt) != "" {
		parts = append(parts, "Backup created: "+pkg.BackupInfo.CreatedAt)
	}
	return strings.Join(parts, "\n")
}

func chooseDestinationImportMode(pkg configurationImportPackage) (aliasImportMode, bool) {
	if !pkg.HasDestinations && !pkg.HasSources {
		return aliasMergeKeepExisting, true
	}
	reloadDestinationAliases()
	reloadSourceAliases()
	total := len(pkg.Destinations) + len(pkg.Sources)
	result, _, _ := procMessageBoxW.Call(destManager.hwnd,
		uintptr(unsafe.Pointer(wstr(fmt.Sprintf("Import %d saved source/destination aliases?\n\nYes = Merge with current aliases\nNo = Replace current aliases\nCancel = Stop import", total)))),
		uintptr(unsafe.Pointer(wstr("Import Configuration — DCM - Quick Sort"))), 0x03|0x20) // MB_YESNOCANCEL | MB_ICONQUESTION
	if result == 2 { // IDCANCEL
		return aliasMergeKeepExisting, false
	}
	if result == 7 { // IDNO
		return aliasReplaceAll, true
	}
	if result != 6 { // IDYES
		return aliasMergeKeepExisting, false
	}
	conflicts := countAliasConflicts(app.destAliases, pkg.Destinations) + countSourceAliasConflicts(app.sourceAliases, pkg.Sources)
	if conflicts == 0 {
		return aliasMergeKeepExisting, true
	}
	result, _, _ = procMessageBoxW.Call(destManager.hwnd,
		uintptr(unsafe.Pointer(wstr(fmt.Sprintf("%d imported alias name(s) already exist.\n\nYes = Use imported destination for conflicts\nNo = Keep current destination for conflicts\nCancel = Stop import", conflicts)))),
		uintptr(unsafe.Pointer(wstr("Resolve Destination Conflicts"))), 0x03|0x20)
	if result == 6 {
		return aliasMergeUseImported, true
	}
	if result == 7 {
		return aliasMergeKeepExisting, true
	}
	return aliasMergeKeepExisting, false
}

func applyImportedConfigurationToUI(pkg configurationImportPackage, result configurationImportResult) {
	if pkg.HasDestinations {
		reloadDestinationAliases()
		if destManager.hwnd != 0 {
			populateDestinationManagerFavorites(-1)
			populateDestinationManagerLocations(strings.TrimSpace(getText(app.destEdit)))
		}
	}
	if pkg.HasSources {
		reloadSourceAliases()
		if sourceManager.hwnd != 0 {
			populateSourceManagerList(-1)
			clearSourceManagerForm()
		}
	}
	if pkg.Settings != nil {
		source := strings.TrimSpace(pkg.Settings.DefaultSource)
		if st, err := os.Stat(source); err == nil && st.IsDir() {
			app.sourceDir = source
			setSourceText(source)
			reloadFiles()
		}
	}
	message := "Configuration imported"
	partsImported := make([]string, 0, 5)
	if result.SettingsImported {
		partsImported = append(partsImported, "settings")
	}
	if result.SourcesImported > 0 {
		partsImported = append(partsImported, fmt.Sprintf("%d source aliases", result.SourcesImported))
	}
	if result.DestinationsImported > 0 {
		partsImported = append(partsImported, fmt.Sprintf("%d destination aliases", result.DestinationsImported))
	}
	if result.LearningImported {
		partsImported = append(partsImported, "learning")
	}
	if result.NamingTemplatesImported {
		partsImported = append(partsImported, "naming templates")
		if namingTemplateManager.hwnd != 0 {
			cfg, err := loadNamingTemplateConfig(namingTemplatesFilePath())
			if err == nil {
				namingTemplateManager.cfg = cfg
				populateNamingTemplateList(-1)
			}
		}
	}
	if len(partsImported) > 0 {
		message += ": " + strings.Join(partsImported, " + ")
	}
	if result.AliasConflicts > 0 {
		message += fmt.Sprintf(" · %d conflict(s) resolved", result.AliasConflicts)
	}
	if result.Warning != "" {
		message += " · " + result.Warning
	}
	setText(destManager.statusLabel, message)
	setText(app.statusLabel, message)
	refreshRestoreLatestButton()
}

func importDestinationManagerConfiguration() {
	path := browseConfigurationFile(destManager.hwnd)
	if path == "" {
		return
	}
	pkg, err := loadConfigurationImport(path)
	if err != nil {
		setText(destManager.statusLabel, "Could not import configuration: "+err.Error())
		return
	}
	mode, ok := chooseDestinationImportMode(pkg)
	if !ok {
		setText(destManager.statusLabel, "Configuration import cancelled.")
		return
	}
	summary := configurationImportSummary(pkg)
	confirm, _, _ := procMessageBoxW.Call(destManager.hwnd,
		uintptr(unsafe.Pointer(wstr("Import this configuration?\n\n"+summary+"\n\nDCM will back up the current configuration before making changes."))),
		uintptr(unsafe.Pointer(wstr("Import Configuration — DCM - Quick Sort"))), 0x01|0x20) // MB_OKCANCEL | MB_ICONQUESTION
	if confirm != 1 { // IDOK
		setText(destManager.statusLabel, "Configuration import cancelled.")
		return
	}
	result, err := applyConfigurationImport(settingsDirectory(), configurationBackupRoot(), currentAppVersion, pkg, mode, time.Now())
	if err != nil {
		setText(destManager.statusLabel, "Configuration import stopped: "+err.Error())
		return
	}
	applyImportedConfigurationToUI(pkg, result)
}

func restoreDestinationManagerLatestConfiguration() {
	root := configurationBackupRoot()
	pkg, err := loadConfigurationSnapshotDirectory(filepath.Join(root, "latest"), configurationImportLatest)
	if err != nil {
		setText(destManager.statusLabel, "No valid latest backup is available: "+err.Error())
		refreshRestoreLatestButton()
		return
	}
	summary := configurationImportSummary(pkg)
	confirm, _, _ := procMessageBoxW.Call(destManager.hwnd,
		uintptr(unsafe.Pointer(wstr("Restore the latest DCM configuration backup?\n\n"+summary+"\n\nYour current configuration will be backed up first."))),
		uintptr(unsafe.Pointer(wstr("Restore Latest Backup — DCM - Quick Sort"))), 0x01|0x20)
	if confirm != 1 {
		setText(destManager.statusLabel, "Restore cancelled.")
		return
	}
	result, err := restoreLatestConfiguration(settingsDirectory(), root, currentAppVersion, time.Now())
	if err != nil {
		setText(destManager.statusLabel, "Restore stopped: "+err.Error())
		return
	}
	applyImportedConfigurationToUI(pkg, result)
	setText(destManager.statusLabel, "Latest configuration backup restored successfully.")
	setText(app.statusLabel, "Latest configuration backup restored successfully.")
}

func refreshRestoreLatestButton() {
	if destManager.restoreLatestBtn == 0 {
		return
	}
	_, ok := latestConfigurationBackupInfo(configurationBackupRoot())
	flag := uintptr(0)
	if ok {
		flag = 1
	}
	procEnableWindow.Call(destManager.restoreLatestBtn, flag)
}

func openConfigurationBackupFolder() {
	root := configurationBackupRoot()
	if root == "" {
		setText(destManager.statusLabel, "Backup folder is unavailable.")
		return
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		setText(destManager.statusLabel, "Could not open backup folder: "+err.Error())
		return
	}
	shellOpen(root)
	setText(destManager.statusLabel, "Backup folder: "+root)
}

func destinationManagerFocusOrder() []uintptr {
	return []uintptr{
		destManager.searchEdit, destManager.locationList,
		destManager.childNameEdit, destManager.createChildBtn,
		destManager.favoriteNameEdit, destManager.saveFavoriteBtn, destManager.favoritesList, destManager.deleteFavoriteBtn,
		destManager.changeVaultBtn, destManager.rebuildIndexBtn, destManager.openLocationBtn, destManager.useLocationBtn,
		destManager.exportConfigBtn, destManager.importConfigBtn, destManager.restoreLatestBtn, destManager.backupFolderBtn, destManager.closeBtn,
	}
}

func focusWithinDestinationManager(focus uintptr) bool {
	if destManager.hwnd == 0 || focus == 0 {
		return false
	}
	for _, hwnd := range destinationManagerFocusOrder() {
		if focus == hwnd {
			return true
		}
	}
	return false
}

func moveDestinationManagerFocus(reverse bool) bool {
	order := destinationManagerFocusOrder()
	if len(order) == 0 {
		return false
	}
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

func createDestinationManagerUI(hwnd uintptr) {
	destManager.selectedLocation = -1
	destManager.selectedFavorite = -1
	rootLabel := createControl("STATIC", "Vault:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 22, 80, 22, hwnd, 0)
	destManager.vaultRootEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL|ES_READONLY, 110, 17, 650, 28, hwnd, ID_DM_VAULT_ROOT)
	destManager.changeVaultBtn = createControl("BUTTON", "&Change Vault…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 775, 16, 125, 30, hwnd, ID_DM_CHANGE_VAULT)

	searchLabel := createControl("STATIC", "Find location:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 62, 85, 22, hwnd, 0)
	destManager.searchEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 110, 57, 650, 28, hwnd, ID_DM_SEARCH)
	destManager.rebuildIndexBtn = createControl("BUTTON", "&Rebuild Index", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 775, 56, 125, 30, hwnd, ID_DM_REBUILD_INDEX)

	locationsLabel := createControl("STATIC", "Vault Locations", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 102, 180, 22, hwnd, 0)
	destManager.locationList = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|WS_VSCROLL, 20, 126, 500, 350, hwnd, ID_DM_LOCATION_LIST)

	selectedLabel := createControl("STATIC", "Vault Location:", WS_CHILD|WS_VISIBLE|SS_LEFT, 540, 102, 160, 22, hwnd, 0)
	destManager.logicalEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL|ES_READONLY, 540, 126, 360, 28, hwnd, ID_DM_LOGICAL)
	physicalLabel := createControl("STATIC", "Physical location (derived):", WS_CHILD|WS_VISIBLE|SS_LEFT, 540, 164, 210, 22, hwnd, 0)
	destManager.physicalEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL|ES_READONLY, 540, 188, 360, 28, hwnd, ID_DM_PHYSICAL)

	childLabel := createControl("STATIC", "New child folder:", WS_CHILD|WS_VISIBLE|SS_LEFT, 540, 230, 170, 22, hwnd, 0)
	destManager.childNameEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 540, 254, 225, 28, hwnd, ID_DM_CHILD_NAME)
	destManager.createChildBtn = createControl("BUTTON", "&Create Child", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 775, 253, 125, 30, hwnd, ID_DM_CREATE_CHILD)

	favoriteLabel := createControl("STATIC", "Favorite name:", WS_CHILD|WS_VISIBLE|SS_LEFT, 540, 294, 150, 22, hwnd, 0)
	destManager.favoriteNameEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 540, 318, 225, 28, hwnd, ID_DM_FAVORITE_NAME)
	destManager.saveFavoriteBtn = createControl("BUTTON", "&Save Favorite", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 775, 317, 125, 30, hwnd, ID_DM_SAVE_FAVORITE)
	favoritesLabel := createControl("STATIC", "Favorites", WS_CHILD|WS_VISIBLE|SS_LEFT, 540, 358, 120, 22, hwnd, 0)
	destManager.favoritesList = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|WS_VSCROLL, 540, 382, 360, 94, hwnd, ID_DM_FAVORITES_LIST)
	destManager.deleteFavoriteBtn = createControl("BUTTON", "&Remove Favorite", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 775, 484, 125, 30, hwnd, ID_DM_DELETE_FAVORITE)

	destManager.openLocationBtn = createControl("BUTTON", "&Open Location", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 20, 496, 135, 32, hwnd, ID_DM_OPEN_LOCATION)
	destManager.useLocationBtn = createControl("BUTTON", "&Use Location", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 165, 496, 135, 32, hwnd, ID_DM_USE_LOCATION)
	destManager.closeBtn = createControl("BUTTON", "Close", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 385, 496, 135, 32, hwnd, ID_DM_CLOSE)

	backupLabel := createControl("STATIC", "Configuration", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 548, 150, 22, hwnd, 0)
	destManager.exportConfigBtn = createControl("BUTTON", "&Export", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 20, 574, 150, 31, hwnd, ID_DM_EXPORT_CONFIG)
	destManager.importConfigBtn = createControl("BUTTON", "&Import…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 180, 574, 150, 31, hwnd, ID_DM_IMPORT_CONFIG)
	destManager.restoreLatestBtn = createControl("BUTTON", "&Restore Latest", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 340, 574, 160, 31, hwnd, ID_DM_RESTORE_LATEST)
	destManager.backupFolderBtn = createControl("BUTTON", "Open &Backup Folder", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 510, 574, 175, 31, hwnd, ID_DM_BACKUP_FOLDER)
	destManager.statusLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 620, 880, 50, hwnd, 0)

	for _, c := range []uintptr{rootLabel, searchLabel, locationsLabel, selectedLabel, physicalLabel, childLabel, favoriteLabel, favoritesLabel, backupLabel,
		destManager.vaultRootEdit, destManager.changeVaultBtn, destManager.searchEdit, destManager.rebuildIndexBtn, destManager.locationList,
		destManager.logicalEdit, destManager.physicalEdit, destManager.childNameEdit, destManager.createChildBtn,
		destManager.favoriteNameEdit, destManager.saveFavoriteBtn, destManager.favoritesList, destManager.deleteFavoriteBtn,
		destManager.openLocationBtn, destManager.useLocationBtn, destManager.closeBtn,
		destManager.exportConfigBtn, destManager.importConfigBtn, destManager.restoreLatestBtn, destManager.backupFolderBtn, destManager.statusLabel} {
		setControlFont(c, app.uiFont)
	}
	for _, c := range destinationManagerFocusOrder() {
		setExplorerTheme(c)
	}
	for _, c := range []uintptr{destManager.vaultRootEdit, destManager.logicalEdit, destManager.physicalEdit} {
		setExplorerTheme(c)
	}
	procEnableWindow.Call(destManager.deleteFavoriteBtn, 0)
	procEnableWindow.Call(destManager.openLocationBtn, 0)
	procEnableWindow.Call(destManager.useLocationBtn, 0)
	procEnableWindow.Call(destManager.createChildBtn, 0)
	refreshRestoreLatestButton()
	preferred := strings.TrimSpace(getText(app.destEdit))
	refreshDestinationManagerVault(preferred)
	procSetFocus.Call(destManager.searchEdit)
}

func destinationManagerWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		destManager.hwnd = hwnd
		createDestinationManagerUI(hwnd)
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
			case ID_DM_CHANGE_VAULT:
				chooseDestinationManagerVault()
			case ID_DM_CREATE_CHILD:
				createDestinationManagerChild()
			case ID_DM_SAVE_FAVORITE:
				saveDestinationManagerFavorite()
			case ID_DM_DELETE_FAVORITE:
				deleteDestinationManagerFavorite()
			case ID_DM_OPEN_LOCATION:
				openDestinationManagerLocation()
			case ID_DM_REBUILD_INDEX:
				rebuildDestinationManagerIndex()
			case ID_DM_USE_LOCATION:
				useDestinationManagerLocation()
			case ID_DM_EXPORT_CONFIG:
				exportDestinationManagerConfiguration()
			case ID_DM_IMPORT_CONFIG:
				importDestinationManagerConfiguration()
			case ID_DM_RESTORE_LATEST:
				restoreDestinationManagerLatestConfiguration()
			case ID_DM_BACKUP_FOLDER:
				openConfigurationBackupFolder()
			case ID_DM_CLOSE:
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			}
		}
		if notify == EN_CHANGE && id == ID_DM_SEARCH {
			populateDestinationManagerLocations(strings.TrimSpace(getText(destManager.logicalEdit)))
		}
		if notify == LBN_SELCHANGE {
			switch id {
			case ID_DM_LOCATION_LIST:
				if idx, _, _ := procSendMessageW.Call(destManager.locationList, LB_GETCURSEL, 0, 0); idx != LB_ERR && int(idx) < len(destManager.locations) {
					setDestinationManagerLocation(destManager.locations[int(idx)].Address)
				}
			case ID_DM_FAVORITES_LIST:
				if idx, _, _ := procSendMessageW.Call(destManager.favoritesList, LB_GETCURSEL, 0, 0); idx != LB_ERR {
					loadDestinationManagerFavoriteSelection(int(idx))
				}
			}
		}
		return 0
	case WM_CLOSE:
		procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return 0
	case WM_DESTROY:
		destManager = destinationManagerState{selectedLocation: -1, selectedFavorite: -1}
		reloadDestinationAliases()
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func openDestinationManager() {
	if destManager.hwnd != 0 {
		procShowWindow.Call(destManager.hwnd, SW_SHOW)
		procUpdateWindow.Call(destManager.hwnd)
		procSetForegroundWindow.Call(destManager.hwnd)
		refreshDestinationManagerVault(strings.TrimSpace(getText(app.destEdit)))
		procSetFocus.Call(destManager.searchEdit)
		return
	}
	reloadDestinationAliases()
	hInst, _, _ := procGetModuleHandleW.Call(0)
	style := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE | WS_CLIPCHILDREN)
	hwnd, _, _ := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(wstr("QuickSortDestinations"))),
		uintptr(unsafe.Pointer(wstr("Vault Destinations — DCM - Quick Sort"))),
		style, CW_USEDEFAULT, CW_USEDEFAULT, 940, 720, app.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		showError("Could not open Vault Destinations.")
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
	procSetForegroundWindow.Call(hwnd)
	procSetFocus.Call(destManager.searchEdit)
}

func readTextEnabled() bool {
	if app.readTextCheck == 0 {
		return false
	}
	state, _, _ := procSendMessageW.Call(app.readTextCheck, BM_GETCHECK, 0, 0)
	return state == BST_CHECKED
}

func updateReadTextAvailability(path string) {
	if app.readTextCheck == 0 {
		return
	}
	enabled := canReadAsText(path)
	if !enabled {
		procSendMessageW.Call(app.readTextCheck, BM_SETCHECK, 0, 0)
	}
	flag := uintptr(0)
	if enabled {
		flag = 1
	}
	procEnableWindow.Call(app.readTextCheck, flag)
}

func hideTextPreview() {
	if app.textPreviewEdit != 0 {
		procShowWindow.Call(app.textPreviewEdit, 0)
		setText(app.textPreviewEdit, "")
		if app.textPreviewFont != 0 {
			setControlFont(app.textPreviewEdit, app.uiFont)
			procDeleteObject.Call(app.textPreviewFont)
			app.textPreviewFont = 0
		}
	}
	app.textPreviewActive = false
	app.textPreviewKind = ""
}

func showTextPreview(path string) bool {
	text, err := readTextPreview(path, defaultTextPreviewLimit)
	if err != nil {
		app.fallback = "Could not read this file as text.\n\n" + err.Error()
		setText(app.statusLabel, "Text preview unavailable: "+err.Error())
		updatePreviewZoomControls()
		procInvalidateRect.Call(app.previewHwnd, 0, 1)
		return false
	}
	if text == "" {
		text = "[Empty file]"
	}
	return showTextPreviewContentKind(text, "Text")
}

func showTextPreviewContent(text string) bool {
	return showTextPreviewContentKind(text, "Document")
}

func showTextPreviewContentKind(text, kind string) bool {
	if app.textPreviewEdit == 0 {
		return false
	}
	app.textPreviewActive = true
	app.textPreviewKind = strings.TrimSpace(kind)
	if app.textPreviewKind == "" {
		app.textPreviewKind = "Document"
	}
	setText(app.textPreviewEdit, text)
	applyTextPreviewZoom()
	resizeTextPreview()
	procShowWindow.Call(app.textPreviewEdit, SW_SHOW)
	app.fallback = ""
	updatePreviewZoomControls()
	return true
}

func applyTextPreviewZoom() {
	if app.textPreviewEdit == 0 || !app.textPreviewActive {
		return
	}
	if app.textPreviewFont != 0 {
		setControlFont(app.textPreviewEdit, app.uiFont)
		procDeleteObject.Call(app.textPreviewFont)
		app.textPreviewFont = 0
	}
	px := textPreviewFontPixelHeight(app.previewZoom)
	app.textPreviewFont = createUIFont(-int32(px), FW_NORMAL)
	setControlFont(app.textPreviewEdit, app.textPreviewFont)
}

func setTextPreviewZoom(percent int) {
	app.previewFit = false
	app.previewFitWidth = false
	app.previewZoom = clampPreviewZoom(percent)
	applyTextPreviewZoom()
	kind := app.textPreviewKind
	if kind == "" {
		kind = "Document"
	}
	setText(app.statusLabel, fmt.Sprintf("%s preview zoom: %d%%", kind, app.previewZoom))
}

func resizeTextPreview() {
	if app.textPreviewEdit == 0 || app.previewHwnd == 0 {
		return
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	w := rc.Right - rc.Left
	h := rc.Bottom - rc.Top
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	procMoveWindow.Call(app.textPreviewEdit, 0, 0, uintptr(w), uintptr(h), 1)
}

func hideXLSXPreview() {
	if app.xlsxSheetCombo != 0 {
		procShowWindow.Call(app.xlsxSheetCombo, 0)
		procSendMessageW.Call(app.xlsxSheetCombo, CB_RESETCONTENT, 0, 0)
	}
	if app.xlsxGridEdit != 0 {
		procShowWindow.Call(app.xlsxGridEdit, 0)
		setText(app.xlsxGridEdit, "")
		if app.xlsxGridFont != 0 {
			setControlFont(app.xlsxGridEdit, app.uiFont)
			procDeleteObject.Call(app.xlsxGridFont)
			app.xlsxGridFont = 0
		}
	}
	app.xlsxPreviewActive = false
	app.xlsxWorkbook = XLSXWorkbookPreview{}
	app.xlsxSheetIndex = -1
}

func applyXLSXPreviewZoom() {
	if app.xlsxGridEdit == 0 || !app.xlsxPreviewActive {
		return
	}
	if app.xlsxGridFont != 0 {
		setControlFont(app.xlsxGridEdit, app.uiFont)
		procDeleteObject.Call(app.xlsxGridFont)
		app.xlsxGridFont = 0
	}
	px := textPreviewFontPixelHeight(app.previewZoom)
	app.xlsxGridFont = createFontFace(-int32(px), FW_NORMAL, "Consolas")
	setControlFont(app.xlsxGridEdit, app.xlsxGridFont)
}

func setXLSXPreviewZoom(percent int) {
	app.previewFit = false
	app.previewFitWidth = false
	app.previewZoom = clampPreviewZoom(percent)
	applyXLSXPreviewZoom()
	setText(app.statusLabel, fmt.Sprintf("XLSX preview zoom: %d%%", app.previewZoom))
}

func resizeXLSXPreview() {
	if app.xlsxSheetCombo == 0 || app.xlsxGridEdit == 0 || app.previewHwnd == 0 || !app.xlsxPreviewActive {
		return
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	w := rc.Right - rc.Left
	h := rc.Bottom - rc.Top
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	const sheetBarH int32 = 34
	procMoveWindow.Call(app.xlsxSheetCombo, 6, 4, uintptr(w-12), 280, 1)
	gridH := h - sheetBarH - 4
	if gridH < 1 {
		gridH = 1
	}
	procMoveWindow.Call(app.xlsxGridEdit, 0, uintptr(sheetBarH), uintptr(w), uintptr(gridH), 1)
}

func showXLSXSheet(index int) {
	if !app.xlsxPreviewActive || index < 0 || index >= len(app.xlsxWorkbook.Sheets) {
		return
	}
	app.xlsxSheetIndex = index
	procSendMessageW.Call(app.xlsxSheetCombo, CB_SETCURSEL, uintptr(index), 0)
	sheet := app.xlsxWorkbook.Sheets[index]
	setText(app.xlsxGridEdit, renderXLSXSheetGrid(sheet))
	applyXLSXPreviewZoom()
	setText(app.statusLabel, fmt.Sprintf("DCM XLSX Preview: %s · %d rows × %d columns", sheet.Name, sheet.MaxRow, sheet.MaxCol))
}

func showXLSXPreview(wb XLSXWorkbookPreview) bool {
	if app.xlsxSheetCombo == 0 || app.xlsxGridEdit == 0 || len(wb.Sheets) == 0 {
		return false
	}
	hideTextPreview()
	app.xlsxPreviewActive = true
	app.xlsxWorkbook = wb
	app.xlsxSheetIndex = 0
	procSendMessageW.Call(app.xlsxSheetCombo, CB_RESETCONTENT, 0, 0)
	for _, sheet := range wb.Sheets {
		procSendMessageW.Call(app.xlsxSheetCombo, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(sheet.Name))))
	}
	procShowWindow.Call(app.xlsxSheetCombo, SW_SHOW)
	procShowWindow.Call(app.xlsxGridEdit, SW_SHOW)
	resizeXLSXPreview()
	showXLSXSheet(0)
	app.fallback = ""
	updatePreviewZoomControls()
	return true
}

func releasePPTXImages() {
	for _, item := range app.pptxImages {
		if item.image != 0 {
			procGdipDisposeImage.Call(item.image)
		}
		if item.stream != nil {
			releaseUnknown(item.stream)
		}
	}
	app.pptxImages = nil
}

func hidePPTXPreview() {
	if app.pptxSlideCombo != 0 {
		procShowWindow.Call(app.pptxSlideCombo, 0)
		procSendMessageW.Call(app.pptxSlideCombo, CB_RESETCONTENT, 0, 0)
	}
	releasePPTXImages()
	app.pptxPreviewActive = false
	app.pptxDeck = PPTXPresentationPreview{}
	app.pptxSlideIndex = -1
}

func resizePPTXPreview() {
	if app.pptxSlideCombo == 0 || app.previewHwnd == 0 || !app.pptxPreviewActive {
		return
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	w := rc.Right - rc.Left
	if w < 1 {
		w = 1
	}
	comboW := w - 12
	if comboW < 1 {
		comboW = 1
	}
	procMoveWindow.Call(app.pptxSlideCombo, 6, 4, uintptr(comboW), 280, 1)
}

func pptxSlideLabel(slide PPTXSlidePreview) string {
	label := fmt.Sprintf("Slide %d", slide.Number)
	if slide.Hidden {
		label += " · Hidden"
	}
	if slide.Title != "" {
		label += " · " + slide.Title
	} else if slide.Error != "" {
		label += " · Preview unavailable"
	}
	return label
}

func loadGDIImageBytes(data []byte) (uintptr, *iUnknown, bool) {
	if len(data) == 0 {
		return 0, nil, false
	}
	hglobal, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE, uintptr(len(data)))
	if hglobal == 0 {
		return 0, nil, false
	}
	mem, _, _ := procGlobalLock.Call(hglobal)
	if mem == 0 {
		procGlobalFree.Call(hglobal)
		return 0, nil, false
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(mem)), len(data)), data)
	procGlobalUnlock.Call(hglobal)

	var stream *iUnknown
	hr, _, _ := procCreateStreamOnHGlobal.Call(hglobal, 1, uintptr(unsafe.Pointer(&stream)))
	if hresultFailed(hr) || stream == nil {
		procGlobalFree.Call(hglobal)
		return 0, nil, false
	}
	var image uintptr
	status, _, _ := procGdipLoadImageStream.Call(uintptr(unsafe.Pointer(stream)), uintptr(unsafe.Pointer(&image)))
	if status != 0 || image == 0 {
		releaseUnknown(stream)
		return 0, nil, false
	}
	return image, stream, true
}

func loadPPTXSlideImages() {
	releasePPTXImages()
	if !app.pptxPreviewActive || app.pptxSlideIndex < 0 || app.pptxSlideIndex >= len(app.pptxDeck.Slides) {
		return
	}
	slide := app.pptxDeck.Slides[app.pptxSlideIndex]
	for i, shape := range slide.Shapes {
		if shape.Kind != PPTXShapeImage || shape.ImagePart == "" {
			continue
		}
		data, err := readOpenXMLPart(app.pptxDeck.PackagePath, shape.ImagePart, pptxImageReadLimit)
		if err != nil {
			continue
		}
		image, stream, ok := loadGDIImageBytes(data)
		if !ok {
			continue
		}
		app.pptxImages = append(app.pptxImages, pptxLoadedImage{shapeIndex: i, image: image, stream: stream})
	}
}

func showPPTXSlide(index int) {
	if !app.pptxPreviewActive || index < 0 || index >= len(app.pptxDeck.Slides) {
		return
	}
	app.pptxSlideIndex = index
	procSendMessageW.Call(app.pptxSlideCombo, CB_SETCURSEL, uintptr(index), 0)
	app.previewPanX, app.previewPanY = 0, 0
	loadPPTXSlideImages()
	slide := app.pptxDeck.Slides[index]
	status := fmt.Sprintf("DCM PPTX Preview: Slide %d of %d", index+1, len(app.pptxDeck.Slides))
	if slide.Hidden {
		status += " · hidden slide"
	}
	if slide.Error != "" {
		status += " · slide preview unavailable"
	} else {
		images := 0
		for _, shape := range slide.Shapes {
			if shape.Kind == PPTXShapeImage {
				images++
			}
		}
		status += fmt.Sprintf(" · %d objects · %d images", len(slide.Shapes), images)
	}
	setText(app.statusLabel, status)
	procInvalidateRect.Call(app.previewHwnd, 0, 1)
}

func showPPTXPreview(deck PPTXPresentationPreview) bool {
	if app.pptxSlideCombo == 0 || len(deck.Slides) == 0 {
		return false
	}
	hideTextPreview()
	hideXLSXPreview()
	app.pptxPreviewActive = true
	app.pptxDeck = deck
	app.pptxSlideIndex = 0
	app.previewFit = true
	app.previewFitWidth = false
	app.previewZoom = 100
	app.previewPanX, app.previewPanY = 0, 0
	procSendMessageW.Call(app.pptxSlideCombo, CB_RESETCONTENT, 0, 0)
	for _, slide := range deck.Slides {
		label := pptxSlideLabel(slide)
		procSendMessageW.Call(app.pptxSlideCombo, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(label))))
	}
	procShowWindow.Call(app.pptxSlideCombo, SW_SHOW)
	resizePPTXPreview()
	showPPTXSlide(0)
	app.fallback = ""
	updatePreviewZoomControls()
	return true
}

func setPPTXPreviewZoom(percent int) {
	if !app.pptxPreviewActive {
		return
	}
	app.previewFit = false
	app.previewFitWidth = false
	app.previewZoom = clampPreviewZoom(percent)
	app.previewPanX, app.previewPanY = 0, 0
	setText(app.statusLabel, fmt.Sprintf("PPTX preview zoom: %d%% of Fit · drag or mouse wheel to pan", app.previewZoom))
	procInvalidateRect.Call(app.previewHwnd, 0, 1)
	updatePreviewZoomControls()
}

func pptxLoadedImageForShape(shapeIndex int) uintptr {
	for _, item := range app.pptxImages {
		if item.shapeIndex == shapeIndex {
			return item.image
		}
	}
	return 0
}

func drawPPTXText(hdc uintptr, text string, rc RECT, title bool, table bool) {
	if strings.TrimSpace(text) == "" || rc.Right <= rc.Left || rc.Bottom <= rc.Top {
		return
	}
	fontPx := int32(14)
	weight := int32(FW_NORMAL)
	if title {
		fontPx = 20
		weight = FW_SEMIBOLD
	} else if table {
		fontPx = 12
	}
	if !app.previewFit {
		fontPx = int32(int64(fontPx) * int64(app.previewZoom) / 100)
	}
	if fontPx < 8 {
		fontPx = 8
	}
	if fontPx > 48 {
		fontPx = 48
	}
	font := createUIFont(-fontPx, weight)
	oldFont, _, _ := procSelectObject.Call(hdc, font)
	procSetBkMode.Call(hdc, TRANSPARENT)
	procSetTextColor.Call(hdc, colorInk)
	flags := uintptr(0x00000010 | 0x00000800 | 0x00000040) // DT_WORDBREAK | DT_NOPREFIX | DT_EXPANDTABS
	ptr := wstr(text)
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(ptr)), ^uintptr(0), uintptr(unsafe.Pointer(&rc)), flags)
	procSelectObject.Call(hdc, oldFont)
	procDeleteObject.Call(font)
}

func drawPPTXImagePlaceholder(hdc uintptr, rc RECT, shape PPTXShapePreview) {
	brush, _, _ := procCreateSolidBrush.Call(rgb(238, 242, 247))
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), brush)
	procDeleteObject.Call(brush)
	label := "Image"
	if shape.AltText != "" {
		label = shape.AltText
	} else if shape.Name != "" {
		label = shape.Name
	}
	if shape.ImagePart != "" {
		label += "\n" + filepath.Base(filepath.FromSlash(shape.ImagePart))
	}
	inner := RECT{Left: rc.Left + 6, Top: rc.Top + 6, Right: rc.Right - 6, Bottom: rc.Bottom - 6}
	drawPPTXText(hdc, label, inner, false, false)
}

func drawPPTXPreview(hdc uintptr, rc RECT) {
	if !app.pptxPreviewActive || app.pptxSlideIndex < 0 || app.pptxSlideIndex >= len(app.pptxDeck.Slides) {
		return
	}
	const comboBarH int32 = 36
	availW := rc.Right - rc.Left - 16
	availH := rc.Bottom - rc.Top - comboBarH - 12
	if availW < 1 || availH < 1 {
		return
	}
	dw, dh := pptxSlidePixelSizeMode(app.pptxDeck.Width, app.pptxDeck.Height, availW, availH, app.previewZoom, app.previewFit, app.previewFitWidth)
	if dw <= 0 || dh <= 0 {
		return
	}
	app.previewPanX = clampPreviewPan(dw, availW, app.previewPanX)
	app.previewPanY = clampPreviewPan(dh, availH, app.previewPanY)
	slideX := rc.Left + 8 + (availW-dw)/2 + app.previewPanX
	slideY := rc.Top + comboBarH + 4 + (availH-dh)/2 + app.previewPanY

	brush, _, _ := procCreateSolidBrush.Call(rgb(255, 255, 255))
	pen, _, _ := procCreatePen.Call(PS_SOLID, 1, colorBorder)
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	procRectangle.Call(hdc, uintptr(slideX), uintptr(slideY), uintptr(slideX+dw), uintptr(slideY+dh))
	procSelectObject.Call(hdc, oldBrush)
	procSelectObject.Call(hdc, oldPen)
	procDeleteObject.Call(brush)
	procDeleteObject.Call(pen)

	slide := app.pptxDeck.Slides[app.pptxSlideIndex]
	if slide.Error != "" {
		inner := RECT{Left: slideX + 24, Top: slideY + 24, Right: slideX + dw - 24, Bottom: slideY + dh - 24}
		drawPPTXText(hdc, renderPPTXSlideText(slide), inner, false, false)
		return
	}

	fallbackIndex := 0
	for i, shape := range slide.Shapes {
		x, y, w, h := pptxShapePixelRect(shape, app.pptxDeck.Width, app.pptxDeck.Height, slideX, slideY, dw, dh, fallbackIndex)
		if shape.CX <= 0 || shape.CY <= 0 {
			fallbackIndex++
		}
		if w <= 0 || h <= 0 {
			continue
		}
		sr := RECT{Left: x, Top: y, Right: x + w, Bottom: y + h}
		switch shape.Kind {
		case PPTXShapeImage:
			image := pptxLoadedImageForShape(i)
			if image == 0 {
				drawPPTXImagePlaceholder(hdc, sr, shape)
				continue
			}
			var iw, ih uint32
			procGdipGetImageWidth.Call(image, uintptr(unsafe.Pointer(&iw)))
			procGdipGetImageHeight.Call(image, uintptr(unsafe.Pointer(&ih)))
			drawW, drawH := fitDimensions(int32(iw), int32(ih), w, h)
			ix := x + (w-drawW)/2
			iy := y + (h-drawH)/2
			var graphics uintptr
			if st, _, _ := procGdipCreateFromHDC.Call(hdc, uintptr(unsafe.Pointer(&graphics))); st != 0 || graphics == 0 {
				drawPPTXImagePlaceholder(hdc, sr, shape)
				continue
			}
			procGdipDrawImageRectI.Call(graphics, image, uintptr(ix), uintptr(iy), uintptr(drawW), uintptr(drawH))
			procGdipDeleteGraphics.Call(graphics)
		case PPTXShapeTable:
			bg, _, _ := procCreateSolidBrush.Call(rgb(250, 251, 253))
			procFillRect.Call(hdc, uintptr(unsafe.Pointer(&sr)), bg)
			procDeleteObject.Call(bg)
			inner := RECT{Left: sr.Left + 5, Top: sr.Top + 5, Right: sr.Right - 5, Bottom: sr.Bottom - 5}
			drawPPTXText(hdc, shape.Text, inner, false, true)
		case PPTXShapeText:
			inner := RECT{Left: sr.Left + 3, Top: sr.Top + 2, Right: sr.Right - 3, Bottom: sr.Bottom - 2}
			isTitle := strings.Contains(strings.ToLower(shape.Name), "title") || (slide.Title != "" && strings.HasPrefix(shape.Text, slide.Title))
			drawPPTXText(hdc, shape.Text, inner, isTitle, false)
		}
	}
	if slide.Hidden {
		label := "Hidden slide"
		badge := RECT{Left: slideX + 8, Top: slideY + 8, Right: slideX + 120, Bottom: slideY + 30}
		drawPPTXText(hdc, label, badge, false, false)
	}
}

func setDestinationText(path string) {
	app.updatingSuggestions = true
	setText(app.destEdit, path)
	app.updatingSuggestions = false
	app.destDir = strings.TrimSpace(path)
	moveEditCaretToEnd(app.destEdit)
	hideDestinationSuggestions()
	syncDestinationAliasSelection(path)
	updateOpenDestinationAvailability()
}

func moveEditCaretToEnd(hwnd uintptr) {
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	procSendMessageW.Call(hwnd, EM_SETSEL, n, n)
}

func hideDestinationSuggestions() {
	if app.destSuggestionList != 0 {
		procShowWindow.Call(app.destSuggestionList, 0) // SW_HIDE
	}
	app.destSuggestionsVisible = false
}

func updateDestinationSuggestions() {
	// Destination is a Vault logical location. Type-ahead comes only from the
	// configured Vault address index; arbitrary filesystem paths are never offered.
	typed := strings.TrimSpace(getText(app.destEdit))
	app.destDir = typed
	syncDestinationAliasSelection(typed)
	updateOpenDestinationAvailability()
	app.destSuggestions = nil
	procSendMessageW.Call(app.destSuggestionList, LB_RESETCONTENT, 0, 0)
	root := loadVaultRootSetting()
	if root == "" || typed == "" {
		hideDestinationSuggestions()
		return
	}
	idx, err := loadVaultAddressIndex(root)
	if err != nil {
		hideDestinationSuggestions()
		return
	}
	entries := vaultDestinationFolderSuggestions(idx, typed, 12)
	for _, entry := range entries {
		app.destSuggestions = append(app.destSuggestions, entry.Address)
		procSendMessageW.Call(app.destSuggestionList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(entry.Address))))
	}
	if len(app.destSuggestions) == 0 {
		hideDestinationSuggestions()
		return
	}
	procSendMessageW.Call(app.destSuggestionList, LB_SETCURSEL, ^uintptr(0), 0)
	procShowWindow.Call(app.destSuggestionList, SW_SHOW)
	app.destSuggestionsVisible = true
}

func focusWithinDestination() bool {
	focus, _, _ := procGetFocus.Call()
	return focus == app.destEdit || focus == app.destSuggestionList
}

func selectedDestinationSuggestion() (string, bool) {
	idx, _, _ := procSendMessageW.Call(app.destSuggestionList, LB_GETCURSEL, 0, 0)
	if idx == LB_ERR || int(idx) < 0 || int(idx) >= len(app.destSuggestions) {
		return "", false
	}
	return app.destSuggestions[int(idx)], true
}

func acceptDestinationSuggestion() bool {
	p, ok := selectedDestinationSuggestion()
	if !ok {
		return false
	}
	setDestinationText(p)
	procSetFocus.Call(app.destEdit)
	return true
}

func moveDestinationSuggestionSelection(delta int) bool {
	if !app.destSuggestionsVisible || len(app.destSuggestions) == 0 {
		return false
	}
	idx, _, _ := procSendMessageW.Call(app.destSuggestionList, LB_GETCURSEL, 0, 0)
	current := -1
	if idx != LB_ERR {
		current = int(idx)
	}
	if delta > 0 {
		current++
	} else {
		current--
	}
	if current < 0 {
		current = 0
	}
	if current >= len(app.destSuggestions) {
		current = len(app.destSuggestions) - 1
	}
	procSendMessageW.Call(app.destSuggestionList, LB_SETCURSEL, uintptr(current), 0)
	return true
}

func keyDown(vk uintptr) bool {
	state, _, _ := procGetKeyState.Call(vk)
	return uint16(state)&0x8000 != 0
}

func clickMnemonicControl(hwnd uintptr) bool {
	if !focusableControl(hwnd) {
		return false
	}
	procSendMessageW.Call(hwnd, BM_CLICK, 0, 0)
	return true
}

func activateAltShortcut(key rune) bool {
	switch altShortcutAction(key) {
	case ShortcutBrowseSource:
		return clickMnemonicControl(app.sourceBtn)
	case ShortcutReadText:
		return clickMnemonicControl(app.readTextCheck)
	case ShortcutBrowseDestination:
		return clickMnemonicControl(app.destBtn)
	case ShortcutManageDestinations:
		openDestinationManager()
		return true
	case ShortcutRename:
		return clickMnemonicControl(app.renameBtn)
	case ShortcutMove:
		return clickMnemonicControl(app.moveBtn)
	case ShortcutNext:
		return clickMnemonicControl(app.nextBtn)
	case ShortcutSkip:
		// Backward-compatible Alt+S from earlier builds now performs Next.
		return clickMnemonicControl(app.nextBtn)
	case ShortcutUndo:
		return clickMnemonicControl(app.undoBtn)
	case ShortcutOpen:
		return clickMnemonicControl(app.openBtn)
	case ShortcutOpenFolder:
		return clickMnemonicControl(app.openFolderBtn)
	case ShortcutOpenDestination:
		return clickMnemonicControl(app.destOpenBtn)
	case ShortcutApplyRenameTool:
		return clickMnemonicControl(app.renameToolBtn)
	case ShortcutFocusNewName:
		focusNewNameForTyping()
		return true
	case ShortcutFocusDestinationPath:
		focusDestinationForTyping()
		return true
	case ShortcutFocusSourcePath:
		focusSourceForTyping()
		return true
	case ShortcutFocusFileList:
		return focusFileList()
	default:
		return false
	}
}

func showKeyboardCues(hwnd uintptr) {
	procSendMessageW.Call(hwnd, WM_CHANGEUISTATE, makeWParam(UIS_CLEAR, UISF_HIDEACCEL), 0)
}

func primaryFocusOrder() []uintptr {
	// Workflow-first order rather than raw Win32 creation order. Read-only
	// labels/source text and the transient destination suggestion list are
	// intentionally excluded.
	return []uintptr{
		app.sourceAliasCombo,
		app.sourceEdit,
		app.sourceBtn,
		app.recoveryBtn,
		app.reloadBtn,
		app.filterNameEdit,
		app.openFolderBtn,
		app.categoryCombo,
		app.extensionCombo,
		app.filterAgeCombo,
		app.filterSizeCombo,
		app.filterSortCombo,
		app.filterClearBtn,
		app.fileList,
		app.readTextCheck,
		app.zoomOutBtn,
		app.zoom100Btn,
		app.zoomInBtn,
		app.zoomFitBtn,
		app.zoomFitWidthBtn,
		app.xlsxSheetCombo,
		app.pptxSlideCombo,
		app.nameEdit,
		app.renameToolCombo,
		app.renameToolBtn,
		app.destAliasCombo,
		app.destEdit,
		app.destBtn,
		app.destOpenBtn,
		app.renameBtn,
		app.moveBtn,
		app.nextBtn,
		app.openBtn,
		app.fileFlowBtn,
		app.processedSearchEdit,
		app.processedList,
		app.undoBatchBtn,
		app.undoBtn,
	}
}

func focusableControl(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	enabled, _, _ := procIsWindowEnabled.Call(hwnd)
	return visible != 0 && enabled != 0
}

func movePrimaryFocus(reverse bool) bool {
	focus, _, _ := procGetFocus.Call()
	if focus == app.fileList && !reverse {
		focusNewNameForTyping()
		return true
	}
	order := primaryFocusOrder()
	if len(order) == 0 {
		return false
	}
	current := -1
	for i, hwnd := range order {
		if focus == hwnd {
			current = i
			break
		}
	}
	// Treat transient source/destination suggestion lists as part of their edit fields.
	if focus == app.sourceSuggestionList {
		for i, hwnd := range order {
			if hwnd == app.sourceEdit {
				current = i
				break
			}
		}
	}
	if focus == app.destSuggestionList {
		for i, hwnd := range order {
			if hwnd == app.destEdit {
				current = i
				break
			}
		}
	}
	enabled := make([]bool, len(order))
	for i, hwnd := range order {
		enabled[i] = focusableControl(hwnd)
	}
	next := nextEnabledIndex(current, enabled, reverse)
	if next < 0 {
		return false
	}
	if app.sourceSuggestionsVisible && order[next] != app.sourceEdit {
		hideSourceSuggestions()
	}
	if app.destSuggestionsVisible && order[next] != app.destEdit {
		hideDestinationSuggestions()
	}
	procSetFocus.Call(order[next])
	return true
}

func focusNewNameForTyping() {
	if !focusableControl(app.nameEdit) {
		return
	}
	procSetFocus.Call(app.nameEdit)
	procSendMessageW.Call(app.nameEdit, EM_SETSEL, 0, ^uintptr(0))
}

func focusSourceForTyping() {
	if !focusableControl(app.sourceEdit) {
		return
	}
	procSetFocus.Call(app.sourceEdit)
	moveEditCaretToEnd(app.sourceEdit)
}

func focusFileList() bool {
	if !focusableControl(app.fileList) || len(app.files) == 0 {
		return false
	}
	hideSourceSuggestions()
	hideDestinationSuggestions()
	syncFileListSelection()
	procSetFocus.Call(app.fileList)
	return true
}

func focusDestinationForTyping() {
	procSetFocus.Call(app.destEdit)
	moveEditCaretToEnd(app.destEdit)
}

func setBackgroundMoveBusy(busy bool) {
	app.backgroundMoveBusy = busy
	controls := []uintptr{
		app.sourceAliasCombo, app.sourceEdit, app.sourceBtn, app.reloadBtn, app.categoryCombo, app.extensionCombo, app.filterNameEdit, app.filterAgeCombo, app.filterSizeCombo, app.filterSortCombo, app.filterClearBtn, app.readTextCheck,
		app.fileList, app.nameEdit, app.renameToolCombo, app.renameToolBtn,
		app.destAliasCombo, app.destEdit, app.destBtn,
		app.renameBtn, app.fileFlowBtn, app.moveBtn, app.nextBtn, app.processedSearchEdit, app.processedList, app.undoBatchBtn, app.undoBtn, app.openBtn, app.openFolderBtn,
	}
	flag := uintptr(1)
	if busy {
		flag = 0
	}
	for _, hwnd := range controls {
		if hwnd != 0 {
			procEnableWindow.Call(hwnd, flag)
		}
	}
	if !busy {
		updateReadTextAvailability(currentFile())
		updateOpenDestinationAvailability()
		if !app.batchMoveActive {
			populateProcessedList(-1)
		}
		if app.fileList != 0 && len(app.files) == 0 {
			procEnableWindow.Call(app.fileList, 0)
		}
	}
}

func finishSuccessfulFileOperation(source, target string, rec MoveRecord, sourceRemoved bool) {
	mode := app.pendingFileOperation
	app.pendingFileOperation = fileOperationNone
	if mode == fileOperationMove && app.batchMoveActive {
		defer batchMoveStepFinished(true)
	}
	if mode == fileOperationRename {
		if err := migrateFileFlowDraft(fileFlowDraftStorePath(), source, target); err != nil {
			showError("The rename succeeded, but staged FileFlow metadata could not follow the renamed path:\n" + err.Error())
		}
	} else if mode == fileOperationMove {
		commitStagedFileFlowMetadata(source, target, rec)
	}
	recordProcessed(rec)
	if err := syncConfiguredVaultMove(source, target); err != nil {
		setText(app.statusLabel, "File operation completed, but the Vault address index could not be updated: "+err.Error())
	}
	if source != currentFile() {
		reloadFiles()
		return
	}
	if mode == fileOperationRename && !sourceRemoved {
		app.files[app.index] = target
		populateFileList()
		showCurrent(false)
		focusFileList()
		return
	}
	removeCurrentFromWorkingList()
}

func cancelPendingFileOperation() {
	mode := app.pendingFileOperation
	app.pendingFileOperation = fileOperationNone
	if mode == fileOperationMove && app.batchMoveActive {
		batchMoveStepFinished(false)
	}
}

func beginBackgroundFallbackMove(source, target string, journalRec MoveRecord) {
	if app.backgroundMoveBusy {
		setText(app.statusLabel, "A background move is already running.")
		return
	}
	app.activeMoveJournal = journalRec
	setBackgroundMoveBusy(true)
	setText(app.statusLabel, "Moving in background: "+filepath.Base(source)+" → "+target+" | Durable journal active; the window stays responsive.")
	hwnd := app.hwnd
	progressFn := func(copied, total int64) {
		progress := BackgroundMoveProgress{Source: source, Target: target, Copied: copied, Total: total}
		select {
		case backgroundMoveProgress <- progress:
			procPostMessageW.Call(hwnd, WM_BACKGROUND_MOVE_PROGRESS, 0, 0)
		default:
		}
	}
	results := startBackgroundCopyMoveWithProgress(source, target, copyFileVerifiedWithProgress, removeSourceAfterAdvance, progressFn)
	go func() {
		result, ok := <-results
		if !ok {
			return
		}
		backgroundMoveResults <- result
		procPostMessageW.Call(hwnd, WM_BACKGROUND_MOVE_DONE, 0, 0)
	}()
}

func finishBackgroundMoveProgress() {
	var latest BackgroundMoveProgress
	found := false
	for {
		select {
		case latest = <-backgroundMoveProgress:
			found = true
		default:
			if !found || !app.backgroundMoveBusy {
				return
			}
			if latest.Total <= 0 {
				setText(app.statusLabel, "Moving in background: "+filepath.Base(latest.Source)+" | The window stays responsive.")
				return
			}
			pct := float64(latest.Copied) * 100 / float64(latest.Total)
			copiedMB := float64(latest.Copied) / (1024 * 1024)
			totalMB := float64(latest.Total) / (1024 * 1024)
			setText(app.statusLabel, fmt.Sprintf("Moving in background: %.1f%% · %.1f / %.1f MB · %s", pct, copiedMB, totalMB, filepath.Base(latest.Source)))
			return
		}
	}
}

func finishBackgroundFallbackMove() {
	var result BackgroundMoveResult
	select {
	case result = <-backgroundMoveResults:
	default:
		return
	}
	journalRec := app.activeMoveJournal
	app.activeMoveJournal = MoveRecord{}
	setBackgroundMoveBusy(false)
	if result.Source != currentFile() {
		reloadFiles()
	}
	switch result.Status {
	case BackgroundMoveSucceeded:
		result.Record.JournalID = journalRec.JournalID
		result.Record.BatchID = journalRec.BatchID
		journalErr := completeMoveJournal(moveJournalFilePath(), result.Record, time.Now())
		finishSuccessfulFileOperation(result.Source, result.Target, result.Record, false)
		if journalErr != nil {
			setText(app.statusLabel, "Background move completed, but journal finalization needs recovery on next launch: "+result.Target)
			showError("The file move completed safely, but DCM could not finalize its durable journal:\n" + journalErr.Error() + "\n\nDo not edit or delete the move journal manually. DCM will reconcile the pending entry on the next launch.")
		} else {
			setText(app.statusLabel, "Background move complete and journaled: "+result.Target)
		}
	case BackgroundMoveCopiedSourceRetained:
		cancelPendingFileOperation()
		if journalRec.JournalID != "" {
			_ = failMoveJournal(moveJournalFilePath(), journalRec, MoveJournalAttention, "copy step finished but verified source retirement was refused; review source and destination", time.Now())
		}
		if result.Source == currentFile() {
			loadPreview(result.Source)
		} else {
			reloadFiles()
		}
		setText(app.statusLabel, "Source retained; the move needs attention before retrying: "+result.Target)
		showError("DCM could not verify and protect the source and destination for source removal:\n" + result.Err.Error() + "\n\nSource:\n" + result.Source + "\n\nDestination:\n" + result.Target + "\n\nReview these files before retrying. DCM marked the durable journal for attention.")
	default:
		cancelPendingFileOperation()
		if journalRec.JournalID != "" {
			_ = failMoveJournal(moveJournalFilePath(), journalRec, MoveJournalFailed, "source snapshot or background copy failed; source retirement was not attempted", time.Now())
		}
		if result.Source == currentFile() {
			loadPreview(result.Source)
		} else {
			reloadFiles()
		}
		setText(app.statusLabel, "Background move failed before verified source retirement; review the error before retrying.")
		if result.Err != nil {
			showError("Could not copy file to destination:\n" + result.Err.Error())
		}
	}
}

func continueMoveAfterConflict(source, target string) {
	rec, journalErr := beginMoveJournal(moveJournalFilePath(), MoveRecord{From: source, To: target, Kind: MoveKindMove, BatchID: app.batchMoveID}, time.Now())
	if journalErr != nil {
		cancelPendingFileOperation()
		loadPreview(source)
		setText(app.statusLabel, "Move not started: durable journal is unavailable.")
		showError("DCM refused to change the file because it could not persist the durable move journal:\n" + journalErr.Error() + "\n\nThe source file was left unchanged.")
		return
	}
	if err := renameFileNoReplace(source, target); err == nil {
		journalErr = completeMoveJournal(moveJournalFilePath(), rec, time.Now())
		finishSuccessfulFileOperation(source, target, rec, false)
		if journalErr != nil {
			setText(app.statusLabel, "Moved safely, but journal finalization will be reconciled on next launch: "+target)
			showError("The file moved successfully, but DCM could not finalize the durable journal:\n" + journalErr.Error() + "\n\nThe pending journal entry will be reconciled from the filesystem on the next launch.")
		} else {
			setText(app.statusLabel, "Moved and journaled: "+target)
		}
		return
	}
	beginBackgroundFallbackMove(source, target, rec)
}

func beginBackgroundConflictInspection(source, target string) {
	if app.backgroundMoveBusy {
		setText(app.statusLabel, "A background file operation is already running.")
		return
	}
	setBackgroundMoveBusy(true)
	setText(app.statusLabel, "Checking destination in background: "+filepath.Base(source)+" | The window stays responsive.")
	results := startBackgroundConflictInspection(source, target, inspectDestinationConflict)
	hwnd := app.hwnd
	go func() {
		result, ok := <-results
		if !ok {
			return
		}
		backgroundConflictResults <- result
		procPostMessageW.Call(hwnd, WM_BACKGROUND_CONFLICT_DONE, 0, 0)
	}()
}

func beginBackgroundConflictAction(kind BackgroundConflictActionKind, source, target string) {
	if app.backgroundMoveBusy {
		setText(app.statusLabel, "A background file operation is already running.")
		return
	}
	// Capture admission metadata before the worker can mutate files or append
	// its first journal event. Completion must not read a later active batch.
	batchID := ""
	if app.batchMoveActive {
		batchID = app.batchMoveID
	}
	setBackgroundMoveBusy(true)
	var label string
	var action func() (MoveRecord, error)
	switch kind {
	case BackgroundConflictRemoveDuplicate:
		label = "Verifying and removing exact duplicate in background: " + filepath.Base(source)
		action = func() (MoveRecord, error) {
			return removeExactDuplicateSourceJournaled(moveJournalFilePath(), source, target, batchID)
		}
	default:
		label = "Replacing destination safely in background: " + filepath.Base(source)
		action = func() (MoveRecord, error) {
			return replaceMoveWithRecoveryJournaled(moveJournalFilePath(), source, target, replacementRecoveryDirectory(), batchID)
		}
	}
	setText(app.statusLabel, label+" | The window stays responsive.")
	results := startBackgroundConflictAction(kind, source, target, action)
	hwnd := app.hwnd
	go func() {
		result, ok := <-results
		if !ok {
			return
		}
		backgroundConflictActionResults <- result
		procPostMessageW.Call(hwnd, WM_BACKGROUND_ACTION_DONE, 0, 0)
	}()
}

func finishBackgroundConflictInspection() {
	var result BackgroundConflictResult
	select {
	case result = <-backgroundConflictResults:
	default:
		return
	}
	setBackgroundMoveBusy(false)
	if result.Err != nil {
		cancelPendingFileOperation()
		if result.Source == currentFile() {
			loadPreview(result.Source)
		} else {
			reloadFiles()
		}
		setText(app.statusLabel, "Destination check failed; no files were changed.")
		showError("Could not inspect destination safely:\n" + result.Err.Error())
		return
	}

	switch result.Info.Kind {
	case ConflictExactDuplicate:
		switch promptExactDuplicate(result.Info) {
		case 6: // IDYES
			beginBackgroundConflictAction(BackgroundConflictRemoveDuplicate, result.Source, result.Target)
		case 7: // IDNO
			cancelPendingFileOperation()
			loadPreview(result.Source)
			setText(app.statusLabel, "Exact duplicate action skipped; source file was kept unchanged.")
		default:
			cancelPendingFileOperation()
			loadPreview(result.Source)
			setText(app.statusLabel, "Duplicate action cancelled; no files were changed.")
		}
	case ConflictDifferent:
		switch promptDifferentFileConflict(result.Info) {
		case 6: // IDYES: Keep Both
			continueMoveAfterConflict(result.Source, keepBothTarget(result.Target))
		case 7: // IDNO: Replace
			beginBackgroundConflictAction(BackgroundConflictReplace, result.Source, result.Target)
		default:
			cancelPendingFileOperation()
			loadPreview(result.Source)
			setText(app.statusLabel, "Filename conflict action cancelled; both files were kept unchanged.")
		}
	default:
		continueMoveAfterConflict(result.Source, result.Target)
	}
}

func finishBackgroundConflictAction() {
	var result BackgroundConflictActionResult
	select {
	case result = <-backgroundConflictActionResults:
	default:
		return
	}
	setBackgroundMoveBusy(false)
	if result.Err != nil {
		cancelPendingFileOperation()
		if result.Source == currentFile() {
			loadPreview(result.Source)
		} else {
			reloadFiles()
		}
		if result.Kind == BackgroundConflictRemoveDuplicate {
			setText(app.statusLabel, "Duplicate verification/removal failed; source was kept unless the error message states otherwise.")
			showError("Could not remove duplicate source safely:\n" + result.Err.Error())
		} else {
			setText(app.statusLabel, "Safe replacement failed; recovery rules were applied.")
			showError("Could not replace destination safely:\n" + result.Err.Error())
		}
		return
	}
	finishSuccessfulFileOperation(result.Source, result.Target, result.Record, result.Kind == BackgroundConflictRemoveDuplicate)
	if result.Kind == BackgroundConflictRemoveDuplicate {
		setText(app.statusLabel, "Removed exact duplicate source after background SHA-256 verification. Destination remains at: "+result.Target)
	} else {
		setText(app.statusLabel, "Replaced destination safely in background: "+result.Target+" — Alt+U can restore the previous destination this session.")
	}
}

func cleanupSessionReplacementBackups(history []MoveRecord) {
	for _, rec := range history {
		if rec.Kind == MoveKindReplace && strings.TrimSpace(rec.ReplacedBackup) != "" {
			_ = os.Remove(rec.ReplacedBackup)
		}
	}
}

func removeCurrentFromWorkingList() {
	if app.index < 0 || app.index >= len(app.files) {
		return
	}
	app.files = append(app.files[:app.index], app.files[app.index+1:]...)
	populateFileList()
	showCurrent(false)
	focusFileList()
}

func promptExactDuplicate(info DestinationConflictInfo) int {
	msg := "An exact duplicate already exists at the destination.\n\n" + info.Target +
		"\n\nDCM verified the duplicate using file size and SHA-256.\n\n" +
		"Yes = Remove the source duplicate\n" +
		"No = Keep the source unchanged\n" +
		"Cancel = Stay on this file"
	result, _, _ := procMessageBoxW.Call(app.hwnd, uintptr(unsafe.Pointer(wstr(msg))), uintptr(unsafe.Pointer(wstr("Exact Duplicate — DCM - Quick Sort"))), 0x03|0x20)
	return int(result)
}

func promptDifferentFileConflict(info DestinationConflictInfo) int {
	detail := "The destination already contains a different file with this name.\n\n" + info.Target + "\n\n"
	if info.SourceSize != info.TargetSize {
		detail += fmt.Sprintf("The file sizes differ (%d bytes vs %d bytes).\n\n", info.SourceSize, info.TargetSize)
	} else {
		detail += "The files are the same size but their SHA-256 checksums differ.\n\n"
	}
	detail += "Yes = Keep Both (DCM chooses a numbered filename)\n" +
		"No = Replace the existing destination (recoverable with Undo this session)\n" +
		"Cancel = Skip this file and keep both files unchanged"
	result, _, _ := procMessageBoxW.Call(app.hwnd, uintptr(unsafe.Pointer(wstr(detail))), uintptr(unsafe.Pointer(wstr("Filename Conflict — DCM - Quick Sort"))), 0x03|0x30)
	return int(result)
}

func showError(msg string) {
	procMessageBoxW.Call(app.hwnd, uintptr(unsafe.Pointer(wstr(msg))), uintptr(unsafe.Pointer(wstr("DCM - Quick Sort"))), 0x10)
}

func mainWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		app.hwnd = hwnd
		app.processedCollapsed = make(map[string]bool)
		initVisualResources()
		styleTitleBar(hwnd)
		createUI(hwnd)
		app.pdfDefaultView = loadPDFDefaultViewSetting()
		app.mainMenu = createMainMenu(hwnd)
		showKeyboardCues(hwnd)
		initDefaultFolders()
		applyStartupArgument()
		reloadSourceAliases()
		reloadDestinationAliases()
		populateCategoryFilter()
		populateExtensionFilter(CategoryAll)
		populateInboxFilters()
		populateRenameTools()
		reloadFiles()
		journalSummary := recoverDurableMoveHistory()
		populateProcessedList(-1)
		refreshRecoveryCenterBadge()
		refreshLearningCenterBadge()
		if compactSummary, compactErr := compactMoveJournal(moveJournalFilePath(), defaultJournalCompactionThreshold); compactErr == nil && compactSummary.Compacted {
			journalSummary = strings.TrimSpace(journalSummary + fmt.Sprintf(" · journal compacted %d→%d events", compactSummary.BeforeEvents, compactSummary.AfterEvents))
		}
		if journalSummary != "" {
			setText(app.statusLabel, journalSummary+" | Alt+U can undo restored completed moves")
		} else {
			setText(app.statusLabel, "Select a file or processed transaction. F5 reloads the source; Ctrl+\\ fits the preview.")
		}
		return 0
	case wmLayoutTraceFence:
		layoutTraceFence(hwnd, wParam)
		return 0
	case WM_PAINT:
		recordLayoutTrace(hwnd, "paint-enter", 0)
		r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		recordLayoutTrace(hwnd, "paint-exit", 0)
		return r
	case WM_SIZE:
		recordLayoutTrace(hwnd, "size-enter", 0)
		layoutUI(hwnd)
		resizePreviewHandler()
		resizeTextPreview()
		resizeXLSXPreview()
		layoutPDFPreviewControls()
		recordLayoutTrace(hwnd, "size-exit", 0)
		return 0
	case WM_ERASEBKGND:
		recordLayoutTrace(hwnd, "erase", 0)
		if app.brushWindow != 0 {
			var rc RECT
			procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
			procFillRect.Call(wParam, uintptr(unsafe.Pointer(&rc)), app.brushWindow)
			return 1
		}
	case WM_CTLCOLORSTATIC:
		child := lParam
		procSetBkMode.Call(wParam, TRANSPARENT)
		if child == app.statusLabel {
			procSetTextColor.Call(wParam, colorAccentHot)
			procSetBkMode.Call(wParam, OPAQUE)
			procSetBkColor.Call(wParam, colorStatusBG)
			return app.brushStatus
		}
		if child == app.countLabel || child == app.currentLabel || child == app.processedTitleLabel {
			procSetTextColor.Call(wParam, colorAccentHot)
		} else {
			procSetTextColor.Call(wParam, colorInk)
		}
		return app.brushWindow
	case WM_CTLCOLOREDIT, WM_CTLCOLORLISTBOX:
		procSetTextColor.Call(wParam, colorInk)
		procSetBkColor.Call(wParam, colorPanelBG)
		return app.brushPanel
	case WM_CTLCOLORBTN:
		procSetTextColor.Call(wParam, colorInk)
		procSetBkMode.Call(wParam, TRANSPARENT)
		return app.brushWindow
	case WM_DRAWITEM:
		dis := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if dis != nil && int(dis.CtlID) == ID_MOVE {
			drawPrimaryButton(dis)
			return 1
		}
	case WM_COMMAND:
		id := int(loword(wParam))
		notify := hiword(wParam)
		if notify == BN_CLICKED {
			switch id {
			case ID_SOURCE_BTN:
				if p := browseFolder(hwnd, "Choose folder to sort", app.sourceDir); p != "" {
					activateSourcePath(p)
				}
			case ID_SOURCE_CONFIG:
				openSourceManager()
			case ID_DEST_BTN:
				openDestinationManager()
			case ID_OPEN_DEST:
				openDestinationFolder()
			case ID_DEST_CONFIG:
				openDestinationManager()
			case ID_APPLY_RENAME:
				applySelectedRenameTool()
			case ID_FILTER_CLEAR:
				clearInboxFilters()
			case ID_RECOVERY_CENTER:
				openRecoveryCenter()
			case ID_LEARNING_CENTER:
				openLearningCenter()
			case ID_SUGGESTION_CENTER:
				openSuggestionCenter()
			case ID_FILEFLOW_HANDOFF:
				queueSelectedForFileFlow()
			case ID_RENAME:
				renameCurrent()
			case ID_MOVE:
				moveCurrent()
			case ID_NEXT:
				nextFile()
			case ID_UNDO:
				undoSelectedProcessed()
			case ID_UNDO_BATCH:
				undoLatestBatch()
			case ID_OPEN:
				openCurrent()
			case ID_OPEN_FOLDER:
				if current := currentFile(); current != "" {
					shellOpen(filepath.Dir(current))
				}
			case ID_RELOAD:
				typed := strings.TrimSpace(getText(app.sourceEdit))
				if typed != "" && !strings.EqualFold(filepath.Clean(typed), filepath.Clean(app.sourceDir)) {
					activateSourcePath(typed)
				} else {
					reloadFiles()
				}
				reloadSourceAliases()
				reloadDestinationAliases()
			case ID_ZOOM_OUT:
				app.previewZoomWheelRemainder = 0
				adjustPreviewZoom(-1)
			case ID_ZOOM_100:
				setPreviewZoom100()
			case ID_ZOOM_IN:
				app.previewZoomWheelRemainder = 0
				adjustPreviewZoom(1)
			case ID_ZOOM_FIT:
				fitPreview()
			case ID_ZOOM_FIT_WIDTH:
				fitPreviewWidth()
			case ID_VAULT_LIBRARY:
				chooseVaultRoot()
			case ID_NAMING_TEMPLATES:
				openNamingTemplateManager()
			case ID_PDF_DEFAULT_FIT_WIDTH:
				setPDFDefaultViewPreference(pdfDefaultViewFitWidth)
			case ID_PDF_DEFAULT_FIT_PAGE:
				setPDFDefaultViewPreference(pdfDefaultViewFitPage)
			case ID_MENU_ABOUT:
				showAboutDialog()
			case ID_MENU_EXIT:
				procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
			case ID_READ_TEXT:
				unloadPreview()
				if current := currentFile(); current != "" {
					loadPreview(current)
				}
			}
		}
		if notify == CBN_SELCHANGE {
			switch id {
			case ID_SOURCE_ALIAS:
				if !app.updatingSourceAlias {
					if alias, ok := selectedSourceAlias(); ok {
						activateSourcePath(alias.Path)
						setText(app.statusLabel, sourceStatus(alias))
					}
				}
			case ID_CATEGORY:
				category := selectedCategory()
				populateExtensionFilter(category)
				reloadFilesWithFocus(false)
			case ID_EXTENSION:
				reloadFilesWithFocus(false)
			case ID_RENAME_TOOL:
				applySelectedRenameTool()
			case ID_FILTER_AGE, ID_FILTER_SIZE:
				reloadFilesWithFocus(false)
			case ID_FILTER_SORT:
				reloadFilesWithFocus(false)
			case ID_DEST_ALIAS:
				if !app.updatingDestinationAlias {
					if alias, ok := selectedDestinationAlias(); ok {
						setDestinationText(alias.Path)
						setText(app.statusLabel, alias.Name+" → "+alias.Path)
					}
				}
			}
		}
		if notify == EN_CHANGE {
			switch id {
			case ID_SOURCE_EDIT:
				if !app.updatingSourceSuggestions {
					updateSourceSuggestions()
				}
			case ID_DEST_EDIT:
				if !app.updatingSuggestions {
					updateDestinationSuggestions()
				}
			case ID_FILTER_NAME:
				if !app.backgroundMoveBusy && !app.updatingInboxFilters {
					reloadFilesWithFocus(false)
				}
			case ID_PROCESSED_SEARCH:
				if !app.updatingProcessedList {
					populateProcessedList(-1)
				}
			}
		}
		if notify == LBN_SELCHANGE {
			switch id {
			case ID_SOURCE_LIST:
				acceptSourceSuggestion()
			case ID_DEST_LIST:
				acceptDestinationSuggestion()
			case ID_FILE_LIST:
				if !app.updatingFileList {
					updateCurrentFromSourceSelection()
				}
			case ID_PROCESSED_LIST:
				if !app.updatingProcessedList {
					row := selectedProcessedViewRow()
					if row != nil && row.Header {
						toggleProcessedGroup(row.GroupKey)
					} else {
						showSelectedProcessedStatus()
					}
				}
			}
		}
		return 0
	case WM_BACKGROUND_MOVE_DONE:
		finishBackgroundFallbackMove()
		return 0
	case WM_BACKGROUND_CONFLICT_DONE:
		finishBackgroundConflictInspection()
		return 0
	case WM_BACKGROUND_ACTION_DONE:
		finishBackgroundConflictAction()
		return 0
	case WM_BACKGROUND_MOVE_PROGRESS:
		finishBackgroundMoveProgress()
		return 0
	case WM_BATCH_MOVE_NEXT:
		advanceBatchMove()
		return 0
	case 0x0113: // WM_TIMER: owned Sumatra startup/liveness, never a rendering wait.
		if wParam == sumatraTimerID {
			pollSumatraPreview()
			return 0
		}
	case WM_PDF_RENDER_DONE:
		finishPDFRender()
		return 0
	case WM_CLOSE:
		if app.backgroundMoveBusy {
			procMessageBoxW.Call(app.hwnd, uintptr(unsafe.Pointer(wstr("A file operation is still running in the background. DCM will remain open until the operation finishes safely."))), uintptr(unsafe.Pointer(wstr("Background Operation — DCM - Quick Sort"))), 0x40)
			return 0
		}
		unloadPreview()
		procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return 0
	case WM_DESTROY:
		unloadPreview()
		if app.mainMenu != 0 {
			procSetMenu.Call(hwnd, 0)
			procDestroyMenu.Call(app.mainMenu)
			app.mainMenu = 0
			app.pdfDefaultViewMenu = 0
		}
		if app.gpToken != 0 {
			procGdiplusShutdown.Call(app.gpToken)
			app.gpToken = 0
		}
		freeVisualResources()
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func createUI(hwnd uintptr) {
	createControl("STATIC", "Source:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 18, 60, 20, hwnd, 0)
	app.sourceAliasCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 85, 13, 150, 240, hwnd, ID_SOURCE_ALIAS)
	app.sourceEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 245, 14, 370, 25, hwnd, ID_SOURCE_EDIT)
	app.sourceBtn = createControl("BUTTON", "&Browse…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 625, 13, 80, 27, hwnd, ID_SOURCE_BTN)
	app.recoveryBtn = createControl("BUTTON", "Recovery", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 811, 13, 100, 27, hwnd, ID_RECOVERY_CENTER)
	app.reloadBtn = createControl("BUTTON", "Reload", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 919, 13, 80, 27, hwnd, ID_RELOAD)

	createControl("STATIC", "Name:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 54, 60, 20, hwnd, 0)
	app.filterNameEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 85, 49, 1000, 25, hwnd, ID_FILTER_NAME)
	app.openFolderBtn = createControl("BUTTON", "Open &Folder", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 1095, 48, 110, 27, hwnd, ID_OPEN_FOLDER)

	createControl("STATIC", "File Type:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 87, 60, 20, hwnd, 0)
	app.categoryCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 85, 82, 165, 260, hwnd, ID_CATEGORY)
	createControl("STATIC", "Extension:", WS_CHILD|WS_VISIBLE|SS_LEFT, 265, 87, 70, 20, hwnd, 0)
	app.extensionCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 340, 82, 150, 320, hwnd, ID_EXTENSION)
	createControl("STATIC", "Age:", WS_CHILD|WS_VISIBLE|SS_LEFT, 505, 87, 35, 20, hwnd, 0)
	app.filterAgeCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 545, 82, 135, 220, hwnd, ID_FILTER_AGE)
	createControl("STATIC", "Size:", WS_CHILD|WS_VISIBLE|SS_LEFT, 695, 87, 35, 20, hwnd, 0)
	app.filterSizeCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 735, 82, 135, 220, hwnd, ID_FILTER_SIZE)
	app.filterSortLabel = createControl("STATIC", "Sort By:", WS_CHILD|WS_VISIBLE|SS_LEFT, 885, 87, 50, 20, hwnd, 0)
	app.filterSortCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 940, 82, 190, 220, hwnd, ID_FILTER_SORT)
	app.filterClearBtn = createControl("BUTTON", "Clear &Filters", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 1138, 81, 105, 28, hwnd, ID_FILTER_CLEAR)

	app.countLabel = createControl("STATIC", "FILES", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 120, 220, 24, hwnd, 0)
	app.currentLabel = createControl("STATIC", "PREVIEW", WS_CHILD|WS_VISIBLE|SS_LEFT, 250, 120, 720, 24, hwnd, 0)
	app.readTextCheck = createControl("BUTTON", "Read as &text", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, 650, 116, 105, 27, hwnd, ID_READ_TEXT)
	app.processedTitleLabel = createControl("STATIC", "PROCESSED · 0", WS_CHILD|WS_VISIBLE|SS_LEFT, 980, 120, 92, 24, hwnd, 0)
	app.undoBatchBtn = createControl("BUTTON", "Undo &Batch", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 1076, 116, 88, 27, hwnd, ID_UNDO_BATCH)
	app.undoBtn = createControl("BUTTON", "&Undo", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 1168, 116, 72, 27, hwnd, ID_UNDO)
	setControlFont(app.countLabel, app.headingFont)
	setControlFont(app.currentLabel, app.headingFont)
	setControlFont(app.processedTitleLabel, app.headingFont)
	app.zoomOutBtn = createControl("BUTTON", "-", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 790, 116, 34, 27, hwnd, ID_ZOOM_OUT)
	app.zoom100Btn = createControl("BUTTON", "100%", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 828, 116, 56, 27, hwnd, ID_ZOOM_100)
	app.zoomInBtn = createControl("BUTTON", "+", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 888, 116, 34, 27, hwnd, ID_ZOOM_IN)
	app.zoomFitBtn = createControl("BUTTON", "Fit", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 926, 116, 44, 27, hwnd, ID_ZOOM_FIT)
	app.zoomFitWidthBtn = createControl("BUTTON", "Width", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 974, 116, 54, 27, hwnd, ID_ZOOM_FIT_WIDTH)
	app.fileList = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|LBS_EXTENDEDSEL|WS_VSCROLL, 20, 148, 220, 362, hwnd, ID_FILE_LIST)
	app.previewHwnd = createControl("QuickSortPreview", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_CLIPCHILDREN, 250, 148, 720, 362, hwnd, 0)
	createPDFPreviewControls()
	app.processedSearchEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 980, 148, 260, 25, hwnd, ID_PROCESSED_SEARCH)
	procSendMessageW.Call(app.processedSearchEdit, EM_SETCUEBANNER, 1, uintptr(unsafe.Pointer(wstr("Search processed..."))))
	app.processedList = createControl("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|LBS_NOTIFY|WS_VSCROLL, 980, 181, 260, 329, hwnd, ID_PROCESSED_LIST)
	setExplorerTheme(app.fileList)
	setExplorerTheme(app.processedSearchEdit)
	setExplorerTheme(app.processedList)
	setExplorerTheme(app.categoryCombo)
	setExplorerTheme(app.extensionCombo)
	setExplorerTheme(app.filterNameEdit)
	setExplorerTheme(app.filterAgeCombo)
	setExplorerTheme(app.filterSizeCombo)
	setExplorerTheme(app.filterSortCombo)
	setExplorerTheme(app.filterClearBtn)
	setExplorerTheme(app.renameToolCombo)
	setExplorerTheme(app.renameToolBtn)
	setExplorerTheme(app.sourceAliasCombo)
	setExplorerTheme(app.destAliasCombo)
	setExplorerTheme(app.sourceBtn)
	setExplorerTheme(app.reloadBtn)
	setExplorerTheme(app.recoveryBtn)
	setExplorerTheme(app.destBtn)
	setExplorerTheme(app.destOpenBtn)
	setExplorerTheme(app.renameBtn)
	setExplorerTheme(app.nextBtn)
	setExplorerTheme(app.undoBatchBtn)
	setExplorerTheme(app.undoBtn)
	setExplorerTheme(app.openBtn)
	setExplorerTheme(app.openFolderBtn)
	setExplorerTheme(app.zoomOutBtn)
	setExplorerTheme(app.zoom100Btn)
	setExplorerTheme(app.zoomInBtn)
	setExplorerTheme(app.zoomFitBtn)
	setExplorerTheme(app.zoomFitWidthBtn)
	app.textPreviewEdit = createControl("EDIT", "", WS_CHILD|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 0, 0, 720, 395, app.previewHwnd, 0)
	procSendMessageW.Call(app.textPreviewEdit, EM_SETLIMITTEXT, uintptr(defaultTextPreviewLimit+4096), 0)
	app.xlsxSheetCombo = createControl("COMBOBOX", "", WS_CHILD|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 6, 4, 708, 280, app.previewHwnd, ID_XLSX_SHEET)
	app.xlsxGridEdit = createControl("EDIT", "", WS_CHILD|WS_VSCROLL|WS_HSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_AUTOHSCROLL|ES_READONLY, 0, 34, 720, 361, app.previewHwnd, 0)
	procSendMessageW.Call(app.xlsxGridEdit, EM_SETLIMITTEXT, uintptr(1024*1024), 0)
	app.pptxSlideCombo = createControl("COMBOBOX", "", WS_CHILD|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 6, 4, 708, 280, app.previewHwnd, ID_PPTX_SLIDE)
	setExplorerTheme(app.xlsxSheetCombo)
	setExplorerTheme(app.pptxSlideCombo)

	app.nameLabel = createControl("STATIC", "New Name:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 530, 70, 20, hwnd, 0)
	app.nameEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 95, 525, 655, 28, hwnd, ID_NAME_EDIT)
	app.renameToolLabel = createControl("STATIC", "Tool:", WS_CHILD|WS_VISIBLE|SS_LEFT, 765, 530, 35, 20, hwnd, 0)
	app.renameToolCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 805, 525, 165, 220, hwnd, ID_RENAME_TOOL)
	app.renameToolBtn = createControl("BUTTON", "&Apply Tool", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 980, 524, 105, 29, hwnd, ID_APPLY_RENAME)
	setExplorerTheme(app.renameToolCombo)
	setExplorerTheme(app.renameToolBtn)
	app.destinationLabel = createControl("STATIC", "Favorite:", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 568, 70, 20, hwnd, 0)
	app.destAliasCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 95, 563, 205, 260, hwnd, ID_DEST_ALIAS)
	app.pathLabel = createControl("STATIC", "Vault Location:", WS_CHILD|WS_VISIBLE|SS_LEFT, 295, 568, 90, 20, hwnd, 0)
	app.destEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 390, 563, 645, 28, hwnd, ID_DEST_EDIT)
	app.destBtn = createControl("BUTTON", "&Manage…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 1045, 562, 100, 29, hwnd, ID_DEST_BTN)
	app.destOpenBtn = createControl("BUTTON", "Open Locatio&n", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 1155, 562, 135, 29, hwnd, ID_OPEN_DEST)

	app.renameBtn = createControl("BUTTON", "&Rename", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 20, 610, 90, 34, hwnd, ID_RENAME)
	app.moveBtn = createControl("BUTTON", "Move to &Vault", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 120, 610, 145, 34, hwnd, ID_MOVE)
	app.nextBtn = createControl("BUTTON", "Ne&xt", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 275, 610, 80, 34, hwnd, ID_NEXT)
	app.openBtn = createControl("BUTTON", "&Open", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 365, 610, 80, 34, hwnd, ID_OPEN)
	app.fileFlowBtn = createControl("BUTTON", "Handoff to &FileFlow", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 455, 610, 170, 34, hwnd, ID_FILEFLOW_HANDOFF)
	for _, c := range []uintptr{app.moveBtn, app.fileFlowBtn} {
		setExplorerTheme(c)
	}
	app.statusLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 658, 1010, 20, hwnd, 0)
	setControlFont(app.statusLabel, app.uiFont)
	app.keyboardHintLabel = createControl("STATIC", "Alt+W Name   Alt+P Destination   Alt+C Source   Alt+L Files   ↑↓ Navigate   Ctrl/Shift Select   Ctrl+A All   Tab → Name   Esc → Files", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 681, 1010, 18, hwnd, 0)
	setControlFont(app.keyboardHintLabel, app.smallFont)
	// Separate suggestion lists avoid editable COMBOBOX auto-selection quirks when
	// typing Windows path separators such as backslash. They are shown only while
	// type-ahead has matching folders.
	app.sourceSuggestionList = createControl("LISTBOX", "", WS_CHILD|WS_BORDER|LBS_NOTIFY|WS_VSCROLL, 245, 43, 370, 140, hwnd, ID_SOURCE_LIST)
	app.destSuggestionList = createControl("LISTBOX", "", WS_CHILD|WS_BORDER|LBS_NOTIFY|WS_VSCROLL, 85, 592, 760, 140, hwnd, ID_DEST_LIST)
	resetPreviewZoomState()
	setExplorerTheme(app.sourceSuggestionList)
	setExplorerTheme(app.destSuggestionList)
}

func layoutUI(hwnd uintptr) {
	beginLayoutTrace(hwnd)
	defer endLayoutTrace(hwnd)
	var rc RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	w := rc.Right - rc.Left
	h := rc.Bottom - rc.Top
	if w < 760 {
		w = 760
	}
	if h < 650 {
		h = 650
	}
	// About/help copy moved to the menu, so the old three-line footer no longer
	// consumes client height. Give that space back to the preview pane.
	previewH := h - 335
	if previewH < 245 {
		previewH = 245
	}
	listW := int32(220)
	processedW := int32(260)
	if sumatraView != nil && sumatraView.child != 0 {
		listW, processedW = sumatraSidebarWidths(w)
	}
	gap := int32(10)
	previewX := int32(20) + listW + gap
	processedX := w - 20 - processedW
	previewW := processedX - gap - previewX
	if previewW < 420 {
		previewW = 420
	}
	aliasWTop := int32(150)
	topGap := int32(8)
	reloadW, recoveryW := int32(80), int32(100)
	sourceBrowseW := int32(80)
	reloadX := w - 20 - reloadW
	recoveryX := reloadX - topGap - recoveryW
	sourceBrowseX := recoveryX - topGap - sourceBrowseW
	sourcePathX := int32(85) + aliasWTop + 10
	sourcePathW := sourceBrowseX - topGap - sourcePathX
	if sourcePathW < 180 {
		sourcePathW = 180
	}
	procMoveWindow.Call(app.sourceAliasCombo, 85, 13, uintptr(aliasWTop), 240, 1)
	procMoveWindow.Call(app.sourceEdit, uintptr(sourcePathX), 14, uintptr(sourcePathW), 25, 1)
	procMoveWindow.Call(app.sourceBtn, uintptr(sourceBrowseX), 13, uintptr(sourceBrowseW), 27, 1)
	procMoveWindow.Call(app.recoveryBtn, uintptr(recoveryX), 13, uintptr(recoveryW), 27, 1)
	procMoveWindow.Call(app.reloadBtn, uintptr(reloadX), 13, uintptr(reloadW), 27, 1)
	procMoveWindow.Call(app.sourceSuggestionList, uintptr(sourcePathX), 43, uintptr(sourcePathW), 140, 1)
	openFolderW := int32(110)
	openFolderX := w - 20 - openFolderW
	nameFilterX := int32(85)
	nameFilterW := openFolderX - 8 - nameFilterX
	if nameFilterW < 220 {
		nameFilterW = 220
	}
	procMoveWindow.Call(app.filterNameEdit, uintptr(nameFilterX), 49, uintptr(nameFilterW), 25, 1)
	procMoveWindow.Call(app.openFolderBtn, uintptr(openFolderX), 48, uintptr(openFolderW), 27, 1)
	procMoveWindow.Call(app.categoryCombo, 85, 82, 165, 260, 1)
	procMoveWindow.Call(app.extensionCombo, 340, 82, 150, 320, 1)
	procMoveWindow.Call(app.filterAgeCombo, 545, 82, 135, 220, 1)
	procMoveWindow.Call(app.filterSizeCombo, 735, 82, 135, 220, 1)
	clearW := int32(105)
	clearX := w - 20 - clearW
	sortW := int32(190)
	sortComboX := clearX - 8 - sortW
	sortLabelW := int32(50)
	sortLabelX := sortComboX - 5 - sortLabelW
	procMoveWindow.Call(app.filterSortLabel, uintptr(sortLabelX), 87, uintptr(sortLabelW), 20, 1)
	procMoveWindow.Call(app.filterSortCombo, uintptr(sortComboX), 82, uintptr(sortW), 220, 1)
	procMoveWindow.Call(app.filterClearBtn, uintptr(clearX), 81, uintptr(clearW), 28, 1)
	procMoveWindow.Call(app.countLabel, 20, 120, uintptr(listW), 24, 1)
	zoomGap := int32(4)
	zoomOutW, zoom100W, zoomInW, zoomFitW, zoomFitWidthW := int32(34), int32(56), int32(34), int32(44), int32(54)
	zoomToolbarW := zoomOutW + zoom100W + zoomInW + zoomFitW + zoomFitWidthW + 4*zoomGap
	zoomX := previewX + previewW - zoomToolbarW
	readTextW := int32(105)
	readTextX := zoomX - 8 - readTextW
	currentLabelW := readTextX - previewX - 8
	if currentLabelW < 90 {
		currentLabelW = 90
	}
	procMoveWindow.Call(app.currentLabel, uintptr(previewX), 120, uintptr(currentLabelW), 24, 1)
	procMoveWindow.Call(app.readTextCheck, uintptr(readTextX), 116, uintptr(readTextW), 27, 1)
	procMoveWindow.Call(app.zoomOutBtn, uintptr(zoomX), 116, uintptr(zoomOutW), 27, 1)
	zoomX += zoomOutW + zoomGap
	procMoveWindow.Call(app.zoom100Btn, uintptr(zoomX), 116, uintptr(zoom100W), 27, 1)
	zoomX += zoom100W + zoomGap
	procMoveWindow.Call(app.zoomInBtn, uintptr(zoomX), 116, uintptr(zoomInW), 27, 1)
	zoomX += zoomInW + zoomGap
	procMoveWindow.Call(app.zoomFitBtn, uintptr(zoomX), 116, uintptr(zoomFitW), 27, 1)
	zoomX += zoomFitW + zoomGap
	procMoveWindow.Call(app.zoomFitWidthBtn, uintptr(zoomX), 116, uintptr(zoomFitWidthW), 27, 1)
	undoW := int32(68)
	undoBatchW := int32(88)
	headerGap := int32(4)
	titleW := processedW - undoBatchW - undoW - (2 * headerGap)
	procMoveWindow.Call(app.processedTitleLabel, uintptr(processedX), 120, uintptr(titleW), 24, 1)
	procMoveWindow.Call(app.undoBatchBtn, uintptr(processedX+titleW+headerGap), 116, uintptr(undoBatchW), 27, 1)
	procMoveWindow.Call(app.undoBtn, uintptr(processedX+processedW-undoW), 116, uintptr(undoW), 27, 1)
	procMoveWindow.Call(app.fileList, 20, 148, uintptr(listW), uintptr(previewH), 1)
	procMoveWindow.Call(app.previewHwnd, uintptr(previewX), 148, uintptr(previewW), uintptr(previewH), 1)
	procMoveWindow.Call(app.processedSearchEdit, uintptr(processedX), 148, uintptr(processedW), 25, 1)
	processedListY := int32(181)
	processedListH := previewH - (processedListY - 148)
	if processedListH < 120 {
		processedListH = 120
	}
	procMoveWindow.Call(app.processedList, uintptr(processedX), uintptr(processedListY), uintptr(processedW), uintptr(processedListH), 1)
	y := int32(158) + previewH
	// Keep the lower form on a single alignment grid. Labels and controls move
	// together so DPI/window changes cannot make rows drift independently.
	labelX := int32(20)
	labelW := int32(70)
	fieldX := labelX + labelW + 5
	rowH := int32(28)
	procMoveWindow.Call(app.nameLabel, uintptr(labelX), uintptr(y+4), uintptr(labelW), 20, 1)
	renameApplyW := int32(110)
	renameComboW := int32(165)
	renameLabelW := int32(35)
	renameGap := int32(8)
	renameApplyX := w - 20 - renameApplyW
	renameComboX := renameApplyX - renameGap - renameComboW
	renameLabelX := renameComboX - 5 - renameLabelW
	nameW := renameLabelX - renameGap - fieldX
	if nameW < 220 {
		nameW = 220
	}
	procMoveWindow.Call(app.nameEdit, uintptr(fieldX), uintptr(y), uintptr(nameW), uintptr(rowH), 1)
	procMoveWindow.Call(app.renameToolLabel, uintptr(renameLabelX), uintptr(y+4), uintptr(renameLabelW), 20, 1)
	procMoveWindow.Call(app.renameToolCombo, uintptr(renameComboX), uintptr(y), uintptr(renameComboW), 220, 1)
	procMoveWindow.Call(app.renameToolBtn, uintptr(renameApplyX), uintptr(y-1), uintptr(renameApplyW), 29, 1)

	destY := y + 38
	aliasW := int32(185)
	pathLabelX := fieldX + aliasW + 15
	pathLabelW := int32(90)
	pathX := pathLabelX + pathLabelW + 5
	browseW := int32(105)
	openDestW := int32(140)
	rightGap := int32(8)
	openDestX := w - 20 - openDestW
	browseX := openDestX - rightGap - browseW
	pathW := browseX - rightGap - pathX
	if pathW < 180 {
		pathW = 180
	}
	procMoveWindow.Call(app.destinationLabel, uintptr(labelX), uintptr(destY+4), uintptr(labelW), 20, 1)
	procMoveWindow.Call(app.destAliasCombo, uintptr(fieldX), uintptr(destY), uintptr(aliasW), 260, 1)
	procMoveWindow.Call(app.pathLabel, uintptr(pathLabelX), uintptr(destY+4), uintptr(pathLabelW), 20, 1)
	procMoveWindow.Call(app.destEdit, uintptr(pathX), uintptr(destY), uintptr(pathW), uintptr(rowH), 1)
	procMoveWindow.Call(app.destBtn, uintptr(browseX), uintptr(destY-1), uintptr(browseW), 29, 1)
	procMoveWindow.Call(app.destOpenBtn, uintptr(openDestX), uintptr(destY-1), uintptr(openDestW), 29, 1)
	by := y + 85
	procMoveWindow.Call(app.renameBtn, 20, uintptr(by), 90, 34, 1)
	procMoveWindow.Call(app.moveBtn, 120, uintptr(by), 145, 34, 1)
	procMoveWindow.Call(app.nextBtn, 275, uintptr(by), 80, 34, 1)
	procMoveWindow.Call(app.openBtn, 365, uintptr(by), 80, 34, 1)
	procMoveWindow.Call(app.fileFlowBtn, 455, uintptr(by), 170, 34, 1)
	procMoveWindow.Call(app.statusLabel, 20, uintptr(by+43), uintptr(w-40), 20, 1)
	procMoveWindow.Call(app.keyboardHintLabel, 20, uintptr(by+64), uintptr(w-40), 18, 1)
	procMoveWindow.Call(app.destSuggestionList, uintptr(pathX), uintptr(destY+31), uintptr(pathW), 140, 1)
	// Moving overlapping child controls can leave copied pixels in a sibling or
	// transparent label. Invalidate the completed layout, including child frames.
	// Queue repaint only: never synchronously wait on the foreign PDF renderer.
	procRedrawWindow.Call(hwnd, 0, 0, 0x0001|0x0004|0x0080)
}

func populateFileList() {
	if app.fileList == 0 {
		return
	}
	app.updatingFileList = true
	defer func() { app.updatingFileList = false }()
	procSendMessageW.Call(app.fileList, LB_RESETCONTENT, 0, 0)
	for _, path := range app.files {
		procSendMessageW.Call(app.fileList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(filepath.Base(path)))))
	}
	syncFileListSelection()
}

func fileListSelectionCount() int {
	if app.fileList == 0 {
		return 0
	}
	count, _, _ := procSendMessageW.Call(app.fileList, LB_GETSELCOUNT, 0, 0)
	if count == LB_ERR {
		return 0
	}
	return int(count)
}

func selectedSourceFiles() []string {
	indexes := selectedSourceIndexes()
	selected := make([]string, 0, len(indexes))
	for _, raw := range indexes {
		selected = append(selected, app.files[int(raw)])
	}
	return selected
}

func selectAllVisibleSourceFiles() bool {
	if app.fileList == 0 || len(app.files) == 0 {
		return false
	}
	app.updatingFileList = true
	procSendMessageW.Call(app.fileList, LB_SETSEL, 1, ^uintptr(0))
	if app.index < 0 || app.index >= len(app.files) {
		app.index = 0
	}
	procSendMessageW.Call(app.fileList, LB_SETCARETINDEX, uintptr(app.index), 0)
	app.updatingFileList = false
	showCurrent(false)
	procSetFocus.Call(app.fileList)
	return true
}

func syncFileListSelection() {
	if app.fileList == 0 {
		return
	}
	app.updatingFileList = true
	defer func() { app.updatingFileList = false }()
	if app.index < 0 || app.index >= len(app.files) {
		procSendMessageW.Call(app.fileList, LB_SETSEL, 0, ^uintptr(0))
		return
	}
	if fileListSelectionCount() == 0 {
		procSendMessageW.Call(app.fileList, LB_SETSEL, 1, uintptr(app.index))
	}
	procSendMessageW.Call(app.fileList, LB_SETCARETINDEX, uintptr(app.index), 0)
}

func selectedProcessedViewRow() *processedViewRow {
	if app.processedList == 0 || len(app.processedRows) == 0 {
		return nil
	}
	idx, _, _ := procSendMessageW.Call(app.processedList, LB_GETCURSEL, 0, 0)
	if idx == LB_ERR || int(idx) < 0 || int(idx) >= len(app.processedRows) {
		return nil
	}
	return &app.processedRows[int(idx)]
}

func selectedProcessedIndex() int {
	row := selectedProcessedViewRow()
	if row == nil || row.Header || row.HistoryIndex < 0 || row.HistoryIndex >= len(app.history) {
		return -1
	}
	return row.HistoryIndex
}

func toggleProcessedGroup(key string) {
	key = strings.TrimSpace(key)
	if key == "" || strings.TrimSpace(getText(app.processedSearchEdit)) != "" {
		return
	}
	if app.processedCollapsed == nil {
		app.processedCollapsed = make(map[string]bool)
	}
	app.processedCollapsed[key] = !app.processedCollapsed[key]
	populateProcessedList(-1)
}

func populateProcessedList(preferredSelection int) {
	if app.processedList == 0 {
		return
	}
	app.updatingProcessedList = true
	defer func() { app.updatingProcessedList = false }()

	if app.processedCollapsed == nil {
		app.processedCollapsed = make(map[string]bool)
	}
	ensureProcessedGroupState(app.history, app.processedCollapsed)
	if preferredSelection >= 0 && preferredSelection < len(app.history) && strings.TrimSpace(getText(app.processedSearchEdit)) == "" {
		key := processedDateKey(app.history[preferredSelection], time.Now())
		app.processedCollapsed[key] = false
	}
	query := strings.TrimSpace(getText(app.processedSearchEdit))
	app.processedRows = buildProcessedView(app.history, query, app.processedCollapsed, time.Now())

	procSendMessageW.Call(app.processedList, LB_RESETCONTENT, 0, 0)
	visibleRecords := 0
	selectedRow := -1
	for rowIndex, row := range app.processedRows {
		procSendMessageW.Call(app.processedList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(row.Label))))
		if !row.Header {
			visibleRecords++
		}
		if row.HistoryIndex == preferredSelection {
			selectedRow = rowIndex
		}
	}
	if query != "" {
		setText(app.processedTitleLabel, fmt.Sprintf("PROCESSED  ·  %d / %d", visibleRecords, len(app.history)))
	} else {
		setText(app.processedTitleLabel, fmt.Sprintf("PROCESSED  ·  %d", len(app.history)))
	}

	if selectedRow >= 0 {
		procSendMessageW.Call(app.processedList, LB_SETCURSEL, uintptr(selectedRow), 0)
		procEnableWindow.Call(app.undoBtn, 1)
	} else {
		procSendMessageW.Call(app.processedList, LB_SETCURSEL, ^uintptr(0), 0)
		procEnableWindow.Call(app.undoBtn, 0)
	}
	if latestUndoableBatchID(app.history) != "" {
		procEnableWindow.Call(app.undoBatchBtn, 1)
	} else {
		procEnableWindow.Call(app.undoBatchBtn, 0)
	}
	if len(app.processedRows) > 0 {
		procEnableWindow.Call(app.processedList, 1)
	} else {
		procEnableWindow.Call(app.processedList, 0)
	}
}

func showSelectedProcessedStatus() {
	idx := selectedProcessedIndex()
	if idx < 0 {
		return
	}
	rec := app.history[idx]
	setText(app.statusLabel, "Processed: "+rec.From+"  →  "+rec.To+"   |   Alt+U to undo selected")
}

func recordProcessed(rec MoveRecord) {
	app.history = append(app.history, rec)
	if app.processedCollapsed == nil {
		app.processedCollapsed = make(map[string]bool)
	}
	app.processedCollapsed[processedDateKey(rec, time.Now())] = false
	if !app.batchMoveActive {
		populateProcessedList(len(app.history) - 1)
	}
	if err := observeCompletedMoveLearning(learningFilePath(), rec, time.Now()); err != nil {
		setText(app.statusLabel, "Move completed, but local learning could not be updated: "+err.Error())
	}
	refreshLearningCenterBadge()
}

func undoSelectedProcessed() {
	if len(app.history) == 0 {
		setText(app.statusLabel, "Nothing to undo.")
		return
	}
	idx := selectedProcessedIndex()
	if idx < 0 {
		setText(app.statusLabel, "Select a processed transaction to undo.")
		return
	}

	if !releaseSumatraForOperation() {
		return
	}
	remaining, rec, err := undoHistoryAt(app.history, idx)
	if err != nil {
		showError(err.Error())
		return
	}
	vaultErr := syncConfiguredVaultMove(rec.To, rec.From)
	if err := unobserveCompletedMoveLearning(learningFilePath(), rec, time.Now()); err != nil {
		// Undo is authoritative for files even if optional learning data cannot be updated.
		setText(app.statusLabel, "Undo completed; local learning correction could not be recorded: "+err.Error())
	}
	refreshLearningCenterBadge()
	journalErr := error(nil)
	if strings.TrimSpace(rec.JournalID) != "" {
		journalErr = markMoveJournalUndone(moveJournalFilePath(), rec.JournalID, time.Now())
	}
	app.history = remaining
	nextSelection := idx
	if nextSelection >= len(app.history) {
		nextSelection = len(app.history) - 1
	}
	populateProcessedList(nextSelection)
	// Undo changes both directories: refresh even when viewing the moved file
	// in its destination, where it has just been removed.
	if filepath.Clean(filepath.Dir(rec.From)) == filepath.Clean(app.sourceDir) ||
		filepath.Clean(filepath.Dir(rec.To)) == filepath.Clean(app.sourceDir) {
		reloadFiles()
	}
	if journalErr != nil {
		setText(app.statusLabel, "Undo completed, but durable journal reconciliation is pending.")
		showError("The Undo completed successfully, but DCM could not record the journal update:\n" + journalErr.Error() + "\n\nOn the next launch DCM will reconcile the journal against the filesystem.")
		return
	}
	if vaultErr != nil {
		setText(app.statusLabel, "Undo completed, but the Vault address index could not be updated: "+vaultErr.Error())
		return
	}
	setText(app.statusLabel, "Undid processed transaction: "+processedRecordLabel(rec))
}

func undoLatestBatch() {
	if app.batchMoveActive || app.backgroundMoveBusy {
		setText(app.statusLabel, "Wait for the current file operation to finish before undoing a batch.")
		return
	}
	batchID := latestUndoableBatchID(app.history)
	if batchID == "" {
		setText(app.statusLabel, "No completed batch is available to undo.")
		return
	}
	indexes := batchHistoryIndexes(app.history, batchID)
	if len(indexes) == 0 {
		setText(app.statusLabel, "No completed batch is available to undo.")
		return
	}
	msg := fmt.Sprintf("Undo the latest batch of %d processed file(s)?\n\nDCM will preflight every file before changing anything, then restore the batch newest-first.", len(indexes))
	result, _, _ := procMessageBoxW.Call(app.hwnd, uintptr(unsafe.Pointer(wstr(msg))), uintptr(unsafe.Pointer(wstr("Undo Latest Batch — DCM - Quick Sort"))), 0x04|0x20)
	if int(result) != 6 {
		setText(app.statusLabel, "Batch undo cancelled; no files were changed.")
		return
	}

	if !releaseSumatraForOperation() {
		return
	}
	remaining, undone, undoErr := undoHistoryBatch(app.history, batchID)
	app.history = remaining
	var reconciliationIssues []string
	reloadSource := false
	for _, rec := range undone {
		if filepath.Clean(filepath.Dir(rec.From)) == filepath.Clean(app.sourceDir) {
			reloadSource = true
		}
		if err := syncConfiguredVaultMove(rec.To, rec.From); err != nil {
			reconciliationIssues = append(reconciliationIssues, "Vault index: "+err.Error())
		}
		if err := unobserveCompletedMoveLearning(learningFilePath(), rec, time.Now()); err != nil {
			reconciliationIssues = append(reconciliationIssues, "learning correction: "+err.Error())
		}
		if strings.TrimSpace(rec.JournalID) != "" {
			if err := markMoveJournalUndone(moveJournalFilePath(), rec.JournalID, time.Now()); err != nil {
				reconciliationIssues = append(reconciliationIssues, "journal: "+err.Error())
			}
		}
	}
	refreshLearningCenterBadge()
	populateProcessedList(len(app.history) - 1)
	if reloadSource {
		reloadFiles()
	}
	if undoErr != nil {
		setText(app.statusLabel, fmt.Sprintf("Batch undo restored %d file(s) before stopping safely.", len(undone)))
		showError(undoErr.Error())
		return
	}
	if len(reconciliationIssues) > 0 {
		setText(app.statusLabel, fmt.Sprintf("Batch undo restored %d file(s); some metadata reconciliation needs attention.", len(undone)))
		showError("The files were restored, but DCM could not finish every metadata update:\n\n" + strings.Join(reconciliationIssues, "\n"))
		return
	}
	setText(app.statusLabel, fmt.Sprintf("Undid latest batch: %d file(s) restored.", len(undone)))
}

func recoverDurableMoveHistory() string {
	recovery, err := recoverMoveJournal(moveJournalFilePath(), time.Now())
	if err != nil {
		showError("DCM could not read its durable move journal:\n" + err.Error() + "\n\nExisting files were not changed. New standard moves will be blocked until the journal is readable again.")
		return "Durable move journal unavailable; new moves will fail closed."
	}
	app.history = append([]MoveRecord(nil), recovery.History...)
	if len(recovery.Attention) > 0 {
		var b strings.Builder
		b.WriteString("DCM found unfinished move state that needs review. No source file was auto-deleted.\n\n")
		limit := len(recovery.Attention)
		if limit > 3 {
			limit = 3
		}
		for i := 0; i < limit; i++ {
			entry := recovery.Attention[i]
			b.WriteString("• ")
			b.WriteString(entry.Record.From)
			b.WriteString("\n  → ")
			b.WriteString(entry.Record.To)
			if strings.TrimSpace(entry.Detail) != "" {
				b.WriteString("\n  ")
				b.WriteString(entry.Detail)
			}
			b.WriteString("\n\n")
		}
		if len(recovery.Attention) > limit {
			b.WriteString(fmt.Sprintf("…and %d more journal entries.\n", len(recovery.Attention)-limit))
		}
		_ = b.String() // Details are available non-destructively in Recovery Center.
	}
	parts := make([]string, 0, 4)
	if len(recovery.History) > 0 {
		parts = append(parts, fmt.Sprintf("restored %d undoable move(s)", len(recovery.History)))
	}
	if recovery.AutoFinalized > 0 {
		parts = append(parts, fmt.Sprintf("finalized %d interrupted move(s)", recovery.AutoFinalized))
	}
	if recovery.Interrupted > 0 {
		parts = append(parts, fmt.Sprintf("%d interrupted move(s) left source unchanged", recovery.Interrupted))
	}
	if len(recovery.Attention) > 0 {
		parts = append(parts, fmt.Sprintf("%d move(s) need review", len(recovery.Attention)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Journal recovery: " + strings.Join(parts, " · ")
}

func refreshSuggestionForCurrent() {
	source := currentFile()
	app.currentSuggestion = LearnedSuggestion{}
	setSuggestionButtonState := func(text string, enabled bool) {
		if app.suggestionBtn == 0 {
			return
		}
		setText(app.suggestionBtn, text)
		flag := uintptr(0)
		if enabled {
			flag = 1
		}
		procEnableWindow.Call(app.suggestionBtn, flag)
	}
	if source == "" {
		setSuggestionButtonState("Suggestion", false)
		return
	}
	if strings.EqualFold(strings.TrimSpace(app.suggestionDismissedFor), strings.TrimSpace(source)) {
		setSuggestionButtonState("Suggestion off", false)
		return
	}
	suggestion, err := loadLearnedSuggestion(learningFilePath(), source)
	if err != nil {
		setSuggestionButtonState("Suggestion !", false)
		return
	}
	app.currentSuggestion = suggestion
	if !suggestionAvailable(suggestion) {
		setSuggestionButtonState("Suggestion", false)
		return
	}
	setSuggestionButtonState(fmt.Sprintf("Suggestion %.0f%%", suggestionSummaryConfidence(suggestion)*100), true)
	if suggestionCenter.hwnd != 0 {
		refreshSuggestionCenter()
	}
}

func queueSelectedForFileFlow() {
	idx := selectedProcessedIndex()
	if idx < 0 || idx >= len(app.history) {
		setText(app.statusLabel, "Select a completed transaction in Processed before handing it to FileFlow.")
		return
	}
	rec := app.history[idx]
	added, err := queueFileFlowHandoff(fileFlowHandoffQueuePath(), rec, time.Now())
	if err != nil {
		showError("Could not queue this file for FileFlow:\n" + err.Error())
		return
	}
	if !added {
		setText(app.statusLabel, "This completed transaction is already queued for FileFlow: "+rec.To)
		return
	}
	setText(app.statusLabel, "Queued for FileFlow without copying the file: "+rec.To)
}

func refreshFileFlowMetadataControls() {
	source := currentFile()
	moveLabel := "Move to &Vault"
	selectedCount := fileListSelectionCount()
	if selectedCount > 1 {
		moveLabel = fmt.Sprintf("Move %d to &Vault", selectedCount)
	} else if source != "" {
		if draft, ok, err := loadFileFlowDraft(fileFlowDraftStorePath(), source); err == nil && ok && !draft.Empty() {
			moveLabel = "Update + Move to &Vault"
		}
	}
	if app.moveBtn != 0 {
		setText(app.moveBtn, moveLabel)
		procInvalidateRect.Call(app.moveBtn, 0, 1)
	}
}

func commitStagedFileFlowMetadata(source, target string, rec MoveRecord) {
	draft, ok, err := loadFileFlowDraft(fileFlowDraftStorePath(), source)
	if err != nil {
		showError("The file operation succeeded, but DCM could not read its staged FileFlow metadata:\n" + err.Error())
		return
	}
	if !ok || draft.Empty() {
		return
	}
	now := time.Now()
	queuePath := fileFlowHandoffQueuePath()
	added, queueErr := queueFileFlowMetadataUpdate(queuePath, rec, draft, now)
	if queueErr != nil {
		_ = migrateFileFlowDraft(fileFlowDraftStorePath(), source, target)
		showError("The file moved successfully, but its FileFlow metadata could not be queued.\n\nThe staged metadata was retained against the new path and can be retried later.\n\n" + queueErr.Error())
		return
	}
	if err := clearFileFlowDraft(fileFlowDraftStorePath(), source); err != nil {
		showError("The FileFlow update was queued, but DCM could not clear its local staged copy:\n" + err.Error())
		return
	}
	if added {
		eventID := fileFlowMetadataUpdateID(rec)
		if err := notifyFileFlowService(queuePath, eventID, now); err != nil {
			setText(app.statusLabel, "FileFlow metadata queued durably; Flowverse service will pick it up when available: "+target)
		} else {
			setText(app.statusLabel, "FileFlow metadata queued and Flowverse service notified: "+target)
		}
	}
}

func initDefaultFolders() {
	home, _ := os.UserHomeDir()
	source := loadDefaultSourceSetting()
	if source == "" {
		source = filepath.Join(home, "Downloads")
		if st, err := os.Stat(source); err != nil || !st.IsDir() {
			source = home
		}
	}
	app.sourceDir = source
	setSourceText(source)
	// v1.0.69+ destinations are Vault logical locations only. Never seed the
	// destination field with Documents/Home or another arbitrary disk folder.
	setDestinationText("")
}

func applyStartupArgument() {
	if len(os.Args) < 2 {
		return
	}
	selection, err := resolveStartupSelection(os.Args[1])
	if err != nil {
		setText(app.statusLabel, "Explorer startup path could not be opened: "+err.Error())
		return
	}
	if selection.SourceDir == "" {
		return
	}
	app.sourceDir = selection.SourceDir
	app.startupPreferredFile = selection.PreferredFile
	setSourceText(selection.SourceDir)
}

func currentFile() string {
	if app.index < 0 || app.index >= len(app.files) {
		return ""
	}
	return app.files[app.index]
}

func reloadFiles() {
	reloadFilesWithFocus(true)
}

func reloadFilesWithFocus(focusName bool) {
	previous := currentFile()
	if previous == "" && app.startupPreferredFile != "" {
		previous = app.startupPreferredFile
	}
	files, err := listFilesInbox(app.sourceDir, selectedInboxFilter(), time.Now())
	if err != nil {
		// A failed refresh must not leave cached paths as actionable targets.
		app.files = nil
		app.index = -1
		app.startupPreferredFile = ""
		populateFileList()
		showCurrent(false)
		setText(app.currentLabel, "PREVIEW  ·  Source unavailable")
		app.fallback = "Cannot read source folder: " + err.Error() + "\n\nRestore access or choose another folder, then Reload."
		setText(app.statusLabel, "Cannot read source folder: "+err.Error())
		procInvalidateRect.Call(app.previewHwnd, 0, 1)
		return
	}
	app.files = files
	app.index = 0
	if idx := preferredFileIndex(files, previous); idx >= 0 {
		app.index = idx
	}
	app.startupPreferredFile = ""
	populateFileList()
	showCurrent(focusName)
}

func showCurrent(focusName bool) {
	showCurrentView(focusName, true)
}

// Native selection notifications must not move the caret/anchor or reselect
// a row the user just Ctrl-deselected. Programmatic navigation still syncs.
func showCurrentView(focusName, syncSelection bool) {
	unloadPreview()
	f := currentFile()
	if f == "" {
		updateReadTextAvailability("")
		heading, hint := inboxEmptyState(selectedInboxFilter(), len(app.files))
		setText(app.currentLabel, "PREVIEW  ·  "+heading)
		setText(app.countLabel, "FILES  ·  0")
		setText(app.nameEdit, "")
		app.fallback = hint
		refreshSuggestionForCurrent()
		refreshFileFlowMetadataControls()
		procInvalidateRect.Call(app.previewHwnd, 0, 1)
		return
	}
	base := filepath.Base(f)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	setText(app.currentLabel, "PREVIEW  ·  "+base)
	if syncSelection {
		syncFileListSelection()
	}
	selectionCount := fileListSelectionCount()
	if selectionCount > 1 {
		setText(app.countLabel, fmt.Sprintf("FILES  ·  %d selected  ·  %s", selectionCount, filePositionLabel(app.index, len(app.files))))
	} else {
		setText(app.countLabel, "FILES  ·  "+filePositionLabel(app.index, len(app.files)))
	}
	setText(app.nameEdit, stem)
	procSendMessageW.Call(app.nameEdit, 0x00B1, 0, ^uintptr(0)) // EM_SETSEL all
	updateReadTextAvailability(f)
	loadPreview(f)
	refreshSuggestionForCurrent()
	refreshFileFlowMetadataControls()
	if focusName {
		procSetFocus.Call(app.nameEdit)
	}
}

func nextFile() {
	if app.backgroundMoveBusy {
		setText(app.statusLabel, "Finish the current file operation before moving to the next file.")
		return
	}
	if currentFile() == "" {
		return
	}
	app.index++
	// Next is single-file navigation: discard the old selection before
	// showCurrent selects the new row (or leaves none at end of list).
	app.updatingFileList = true
	procSendMessageW.Call(app.fileList, LB_SETSEL, 0, ^uintptr(0))
	app.updatingFileList = false
	showCurrent(false)
	focusFileList()
}

func currentOriginalStem() string {
	f := currentFile()
	if f == "" {
		return ""
	}
	base := filepath.Base(f)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func applySelectedRenameTool() {
	if currentFile() == "" {
		setText(app.statusLabel, "No file is selected for renaming.")
		return
	}
	tool := strings.TrimSpace(getText(app.renameToolCombo))
	if tool == "" {
		tool = RenameToolClean
	}
	st, statErr := os.Stat(currentFile())
	if statErr != nil {
		showError("Could not read the selected file date:\n" + statErr.Error())
		return
	}
	ctx := RenameToolContext{FileDate: st.ModTime()}
	templateName := ""
	if tool == RenameToolTemplate {
		tpl, tplErr := activeNamingTemplate()
		if tplErr != nil {
			showError("Could not load the active naming template:\n" + tplErr.Error())
			return
		}
		ctx.Template = tpl.Pattern
		ctx.Variables = namingVariableMap(tpl.Variables)
		templateName = tpl.Name
	}
	result, err := applyRenameToolWithContext(getText(app.nameEdit), currentOriginalStem(), tool, ctx)
	if err != nil {
		showError("Could not apply rename tool:\n" + err.Error())
		return
	}
	setText(app.nameEdit, result)
	procSendMessageW.Call(app.nameEdit, 0x00B1, 0, ^uintptr(0)) // EM_SETSEL all
	procSetFocus.Call(app.nameEdit)
	if templateName != "" {
		setText(app.statusLabel, "Applied naming template "+templateName+" to the Name field. The file is unchanged until Rename.")
	} else {
		setText(app.statusLabel, "Applied "+tool+" to the Name field. The file is unchanged until Rename.")
	}
}

func removeSourceAfterAdvance(source, target string, expected sourceSnapshot) error {
	var err error
	for attempt := 0; attempt < 8; attempt++ {
		err = retireVerifiedSource(source, target, expected)
		if err == nil {
			return nil
		}
		if !errors.Is(err, syscall.Errno(32)) && !errors.Is(err, syscall.Errno(33)) {
			return err
		}
		if attempt < 7 {
			time.Sleep(60 * time.Millisecond)
		}
	}
	return err
}

func beginFileOperation(mode fileOperationMode, source, target string) {
	if !releaseSumatraForOperation() {
		return
	}
	app.pendingFileOperation = mode
	// Release the active preview before hashing, deleting, renaming or copying.
	// Some third-party preview handlers keep sharing locks while loaded.
	unloadPreview()
	beginBackgroundConflictInspection(source, target)
}

func renameCurrent() {
	if app.backgroundMoveBusy {
		setText(app.statusLabel, "A background move is already running.")
		return
	}
	src := currentFile()
	if src == "" {
		return
	}
	name := strings.TrimSpace(getText(app.nameEdit))
	if name == "" {
		showError("Enter a filename first.")
		return
	}
	target, err := targetFor(src, filepath.Dir(src), name)
	if err != nil {
		showError(err.Error())
		return
	}
	if filepath.Clean(target) == filepath.Clean(src) {
		setText(app.statusLabel, "Name unchanged; no rename was needed.")
		return
	}
	beginFileOperation(fileOperationRename, src, target)
}

func promptBatchMove(plan []batchMovePlanItem, logicalDestination string) bool {
	if len(plan) < 2 {
		return true
	}
	physical := filepath.Dir(plan[0].Target)
	msg := fmt.Sprintf("Move %d selected files into the Vault?\n\nLogical location:\n%s\n\nPhysical location:\n%s\n\nEach file keeps its current filename. Conflicts are handled one file at a time and every successful move remains individually undoable.", len(plan), logicalDestination, physical)
	result, _, _ := procMessageBoxW.Call(app.hwnd, uintptr(unsafe.Pointer(wstr(msg))), uintptr(unsafe.Pointer(wstr("Move Selected Files to Vault — DCM - Quick Sort"))), 0x04|0x20)
	return int(result) == 6 // IDYES
}

func startBatchMove(selected []string, dest, logicalDestination string) {
	plan, err := buildBatchMovePlan(selected, dest)
	if err != nil {
		showError(err.Error())
		return
	}
	if len(plan) < 2 {
		return
	}
	if !promptBatchMove(plan, logicalDestination) {
		setText(app.statusLabel, "Batch move cancelled; no files were changed.")
		return
	}
	app.destDir = logicalDestination
	setDestinationText(logicalDestination)
	app.batchMoveActive = true
	app.batchMoveID = fmt.Sprintf("batch-%d", time.Now().UTC().UnixNano())
	app.batchMovePlan = append(app.batchMovePlan[:0], plan...)
	app.batchMoveTotal = len(plan)
	app.batchMoveCompleted = 0
	app.batchMoveIssues = 0
	app.batchProcessedStart = len(app.history)
	setText(app.statusLabel, fmt.Sprintf("Batch move ready: %d selected files → Vault/%s", len(plan), logicalDestination))
	procPostMessageW.Call(app.hwnd, WM_BATCH_MOVE_NEXT, 0, 0)
}

func advanceBatchMove() {
	if !app.batchMoveActive || app.backgroundMoveBusy {
		return
	}
	for len(app.batchMovePlan) > 0 {
		item := app.batchMovePlan[0]
		app.batchMovePlan = app.batchMovePlan[1:]
		if _, err := os.Stat(item.Source); err != nil {
			app.batchMoveCompleted++
			app.batchMoveIssues++
			continue
		}
		setText(app.statusLabel, fmt.Sprintf("Batch move %d/%d: %s", app.batchMoveCompleted+1, app.batchMoveTotal, filepath.Base(item.Source)))
		beginFileOperation(fileOperationMove, item.Source, item.Target)
		return
	}
	finishBatchMove()
}

func batchMoveStepFinished(success bool) {
	if !app.batchMoveActive {
		return
	}
	app.batchMoveCompleted++
	if !success {
		app.batchMoveIssues++
	}
	procPostMessageW.Call(app.hwnd, WM_BATCH_MOVE_NEXT, 0, 0)
}

func finishBatchMove() {
	if !app.batchMoveActive {
		return
	}
	total := app.batchMoveTotal
	issues := app.batchMoveIssues
	moved := app.batchMoveCompleted - issues
	preferredProcessed := -1
	if len(app.history) > app.batchProcessedStart {
		preferredProcessed = len(app.history) - 1
	}
	app.batchMoveActive = false
	app.batchMovePlan = nil
	app.batchMoveTotal = 0
	app.batchMoveCompleted = 0
	app.batchMoveIssues = 0
	app.batchProcessedStart = 0
	app.batchMoveID = ""
	reloadFilesWithFocus(false)
	populateProcessedList(preferredProcessed)
	focusFileList()
	if issues == 0 {
		setText(app.statusLabel, fmt.Sprintf("Batch move complete: %d of %d files moved. Use Undo Batch to restore the whole batch, or Undo for one transaction.", moved, total))
	} else {
		setText(app.statusLabel, fmt.Sprintf("Batch move finished: %d moved; %d skipped or failed out of %d. Review Processed and Recovery for details.", moved, issues, total))
	}
}

func moveCurrent() {
	if app.batchMoveActive {
		setText(app.statusLabel, "A batch move is already running.")
		return
	}
	if app.backgroundMoveBusy {
		setText(app.statusLabel, "A background move is already running.")
		return
	}
	root := loadVaultRootSetting()
	physicalDest, logicalDest, err := resolveExistingVaultDestination(root, strings.TrimSpace(getText(app.destEdit)))
	if err != nil {
		showError("Choose an existing logical location inside the configured Vault.\n\n" + err.Error())
		return
	}
	selected := selectedSourceFiles()
	if len(selected) == 0 {
		return
	}
	if len(selected) > 1 {
		startBatchMove(selected, physicalDest, logicalDest)
		return
	}
	// Move consumes the actual selected set, never an unselected caret row.
	src := selected[0]
	app.destDir = logicalDest
	setDestinationText(logicalDest)
	currentName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	target, err := targetFor(src, physicalDest, currentName)
	if err != nil {
		showError(err.Error())
		return
	}
	if filepath.Clean(target) == filepath.Clean(src) {
		showError("This file is already in the selected Vault location. Choose another Vault location or use Rename.")
		return
	}
	beginFileOperation(fileOperationMove, src, target)
}

func updateOpenDestinationAvailability() {
	if app.destOpenBtn == 0 {
		return
	}
	_, _, err := resolveExistingVaultDestination(loadVaultRootSetting(), getText(app.destEdit))
	if err == nil {
		procEnableWindow.Call(app.destOpenBtn, 1)
	} else {
		procEnableWindow.Call(app.destOpenBtn, 0)
	}
}

func openDestinationFolder() {
	path, logical, err := resolveExistingVaultDestination(loadVaultRootSetting(), getText(app.destEdit))
	if err != nil {
		updateOpenDestinationAvailability()
		showError("Vault location is unavailable.\n\n" + err.Error())
		return
	}
	shellOpen(path)
	setText(app.statusLabel, "Opened Vault location: "+logical)
}

func chooseVaultRoot() {
	initial := loadVaultRootSetting()
	if initial == "" {
		if st, err := os.Stat(`D:\`); err == nil && st.IsDir() {
			initial = `D:\`
		} else {
			initial = app.sourceDir
		}
	}
	root := browseFolder(app.hwnd, "Choose Flowverse Vault / Library", initial)
	if root == "" {
		return
	}
	if err := saveVaultRootSetting(root, app.sourceDir); err != nil {
		showError("Could not save Vault Library folder:\n" + err.Error())
		return
	}
	setText(app.statusLabel, "Indexing Vault Library: "+root)
	if err := rebuildVaultAddressIndex(root); err != nil {
		showError("Vault Library was saved, but its address index could not be built:\n" + err.Error())
		return
	}
	current := strings.TrimSpace(getText(app.destEdit))
	if _, _, err := resolveExistingVaultDestination(root, current); err != nil {
		setDestinationText("")
	}
	reloadDestinationAliases()
	if destManager.hwnd != 0 {
		setText(destManager.searchEdit, "")
		refreshDestinationManagerVault(current)
	}
	idx, err := loadVaultAddressIndex(root)
	if err != nil {
		setText(app.statusLabel, "Vault Library configured: "+root)
		return
	}
	setText(app.statusLabel, fmt.Sprintf("Vault Library: %s · %d logical addresses indexed", root, len(idx.Entries)))
}

func openCurrent() {
	if f := currentFile(); f != "" {
		shellOpen(f)
	}
}
func shellOpen(path string) {
	verb := wstr("open")
	p := wstr(path)
	procShellExecuteW.Call(app.hwnd, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(p)), 0, 0, SW_SHOWNORMAL)
}

func browseFolder(owner uintptr, title, initial string) string {
	var clsidFileOpenDialog GUID
	var iidFileOpenDialog GUID
	if !guidFromString("{DC1C5A9C-E88A-4DDE-A5A1-60F82A20AEF7}", &clsidFileOpenDialog) ||
		!guidFromString("{D57C7288-D4AD-4768-BE02-9D969532D960}", &iidFileOpenDialog) {
		return browseFolderLegacy(owner, title)
	}

	var dialog *iFileOpenDialog
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)),
		0,
		CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(&iidFileOpenDialog)),
		uintptr(unsafe.Pointer(&dialog)),
	)
	if hresultFailed(hr) || dialog == nil {
		return browseFolderLegacy(owner, title)
	}
	self := uintptr(unsafe.Pointer(dialog))
	defer syscall.SyscallN(dialog.Vtbl.Release, self)

	var options uint32
	hr, _, _ = syscall.SyscallN(dialog.Vtbl.GetOptions, self, uintptr(unsafe.Pointer(&options)))
	if !hresultFailed(hr) {
		options |= FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM | FOS_PATHMUSTEXIST
		syscall.SyscallN(dialog.Vtbl.SetOptions, self, uintptr(options))
	}
	if title != "" {
		syscall.SyscallN(dialog.Vtbl.SetTitle, self, uintptr(unsafe.Pointer(wstr(title))))
	}

	initial = strings.TrimSpace(initial)
	if initial != "" {
		if info, err := os.Stat(initial); err == nil && info.IsDir() {
			if item := shellItemFromPath(initial); item != nil {
				syscall.SyscallN(dialog.Vtbl.SetFolder, self, uintptr(unsafe.Pointer(item)))
				releaseShellItem(item)
			}
		}
	}

	hr, _, _ = syscall.SyscallN(dialog.Vtbl.Show, self, owner)
	if hresultFailed(hr) {
		return ""
	}

	var item *iShellItem
	hr, _, _ = syscall.SyscallN(dialog.Vtbl.GetResult, self, uintptr(unsafe.Pointer(&item)))
	if hresultFailed(hr) || item == nil {
		return ""
	}
	defer releaseShellItem(item)

	var raw *uint16
	hr, _, _ = syscall.SyscallN(item.Vtbl.GetDisplayName, uintptr(unsafe.Pointer(item)), SIGDN_FILESYSPATH, uintptr(unsafe.Pointer(&raw)))
	if hresultFailed(hr) || raw == nil {
		return ""
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(raw)))
	return utf16PtrString(raw)
}

func browseConfigurationFile(owner uintptr) string {
	var clsidFileOpenDialog GUID
	var iidFileOpenDialog GUID
	if !guidFromString("{DC1C5A9C-E88A-4DDE-A5A1-60F82A20AEF7}", &clsidFileOpenDialog) ||
		!guidFromString("{D57C7288-D4AD-4768-BE02-9D969532D960}", &iidFileOpenDialog) {
		return ""
	}
	var dialog *iFileOpenDialog
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidFileOpenDialog)), 0, CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(&iidFileOpenDialog)), uintptr(unsafe.Pointer(&dialog)))
	if hresultFailed(hr) || dialog == nil {
		return ""
	}
	self := uintptr(unsafe.Pointer(dialog))
	defer syscall.SyscallN(dialog.Vtbl.Release, self)

	var options uint32
	if hr, _, _ = syscall.SyscallN(dialog.Vtbl.GetOptions, self, uintptr(unsafe.Pointer(&options))); !hresultFailed(hr) {
		options |= FOS_FORCEFILESYSTEM | FOS_PATHMUSTEXIST | FOS_FILEMUSTEXIST
		syscall.SyscallN(dialog.Vtbl.SetOptions, self, uintptr(options))
	}
	syscall.SyscallN(dialog.Vtbl.SetTitle, self, uintptr(unsafe.Pointer(wstr("Import DCM configuration"))))
	filters := []COMDLG_FILTERSPEC{
		{PszName: wstr("DCM configuration"), PszSpec: wstr("*.dcmconf;*.conf;*.json")},
		{PszName: wstr("DCM export (*.dcmconf)"), PszSpec: wstr("*.dcmconf")},
		{PszName: wstr("Destination aliases (*.conf)"), PszSpec: wstr("*.conf")},
		{PszName: wstr("Settings (*.json)"), PszSpec: wstr("*.json")},
	}
	syscall.SyscallN(dialog.Vtbl.SetFileTypes, self, uintptr(len(filters)), uintptr(unsafe.Pointer(&filters[0])))
	syscall.SyscallN(dialog.Vtbl.SetFileTypeIndex, self, 1)
	if root := configurationBackupRoot(); root != "" {
		initial := filepath.Join(root, "exports")
		if st, err := os.Stat(initial); err == nil && st.IsDir() {
			if item := shellItemFromPath(initial); item != nil {
				syscall.SyscallN(dialog.Vtbl.SetFolder, self, uintptr(unsafe.Pointer(item)))
				releaseShellItem(item)
			}
		}
	}
	hr, _, _ = syscall.SyscallN(dialog.Vtbl.Show, self, owner)
	if hresultFailed(hr) {
		return ""
	}
	var item *iShellItem
	hr, _, _ = syscall.SyscallN(dialog.Vtbl.GetResult, self, uintptr(unsafe.Pointer(&item)))
	if hresultFailed(hr) || item == nil {
		return ""
	}
	defer releaseShellItem(item)
	var raw *uint16
	hr, _, _ = syscall.SyscallN(item.Vtbl.GetDisplayName, uintptr(unsafe.Pointer(item)), SIGDN_FILESYSPATH, uintptr(unsafe.Pointer(&raw)))
	if hresultFailed(hr) || raw == nil {
		return ""
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(raw)))
	return utf16PtrString(raw)
}

func shellItemFromPath(path string) *iShellItem {
	var iidShellItem GUID
	if !guidFromString("{43826D1E-E718-42EE-BC55-A1E261C37BFE}", &iidShellItem) {
		return nil
	}
	var item *iShellItem
	hr, _, _ := procSHCreateItemFromParsingName.Call(
		uintptr(unsafe.Pointer(wstr(path))),
		0,
		uintptr(unsafe.Pointer(&iidShellItem)),
		uintptr(unsafe.Pointer(&item)),
	)
	if hresultFailed(hr) {
		return nil
	}
	return item
}

func releaseShellItem(item *iShellItem) {
	if item != nil {
		syscall.SyscallN(item.Vtbl.Release, uintptr(unsafe.Pointer(item)))
	}
}

func utf16PtrString(p *uint16) string {
	if p == nil {
		return ""
	}
	buf := make([]uint16, 0, 260)
	for i := uintptr(0); ; i++ {
		v := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + i*2))
		if v == 0 {
			break
		}
		buf = append(buf, v)
	}
	return syscall.UTF16ToString(buf)
}

func browseFolderLegacy(owner uintptr, title string) string {
	display := make([]uint16, 260)
	bi := BROWSEINFO{HwndOwner: owner, PszDisplayName: &display[0], LpszTitle: wstr(title), UlFlags: BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	path := make([]uint16, 32768)
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(path)
}

func isImage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".bmp", ".gif", ".tif", ".tiff":
		return true
	}
	return false
}

func loadPreview(path string) {
	hideTextPreview()
	hideXLSXPreview()
	hidePPTXPreview()
	// Every newly opened file starts from the same predictable viewport state.
	// Manual zoom/pan from the previous file must never leak into the next one.
	resetPreviewZoomState()
	if shouldUsePlainTextPreview(path) {
		if showTextPreview(path) {
			setText(app.statusLabel, "Plain-text preview: "+strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), ".")))
		}
		// Preserve the specific text-read/encoding error; generic handler
		// fallback must not overwrite it or retain a previous success status.
		return
	}
	if readTextEnabled() {
		if showTextPreview(path) {
			setText(app.statusLabel, "Read as text: "+strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), ".")))
		}
		return
	}
	if isPDFPath(path) && startSumatraPreview(path) {
		return
	}
	if isPDFPath(path) && startPDFPreview(path) {
		return
	}
	if isImage(path) {
		if err := loadImageWithError(path); err != nil {
			app.fallback = "Could not preview this image.\n\n" + err.Error()
			setText(app.statusLabel, "Image preview unavailable: "+err.Error())
			updatePreviewZoomControls()
			procInvalidateRect.Call(app.previewHwnd, 0, 1)
			return
		}
		app.fallback = ""
		app.previewFit = true
		app.previewFitWidth = false
		app.previewZoom = imageFitZoomPercent()
		app.previewPanX, app.previewPanY = 0, 0
		updatePreviewZoomControls()
		procInvalidateRect.Call(app.previewHwnd, 0, 1)
		setText(app.statusLabel, fmt.Sprintf("Image preview: Fit · %d%%", app.previewZoom))
		return
	}

	hasHandler := previewHandlerCLSID(path) != ""
	if hasHandler && loadWindowsPreviewHandler(path) {
		app.fallback = ""
		app.previewActive = true
		return
	}

	// Prefer Windows' native Office preview when available. If it is missing or
	// fails, DCM renders the actual DOCX body locally from the Open XML package.
	// XLSX now has a built-in grid fallback with sheet switching; PPTX now has a built-in slide renderer with navigation, text, tables and embedded images.
	if text, stats, ok := docxPreviewText(path); ok {
		if showTextPreviewContentKind(text, "DOCX") {
			setText(app.statusLabel, fmt.Sprintf("DCM DOCX Preview: %d paragraphs · %d tables · %d images", stats.Paragraphs, stats.Tables, stats.Images))
			return
		}
	}
	if wb, ok := xlsxPreviewForPath(path); ok {
		if showXLSXPreview(wb) {
			return
		}
	}
	if deck, ok := pptxPreviewForPath(path); ok {
		if showPPTXPreview(deck) {
			return
		}
	}
	if text, ok := officeFoundationPreviewText(path); ok {
		if showTextPreviewContentKind(text, "Office") {
			setText(app.statusLabel, "DCM Office Preview: Open XML package validated")
			return
		}
	}

	app.fallback = previewFallbackText(path, hasHandler)
	procInvalidateRect.Call(app.previewHwnd, 0, 1)
}

func loadImage(path string) bool {
	// Retain the bool contract used by the owned PDF raster-loading path.
	return loadImageWithError(path) == nil
}

func loadImageWithError(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return errors.New("empty image file")
	}

	// Keep the decoder backed by an owned memory stream, never a source handle.
	hglobal, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE, uintptr(len(data)))
	if hglobal == 0 {
		return errors.New("could not allocate image memory")
	}
	mem, _, _ := procGlobalLock.Call(hglobal)
	if mem == 0 {
		procGlobalFree.Call(hglobal)
		return errors.New("could not access allocated image memory")
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(mem)), len(data)), data)
	procGlobalUnlock.Call(hglobal)

	var stream *iUnknown
	hr, _, _ := procCreateStreamOnHGlobal.Call(hglobal, 1, uintptr(unsafe.Pointer(&stream)))
	if hresultFailed(hr) || stream == nil {
		procGlobalFree.Call(hglobal)
		return fmt.Errorf("could not create image stream (HRESULT %#x)", hr)
	}

	var image uintptr
	status, _, _ := procGdipLoadImageStream.Call(uintptr(unsafe.Pointer(stream)), uintptr(unsafe.Pointer(&image)))
	if status != 0 || image == 0 {
		if image != 0 {
			procGdipDisposeImage.Call(image)
		}
		releaseUnknown(stream)
		return fmt.Errorf("image decoder could not load %q (GDI+ status %d)", filepath.Base(path), status)
	}
	app.image = image
	app.imageStream = stream
	return nil
}

func unloadPreview() {
	if err := stopSumatraPreview(); err != nil {
		sumatraError(err)
	}
	stopPDFPreview()
	hideTextPreview()
	hideXLSXPreview()
	hidePPTXPreview()
	if app.preview != nil {
		self := uintptr(unsafe.Pointer(app.preview))
		syscall.SyscallN(app.preview.Vtbl.Unload, self)
		procIUnknownSetSite.Call(self, 0)
		syscall.SyscallN(app.preview.Vtbl.Release, self)
		app.preview = nil
	}
	app.previewActive = false
	releasePreviewImage()
	resetPreviewZoomState()
}

func previewWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_SIZE:
		resizeSumatraPreview()
		resizePreviewHandler()
		resizeTextPreview()
		resizeXLSXPreview()
		resizePPTXPreview()
		layoutPDFPreviewControls()
		procInvalidateRect.Call(hwnd, 0, 1)
		return 0
	case WM_COMMAND:
		id := int(loword(wParam))
		notify := hiword(wParam)
		if notify == BN_CLICKED {
			switch id {
			case ID_PDF_PREV:
				goToPDFPage(-1)
				return 0
			case ID_PDF_NEXT:
				goToPDFPage(1)
				return 0
			}
		}
		if id == ID_XLSX_SHEET && notify == CBN_SELCHANGE && app.xlsxPreviewActive {
			idx, _, _ := procSendMessageW.Call(app.xlsxSheetCombo, CB_GETCURSEL, 0, 0)
			if idx != ^uintptr(0) {
				showXLSXSheet(int(idx))
			}
			return 0
		}
		if id == ID_PPTX_SLIDE && notify == CBN_SELCHANGE && app.pptxPreviewActive {
			idx, _, _ := procSendMessageW.Call(app.pptxSlideCombo, CB_GETCURSEL, 0, 0)
			if idx != ^uintptr(0) {
				showPPTXSlide(int(idx))
			}
			return 0
		}
	case WM_MOUSEWHEEL:
		delta := int(int16(hiword(wParam)))
		if keyDown(VK_CONTROL) && previewSupportsZoom() {
			if detents := consumePreviewWheelDetents(&app.previewZoomWheelRemainder, delta); detents != 0 {
				adjustPreviewZoom(detents)
			}
			return 0
		}
		if handlePreviewWheelPan(delta, keyDown(VK_SHIFT)) {
			return 0
		}
	case WM_LBUTTONDOWN:
		if (app.image != 0 || app.pptxPreviewActive) && !app.previewFit {
			app.previewDragging = true
			app.previewDragX, app.previewDragY = signedLoword(lParam), signedHiword(lParam)
			app.previewPanStartX, app.previewPanStartY = app.previewPanX, app.previewPanY
			procSetCapture.Call(hwnd)
			return 0
		}
	case WM_MOUSEMOVE:
		if app.previewDragging && wParam&MK_LBUTTON != 0 {
			x, y := signedLoword(lParam), signedHiword(lParam)
			app.previewPanX = app.previewPanStartX + (x - app.previewDragX)
			app.previewPanY = app.previewPanStartY + (y - app.previewDragY)
			procInvalidateRect.Call(hwnd, 0, 1)
			return 0
		}
	case WM_LBUTTONUP:
		if app.previewDragging {
			app.previewDragging = false
			procReleaseCapture.Call()
			return 0
		}
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		var rc RECT
		procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		brush, _, _ := procGetStockObject.Call(WHITE_BRUSH)
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), brush)
		if app.image != 0 {
			drawImagePreview(hdc, rc)
		} else if app.pptxPreviewActive {
			drawPPTXPreview(hdc, rc)
		} else if !app.previewActive && !app.textPreviewActive && !app.xlsxPreviewActive && !app.pptxPreviewActive {
			text := app.fallback
			if text == "" {
				text = "Preview"
			}
			procSetBkMode.Call(hdc, 1)
			procSetTextColor.Call(hdc, 0x00606060)
			inner := RECT{40, 40, rc.Right - 40, rc.Bottom - 40}
			textPtr := wstr(text)
			// Multi-line centered fallback. First calculate its height, then center it vertically.
			calc := inner
			const drawFlags = 0x00000001 | 0x00000010 | 0x00000800 // DT_CENTER | DT_WORDBREAK | DT_NOPREFIX
			const dtCalcRect = 0x00000400
			procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(textPtr)), ^uintptr(0), uintptr(unsafe.Pointer(&calc)), drawFlags|dtCalcRect)
			textH := calc.Bottom - calc.Top
			if textH > 0 && textH < rc.Bottom-80 {
				inner.Top = (rc.Bottom - textH) / 2
				inner.Bottom = inner.Top + textH + 8
			}
			procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(textPtr)), ^uintptr(0), uintptr(unsafe.Pointer(&inner)), drawFlags)
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func drawImagePreview(hdc uintptr, rc RECT) {
	if app.image == 0 {
		return
	}
	var iw, ih uint32
	procGdipGetImageWidth.Call(app.image, uintptr(unsafe.Pointer(&iw)))
	procGdipGetImageHeight.Call(app.image, uintptr(unsafe.Pointer(&ih)))
	if iw == 0 || ih == 0 {
		return
	}
	// Fit is the default. Manual zoom uses actual image pixels scaled by the
	// selected percentage. Large zoomed images can be dragged inside the pane.
	rc = pdfContentRect(rc)
	availW, availH := rc.Right-rc.Left-8, rc.Bottom-rc.Top-8
	var dw, dh int32
	if pdfPreview.active {
		// Every owned PDF mode (Fit, Fit Width, and manual zoom) is rendered
		// by Windows.Data.Pdf at its final pixel dimensions. Draw those pixels
		// 1:1; scaling them again here would make wheel zoom nonlinear.
		dw, dh = int32(iw), int32(ih)
	} else if app.previewFit {
		dw, dh = fitDimensions(int32(iw), int32(ih), availW, availH)
	} else if app.previewFitWidth {
		dw, dh = fitWidthDimensions(int32(iw), int32(ih), availW)
	} else {
		dw, dh = zoomDimensions(int32(iw), int32(ih), app.previewZoom)
	}
	if dw <= 0 || dh <= 0 {
		return
	}
	app.previewPanX = clampPreviewPan(dw, availW, app.previewPanX)
	app.previewPanY = clampPreviewPan(dh, availH, app.previewPanY)
	x := rc.Left + (rc.Right-rc.Left-dw)/2 + app.previewPanX
	y := rc.Top + (rc.Bottom-rc.Top-dh)/2 + app.previewPanY
	var graphics uintptr
	if st, _, _ := procGdipCreateFromHDC.Call(hdc, uintptr(unsafe.Pointer(&graphics))); st != 0 || graphics == 0 {
		return
	}
	procGdipDrawImageRectI.Call(graphics, app.image, uintptr(x), uintptr(y), uintptr(dw), uintptr(dh))
	procGdipDeleteGraphics.Call(graphics)
}

func previewSupportsZoom() bool {
	return pdfPreview.active || app.image != 0 || app.preview != nil || app.textPreviewActive || app.xlsxPreviewActive || app.pptxPreviewActive
}

func previewSupportsFit() bool {
	if pdfPreview.active || app.image != 0 || app.textPreviewActive || app.xlsxPreviewActive || app.pptxPreviewActive {
		return true
	}
	// A native IPreviewHandler does not expose one standard document-zoom API,
	// but DCM can always fit the handler's host rectangle to the preview pane.
	// Known handlers such as the Edge PDF fallback additionally receive their
	// native fit shortcut in fitPreview/autoFitPreviewIfSupported.
	return app.preview != nil
}

func previewSupportsFitWidth() bool {
	return pdfPreview.active || app.image != 0 || app.pptxPreviewActive
}

func updatePreviewZoomControls() {
	// The embedded viewer supplies its own controls. Restore ours on unload.
	show := uintptr(5) // SW_SHOW
	if sumatraView != nil {
		show = 0 // SW_HIDE
	}
	for _, hwnd := range []uintptr{app.zoomOutBtn, app.zoom100Btn, app.zoomInBtn, app.zoomFitBtn, app.zoomFitWidthBtn} {
		if hwnd != 0 {
			procShowWindow.Call(hwnd, show)
		}
	}
	zoomEnabled := uintptr(0)
	fitEnabled := uintptr(0)
	fitWidthEnabled := uintptr(0)
	if previewSupportsZoom() {
		zoomEnabled = 1
	}
	if previewSupportsFit() {
		fitEnabled = 1
	}
	if previewSupportsFitWidth() {
		fitWidthEnabled = 1
	}
	for _, hwnd := range []uintptr{app.zoomOutBtn, app.zoom100Btn, app.zoomInBtn} {
		if hwnd != 0 {
			procEnableWindow.Call(hwnd, zoomEnabled)
		}
	}
	if app.zoomFitBtn != 0 {
		procEnableWindow.Call(app.zoomFitBtn, fitEnabled)
	}
	if app.zoomFitWidthBtn != 0 {
		procEnableWindow.Call(app.zoomFitWidthBtn, fitWidthEnabled)
	}
}

func resetPreviewZoomState() {
	app.previewFit = true
	app.previewFitWidth = false
	app.previewZoom = 100
	app.previewZoomWheelRemainder = 0
	app.previewPanX, app.previewPanY = 0, 0
	app.previewDragging = false
	updatePreviewZoomControls()
}

func imageFitWidthZoomPercent() int {
	if app.image == 0 || app.previewHwnd == 0 {
		return 100
	}
	var iw, ih uint32
	procGdipGetImageWidth.Call(app.image, uintptr(unsafe.Pointer(&iw)))
	procGdipGetImageHeight.Call(app.image, uintptr(unsafe.Pointer(&ih)))
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	rc = pdfContentRect(rc)
	availW := rc.Right - rc.Left - 8
	dw, _ := fitWidthDimensions(int32(iw), int32(ih), availW)
	if iw == 0 || dw <= 0 {
		return 100
	}
	percent := int((int64(dw)*100 + int64(iw)/2) / int64(iw))
	return clampPreviewZoom(percent)
}

func imageFitZoomPercent() int {
	if app.image == 0 || app.previewHwnd == 0 {
		return 100
	}
	var iw, ih uint32
	procGdipGetImageWidth.Call(app.image, uintptr(unsafe.Pointer(&iw)))
	procGdipGetImageHeight.Call(app.image, uintptr(unsafe.Pointer(&ih)))
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	availW, availH := rc.Right-rc.Left-8, rc.Bottom-rc.Top-8
	dw, _ := fitDimensions(int32(iw), int32(ih), availW, availH)
	if iw == 0 || dw <= 0 {
		return 100
	}
	percent := int((int64(dw)*100 + int64(iw)/2) / int64(iw))
	return clampPreviewZoom(percent)
}

func setImageZoom(percent int) {
	app.previewFit = false
	app.previewFitWidth = false
	app.previewZoom = clampPreviewZoom(percent)
	app.previewPanX, app.previewPanY = 0, 0
	procInvalidateRect.Call(app.previewHwnd, 0, 1)
	setText(app.statusLabel, fmt.Sprintf("Image preview zoom: %d%% | Ctrl+mouse wheel or drag to inspect", app.previewZoom))
}

func sendPreviewCtrlKey(vk uintptr, shift bool) bool {
	if app.preview == nil {
		return false
	}
	self := uintptr(unsafe.Pointer(app.preview))
	previousFocus, _, _ := procGetFocus.Call()
	hr, _, _ := syscall.SyscallN(app.preview.Vtbl.SetFocus, self)
	if hresultFailed(hr) {
		return false
	}
	var target uintptr
	hr, _, _ = syscall.SyscallN(app.preview.Vtbl.QueryFocus, self, uintptr(unsafe.Pointer(&target)))
	if hresultFailed(hr) || target == 0 {
		if previousFocus != 0 {
			procSetFocus.Call(previousFocus)
		}
		return false
	}

	// PostMessage delivers WM_KEYDOWN/WM_KEYUP records but does not update the
	// process-wide keyboard state queried by native preview handlers. Edge's PDF
	// preview therefore received '+'/'-' while GetKeyState(VK_CONTROL) still
	// reported Ctrl as up. Inject a real keyboard sequence after IPreviewHandler
	// has moved focus into its child window so Ctrl+/Ctrl-/Ctrl+0/Ctrl+\ are
	// interpreted exactly like user keyboard shortcuts.
	ctrlWasDown := keyDown(VK_CONTROL)
	shiftWasDown := keyDown(VK_SHIFT)
	if !ctrlWasDown {
		procKeybdEvent.Call(VK_CONTROL, 0, 0, 0)
	}
	if shift && !shiftWasDown {
		procKeybdEvent.Call(VK_SHIFT, 0, 0, 0)
	}
	procKeybdEvent.Call(vk, 0, 0, 0)
	procKeybdEvent.Call(vk, 0, KEYEVENTF_KEYUP, 0)
	if shift && !shiftWasDown {
		procKeybdEvent.Call(VK_SHIFT, 0, KEYEVENTF_KEYUP, 0)
	}
	if !ctrlWasDown {
		procKeybdEvent.Call(VK_CONTROL, 0, KEYEVENTF_KEYUP, 0)
	}
	time.Sleep(30 * time.Millisecond)
	if previousFocus != 0 {
		procSetFocus.Call(previousFocus)
	}
	return true
}

func setPreviewZoom100() {
	app.previewZoomWheelRemainder = 0
	if pdfPreview.active {
		app.previewFit = false
		app.previewFitWidth = false
		app.previewZoom = 100
		app.previewPanX, app.previewPanY = 0, 0
		requestPDFPreviewRender()
		return
	}
	if app.pptxPreviewActive {
		setPPTXPreviewZoom(100)
		return
	}
	if app.xlsxPreviewActive {
		setXLSXPreviewZoom(100)
		return
	}
	if app.textPreviewActive {
		setTextPreviewZoom(100)
		return
	}
	if app.image != 0 {
		setImageZoom(100)
		return
	}
	if app.preview != nil && sendPreviewCtrlKey(VK_0, false) {
		app.previewFit = false
		app.previewFitWidth = false
		setText(app.statusLabel, "Native preview: requested 100% zoom")
	}
}

func fitPreview() {
	app.previewZoomWheelRemainder = 0
	if pdfPreview.active {
		app.previewFit = true
		app.previewFitWidth = false
		app.previewPanX, app.previewPanY = 0, 0
		requestPDFPreviewRender()
		return
	}
	if app.pptxPreviewActive {
		app.previewFit = true
		app.previewFitWidth = false
		app.previewZoom = 100
		app.previewPanX, app.previewPanY = 0, 0
		procInvalidateRect.Call(app.previewHwnd, 0, 1)
		setText(app.statusLabel, "PPTX preview: Fit")
		updatePreviewZoomControls()
		return
	}
	if app.textPreviewActive {
		app.previewFit = true
		app.previewFitWidth = false
		app.previewZoom = 100
		app.previewPanX, app.previewPanY = 0, 0
		applyTextPreviewZoom()
		resizeTextPreview()
		kind := app.textPreviewKind
		if kind == "" {
			kind = "Text"
		}
		setText(app.statusLabel, kind+" preview: Fit to pane")
		updatePreviewZoomControls()
		return
	}
	if app.xlsxPreviewActive {
		app.previewFit = true
		app.previewFitWidth = false
		app.previewZoom = 100
		app.previewPanX, app.previewPanY = 0, 0
		applyXLSXPreviewZoom()
		resizeXLSXPreview()
		setText(app.statusLabel, "XLSX preview: Fit to pane")
		updatePreviewZoomControls()
		return
	}
	if app.image != 0 {
		app.previewFit = true
		app.previewFitWidth = false
		app.previewZoom = imageFitZoomPercent()
		app.previewPanX, app.previewPanY = 0, 0
		procInvalidateRect.Call(app.previewHwnd, 0, 1)
		setText(app.statusLabel, fmt.Sprintf("Image preview: Fit · %d%%", app.previewZoom))
		updatePreviewZoomControls()
		return
	}
	if app.preview != nil {
		app.previewFit = true
		app.previewFitWidth = false
		app.previewZoom = 100
		app.previewPanX, app.previewPanY = 0, 0
		resizePreviewHandler()
		path := currentFile()
		if path != "" && toggleCurrentPDFPreviewFit(path, previewHandlerCLSID(path)) {
			setText(app.statusLabel, "Native PDF preview: Fit")
		} else {
			setText(app.statusLabel, "Native preview: Fit to pane")
		}
		updatePreviewZoomControls()
	}
}

func fitPreviewWidth() {
	app.previewZoomWheelRemainder = 0
	if pdfPreview.active {
		app.previewFit = false
		app.previewFitWidth = true
		app.previewPanX, app.previewPanY = 0, 0
		requestPDFPreviewRender()
		return
	}
	if app.pptxPreviewActive {
		app.previewFit = false
		app.previewFitWidth = true
		app.previewPanX = 0
		var rc RECT
		procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
		const comboBarH int32 = 36
		availW := rc.Right - rc.Left - 16
		availH := rc.Bottom - rc.Top - comboBarH - 12
		_, dh := pptxSlidePixelSizeMode(app.pptxDeck.Width, app.pptxDeck.Height, availW, availH, app.previewZoom, false, true)
		app.previewPanY = previewTopPanOffset(dh, availH)
		procInvalidateRect.Call(app.previewHwnd, 0, 1)
		setText(app.statusLabel, "PPTX preview: Fit Width · top · mouse wheel to scroll")
		updatePreviewZoomControls()
		return
	}
	if app.image != 0 {
		app.previewFit = false
		app.previewFitWidth = true
		app.previewZoom = imageFitWidthZoomPercent()
		app.previewPanX = 0
		var iw, ih uint32
		procGdipGetImageWidth.Call(app.image, uintptr(unsafe.Pointer(&iw)))
		procGdipGetImageHeight.Call(app.image, uintptr(unsafe.Pointer(&ih)))
		var rc RECT
		procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
		rc = pdfContentRect(rc)
		availW, availH := rc.Right-rc.Left-8, rc.Bottom-rc.Top-8
		_, dh := fitWidthDimensions(int32(iw), int32(ih), availW)
		app.previewPanY = previewTopPanOffset(dh, availH)
		procInvalidateRect.Call(app.previewHwnd, 0, 1)
		setText(app.statusLabel, fmt.Sprintf("Image preview: Fit Width · top · %d%%", app.previewZoom))
		updatePreviewZoomControls()
	}
}

func adjustPreviewZoom(detents int) {
	if detents == 0 {
		return
	}
	if pdfPreview.active {
		base := app.previewZoom
		if (app.previewFit || app.previewFitWidth) && pdfPreview.lastScalePercent > 0 {
			base = pdfPreview.lastScalePercent
		}
		app.previewFit = false
		app.previewFitWidth = false
		app.previewZoom = nextPreviewZoomBounded(base, detents, pdfPreviewMinZoom, pdfPreviewMaxZoom)
		requestPDFPreviewRender()
		return
	}
	if app.pptxPreviewActive {
		base := app.previewZoom
		if app.previewFit {
			base = 100
		} else if app.previewFitWidth {
			base = pptxFitWidthZoomPercent()
		}
		setPPTXPreviewZoom(nextPreviewZoom(base, detents))
		return
	}
	if app.xlsxPreviewActive {
		setXLSXPreviewZoom(nextPreviewZoom(app.previewZoom, detents))
		return
	}
	if app.textPreviewActive {
		setTextPreviewZoom(nextPreviewZoom(app.previewZoom, detents))
		return
	}
	if app.image != 0 {
		base := app.previewZoom
		if app.previewFit {
			base = imageFitZoomPercent()
		} else if app.previewFitWidth {
			base = imageFitWidthZoomPercent()
		}
		setImageZoom(nextPreviewZoom(base, detents))
		return
	}
	path := currentFile()
	if app.preview == nil {
		return
	}
	if app.previewFit && shouldAutoFitPreview(path, previewHandlerCLSID(path)) {
		// Edge PDF fit mode has no exposed percentage. Reset to a known 100%
		// baseline before the first manual zoom step.
		if !sendPreviewCtrlKey(VK_0, false) {
			return
		}
		app.previewFit = false
		app.previewFitWidth = false
	}
	if detents > 0 {
		for i := 0; i < detents; i++ {
			if !sendPreviewCtrlKey(VK_OEM_PLUS, true) {
				break
			}
		}
		setText(app.statusLabel, "Native preview: zoom in command sent")
	} else {
		for i := 0; i > detents; i-- {
			if !sendPreviewCtrlKey(VK_OEM_MINUS, false) {
				break
			}
		}
		setText(app.statusLabel, "Native preview: zoom out command sent")
	}
}

func pptxFitWidthZoomPercent() int {
	if !app.pptxPreviewActive || app.previewHwnd == 0 || app.pptxDeck.Width <= 0 || app.pptxDeck.Height <= 0 {
		return 100
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	const comboBarH int32 = 34
	availW := rc.Right - rc.Left - 16
	availH := rc.Bottom - rc.Top - comboBarH - 8
	fitW, _ := pptxSlidePixelSize(app.pptxDeck.Width, app.pptxDeck.Height, availW, availH, 100, true)
	if fitW <= 0 || availW <= 0 {
		return 100
	}
	return clampPreviewZoom(int((int64(availW)*100 + int64(fitW)/2) / int64(fitW)))
}

func handlePreviewWheelPan(delta int, horizontal bool) bool {
	if delta == 0 || app.previewHwnd == 0 {
		return false
	}
	if app.textPreviewActive && app.textPreviewEdit != 0 {
		if horizontal {
			return true // wrapped text has no horizontal scroll surface
		}
		return scrollPreviewEdit(app.textPreviewEdit, delta, false)
	}
	if app.xlsxPreviewActive && app.xlsxGridEdit != 0 {
		return scrollPreviewEdit(app.xlsxGridEdit, delta, horizontal)
	}
	if app.image != 0 || app.pptxPreviewActive {
		var rc RECT
		procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
		viewport := rc.Bottom - rc.Top
		if horizontal {
			viewport = rc.Right - rc.Left
		}
		amount := previewPanPixels(delta, viewport)
		if horizontal {
			app.previewPanX += amount
		} else {
			app.previewPanY += amount
		}
		procInvalidateRect.Call(app.previewHwnd, 0, 1)
		return true
	}
	// Native preview handlers retain their own scrolling semantics because DCM
	// does not own the document surface inside the handler.
	return false
}

func scrollPreviewEdit(hwnd uintptr, delta int, horizontal bool) bool {
	if hwnd == 0 || delta == 0 || app.previewHwnd == 0 {
		return false
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	viewport := rc.Bottom - rc.Top
	if horizontal {
		viewport = rc.Right - rc.Left
	}
	pixels := previewPanPixels(delta, viewport)
	if pixels == 0 {
		return true
	}
	unit := int32(16)
	if !horizontal {
		unit = int32(textPreviewFontPixelHeight(app.previewZoom))
		if unit < 8 {
			unit = 8
		}
	}
	steps := int(abs32(pixels) / unit)
	if steps < 1 {
		steps = 1
	}
	message := uintptr(WM_VSCROLL)
	up, down := uintptr(SB_LINEUP), uintptr(SB_LINEDOWN)
	if horizontal {
		message = WM_HSCROLL
		up, down = SB_LINELEFT, SB_LINERIGHT
	}
	command := down
	if delta > 0 {
		command = up
	}
	for i := 0; i < steps; i++ {
		procSendMessageW.Call(hwnd, message, command, 0)
	}
	return true
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func resizePreviewHandler() {
	if app.preview == nil {
		return
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	syscall.SyscallN(app.preview.Vtbl.SetRect, uintptr(unsafe.Pointer(app.preview)), uintptr(unsafe.Pointer(&rc)))
}

func toggleCurrentPDFPreviewFit(path, clsidText string) bool {
	if app.preview == nil || !shouldAutoFitPreview(path, clsidText) {
		return false
	}
	// Edge documents Ctrl+\ as the PDF fit-to-page / fit-to-width toggle.
	return sendPreviewCtrlKey(VK_OEM_5, false)
}

func autoFitPreviewIfSupported(path, clsidText string) {
	if toggleCurrentPDFPreviewFit(path, clsidText) {
		resizePreviewHandler()
	}
}

func loadWindowsPreviewHandler(path string) bool {
	clsidText := previewHandlerCLSID(path)
	if clsidText == "" {
		return false
	}
	var clsid, iidPreview GUID
	if !guidFromString(clsidText, &clsid) || !guidFromString("{8895b1c6-b41f-4c1c-a562-0d564250836f}", &iidPreview) {
		return false
	}
	ph := createPreviewHandler(&clsid, &iidPreview)
	if ph == nil {
		return false
	}
	self := uintptr(unsafe.Pointer(ph))
	// Preview handlers are designed to communicate with their host through
	// IObjectWithSite/IPreviewHandlerFrame. Office previewers in particular
	// can fail when hosted without a site even though DoPreview returns.
	procIUnknownSetSite.Call(self, uintptr(unsafe.Pointer(&previewSite)))
	if !initializePreviewObject((*iUnknown)(unsafe.Pointer(ph)), path) {
		procIUnknownSetSite.Call(self, 0)
		syscall.SyscallN(ph.Vtbl.Release, self)
		return false
	}
	var rc RECT
	procGetClientRect.Call(app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	hr, _, _ := syscall.SyscallN(ph.Vtbl.SetWindow, self, app.previewHwnd, uintptr(unsafe.Pointer(&rc)))
	if hresultFailed(hr) {
		procIUnknownSetSite.Call(self, 0)
		syscall.SyscallN(ph.Vtbl.Release, self)
		return false
	}
	hr, _, _ = syscall.SyscallN(ph.Vtbl.DoPreview, self)
	if hresultFailed(hr) {
		syscall.SyscallN(ph.Vtbl.Unload, self)
		procIUnknownSetSite.Call(self, 0)
		syscall.SyscallN(ph.Vtbl.Release, self)
		return false
	}
	app.preview = ph
	app.previewFit = true
	app.previewFitWidth = false
	autoFitPreviewIfSupported(path, clsidText)
	updatePreviewZoomControls()
	return true
}

func createPreviewHandler(clsid, iidPreview *GUID) *iPreviewHandler {
	// Explorer-style preview handlers are intended to run out of process in
	// Prevhost.exe. Prefer LOCAL_SERVER so COM can use that surrogate. Some
	// third-party handlers are registered only in-process, so keep a guarded
	// compatibility fallback.
	for _, ctx := range []uintptr{CLSCTX_LOCAL_SERVER, CLSCTX_INPROC_SERVER} {
		var ph *iPreviewHandler
		hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(clsid)), 0, ctx, uintptr(unsafe.Pointer(iidPreview)), uintptr(unsafe.Pointer(&ph)))
		if !hresultFailed(hr) && ph != nil {
			return ph
		}
	}
	return nil
}

func initializePreviewObject(obj *iUnknown, path string) bool {
	self := uintptr(unsafe.Pointer(obj))
	var iidStream GUID
	if guidFromString("{b824b49d-22ac-4161-ac8a-9916e8fa3f7f}", &iidStream) {
		var initStream *iInitializeWithStream
		hr, _, _ := syscall.SyscallN(obj.Vtbl.QueryInterface, self, uintptr(unsafe.Pointer(&iidStream)), uintptr(unsafe.Pointer(&initStream)))
		if !hresultFailed(hr) && initStream != nil {
			var stream *iUnknown
			p := wstr(path)
			shr, _, _ := procSHCreateStreamOnFileEx.Call(uintptr(unsafe.Pointer(p)), STGM_READ|STGM_SHARE_DENY_NONE, FILE_ATTRIBUTE_NORMAL, 0, 0, uintptr(unsafe.Pointer(&stream)))
			if !hresultFailed(shr) && stream != nil {
				initSelf := uintptr(unsafe.Pointer(initStream))
				ihr, _, _ := syscall.SyscallN(initStream.Vtbl.Initialize, initSelf, uintptr(unsafe.Pointer(stream)), STGM_READ)
				releaseUnknown(stream)
				syscall.SyscallN(initStream.Vtbl.Release, initSelf)
				if !hresultFailed(ihr) {
					return true
				}
			} else {
				syscall.SyscallN(initStream.Vtbl.Release, uintptr(unsafe.Pointer(initStream)))
			}
		}
	}
	var iidItem GUID
	if guidFromString("{7F73BE3F-FB79-493C-A6C7-7EE14E245841}", &iidItem) {
		var initItem *iInitializeWithItem
		hr, _, _ := syscall.SyscallN(obj.Vtbl.QueryInterface, self, uintptr(unsafe.Pointer(&iidItem)), uintptr(unsafe.Pointer(&initItem)))
		if !hresultFailed(hr) && initItem != nil {
			var iidShellItem GUID
			if guidFromString("{43826D1E-E718-42EE-BC55-A1E261C37BFE}", &iidShellItem) {
				var shellItem *iUnknown
				p := wstr(path)
				shr, _, _ := procSHCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(&iidShellItem)), uintptr(unsafe.Pointer(&shellItem)))
				if !hresultFailed(shr) && shellItem != nil {
					initSelf := uintptr(unsafe.Pointer(initItem))
					ihr, _, _ := syscall.SyscallN(initItem.Vtbl.Initialize, initSelf, uintptr(unsafe.Pointer(shellItem)), STGM_READ)
					releaseUnknown(shellItem)
					syscall.SyscallN(initItem.Vtbl.Release, initSelf)
					if !hresultFailed(ihr) {
						return true
					}
				} else {
					syscall.SyscallN(initItem.Vtbl.Release, uintptr(unsafe.Pointer(initItem)))
				}
			} else {
				syscall.SyscallN(initItem.Vtbl.Release, uintptr(unsafe.Pointer(initItem)))
			}
		}
	}
	var iidFile GUID
	if !guidFromString("{b7d14566-0509-4cce-a71f-0a554233bd9b}", &iidFile) {
		return false
	}
	var initFile *iInitializeWithFile
	hr, _, _ := syscall.SyscallN(obj.Vtbl.QueryInterface, self, uintptr(unsafe.Pointer(&iidFile)), uintptr(unsafe.Pointer(&initFile)))
	if hresultFailed(hr) || initFile == nil {
		return false
	}
	initSelf := uintptr(unsafe.Pointer(initFile))
	p := wstr(path)
	ihr, _, _ := syscall.SyscallN(initFile.Vtbl.Initialize, initSelf, uintptr(unsafe.Pointer(p)), STGM_READ)
	syscall.SyscallN(initFile.Vtbl.Release, initSelf)
	return !hresultFailed(ihr)
}

func releaseUnknown(ptr *iUnknown) {
	if ptr == nil {
		return
	}
	syscall.SyscallN(ptr.Vtbl.Release, uintptr(unsafe.Pointer(ptr)))
}

func guidFromString(s string, out *GUID) bool {
	hr, _, _ := procCLSIDFromString.Call(uintptr(unsafe.Pointer(wstr(s))), uintptr(unsafe.Pointer(out)))
	return !hresultFailed(hr)
}

func previewHandlerCLSID(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return ""
	}
	const key = `{8895b1c6-b41f-4c1c-a562-0d564250836f}`
	prog, _ := regDefault(ext)
	candidates := []string{}
	if prog != "" {
		candidates = append(candidates, prog+`\shellex\`+key)
	}
	candidates = append(candidates, ext+`\shellex\`+key, `SystemFileAssociations\`+ext+`\shellex\`+key)
	for _, c := range candidates {
		if v, err := regDefault(c); err == nil && strings.HasPrefix(strings.TrimSpace(v), "{") {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func regDefault(subkey string) (string, error) {
	const HKEY_CLASSES_ROOT = uintptr(0x80000000)
	p := wstr(subkey)
	var typ uint32
	var bytes uint32
	r, _, _ := procRegGetValueW.Call(HKEY_CLASSES_ROOT, uintptr(unsafe.Pointer(p)), 0, RRF_RT_REG_SZ, uintptr(unsafe.Pointer(&typ)), 0, uintptr(unsafe.Pointer(&bytes)))
	if r != 0 || bytes < 2 {
		return "", fmt.Errorf("registry value missing")
	}
	buf := make([]uint16, (bytes+1)/2)
	r, _, _ = procRegGetValueW.Call(HKEY_CLASSES_ROOT, uintptr(unsafe.Pointer(p)), 0, RRF_RT_REG_SZ, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bytes)))
	if r != 0 {
		return "", fmt.Errorf("registry read failed")
	}
	return syscall.UTF16ToString(buf), nil
}

func registerClasses(hInst uintptr) bool {
	cursor, _, _ := procLoadCursorW.Call(0, IDC_ARROW)
	mainClass := wstr("QuickSortMain")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: mainWndProc, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: mainClass}
	if r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return false
	}
	prevClass := wstr("QuickSortPreview")
	pwc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: previewWndProc, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: prevClass}
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&pwc)))
	if r == 0 {
		return false
	}
	dmClass := wstr("QuickSortDestinations")
	dmwc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: destinationManagerWndProcCB, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: dmClass}
	r, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&dmwc)))
	if r == 0 {
		return false
	}
	if !registerSourceManagerClass(hInst, cursor) {
		return false
	}
	if !registerRecoveryCenterClass(hInst, cursor) {
		return false
	}
	if !registerLearningCenterClass(hInst, cursor) {
		return false
	}
	if !registerSuggestionCenterClass(hInst, cursor) {
		return false
	}
	if !registerFileFlowMetadataPromptClass(hInst, cursor) {
		return false
	}
	return registerNamingTemplateManagerClass(hInst, cursor)
}

func initGDIPlus() bool {
	input := GdiplusStartupInput{GdiplusVersion: 1}
	status, _, _ := procGdiplusStartup.Call(uintptr(unsafe.Pointer(&app.gpToken)), uintptr(unsafe.Pointer(&input)), 0)
	return status == 0
}

func main() {
	defer flushLayoutTrace()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procCoInitializeEx.Call(0, COINIT_APARTMENTTHREADED)
	defer procCoUninitialize.Call()
	initGDIPlus()
	hInst, _, _ := procGetModuleHandleW.Call(0)
	if !registerClasses(hInst) {
		return
	}
	// V1 is intentionally a fixed-size utility window. Removing the thick frame
	// and maximize box prevents layout drift and keeps the preview pane stable.
	windowStyle := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE | WS_CLIPCHILDREN)
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(wstr("QuickSortMain"))), uintptr(unsafe.Pointer(wstr("DCM - Quick Sort (Downloaded Content Management) — "+currentAppVersion))), windowStyle, CW_USEDEFAULT, CW_USEDEFAULT, 1280, 820, 0, 0, hInst, 0)
	if hwnd == 0 {
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if msg.Message == WM_MOUSEWHEEL && (msg.Hwnd == app.previewHwnd || msg.Hwnd == app.textPreviewEdit || msg.Hwnd == app.xlsxGridEdit || pointInWindow(app.previewHwnd, msg.Pt)) {
			delta := int(int16(hiword(msg.WParam)))
			if keyDown(VK_CONTROL) && previewSupportsZoom() {
				if detents := consumePreviewWheelDetents(&app.previewZoomWheelRemainder, delta); detents != 0 {
					adjustPreviewZoom(detents)
				}
				continue
			}
			if handlePreviewWheelPan(delta, keyDown(VK_SHIFT)) {
				continue
			}
		}
		if msg.Message == WM_SYSKEYDOWN && !keyDown(VK_CONTROL) {
			// Native-looking Alt+letter mnemonics. Ctrl+Alt is deliberately left
			// untouched so AltGr keyboard input keeps working. Main-window
			// mnemonics stay inactive while Manage Destinations has focus.
			focus, _, _ := procGetFocus.Call()
			if !focusWithinDestinationManager(focus) && !focusWithinSourceManager(focus) && !focusWithinNamingTemplateManager(focus) && activateAltShortcut(rune(msg.WParam)) {
				continue
			}
		}
		if msg.Message == 0x0100 { // WM_KEYDOWN, including child controls
			handled := false
			focus, _, _ := procGetFocus.Call()
			if focusWithinSourceManager(focus) {
				switch msg.WParam {
				case VK_TAB:
					handled = moveSourceManagerFocus(keyDown(VK_SHIFT))
				case VK_ESCAPE:
					procPostMessageW.Call(sourceManager.hwnd, WM_CLOSE, 0, 0)
					handled = true
				case VK_RETURN:
					switch focus {
					case sourceManager.nameEdit, sourceManager.folderEdit:
						saveSourceManagerAlias()
						handled = true
					case sourceManager.browseBtn, sourceManager.saveBtn, sourceManager.deleteBtn, sourceManager.newBtn, sourceManager.useBtn, sourceManager.closeBtn:
						procSendMessageW.Call(focus, BM_CLICK, 0, 0)
						handled = true
					}
				}
				if handled {
					continue
				}
				procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
				procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
				continue
			}
			if focusWithinNamingTemplateManager(focus) {
				switch msg.WParam {
				case VK_TAB:
					handled = moveNamingTemplateManagerFocus(keyDown(VK_SHIFT))
				case VK_ESCAPE:
					procPostMessageW.Call(namingTemplateManager.hwnd, WM_CLOSE, 0, 0)
					handled = true
				case VK_RETURN:
					switch focus {
					case namingTemplateManager.nameEdit, namingTemplateManager.patternEdit:
						handled = saveNamingTemplateFromManager()
					case namingTemplateManager.variableNameEdit, namingTemplateManager.variableValueEdit:
						saveNamingVariableFromManager()
						handled = true
					case namingTemplateManager.saveBtn, namingTemplateManager.newBtn, namingTemplateManager.deleteBtn, namingTemplateManager.activeBtn, namingTemplateManager.variableSaveBtn, namingTemplateManager.variableDeleteBtn, namingTemplateManager.closeBtn:
						procSendMessageW.Call(focus, BM_CLICK, 0, 0)
						handled = true
					}
				}
				if handled {
					continue
				}
				procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
				procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
				continue
			}
			if focusWithinDestinationManager(focus) {
				switch msg.WParam {
				case VK_TAB:
					handled = moveDestinationManagerFocus(keyDown(VK_SHIFT))
				case VK_ESCAPE:
					procPostMessageW.Call(destManager.hwnd, WM_CLOSE, 0, 0)
					handled = true
				case VK_RETURN:
					switch focus {
					case destManager.searchEdit:
						if len(destManager.locations) > 0 {
							setDestinationManagerLocation(destManager.locations[0].Address)
							handled = true
						}
					case destManager.childNameEdit:
						createDestinationManagerChild()
						handled = true
					case destManager.favoriteNameEdit:
						saveDestinationManagerFavorite()
						handled = true
					case destManager.locationList:
						useDestinationManagerLocation()
						handled = true
					case destManager.favoritesList:
						useDestinationManagerLocation()
						handled = true
					case destManager.changeVaultBtn, destManager.createChildBtn, destManager.saveFavoriteBtn, destManager.deleteFavoriteBtn, destManager.openLocationBtn, destManager.rebuildIndexBtn, destManager.useLocationBtn, destManager.exportConfigBtn, destManager.importConfigBtn, destManager.restoreLatestBtn, destManager.backupFolderBtn, destManager.closeBtn:
						procSendMessageW.Call(focus, BM_CLICK, 0, 0)
						handled = true
					}
				}
				if handled {
					continue
				}
				procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
				procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
				continue
			}
			switch msg.WParam {
			case VK_TAB:
				handled = movePrimaryFocus(keyDown(VK_SHIFT))
			case VK_DOWN:
				if focusWithinSource() {
					handled = moveSourceSuggestionSelection(1)
				} else if focusWithinDestination() {
					handled = moveDestinationSuggestionSelection(1)
				}
			case VK_UP:
				if focusWithinSource() {
					handled = moveSourceSuggestionSelection(-1)
				} else if focusWithinDestination() {
					handled = moveDestinationSuggestionSelection(-1)
				}
			case VK_RETURN:
				if keyDown(VK_CONTROL) {
					moveCurrent()
					handled = true
					break
				}
				if focusWithinSource() && acceptSourceSuggestion() {
					handled = true
					break
				}
				if focus == app.sourceEdit {
					activateSourcePath(getText(app.sourceEdit))
					handled = true
					break
				}
				if focusWithinDestination() && acceptDestinationSuggestion() {
					handled = true
					break
				}
				if focus == app.nameEdit {
					focusDestinationForTyping()
					handled = true
					break
				}
				if focus == app.destEdit {
					moveCurrent()
					handled = true
				}
			case VK_ESCAPE:
				if app.sourceSuggestionsVisible && focusWithinSource() {
					hideSourceSuggestions()
					handled = true
					break
				}
				if app.destSuggestionsVisible && focusWithinDestination() {
					hideDestinationSuggestions()
					handled = true
					break
				}
				if focus == app.nameEdit || focus == app.destEdit || focus == app.sourceEdit || focus == app.filterNameEdit || focus == app.processedSearchEdit {
					handled = focusFileList()
				}
			case VK_OEM_PLUS, VK_ADD:
				if keyDown(VK_CONTROL) {
					app.previewZoomWheelRemainder = 0
					adjustPreviewZoom(1)
					handled = true
				}
			case VK_OEM_MINUS, VK_SUBTRACT:
				if keyDown(VK_CONTROL) {
					app.previewZoomWheelRemainder = 0
					adjustPreviewZoom(-1)
					handled = true
				}
			case VK_0:
				if keyDown(VK_CONTROL) {
					setPreviewZoom100()
					handled = true
				}
			case VK_F5:
				reloadFiles()
				handled = true
			case VK_A:
				if focus == app.fileList && keyDown(VK_CONTROL) {
					handled = selectAllVisibleSourceFiles()
				}
			case VK_Z:
				if keyDown(VK_CONTROL) {
					undoSelectedProcessed()
					handled = true
				}
			case VK_OEM_5:
				if keyDown(VK_CONTROL) && previewSupportsZoom() {
					if keyDown(VK_SHIFT) && previewSupportsFitWidth() {
						fitPreviewWidth()
					} else {
						fitPreview()
					}
					handled = true
				}
			}
			if handled {
				continue
			}
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
