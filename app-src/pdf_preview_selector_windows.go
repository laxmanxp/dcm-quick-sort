//go:build windows

package main

import (
 "fmt"
 "os"
 "strings"
)

const (
 ID_PDF_PROVIDER_AUTO = 2010
 ID_PDF_PROVIDER_PDFIUM = 2011
 ID_PDF_PROVIDER_WINDOWS = 2012
)

func pdfPreviewChoice() string {
 switch strings.ToLower(strings.TrimSpace(os.Getenv("DCM_PDF_PROVIDER"))) {
 case "pdfium","windows": return strings.ToLower(strings.TrimSpace(os.Getenv("DCM_PDF_PROVIDER")))
 default: return "auto"
 }
}

func pdfPreviewChoiceName(choice string) string {
 switch choice {
 case "pdfium": return "PDFium (FlowPDF)"
 case "windows": return "Windows.Data.Pdf"
 default: return "Auto (development default)"
 }
}

func updatePDFProviderMenuChecks() {
 if app.pdfProviderMenu==0 {return}
 choice:=pdfPreviewChoice()
 for _,item:=range []struct{id uintptr;value string}{
  {ID_PDF_PROVIDER_AUTO,"auto"},
  {ID_PDF_PROVIDER_PDFIUM,"pdfium"},
  {ID_PDF_PROVIDER_WINDOWS,"windows"},
 } {
  flags:=uintptr(0)
  if choice==item.value {flags=MF_CHECKED}
  procCheckMenuItem.Call(app.pdfProviderMenu,item.id,flags)
 }
 procDrawMenuBar.Call(app.hwnd)
}

func validatePDFPreviewChoice(choice string) error {
 switch choice {
 case "auto","windows":
  return nil
 case "pdfium":
  if _,err:=flowPDFiumWorkerPath();err!=nil{return fmt.Errorf("FlowPDF worker is unavailable: %w",err)}
  if !developmentPDFiumRuntimeAvailable(){return fmt.Errorf("PDFium DLL runtime is unavailable")}
  return nil
 default:
  return fmt.Errorf("unknown PDF previewer: %s",choice)
 }
}

// Development-only picker. This is a session choice: it leaves user,
// machine and release settings untouched and reopens only the active PDF.
func choosePDFPreviewer(choice string) {
 if !defaultPersistentPDFiumDevelopment {return}
 if choice==pdfPreviewChoice(){return}
 if err:=validatePDFPreviewChoice(choice);err!=nil {
  showError("Cannot switch to "+pdfPreviewChoiceName(choice)+":\n"+err.Error())
  return
 }
 if pdfShutdown.closing {return}
 current:=currentFile()
 if current!="" && isPDFPath(current) {unloadPreview()}
 if err:=os.Setenv("DCM_PDF_PROVIDER",choice);err!=nil {
  showError("Cannot select PDF previewer: "+err.Error())
  return
 }
 updatePDFProviderMenuChecks()
 if current!="" && isPDFPath(current) {
  loadPreview(current)
 } else {
  setText(app.statusLabel,"PDF previewer selected: "+pdfPreviewChoiceName(choice)+" (session only)")
 }
}
