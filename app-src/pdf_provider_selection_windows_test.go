//go:build windows

package main

import (
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestPDFProviderDevelopmentPolicy(t *testing.T) {
 tests:=[]struct{
  name,setting string
  dev,worker bool
  want string
 }{
  {"release-with-worker","",false,true,pdfProviderWindows},
  {"release-without-worker","",false,false,pdfProviderWindows},
  {"dev-with-worker","",true,true,pdfProviderPDFium},
  {"dev-without-worker","",true,false,pdfProviderWindows},
  {"dev-auto","auto",true,true,pdfProviderPDFium},
  {"dev-auto-missing","auto",true,false,pdfProviderWindows},
  {"explicit-pdfium","pdfium",false,false,pdfProviderPDFium},
  {"force-windows","windows",true,true,pdfProviderWindows},
  {"sumatra-unchanged","sumatra",true,true,pdfProviderWindows},
  {"unknown-fallback","other",true,true,pdfProviderWindows},
 }
 for _,tc:=range tests {
  t.Run(tc.name,func(t *testing.T){
   got:=selectPDFBackend(tc.setting,tc.dev,tc.worker)
   if got!=tc.want {t.Fatalf("setting=%q dev=%v worker=%v got=%q want=%q",tc.setting,tc.dev,tc.worker,got,tc.want)}
  })
 }
}

func TestPDFProviderBundledWorkerDiscovery(t *testing.T) {
 root:=t.TempDir()
 candidate:=filepath.Join(root,"build","DCMQuickSort.exe")
 worker:=filepath.Join(root,"build","flowpdf-preview-worker-persistent.exe")
 runtimeWorker:=filepath.Join(root,"runtime","pdfium","bin","flowpdf-preview-worker-persistent.exe")
 if _,err:=findBundledFlowPDFiumWorker(candidate,root);err==nil{
  t.Fatal("missing worker unexpectedly detected")
 }
 if err:=os.MkdirAll(filepath.Dir(worker),0700);err!=nil {t.Fatal(err)}
 if err:=os.WriteFile(worker,[]byte("fixture"),0600);err!=nil {t.Fatal(err)}
 found,err:=findBundledFlowPDFiumWorker(candidate,root)
 if err!=nil||!strings.EqualFold(found,worker){t.Fatalf("colocated worker=%q err=%v",found,err)}
 if err:=os.Remove(worker);err!=nil{t.Fatal(err)}
 if err:=os.MkdirAll(filepath.Dir(runtimeWorker),0700);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(runtimeWorker,[]byte("fixture"),0600);err!=nil{t.Fatal(err)}
 found,err=findBundledFlowPDFiumWorker(candidate,filepath.Join(root,"runtime"))
 if err!=nil||!strings.EqualFold(found,runtimeWorker){t.Fatalf("managed worker=%q err=%v",found,err)}
}

func TestPDFProviderDefaultAndRollback(t *testing.T){
 root:=t.TempDir()
 worker:=filepath.Join(root,"worker.exe")
 if err:=os.WriteFile(worker,[]byte("fake worker"),0600);err!=nil{t.Fatal(err)}
 t.Setenv("DCM_FLOWPDF_WORKER",worker)
 // A dummy DLL is sufficient for provider selection; no DLL is loaded
 // by this policy test.
 library:=filepath.Join(root,"pdfium.dll")
 if err:=os.WriteFile(library,[]byte("dummy"),0600);err!=nil{t.Fatal(err)}
 t.Setenv("FILEWORKSPACE_PDFIUM",library)
 t.Setenv("DCM_PDF_PROVIDER","")
 t.Setenv("DCM_FLOWPDF_PERSISTENT","")
 if defaultPersistentPDFiumDevelopment {
  if !usingFlowPDFiumProvider() || !pdfiumPersistentOptIn() {
   t.Fatal("tagged development build did not select persistent PDFium")
  }
 }else{
  if usingFlowPDFiumProvider() || pdfiumPersistentOptIn() {
   t.Fatal("untagged release build changed the default")
  }
 }
 t.Setenv("DCM_PDF_PROVIDER","windows")
 if usingFlowPDFiumProvider() || pdfiumPersistentOptIn(){t.Fatal("explicit Windows rollback ignored")}
 cmd,err:=pdfRendererExecutable()
 if err!=nil||!(strings.HasSuffix(strings.ToLower(cmd),"powershell.exe")||strings.HasSuffix(strings.ToLower(cmd),"pwsh.exe")){
  t.Fatalf("rollback did not select Windows PowerShell: path=%q err=%v",cmd,err)
 }
 t.Setenv("DCM_PDF_PROVIDER","pdfium")
 t.Setenv("DCM_FLOWPDF_PERSISTENT","0")
 if !usingFlowPDFiumProvider() || pdfiumPersistentOptIn(){t.Fatal("one-shot PDFium override ignored")}
 t.Setenv("DCM_FLOWPDF_PERSISTENT","1")
 if !pdfiumPersistentOptIn(){t.Fatal("persistent override ignored")}
 t.Setenv("DCM_PDF_PROVIDER","sumatra")
 if usingFlowPDFiumProvider(){t.Fatal("Sumatra override was hijacked")}
}

func TestPDFProviderWorkerMissingAndExplicitFailClosed(t *testing.T){
 t.Setenv("DCM_FLOWPDF_WORKER",filepath.Join(t.TempDir(),"missing.exe"))
 t.Setenv("DCM_PDF_PROVIDER","")
 if usingFlowPDFiumProvider(){t.Fatal("auto-selected a missing PDFium worker")}
 t.Setenv("DCM_PDF_PROVIDER","pdfium")
 if _,err:=pdfRendererExecutable();err==nil{t.Fatal("explicit PDFium must reject missing worker")}
}

func TestPDFProviderRuntimeMustBePresentForAutomaticDefault(t *testing.T) {
 root:=t.TempDir()
 worker:=filepath.Join(root,"worker.exe")
 if err:=os.WriteFile(worker,[]byte("test worker"),0600);err!=nil{t.Fatal(err)}
 t.Setenv("DCM_FLOWPDF_WORKER",worker)
 t.Setenv("DCM_PDF_PROVIDER","auto")
 t.Setenv("FILEWORKSPACE_PDFIUM","")
 t.Setenv("FILEWORKSPACE_RUNTIME_DIR",filepath.Join(root,"missing-runtime"))
 if usingFlowPDFiumProvider(){t.Fatal("auto-selected PDFium without runtime DLL")}
 runtime:=filepath.Join(root,"runtime")
 if err:=os.MkdirAll(filepath.Join(runtime,"pdfium","bin"),0700);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(runtime,"pdfium","bin","pdfium.dll"),[]byte("dummy"),0600);err!=nil{t.Fatal(err)}
 t.Setenv("FILEWORKSPACE_RUNTIME_DIR",runtime)
 if defaultPersistentPDFiumDevelopment {
  if !usingFlowPDFiumProvider()||!pdfiumPersistentOptIn(){t.Fatal("tagged build did not select PDFium with managed DLL")}
 }else if usingFlowPDFiumProvider(){t.Fatal("release silently changed default")}
}

func TestPDFProviderDevelopmentWorkerDoesNotOverrideConfiguredRuntime(t *testing.T){
 t.Setenv("FILEWORKSPACE_PDFIUM",filepath.Join(t.TempDir(),"custom.dll"))
 t.Setenv("FILEWORKSPACE_RUNTIME_DIR","")
 if env:=developmentPDFiumWorkerEnvironment();env!=nil{t.Fatal("worker environment overrode configured explicit PDFium DLL")}
 t.Setenv("FILEWORKSPACE_PDFIUM","")
 t.Setenv("FILEWORKSPACE_RUNTIME_DIR",filepath.Join(t.TempDir(),"custom-runtime"))
 if env:=developmentPDFiumWorkerEnvironment();env!=nil{t.Fatal("worker environment overrode configured runtime directory")}
}
