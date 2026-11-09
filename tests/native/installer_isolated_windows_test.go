//go:build windows

// The FlowDev runner copies this test into its staged installer module. Registry
// redirection is process-local; never launch the setup EXE under this override.
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

var (
	qualRegOpen = advapi32.NewProc("RegOpenKeyExW")
	qualRegOpenCurrentUser = advapi32.NewProc("RegOpenCurrentUser")
	qualRegQuery = advapi32.NewProc("RegQueryValueExW")
	qualRegEnumValue = advapi32.NewProc("RegEnumValueW")
	qualRegEnumKey = advapi32.NewProc("RegEnumKeyExW")
	qualRegOverride = advapi32.NewProc("RegOverridePredefKey")
)

type qualRegistryValue struct {
	Type uint32 `json:"type"`
	Data []byte `json:"data"`
}
type qualRegistryNode struct {
	Exists bool `json:"exists"`
	Values map[string]qualRegistryValue `json:"values,omitempty"`
	Children map[string]qualRegistryNode `json:"children,omitempty"`
}

func TestNativeInstallerIsolatedRegistryLifecycle(t *testing.T) {
	// RegOverridePredefKey applies to the whole test process. No parallel tests.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	root := os.Getenv("DCM_INSTALLER_EVIDENCE_ROOT")
	if !filepath.IsAbs(root) { t.Fatal("FlowDev must supply an absolute, uniquely owned evidence root") }
	if err := os.Mkdir(root, 0o700); err != nil { t.Fatal(err) }
	evidence := map[string]any{
		"schema": 1, "scope": "in-process installer engine with private HKCU redirection",
		"root": root, "started_utc": time.Now().UTC().Format(time.RFC3339Nano),
		"limitations": []string{"install() copies this test executable as the uninstaller; packaged setup self-copy is not claimed", "the installed app is not launched by this engine test", "live Explorer is not refreshed or exercised"},
	}
	defer func() {
		evidence["passed"] = !t.Failed()
		evidence["finished_utc"] = time.Now().UTC().Format(time.RFC3339Nano)
		data, err := json.MarshalIndent(evidence, "", "  ")
		if err == nil { err = os.WriteFile(filepath.Join(root, "engine-receipt.json"), append(data, '\n'), 0o600) }
		if err != nil { t.Errorf("write installer evidence: %v", err) }
	}()
	for name, relative := range map[string]string{
		"LOCALAPPDATA": "local", "APPDATA": "roaming", "TEMP": "temp", "TMP": "temp", "DCM_QUICKSORT_BACKUP_DIR": "backups",
	} {
		path := filepath.Join(root, relative)
		qualMust(t, os.MkdirAll(path, 0o700))
		t.Setenv(name, path)
	}
	installDir, appExe, uninstallExe, settingsPath, shortcut := installPaths()
	for _, path := range []string{installDir, appExe, uninstallExe, settingsPath, shortcut, configurationBackupRoot(), os.TempDir()} { qualRequireOwnedPath(t, root, path) }
	evidence["paths"] = map[string]string{"install": installDir, "app": appExe, "uninstaller": uninstallExe, "settings": settingsPath, "shortcut": shortcut, "backup": configurationBackupRoot()}

	// Keep a real-root handle before overriding the predefined HKCU pseudo-handle.
	var realRoot uintptr
	r, _, _ := qualRegOpenCurrentUser.Call(0xF003F, uintptr(unsafe.Pointer(&realRoot)))
	if r != 0 || realRoot == 0 || realRoot == HKEY_CURRENT_USER { t.Fatalf("open independent real HKCU handle: result=%d handle=%x", r, realRoot) }
	defer procRegCloseKey.Call(realRoot)
	keys := []string{
		`Software\Microsoft\Windows\CurrentVersion\Uninstall\DCMQuickSort`,
		`Software\Microsoft\Windows\CurrentVersion\App Paths\DCM-QuickSort.exe`,
		`Software\Classes\*\shell\DCMQuickSort`,
		`Software\Classes\Directory\shell\DCMQuickSort`,
		`Software\Classes\Directory\Background\shell\DCMQuickSort`,
	}
	before := qualRegistrySnapshot(t, realRoot, keys)
	evidence["live_registry_before"] = before
	var random [16]byte
	_, err := rand.Read(random[:]); qualMust(t, err)
	token := hex.EncodeToString(random[:])
	privatePath := `Software\DCMQuickSortQualification-` + token
	var privateRoot uintptr
	var disposition uint32
	// The real installer creates normal nonvolatile child keys. Windows rejects
	// those below a volatile parent, so this uniquely owned root is also normal.
	r, _, _ = procRegCreateKeyExW.Call(realRoot, uintptr(unsafe.Pointer(wstr(privatePath))), 0, 0, 0, 0xF003F, 0, uintptr(unsafe.Pointer(&privateRoot)), uintptr(unsafe.Pointer(&disposition)))
	if r != 0 || disposition != 1 { t.Fatalf("create fresh qualification key: result=%d disposition=%d", r, disposition) }
	evidence["private_registry_root"] = `HKCU\` + privatePath
	t.Logf("owned private registry root: HKCU\\%s", privatePath)
	overridden := false
	defer func() {
		if overridden {
			code, _, _ := qualRegOverride.Call(HKEY_CURRENT_USER, 0)
			evidence["registry_override_reset"] = code == 0
			if code != 0 { t.Errorf("reset HKCU redirection: %d", code) }
		}
		procRegCloseKey.Call(privateRoot)
		code, _, _ := procRegDeleteTreeW.Call(realRoot, uintptr(unsafe.Pointer(wstr(privatePath))))
		evidence["private_registry_removed"] = code == 0
		if code != 0 { t.Errorf("remove owned private registry tree: %d", code) }
		after := qualRegistrySnapshot(t, realRoot, keys)
		evidence["live_registry_after"] = after
		evidence["live_registry_unchanged"] = reflect.DeepEqual(before, after)
		if !reflect.DeepEqual(before, after) { t.Error("live DCM registry keys changed during isolated installer test") }
	}()
	marker, _ := syscall.UTF16FromString(token)
	r, _, _ = procRegSetValueExW.Call(privateRoot, uintptr(unsafe.Pointer(wstr("qualification-owner"))), 0, REG_SZ, uintptr(unsafe.Pointer(&marker[0])), uintptr(len(marker)*2))
	if r != 0 { t.Fatalf("write private hive marker: %d", r) }
	r, _, _ = qualRegOverride.Call(HKEY_CURRENT_USER, privateRoot)
	if r != 0 { t.Fatalf("HKCU override failed; no installer call is permitted: %d", r) }
	overridden = true
	qualAssertMarker(t, token)
	co, _, _ := procCoInitializeEx.Call(0, COINIT_APARTMENTTHREADED)
	if int32(co) < 0 { t.Fatalf("STA COM initialization failed: 0x%08X", uint32(co)) }
	defer procCoUninitialize.Call()

	sourceA := filepath.Join(root, "source-a")
	sourceB := filepath.Join(root, "source-b")
	vault := filepath.Join(root, "Vault")
	qualMust(t, os.MkdirAll(filepath.Join(vault, "Accepted"), 0o700))
	qualAssertMarker(t, token)
	qualMust(t, install(sourceA, false))
	installed, err := os.ReadFile(appExe); qualMust(t, err)
	if !bytes.Equal(installed, appPayload) { t.Fatal("installed app differs from exact embedded payload") }
	self, err := os.Executable(); qualMust(t, err)
	selfBytes, err := os.ReadFile(self); qualMust(t, err)
	uninstallerBytes, err := os.ReadFile(uninstallExe); qualMust(t, err)
	if !bytes.Equal(selfBytes, uninstallerBytes) { t.Fatal("engine self-copy differs from test executable") }
	linkTarget, linkWorkdir, err := qualShortcutTarget(shortcut); qualMust(t, err)
	if !strings.EqualFold(linkTarget, appExe) || !strings.EqualFold(linkWorkdir, installDir) { t.Fatalf("shortcut target/workdir mismatch: %q / %q", linkTarget, linkWorkdir) }
	qualAssertRegistration(t, appExe, uninstallExe, installDir)
	evidence["fresh_install"] = map[string]any{"installed_app_sha256": qualSHA(installed), "embedded_app_sha256": qualSHA(appPayload), "test_uninstaller_sha256": qualSHA(uninstallerBytes), "shortcut_target": linkTarget, "shortcut_workdir": linkWorkdir, "shadow_registry": qualRegistrySnapshot(t, HKEY_CURRENT_USER, keys)}

	configDir := filepath.Dir(settingsPath)
	cfg, err := json.Marshal(map[string]string{"default_source": sourceA, "pdf_default_view": "fit_page", "vault_root": vault}); qualMust(t, err)
	seed := map[string][]byte{
		"settings.json": cfg,
		"destinations.conf": []byte("Acceptance=Accepted\n"),
		"sources.conf": []byte("Inbox=" + sourceA + "\n"),
		"learning.json": []byte(`{"version":1,"total_observations":7,"destination_counts":{"Accepted":7}}`),
		"naming-templates.json": []byte(`{"active":"Voucher","templates":[{"name":"Voucher","pattern":"{office}-{name}","variables":[{"name":"office","value":"BDO"}]}]}`),
	}
	for name, data := range seed { qualMust(t, os.WriteFile(filepath.Join(configDir, name), data, 0o600)) }
	qualAssertMarker(t, token)
	qualMust(t, install(sourceB, false))
	upgradedSettings, err := os.ReadFile(settingsPath); qualMust(t, err)
	var upgraded map[string]string
	qualMust(t, json.Unmarshal(upgradedSettings, &upgraded))
	if upgraded["default_source"] != sourceB || upgraded["pdf_default_view"] != "fit_page" || upgraded["vault_root"] != vault { t.Fatalf("upgrade lost app settings: %#v", upgraded) }
	seed["settings.json"] = upgradedSettings
	qualAssertConfiguration(t, configDir, seed)
	qualAssertRegistration(t, appExe, uninstallExe, installDir)
	evidence["upgrade_configuration"] = qualConfigHashes(seed)
	qualRequireOwnedPath(t, root, configDir)
	qualMust(t, os.RemoveAll(configDir))
	qualAssertMarker(t, token)
	qualMust(t, install(sourceA, true))
	qualAssertConfiguration(t, configDir, seed)
	qualAssertConfiguration(t, filepath.Join(configurationBackupRoot(), "latest"), seed)
	evidence["restored_configuration"] = qualConfigHashes(seed)
	qualAssertRegistration(t, appExe, uninstallExe, installDir)

	qualAssertMarker(t, token)
	cleanupStart := time.Now()
	qualMust(t, uninstall(true))
	children, err := qualOwnedCleanupChildren(cleanupStart); qualMust(t, err)
	defer func() { for _, child := range children { syscall.CloseHandle(child.Handle) } }()
	var childIDs []uint32
	evidence["cleanup_child_count"] = len(children)
	for _, child := range children {
		childIDs = append(childIDs, child.PID)
		wait, err := syscall.WaitForSingleObject(child.Handle, 15000)
		if err != nil || wait != syscall.WAIT_OBJECT_0 {
			_ = syscall.TerminateProcess(child.Handle, 1)
			_, _ = syscall.WaitForSingleObject(child.Handle, 5000)
			t.Fatalf("owned uninstall cleanup did not exit: pid=%d wait=%d err=%v", child.PID, wait, err)
		}
		var exitCode uint32
		qualMust(t, syscall.GetExitCodeProcess(child.Handle, &exitCode))
		if exitCode != 0 { t.Fatalf("owned uninstall cleanup exited %d", exitCode) }
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		_, dirErr := os.Stat(installDir)
		scripts, globErr := filepath.Glob(filepath.Join(os.TempDir(), "dcm-quicksort-uninstall-*.cmd")); qualMust(t, globErr)
		if os.IsNotExist(dirErr) && len(scripts) == 0 { break }
		if time.Now().After(deadline) { t.Fatal("owned uninstall directory/script cleanup deadline") }
		time.Sleep(100 * time.Millisecond)
	}
	for _, path := range []string{appExe, uninstallExe, installDir, settingsPath, shortcut} { if _, err := os.Stat(path); !os.IsNotExist(err) { t.Errorf("uninstall left %s (err=%v)", path, err) } }
	afterUninstall := qualRegistrySnapshot(t, HKEY_CURRENT_USER, keys)
	for path, node := range afterUninstall { if node.Exists { t.Errorf("shadow registration survived uninstall: %s", path) } }
	qualAssertConfiguration(t, filepath.Join(configurationBackupRoot(), "latest"), seed)
	evidence["uninstall"] = map[string]any{"cleanup_pids": childIDs, "install_directory_removed": true, "cleanup_scripts_removed": true, "shadow_registry": afterUninstall, "retained_backup": qualConfigHashes(seed)}
}

func qualMust(t *testing.T, err error) { t.Helper(); if err != nil { t.Fatal(err) } }
func qualSHA(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func qualRequireOwnedPath(t *testing.T, root, path string) {
	t.Helper(); rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) { t.Fatalf("installer path escapes owned fixture: %q", path) }
}
func qualAssertConfiguration(t *testing.T, dir string, expected map[string][]byte) {
	t.Helper(); for name, want := range expected { got, err := os.ReadFile(filepath.Join(dir, name)); qualMust(t, err); if !bytes.Equal(got, want) { t.Fatalf("configuration bytes changed: %s", name) } }
}
func qualConfigHashes(files map[string][]byte) map[string]string { out := map[string]string{}; for name, data := range files { out[name] = qualSHA(data) }; return out }
func qualAssertMarker(t *testing.T, token string) {
	t.Helper(); var kind, size uint32
	r, _, _ := qualRegQuery.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(wstr("qualification-owner"))), 0, uintptr(unsafe.Pointer(&kind)), 0, uintptr(unsafe.Pointer(&size)))
	if r != 0 || kind != REG_SZ || size == 0 || size > 1024 { t.Fatal("HKCU is not bound to the owned qualification root; refusing installer call") }
	data := make([]uint16, (size+1)/2)
	r, _, _ = qualRegQuery.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(wstr("qualification-owner"))), 0, uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&size)))
	if r != 0 || syscall.UTF16ToString(data) != token { t.Fatal("private HKCU marker mismatch; refusing installer call") }
}
func qualRegistrySnapshot(t *testing.T, root uintptr, paths []string) map[string]qualRegistryNode {
	t.Helper(); out := map[string]qualRegistryNode{}
	for _, path := range paths { node, err := qualReadRegistry(root, path); qualMust(t, err); out[path] = node }
	return out
}
func qualReadRegistry(root uintptr, path string) (qualRegistryNode, error) {
	var key uintptr
	r, _, _ := qualRegOpen.Call(root, uintptr(unsafe.Pointer(wstr(path))), 0, 0x20019, uintptr(unsafe.Pointer(&key)))
	if r == 2 { return qualRegistryNode{}, nil }
	if r != 0 { return qualRegistryNode{}, fmt.Errorf("registry open %s: %d", path, r) }
	defer procRegCloseKey.Call(key)
	node := qualRegistryNode{Exists: true, Values: map[string]qualRegistryValue{}, Children: map[string]qualRegistryNode{}}
	for index := uint32(0); ; index++ {
		name := make([]uint16, 16384); nameSize := uint32(len(name)); data := make([]byte, 1<<20); dataSize := uint32(len(data)); var kind uint32
		r, _, _ := qualRegEnumValue.Call(key, uintptr(index), uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&nameSize)), 0, uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&dataSize)))
		if r == 259 { break }; if r != 0 { return node, fmt.Errorf("registry enumerate values: %d", r) }
		node.Values[syscall.UTF16ToString(name[:nameSize])] = qualRegistryValue{Type: kind, Data: append([]byte(nil), data[:dataSize]...)}
	}
	for index := uint32(0); ; index++ {
		name := make([]uint16, 256); size := uint32(len(name))
		r, _, _ := qualRegEnumKey.Call(key, uintptr(index), uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&size)), 0, 0, 0, 0)
		if r == 259 { break }; if r != 0 { return node, fmt.Errorf("registry enumerate child keys: %d", r) }
		childName := syscall.UTF16ToString(name[:size]); child, err := qualReadRegistry(key, childName); if err != nil { return node, err }; node.Children[childName] = child
	}
	return node, nil
}
func qualRegistryString(t *testing.T, path, name string) string {
	t.Helper(); node, err := qualReadRegistry(HKEY_CURRENT_USER, path); qualMust(t, err)
	value, ok := node.Values[name]; if !node.Exists || !ok || value.Type != REG_SZ || len(value.Data)%2 != 0 { t.Fatalf("missing registry string %s [%s]", path, name) }
	words := make([]uint16, len(value.Data)/2); for i := range words { words[i] = binary.LittleEndian.Uint16(value.Data[i*2:]) }; return syscall.UTF16ToString(words)
}
func qualAssertRegistration(t *testing.T, appExe, uninstallExe, installDir string) {
	t.Helper()
	expected := map[string]map[string]string{
		`Software\Microsoft\Windows\CurrentVersion\Uninstall\DCMQuickSort`: {"DisplayVersion": productVersion, "InstallLocation": installDir, "UninstallString": `"`+uninstallExe+`" /uninstall`, "QuietUninstallString": `"`+uninstallExe+`" /uninstall /S`},
		`Software\Microsoft\Windows\CurrentVersion\App Paths\DCM-QuickSort.exe`: {"": appExe, "Path": installDir},
		`Software\Classes\*\shell\DCMQuickSort\command`: {"": `"`+appExe+`" "%1"`},
		`Software\Classes\Directory\shell\DCMQuickSort\command`: {"": `"`+appExe+`" "%1"`},
		`Software\Classes\Directory\Background\shell\DCMQuickSort\command`: {"": `"`+appExe+`" "%V"`},
	}
	for path, values := range expected { for name, want := range values { if got := qualRegistryString(t, path, name); got != want { t.Fatalf("registration mismatch %s [%s]: got %q want %q", path, name, got, want) } } }
}
func qualShortcutTarget(path string) (string, string, error) {
	var clsid, iidLink, iidPersist GUID
	if !guidFromString("{00021401-0000-0000-C000-000000000046}", &clsid) || !guidFromString("{000214F9-0000-0000-C000-000000000046}", &iidLink) || !guidFromString("{0000010B-0000-0000-C000-000000000046}", &iidPersist) { return "", "", fmt.Errorf("ShellLink GUIDs") }
	var link *iShellLinkW
	r, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsid)), 0, CLSCTX_INPROC_SERVER, uintptr(unsafe.Pointer(&iidLink)), uintptr(unsafe.Pointer(&link)))
	if hresultFailed(r) || link == nil { return "", "", fmt.Errorf("open ShellLink: 0x%08X", uint32(r)) }
	self := uintptr(unsafe.Pointer(link)); defer syscall.SyscallN(link.Vtbl.Release, self)
	var persist *iPersistFile
	r, _, _ = syscall.SyscallN(link.Vtbl.QueryInterface, self, uintptr(unsafe.Pointer(&iidPersist)), uintptr(unsafe.Pointer(&persist)))
	if hresultFailed(r) || persist == nil { return "", "", fmt.Errorf("ShellLink persistence") }
	defer syscall.SyscallN(persist.Vtbl.Release, uintptr(unsafe.Pointer(persist)))
	r, _, _ = syscall.SyscallN(persist.Vtbl.Load, uintptr(unsafe.Pointer(persist)), uintptr(unsafe.Pointer(wstr(path))), 0)
	if hresultFailed(r) { return "", "", fmt.Errorf("load shortcut: 0x%08X", uint32(r)) }
	buf := make([]uint16, 32768)
	r, _, _ = syscall.SyscallN(link.Vtbl.GetPath, self, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, 4)
	if hresultFailed(r) { return "", "", fmt.Errorf("read shortcut target") }
	target := syscall.UTF16ToString(buf)
	for i := range buf { buf[i] = 0 }
	r, _, _ = syscall.SyscallN(link.Vtbl.GetWorkingDirectory, self, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if hresultFailed(r) { return "", "", fmt.Errorf("read shortcut working directory") }
	return target, syscall.UTF16ToString(buf), nil
}
type qualCleanupChild struct { PID uint32; Handle syscall.Handle }
func qualOwnedCleanupChildren(start time.Time) ([]qualCleanupChild, error) {
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0); if err != nil { return nil, err }; defer syscall.CloseHandle(snapshot)
	var entry syscall.ProcessEntry32; entry.Size = uint32(unsafe.Sizeof(entry))
	var out []qualCleanupChild
	err = syscall.Process32First(snapshot, &entry)
	for err == nil {
		if entry.ParentProcessID == uint32(os.Getpid()) && strings.EqualFold(syscall.UTF16ToString(entry.ExeFile[:]), "cmd.exe") {
			handle, openErr := syscall.OpenProcess(0x100000|0x1000|0x1, false, entry.ProcessID)
			if openErr == nil {
				var created, exited, kernel, user syscall.Filetime
				if syscall.GetProcessTimes(handle, &created, &exited, &kernel, &user) == nil && time.Unix(0, created.Nanoseconds()).After(start.Add(-time.Millisecond)) { out = append(out, qualCleanupChild{entry.ProcessID, handle}) } else { syscall.CloseHandle(handle) }
			}
		}
		err = syscall.Process32Next(snapshot, &entry)
	}
	if err != syscall.ERROR_NO_MORE_FILES { return out, err }
	return out, nil
}
