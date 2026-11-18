package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Explicitly opt-in diagnostic event stream; never changes rendering decisions,
// delays workers, records document contents, or creates parent directories.
// The caller supplies an absolute path in its disposable qualification report.
const pdfTraceMaxBytes int64 = 4 * 1024 * 1024

var pdfTraceMutex sync.Mutex
var pdfTraceSequence uint64

type PDFTraceEvent struct {
	Schema      int            `json:"schema"`
	Sequence    uint64         `json:"sequence"`
	UTC         string         `json:"utc"`
	HostPID     int            `json:"host_pid"`
	Event       string         `json:"event"`
	Generation  uint64         `json:"generation"`
	Rotation    int            `json:"rotation,omitempty"`
	InputPath   string         `json:"input_path"`
	OutputPath  string         `json:"output_path"`
	RendererPID int            `json:"renderer_pid"`
	Failed      bool           `json:"failed"`
	State       *PDFTraceState `json:"state,omitempty"`
}
type PDFTraceState struct {
	Generation       uint64 `json:"generation"`
	WorkerGeneration uint64 `json:"worker_generation"`
	Active           bool   `json:"active"`
	Busy             bool   `json:"busy"`
	Pending          bool   `json:"pending"`
	Path             string `json:"path"`
}

func recordPDFTrace(event PDFTraceEvent) { _ = appendPDFTrace(os.Getenv("DCM_PDF_TRACE"), event) }
func appendPDFTrace(path string, event PDFTraceEvent) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if !filepath.IsAbs(path) {
		return errors.New("PDF trace requires an absolute path")
	}
	pdfTraceMutex.Lock()
	defer pdfTraceMutex.Unlock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() >= pdfTraceMaxBytes {
		return errors.New("PDF trace is nonregular or full")
	}
	pdfTraceSequence++
	event.Schema = 1
	event.Sequence = pdfTraceSequence
	event.UTC = time.Now().UTC().Format(time.RFC3339Nano)
	event.HostPID = os.Getpid()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if info.Size()+int64(len(data))+1 > pdfTraceMaxBytes {
		return errors.New("PDF trace size limit")
	}
	_, err = f.Write(append(data, '\n'))
	return err
}
