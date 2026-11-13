from pathlib import Path
import hashlib
ROOT=Path(__file__).resolve().parents[1]
def edit(name,before,after,changes):
 p=ROOT/name;b=p.read_bytes();assert hashlib.sha256(b).hexdigest()==before,(name,'input changed')
 s=b.decode()
 for old,new in changes:
  assert s.count(old)==1,(name,'nonunique replacement')
  s=s.replace(old,new)
 assert hashlib.sha256(s.encode()).hexdigest()==after,(name,'output differs')
 p.write_bytes(s.encode())
edit('tools/test-ui-sumatra-lifecycle.ps1','1d4a065058f3a778ff4c65c94c09f3dda7063b701bdb4f101cf19cd3e4f46376','dbaec9cdedc969afde32b564bb8ab88133de22d11cfcf133d144d2f8783f30c4',[
 ("$null=SelectFile '10-empty.pdf';EmptyViewer\n    if((Status)","$null=SelectFile '10-empty.pdf'\n    Wait 'empty PDF validation result' {(Status) -like 'Sumatra preview unavailable:*nonempty regular file*'}\n    EmptyViewer\n    if((Status)"),
 ("$null=SelectFile '11-invalid.pdf';EmptyViewer\n    if((Status)","$null=SelectFile '11-invalid.pdf'\n    Wait 'invalid PDF validation result' {(Status) -like 'Sumatra preview unavailable:*no PDF header*'}\n    EmptyViewer\n    if((Status)")])
edit('app-src/sumatra_pending_windows_test.go','e7489368fbb8fe42983f0b8592b7f5b31ccf407c4b5b3d3a1b4f360562144716','d4e837ac186444932ed544620002be7ccce28e880d6b8b9ba939ff6502226c7b',[
 ('\truntime.LockOSThread()\n\tdefer runtime.UnlockOSThread()\n',''),
 ('\t\tt.Run(operation, func(t *testing.T) {','\t\tt.Run(operation, func(t *testing.T) {\n\t\t\truntime.LockOSThread()\n\t\t\tdefer runtime.UnlockOSThread()'),
 ('\t\t\tos.WriteFile(src, []byte("original bytes"), 0600)','\t\t\tif err := os.WriteFile(src, []byte("original bytes"), 0600); err != nil {\n\t\t\t\tt.Fatal(err)\n\t\t\t}')])
