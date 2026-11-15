package main

// The observed Windows PowerPoint preview handler consumes physical pane
// pixels for its SetWindow and SetRect rectangles when the host thread is
// DPI-virtualized. Do not change other preview providers or guess when the
// owned pane could not be measured in a physical DPI context.
func pptxCOMRectSize(logicalW, logicalH, physicalW, physicalH int32, pptx, available bool) (int32, int32) {
	if pptx && available && physicalW > 0 && physicalH > 0 {
		return physicalW, physicalH
	}
	return logicalW, logicalH
}

// Out-of-process PowerPoint may reject SetRect from synchronous WM_SIZE with
// RPC_E_CANTCALLOUT_ININPUTSYNCCALL. Queue exactly one safe message after
// window sizing unwinds. No foreign child HWND is manipulated here.
func shouldPostPptxPreviewResize(hr uintptr, pptxActive, retryRunning, pending bool) bool {
	return uint32(hr) == 0x8001010d && pptxActive && !retryRunning && !pending
}

// Ignore stale messages after the PPTX preview was replaced, unloaded, or
// moved to another owned pane.
func shouldRunPptxPreviewResizeRetry(pending, pptxActive bool, messageEpoch, currentEpoch uint64, messagePane, currentPane uintptr) bool {
	return pending && pptxActive && messageEpoch == currentEpoch && messagePane == currentPane
}
