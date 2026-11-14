from pathlib import Path
import hashlib
root=Path(__file__).resolve().parents[1]
p=root/'tools/test-ui-docx.ps1';b=p.read_bytes();s=b.decode()
assert hashlib.sha256(b).hexdigest()=='2cb51112f0298b3c76b4bad0c7b075746897e26c0709e1424c6f9696c8040958'
old="function Preview {return [DcmTextProbe]::Preview($hwnd,$app.Id)}"
new="""function Preview {
 $nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id))
 if(@($nodes|Where-Object {$_.Visible -and $_.PID -ne $app.Id}).Count){return $null}
 $edits=@($nodes|Where-Object {$_.Visible -and $_.PID -eq $app.Id -and $_.Class -eq 'Edit'})
 if($edits.Count -gt 1){throw 'Multiple owned preview edits'}
 if($edits.Count -eq 1){return [pscustomobject]@{Handle=$edits[0].Handle;Text=[DcmNative]::Text([IntPtr]$edits[0].Handle,$app.Id)}}
 return $null
}"""
assert s.count(old)==1;s=s.replace(old,new)
old=" $nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$native=@($nodes|Where-Object {$_.Visible -and $_.PID -ne $app.Id}).Count -gt 0"
new=""" Wait 'native DOCX or completed fallback body' {
  $n=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id))
  if(@($n|Where-Object {$_.Visible -and $_.PID -ne $app.Id}).Count){return $true}
  $p=Preview;return ($null -ne $p -and $p.Text.StartsWith('DCM DOCX Preview'))
 }
 $nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$native=@($nodes|Where-Object {$_.Visible -and $_.PID -ne $app.Id}).Count -gt 0"""
assert s.count(old)==1;s=s.replace(old,new)
old=" Own;$nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$foreign=@($nodes|Where-Object {$_.Visible -and $_.PID -ne $app.Id})"
new=""" Wait 'visible native DOCX subtree' {@([DcmDocxProbe]::Snapshot($hwnd,$app.Id)|Where-Object {$_.Visible -and $_.PID -ne $app.Id}).Count -gt 0}
 Own;$nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$foreign=@($nodes|Where-Object {$_.Visible -and $_.PID -ne $app.Id})"""
assert s.count(old)==1;s=s.replace(old,new)
assert hashlib.sha256(s.encode()).hexdigest()=='2cbb94fb877aac4aa035985ad376bff3d88acfe2431bcca2e0624f8ec6fe4234'
p.write_bytes(s.encode());print('Native readiness output',hashlib.sha256(s.encode()).hexdigest())
