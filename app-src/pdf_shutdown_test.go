package main

import "testing"

// These tests cover state transitions without requiring the Windows renderer.
// Only the GUI thread calls these methods; no synthetic worker timing is used.
func TestPDFShutdownCoordinatorSingleWorker(t *testing.T) {
	var s pdfShutdownCoordinator
	if deferClose, first := s.requestClose(); deferClose || first {
		t.Fatal("close should proceed when no renderer is outstanding")
	}
	if !s.workerStarted() {
		t.Fatal("first worker was rejected")
	}
	if deferClose, first := s.requestClose(); !deferClose || !first {
		t.Fatal("first close should defer for the live worker")
	}
	if deferClose, first := s.requestClose(); !deferClose || first {
		t.Fatal("repeat close must not retrigger invalidation")
	}
	if s.workerStarted() {
		t.Fatal("a new renderer started after close request")
	}
	if s.outstanding != 1 {
		t.Fatalf("unexpected outstanding count %d", s.outstanding)
	}
	if !s.resultConsumed() || s.outstanding != 0 || s.closing {
		t.Fatalf("last consumed result should resume close once: %+v", s)
	}
	if s.resultConsumed() {
		t.Fatal("duplicate completion resumed close twice")
	}
	if deferClose, first := s.requestClose(); deferClose || first {
		t.Fatal("final close still deferred after drain")
	}
}

func TestPDFShutdownCoordinatorOverlappingSelections(t *testing.T) {
	var s pdfShutdownCoordinator
	for i := 0; i < 3; i++ {
		if !s.workerStarted() {
			t.Fatal("worker rejected before close")
		}
	}
	if deferClose, first := s.requestClose(); !deferClose || !first {
		t.Fatal("close failed to defer while multiple selection workers remain")
	}
	for i := 2; i >= 1; i-- {
		if s.resultConsumed() || s.outstanding != i {
			t.Fatalf("closed before all obsolete results were consumed: %+v", s)
		}
	}
	if !s.resultConsumed() {
		t.Fatal("close not resumed by final result")
	}
	if s.outstanding != 0 || s.closing {
		t.Fatalf("final drain left state behind: %+v", s)
	}
}

func TestPDFShutdownCoordinatorNormalPreview(t *testing.T) {
	var s pdfShutdownCoordinator
	if !s.workerStarted() {
		t.Fatal("worker not started")
	}
	if s.resultConsumed() || s.outstanding != 0 {
		t.Fatal("ordinary preview completion should never request close")
	}
	if !s.workerStarted() || s.resultConsumed() {
		t.Fatal("ordinary subsequent render should still work")
	}
	if deferClose, first := s.requestClose(); deferClose || first {
		t.Fatal("no live work, so a close must not defer")
	}
}
