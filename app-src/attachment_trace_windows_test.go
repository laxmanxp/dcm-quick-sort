//go:build windows

package main

import "testing"

func TestAttachmentProfileScope(t *testing.T) {
	oldEnabled, oldSequence := startupProbeEnabled, attachmentTraceSequence
	oldEvents := startupProbeEvents
	defer func() {
		startupProbeEnabled, attachmentTraceSequence = oldEnabled, oldSequence
		startupProbeEvents = oldEvents
	}()
	for _, enabled := range []bool{false, true} {
		startupProbeEnabled, attachmentTraceSequence = enabled, 0
		startupProbeEvents = nil
		end := beginAttachmentProfile(7)
		recordAttachmentTrace("checkpoint")
		innerEnd := beginAttachmentProfile(8)
		recordAttachmentTrace("inner")
		innerEnd()
		recordAttachmentTrace("restored")
		end()
		recordAttachmentTrace("outside")
		if attachmentTraceSequence != 0 {
			t.Fatal("scope leaked")
		}
		if !enabled {
			if len(startupProbeEvents) != 0 {
				t.Fatal("disabled trace recorded events")
			}
			continue
		}
		want := []struct {
			id    uint64
			stage string
		}{
			{7, "attachment-ui-enter"}, {7, "attachment-checkpoint"},
			{8, "attachment-ui-enter"}, {8, "attachment-inner"},
			{8, "attachment-ui-return"}, {7, "attachment-restored"},
			{7, "attachment-ui-return"},
		}
		if len(startupProbeEvents) != len(want) {
			t.Fatalf("events: %d", len(startupProbeEvents))
		}
		for i, w := range want {
			if startupProbeEvents[i].Sequence != w.id || startupProbeEvents[i].Stage != w.stage {
				t.Fatalf("event %d: %+v", i, startupProbeEvents[i])
			}
		}
	}
	startupProbeEnabled, attachmentTraceSequence = true, 0
	startupProbeEvents = nil
	beginAttachmentProfile(0)()
	if len(startupProbeEvents) != 0 {
		t.Fatal("zero ID unexpectedly recorded")
	}
}
