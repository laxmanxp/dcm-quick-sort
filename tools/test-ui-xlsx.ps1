[CmdletBinding()]
param(
 [string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,
 [string]$Workbook='D:\FlowverseReportsRepo\dcm-flowuivision-20261103\inbox\06-orders.xlsx',
 [switch]$ObserveNativeLayout,
 [switch]$RequireNativeBounds
)
# DCM-31: natural installed-handler/fallback selection; no registration changes.
# Native cells require separate image review. Fallback sheet/cell readback is asserted.
$ErrorActionPreference='Stop';Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required'}
if(-not $BuildReceipt -or -not $env:FLOWDEV_WORKSPACE_ROOT){throw 'Exact build and workspace required'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-xlsx-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Test lab must be outside source'}
$null=New-Item -ItemType Directory $lab;$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-31';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;feature_accepted=$false;cases=$cases;captures=$shots;visual_review_required=$true;user_test='not_recorded';provider='not_observed'}
$receipt['native_layout_diagnostic']=[bool]$ObserveNativeLayout
$receipt['native_bounds_required']=[bool]$RequireNativeBounds
if($RequireNativeBounds){$receipt['native_bounds_checks']=New-Object 'Collections.Generic.List[object]'}
if($ObserveNativeLayout){$receipt['layout_observations']=New-Object 'Collections.Generic.List[object]'}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$head='';$manifest=@();$fixture=$null;$expected=@{};$exe=''
$workbookHash='5059729f4390ecd2e169bb2d1eb850c768532ebbb24b9733e87b82b012d6d24f'
$sheetNames=@('Orders','Read me')
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
function ObserveLayout([string]$name){
 if(-not $ObserveNativeLayout){return};Own
 $path=Join-Path $lab ($name+'.geometry.json')
 Json $path ([ordered]@{utc=[DateTime]::UtcNow.ToString('o');geometry=[DcmPreviewGeometry]::Observe($hwnd,$app.Id);host_layout=[DcmLayoutProbe]::Observe($hwnd,$app.Id)})
 $receipt.layout_observations.Add([ordered]@{name=$name;path=$path;sha256=(Hash $path)})
}
function CaptureFrame([string]$name){
 Own;ObserveLayout ($name+'-before-capture')
 $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false)
 ObserveLayout ($name+'-after-capture')
 $shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false})
 if($RequireNativeBounds -and ((HostCaptions) -ccontains 'Windows preview: XLSX')){CheckNativeBounds $name}
}
function Shot([string]$name){Own;$screen=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea;[DcmNative]::Restore($hwnd,$app.Id,$screen.Left+8,$screen.Top+8);Start-Sleep -Milliseconds 300;CaptureFrame $name}
function CheckNativeBounds([string]$name){
 Own;$geometry=[DcmPreviewGeometry]::Observe($hwnd,$app.Id)
 $path=Join-Path $lab ($name+'.bounds.json');Json $path $geometry
 $contexts=@($geometry.Contexts|Where-Object {$_.CallerContextLabel -eq 'per_monitor_aware_v2_requested_minus4'})
 if($contexts.Count -ne 1){throw 'Native bounds observer context missing'}
 $windows=@($contexts[0].Windows|Where-Object {$_.Available -and $_.Visible})
 $panes=@($windows|Where-Object {$_.Role -eq 'owned_preview_pane'})
 if($panes.Count -ne 1 -or -not $panes[0].ClientRectAvailable -or -not $panes[0].WindowRectAvailable){throw 'Native bounds pane unavailable'}
 $pane=$panes[0];$width=$pane.ClientRect.Right-$pane.ClientRect.Left;$height=$pane.ClientRect.Bottom-$pane.ClientRect.Top
 $outer=@($windows|Where-Object {$_.Class -eq 'EXCEL9' -and $_.Parent -eq $geometry.PreviewPane})
 $inner=@($windows|Where-Object {$_.Class -eq 'EXCEL7'})
 $ok=$width -gt 0 -and $height -gt 0 -and $outer.Count -eq 1 -and $inner.Count -eq 1
 if($ok){$ok=$inner[0].Parent -eq $outer[0].Handle}
 $observed=@()
 foreach($window in @($outer)+@($inner)){
  $rect=$window.WindowRect;$w=$rect.Right-$rect.Left;$h=$rect.Bottom-$rect.Top
  $fits=$window.WindowRectAvailable -and [Math]::Abs($w-$width) -le 4 -and [Math]::Abs($h-$height) -le 4 -and [Math]::Abs($rect.Left-$pane.WindowRect.Left) -le 4 -and [Math]::Abs($rect.Top-$pane.WindowRect.Top) -le 4
  $ok=$ok -and $fits
  $observed+=[ordered]@{class=$window.Class;pid=$window.PID;window=$window.Handle;width=$w;height=$h;left=$rect.Left;top=$rect.Top;fits=$fits}
 }
 # Keep every frame and the recovery/close flow before failing the gate. This
 # asserts viewport geometry, not cells, Excel fidelity or screenshot readability.
 $receipt.native_bounds_checks.Add([ordered]@{capture=$name;passed=[bool]$ok;scope='Installed Excel EXCEL9/EXCEL7 viewport in PMv2 observer coordinates';expected_client_width=$width;expected_client_height=$height;tolerance_pixels=4;windows=$observed;observation_path=$path;sha256=(Hash $path);observed_utc=$geometry.StartedUtc})
}
function Record([string]$name,$detail){$cases.Add([ordered]@{name=$name;passed=$true;detail=$detail});Files}
function SelectFile([string]$name){$items=@([DcmNative]::ListItems($hwnd,$app.Id,1016));$index=[Array]::IndexOf($items,$name);if($index -lt 0){throw 'Missing fixture in list'};[DcmUndoInput]::Select($hwnd,$app.Id,$index);$stem=[IO.Path]::GetFileNameWithoutExtension($name);Wait ('selection '+$name) {([DcmUndoInput]::TryText($hwnd,$app.Id,1005)) -ceq $stem};$selected=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016));if($selected.Count -ne 1 -or $selected[0] -ne $index){throw 'Selected fixture index mismatch'}}
function HostCaptions {return @([DcmDocxProbe]::HostStatics($hwnd,$app.Id)|ForEach-Object {[DcmNative]::Text([IntPtr]$_,$app.Id)})}
function RequireStatus([string]$text){if(-not ((HostCaptions) -ccontains $text)){throw ('XLSX provider/status mismatch: '+$text)}}
function NativeNodes {
 # A native handler may be in the host process. Require its nonempty visible
 # subtree as well as the exact native-provider footer, not a foreign PID alone.
 return @([DcmDocxProbe]::Snapshot($hwnd,$app.Id)|Where-Object {$_.Visible -and $_.Right -gt $_.Left -and $_.Bottom -gt $_.Top -and ($_.PID -ne $app.Id -or $_.Class -notin @('Edit','ComboBox'))})
}
function Preview {
 if((HostCaptions) -ccontains 'Windows preview: XLSX'){return $null}
 $nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id))
 if(@($nodes|Where-Object {$_.Visible -and $_.PID -ne $app.Id}).Count){return $null}
 $edits=@($nodes|Where-Object {$_.Visible -and $_.PID -eq $app.Id -and $_.Class -eq 'Edit'})
 if($edits.Count -gt 1){throw 'Multiple owned preview edits'}
 if($edits.Count -eq 1){return [pscustomobject]@{Handle=$edits[0].Handle;Text=[DcmNative]::Text([IntPtr]$edits[0].Handle,$app.Id)}}
 return $null
}
function ExpectBody([string]$name,[string[]]$tokens,[string[]]$absent){
 if(-not $tokens.Count -or -not $tokens[0]){throw 'Expected workbook content must not be empty'}
 Wait ('body '+$name) {$p=Preview;($null -ne $p) -and $p.Text.Contains($tokens[0])}
 $p=Preview;$last=-1;foreach($token in $tokens){$at=$p.Text.IndexOf($token,$last+1,[StringComparison]::Ordinal);if($at -lt 0 -or $at -lt $last){throw ('Missing/out-of-order XLSX content: '+$token)};$last=$at}
 foreach($token in $absent){if($p.Text.Contains($token)){throw ('Stale or wrong worksheet content: '+$token)}}
 [IO.File]::WriteAllText((Join-Path $lab ($name+'.displayed.txt')),$p.Text,$utf8)
 return $p
}
function RowCells([string]$text,[int]$row){$lines=@(($text -split '\r?\n')|Where-Object {$_ -match ('^\s*'+$row+'\s*\|')});if($lines.Count -ne 1){throw ('Expected one displayed grid row '+$row)};return @($lines[0].Split('|')|ForEach-Object {$_.Trim()})}
function Orders([string]$name){
 $p=ExpectBody $name @('Sheet: Orders','office','order_number','date','recipient_name','subject','amount','Sample Office','2026-09-26','Sample Recipient 01','10500','Sample Recipient 20','20000') @('FLOWDOC 0.10.0','Sheet: Read me')
 $first=RowCells $p.Text 2;$last=RowCells $p.Text 21
 if($first.Count -ne 7 -or $first[1] -cne 'Sample Office' -or $first[2] -cne '1' -or $first[3] -cne '2026-09-26' -or $first[4] -cne 'Sample Recipient 01' -or -not $first[5].StartsWith('Demonstration document',[StringComparison]::Ordinal) -or $first[6] -cne '10500'){throw 'First displayed Orders row/cell coordinates differ'}
 if($last.Count -ne 7 -or $last[2] -cne '20' -or $last[4] -cne 'Sample Recipient 20' -or $last[6] -cne '20000'){throw 'Last displayed Orders row/cell coordinates differ'}
 RequireStatus ('DCM XLSX Preview: Orders '+[char]0x00B7+' 21 rows '+[char]0x00D7+' 6 columns')
 if([DcmXlsxInput]::SelectedSheet($hwnd,$app.Id) -cne 'Orders'){throw 'Orders grid and selected worksheet disagree'}
 Record $name ([ordered]@{displayed_sha256=(Hash (Join-Path $lab ($name+'.displayed.txt')));first_data_row=$first;last_data_row=$last;layout_fidelity='bounded monospaced grid; long cells truncated'})
}
function NativeShot([string]$name){
 Wait 'native XLSX provider and visible subtree' {((HostCaptions) -ccontains 'Windows preview: XLSX') -and @(NativeNodes).Count -gt 0}
 Own;$nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$native=@(NativeNodes)
 if(-not $native.Count){throw 'Native XLSX subtree absent'}
 Json (Join-Path $lab ($name+'.native-windows.json')) $nodes
 $identities=@($native|Select-Object -ExpandProperty PID -Unique|ForEach-Object {$p=Get-Process -Id $_;[ordered]@{pid=$p.Id;path=$p.Path;start=$p.StartTime.ToUniversalTime().ToString('o');is_host=($p.Id -eq $app.Id)}})
 $text=[DcmDocxProbe]::ReadAccessibleText($hwnd,$app.Id);[IO.File]::WriteAllText((Join-Path $lab ($name+'.accessible.txt')),$text,$utf8)
 Start-Sleep -Milliseconds 500
 if($ObserveNativeLayout){CaptureFrame ($name+'-before-restore')}
 Shot $name
 if($ObserveNativeLayout){
  # Retain raw output, then observe natural settling. These captures issue no
  # Restore, resize, SetRect, repaint request, fence, or native-provider input.
  Start-Sleep -Milliseconds 250;CaptureFrame ($name+'-natural-250')
  Start-Sleep -Milliseconds 750;CaptureFrame ($name+'-natural-1000')
  Start-Sleep -Milliseconds 3000;CaptureFrame ($name+'-natural-4000')
 }
 RequireStatus 'Windows preview: XLSX'
 Record $name ([ordered]@{native_identities=$identities;accessible_text_sha256=(Hash (Join-Path $lab ($name+'.accessible.txt')));rendered_cells_machine_asserted=$false;rendered_content_review='required_separately';native_sheet_navigation='not_exercised';status='Windows preview: XLSX'})
}
function Exclusive {$h=[IO.File]::Open((Join-Path $fixture.inbox '06-orders.xlsx'),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()}
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
 if((Hash $Workbook) -ne $workbookHash){throw 'Historical XLSX fixture identity changed'}
 Add-Type -AssemblyName System.Drawing;Add-Type -AssemblyName System.Windows.Forms
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-xlsx-input.cs')
 if($ObserveNativeLayout -or $RequireNativeBounds){
  Add-Type -Path (Join-Path $repo 'tests/native/dcm-preview-geometry.cs')
 }
 if($ObserveNativeLayout){
  Add-Type -Path (Join-Path $repo 'tests/native/dcm-layout-probe.cs')
 }
 Add-Type -AssemblyName UIAutomationClient;Add-Type -AssemblyName UIAutomationTypes;Add-Type -AssemblyName WindowsBase
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-docx-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll',([Windows.Automation.AutomationElement]).Assembly.Location,([Windows.Automation.AutomationPattern]).Assembly.Location,([Windows.Point]).Assembly.Location)
 . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
 Copy-Item -LiteralPath $Workbook -Destination (Join-Path $fixture.inbox '06-orders.xlsx')
 Get-ChildItem $fixture.inbox -File|ForEach-Object {$expected[$_.Name]=Hash $_.FullName};$receipt['source_fixture_hashes']=$expected
 $receipt['historical_workbook_source']=$Workbook;$receipt['historical_workbook_sha256']=$workbookHash;$receipt['expected_sheet_names']=$sheetNames
 $receipt['fixture_scope']='Synthetic Orders workbook; expected first data row Sample Office / 1 / 2026-09-26 / Sample Recipient 01 / Demonstration document generation / 10500. Read me is a distinct second worksheet. Native provider formatting may differ from the bounded built-in grid.'
 $key='{8895b1c6-b41f-4c1c-a562-0d564250836f}';$reg=[Microsoft.Win32.Registry]::ClassesRoot;$r=$reg.OpenSubKey('.xlsx');$prog='';if($r){try{$prog=[string]$r.GetValue('')}finally{$r.Dispose()}}
 $registrations=@();foreach($k in @(($prog+'\shellex\'+$key),('.xlsx\shellex\'+$key),('SystemFileAssociations\.xlsx\shellex\'+$key))){$r=$reg.OpenSubKey($k);$value=$null;if($r){try{$value=$r.GetValue('')}finally{$r.Dispose()}};$registrations+=[ordered]@{key=$k;value=$value}};$receipt['handler_registrations_read_only']=$registrations
 $exe=Join-Path $lab ('DCM-XLSX-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item $build.app_artifact.path $exe
 $temp=Join-Path $lab 'temp';$null=New-Item -ItemType Directory $temp
 $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
 $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming;$psi.EnvironmentVariables['TEMP']=$temp;$psi.EnvironmentVariables['TMP']=$temp;$psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
 foreach($var in @('DCM_PDF_PROVIDER','DCM_SUMATRA_EXE','DCM_STARTUP_TRACE','DCM_LAYOUT_TRACE')){$psi.EnvironmentVariables.Remove($var)}
 if($ObserveNativeLayout){$psi.EnvironmentVariables['DCM_LAYOUT_TRACE']='1';$receipt['host_layout_trace']=Join-Path $temp 'dcm-layout-trace.json'}
 $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'Host launch failed'};$owned=[ordered]@{pid=$app.Id;path=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')};$receipt['owned_process']=$owned
 $until=[DateTime]::UtcNow.AddSeconds(20);do{$app.Refresh();if($app.HasExited){throw 'Host exited before window'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
 Wait 'fixture list' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $expected.Count}
 SelectFile '01-alpha.txt';$plain=[IO.File]::ReadAllText((Join-Path $fixture.inbox '01-alpha.txt'));$p=ExpectBody '01-initial-text' @($plain) @('Sheet: Orders','Sample Recipient');RequireStatus 'Plain-text preview: TXT';Shot '01-initial-text';Record '01-initial-text' ([ordered]@{displayed_sha256=(Hash (Join-Path $lab '01-initial-text.displayed.txt'))})
 SelectFile '06-orders.xlsx'
 Wait 'native XLSX or completed fallback grid' {
  if((HostCaptions) -ccontains 'Windows preview: XLSX'){return (@(NativeNodes).Count -gt 0)}
  $p=Preview;return ($null -ne $p -and $p.Text.StartsWith('Sheet: Orders',[StringComparison]::Ordinal))
 }
 $native=(HostCaptions) -ccontains 'Windows preview: XLSX'
 if($native){
  $receipt.provider='Installed Windows XLSX preview handler';$receipt['native_sheet_navigation']='not_exercised; no foreign input adapter';NativeShot '02-native-workbook'
 }else{
  $receipt.provider='Built-in XLSX Open XML grid fallback'
  $actual=@([DcmXlsxInput]::SheetNames($hwnd,$app.Id));if(($actual -join '|') -cne ($sheetNames -join '|')){throw 'Fallback worksheet names/order differ'}
  $receipt['observed_fallback_sheet_names']=$actual
  Orders '02-orders-content';Shot '02-orders-content'
  [DcmXlsxInput]::SelectSheet($hwnd,$app.Id,'Read me')
  Wait 'Read me selection and body' {([DcmXlsxInput]::SelectedSheet($hwnd,$app.Id) -ceq 'Read me') -and $null -ne (Preview) -and (Preview).Text.StartsWith('Sheet: Read me',[StringComparison]::Ordinal)}
  $p=ExpectBody '03-readme-content' @('Sheet: Read me','FLOWDOC 0.10.0','Purpose','Demonstration only.','Data worksheet','Choose Orders','Template','Open the included') @('Sample Recipient 01','Sheet: Orders')
  $purpose=RowCells $p.Text 3;if($purpose.Count -ne 5 -or $purpose[1] -cne 'Purpose' -or -not $purpose[2].StartsWith('Demonstration only.',[StringComparison]::Ordinal)){throw 'Read me cell coordinates differ'}
  RequireStatus ('DCM XLSX Preview: Read me '+[char]0x00B7+' 14 rows '+[char]0x00D7+' 4 columns');Shot '03-readme-content';Record '03-readme-content' ([ordered]@{displayed_sha256=(Hash (Join-Path $lab '03-readme-content.displayed.txt'));purpose_row=$purpose;selected_sheet='Read me'})
 }
 SelectFile '01-alpha.txt';$p=ExpectBody '04-text-recovery' @($plain) @('Sheet: Orders','Sheet: Read me','Sample Recipient','FLOWDOC 0.10.0');RequireStatus 'Plain-text preview: TXT';Shot '04-text-recovery'
 if(@(NativeNodes).Count){throw 'Native XLSX subtree retained after text selection'}
 Record '04-text-recovery' ([ordered]@{displayed_sha256=(Hash (Join-Path $lab '04-text-recovery.displayed.txt'))})
 Exclusive;Record '05-workbook-released' $null
 SelectFile '06-orders.xlsx'
 if($native){NativeShot '06-native-workbook-reopened'}else{Orders '06-orders-reopened';Shot '06-orders-reopened'}
 Files;StopApp;Source;Exclusive
 if($ObserveNativeLayout){$receipt['host_layout_trace_sha256']=Hash $receipt.host_layout_trace}
 if((Hash $Workbook) -ne $workbookHash -or (Hash $exe) -ne $build.app_artifact.sha256){throw 'Original workbook/app changed'}
 if($cases.Count -lt 5 -or $shots.Count -lt 4){throw 'Incomplete XLSX qualification records/captures'}
 Record '07-normal-close-and-original-bytes' $null;$receipt['source_unchanged']=$true;$receipt['fixtures_preserved']=$true;$receipt['journal_empty']=$true
 if($RequireNativeBounds -and $native){
  if($receipt.native_bounds_checks.Count -lt 2){throw 'Native bounds checks incomplete'}
  $bad=@($receipt.native_bounds_checks|Where-Object {-not $_.passed})
  if($bad.Count){throw ('DCM31_XLSX_NATIVE_BOUNDS: Excel viewport does not fill the preview pane in '+(($bad|ForEach-Object {$_.capture}) -join ', '))}
 }
 $receipt.passed=$true
}catch{
 $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)
 if($app){try{Own;Json (Join-Path $lab 'failure-preview-windows.json') @([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$receipt['failure_host_captions']=HostCaptions}catch{$receipt['diagnostic_error']=$_.Exception.Message};try{Shot 'failure'}catch{$receipt['capture_error']=$_.Exception.Message}}
}finally{try{StopApp}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message};$receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt;Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))}
if(-not $receipt.passed){exit 1}
