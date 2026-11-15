package main

import "strings"

// DCM has no document-readiness signal for the native PDF fallback: a handler
// can attach successfully yet display an error or password prompt. Do not send
// host sizing shortcuts or report sizing success merely because it attached.
// This deliberately also covers valid native PDFs until readiness is proven.
// The primary Windows.Data.Pdf renderer and provider-owned controls are unchanged.
func nativePreviewHostSizingAllowed(path string) bool {
	return strings.TrimSpace(path) != "" && !isPDFPath(path)
}
