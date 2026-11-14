[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,[string]$Runtime=$env:DCM_SUMATRA_SOURCE_EXE)
# Source-bound active-PDF Undo and controlled owned-viewer failure acceptance.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt -or -not $Runtime){throw 'Workspace, exact build receipt and explicit runtime required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-sumatra-undo-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]';$viewers=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-SUMATRA-UNDO-01';passed=$false;report=$lab;cases=$cases;captures=$shots;viewers=$viewers;visual_reviewed=$false}
$selectionTimes=New-Object 'Collections.Generic.List[object]';$receipt['selection_readiness']=$selectionTimes
$app=$null;$hwnd=[IntPtr]::Zero;$head='';$manifest=@();$fixture=$null
Write-Output ('REPORT_ROOT='+$lab)
function Hash([string]$p){return (Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Json([string]$p,$v){[IO.File]::WriteAllText($p,($v|ConvertTo-Json -Depth 25),$utf8)}
function Git([string[]]$a){$v=(& git.exe -C $repo @a|Out-String).Trim();if($LASTEXITCODE -ne 0){throw 'Git failed'};return $v}
function CheckSource {if((Git @('rev-parse','HEAD')) -ne $head -or (Git @('status','--porcelain=v1','--untracked-files=all'))){throw 'Source identity changed'};foreach($f in $manifest){if((Hash (Join-Path $repo $f.path)) -ne $f.sha256){throw 'Source manifest changed'}}}
function Own {$app.Refresh();if($app.HasExited -or $app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $receipt.host_start){throw 'Owned app identity/liveness changed'};[DcmNative]::RequireOwned($hwnd,$app.Id)}
function Wait([string]$Label,[scriptblock]$Check){$until=[DateTime]::UtcNow.AddSeconds(20);do{Own;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw ('Unexpected owned dialog: '+$Label)};if(& $Check){return};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until);throw ('Deadline: '+$Label)}
function Text([int]$id){return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,$id),$app.Id)}
function SelectFile([string]$file){
    $items=[DcmNative]::ListItems($hwnd,$app.Id,1016);$index=[Array]::IndexOf($items,$file)
    if($index -lt 0){throw 'Selected fixture not listed'}
    $clock=[Diagnostics.Stopwatch]::StartNew();[DcmUndoInput]::Select($hwnd,$app.Id,$index);$stem=[IO.Path]::GetFileNameWithoutExtension($file)
    try{Wait ('selection '+$file) {([DcmUndoInput]::TryText($hwnd,$app.Id,1005)) -ceq $stem -and (@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016)) -join ',') -eq [string]$index};$clock.Stop();$selectionTimes.Add([ordered]@{file=$file;milliseconds=$clock.ElapsedMilliseconds})}catch{
        Json (Join-Path $lab 'selection-failure.json') ([ordered]@{expected=$stem;actual=(Text 1005);items=@([DcmNative]::ListItems($hwnd,$app.Id,1016));selected=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016));embedded=[DcmSumatraProbe]::Find($hwnd,$app.Id)})
        throw
    }
}
function Status {return (@([DcmLifecycleProbe]::Status($hwnd,$app.Id)) -join "`n")}
function Shot([string]$name,[string]$color=''){
    Own;$area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
    [DcmNative]::Restore($hwnd,$app.Id,$area.Left+8,$area.Top+8);Start-Sleep -Milliseconds 400
    $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false)
    $shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false})
    if($color){
        $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'-content.png')),$true)
        $shots.Add([ordered]@{name=$name+'-content';path=$s.Path;sha256=(Hash $s.Path);teal=$s.TealSamples;orange=$s.OrangeSamples;visual_reviewed=$false})
        if($s.TealSamples -lt 1000 -or $s.OrangeSamples -gt 100){throw 'Expected original teal PDF page, not stale or blank content'}
    }
}
function Viewer([string]$document) {
    Wait 'embedded Sumatra child' { $null -ne [DcmSumatraProbe]::Find($hwnd,$app.Id) }
    Wait 'host ready' {(Status) -like 'Sumatra PDF preview*'};Start-Sleep -Milliseconds 200
    $v=[DcmSumatraProbe]::Find($hwnd,$app.Id);$p=Get-Process -Id $v.PID
    $w=Get-CimInstance Win32_Process -Filter ('ProcessId='+$v.PID)
    if($p.Path -ne $runtimeExe -or $w.ParentProcessId -ne $app.Id -or -not $w.CommandLine.Contains($document)){throw 'Viewer process/document identity mismatch'}
    if($w.CommandLine -notmatch '\-restrict' -or $w.CommandLine -notmatch '\-plugin'){throw 'Viewer not restricted/embedded'}
    $entry=[ordered]@{pid=$v.PID;path=$p.Path;start=$p.StartTime.ToUniversalTime().ToString('o');hwnd=$v.Window;command_line=$w.CommandLine};$viewers.Add($entry)
    [DcmSumatraProbe]::CheckGeometry($hwnd,$app.Id,$v.PID);return $entry
}
function Gone($v){Wait 'owned viewer released' {$p=Get-Process -Id $v.pid -ErrorAction SilentlyContinue; -not $p -or $p.StartTime.ToUniversalTime().ToString('o') -ne $v.start}}
function AddPass([string]$n){$cases.Add([ordered]@{name=$n;passed=$true})}

function Journal([string]$from,[string]$to,[string]$state,[int]$count){
    $latest=@{}
    if(Test-Path -LiteralPath $journal){foreach($line in [IO.File]::ReadAllLines($journal)){if($line.Trim()){$j=$line|ConvertFrom-Json;$latest[$j.id]=$j}}}
    if($latest.Count -ne $count){throw ('Unexpected journal transaction count '+$latest.Count)}
    $match=@($latest.Values|Where-Object {$_.record.from -eq $from -and $_.record.to -eq $to})
    if($match.Count -ne 1 -or $match[0].state -ne $state){throw ('Journal state mismatch: '+$state)}
    return $match[0].id
}
function SelectUndoRecord([string]$name){
    Wait 'Undo ready' {[DcmNative]::IsWindowEnabled([DcmNative]::Control($hwnd,$app.Id,1008))}
    $items=@([DcmNative]::ListItems($hwnd,$app.Id,1017));$hits=@()
    for($i=0;$i -lt $items.Count;$i++){if($items[$i].Contains($name)){$hits+=,$i}}
    if($hits.Count -ne 1){throw 'Expected exactly one matching Processed transaction'}
    $null=[DcmNative]::SelectSingleItem($hwnd,$app.Id,1017,$hits[0])
}
function SwitchSource([string]$path,[int]$count){
    [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1001,$path);[DcmNative]::ClickAsync($hwnd,$app.Id,1011)
    Wait 'source switch' {(Text 1001) -eq $path -and @([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $count}
}
function Exclusive([string]$path){$h=[IO.File]::Open($path,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()}

try{
    $build=Get-Content -LiteralPath $BuildReceipt -Raw|ConvertFrom-Json
    $head=Git @('rev-parse','HEAD')
    if(-not $build.passed -or $build.mode -ne 'build' -or -not $build.source_unchanged -or $head -ne $build.commit){throw 'Exact successful build required'}
    if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256 -or (Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'Build changed'}
    $manifest=Get-Content -LiteralPath $build.source_hash_manifest.path -Raw|ConvertFrom-Json;CheckSource
    $receipt['commit']=$head;$receipt['build_receipt']=$BuildReceipt;$receipt['app_sha256']=$build.app_artifact.sha256
    if((Hash $Runtime) -ne '719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0'){throw 'Not the pinned unmodified Sumatra 3.6.1 runtime'}
    $runtimeDir=Join-Path $lab 'runtime';$null=New-Item -ItemType Directory -Path $runtimeDir
    $runtimeExe=Join-Path $runtimeDir 'SumatraPDF.exe';Copy-Item -LiteralPath $Runtime -Destination $runtimeExe
    $policy="[Policies]`nInternetAccess = 0`nDiskAccess = 0`nSavePreferences = 0`nRegistryAccess = 0`nPrinterAccess = 0`nCopySelection = 0`nFullscreenAccess = 0`nLinkProtocols =`nSafeFileTypes =`n"
    [IO.File]::WriteAllText((Join-Path $runtimeDir 'sumatrapdfrestrict.ini'),$policy,$utf8)
    $receipt['runtime_sha256']=Hash $runtimeExe;$receipt['policy_sha256']=Hash (Join-Path $runtimeDir 'sumatrapdfrestrict.ini')
    Add-Type -AssemblyName System.Drawing;Add-Type -AssemblyName System.Windows.Forms
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-sumatra-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','Accessibility.dll','System.Drawing.dll')
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-lifecycle-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
    . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
    $original=Join-Path $fixture.inbox '08-preview.pdf';$pdfHash=Hash $original
    $journal=Join-Path $fixture.config 'recovery\move-journal.jsonl'
    $otherHashes=@{};Get-ChildItem -LiteralPath $fixture.inbox -Filter '*.txt' | ForEach-Object {$otherHashes[$_.Name]=Hash $_.FullName}
    $exe=Join-Path $lab ('DCM-Sumatra-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
    $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming
    $psi.EnvironmentVariables['TEMP']=$lab;$psi.EnvironmentVariables['TMP']=$lab
    $psi.EnvironmentVariables['DCM_PDF_PROVIDER']='sumatra';$psi.EnvironmentVariables['DCM_SUMATRA_EXE']=$runtimeExe
    $psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'App start failed'}
    $receipt['host_pid']=$app.Id;$receipt['host_start']=$app.StartTime.ToUniversalTime().ToString('o')
    $until=[DateTime]::UtcNow.AddSeconds(15);do{$app.Refresh();$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    Wait 'fixture list' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 4}
    [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1003,'Accepted')

    SelectFile '08-preview.pdf';$v=Viewer $original;Shot '01-original-active' 'teal';AddPass 'original-PDF-active'
    [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1005,'undo-preview');[DcmNative]::ClickAsync($hwnd,$app.Id,1006)
    $renamed=Join-Path $fixture.inbox 'undo-preview.pdf'
    Wait 'rename completed' {(Test-Path -LiteralPath $renamed) -and -not(Test-Path -LiteralPath $original)};Gone $v
    if((Hash $renamed) -ne $pdfHash){throw 'Rename changed bytes'}
    $renameID=Journal $original $renamed 'completed' 1
    Wait 'renamed selection' {(Text 1005) -ceq 'undo-preview'};$v=Viewer $renamed
    Shot '02-renamed-active-before-undo' 'teal';AddPass 'rename-reopens-PDF-with-completed-journal'
    SelectUndoRecord 'undo-preview.pdf';[DcmNative]::ClickAsync($hwnd,$app.Id,1008)
    Wait 'Undo Rename completed' {(Test-Path -LiteralPath $original) -and -not(Test-Path -LiteralPath $renamed)};Gone $v
    if((Hash $original) -ne $pdfHash -or (Journal $original $renamed 'undone' 1) -ne $renameID){throw 'Undo Rename content/identity mismatch'}
    if(@([DcmNative]::ListItems($hwnd,$app.Id,1017)).Count){throw 'Undone Rename remains in Processed'}
    SelectFile '08-preview.pdf';$v=Viewer $original;Shot '03-rename-undone-and-reopened' 'teal';AddPass 'Undo-Rename-active-PDF-restores-name-bytes-and-journal'
    [DcmNative]::ClickAsync($hwnd,$app.Id,1041);$moved=Join-Path $fixture.destination '08-preview.pdf'
    Wait 'Move completed' {(Test-Path -LiteralPath $moved) -and -not(Test-Path -LiteralPath $original)};Gone $v
    if((Hash $moved) -ne $pdfHash){throw 'Move changed bytes'}
    $moveID=Journal $original $moved 'completed' 2
    SwitchSource $fixture.destination 1;SelectFile '08-preview.pdf';$v=Viewer $moved
    Shot '04-moved-PDF-reopened-before-undo' 'teal';AddPass 'moved-PDF-opened-from-destination'
    SelectUndoRecord '08-preview.pdf';[DcmNative]::ClickAsync($hwnd,$app.Id,1008)
    Wait 'Undo Move completed' {(Test-Path -LiteralPath $original) -and -not(Test-Path -LiteralPath $moved)};Gone $v
    if((Hash $original) -ne $pdfHash -or (Journal $original $moved 'undone' 2) -ne $moveID){throw 'Undo Move content/identity mismatch'}
    AddPass 'Undo-Move-active-destination-PDF-restores-bytes-and-journal'
    # The selected destination file was moved away by Undo. No stale target may survive.
    Wait 'destination clears stale selection after Undo' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 0 -and (Text 1005) -ceq ''}
    if($null -ne [DcmSumatraProbe]::Find($hwnd,$app.Id)){throw 'Stale destination viewer after Undo'}
    Exclusive $original;Shot '05-destination-empty-after-undo';AddPass 'Undo-Move-refreshes-viewed-destination-and-clears-target'
    SwitchSource $fixture.inbox 4;SelectFile '08-preview.pdf';$v=Viewer $original
    Shot '06-restored-PDF-before-viewer-failure' 'teal'
    $beforeJournal=Hash $journal
    # Deliberate failure of only this private, recorded viewer. Never kill by process name.
    $p=Get-Process -Id $v.pid;$w=Get-CimInstance Win32_Process -Filter ('ProcessId='+$v.pid)
    if($p.Path -ne $runtimeExe -or $p.StartTime.ToUniversalTime().ToString('o') -ne $v.start -or $w.ParentProcessId -ne $app.Id){throw 'Refusing failure injection: viewer identity changed'}
    $p.Kill();if(-not $p.WaitForExit(5000)){throw 'Owned injected-failure process did not exit'};$p.Dispose();Gone $v
    Wait 'viewer failure reported' {(Status) -like '*owned Sumatra viewer exited*'}
    if($null -ne [DcmSumatraProbe]::Find($hwnd,$app.Id)){throw 'Viewer remained after failure'}
    Exclusive $original
    if((Hash $journal) -ne $beforeJournal -or (Hash $original) -ne $pdfHash){throw 'Viewer failure mutated data/history'}
    Shot '07-viewer-exit-error';AddPass 'controlled-viewer-exit-is-reported-with-data-unchanged'
    [DcmNative]::ClickAsync($hwnd,$app.Id,1011);$v=Viewer $original
    Shot '08-reload-recovers-correct-PDF' 'teal';AddPass 'Reload-recovers-PDF-after-viewer-exit'
    foreach($f in $otherHashes.Keys){if((Hash (Join-Path $fixture.inbox $f)) -ne $otherHashes[$f]){throw 'Unselected file changed'}}
    if(@(Get-ChildItem -LiteralPath $fixture.inbox -File).Count -ne 4 -or @(Get-ChildItem -LiteralPath $fixture.destination -Force).Count){throw 'Unexpected final file set'}
    if(@([DcmNative]::ListItems($hwnd,$app.Id,1017)).Count){throw 'Undone history still displayed'}
    $null=Journal $original $renamed 'undone' 2;$null=Journal $original $moved 'undone' 2
    [DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000) -or $app.ExitCode -ne 0){throw 'Host did not close normally'}
    $receipt['exit_code']=$app.ExitCode;$app.Dispose();$app=$null
    foreach($v in $viewers){$p=Get-Process -Id $v.pid -ErrorAction SilentlyContinue;if($p -and $p.StartTime.ToUniversalTime().ToString('o') -eq $v.start){throw 'Owned viewer still alive'}}
    if(@(Get-ChildItem -LiteralPath $lab -Directory -Filter 'dcm-sumatra-*').Count){throw 'Viewer profile remains'}
    Exclusive $original
    if((Hash $runtimeExe) -ne $receipt.runtime_sha256 -or (Hash (Join-Path $runtimeDir 'sumatrapdfrestrict.ini')) -ne $receipt.policy_sha256){throw 'Runtime/policy changed'}
    CheckSource;$receipt['source_unchanged']=$true;$receipt['fixtures_restored']=$true;$receipt['journal_final']='two original transactions undone';$receipt.passed=$true
    AddPass 'normal-close-no-owned-viewer-profile-or-PDF-lock'
}catch{$receipt['error']=$_.Exception.Message;Write-Output ('FAIL='+$_.Exception.Message);if($app -and -not $app.HasExited){try{Shot 'failure'}catch{$receipt['capture_error']=$_.Exception.Message}}}
finally{
    if($app){try{Own;[DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000)){throw 'Owned host cleanup timeout'};$receipt.passed=$false;$receipt['cleanup_exit']=$app.ExitCode}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
