package main

import (
 "reflect"
 "testing"
)

func TestPDFiumRendererArgsAndResultContract(t *testing.T) {
 req:=PDFRenderRequest{InputPath:"C:/fixtures/sample.pdf",OutputPath:"C:/temp/preview.png",PageIndex:2,Zoom:125,ViewportWidth:800,ViewportHeight:600}
 want:=[]string{"-InputPath",req.InputPath,"-OutputPath",req.OutputPath,"-PageIndex","2","-Mode","zoom","-Zoom","125","-ViewportWidth","800","-ViewportHeight","600"}
 if got:=pdfiumRendererArgs(req);!reflect.DeepEqual(got,want){t.Fatalf("args=%v",got)}
 meta,err:=parsePDFRendererOutput("OK|300|2|1250|900|125")
 if err!=nil{t.Fatal(err)}
 if meta.PageCount!=300||meta.PageIndex!=2||meta.ScalePercent!=125{t.Fatalf("meta=%+v",meta)}
}

func TestPDFiumRendererFitModes(t *testing.T) {
 req:=PDFRenderRequest{Fit:true,FitWidth:false,Zoom:100}
 got:=pdfiumRendererArgs(req)
 if got[7]!="fit"{t.Fatalf("fit args %v",got)}
 req.FitWidth=true
 got=pdfiumRendererArgs(req)
 if got[7]!="fitwidth"{t.Fatalf("fitwidth args %v",got)}
}
