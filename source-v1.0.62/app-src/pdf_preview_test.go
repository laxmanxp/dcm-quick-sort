package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestPDFPathDetection(t *testing.T) {
	if !isPDFPath(`C:\Docs\Report.PDF`) || isPDFPath(`C:\Docs\Report.docx`) {
		t.Fatal("PDF path detection failed")
	}
}

func TestPDFRendererOutputParsing(t *testing.T) {
	got, err := parsePDFRendererOutput("noise\nOK|12|2|816|1056\n")
	if err != nil {
		t.Fatal(err)
	}
	want := PDFRenderMetadata{PageCount: 12, PageIndex: 2, Width: 816, Height: 1056}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestPDFRendererArgsOwnZoomAndFit(t *testing.T) {
	zoom := pdfRendererArgs(`C:\app\pdf_render.ps1`, PDFRenderRequest{InputPath: `C:\in.pdf`, OutputPath: `C:\out.png`, PageIndex: 3, Zoom: 150, ViewportWidth: 900, ViewportHeight: 700})
	joined := strings.Join(zoom, " ")
	for _, want := range []string{"-Mode zoom", "-Zoom 150", "-PageIndex 3"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("zoom args missing %q: %s", want, joined)
		}
	}
	fit := strings.Join(pdfRendererArgs(`C:\app\pdf_render.ps1`, PDFRenderRequest{Fit: true, Zoom: 100, ViewportWidth: 900, ViewportHeight: 700}), " ")
	if !strings.Contains(fit, "-Mode fit") {
		t.Fatalf("fit args missing fit mode: %s", fit)
	}
}

func TestBundledPDFRendererUsesWindowsDataPdf(t *testing.T) {
	data, err := os.ReadFile("assets/pdf_render.ps1")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, want := range []string{"Windows.Data.Pdf.PdfDocument", "PdfPageRenderOptions", "DestinationWidth", "RenderToStreamAsync"} {
		if !strings.Contains(src, want) {
			t.Fatalf("PDF renderer script missing %q", want)
		}
	}
}
