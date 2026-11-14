"""Apply locally tested async startup edits; fail on changed inputs or outputs."""
from pathlib import Path
import hashlib,json,subprocess
R=Path(__file__).resolve().parents[1]
H=json.loads((R/'qualification/async-startup-hashes.json').read_text())
for n,v in H.items():
 p=R/n
 if v['before'] is not None:
  assert hashlib.sha256(p.read_bytes()).hexdigest()==v['before'],(n,'input changed')
def edit(n,changes):
 p=R/n;s=p.read_text()
 for a,b in changes:
  assert s.count(a)==1,(n,'replacement not unique',a[:80])
  s=s.replace(a,b)
 p.write_text(s)
p=R/'app-src/sumatra_preview_windows.go';s=p.read_text()
s=s.replace('\t"path/filepath"\n','').replace('\t"strings"\n','')
s=s.replace('process, job, child uintptr','process, job, child uintptr\n\tthread uintptr\n\ttraceID uint64')
a=s.index('// Startup returns immediately');b=s.index('\nfunc pollSumatraPreview()',a)
s=s[:a]+'''// Validation and process creation run off the UI thread. One pending worker
// owns the suspended process until the UI accepts its still-current request.
func startSumatraPreview(path string) bool {
 if !useSumatraProvider(){return false}
 id:=beginStartupTrace()
 defer recordStartupTrace(id,"start-return")
 if err:=stopSumatraPreview();err!=nil{sumatraError(err);return true}
 recordStartupTrace(id,"previous-released")
 if sumatraClosing{return true}
 if sumatraRetired!=nil{sumatraError(errors.New("previous PDF startup cleanup is not confirmed; retry Reload"));return true}
 if !ensureSumatraTimer(){sumatraError(errors.New("could not start viewer readiness timer"));return true}
 sumatraQueued=&sumatraLaunchSpec{path:path,runtime:os.Getenv("DCM_SUMATRA_EXE"),parent:app.previewHwnd,traceID:id}
 app.fallback="Loading PDF in Sumatra…"
 setText(app.statusLabel,"Loading PDF in restricted Sumatra preview…")
 procInvalidateRect.Call(app.previewHwnd,0,1)
 advanceSumatraPreparation()
 return true
}
'''+s[b:]
s=s.replace('func pollSumatraPreview() {\n\tv := sumatraView','func pollSumatraPreview() {\n\tadvanceSumatraPreparation()\n\tv := sumatraView')
s=s.replace('recordStartupTrace(startupProbeSequence,','recordStartupTrace(v.traceID,')
s=s.replace('func stopSumatraPreview() error {\n\tv := sumatraView','''func stopSumatraPreview() error {
 cancelSumatraPreparation()
 if sumatraRetired!=nil && sumatraPending==nil {
  if !ensureSumatraTimer(){return errors.New("cannot observe pending viewer cleanup")}
  retired:=sumatraRetired;sumatraRetired=nil
  sumatraPending=newPreparedResource(func(_ context.Context)(*sumatraSession,error){return retired,nil},disposePreparedSumatra)
  sumatraPending.Cancel()
 }
 if sumatraPending!=nil && sumatraPending.decided {
  if out,done:=sumatraPending.PollDone();done {
   sumatraPending=nil
   if out.err!=nil{sumatraRetired=out.value;return out.err}
  }
 }
 v := sumatraView''')
s=s.replace('import (','import (\n "context"')
s=s.replace('\tspKillTimer.Call(app.hwnd, sumatraTimerID)','\tif !sumatraPreparationBusy(){spKillTimer.Call(app.hwnd, sumatraTimerID)}')
s=s.replace('\tspClose.Call(v.process)\n\tspClose.Call(v.job)','\tif v.thread!=0{spClose.Call(v.thread);v.thread=0}\n\tspClose.Call(v.process)\n\tspClose.Call(v.job)')
a=s.index('func releaseSumatraForOperation()')
s=s[:a]+s[a:].replace('\treturn true','\tif sumatraPreparationBusy(){sumatraError(errors.New("PDF startup is being cancelled; retry the file operation after cleanup"));return false}\n\treturn true',1)
p.write_text(s)
edit('app-src/main_windows.go',[
 ('\t\tunloadPreview()\n\t\tprocDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)','\t\tif !closeSumatraPreview(){return 0}\n\t\tunloadPreview()\n\t\tprocDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)'),
 ('\tcase WM_COMMAND:\n\t\tid := int(loword(wParam))\n\t\tnotify := hiword(wParam)\n\t\tif notify == BN_CLICKED {\n\t\t\tswitch id {\n\t\t\tcase ID_SOURCE_BTN:', '\tcase WM_COMMAND:\n\t\tif sumatraClosing{return 0}\n\t\tid := int(loword(wParam))\n\t\tnotify := hiword(wParam)\n\t\tif notify == BN_CLICKED {\n\t\t\tswitch id {\n\t\t\tcase ID_SOURCE_BTN:')])
edit('app-src/startup_trace_windows.go',[
 ('\t"strings"','\t"strings"\n "sync"'),
 ('var startupProbeEvents =','var startupProbeEventsMu sync.Mutex\nvar startupProbeEvents ='),
 ('if !startupProbeEnabled || sequence == 0 || len(startupProbeEvents) >= 1024 {','if !startupProbeEnabled || sequence == 0 {\n return\n }\n startupProbeEventsMu.Lock()\n defer startupProbeEventsMu.Unlock()\n if len(startupProbeEvents) >= 1024 {'),
 ('func flushStartupTrace() {','func flushStartupTrace() {\n startupProbeEventsMu.Lock()\n defer startupProbeEventsMu.Unlock()'),
 ('func recordStartupReady(status string)','func recordStartupReady(sequence uint64,status string)'),
 ('startupReadySequence != startupProbeSequence','startupReadySequence != sequence'),
 ('startupReadySequence = startupProbeSequence','startupReadySequence = sequence'),
 ('recordStartupTrace(startupProbeSequence, "document-ready-status")','recordStartupTrace(sequence, "document-ready-status")'),
 ('uintptr(startupProbeSequence))','uintptr(sequence))')])
edit('app-src/sumatra_document_windows.go',[('recordStartupReady(status)','recordStartupReady(v.traceID,status)')])
edit('tools/flowdev-dcm/config/checks.json',[(
 '        "ui-sumatra-lifecycle":',
 '        "ui-sumatra-pending-close": ["powershell.exe", "-NoProfile", "-NonInteractive", "-File", "tools/test-ui-sumatra-lifecycle.ps1", "-CloseWhileLoading"],\n        "ui-sumatra-lifecycle":')])
p=R/'tools/test-ui-sumatra-lifecycle.ps1';s=p.read_text()
s=s.replace("[string]$Runtime=$env:DCM_SUMATRA_SOURCE_EXE)","[string]$Runtime=$env:DCM_SUMATRA_SOURCE_EXE,[switch]$CloseWhileLoading)")
s=s.replace("    $psi.EnvironmentVariables['DCM_PDF_PROVIDER']='sumatra';", "    if($CloseWhileLoading){$psi.EnvironmentVariables['DCM_STARTUP_TRACE']='1'}\n    $psi.EnvironmentVariables['DCM_PDF_PROVIDER']='sumatra';")
a=s.index("    $null=SelectFile '08-preview.pdf';$a=Viewer");b=s.index('    CheckFiles\n    $sw=',a)
s=s[:a]+"    if(-not $CloseWhileLoading){\n"+s[a:b]+"    }else{\n        $s=SelectFile '08-preview.pdf'\n        if($s -notlike 'Loading PDF*'){throw 'Pending-close test did not observe loading; no acceptance claimed.'}\n        $receipt['status_at_close_request']=$s\n        AddPass 'loading-observed-before-close-request'\n    }\n"+s[b:]
s=s.replace("    AddPass 'close-with-active-PDF-drains-viewers-profiles-and-file-handles'","    if($CloseWhileLoading){\n        $tracePath=Join-Path $lab 'dcm-startup-trace.json'\n        if(-not(Test-Path $tracePath)){throw 'Pending-close trace missing'}\n        $receipt['startup_trace']=Get-Content $tracePath -Raw|ConvertFrom-Json\n        $receipt['startup_trace_sha256']=Hash $tracePath\n        AddPass 'close-during-loading-drains-preparation-and-private-resources'\n    }else{AddPass 'close-with-active-PDF-drains-viewers-profiles-and-file-handles'}")
p.write_text(s)

subprocess.run(['gofmt','-w']+[str(R/n) for n in H if n.endswith('.go')],check=True)
for n,v in H.items():assert hashlib.sha256((R/n).read_bytes()).hexdigest()==v['after'],(n,'output mismatch')
print('All ten locally validated source/test identities match.')
