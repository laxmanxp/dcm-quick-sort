"""Replay locally authored test-adapter changes; no application changes."""
from pathlib import Path
import hashlib
root=Path(__file__).resolve().parents[1]
p=root/'tools/test-ui-docx.ps1';b=p.read_bytes()
after='2cb51112f0298b3c76b4bad0c7b075746897e26c0709e1424c6f9696c8040958'
if hashlib.sha256(b).hexdigest()!=after:
 assert hashlib.sha256(b).hexdigest()=='4b55eabe42aea78890e17526b4322f1ca9409a825a2f9b40b2cb604e67cae392','source changed'
 s=b.decode()
 def change(old,new):
  global s
  assert s.count(old)==1,old
  s=s.replace(old,new)
 change('# The text assertions qualify the observed Open XML fallback, not Word fidelity.','# Native handler content requires separate screenshot review; fallback text is asserted.')
 change('function Exclusive', '''function NativeShot([string]$name){
 Own;$nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$foreign=@($nodes|Where-Object {$_.Visible -and $_.PID -ne $app.Id})
 if(-not $foreign.Count){throw 'Expected embedded native DOCX subtree missing'}
 Json (Join-Path $lab ($name+'.native-windows.json')) $nodes
 $identities=@($foreign|Select-Object -ExpandProperty PID -Unique|ForEach-Object {$p=Get-Process -Id $_;[ordered]@{pid=$p.Id;path=$p.Path;start=$p.StartTime.ToUniversalTime().ToString('o')}})
 $text=[DcmDocxProbe]::ReadAccessibleText($hwnd,$app.Id);[IO.File]::WriteAllText((Join-Path $lab ($name+'.accessible.txt')),$text,$utf8)
 Start-Sleep -Milliseconds 500;Shot $name
 $captions=@([DcmDocxProbe]::HostStatics($hwnd,$app.Id)|ForEach-Object {[DcmNative]::Text([IntPtr]$_,$app.Id)})
 $receipt['last_native_status_captions']=$captions
 if(-not ($captions -ccontains 'Windows preview: DOCX')){throw 'DCM30_STALE_PROVIDER_STATUS: native DOCX preview is active but its provider/type is not identified'}
 Record $name ([ordered]@{native_identities=$identities;accessible_text_sha256=(Hash (Join-Path $lab ($name+'.accessible.txt')));rendered_content_review='required_separately';status='Windows preview: DOCX'})
}
function Exclusive''')
 change("Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs')", '''Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs')
 Add-Type -AssemblyName UIAutomationClient;Add-Type -AssemblyName UIAutomationTypes;Add-Type -AssemblyName WindowsBase
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-docx-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll',([Windows.Automation.AutomationElement]).Assembly.Location,([Windows.Automation.AutomationPattern]).Assembly.Location,([Windows.Point]).Assembly.Location)''')
 change("$receipt['historical_currency_note']='The old fixture stores a literal question mark before 25,000; this is not an inferred rendering defect.'", "$receipt['historical_currency_note']='Console text conversion is not used as evidence of the currency glyph; the original fixture bytes are pinned.'")
 change(" SelectFile '05-letter.docx'\n $letterTokens=", """ SelectFile '05-letter.docx'
 $nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$native=@($nodes|Where-Object {$_.Visible -and $_.PID -ne $app.Id}).Count -gt 0
 if($native){
  $receipt.provider='Installed Windows DOCX preview handler';$receipt['native_visual_review_required']=$true
  NativeShot '02-native-letter'
  SelectFile '06-structured.DOCX';NativeShot '04-native-structured'
  SelectFile '01-alpha.txt';$p=ExpectBody '08-text-recovery' @($plain) @('DCM-30 FIELD REPORT');Shot '08-text-recovery'
  if(@([DcmDocxProbe]::Snapshot($hwnd,$app.Id)|Where-Object {$_.Visible -and $_.PID -ne $app.Id}).Count){throw 'Native viewer retained after text selection'}
  Exclusive '05-letter.docx';Exclusive '06-structured.DOCX';Record '09-both-docx-files-released' $null
  SelectFile '05-letter.docx';NativeShot '10-native-letter-reopened'
 }else{
 $letterTokens=""")
 change("'Sanctioned amount: ?25,000'", "('Sanctioned amount: '+[char]0x20B9+'25,000')")
 change(' Files;StopApp;Source;Exclusive', ' }\n Files;StopApp;Source;Exclusive')
 assert hashlib.sha256(s.encode()).hexdigest()==after,'output differs'
 p.write_bytes(s.encode())
print('Exact native test adapter patch verified')
