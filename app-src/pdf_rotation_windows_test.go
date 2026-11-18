//go:build windows

package main

import (
 "bytes"
 "context"
 "image/png"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "syscall"
 "strconv"
 "testing"
 "time"
)

// Opt-in integration gate for the Windows.Data.Pdf rollback renderer.
// Runs with a disposable PDF fixture, not user documents. The default test
// suite skips this external Windows API/PowerShell round-trip.
func TestWindowsDataPDFRotationQuarterTurns(t *testing.T) {
 if strings.TrimSpace(os.Getenv("DCM_PDF_ROTATION_ACCEPTANCE"))!="1" {
  t.Skip("set DCM_PDF_ROTATION_ACCEPTANCE=1 for real WinRT rotation acceptance")
 }
 input:=strings.TrimSpace(os.Getenv("DCM_PDFIUM_TEST_PDF"))
 if input==""{t.Fatal("set DCM_PDFIUM_TEST_PDF to a disposable, non-square PDF fixture")}
 if _,err:=os.Stat(input);err!=nil{t.Fatal(err)}
 interpreter,err:=exec.LookPath("powershell.exe")
 if err!=nil{t.Fatal(err)}
 root:=t.TempDir()
 script:=filepath.Join(root,"pdf_render.ps1")
 if err:=os.WriteFile(script,bundledPDFRendererScript,0600);err!=nil{t.Fatal(err)}
 type dims struct{width,height int}
 sizes:=map[int]dims{}
 before,err:=os.ReadFile(input);if err!=nil{t.Fatal(err)}
 for _,degrees:=range []int{0,90,180,270}{
  output:=filepath.Join(root,fmtRotationName(degrees))
  req:=PDFRenderRequest{InputPath:input,OutputPath:output,PageIndex:0,
   Rotation:degrees,Zoom:100,ViewportWidth:900,ViewportHeight:700}
  ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
  cmd:=exec.CommandContext(ctx,interpreter,pdfRendererArgs(script,req)...)
  cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:createNoWindow}
  stdout,runErr:=cmd.CombinedOutput()
  timedOut:=ctx.Err()!=nil
  cancel()
  if timedOut{t.Fatalf("Windows.Data.Pdf rotation %d timed out",degrees)}
  if runErr!=nil{t.Fatalf("Windows.Data.Pdf rotation %d failed: %v %s",degrees,runErr,stdout)}
  meta,err:=parsePDFRendererOutput(string(stdout));if err!=nil{t.Fatal(err)}
  data,err:=os.ReadFile(output);if err!=nil{t.Fatal(err)}
  cfg,err:=png.DecodeConfig(bytes.NewReader(data));if err!=nil{t.Fatal(err)}
  if cfg.Width!=meta.Width||cfg.Height!=meta.Height {t.Fatalf("rotation %d PNG dimensions disagree with renderer: %+v %dx%d",degrees,meta,cfg.Width,cfg.Height)}
  sizes[degrees]=dims{cfg.Width,cfg.Height}
 }
 if sizes[0].width==sizes[0].height{t.Fatal("test requires a non-square document")}
 if sizes[90].width!=sizes[0].height||sizes[90].height!=sizes[0].width{
  t.Fatalf("Windows preview rotation did not swap dimensions: %+v",sizes)
 }
 if sizes[180]!=sizes[0] || sizes[270]!=sizes[90]{
  t.Fatalf("Windows preview quarter-turn output geometry differs: %+v",sizes)
 }
 after,err:=os.ReadFile(input);if err!=nil{t.Fatal(err)}
 if !bytes.Equal(before,after){t.Fatal("preview rotation changed the source PDF")}
 t.Logf("Windows.Data.Pdf quarter-turn previews verified: %+v",sizes)
}

func fmtRotationName(degrees int) string {
 return "preview-rotate-"+strconv.Itoa(degrees)+".png"
}
