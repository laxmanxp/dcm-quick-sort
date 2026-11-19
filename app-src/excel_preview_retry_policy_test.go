package main

import "testing"

func TestDeferredExcelPreviewResizePolicy(t *testing.T) {
	cases := []struct { name string; hr uintptr; active, running, pending, want bool }{
		{"blank-no-handler", 0x8001010d, false, false, false, false},
		{"error-native-input-sync", 0x8001010d, true, false, false, true},
		{"success-no-retry", 0, true, false, false, false},
		{"other-error-no-retry", 0x80004005, true, false, false, false},
		{"retry-already-running", 0x8001010d, true, true, false, false},
		{"retry-already-posted", 0x8001010d, true, false, true, false},
	}
	for _, tt := range cases {t.Run(tt.name, func(t *testing.T) {
		if got := shouldPostExcelPreviewResize(tt.hr, tt.active, tt.running, tt.pending); got != tt.want {t.Fatalf("got %t want %t", got, tt.want)}
	})}
}

func TestDeferredExcelPreviewIdentity(t *testing.T) {
	cases := []struct {name string; pending, active bool; messageEpoch, currentEpoch uint64; messagePane, currentPane uintptr; want bool}{
		{"current",true,true,7,7,100,100,true},
		{"cleared",false,true,7,7,100,100,false},
		{"handler-unloaded",true,false,7,7,100,100,false},
		{"obsolete-generation",true,true,6,7,100,100,false},
		{"obsolete-pane",true,true,7,7,99,100,false},
	}
	for _, tt := range cases {t.Run(tt.name, func(t *testing.T){
		got := shouldRunExcelPreviewResizeRetry(tt.pending,tt.active,tt.messageEpoch,tt.currentEpoch,tt.messagePane,tt.currentPane)
		if got!=tt.want {t.Fatalf("got %t want %t",got,tt.want)}
	})}
}
