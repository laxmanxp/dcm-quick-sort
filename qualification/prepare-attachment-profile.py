"""Exact opt-in diagnostic wiring; retain every existing window operation."""
from pathlib import Path
import hashlib,re,subprocess
ROOT=Path(__file__).resolve().parents[1]
def digest(b):return hashlib.sha256(b).hexdigest()
expected = {
 'app-src/main_windows.go':('f826ef012db247c306bf7aa8af5d9b0cc1651b7f','7158979c7e865f6dd34c2df82d789b66615b1c6436c77c74de358b71b0b0b8b0'),
 'app-src/sumatra_preview_windows.go':('3b7041b3115ecbebd0c0f247f8a6e52d05aad93c','17d9e0b65808681e51847bfb310b9aacaedb016aef354d00e1c896b525ebb50f'),
}
REPLACEMENTS = [['\t\tspEnumChildren.Call(app.previewHwnd, sumatraFindChild, 0)\n\t\tif v.child != 0 {\n\t\t\trecordStartupTrace(v.traceID, "child-attached")\n\t\t\tlayoutUI(app.hwnd)\n\t\t\tresizeSumatraPreview()\n\t\t\trecordStartupTrace(v.traceID, "attachment-layout-complete")\n\t\t\tapp.fallback = ""\n\t\t\tupdateSumatraStatus(v, sumatraCurrentDocumentStatus(v))\n\t\t\tupdatePreviewZoomControls()\n\t\t\tprocInvalidateRect.Call(app.previewHwnd, 0, 1)\n\t\t} else if time.Now().After(v.deadline) {\n\t\t\t_ = stopSumatraPreview()\n\t\t\tsumatraError(errors.New("Sumatra did not attach within the startup deadline"))\n\t\t}\n\t}\n\tif sumatraView == v && v.child != 0 {\n\t\trepairSumatraToolbar(v)\n\t\tupdateSumatraStatus(v, sumatraCurrentDocumentStatus(v))\n\t}\n}\nfunc resizeSumatraPreview() {\n', '\t\tspEnumChildren.Call(app.previewHwnd, sumatraFindChild, 0)\n\t\tif v.child != 0 {\n\t\t\trecordStartupTrace(v.traceID, "child-attached")\n\t\t\tendAttachmentProfile := beginAttachmentProfile(v.traceID)\n\t\t\tdefer endAttachmentProfile()\n\t\t\tlayoutUI(app.hwnd)\n\t\t\trecordAttachmentTrace("layout-complete-before-viewer-resize")\n\t\t\tresizeSumatraPreview()\n\t\t\trecordStartupTrace(v.traceID, "attachment-layout-complete")\n\t\t\trecordAttachmentTrace("viewer-resize-complete")\n\t\t\tapp.fallback = ""\n\t\t\trecordAttachmentTrace("initial-status-begin")\n\t\t\tupdateSumatraStatus(v, sumatraCurrentDocumentStatus(v))\n\t\t\trecordAttachmentTrace("initial-status-end")\n\t\t\tupdatePreviewZoomControls()\n\t\t\trecordAttachmentTrace("host-zoom-controls-end")\n\t\t\tprocInvalidateRect.Call(app.previewHwnd, 0, 1)\n\t\t\trecordAttachmentTrace("preview-invalidation-end")\n\t\t} else if time.Now().After(v.deadline) {\n\t\t\t_ = stopSumatraPreview()\n\t\t\tsumatraError(errors.New("Sumatra did not attach within the startup deadline"))\n\t\t}\n\t}\n\tif sumatraView == v && v.child != 0 {\n\t\trecordAttachmentTrace("toolbar-repair-begin")\n\t\trepairSumatraToolbar(v)\n\t\trecordAttachmentTrace("toolbar-repair-end")\n\t\tupdateSumatraStatus(v, sumatraCurrentDocumentStatus(v))\n\t\trecordAttachmentTrace("final-status-end")\n\t}\n}\nfunc resizeSumatraPreview() {\n']]
for name,(before,after) in expected.items():
 p=ROOT/name;raw=p.read_bytes()
 if digest(raw)==after:continue
 assert hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest()==before,(name,'input changed')
 s=raw.decode()
 if name.endswith('main_windows.go'):
  a=s.index('func layoutUI(hwnd uintptr) {');b=s.index('\nfunc populateFileList()',a)
  out=[];count=0
  for line in s[a:b].splitlines():
   m=re.match(r'\tprocMoveWindow.Call\(app\.(\w+),',line)
   if m:
    count+=1;label=m.group(1)
    out.extend(['\trecordAttachmentTrace("layout-before-'+label+'")',line,'\trecordAttachmentTrace("layout-after-'+label+'")'])
   else:out.append(line)
  assert count==49
  chunk='\n'.join(out)+'\n'
  line='\tprocRedrawWindow.Call(hwnd, 0, 0, 0x0001|0x0004|0x0080|0x0400)'
  assert chunk.count(line)==1
  chunk=chunk.replace(line,'\trecordAttachmentTrace("layout-before-invalidation")\n'+line+'\n\trecordAttachmentTrace("layout-after-invalidation")')
  s=s[:a]+chunk+s[b:]
 else:
  for old,new in REPLACEMENTS:
   assert s.count(old)==1,(name,'nonunique edit')
   s=s.replace(old,new)
 p.write_bytes(s.encode())
 subprocess.run(['gofmt','-w',str(p)],check=True)
 assert digest(p.read_bytes())==after,(name,'output differs')
 print(name,after)
