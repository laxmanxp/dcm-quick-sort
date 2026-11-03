. (Join-Path $PSScriptRoot 'common.ps1')
Add-Type @'
using System;using System.Runtime.InteropServices;
public static class QSWindowState {
 [DllImport("user32.dll")]public static extern bool ShowWindow(IntPtr h,int cmd);
 [DllImport("user32.dll")]public static extern bool SetWindowPos(IntPtr h,IntPtr after,int x,int y,int w,int height,uint flags);
}
'@
$wasTopmost=(([QSEvidence]::Style($hwnd,-20).ToInt64() -band 8) -ne 0)
try{
 [void][QSWindowState]::ShowWindow($hwnd,9)
 [void][QSWindowState]::SetWindowPos($hwnd,[IntPtr](-1),0,0,1280,820,0x0050)
 Start-Sleep -Milliseconds 300
 [void](Run-Case 'pptx-slide2-visible' @'
click clear
type name "07-slides"
wait_ready type=textbox target=newname text="07-slides" timeout=5s
delay 1000ms
select slides "Slide 2 · SECOND SLIDE"
delay 900ms
inspect main
expect slides text~="SECOND SLIDE"
'@ $null)
 [void](Run-Case 'pptx-slide1-return-visible' @'
select slides "Slide 1 · FILEFLOW OFFICE PREVIEW"
delay 900ms
inspect main
expect slides text~="FILEFLOW OFFICE PREVIEW"
'@ $null)
 [void](Run-Case 'xlsx-sheet2-visible' @'
click clear
type name "06-orders"
wait_ready type=textbox target=newname text="06-orders" timeout=5s
delay 900ms
select sheets "Read me"
delay 900ms
inspect main
expect sheets text="Read me"
'@ $null)
 [void](Run-Case 'pdf-reset-ready' @'
click clear
type name "08-print"
wait_ready type=textbox target=newname text="08-print" timeout=5s
delay 1200ms
click zoomreset
wait_ready type=control target=pdfstatus text~="100%" timeout=10s
inspect main
expect pdfstatus text~="100%"
'@ $null)
 [void](Run-Case 'pdf-plus-ready' @'
click zoomin
wait_ready type=control target=pdfstatus text~="110%" timeout=10s
inspect main
expect pdfstatus text~="110%"
'@ $null)
 [void](Run-Case 'pdf-minus-ready' @'
click zoomout
wait_ready type=control target=pdfstatus text~="100%" timeout=10s
inspect main
expect pdfstatus text~="100%"
'@ $null)
 [void](Run-Case 'pdf-width-ready' @'
click width
wait_ready type=control target=pdfstatus text~="Fit Width" timeout=10s
inspect main
expect pdfstatus text~="Fit Width"
'@ $null)
}finally{if(-not $wasTopmost){[void][QSWindowState]::SetWindowPos($hwnd,[IntPtr](-2),0,0,0,0,0x0013)}}
Write-Output ('REPORT='+$report)
