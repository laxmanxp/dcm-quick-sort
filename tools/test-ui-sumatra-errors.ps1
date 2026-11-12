[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,[string]$Runtime=$env:DCM_SUMATRA_SOURCE_EXE)
# Source-bound PDF error/recovery acceptance; disposable documents/runtime only.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt -or -not $Runtime){throw 'Workspace, exact build receipt and explicit runtime required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-sumatra-errors-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]';$viewers=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-SUMATRA-ERRORS-01';passed=$false;report=$lab;cases=$cases;captures=$shots;viewers=$viewers;visual_reviewed=$false}
$issues=New-Object 'Collections.Generic.List[string]';$receipt['issues']=$issues
$observations=New-Object 'Collections.Generic.List[object]';$receipt['observations']=$observations
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
        if($color -eq 'teal' -and ($s.TealSamples -lt 1000 -or $s.OrangeSamples -gt 100)){throw 'Expected original teal PDF page'}
        if($color -eq 'orange' -and ($s.OrangeSamples -lt 1000 -or $s.TealSamples -gt 100)){throw 'Expected decrypted orange PDF page'}
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

function PrivateViewer {
    $matches=@(Get-Process -Name 'SumatraPDF' -ErrorAction SilentlyContinue | Where-Object {try{$_.Path -eq $runtimeExe}catch{$false}})
    if($matches.Count -gt 1){throw 'More than one private viewer process'}
    if($matches.Count -eq 0){return $null}
    $p=$matches[0];$w=Get-CimInstance Win32_Process -Filter ('ProcessId='+$p.Id)
    if($w.ParentProcessId -ne $app.Id -or $w.CommandLine -notmatch '\-restrict' -or $w.CommandLine -notmatch '\-plugin'){throw 'Private viewer ownership/restrictions changed'}
    return $p
}
function NoViewer {
    Wait 'private viewer released' {($null -eq (PrivateViewer)) -and ($null -eq [DcmSumatraProbe]::Find($hwnd,$app.Id))}
    if(@(Get-ChildItem -LiteralPath $lab -Directory -Filter 'dcm-sumatra-*').Count){throw 'Private profile remains after release'}
}
function TextRecovery {
    SelectFile '01-alpha.txt';NoViewer
    Wait 'exact text recovery' {$p=[DcmTextProbe]::Preview($hwnd,$app.Id);$p -and $p.Text -ceq $textExpected}
    if((Status) -notlike 'Plain-text preview: TXT*'){throw 'Stale error status on text'}
}
function Observe([string]$name){
    $p=PrivateViewer;$items=@();if($p){$items=@([DcmPdfErrorProbe]::Windows($p.Id))}
    $o=[ordered]@{name=$name;status=(Status);draft=(Text 1005);viewer_pid=$(if($p){$p.Id}else{0});windows=$items}
    $observations.Add($o);Json (Join-Path $lab ($name+'-windows.json')) $o
    return $p
}
function CheckFiles {
    $files=@(Get-ChildItem -LiteralPath $fixture.inbox -File)
    if($files.Count -ne $expected.Count){throw 'Changed fixture set'}
    foreach($f in $files){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Fixture modified'}}
    if(@(Get-ChildItem -LiteralPath $fixture.destination -Force).Count){throw 'Preview test changed destination'}
    if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Preview test created operation history'}
}
function ExclusiveAll {
    foreach($p in @(Get-ChildItem -LiteralPath $fixture.inbox -Filter '*.pdf')){$h=[IO.File]::Open($p.FullName,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()}
}

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
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-pdf-error-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-text-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
    . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
    $original=Join-Path $fixture.inbox '08-preview.pdf';$pdfHash=Hash $original
    $journal=Join-Path $fixture.config 'recovery\move-journal.jsonl'
    # This fixture has a real PDF signature but no pages/catalog/trailer; preflight header check must not masquerade as decoder coverage.
    [IO.File]::WriteAllText((Join-Path $fixture.inbox '10-damaged.pdf'),"%PDF-1.7`n1 0 obj`n<< /Type /Catalog /Pages 99 0 R >>`nendobj`n",$utf8)
    $protected=Get-Content -LiteralPath (Join-Path $repo 'tests/fixtures/sumatra-password.json') -Raw|ConvertFrom-Json
    $protectedPath=Join-Path $fixture.inbox $protected.file
    [IO.File]::WriteAllBytes($protectedPath,[Convert]::FromBase64String($protected.base64))
    if((Hash $protectedPath) -ne $protected.sha256){throw 'Encrypted fixture identity changed'}
    $expected=@{};Get-ChildItem -LiteralPath $fixture.inbox -File|ForEach-Object {$expected[$_.Name]=Hash $_.FullName}
    $textExpected=[IO.File]::ReadAllText((Join-Path $fixture.inbox '01-alpha.txt'))
    $receipt['initial_hashes']=$expected
    # Remove only our disposable runtime before the first PDF selection.
    $runtimeStash=Join-Path $runtimeDir 'SumatraPDF.absent';Move-Item -LiteralPath $runtimeExe -Destination $runtimeStash
    $exe=Join-Path $lab ('DCM-Sumatra-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
    $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming
    $psi.EnvironmentVariables['TEMP']=$lab;$psi.EnvironmentVariables['TMP']=$lab
    $psi.EnvironmentVariables['DCM_PDF_PROVIDER']='sumatra';$psi.EnvironmentVariables['DCM_SUMATRA_EXE']=$runtimeExe
    $psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'App start failed'}
    $receipt['host_pid']=$app.Id;$receipt['host_start']=$app.StartTime.ToUniversalTime().ToString('o')
    $until=[DateTime]::UtcNow.AddSeconds(15);do{$app.Refresh();$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    Wait 'fixture list' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 6}
    TextRecovery;AddPass 'text-works-with-missing-runtime'
    SelectFile '08-preview.pdf';Wait 'missing-runtime error' {(Status) -like 'Sumatra preview unavailable*'};NoViewer
    if(-not (Status).Contains('SumatraPDF.exe')){throw 'Missing runtime not identified'}
    Shot '01-runtime-missing';AddPass 'missing-runtime-truthful-error-without-viewer'
    TextRecovery;AddPass 'text-recovers-after-missing-runtime'
    Move-Item -LiteralPath $runtimeStash -Destination $runtimeExe
    SelectFile '08-preview.pdf';$v=Viewer $original;Shot '02-runtime-restored' 'teal';AddPass 'restored-runtime-recovers-correct-PDF'
    SelectFile '10-damaged.pdf';Gone $v;Start-Sleep -Milliseconds 2200
    $p=Observe '03-damaged';Shot '03-damaged'
    if((Status) -like 'Sumatra PDF preview*'){$issues.Add('DAMAGED_PDF_HOST_CLAIMS_READY: header-valid damaged file has no rendered page')}
    TextRecovery;ExclusiveAll;AddPass 'text-and-file-release-recover-after-damaged-PDF'
    SelectFile '12-protected.pdf';Start-Sleep -Milliseconds 1500;$p=Observe '04-password'
    $dialog=if($p){[DcmPdfErrorProbe]::PasswordDialog($p.Id)}else{0}
    if($dialog -eq 0){
        Shot '04-password-no-prompt';$issues.Add('PASSWORD_PROMPT_NOT_VISIBLE');TextRecovery
    }else{
        # Only the verified disposable viewer and known generated fixture are accessed.
        $s=[DcmNative]::Capture([IntPtr]$dialog,$p.Id,(Join-Path $lab '04-password-prompt.png'),$false)
        $shots.Add([ordered]@{name='04-password-prompt';path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false})
        [DcmPdfErrorProbe]::Password($dialog,$p.Id,'',$true);Start-Sleep -Milliseconds 400
        TextRecovery;ExclusiveAll;Shot '05-password-cancelled-to-text';AddPass 'password-prompt-cancel-and-text-recovery'
        SelectFile '12-protected.pdf';Start-Sleep -Milliseconds 1500;$p=PrivateViewer
        if(-not $p){throw 'No private password viewer on retry'};$dialog=[DcmPdfErrorProbe]::PasswordDialog($p.Id)
        if($dialog -eq 0){throw 'Password dialog not returned on retry'}
        [DcmPdfErrorProbe]::Password($dialog,$p.Id,$protected.password,$false)
        $v=Viewer $protectedPath;Shot '06-password-unlocked' 'orange';AddPass 'known-fixture-password-renders-correct-content'
        TextRecovery;ExclusiveAll;AddPass 'decrypted-viewer-releases-file'
    }
    SelectFile '08-preview.pdf';$v=Viewer $original;Shot '07-final-valid-recovery' 'teal';CheckFiles
    [DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000) -or $app.ExitCode -ne 0){throw 'Host did not close normally'}
    $receipt['exit_code']=$app.ExitCode;$app.Dispose();$app=$null
    foreach($v in $viewers){$p=Get-Process -Id $v.pid -ErrorAction SilentlyContinue;if($p -and $p.StartTime.ToUniversalTime().ToString('o') -eq $v.start){throw 'Owned viewer remains'}}
    if(@(Get-ChildItem -LiteralPath $lab -Directory -Filter 'dcm-sumatra-*').Count){throw 'Viewer profile remains'}
    ExclusiveAll;CheckFiles
    if((Hash $runtimeExe) -ne $receipt.runtime_sha256 -or (Hash (Join-Path $runtimeDir 'sumatrapdfrestrict.ini')) -ne $receipt.policy_sha256){throw 'Private runtime/policy not restored'}
    CheckSource;$receipt['source_unchanged']=$true;$receipt['fixtures_preserved']=$true
    AddPass 'normal-close-source-bytes-and-private-runtime-restored'
    if($issues.Count){throw ($issues -join '; ')};$receipt.passed=$true
}catch{$receipt['error']=$_.Exception.Message;Write-Output ('FAIL='+$_.Exception.Message);if($app -and -not $app.HasExited){try{Shot 'failure'}catch{$receipt['capture_error']=$_.Exception.Message}}}
finally{
    if($app){try{Own;[DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000)){throw 'Owned host cleanup timeout'};$receipt.passed=$false;$receipt['cleanup_exit']=$app.ExitCode}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
