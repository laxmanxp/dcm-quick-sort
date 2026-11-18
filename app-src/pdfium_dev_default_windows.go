//go:build windows && pdfium_dev

package main

// The pdfium_dev build tag enables persistent PDFium as the preferred
// development-preview backend. It is deliberately absent in release builds.
const defaultPersistentPDFiumDevelopment = true
