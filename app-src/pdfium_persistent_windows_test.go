//go:build windows

package main

import (
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

// A helper launched as this test binary intercepts -ServeJSONL in init,
// before Go's test flag parser. It cannot access an actual PDF or the user UI.
// This tests that a stalled untrusted child has a bounded lifetime.
func init() {
 if os.Getenv("DCM_PDFIUM_TEST_HELPER")!="hang" {return}
 for _,arg:=range os.Args {
  if arg=="-ServeJSONL" {
   for {time.Sleep(time.Second)}
  }
 }
}

func TestPersistentPDFiumTimeoutReapsOnlyOwnedChildren(t *testing.T){
 executable,err:=os.Executable()
 if err!=nil{t.Fatal(err)}
 t.Setenv("DCM_PDFIUM_TEST_HELPER","hang")
 var childPIDs []int
 host:=&flowPDFiumProcess{}
 req:=PDFRenderRequest{InputPath:filepath.Join(t.TempDir(),"sample.pdf"),
  OutputPath:filepath.Join(t.TempDir(),"page.png"),
  PageIndex:0,Zoom:100,ViewportWidth:800,ViewportHeight:600,
  Generation:7}
 start:=time.Now()
 _,_,err=host.render(executable,req,100*time.Millisecond,func(pid int){childPIDs=append(childPIDs,pid)})
 if err==nil||!strings.Contains(err.Error(),"deadline") {
  t.Fatalf("stalled child must time out: %v",err)
 }
 if time.Since(start)>6*time.Second{t.Fatalf("timeout was not bounded")}
 if len(childPIDs)!=2||childPIDs[0]==childPIDs[1]{t.Fatalf("expected two supervised retries: %v",childPIDs)}
 if host.cmd!=nil {t.Fatal("timed-out child was not reaped")}
 host.close()
}

func TestPersistentPDFiumLiveRestartAndDecoderRefusal(t *testing.T){
 worker:=strings.TrimSpace(os.Getenv("DCM_FLOWPDF_WORKER"))
 source:=strings.TrimSpace(os.Getenv("DCM_PDFIUM_TEST_PDF"))
 if worker==""||source==""{t.Skip("persistent acceptance worker and fixture not configured")}
 if _,err:=os.Stat(worker);err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(source);err!=nil{t.Fatal(err)}
 root:=t.TempDir()
 h:=&flowPDFiumProcess{}
 defer h.close()
 render:=func(page int)(PDFRenderMetadata,int,error){
  output:=filepath.Join(root,fmt.Sprintf("page-%d-%d.png",page,time.Now().UnixNano()))
  req:=PDFRenderRequest{InputPath:source,OutputPath:output,PageIndex:page,
   Zoom:100,ViewportWidth:800,ViewportHeight:600,Generation:uint64(page+10)}
  result,pid,err:=h.render(worker,req,8*time.Second,nil)
  if err==nil {
   data,readErr:=os.ReadFile(output)
   if readErr!=nil||len(data)<16{t.Fatalf("missing output from %d: %v",pid,readErr)}
   if err:=os.Remove(output);err!=nil{t.Fatal(err)}
  }
  return result,pid,err
 }
 first,pid1,err:=render(0)
 if err!=nil{t.Fatalf("initial service render failed: %v",err)}
 _,pid2,err:=render(1)
 if err!=nil{t.Fatalf("second render failed: %v",err)}
 if pid1!=pid2{t.Fatalf("persistent renderer changed PID unexpectedly: %d %d",pid1,pid2)}
 if first.PageCount<2{t.Fatalf("invalid page count %+v",first)}
 // Decoder errors must be returned without restarting or killing the worker.
 locked:=filepath.Join(root,"protected.pdf")
 if err:=os.WriteFile(locked,dcmProtectedFixture(t),0600);err!=nil{t.Fatal(err)}
 _,lockedPID,err:=h.render(worker,PDFRenderRequest{
  InputPath:locked,OutputPath:filepath.Join(root,"protected.png"),
  PageIndex:0,Zoom:100,ViewportWidth:800,ViewportHeight:600,Generation:20,
 },8*time.Second,nil)
 if err==nil{t.Fatal("protected document unexpectedly rendered")}
 if lockedPID!=pid1{t.Fatalf("decoder refusal restarted worker: %d != %d",lockedPID,pid1)}
 if _,err:=os.Stat(filepath.Join(root,"protected.png"));!os.IsNotExist(err){t.Fatal("protected document left a PNG")}
 _,pid3,err:=render(0)
 if err!=nil||pid3!=pid1{t.Fatalf("worker did not recover after decoder refusal: %d %v",pid3,err)}
 // Simulate an actual worker crash between requests. The next request
 // should detect the closed pipe and restart the child automatically.
 h.mu.Lock()
 if h.cmd==nil{h.mu.Unlock();t.Fatal("worker not running")}
 if err:=h.cmd.Process.Kill();err!=nil{h.mu.Unlock();t.Fatal(err)}
 h.mu.Unlock()
 _,restarted,err:=render(1)
 if err!=nil{t.Fatalf("failed to recover from worker kill: %v",err)}
 if restarted==pid1{t.Fatalf("dead worker was reused: %d",pid1)}
 h.close()
 if h.cmd!=nil{t.Fatal("shutdown left an owned child")}
 t.Logf("worker PID reused %d through decoder refusal; restarted as %d after injected crash",pid1,restarted)
}

func TestPersistentPDFiumOptInIsStrict(t *testing.T) {
 t.Setenv("DCM_PDF_PROVIDER","pdfium")
 t.Setenv("DCM_FLOWPDF_PERSISTENT","1")
 if !pdfiumPersistentOptIn(){t.Fatal("opt-in unavailable")}
 t.Setenv("DCM_FLOWPDF_PERSISTENT","0")
 if pdfiumPersistentOptIn(){t.Fatal("disabled mode started persistent worker")}
 t.Setenv("DCM_PDF_PROVIDER","")
 t.Setenv("DCM_FLOWPDF_PERSISTENT","1")
 if pdfiumPersistentOptIn(){t.Fatal("non-PDFium provider started persistent worker")}
}

func TestPersistentPDFiumLegacyOneShotFallback(t *testing.T) {
 legacy:=strings.TrimSpace(os.Getenv("DCM_PDFIUM_ONESHOT_WORKER"))
 fixture:=strings.TrimSpace(os.Getenv("DCM_PDFIUM_TEST_PDF"))
 if legacy==""||fixture==""{t.Skip("legacy worker and fixture not configured")}
 if _,err:=os.Stat(legacy);err!=nil{t.Fatal(err)}
 host:=&flowPDFiumProcess{}
 defer host.close()
 output:=filepath.Join(t.TempDir(),"legacy-fallback.png")
 req:=PDFRenderRequest{InputPath:fixture,OutputPath:output,PageIndex:1,Zoom:100,
  ViewportWidth:800,ViewportHeight:600,Generation:31}
 _,_,err:=host.render(legacy,req,3*time.Second,nil)
 if err==nil{t.Fatal("older one-shot executable unexpectedly spoke persistent JSONL")}
 if host.cmd!=nil {t.Fatal("failed persistent negotiation left an owned worker")}
 meta,pid,err:=flowPDFiumOneShotFallback(legacy,req)
 if err!=nil{t.Fatalf("one-shot fallback failed: %v",err)}
 if pid<=0||meta.PageCount<2||meta.PageIndex!=1{t.Fatalf("invalid fallback result pid=%d meta=%+v",pid,meta)}
 data,err:=os.ReadFile(output)
 if err!=nil||len(data)<24{t.Fatalf("missing fallback PNG: %v",err)}
 if err:=os.Remove(output);err!=nil{t.Fatal(err)}
 t.Logf("unsupported persistent worker safely recovered through legacy one-shot renderer PID %d",pid)
}
