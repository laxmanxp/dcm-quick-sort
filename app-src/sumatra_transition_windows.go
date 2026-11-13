//go:build windows

package main

import "unsafe"

// A viewer transition changes host geometry only when it borrows sidebar
// space. At the normal width, keep all 49 host controls where they already are;
// zoom-button visibility and the viewer's own size are updated separately.
// Actual WM_SIZE handling still always runs the complete layout.
func layoutSumatraTransition() {
	var rc RECT
	ok, _, _ := procGetClientRect.Call(app.hwnd, uintptr(unsafe.Pointer(&rc)))
	left, right := sumatraSidebarWidths(rc.Right - rc.Left)
	if ok == 0 || left != 220 || right != 260 {
		layoutUI(app.hwnd)
	}
}
