package main

// A resize issued from a synchronous native input message cannot safely call
// an out-of-process Office COM handler. Defer only this known COM context
// failure, only for the current Excel viewer, and only once per queued request.
func shouldPostExcelPreviewResize(hr uintptr, excelActive, retryRunning, pending bool) bool {
	return uint32(hr) == 0x8001010d && excelActive && !retryRunning && !pending
}

// Ignore messages posted for a handler that has since been unloaded or whose
// preview pane has been replaced. Only the current owned Excel viewer retries.
func shouldRunExcelPreviewResizeRetry(pending, excelActive bool, messageEpoch, currentEpoch uint64, messagePane, currentPane uintptr) bool {
	return pending && excelActive && messageEpoch == currentEpoch && messagePane == currentPane
}
