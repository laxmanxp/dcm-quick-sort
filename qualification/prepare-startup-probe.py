"""Apply only the locally authored diagnostic wiring, preserving runtime policy."""
from pathlib import Path
import hashlib
ROOT=Path(__file__).resolve().parents[1]
def sha(b): return hashlib.sha256(b).hexdigest()
def edit(path,before,after,replacements):
 p=ROOT/path;b=p.read_bytes()
 if sha(b)==after:return
 assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()==before,(path,'source changed')
 s=b.decode()
 for old,new in replacements:
  assert s.count(old)==1,(path,old)
  s=s.replace(old,new)
 assert sha(s.encode())==after,(path,'output mismatch')
 p.write_bytes(s.encode())
for path,expected in {
 'app-src/startup_trace_windows.go':'9794bbc291e470f3fe1c93e261557e590fe8877ad76030fd4281ed1d92ce7a3e',
 'tests/native/dcm-startup-probe.cs':'5f0fcd2aebef1371e892adb93fa8fb57762a0d81e20105dd93372c5d3971a088',
 'tools/test-ui-sumatra-startup.ps1':'2876e842fe016527689fd39c4c2d7f577a368768d16ab67ed5c44f6ef988823b',
}.items(): assert sha((ROOT/path).read_bytes())==expected,(path,'authored source differs')
edit('app-src/main_windows.go','e99ed091dd2c5debe99c8610a3c61ab642f2bb05','c110cecc0406830c10c45912ff5b8ab1834c98263ebbf7221cb264e8911b8195',[
 ('case wmLayoutTraceFence:','case wmStartupProbe:\n\t\tstartupHeartbeat(hwnd, wParam)\n\t\treturn 0\n\tcase wmLayoutTraceFence:'),
 ('\tdefer flushLayoutTrace()','\tdefer flushStartupTrace()\n\tdefer flushLayoutTrace()')])
edit('app-src/sumatra_preview_windows.go','57e19f10c3f90580284eee3e0969fd47e51af09f','7337a449a24b97c54d6cccb2176590f10879bf2e4d7811cb73b3097e6d458269',[
 ('\tif !useSumatraProvider() {\n\t\treturn false\n\t}\n\tif err := stopSumatraPreview(); err != nil {','\tif !useSumatraProvider() {\n\t\treturn false\n\t}\n\tid := beginStartupTrace()\n\tdefer recordStartupTrace(id, "start-return")\n\tif err := stopSumatraPreview(); err != nil {'),
 ('\tdocument, err := validateSumatraDocument(path)','\trecordStartupTrace(id, "previous-released")\n\tdocument, err := validateSumatraDocument(path)'),
 ('\texe, err := validateSumatraRuntime(os.Getenv("DCM_SUMATRA_EXE"))','\trecordStartupTrace(id, "document-validated")\n\texe, err := validateSumatraRuntime(os.Getenv("DCM_SUMATRA_EXE"))'),
 ('\tprofile, err := os.MkdirTemp("", "dcm-sumatra-")','\trecordStartupTrace(id, "runtime-validated")\n\tprofile, err := os.MkdirTemp("", "dcm-sumatra-")'),
 ('\tvar pi syscall.ProcessInformation','\trecordStartupTrace(id, "profile-job-prepared")\n\tvar pi syscall.ProcessInformation'),
 ('\truntime.KeepAlive(command)','\truntime.KeepAlive(command)\n\trecordStartupTrace(id, "create-process-return")'),
 ('\tapp.fallback = "Loading PDF in Sumatra…"','\trecordStartupTrace(id, "process-resumed")\n\tapp.fallback = "Loading PDF in Sumatra…"'),
 ('\t\tif v.child != 0 {\n\t\t\tlayoutUI','\t\tif v.child != 0 {\n\t\t\trecordStartupTrace(startupProbeSequence, "child-attached")\n\t\t\tlayoutUI'),
 ('\t\t\tapp.fallback = ""','\t\t\trecordStartupTrace(startupProbeSequence, "attachment-layout-complete")\n\t\t\tapp.fallback = ""')])
edit('app-src/sumatra_document_windows.go','832acab847bdec9183a6fec42b35f5d9de5ed04b','2e1f2f091f710d7f39e1cd61d240aa1b3168dc6448de20589380e4aa80b1d8fa',[
 ('func updateSumatraStatus(v *sumatraSession, status string) {','func updateSumatraStatus(v *sumatraSession, status string) {\n\trecordStartupReady(status)')])
edit('tools/flowdev-dcm/config/checks.json','8cc9cfb52887027c0d3dabbf4ac8912ebc48d6e5','ef204c5b8a56d9f678ea37d2376e1cccddfbeeaef6767cc9322ab4d4e1ce19a1',[
 ('        "ui-sumatra-lifecycle":','        "ui-sumatra-startup": ["powershell.exe", "-NoProfile", "-NonInteractive", "-File", "tools/test-ui-sumatra-startup.ps1"],\n        "ui-sumatra-lifecycle":')])
print('Exact startup diagnostic source identities verified.')
