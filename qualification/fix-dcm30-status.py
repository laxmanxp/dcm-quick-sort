"""One-line native-provider status correction, authored and compiled locally."""
from pathlib import Path
import hashlib
p=Path(__file__).resolve().parents[1]/'app-src/main_windows.go';b=p.read_bytes()
after='a03318738a16e53147d3296352063a27ab8ded97353cfd54cb28dd22d9845005'
if hashlib.sha256(b).hexdigest()!=after:
 assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()=='9f2c5bcf2ea5b71d7143ad92992515e179927719'
 old="\t\tapp.previewActive = true\n\t\treturn\n\t}\n\n\t// Prefer Windows' native Office preview"
 new="\t\tapp.previewActive = true\n\t\tsetText(app.statusLabel, \"Windows preview: \"+strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), \".\")))\n\t\treturn\n\t}\n\n\t// Prefer Windows' native Office preview"
 s=b.decode();assert s.count(old)==1;s=s.replace(old,new)
 assert hashlib.sha256(s.encode()).hexdigest()==after
 p.write_bytes(s.encode())
print('Native preview status patch verified')
