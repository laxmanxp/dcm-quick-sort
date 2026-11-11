[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,[string]$Runtime=$env:DCM_SUMATRA_SOURCE_EXE)
# User-approved optional provider walking skeleton; only isolated fixtures/runtime.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt -or -not $Runtime){throw 'Workspace, exact build receipt and explicit runtime required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-sumatra-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]';$viewers=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-SUMATRA-01';passed=$false;report=$lab;cases=$cases;captures=$shots;viewers=$viewers;visual_reviewed=$false}
$app=$null;$hwnd=[IntPtr]::Zero;$head='';$manifest=@();$fixture=$null
Write-Output ('REPORT_ROOT='+$lab)
function Hash([string]$p){return (Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Json([string]$p,$v){[IO.File]::WriteAllText($p,($v|ConvertTo-Json -Depth 25),$utf8)}
function Git([string[]]$a){$v=(& git.exe -C $repo @a|Out-String).Trim();if($LASTEXITCODE -ne 0){throw 'Git failed'};return $v}
function CheckSource {if((Git @('rev-parse','HEAD')) -ne $head -or (Git @('status','--porcelain=v1','--untracked-files=all'))){throw 'Source identity changed'};foreach($f in $manifest){if((Hash (Join-Path $repo $f.path)) -ne $f.sha256){throw 'Source manifest changed'}}}
function Own {$app.Refresh();if($app.HasExited -or $app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $receipt.host_start){throw 'Owned app identity/liveness changed'};[DcmNative]::RequireOwned($hwnd,$app.Id)}
function Wait([string]$Label,[scriptblock]$Check){$until=[DateTime]::UtcNow.AddSeconds(20);do{Own;if(& $Check){return};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until);throw ('Deadline: '+$Label)}
function Text([int]$id){return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,$id),$app.Id)}
function SelectFile([string]$file){$null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@($file));$stem=[IO.Path]::GetFileNameWithoutExtension($file);Wait ('selection '+$file) {(Text 1005) -ceq $stem}}
function Shot([string]$name,[string]$color=''){
    Own;$area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea;[DcmNative]::Restore($hwnd,$app.Id,$area.Left+8,$area.Top+8);Start-Sleep -Milliseconds 400
    $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false)
    $shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);teal=$s.TealSamples;orange=$s.OrangeSamples;visual_reviewed=$false})
    if($color -eq 'teal' -and ($s.TealSamples -lt 1000 -or $s.OrangeSamples -gt 100)){throw 'Wrong visible PDF page: expected teal'}
    if($color -eq 'orange' -and ($s.OrangeSamples -lt 1000 -or $s.TealSamples -gt 100)){throw 'Wrong visible PDF page: expected orange'}
}
function Viewer {
    Wait 'embedded Sumatra child' { $null -ne [DcmSumatraProbe]::Find($hwnd,$app.Id) }
    $v=[DcmSumatraProbe]::Find($hwnd,$app.Id);$p=Get-Process -Id $v.PID
    $w=Get-CimInstance Win32_Process -Filter ('ProcessId='+$v.PID)
    if($p.Path -ne $runtimeExe -or $w.ParentProcessId -ne $app.Id){throw 'Foreign viewer process'}
    if($w.CommandLine -notmatch '\-restrict' -or $w.CommandLine -notmatch '\-plugin'){throw 'Viewer not restricted/embedded'}
    $entry=[ordered]@{pid=$v.PID;path=$p.Path;start=$p.StartTime.ToUniversalTime().ToString('o');hwnd=$v.Window;command_line=$w.CommandLine};$viewers.Add($entry)
    [DcmSumatraProbe]::CheckGeometry($hwnd,$app.Id,$v.PID);return $entry
}
function Gone($v){Wait 'owned viewer released' {$p=Get-Process -Id $v.pid -ErrorAction SilentlyContinue; -not $p -or $p.StartTime.ToUniversalTime().ToString('o') -ne $v.start}}
function AddPass([string]$n){$cases.Add([ordered]@{name=$n;passed=$true})}
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
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-sumatra-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
    . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
    $original=Join-Path $fixture.inbox '08-preview.pdf';$pdfHash=Hash $original
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
    SelectFile '08-preview.pdf';$v=Viewer;Start-Sleep -Milliseconds 800;Shot '01-embedded-first-page' 'teal';AddPass 'embedded-owned-restricted-first-page'
    [DcmSumatraProbe]::Command($hwnd,$app.Id,$v.pid,259);Start-Sleep -Milliseconds 600;Shot '02-next-page' 'orange';AddPass 'next-page-visible-content'
    [DcmSumatraProbe]::Command($hwnd,$app.Id,$v.pid,259);Start-Sleep -Milliseconds 300;Shot '03-last-boundary' 'orange';AddPass 'last-page-boundary'
    [DcmSumatraProbe]::Command($hwnd,$app.Id,$v.pid,260);Start-Sleep -Milliseconds 400;Shot '04-previous-page' 'teal';AddPass 'previous-page-visible-content'
    [DcmSumatraProbe]::Command($hwnd,$app.Id,$v.pid,201);Start-Sleep -Milliseconds 400
    if(@([DcmNative]::Dialogs([IntPtr]$v.hwnd,$v.pid)).Count){throw 'Restricted viewer opened a file dialog'}
    Shot '05-open-command-disabled' 'teal';AddPass 'alternate-open-disabled'
    SelectFile '01-alpha.txt';Gone $v
    $h=[IO.File]::Open($original,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose();AddPass 'switch-to-text-releases-PDF'
    SelectFile '08-preview.pdf';$v=Viewer
    [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1005,'sumatra-renamed');[DcmNative]::ClickAsync($hwnd,$app.Id,1006)
    $renamed=Join-Path $fixture.inbox 'sumatra-renamed.pdf';Wait 'tracked rename' {(Test-Path -LiteralPath $renamed) -and -not(Test-Path -LiteralPath $original)};Gone $v
    if((Hash $renamed) -ne $pdfHash){throw 'Rename changed PDF bytes'};AddPass 'rename-releases-viewer-and-preserves-bytes'
    Wait 'renamed selection' {(Text 1005) -ceq 'sumatra-renamed'};$v=Viewer
    [DcmNative]::ClickAsync($hwnd,$app.Id,1041);$moved=Join-Path $fixture.destination 'sumatra-renamed.pdf'
    Wait 'tracked move' {(Test-Path -LiteralPath $moved) -and -not(Test-Path -LiteralPath $renamed)};Gone $v
    if((Hash $moved) -ne $pdfHash){throw 'Move changed PDF bytes'};AddPass 'move-releases-viewer-and-preserves-bytes'
    SelectFile '01-alpha.txt';Shot '06-moved-and-text-restored'
    foreach($f in $otherHashes.Keys){if((Hash (Join-Path $fixture.inbox $f)) -ne $otherHashes[$f]){throw 'Unselected text changed'}}
    [DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000) -or $app.ExitCode -ne 0){throw 'App did not close normally'}
    $receipt['exit_code']=$app.ExitCode;$app.Dispose();$app=$null
    foreach($v in $viewers){$p=Get-Process -Id $v.pid -ErrorAction SilentlyContinue;if($p -and $p.StartTime.ToUniversalTime().ToString('o') -eq $v.start){throw 'Orphaned owned Sumatra process'}}
    CheckSource;$receipt['source_unchanged']=$true;$receipt['fixtures_preserved']=$true;$receipt.passed=$true
}catch{$receipt['error']=$_.Exception.Message;Write-Output ('FAIL='+$_.Exception.Message);if($app -and -not $app.HasExited){try{Shot 'failure'}catch{$receipt['capture_error']=$_.Exception.Message}}}
finally{
    if($app){try{Own;[DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000)){throw 'Owned app cleanup timeout'};$receipt.passed=$false;$receipt['cleanup_exit']=$app.ExitCode}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
