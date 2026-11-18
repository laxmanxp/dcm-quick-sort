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

func TestPDFRendererReportsEffectiveScaleAndSupportsFitWidth(t *testing.T) {
	meta, err := parsePDFRendererOutput("noise\nOK|12|0|612|792|43\n")
	if err != nil {
		t.Fatal(err)
	}
	if meta.ScalePercent != 43 {
		t.Fatalf("scale=%d; want 43", meta.ScalePercent)
	}
	args := strings.Join(pdfRendererArgs(`C:\\app\\pdf_render.ps1`, PDFRenderRequest{FitWidth: true, Zoom: 100, ViewportWidth: 900, ViewportHeight: 700}), " ")
	if !strings.Contains(args, "-Mode fitwidth") {
		t.Fatalf("fit-width args missing fitwidth mode: %s", args)
	}
}

func TestPDFRotationSequence(t *testing.T) {
 rotation:=0
 for _,want:=range []int{90,180,270,0,90}{
  rotation=nextPDFPreviewRotation(rotation)
  if rotation!=want {t.Fatalf("next clockwise rotation=%d want %d",rotation,want)}
 }
 if got:=normalizePDFPreviewRotation(-90);got!=270{t.Fatalf("normalized negative degrees=%d",got)}
 if got:=normalizePDFPreviewRotation(450);got!=90{t.Fatalf("normalized full circle=%d",got)}
}

func TestPDFRotationPropagatesToBothRendererCLIs(t *testing.T){
 req:=PDFRenderRequest{
  InputPath:"source.pdf",OutputPath:"preview.png",
  PageIndex:1,Rotation:270,Zoom:135,ViewportWidth:800,ViewportHeight:600,
 }
 native:=strings.Join(pdfiumRendererArgs(req)," ")
 legacy:=strings.Join(pdfRendererArgs("render.ps1",req)," ")
 for _,cli:=range []string{native,legacy} {
  for _,flag:=range []string{"-Rotation 270","-PageIndex 1","-Zoom 135"} {
   if !strings.Contains(cli,flag) {t.Fatalf("request lost %q: %s",flag,cli)}
  }
 }
 if joined:=strings.Join(pdfiumRendererArgs(PDFRenderRequest{Rotation:360})," ");!strings.Contains(joined,"-Rotation 0") {
  t.Fatalf("full-circle rotation was not reset: %s",joined)
 }
}
