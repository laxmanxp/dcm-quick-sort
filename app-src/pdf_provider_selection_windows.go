//go:build windows

package main

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
)

const (
 pdfProviderWindows = "windows"
 pdfProviderPDFium  = "pdfium"
)

// selectPDFBackend leaves all release/untagged builds on Windows.Data.Pdf.
// An explicit PDFium selection remains explicit (missing worker fails closed).
// In a tagged development build, an absent bundled worker falls back safely
// to the pre-existing Windows preview rather than making all PDFs unavailable.
func selectPDFBackend(setting string, devDefault, workerReady bool) string {
 switch strings.ToLower(strings.TrimSpace(setting)) {
 case pdfProviderPDFium:
  return pdfProviderPDFium
 case "", "auto":
  if devDefault && workerReady {
   return pdfProviderPDFium
  }
 }
 // "windows" forces rollback; "sumatra" is still handled independently
 // by its existing provider policy. Unknown settings preserve legacy behavior.
 return pdfProviderWindows
}

func regularPDFWorkerFile(path string) (string, error) {
 if strings.TrimSpace(path) == "" {
  return "", errors.New("empty PDFium worker path")
 }
 absolute, err := filepath.Abs(path)
 if err != nil {
  return "", err
 }
 info, err := os.Stat(absolute)
 if err != nil {
  return "", fmt.Errorf("PDFium worker unavailable: %w", err)
 }
 if !info.Mode().IsRegular() {
  return "", errors.New("PDFium worker path must be a regular file")
 }
 return absolute, nil
}

// Candidate lookup is separate from build/env policy so packaging tests
// can verify the development-binary layout without altering the test binary.
func findBundledFlowPDFiumWorker(executable, runtimeRoot string) (string, error) {
 candidates := []string{
  filepath.Join(filepath.Dir(executable), "flowpdf-preview-worker-persistent.exe"),
 }
 if runtimeRoot = strings.TrimSpace(runtimeRoot); runtimeRoot != "" {
  candidates = append(candidates, filepath.Join(runtimeRoot, "pdfium", "bin", "flowpdf-preview-worker-persistent.exe"))
 }
 for _, candidate := range candidates {
  if valid, err := regularPDFWorkerFile(candidate); err == nil {
   return valid, nil
  }
 }
 return "", errors.New("bundled FlowPDF persistent worker not found")
}

// Bundled development builds locate the qualified persistent worker next
// to the candidate EXE. An explicitly supplied path always takes priority.
// The managed runtime directory is an optional secondary development location.
func flowPDFiumWorkerPath() (string, error) {
 if explicit := strings.TrimSpace(os.Getenv("DCM_FLOWPDF_WORKER")); explicit != "" {
  return regularPDFWorkerFile(explicit)
 }
 if !defaultPersistentPDFiumDevelopment {
  return "", errors.New("PDFium provider requires DCM_FLOWPDF_WORKER")
 }
 executable, err := os.Executable()
 if err != nil {
  return "", err
 }
 return findBundledFlowPDFiumWorker(executable, os.Getenv("FILEWORKSPACE_RUNTIME_DIR"))
}

func selectedPDFBackend() string {
 setting := os.Getenv("DCM_PDF_PROVIDER")
 switch strings.ToLower(strings.TrimSpace(setting)) {
 case pdfProviderPDFium:
  return pdfProviderPDFium
 case "", "auto":
  // Only tagged development builds perform worker discovery for defaults.
  if !defaultPersistentPDFiumDevelopment {
   return pdfProviderWindows
  }
  _, err := flowPDFiumWorkerPath()
  return selectPDFBackend(setting, true, err == nil)
 default:
  return pdfProviderWindows
 }
}

func usingFlowPDFiumProvider() bool { return selectedPDFBackend() == pdfProviderPDFium }

// Empty persistent setting follows the build default. Explicit 0/1 works
// in either build so both regression and rollback paths remain testable.
func pdfiumPersistentOptIn() bool {
 if !usingFlowPDFiumProvider() {
  return false
 }
 switch strings.ToLower(strings.TrimSpace(os.Getenv("DCM_FLOWPDF_PERSISTENT"))) {
 case "1", "true":
  return true
 case "0", "false":
  return false
 default:
  return defaultPersistentPDFiumDevelopment
 }
}
