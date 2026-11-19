from pathlib import Path
import hashlib
R=Path(__file__).resolve().parents[1]
def edit(name,before,after,changes):
 p=R/name;b=p.read_bytes();assert hashlib.sha256(b).hexdigest()==before,(name,'input changed')
 s=b.decode()
 for a,z in changes:
  assert s.count(a)==1,(name,'replacement not unique')
  s=s.replace(a,z)
 assert hashlib.sha256(s.encode()).hexdigest()==after,(name,'output differs')
 p.write_bytes(s.encode())
edit('tests/native/dcm-sumatra-probe.cs','ba0d5b52b809eade26d445110e67f2493f4657a52412fd8f7bdbc3cbd63b8898','6f7e1924ed92085eacbf2f3cf7880c0dbc190d8ad1c4a88186f185de51def628',[
 ('    public static void CheckGeometry(IntPtr host,int hostPID,int expectedPID){','    [DllImport("user32.dll")] static extern bool IsIconic(IntPtr h);\n    public static void CheckGeometry(IntPtr host,int hostPID,int expectedPID){'),
 ('throw new Exception("Embedded viewer escaped preview bounds");','throw new Exception(String.Format("Embedded viewer escaped preview bounds: host_minimized={0}; host_visible={1}; panel=({2},{3},{4},{5}); viewer=({6},{7},{8},{9}); viewer_pid={10}",IsIconic(host),IsWindowVisible(host),p.L,p.T,p.R,p.B,c.L,c.T,c.R,c.B,expectedPID));')])
edit('tools/test-ui-sumatra-startup.ps1','5d221ccd402bf6ca95b569370ceae35d14535d5497cba98eba2d76810dbc1b1e','21b8bb89bc5a09421086384cb7f812d3688b4919ce198b13bed64952d2cff4c8',[
 ("}catch{$receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)}", """}catch{
 $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)
 if($app -and -not $app.HasExited){try{Own;$s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab 'failure.png'),$false);$shots.Add([ordered]@{name='failure';path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false})}catch{$receipt['failure_capture_error']=$_.Exception.Message}}
}""")])
