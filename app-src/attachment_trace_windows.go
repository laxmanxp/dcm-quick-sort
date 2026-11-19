//go:build windows

package main

// UI-thread-only scope for one attachment. It reuses the bounded, opt-in
// startup trace. No windows, timings, ownership checks or capture rules change.
// Workers never read or write this scope; they record their own request ID.
var attachmentTraceSequence uint64

func beginAttachmentProfile(sequence uint64) func() {
	if !startupProbeEnabled || sequence == 0 {
		return func() {}
	}
	previous := attachmentTraceSequence
	attachmentTraceSequence = sequence
	recordAttachmentTrace("ui-enter")
	return func() {
		recordAttachmentTrace("ui-return")
		attachmentTraceSequence = previous
	}
}

func recordAttachmentTrace(stage string) {
	if attachmentTraceSequence != 0 {
		recordStartupTrace(attachmentTraceSequence, "attachment-"+stage)
	}
}
