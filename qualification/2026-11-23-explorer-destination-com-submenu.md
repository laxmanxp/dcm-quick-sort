# DCM QuickSort — dynamic Explorer destination submenu qualification

Date: 2026-11-23
Branch: `feature/quicksort-two-pane-history-tabs`

## Implemented
- Native 64-bit C++ classic Explorer context menu via `IShellExtInit` + `IContextMenu`, with a COM class factory and unloadable DLL.
- Reads current `%LOCALAPPDATA%\DCM Quick Sort\destinations.conf` on invocation; presents one `Send to QuickSort Destination` submenu with available named destination favorites.
- Reads all files in one Explorer `CF_HDROP` selection, up to 256, rejects directories/reparse points, and passes all selected absolute paths to one Go launcher.
- Launcher creates a JSON batch manifest and starts one QuickSort process with the existing `--explorer-send-request` entrypoint.
- QuickSort re-resolves the destination favorite, checks source files, prompts for confirmation, then uses its history-aware batch engine with conflict and Undo Batch handling. No move is performed by the COM DLL.
- Per-user registration and unregistration script is explicit, requires `-ApprovedIsolatedAccount`, verifies installed artifacts and exact install directory, and performs no action by default.
- Reproducible `build.ps1` generates a statically linked native DLL plus the Go launcher. The DLL no longer imports `libc++.dll` or `libunwind.dll`; standard Windows UCRT API-set dependencies remain.
- Existing `Sort with DCM Quick Sort` registry entries are unchanged.

## Lenovo evidence
1. Native COM DLL compiled with LLVM-MinGW `clang++ -std=c++17 -O2 -static -shared`.
2. Isolated `smoke.cpp` loaded the DLL directly (no HKCU Explorer registration) and provided synthetic `CF_HDROP` with two real disposable files.
3. Menu contained one parent submenu and exactly two available saved destinations, in config order; nonexistent destination omitted.
4. `InvokeCommand` triggered the actual Go launcher and delivered one request containing both files to a harmless fake QuickSort receiver. Neither original file changed.
5. Folder selection produced no file-send submenu. COM objects were released, DLL unloaded, and disposable files removed.
6. Reproducible staged DLL/helper build passed and those **exact staged artifacts** passed the complete COM smoke test.
7. Registration script parsed without error, and default invocation explicitly refused registration.
8. Earlier QuickSort main `go test ./... -count=1` and Windows build passed after Explorer handoff implementation. Go helper tests passed.

Staged developer artifacts (Lenovo only): `C:\DCMQuickSortCollisionAcceptance\ExplorerStaging\DCMQuickSortExplorerSend.dll` and `DCMQuickSortExplorerSend.exe`.

## Outstanding release gates
- **Not installed or registered in real Windows Explorer.** Classic menu appears under Windows 11 **Show more options** where supported. A real disposable-account/VM UI test is still needed, including Explorer multi-selection across ordinary file types, menu refresh after favorites changes, invoking confirmation/cancel, conflict handling, and Undo Batch.
- Installer packaging of the DLL and helper and clean uninstall/upgrade must be qualified. Current installer is intentionally unchanged.
- No proof that third-party shell policy permits loading in every deployment configuration; deployed Windows versions need acceptance.
- Launcher command-line length limit (~30,000 UTF-16 characters) may exclude very large selections despite the 256-file cap; it fails without moving in this case. Native manifest writing could remove that limit in a later iteration.
- A real already-running QuickSort instance is not yet reused: one new app process is launched per **batch invocation**, never per file. IPC reuse can be added separately.

No user documents moved; no installed package or production HKCU Explorer registration changed.
