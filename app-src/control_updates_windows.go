//go:build windows

package main

// Initial behavior preserved for the RED contract test. The next change will
// elide only native calls whose current live state already matches the request.
func moveLayoutCombo(hwnd, x, y, width, height uintptr) bool {
	if hwnd == 0 {
		return false
	}
	procMoveWindow.Call(hwnd, x, y, width, height, 0)
	return true
}
func setPreviewControlVisible(hwnd uintptr, show bool) bool {
	if hwnd == 0 {
		return false
	}
	command := uintptr(0)
	if show {
		command = 5
	}
	procShowWindow.Call(hwnd, command)
	return true
}
func setPreviewControlEnabled(hwnd uintptr, enabled bool) bool {
	if hwnd == 0 {
		return false
	}
	value := uintptr(0)
	if enabled {
		value = 1
	}
	procEnableWindow.Call(hwnd, value)
	return true
}
