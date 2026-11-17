//go:build windows

package main

import (
 "bytes"
 "image/png"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestPDFiumProviderSelectionUsesExplicitWorker(t *testing.T){
 t.Setenv("DCM_PDF_PROVIDER","pdfium")
 t.Setenv("DCM_FLOWPDF_WORKER","")
 if _,err:=pdfRendererExecutable();err==nil {t.Fatal("missing worker should fail closed")}
 t.Setenv("DCM_FLOWPDF_WORKER",filepath.Join(t.TempDir(),"missing-worker.exe"))
 if _,err:=pdfRendererExecutable();err==nil {t.Fatal("unavailable worker should fail closed")}
 exe,err:=os.Executable();if err!=nil{t.Fatal(err)}
 t.Setenv("DCM_FLOWPDF_WORKER",exe)
 resolved,err:=pdfRendererExecutable()
 if err!=nil{t.Fatal(err)}
 if !strings.EqualFold(resolved,exe){t.Fatalf("worker path %q != %q",resolved,exe)}
 req:=PDFRenderRequest{InputPath:"sample.pdf",OutputPath:"render.png",Zoom:100,ViewportWidth:900,ViewportHeight:700}
 cmd:=pdfPreviewRenderCommand(resolved,"ignored.ps1",req)
 if len(cmd.Args)<2||cmd.Args[1]!="-InputPath"{t.Fatalf("not PDFium worker args: %v",cmd.Args)}
}

func TestSharedPDFiumWorkerSmoke(t *testing.T){
 worker:=strings.TrimSpace(os.Getenv("DCM_FLOWPDF_WORKER"))
 input:=strings.TrimSpace(os.Getenv("DCM_PDFIUM_TEST_PDF"))
 if worker==""||input=="" {t.Skip("set DCM_FLOWPDF_WORKER and DCM_PDFIUM_TEST_PDF for native integration acceptance")}
 t.Setenv("DCM_PDF_PROVIDER","pdfium")
 exe,err:=pdfRendererExecutable()
 if err!=nil{t.Fatal(err)}
 out:=filepath.Join(t.TempDir(),"preview.png")
 req:=PDFRenderRequest{InputPath:input,OutputPath:out,PageIndex:1,Zoom:120,ViewportWidth:800,ViewportHeight:600}
 cmd:=pdfPreviewRenderCommand(exe,"",req)
 result,err:=cmd.CombinedOutput()
 if err!=nil{t.Fatalf("PDFium worker failed: %v; %s",err,result)}
 meta,err:=parsePDFRendererOutput(string(result))
 if err!=nil{t.Fatal(err)}
 if meta.PageCount<2||meta.PageIndex!=1||meta.ScalePercent!=120 {t.Fatalf("unexpected response: %+v",meta)}
 pngData,err:=os.ReadFile(out)
 if err!=nil{t.Fatal(err)}
 cfg,err:=png.DecodeConfig(bytes.NewReader(pngData))
 if err!=nil{t.Fatal(err)}
 if cfg.Width!=meta.Width||cfg.Height!=meta.Height{t.Fatalf("PNG size %dx%d != metadata %+v",cfg.Width,cfg.Height,meta)}
}
