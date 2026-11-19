//go:build windows

package main

import "unsafe"

func selectedSourceIndexes() []int32 {
	if app.fileList == 0 || len(app.files) == 0 {
		return nil
	}
	count := fileListSelectionCount()
	if count <= 0 || count > len(app.files) {
		return nil
	}
	indexes := make([]int32, count)
	got, _, _ := procSendMessageW.Call(app.fileList, LB_GETSELITEMS, uintptr(count), uintptr(unsafe.Pointer(&indexes[0])))
	if got == LB_ERR || got == 0 || got > uintptr(count) {
		return nil
	}
	valid := indexes[:0]
	for _, index := range indexes[:int(got)] {
		if index >= 0 && int(index) < len(app.files) {
			valid = append(valid, index)
		}
	}
	return valid
}

func updateCurrentFromSourceSelection() {
	caret := -1
	if index, _, _ := procSendMessageW.Call(app.fileList, LB_GETCARETINDEX, 0, 0); index != LB_ERR && index < uintptr(len(app.files)) {
		caret = int(index)
	}
	app.index = activeSourceSelectionIndex(caret, selectedSourceIndexes(), len(app.files))
	// Do not reset Windows' caret or range anchor while handling user input.
	showCurrentView(false, false)
}
