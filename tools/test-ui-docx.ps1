[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,[string]$Letter='D:\FlowverseReportsRepo\dcm-flowuivision-20261103\inbox\05-letter.docx')
# DCM-30. Natural installed-handler/fallback selection; never alter registrations.
# The text assertions qualify the observed Open XML fallback, not Word fidelity.
$ErrorActionPreference='Stop';Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required'}
if(-not $BuildReceipt -or -not $env:FLOWDEV_WORKSPACE_ROOT){throw 'Exact build and workspace required'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-docx-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Test lab must be outside source'}
$null=New-Item -ItemType Directory $lab;$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-30';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;cases=$cases;captures=$shots;user_test='not_recorded';provider='not_observed'}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$head='';$manifest=@();$fixture=$null;$expected=@{};$exe=''
function Hash([string]$p){return (Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Json([string]$p,$v){[IO.File]::WriteAllText($p,($v|ConvertTo-Json -Depth 25),$utf8)}
function Git([string[]]$arguments){$s=(& git.exe -C $repo @arguments|Out-String).Trim();if($LASTEXITCODE){throw 'Git query failed'};return $s}
function Source {
 if((Git @('rev-parse','HEAD')) -ne $head -or (Git @('status','--porcelain=v1','--untracked-files=all'))){throw 'Source identity changed'}
 foreach($f in $manifest){$p=[IO.Path]::GetFullPath((Join-Path $repo $f.path));if(-not $p.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $p) -ne $f.sha256){throw 'Source manifest changed'}}
}
function Own {$app.Refresh();if($app.HasExited -or $app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $owned.start){throw 'Owned host identity changed'};[DcmNative]::RequireOwned($hwnd,$app.Id)}
function Wait([string]$name,[scriptblock]$predicate){$until=[DateTime]::UtcNow.AddSeconds(20);do{Own;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw ('Unexpected owned dialog: '+$name)};if(& $predicate){return};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until);throw ('Deadline: '+$name)}
function Files {
 if(-not $fixture){return};$all=@(Get-ChildItem $fixture.inbox -File)
 if($all.Count -ne $expected.Count){throw 'Fixture names changed'}
 foreach($f in $all){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Fixture content changed'}}
 if(@(Get-ChildItem $fixture.destination -Force).Count){throw 'Destination changed'}
 $p=Join-Path $fixture.config 'recovery\move-journal.jsonl';if((Test-Path $p) -and ([IO.File]::ReadAllText($p)).Trim()){throw 'Unexpected operation journal'}
}
function Shot([string]$name){Own;$screen=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea;[DcmNative]::Restore($hwnd,$app.Id,$screen.Left+8,$screen.Top+8);Start-Sleep -Milliseconds 300;$s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false);$shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false})}
function Record([string]$name,$detail){$cases.Add([ordered]@{name=$name;passed=$true;detail=$detail});Files}
function SelectFile([string]$name){$items=@([DcmNative]::ListItems($hwnd,$app.Id,1016));$index=[Array]::IndexOf($items,$name);if($index -lt 0){throw 'Missing fixture in list'};[DcmUndoInput]::Select($hwnd,$app.Id,$index);$stem=[IO.Path]::GetFileNameWithoutExtension($name);Wait ('selection '+$name) {([DcmUndoInput]::TryText($hwnd,$app.Id,1005)) -ceq $stem}}
function Preview {return [DcmTextProbe]::Preview($hwnd,$app.Id)}
function ExpectBody([string]$name,[string[]]$tokens,[string[]]$absent){
 Wait ('body '+$name) {$p=Preview;($null -ne $p) -and $p.Text.Contains($tokens[0])}
 $p=Preview;$last=-1;foreach($token in $tokens){$at=$p.Text.IndexOf($token,$last+1,[StringComparison]::Ordinal);if($at -lt 0 -or $at -lt $last){throw ('Missing/out-of-order DOCX content: '+$token)};$last=$at}
 foreach($token in $absent){if($p.Text.Contains($token)){throw ('Stale or wrong document content: '+$token)}}
 [IO.File]::WriteAllText((Join-Path $lab ($name+'.displayed.txt')),$p.Text,$utf8)
 Record $name ([ordered]@{displayed_characters=$p.Text.Length;displayed_sha256=(Hash (Join-Path $lab ($name+'.displayed.txt')));ordered_tokens=$tokens})
 return $p
}
function CheckStatus([string]$prefix){$found=@([DcmTextProbe]::Controls($hwnd,$app.Id)|Where-Object {$_.Visible -and $_.Class -eq 'Static' -and $_.Text.StartsWith($prefix,[StringComparison]::Ordinal)});if($found.Count -ne 1){throw ('DOCX provider/status missing: '+$prefix)};return $found[0].Text}
function Exclusive([string]$name){$h=[IO.File]::Open((Join-Path $fixture.inbox $name),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()}
function StopApp {
 if(-not $app){return};Own
 $children=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$app.Id)|ForEach-Object {[ordered]@{pid=[int]$_.ProcessId;path=$_.ExecutablePath;created=$_.CreationDate.ToUniversalTime().ToString('o')}})
 [DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000)){throw 'Owned host did not close normally'}
 $owned['exit_code']=$app.ExitCode
 foreach($c in $children){$live=Get-CimInstance Win32_Process -Filter ('ProcessId='+$c.pid);if($live -and $live.ExecutablePath -eq $c.path -and $live.CreationDate.ToUniversalTime().ToString('o') -eq $c.created){throw 'Owned direct child survived close'}}
 $late=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$app.Id)|Where-Object {$_.CreationDate.ToUniversalTime() -ge [DateTime]::Parse($owned.start).ToUniversalTime() -and $_.CreationDate.ToUniversalTime() -le $app.ExitTime.ToUniversalTime()})
 if($late.Count){throw 'Owned renderer remains after close'};$code=$app.ExitCode;$app.Dispose();$script:app=$null;if($code -ne 0){throw 'Host nonzero exit'}
}
Write-Output ('REPORT_ROOT='+$lab)
try {
 $build=Get-Content $BuildReceipt -Raw -Encoding UTF8|ConvertFrom-Json;$head=Git @('rev-parse','HEAD')
 if(-not $build.passed -or $build.mode -ne 'build' -or -not $build.source_unchanged -or $build.commit -ne $head){throw 'Exact clean build required'}
 if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256 -or (Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'Build identity changed'}
 $manifest=Get-Content $build.source_hash_manifest.path -Raw -Encoding UTF8|ConvertFrom-Json;Source
 $receipt['commit']=$head;$receipt['build_receipt']=$BuildReceipt;$receipt['app_sha256']=$build.app_artifact.sha256
 if((Hash $Letter) -ne '11ca20995a5da66033c8dd75cc55a145bc6179bf189ac7d8bfaf9d90562a72a7'){throw 'Historical letter fixture identity changed'}
 Add-Type -AssemblyName System.Drawing;Add-Type -AssemblyName System.Windows.Forms;Add-Type -AssemblyName System.IO.Compression;Add-Type -AssemblyName System.IO.Compression.FileSystem
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-text-probe.cs')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs')
 . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
 Copy-Item -LiteralPath $Letter -Destination (Join-Path $fixture.inbox '05-letter.docx')
 $definition=Get-Content (Join-Path $repo 'tests/native/docx-content-fixture.json') -Raw -Encoding UTF8|ConvertFrom-Json
 $path=Join-Path $fixture.inbox '06-structured.DOCX';$zip=[IO.Compression.ZipFile]::Open($path,[IO.Compression.ZipArchiveMode]::Create)
 try{foreach($part in $definition.entries){if($part.name.StartsWith('/') -or $part.name.Contains('..')){throw 'Unsafe test ZIP part'};$entry=$zip.CreateEntry($part.name);$stream=$entry.Open();try{$bytes=if($part.PSObject.Properties.Name -contains 'text'){$utf8.GetBytes($part.text)}else{[Convert]::FromBase64String($part.base64)};$stream.Write($bytes,0,$bytes.Length)}finally{$stream.Dispose()}}}finally{$zip.Dispose()}
 Get-ChildItem $fixture.inbox -File|ForEach-Object {$expected[$_.Name]=Hash $_.FullName};$receipt['source_fixture_hashes']=$expected
 $receipt['historical_letter_source']=$Letter;$receipt['historical_currency_note']='The old fixture stores a literal question mark before 25,000; this is not an inferred rendering defect.'
 $key='{8895b1c6-b41f-4c1c-a562-0d564250836f}';$reg=[Microsoft.Win32.Registry]::ClassesRoot;$r=$reg.OpenSubKey('.docx');$prog='';if($r){try{$prog=[string]$r.GetValue('')}finally{$r.Dispose()}}
 $registrations=@();foreach($k in @(($prog+'\shellex\'+$key),('.docx\shellex\'+$key),('SystemFileAssociations\.docx\shellex\'+$key))){$r=$reg.OpenSubKey($k);$value=$null;if($r){try{$value=$r.GetValue('')}finally{$r.Dispose()}};$registrations+=[ordered]@{key=$k;value=$value}};$receipt['handler_registrations_read_only']=$registrations
 $exe=Join-Path $lab ('DCM-DOCX-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item $build.app_artifact.path $exe
 $temp=Join-Path $lab 'temp';$null=New-Item -ItemType Directory $temp
 $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
 $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming;$psi.EnvironmentVariables['TEMP']=$temp;$psi.EnvironmentVariables['TMP']=$temp;$psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
 foreach($var in @('DCM_PDF_PROVIDER','DCM_SUMATRA_EXE','DCM_STARTUP_TRACE','DCM_LAYOUT_TRACE')){$psi.EnvironmentVariables.Remove($var)}
 $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'Host launch failed'};$owned=[ordered]@{pid=$app.Id;path=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')};$receipt['owned_process']=$owned
 $until=[DateTime]::UtcNow.AddSeconds(20);do{$app.Refresh();if($app.HasExited){throw 'Host exited before window'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
 Wait 'fixture list' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $expected.Count}
 SelectFile '01-alpha.txt';$plain=[IO.File]::ReadAllText((Join-Path $fixture.inbox '01-alpha.txt'));$p=ExpectBody '01-initial-text' @($plain) @('DCM DOCX Preview');Shot '01-initial-text'
 SelectFile '05-letter.docx'
 $letterTokens=@('DCM DOCX Preview','Title: FlowDoc P3 Acceptance','Government of Odisha - Gurundia Block','Subject: Road Repair','Sanctioned amount: ?25,000','Generated by FlowDoc.','Block Development Officer','Rendered locally from DOCX Open XML')
 $p=ExpectBody '02-letter-content' $letterTokens @('DCM disposable native qualification fixture','DCM-30 FIELD REPORT');$receipt.provider='Built-in DOCX Open XML structured-text fallback';$receipt['letter_status']=CheckStatus 'DCM DOCX Preview:';Shot '02-letter-content';Exclusive '05-letter.docx';Record '03-letter-exclusive-access' $null
 SelectFile '06-structured.DOCX'
 $tokens=@('DCM DOCX Preview','Title: DCM-30 structured preview','DCM-30 FIELD REPORT','Alpha parcel: 31 units.','Formatting is intentionally mixed.',('Unicode: Caf'+[char]233),'Measurements','TABLE 1','| Item','Quantity','| Bolts','12','| Panels','3','[Image: Green sample square','green-square.png]','END DCM-30 ALPHA','Rendered locally from DOCX Open XML')
 $p=ExpectBody '04-structured-content' $tokens @('Subject: Road Repair','Block Development Officer');$structured=$p.Text;$receipt['structured_status']=CheckStatus 'DCM DOCX Preview:';Shot '04-structured-top'
 $line=[DcmTextProbe]::ScrollEnd($hwnd,$app.Id,[IntPtr]$p.Handle);Shot '05-structured-tail';Record '05-scroll-to-tail' ([ordered]@{first_visible_line=$line})
 [DcmNative]::ClickAsync($hwnd,$app.Id,1022);Wait 'DOCX zoom 110' {(@([DcmTextProbe]::Controls($hwnd,$app.Id)|Where-Object {$_.Visible -and $_.Text -like 'DOCX preview zoom: 110%*'}).Count -eq 1)}
 $p=Preview;if($p.Text -cne $structured){throw 'Zoom changed DOCX content'};Shot '06-docx-zoom-110';Record '06-docx-zoom-content-stable' $null
 [DcmNative]::ClickAsync($hwnd,$app.Id,1021);Wait 'DOCX zoom reset' {(@([DcmTextProbe]::Controls($hwnd,$app.Id)|Where-Object {$_.Visible -and $_.Text -like 'DOCX preview zoom: 100%*'}).Count -eq 1)}
 if((Preview).Text -cne $structured){throw 'Zoom reset changed content'};Record '07-docx-zoom-reset' $null
 SelectFile '01-alpha.txt';$p=ExpectBody '08-text-recovery' @($plain) @('DCM DOCX Preview','DCM-30 FIELD REPORT');Shot '08-text-recovery';Exclusive '05-letter.docx';Exclusive '06-structured.DOCX';Record '09-both-docx-files-released' $null
 SelectFile '05-letter.docx';$p=ExpectBody '10-letter-reopened' $letterTokens @('DCM-30 FIELD REPORT','TABLE 1');Shot '10-letter-reopened'
 Files;StopApp;Source;Exclusive '05-letter.docx';Exclusive '06-structured.DOCX'
 if((Hash $Letter) -ne $expected['05-letter.docx'] -or (Hash $exe) -ne $build.app_artifact.sha256){throw 'Original fixture/app changed'}
 Record '11-normal-close-and-original-bytes' $null;$receipt['source_unchanged']=$true;$receipt['fixtures_preserved']=$true;$receipt['journal_empty']=$true;$receipt.passed=$true
}catch{$receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message);if($app){try{Shot 'failure'}catch{$receipt['capture_error']=$_.Exception.Message}}}
finally{try{StopApp}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message};$receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt;Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))}
if(-not $receipt.passed){exit 1}
