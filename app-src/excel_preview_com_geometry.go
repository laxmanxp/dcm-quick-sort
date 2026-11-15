package main

// Excel's out-of-process preview may interpret IPreviewHandler rectangle
// dimensions as physical pixels even when the host thread is DPI-virtualized.
// Preserve the logical dimensions for other providers or failed measurements.
func excelCOMRectSize(logicalW, logicalH, physicalW, physicalH int32, excel, available bool) (int32, int32) {
	if excel && available && physicalW > 0 && physicalH > 0 {
		return physicalW, physicalH
	}
	return logicalW, logicalH
}


// Excel's initial SetWindow geometry is consumed under the host's logical DPI
// context; passing pre-scaled pixels there makes Excel scale its EXCEL7 child
// a second time. SetRect during an established preview has the opposite
// contract on this native handler and needs physical pane pixels.
func excelCOMRectForCall(method string, logicalW, logicalH, physicalW, physicalH int32, excel, available bool) (int32, int32) {
	if method == "SetRect" {
		return excelCOMRectSize(logicalW, logicalH, physicalW, physicalH, excel, available)
	}
	return logicalW, logicalH
}
