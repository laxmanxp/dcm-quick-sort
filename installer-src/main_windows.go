//go:build windows

package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed payload/DCM-QuickSort.exe
var appPayload []byte

const (
	WM_CREATE                = 0x0001
	WM_DESTROY               = 0x0002
	WM_COMMAND               = 0x0111
	WM_CLOSE                 = 0x0010
	WM_SETFONT               = 0x0030
	WS_OVERLAPPEDWINDOW      = 0x00CF0000
	WS_THICKFRAME            = 0x00040000
	WS_MAXIMIZEBOX           = 0x00010000
	WS_VISIBLE               = 0x10000000
	WS_CHILD                 = 0x40000000
	WS_TABSTOP               = 0x00010000
	WS_BORDER                = 0x00800000
	ES_AUTOHSCROLL           = 0x0080
	BS_PUSHBUTTON            = 0x00000000
	BS_AUTOCHECKBOX          = 0x00000003
	BST_CHECKED              = 1
	BM_GETCHECK              = 0x00F0
	BM_SETCHECK              = 0x00F1
	SS_LEFT                  = 0
	CW_USEDEFAULT            = 0x80000000
	SW_SHOW                  = 5
	IDC_ARROW                = 32512
	COLOR_WINDOW             = 5
	DEFAULT_GUI_FONT         = 17
	BN_CLICKED               = 0
	MB_OK                    = 0
	MB_ICONERROR             = 0x10
	MB_ICONINFORMATION       = 0x40
	MB_YESNO                 = 0x04
	MB_ICONQUESTION          = 0x20
	IDYES                    = 6
	ID_SOURCE_EDIT           = 1001
	ID_BROWSE                = 1002
	ID_INSTALL               = 1003
	ID_CANCEL                = 1004
	ID_LAUNCH                = 1005
	COINIT_APARTMENTTHREADED = 0x2
	CLSCTX_INPROC_SERVER     = 0x1
	FOS_PICKFOLDERS          = 0x20
	FOS_FORCEFILESYSTEM      = 0x40
	FOS_PATHMUSTEXIST        = 0x800
	SIGDN_FILESYSPATH        = 0x80058000
	S_OK                     = 0
	ERROR_SUCCESS            = 0
	HKEY_CURRENT_USER        = uintptr(0x80000001)
	KEY_SET_VALUE            = 0x0002
	REG_SZ                   = 1
	REG_DWORD                = 4
	CREATE_NO_WINDOW         = 0x08000000
)

type POINT struct{ X, Y int32 }
type MSG struct {
	Hwnd           uintptr
	Message        uint32
	_              uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
	LPrivate       uint32
}
type WNDCLASSEXW struct {
	CbSize, Style                            uint32
	LpfnWndProc                              uintptr
	CbClsExtra, CbWndExtra                   int32
	HInstance, HIcon, HCursor, HbrBackground uintptr
	LpszMenuName, LpszClassName              *uint16
	HIconSm                                  uintptr
}
type GUID struct {
	Data1        uint32
	Data2, Data3 uint16
	Data4        [8]byte
}
type iFileOpenDialog struct{ Vtbl *iFileOpenDialogVtbl }
type iFileOpenDialogVtbl struct {
	QueryInterface, AddRef, Release                                                     uintptr
	Show, SetFileTypes, SetFileTypeIndex, GetFileTypeIndex, Advise, Unadvise            uintptr
	SetOptions, GetOptions, SetDefaultFolder, SetFolder, GetFolder, GetCurrentSelection uintptr
	SetFileName, GetFileName, SetTitle, SetOkButtonLabel, SetFileNameLabel, GetResult   uintptr
	AddPlace, SetDefaultExtension, Close, SetClientGuid, ClearClientData, SetFilter     uintptr
	GetResults, GetSelectedItems                                                        uintptr
}
type iShellItem struct{ Vtbl *iShellItemVtbl }
type iShellItemVtbl struct{ QueryInterface, AddRef, Release, BindToHandler, GetParent, GetDisplayName, GetAttributes, Compare uintptr }
type iShellLinkW struct{ Vtbl *iShellLinkWVtbl }
type iShellLinkWVtbl struct {
	QueryInterface, AddRef, Release                                                uintptr
	GetPath, GetIDList, SetIDList, GetDescription, SetDescription                  uintptr
	GetWorkingDirectory, SetWorkingDirectory, GetArguments, SetArguments           uintptr
	GetHotkey, SetHotkey, GetShowCmd, SetShowCmd, GetIconLocation, SetIconLocation uintptr
	SetRelativePath, Resolve, SetPath                                              uintptr
}
type iPersistFile struct{ Vtbl *iPersistFileVtbl }
type iPersistFileVtbl struct{ QueryInterface, AddRef, Release, GetClassID, IsDirty, Load, Save, SaveCompleted, GetCurFile uintptr }

var (
	user32                          = syscall.NewLazyDLL("user32.dll")
	kernel32                        = syscall.NewLazyDLL("kernel32.dll")
	shell32                         = syscall.NewLazyDLL("shell32.dll")
	ole32                           = syscall.NewLazyDLL("ole32.dll")
	advapi32                        = syscall.NewLazyDLL("advapi32.dll")
	procRegisterClassExW            = user32.NewProc("RegisterClassExW")
	procCreateWindowExW             = user32.NewProc("CreateWindowExW")
	procDefWindowProcW              = user32.NewProc("DefWindowProcW")
	procShowWindow                  = user32.NewProc("ShowWindow")
	procUpdateWindow                = user32.NewProc("UpdateWindow")
	procGetMessageW                 = user32.NewProc("GetMessageW")
	procTranslateMessage            = user32.NewProc("TranslateMessage")
	procDispatchMessageW            = user32.NewProc("DispatchMessageW")
	procPostQuitMessage             = user32.NewProc("PostQuitMessage")
	procSetWindowTextW              = user32.NewProc("SetWindowTextW")
	procGetWindowTextW              = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW        = user32.NewProc("GetWindowTextLengthW")
	procSendMessageW                = user32.NewProc("SendMessageW")
	procSetFocus                    = user32.NewProc("SetFocus")
	procLoadCursorW                 = user32.NewProc("LoadCursorW")
	procMessageBoxW                 = user32.NewProc("MessageBoxW")
	procGetModuleHandleW            = kernel32.NewProc("GetModuleHandleW")
	procCoInitializeEx              = ole32.NewProc("CoInitializeEx")
	procCoUninitialize              = ole32.NewProc("CoUninitialize")
	procCoCreateInstance            = ole32.NewProc("CoCreateInstance")
	procCLSIDFromString             = ole32.NewProc("CLSIDFromString")
	procCoTaskMemFree               = ole32.NewProc("CoTaskMemFree")
	procSHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
	procRegCreateKeyExW             = advapi32.NewProc("RegCreateKeyExW")
	procRegSetValueExW              = advapi32.NewProc("RegSetValueExW")
	procRegCloseKey                 = advapi32.NewProc("RegCloseKey")
	procRegDeleteTreeW              = advapi32.NewProc("RegDeleteTreeW")
)

var hwndMain, editSource, checkLaunch, checkRestore uintptr
var previousBackupAvailable bool
var mainWndProc = syscall.NewCallback(windowProc)
var installSucceeded bool

func wstr(s string) *uint16        { p, _ := syscall.UTF16PtrFromString(s); return p }
func loword(v uintptr) uint16      { return uint16(v & 0xffff) }
func hiword(v uintptr) uint16      { return uint16((v >> 16) & 0xffff) }
func hresultFailed(v uintptr) bool { return int32(v) < 0 }

func createControl(class, text string, style uintptr, x, y, w, h int32, parent uintptr, id int) uintptr {
	inst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(wstr(class))), uintptr(unsafe.Pointer(wstr(text))), style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, uintptr(id), inst, 0)
	font, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("GetStockObject").Call(DEFAULT_GUI_FONT)
	procSendMessageW.Call(hwnd, WM_SETFONT, font, 1)
	return hwnd
}

func setText(hwnd uintptr, s string) { procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(wstr(s)))) }
func getText(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	buf := make([]uint16, int(n)+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func message(title, text string, flags uintptr) uintptr {
	r, _, _ := procMessageBoxW.Call(hwndMain, uintptr(unsafe.Pointer(wstr(text))), uintptr(unsafe.Pointer(wstr(title))), flags)
	return r
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		hwndMain = hwnd
		createControl("STATIC", "Install DCM - Quick Sort", WS_CHILD|WS_VISIBLE|SS_LEFT, 24, 20, 550, 26, hwnd, 0)
		createControl("STATIC", "Choose the default folder DCM should open for sorting. This is usually your Downloads folder.", WS_CHILD|WS_VISIBLE|SS_LEFT, 24, 54, 570, 38, hwnd, 0)
		createControl("STATIC", "Default download/source folder:", WS_CHILD|WS_VISIBLE|SS_LEFT, 24, 105, 220, 20, hwnd, 0)
		editSource = createControl("EDIT", installerDefaultSource(), WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 24, 130, 455, 27, hwnd, ID_SOURCE_EDIT)
		createControl("BUTTON", "&Browse…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 489, 129, 100, 29, hwnd, ID_BROWSE)
		_, _, _, settingsPath, _ := installPaths()
		if info, ok := shouldOfferPreviousConfigurationRestore(filepath.Dir(settingsPath), configurationBackupRoot()); ok {
			previousBackupAvailable = true
			label := "&Restore previous DCM configuration backup"
			if strings.TrimSpace(info.AppVersion) != "" {
				label += " (from " + info.AppVersion + ")"
			}
			checkRestore = createControl("BUTTON", label, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, 24, 170, 520, 24, hwnd, 0)
			procSendMessageW.Call(checkRestore, BM_SETCHECK, BST_CHECKED, 0)
		}
		launchY := int32(174)
		infoY := int32(207)
		buttonY := int32(240)
		if previousBackupAvailable {
			launchY, infoY, buttonY = 202, 235, 270
		}
		checkLaunch = createControl("BUTTON", "&Launch DCM - Quick Sort when setup finishes", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, 24, launchY, 350, 24, hwnd, ID_LAUNCH)
		procSendMessageW.Call(checkLaunch, BM_SETCHECK, BST_CHECKED, 0)
		createControl("STATIC", "Installs for this Windows user only. Administrator access is not required.", WS_CHILD|WS_VISIBLE|SS_LEFT, 24, infoY, 560, 20, hwnd, 0)
		createControl("BUTTON", "&Install", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 390, buttonY, 95, 32, hwnd, ID_INSTALL)
		createControl("BUTTON", "Cancel", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 494, buttonY, 95, 32, hwnd, ID_CANCEL)
		procSetFocus.Call(editSource)
		return 0
	case WM_COMMAND:
		if hiword(wParam) == BN_CLICKED {
			switch int(loword(wParam)) {
			case ID_BROWSE:
				if p := browseFolder(hwnd, "Choose default folder to sort", getText(editSource)); p != "" {
					setText(editSource, p)
				}
			case ID_INSTALL:
				source := strings.TrimSpace(getText(editSource))
				if source == "" {
					message(productName+" Setup", "Choose a default source folder.", MB_OK|MB_ICONERROR)
					return 0
				}
				restorePrevious := false
				if checkRestore != 0 {
					checked, _, _ := procSendMessageW.Call(checkRestore, BM_GETCHECK, 0, 0)
					restorePrevious = checked == BST_CHECKED
				}
				if err := install(source, restorePrevious); err != nil {
					message(productName+" Setup", "Installation failed:\n\n"+err.Error(), MB_OK|MB_ICONERROR)
					return 0
				}
				installSucceeded = true
				message(productName+" Setup", "DCM - Quick Sort was installed successfully.", MB_OK|MB_ICONINFORMATION)
				checked, _, _ := procSendMessageW.Call(checkLaunch, BM_GETCHECK, 0, 0)
				if checked == BST_CHECKED {
					launchInstalled()
				}
				procPostQuitMessage.Call(0)
			case ID_CANCEL:
				procPostQuitMessage.Call(0)
			}
		}
		return 0
	case WM_CLOSE:
		procPostQuitMessage.Call(0)
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func installerDefaultSource() string {
	_, _, _, settingsPath, _ := installPaths()
	if data, err := os.ReadFile(settingsPath); err == nil {
		if cfg, err := validateInstallerSettings(data); err == nil {
			return cfg.DefaultSource
		}
	}
	if source := latestBackupDefaultSource(configurationBackupRoot()); strings.TrimSpace(source) != "" {
		return source
	}
	home, _ := os.UserHomeDir()
	return defaultSourceFolder(home)
}

func installPaths() (installDir, appExe, uninstallExe, settingsPath, shortcut string) {
	local := os.Getenv("LOCALAPPDATA")
	roaming := os.Getenv("APPDATA")
	installDir = filepath.Join(local, "Programs", "DCM Quick Sort")
	appExe = filepath.Join(installDir, "DCM-QuickSort.exe")
	uninstallExe = filepath.Join(installDir, "Uninstall DCM Quick Sort.exe")
	settingsPath = filepath.Join(local, "DCM Quick Sort", "settings.json")
	shortcut = filepath.Join(roaming, "Microsoft", "Windows", "Start Menu", "Programs", "DCM Quick Sort.lnk")
	return
}

func install(source string, restorePrevious bool) error {
	source = filepath.Clean(source)
	if err := os.MkdirAll(source, 0o755); err != nil {
		return fmt.Errorf("create source folder: %w", err)
	}
	installDir, appExe, uninstallExe, settingsPath, shortcut := installPaths()
	configDir := filepath.Dir(settingsPath)
	backupRoot := configurationBackupRoot()
	var restoreSnapshot *installerConfigurationSnapshot
	if restorePrevious {
		snapshot, err := loadLatestBackupSnapshot(backupRoot)
		if err != nil {
			return fmt.Errorf("load previous configuration backup: %w", err)
		}
		restoreSnapshot = &snapshot
	}
	if _, err := createConfigurationBackup(configDir, backupRoot, productVersion, "before-install-or-upgrade", time.Now()); err != nil && !errors.Is(err, ErrNoConfiguration) {
		return fmt.Errorf("back up existing configuration: %w", err)
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}
	tmp := appExe + ".new"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, appPayload, 0o755); err != nil {
		return fmt.Errorf("write application: %w", err)
	}
	_ = os.Remove(appExe)
	if err := os.Rename(tmp, appExe); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("activate application: %w", err)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if err := copyFile(self, uninstallExe); err != nil {
		return fmt.Errorf("create uninstaller: %w", err)
	}
	backupReason := "after-install-or-upgrade"
	if restoreSnapshot != nil {
		if err := restoreBackupSnapshotFiles(configDir, *restoreSnapshot); err != nil {
			return fmt.Errorf("restore previous configuration: %w", err)
		}
		if _, ok := restoreSnapshot.Files["settings.json"]; !ok {
			if err := writeSettings(settingsPath, source); err != nil {
				return fmt.Errorf("save default folder: %w", err)
			}
		}
		backupReason = "after-install-restore"
	} else if err := writeSettings(settingsPath, source); err != nil {
		return fmt.Errorf("save default folder: %w", err)
	}
	if _, err := createConfigurationBackup(configDir, backupRoot, productVersion, backupReason, time.Now()); err != nil {
		return fmt.Errorf("back up installed configuration: %w", err)
	}
	_ = os.Remove(shortcut)
	if err := createShortcut(shortcut, appExe, installDir); err != nil {
		return fmt.Errorf("create Start Menu shortcut: %w", err)
	}
	if err := writeUninstallRegistry(appExe, uninstallExe, installDir); err != nil {
		return fmt.Errorf("register uninstall: %w", err)
	}
	if err := writeExplorerIntegration(appExe); err != nil {
		return fmt.Errorf("register Explorer integration: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dst + ".new"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	_ = os.Remove(dst)
	return os.Rename(tmp, dst)
}

func launchInstalled() {
	_, appExe, _, _, _ := installPaths()
	_ = exec.Command(appExe).Start()
}

func uninstall(silent bool) error {
	installDir, appExe, uninstallExe, settingsPath, shortcut := installPaths()
	if !silent {
		if message(productName+" Uninstall", "Remove DCM - Quick Sort from this Windows account?\n\nYour current configuration will be backed up first.", MB_YESNO|MB_ICONQUESTION) != IDYES {
			return nil
		}
	}
	backupRoot := configurationBackupRoot()
	backupResult, backupErr := createConfigurationBackup(filepath.Dir(settingsPath), backupRoot, productVersion, "before-uninstall", time.Now())
	if backupErr != nil && !errors.Is(backupErr, ErrNoConfiguration) {
		return fmt.Errorf("back up configuration before uninstall: %w", backupErr)
	}
	_ = os.Remove(shortcut)
	_ = os.Remove(appExe)
	_ = os.Remove(settingsPath)
	_ = os.Remove(filepath.Dir(settingsPath))
	deleteRegistryTree(`Software\Microsoft\Windows\CurrentVersion\Uninstall\DCMQuickSort`)
	deleteRegistryTree(`Software\Microsoft\Windows\CurrentVersion\App Paths\DCM-QuickSort.exe`)
	for _, entry := range explorerShellEntries(appExe) {
		deleteRegistryTree(entry.Key)
	}
	batch := filepath.Join(os.TempDir(), fmt.Sprintf("dcm-quicksort-uninstall-%d.cmd", time.Now().UnixNano()))
	script := "@echo off\r\nsetlocal DisableDelayedExpansion\r\n" +
		"\"%SystemRoot%\\System32\\WindowsPowerShell\\v1.0\\powershell.exe\" -NoProfile -NonInteractive -Command \"Start-Sleep -Seconds 2\" >nul 2>nul\r\nif errorlevel 1 exit /b 3\r\n" +
		"del /f /q \"" + uninstallExe + "\" >nul 2>nul\r\nif exist \"" + uninstallExe + "\" exit /b 1\r\n" +
		"rmdir /s /q \"" + installDir + "\" >nul 2>nul\r\nif exist \"" + installDir + "\" exit /b 1\r\n" +
		"del /f /q \"%~f0\" >nul 2>nul & if exist \"%~f0\" (exit /b 2) else (exit /b 0)\r\n"
	if err := os.WriteFile(batch, []byte(script), 0o644); err == nil {
		cmd := exec.Command("cmd.exe", "/C", batch)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: CREATE_NO_WINDOW}
		_ = cmd.Start()
	}
	if !silent {
		msg := "DCM - Quick Sort has been removed."
		if backupErr == nil && backupResult.LatestDir != "" {
			msg += "\n\nConfiguration backup saved to:\n" + backupResult.LatestDir
		}
		message(productName+" Uninstall", msg, MB_OK|MB_ICONINFORMATION)
	}
	return nil
}

func createShortcut(path, target, workdir string) error {
	var clsid, iidLink, iidPersist GUID
	if !guidFromString("{00021401-0000-0000-C000-000000000046}", &clsid) ||
		!guidFromString("{000214F9-0000-0000-C000-000000000046}", &iidLink) ||
		!guidFromString("{0000010B-0000-0000-C000-000000000046}", &iidPersist) {
		return fmt.Errorf("invalid ShellLink GUID")
	}
	var link *iShellLinkW
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsid)), 0, CLSCTX_INPROC_SERVER, uintptr(unsafe.Pointer(&iidLink)), uintptr(unsafe.Pointer(&link)))
	if hresultFailed(hr) || link == nil {
		return fmt.Errorf("cannot create ShellLink (0x%08X)", uint32(hr))
	}
	self := uintptr(unsafe.Pointer(link))
	defer syscall.SyscallN(link.Vtbl.Release, self)
	if r, _, _ := syscall.SyscallN(link.Vtbl.SetPath, self, uintptr(unsafe.Pointer(wstr(target)))); hresultFailed(r) {
		return fmt.Errorf("SetPath failed")
	}
	syscall.SyscallN(link.Vtbl.SetWorkingDirectory, self, uintptr(unsafe.Pointer(wstr(workdir))))
	syscall.SyscallN(link.Vtbl.SetDescription, self, uintptr(unsafe.Pointer(wstr("DCM - Quick Sort — Downloaded Content Management"))))
	var persist *iPersistFile
	r, _, _ := syscall.SyscallN(link.Vtbl.QueryInterface, self, uintptr(unsafe.Pointer(&iidPersist)), uintptr(unsafe.Pointer(&persist)))
	if hresultFailed(r) || persist == nil {
		return fmt.Errorf("IPersistFile unavailable")
	}
	defer syscall.SyscallN(persist.Vtbl.Release, uintptr(unsafe.Pointer(persist)))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	r, _, _ = syscall.SyscallN(persist.Vtbl.Save, uintptr(unsafe.Pointer(persist)), uintptr(unsafe.Pointer(wstr(path))), 1)
	if hresultFailed(r) {
		return fmt.Errorf("save shortcut failed (0x%08X)", uint32(r))
	}
	return nil
}

func writeExplorerIntegration(appExe string) error {
	for _, entry := range explorerShellEntries(appExe) {
		if err := regSetString(entry.Key, "", entry.Label); err != nil {
			return err
		}
		if err := regSetString(entry.Key, "Icon", entry.Icon); err != nil {
			return err
		}
		if err := regSetString(entry.Key+`\command`, "", entry.Command); err != nil {
			return err
		}
	}
	return nil
}

func writeUninstallRegistry(appExe, uninstallExe, installDir string) error {
	key := `Software\Microsoft\Windows\CurrentVersion\Uninstall\DCMQuickSort`
	vals := map[string]string{
		"DisplayName":          productName,
		"DisplayVersion":       productVersion,
		"Publisher":            publisherName,
		"InstallLocation":      installDir,
		"DisplayIcon":          appExe,
		"UninstallString":      `"` + uninstallExe + `" /uninstall`,
		"QuietUninstallString": `"` + uninstallExe + `" /uninstall /S`,
	}
	for n, v := range vals {
		if err := regSetString(key, n, v); err != nil {
			return err
		}
	}
	if err := regSetDWORD(key, "NoModify", 1); err != nil {
		return err
	}
	if err := regSetDWORD(key, "NoRepair", 1); err != nil {
		return err
	}
	appKey := `Software\Microsoft\Windows\CurrentVersion\App Paths\DCM-QuickSort.exe`
	if err := regSetString(appKey, "", appExe); err != nil {
		return err
	}
	return regSetString(appKey, "Path", installDir)
}

func regCreate(path string) (uintptr, error) {
	var key uintptr
	var disp uint32
	r, _, _ := procRegCreateKeyExW.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(wstr(path))), 0, 0, 0, KEY_SET_VALUE, 0, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&disp)))
	if r != ERROR_SUCCESS {
		return 0, fmt.Errorf("registry create failed: %d", r)
	}
	return key, nil
}
func regSetString(path, name, value string) error {
	key, err := regCreate(path)
	if err != nil {
		return err
	}
	defer procRegCloseKey.Call(key)
	u, _ := syscall.UTF16FromString(value)
	bytes := uintptr(len(u) * 2)
	var namePtr uintptr
	if name != "" {
		namePtr = uintptr(unsafe.Pointer(wstr(name)))
	}
	r, _, _ := procRegSetValueExW.Call(key, namePtr, 0, REG_SZ, uintptr(unsafe.Pointer(&u[0])), bytes)
	if r != 0 {
		return fmt.Errorf("registry string write failed: %d", r)
	}
	return nil
}
func regSetDWORD(path, name string, value uint32) error {
	key, err := regCreate(path)
	if err != nil {
		return err
	}
	defer procRegCloseKey.Call(key)
	r, _, _ := procRegSetValueExW.Call(key, uintptr(unsafe.Pointer(wstr(name))), 0, REG_DWORD, uintptr(unsafe.Pointer(&value)), 4)
	if r != 0 {
		return fmt.Errorf("registry DWORD write failed: %d", r)
	}
	return nil
}
func deleteRegistryTree(path string) {
	procRegDeleteTreeW.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(wstr(path))))
}

func guidFromString(s string, out *GUID) bool {
	hr, _, _ := procCLSIDFromString.Call(uintptr(unsafe.Pointer(wstr(s))), uintptr(unsafe.Pointer(out)))
	return !hresultFailed(hr)
}

func shellItemFromPath(path string) *iShellItem {
	var iid GUID
	if !guidFromString("{43826D1E-E718-42EE-BC55-A1E261C37BFE}", &iid) {
		return nil
	}
	var item *iShellItem
	hr, _, _ := procSHCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(wstr(path))), 0, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&item)))
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

func browseFolder(owner uintptr, title, initial string) string {
	var clsid, iid GUID
	if !guidFromString("{DC1C5A9C-E88A-4DDE-A5A1-60F82A20AEF7}", &clsid) || !guidFromString("{D57C7288-D4AD-4768-BE02-9D969532D960}", &iid) {
		return ""
	}
	var d *iFileOpenDialog
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsid)), 0, CLSCTX_INPROC_SERVER, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&d)))
	if hresultFailed(hr) || d == nil {
		return ""
	}
	self := uintptr(unsafe.Pointer(d))
	defer syscall.SyscallN(d.Vtbl.Release, self)
	var options uint32
	r, _, _ := syscall.SyscallN(d.Vtbl.GetOptions, self, uintptr(unsafe.Pointer(&options)))
	if !hresultFailed(r) {
		options |= FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM | FOS_PATHMUSTEXIST
		syscall.SyscallN(d.Vtbl.SetOptions, self, uintptr(options))
	}
	syscall.SyscallN(d.Vtbl.SetTitle, self, uintptr(unsafe.Pointer(wstr(title))))
	if st, err := os.Stat(initial); err == nil && st.IsDir() {
		if item := shellItemFromPath(initial); item != nil {
			syscall.SyscallN(d.Vtbl.SetFolder, self, uintptr(unsafe.Pointer(item)))
			releaseShellItem(item)
		}
	}
	r, _, _ = syscall.SyscallN(d.Vtbl.Show, self, owner)
	if hresultFailed(r) {
		return ""
	}
	var item *iShellItem
	r, _, _ = syscall.SyscallN(d.Vtbl.GetResult, self, uintptr(unsafe.Pointer(&item)))
	if hresultFailed(r) || item == nil {
		return ""
	}
	defer releaseShellItem(item)
	var psz *uint16
	r, _, _ = syscall.SyscallN(item.Vtbl.GetDisplayName, uintptr(unsafe.Pointer(item)), SIGDN_FILESYSPATH, uintptr(unsafe.Pointer(&psz)))
	if hresultFailed(r) || psz == nil {
		return ""
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(psz)))
	return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(psz))[:])
}

func runGUI() {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	cursor, _, _ := procLoadCursorW.Call(0, IDC_ARROW)
	class := wstr("DCMQuickSortSetup")
	wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: mainWndProc, HInstance: hInst, HCursor: cursor, HbrBackground: COLOR_WINDOW + 1, LpszClassName: class}
	if r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return
	}
	style := uintptr((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_VISIBLE)
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(wstr("DCM - Quick Sort Setup — "+productVersion))), style, CW_USEDEFAULT, CW_USEDEFAULT, 630, 370, 0, 0, hInst, 0)
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
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procCoInitializeEx.Call(0, COINIT_APARTMENTTHREADED)
	defer procCoUninitialize.Call()
	silent, uninstallMode, source := parseInstallerArgs(os.Args[1:])
	exe, _ := os.Executable()
	if strings.HasPrefix(strings.ToLower(filepath.Base(exe)), "uninstall") {
		uninstallMode = true
	}
	if uninstallMode {
		if err := uninstall(silent); err != nil {
			if silent {
				os.Exit(1)
			}
			message(productName+" Uninstall", "Uninstall stopped:\n"+err.Error(), MB_OK|MB_ICONERROR)
		}
		return
	}
	if silent {
		if source == "" {
			source = installerDefaultSource()
		}
		if err := install(source, hasRestoreBackupArg(os.Args[1:])); err != nil {
			os.Exit(1)
		}
		return
	}
	runGUI()
}
