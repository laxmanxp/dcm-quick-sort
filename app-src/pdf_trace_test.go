package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDCM34PDFTraceDisabled(t *testing.T) {
	t.Setenv("DCM_PDF_TRACE", "")
	if err := appendPDFTrace("", PDFTraceEvent{Event: "worker-started", Generation: 7}); err != nil {
		t.Fatal(err)
	}
	recordPDFTrace(PDFTraceEvent{Event: "worker-started", Generation: 7})
}
func TestDCM34PDFTraceErrors(t *testing.T) {
	for _, p := range []string{"relative.jsonl", filepath.Join(t.TempDir(), "missing", "trace.jsonl")} {
		if err := appendPDFTrace(p, PDFTraceEvent{Event: "worker-started"}); err == nil {
			t.Fatalf("expected trace error for %q", p)
		}
	}
	// Diagnostic failures must never propagate into the preview operation.
	t.Setenv("DCM_PDF_TRACE", t.TempDir())
	recordPDFTrace(PDFTraceEvent{Event: "worker-finished", Generation: 7})
}
func TestDCM34PDFTraceConcurrentRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	const n = 40
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(g uint64) {
			defer wg.Done()
			if err := appendPDFTrace(path, PDFTraceEvent{Event: "worker-finished", Generation: g, InputPath: "synthetic.pdf", RendererPID: 123, Failed: g%2 == 0}); err != nil {
				t.Error(err)
			}
		}(uint64(i + 1))
	}
	wg.Wait()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := 0
	var previous uint64
	for scanner.Scan() {
		var event PDFTraceEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if event.Sequence <= previous || event.HostPID != os.Getpid() || event.UTC == "" || event.Schema != 1 || event.Generation == 0 || event.Event != "worker-finished" {
			t.Fatalf("invalid event %+v", event)
		}
		previous = event.Sequence
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if count != n {
		t.Fatalf("records=%d", count)
	}
}
func TestDCM34PDFTraceBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "full.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(pdfTraceMaxBytes); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := appendPDFTrace(path, PDFTraceEvent{Event: "worker-finished"}); err == nil {
		t.Fatal("trace size cap missing")
	}
	s, _ := os.Stat(path)
	if s.Size() != pdfTraceMaxBytes {
		t.Fatal("full trace was modified")
	}
}
