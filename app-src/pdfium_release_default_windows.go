//go:build windows && !pdfium_dev

package main

// Normal/release builds retain the existing Windows.Data.Pdf default.
const defaultPersistentPDFiumDevelopment = false
