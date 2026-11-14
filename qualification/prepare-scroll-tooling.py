"""Apply locally validated, exact-hash tooling registration and fixture bytes."""
from pathlib import Path
import hashlib, runpy
root=Path(__file__).resolve().parents[1]
changes=[('tools/flowdev-dcm/config/checks.json','9a65b26a6d159b1393b5e638c513260b72986320','78deeb026c149f9ebbdb73760e24ad14f2362baff37cd5e5707a17cf4a365cab','        "ui-sumatra-lifecycle":','        "ui-sumatra-scroll": ["powershell.exe", "-NoProfile", "-NonInteractive", "-File", "tools/test-ui-sumatra-scroll.ps1"],\n        "ui-sumatra-lifecycle":'),('tools/test-ui-tooling.ps1','3b9acb61c5b53cec2a0c6bc8b6e5423eaef1598b','ffcb16d48fe4a5f248620a62baa180317fe7779f781ff8198db2f38e1a0df350',"    $receipt['native_adapter_compiled']=$true","    Add-Type -Path (Join-Path $repo 'tests/native/dcm-scroll-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')\n    $receipt['scroll_probe_compiled']=$true\n    $receipt['native_adapter_compiled']=$true")]
for name,before,after,old,new in changes:
 p=root/name;raw=p.read_bytes()
 assert hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest()==before,(name,'input changed')
 s=raw.decode();assert s.count(old)==1;s=s.replace(old,new)
 assert hashlib.sha256(s.encode()).hexdigest()==after,(name,'output mismatch')
 p.write_bytes(s.encode())
runpy.run_path(str(root/'tests/fixtures/generate-scroll-fixture.py'))
assert hashlib.sha256((root/'tests/fixtures/sumatra-scroll.pdf').read_bytes()).hexdigest()=='4196a31053e64ddeaebd30bb866e2e7a3e05dcf1a9dc260ba627882371893cc3'
