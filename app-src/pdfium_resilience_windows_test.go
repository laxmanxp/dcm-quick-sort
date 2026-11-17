//go:build windows

package main

import (
 "bytes"
 "context"
 "crypto/sha256"
 "fmt"
 "image/png"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

// This exercises the actual shared File Workspace PDFium worker on Windows.
// The fixtures and render outputs live in t.TempDir; no external PDFs are used.
func TestFlowPDFiumWorkerResilience(t *testing.T) {
 worker:=strings.TrimSpace(os.Getenv("DCM_FLOWPDF_WORKER"))
 if worker=="" {t.Skip("DCM_FLOWPDF_WORKER is not set")}
 worker,err:=filepath.Abs(worker);if err!=nil {t.Fatal(err)}
 if _,err:=os.Stat(worker);err!=nil {t.Fatal(err)}
 root:=t.TempDir()
 valid,err:=os.ReadFile(filepath.Join("..","tests","fixtures","sumatra-scroll.pdf"));if err!=nil {t.Fatal(err)}
 damaged,err:=os.ReadFile(filepath.Join("..","tests","fixtures","dcm34-damaged.pdf"));if err!=nil {t.Fatal(err)}
 encrypted:=dcmProtectedFixture(t)
 type fixture struct {name string;data []byte;shouldRender bool}
 fixtures:=[]fixture{
  {"valid.pdf",valid,true},
  {"damaged.pdf",damaged,false},
  {"protected.pdf",encrypted,false},
 }
 for _,f:=range fixtures {
  if err:=os.WriteFile(filepath.Join(root,f.name),f.data,0600);err!=nil {t.Fatal(err)}
 }
 originalHashes:=map[string][32]byte{}
 for _,f:=range fixtures {originalHashes[f.name]=sha256.Sum256(f.data)}
 invoke:=func(filename string,page int,zoom int,output string)(time.Duration,string,error){
  ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second)
  defer cancel()
  req:=PDFRenderRequest{InputPath:filepath.Join(root,filename),OutputPath:output,PageIndex:page,Zoom:zoom,ViewportWidth:800,ViewportHeight:600}
  cmd:=exec.CommandContext(ctx,worker,pdfiumRendererArgs(req)...)
  start:=time.Now()
  result,err:=cmd.CombinedOutput()
  elapsed:=time.Since(start)
  if ctx.Err()!=nil {t.Fatalf("%s renderer timed out after %v: %v",filename,elapsed,ctx.Err())}
  return elapsed,string(result),err
 }
 runValid:=func(label string,page int,zoom int) {
  t.Helper()
  output:=filepath.Join(root,label+".png")
  duration,stdout,err:=invoke("valid.pdf",page,zoom,output)
  if err!=nil {t.Fatalf("%s PDFium render failed: %v output=%q",label,err,stdout)}
  meta,err:=parsePDFRendererOutput(stdout)
  if err!=nil || meta.PageCount<2 || meta.PageIndex!=page {t.Fatalf("%s metadata %+v: %v",label,meta,err)}
  data,err:=os.ReadFile(output);if err!=nil {t.Fatal(err)}
  cfg,err:=png.DecodeConfig(bytes.NewReader(data))
  if err!=nil||cfg.Width!=meta.Width||cfg.Height!=meta.Height {t.Fatalf("%s PNG and metadata mismatch: %v",label,err)}
  if err:=os.Remove(output);err!=nil {t.Fatalf("unable to remove rendered PNG %s: %v",label,err)}
  t.Logf("%s: rendered page %d in %s, dimensions=%dx%d",label,page,duration,cfg.Width,cfg.Height)
 }
 runRefusal:=func(filename string) {
  t.Helper()
  out:=filepath.Join(root,filename+".png")
  duration,stdout,err:=invoke(filename,0,100,out)
  if err==nil {t.Fatalf("%s was incorrectly rendered: %q",filename,stdout)}
  if _,parseErr:=parsePDFRendererOutput(stdout);parseErr==nil {t.Fatalf("%s returned successful metadata: %q",filename,stdout)}
  if _,err:=os.Stat(out);!os.IsNotExist(err) {t.Fatalf("%s left an output artifact: %v",filename,err)}
  for _,f:=range fixtures {
   content,err:=os.ReadFile(filepath.Join(root,f.name));if err!=nil {t.Fatal(err)}
   if sha256.Sum256(content)!=originalHashes[f.name] {t.Fatalf("PDFium mutated input %s",f.name)}
   handle,err:=os.OpenFile(filepath.Join(root,f.name),os.O_RDWR,0)
   if err!=nil {t.Fatalf("%s retained input file lock: %v",f.name,err)}
   handle.Close()
  }
  t.Logf("%s: refusal in %s, diagnostic=%q",filename,duration,strings.TrimSpace(stdout))
 }
 runValid("before",0,100)
 runRefusal("damaged.pdf")
 runValid("after-damaged",1,125)
 runRefusal("protected.pdf")
 runValid("after-protected",0,100)
 for i:=0;i<10;i++ {runValid(fmt.Sprintf("switch-%02d",i),i%2,100+i*5)}
}

func TestFlowPDFiumWorkerLongDocument(t *testing.T){
 worker:=strings.TrimSpace(os.Getenv("DCM_FLOWPDF_WORKER"))
 longPDF:=strings.TrimSpace(os.Getenv("DCM_PDFIUM_LONG_PDF"))
 if worker==""||longPDF=="" {t.Skip("worker/long-PDF acceptance fixture not configured")}
 root:=t.TempDir()
 for _,page:=range []int{0,149,299,0} {
  output:=filepath.Join(root,fmt.Sprintf("page-%03d.png",page))
  ctx,cancel:=context.WithTimeout(context.Background(),25*time.Second)
  cmd:=exec.CommandContext(ctx,worker,pdfiumRendererArgs(PDFRenderRequest{
   InputPath:longPDF,OutputPath:output,PageIndex:page,Zoom:100,
   ViewportWidth:800,ViewportHeight:600,
  })...)
  start:=time.Now()
  result,err:=cmd.CombinedOutput()
  elapsed:=time.Since(start)
  timedOut:=ctx.Err()!=nil
  cancel()
  if timedOut {t.Fatalf("PDFium long-document render timeout on page %d",page)}
  if err!=nil {t.Fatalf("PDFium page %d: %v %s",page,err,result)}
  meta,err:=parsePDFRendererOutput(string(result))
  if err!=nil||meta.PageCount!=300||meta.PageIndex!=page {t.Fatalf("page %d: %+v err=%v",page,meta,err)}
  data,err:=os.ReadFile(output);if err!=nil{t.Fatal(err)}
  cfg,err:=png.DecodeConfig(bytes.NewReader(data))
  if err!=nil||cfg.Width!=meta.Width||cfg.Height!=meta.Height {t.Fatalf("page %d PNG mismatch: %v",page,err)}
  t.Logf("300-page document page %d: %s dimensions %dx%d",page,elapsed,cfg.Width,cfg.Height)
 }
}

func TestFlowPDFiumWorkerComplexPages(t *testing.T) {
 worker:=strings.TrimSpace(os.Getenv("DCM_FLOWPDF_WORKER"))
 raster:=strings.TrimSpace(os.Getenv("DCM_PDFIUM_RASTER_PDF"))
 vector:=strings.TrimSpace(os.Getenv("DCM_PDFIUM_VECTOR_PDF"))
 if worker==""||raster==""||vector==""{t.Skip("complex PDF fixtures are not configured")}
 root:=t.TempDir()
 for _,tc:=range []struct{name,path string;zoom int}{
  {"raster",raster,100},{"vector",vector,150},
 }{
  t.Run(tc.name,func(t *testing.T){
   input,err:=os.ReadFile(tc.path);if err!=nil{t.Fatal(err)}
   original:=sha256.Sum256(input)
   output:=filepath.Join(root,tc.name+".png")
   ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
   cmd:=exec.CommandContext(ctx,worker,pdfiumRendererArgs(PDFRenderRequest{
    InputPath:tc.path,OutputPath:output,PageIndex:0,Zoom:tc.zoom,
    ViewportWidth:900,ViewportHeight:700,
   })...)
   start:=time.Now();stdout,err:=cmd.CombinedOutput();elapsed:=time.Since(start)
   timeout:=ctx.Err()!=nil;cancel()
   if timeout{t.Fatalf("%s workload exceeded 30s deadline",tc.name)}
   if err!=nil{t.Fatalf("%s worker failed: %v %s",tc.name,err,stdout)}
   meta,err:=parsePDFRendererOutput(string(stdout))
   if err!=nil||meta.PageCount!=1||meta.PageIndex!=0 {t.Fatalf("%s metadata: %+v err=%v",tc.name,meta,err)}
   pngBytes,err:=os.ReadFile(output);if err!=nil{t.Fatal(err)}
   config,err:=png.DecodeConfig(bytes.NewReader(pngBytes))
   if err!=nil||config.Width!=meta.Width||config.Height!=meta.Height {t.Fatalf("%s PNG dimensions mismatch: %v",tc.name,err)}
   after,err:=os.ReadFile(tc.path);if err!=nil||sha256.Sum256(after)!=original{t.Fatal("worker changed input PDF")}
   if err:=os.Remove(output);err!=nil{t.Fatalf("output retained lock: %v",err)}
   t.Logf("%s page rendered in %v; png=%d bytes image=%dx%d",tc.name,elapsed,len(pngBytes),config.Width,config.Height)
  })
 }
}
