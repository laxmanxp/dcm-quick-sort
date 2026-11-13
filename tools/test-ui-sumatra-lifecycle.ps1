[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,[string]$Runtime=$env:DCM_SUMATRA_SOURCE_EXE,[switch]$CloseWhileLoading)
# Focused, source-bound lifecycle acceptance; no user documents or installed runtime changes.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt -or -not $Runtime){throw 'Workspace, build receipt and pinned runtime required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-sumatra-lifecycle-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Lab must be outside source.'}
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]';$viewers=New-Object 'Collections.Generic.List[object]';$transitions=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-SUMATRA-LIFECYCLE-01';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;cases=$cases;captures=$shots;viewers=$viewers;transitions=$transitions;visual_reviewed=$false}
$app=$null;$hwnd=[IntPtr]::Zero;$head='';$manifest=@();$expected=@{};$fixture=$null;$runtimeExe='';$exe=''
Write-Output ('REPORT_ROOT='+$lab)
function Hash([string]$p){return (Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Json([string]$p,$value){[IO.File]::WriteAllText($p,($value|ConvertTo-Json -Depth 30),$utf8)}
function Git([string[]]$Arguments){$value=(& git.exe -C $repo @Arguments|Out-String).Trim();if($LASTEXITCODE){throw 'Git query failed.'};return $value}
function CheckSource {
    if((Git @('rev-parse','HEAD')) -ne $head -or (Git @('status','--porcelain=v1','--untracked-files=all'))){throw 'Source checkout changed.'}
    foreach($f in $manifest){$p=[IO.Path]::GetFullPath((Join-Path $repo $f.path));if(-not $p.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $p) -ne $f.sha256){throw 'Source manifest changed.'}}
}
function Own {
    $app.Refresh()
    if($app.HasExited -or $app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $receipt.host_start){throw 'Owned app identity/liveness changed.'}
    [DcmNative]::RequireOwned($hwnd,$app.Id)
}
function Wait([string]$label,[scriptblock]$check){$until=[DateTime]::UtcNow.AddSeconds(15);do{Own;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw ('Unexpected owned dialog: '+$label)};if(& $check){return};Start-Sleep -Milliseconds 30}while([DateTime]::UtcNow -lt $until);throw ('Deadline: '+$label)}
function Text([int]$id){return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,$id),$app.Id)}
function Status {return (@([DcmLifecycleProbe]::Status($hwnd,$app.Id)) -join "`n")}
function SelectFile([string]$file){
    Own;$sw=[Diagnostics.Stopwatch]::StartNew()
    $null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@($file))
    $stem=[IO.Path]::GetFileNameWithoutExtension($file)
    if((Text 1005) -cne $stem){throw ('Selection draft mismatch: '+$file)}
    $s=Status;$sw.Stop()
    $transitions.Add([ordered]@{file=$file;milliseconds=$sw.ElapsedMilliseconds;status_on_return=$s})
    return $s
}
function PrivateProcesses {
    foreach($p in @(Get-Process -Name 'SumatraPDF' -ErrorAction SilentlyContinue)){
        try{if($p.Path -eq $runtimeExe){[pscustomobject]@{pid=$p.Id;start=$p.StartTime.ToUniversalTime().ToString('o')}}}catch{if(-not $p.HasExited){throw}}
    }
}
function EmptyViewer {
    Wait 'viewer released' {($null -eq [DcmSumatraProbe]::Find($hwnd,$app.Id)) -and @((PrivateProcesses)).Count -eq 0}
    if(@(Get-ChildItem -LiteralPath $lab -Directory -Filter 'dcm-sumatra-*').Count){throw 'Private viewer profile not cleaned after release.'}
}
function ExclusivePDFs {
    foreach($f in @('08-preview.pdf','09-second.pdf')){$h=[IO.File]::Open((Join-Path $fixture.inbox $f),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()}
}
function CheckFiles {
    $files=@(Get-ChildItem -LiteralPath $fixture.inbox -File)
    if($files.Count -ne $expected.Count){throw 'Fixture file set changed.'}
    foreach($f in $files){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw ('Fixture bytes changed: '+$f.Name)}}
    if(@(Get-ChildItem -LiteralPath $fixture.destination -Force).Count){throw 'Preview-only test changed destination.'}
    $journal=Join-Path $fixture.config 'recovery\move-journal.jsonl'
    if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Preview-only test appended a file operation.'}
}
function Viewer([string]$file){
    Wait ('attach '+$file) {$null -ne [DcmSumatraProbe]::Find($hwnd,$app.Id)}
    # Attachment can precede the host's asynchronous size request; require both.
    Wait ('ready '+$file) {(Status) -like 'Sumatra PDF preview*'}
    Start-Sleep -Milliseconds 150
    $v=[DcmSumatraProbe]::Find($hwnd,$app.Id);$p=Get-Process -Id $v.PID
    $w=Get-CimInstance Win32_Process -Filter ('ProcessId='+$v.PID)
    $path=Join-Path $fixture.inbox $file
    if($p.Path -ne $runtimeExe -or $w.ParentProcessId -ne $app.Id -or $w.CommandLine -notmatch '\-restrict' -or $w.CommandLine -notmatch '\-plugin' -or -not $w.CommandLine.Contains($path)){throw 'Viewer process/document identity mismatch.'}
    [DcmSumatraProbe]::CheckGeometry($hwnd,$app.Id,$v.PID)
    $entry=[ordered]@{pid=$p.Id;start=$p.StartTime.ToUniversalTime().ToString('o');path=$p.Path;document=$path;hwnd=$v.Window}
    $viewers.Add($entry);return $entry
}
function Gone($v){Wait 'previous viewer exited' {$p=Get-Process -Id $v.pid -ErrorAction SilentlyContinue;(-not $p) -or $p.StartTime.ToUniversalTime().ToString('o') -ne $v.start}}
function TextReady {
    EmptyViewer
    Wait 'correct text preview' {$p=[DcmTextProbe]::Preview($hwnd,$app.Id);$p -and $p.Text -ceq $textExpected}
    if((Status) -notlike 'Plain-text preview: TXT*'){throw 'Text status overwritten by stale viewer.'}
    ExclusivePDFs
}
function Shot([string]$name,[string]$color=''){
    Own;$area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
    [DcmNative]::Restore($hwnd,$app.Id,$area.Left+8,$area.Top+8);Start-Sleep -Milliseconds 350
    $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false)
    $shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false})
    if($color){
        $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'-content.png')),$true)
        $shots.Add([ordered]@{name=$name+'-content';path=$s.Path;sha256=(Hash $s.Path);teal=$s.TealSamples;orange=$s.OrangeSamples;visual_reviewed=$false})
        if($color -eq 'teal' -and ($s.TealSamples -lt 1000 -or $s.OrangeSamples -gt 100)){throw 'Wrong PDF pixels: expected teal.'}
        if($color -eq 'orange' -and ($s.OrangeSamples -lt 1000 -or $s.TealSamples -gt 100)){throw 'Wrong PDF pixels: expected orange.'}
    }
}
function AddPass([string]$name,$detail=$null){$cases.Add([ordered]@{name=$name;passed=$true;detail=$detail})}
try {
    $build=Get-Content -LiteralPath $BuildReceipt -Raw|ConvertFrom-Json;$head=Git @('rev-parse','HEAD')
    if(-not $build.passed -or $build.mode -ne 'build' -or -not $build.source_unchanged -or $build.commit -ne $head){throw 'Exact clean successful build required.'}
    if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256 -or (Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'Build identities changed.'}
    $manifest=Get-Content -LiteralPath $build.source_hash_manifest.path -Raw|ConvertFrom-Json;CheckSource
    $receipt['commit']=$head;$receipt['build_receipt']=$BuildReceipt;$receipt['app_sha256']=$build.app_artifact.sha256
    if((Hash $Runtime) -ne '719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0'){throw 'Pinned runtime required.'}
    $runtimeDir=Join-Path $lab 'runtime';$null=New-Item -ItemType Directory -Path $runtimeDir
    $runtimeExe=Join-Path $runtimeDir 'SumatraPDF.exe';Copy-Item -LiteralPath $Runtime -Destination $runtimeExe
    $policy="[Policies]`nInternetAccess = 0`nDiskAccess = 0`nSavePreferences = 0`nRegistryAccess = 0`nPrinterAccess = 0`nCopySelection = 0`nFullscreenAccess = 0`nLinkProtocols =`nSafeFileTypes =`n"
    $policyPath=Join-Path $runtimeDir 'sumatrapdfrestrict.ini';[IO.File]::WriteAllText($policyPath,$policy,$utf8)
    $receipt['runtime_sha256']=Hash $runtimeExe;$receipt['policy_sha256']=Hash $policyPath
    Add-Type -AssemblyName System.Drawing;Add-Type -AssemblyName System.Windows.Forms
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-text-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-lifecycle-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-sumatra-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','Accessibility.dll','System.Drawing.dll')
    . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
    # Equal-length replacements preserve the existing valid PDF xref/stream lengths.
    $pdfA=[IO.File]::ReadAllText((Join-Path $fixture.inbox '08-preview.pdf'),[Text.Encoding]::ASCII)
    $pdfB=$pdfA.Replace('0.02 0.55 0.65','0.90 0.35 0.08').Replace('PAGE ONE - TEAL','FILE TWO - GOLD')
    if($pdfA.Length -ne $pdfB.Length -or $pdfA -ceq $pdfB){throw 'Distinct PDF fixture construction failed.'}
    [IO.File]::WriteAllText((Join-Path $fixture.inbox '09-second.pdf'),$pdfB,[Text.Encoding]::ASCII)
    [IO.File]::WriteAllBytes((Join-Path $fixture.inbox '10-empty.pdf'),[byte[]]@())
    [IO.File]::WriteAllText((Join-Path $fixture.inbox '11-invalid.pdf'),'This disposable file intentionally lacks a PDF header.',$utf8)
    Get-ChildItem -LiteralPath $fixture.inbox -File|ForEach-Object {$expected[$_.Name]=Hash $_.FullName}
    $textExpected=[IO.File]::ReadAllText((Join-Path $fixture.inbox '01-alpha.txt'))
    $receipt['initial_hashes']=$expected
    $exe=Join-Path $lab ('DCM-Life-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
    $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming
    $psi.EnvironmentVariables['TEMP']=$lab;$psi.EnvironmentVariables['TMP']=$lab
    if($CloseWhileLoading){$psi.EnvironmentVariables['DCM_STARTUP_TRACE']='1'}
    $psi.EnvironmentVariables['DCM_PDF_PROVIDER']='sumatra';$psi.EnvironmentVariables['DCM_SUMATRA_EXE']=$runtimeExe
    $psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'App start failed.'}
    $receipt['host_pid']=$app.Id;$receipt['host_start']=$app.StartTime.ToUniversalTime().ToString('o')
    $until=[DateTime]::UtcNow.AddSeconds(15);do{$app.Refresh();$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 50}while([DateTime]::UtcNow -lt $until)
    Wait 'seven fixture rows' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 7}
    $null=SelectFile '01-alpha.txt';TextReady;AddPass 'initial-text-without-viewer'
    if(-not $CloseWhileLoading){
    $null=SelectFile '08-preview.pdf';$a=Viewer '08-preview.pdf';Shot '01-pdf-A' 'teal';AddPass 'pdf-A-owned-content'
    $null=SelectFile '09-second.pdf';$b=Viewer '09-second.pdf';Gone $a;Shot '02-direct-switch-B' 'orange';AddPass 'direct-PDF-to-PDF-replaces-content-and-process'
    $null=SelectFile '01-alpha.txt';Gone $b;TextReady;Shot '03-text-after-PDF';AddPass 'switch-to-text-releases-both-PDFs'
    # Repeated immediate requests without waiting for a viewer window or render.
    $loading=0;$cancelStates=New-Object 'Collections.Generic.List[object]'
    for($i=0;$i -lt 10;$i++){
        $file=if($i%2 -eq 0){'08-preview.pdf'}else{'09-second.pdf'}
        $s=SelectFile $file;if($s -like 'Loading PDF*'){$loading++}
        $null=SelectFile '01-alpha.txt';TextReady
        $cancelStates.Add([ordered]@{iteration=$i;file=$file;status_before_abandon=$s;after=(Status)})
    }
    $receipt['cancellation']=$cancelStates
    if($loading -lt 1){throw 'No in-flight loading state observed; cancellation not established.'}
    Start-Sleep -Milliseconds 1000;TextReady;Shot '04-loading-cancelled-to-text'
    AddPass 'ten-immediate-abandons-no-stale-viewer-or-status' @{observed_loading=$loading;attempts=10;settle_ms=1000}
    # Repeated attached PDF/text/PDF changes: no extra dwell between transitions.
    for($i=0;$i -lt 4;$i++){
        $null=SelectFile '08-preview.pdf';$a=Viewer '08-preview.pdf'
        $null=SelectFile '09-second.pdf';$b=Viewer '09-second.pdf';Gone $a
        $null=SelectFile '01-alpha.txt';Gone $b;TextReady
    }
    AddPass 'four-attached-PDF-PDF-text-cycles-preserve-targets-and-release'
    $null=SelectFile '10-empty.pdf';EmptyViewer
    if((Status) -notlike 'Sumatra preview unavailable:*nonempty regular file*'){throw 'Empty PDF error missing.'}
    Shot '05-empty-PDF-error';AddPass 'empty-PDF-truthful-error-no-viewer'
    $null=SelectFile '01-alpha.txt';TextReady;AddPass 'text-recovery-after-empty-PDF'
    $null=SelectFile '11-invalid.pdf';EmptyViewer
    if((Status) -notlike 'Sumatra preview unavailable:*no PDF header*'){throw 'Invalid PDF error missing.'}
    Shot '06-invalid-PDF-error';AddPass 'invalid-header-truthful-error-no-viewer'
    $null=SelectFile '09-second.pdf';$last=Viewer '09-second.pdf';Shot '07-valid-PDF-recovery-before-close' 'orange';AddPass 'valid-PDF-recovery-after-invalid-header'
    }else{
        $s=SelectFile '08-preview.pdf'
        if($s -notlike 'Loading PDF*'){throw 'Pending-close test did not observe loading; no acceptance claimed.'}
        $receipt['status_at_close_request']=$s
        AddPass 'loading-observed-before-close-request'
    }
    CheckFiles
    $sw=[Diagnostics.Stopwatch]::StartNew();[DcmNative]::CloseAsync($hwnd,$app.Id)
    if(-not $app.WaitForExit(15000) -or $app.ExitCode -ne 0){throw 'Active-PDF host did not close normally.'}
    $receipt['exit_code']=$app.ExitCode;$sw.Stop();$receipt['close_ms']=$sw.ElapsedMilliseconds
    $app.Dispose();$app=$null
    $until=[DateTime]::UtcNow.AddSeconds(5);do{if(@((PrivateProcesses)).Count -eq 0){break};Start-Sleep -Milliseconds 50}while([DateTime]::UtcNow -lt $until)
    if(@((PrivateProcesses)).Count){throw 'Private viewer survived host close.'}
    foreach($v in $viewers){$p=Get-Process -Id $v.pid -ErrorAction SilentlyContinue;if($p -and $p.StartTime.ToUniversalTime().ToString('o') -eq $v.start){throw 'Recorded viewer survived host close.'}}
    if(@(Get-ChildItem -LiteralPath $lab -Directory -Filter 'dcm-sumatra-*').Count){throw 'Viewer profile survived host close.'}
    ExclusivePDFs;CheckFiles;CheckSource
    if((Hash $runtimeExe) -ne $receipt.runtime_sha256 -or (Hash $policyPath) -ne $receipt.policy_sha256 -or (Hash $exe) -ne $receipt.app_sha256){throw 'Executable/policy changed during test.'}
    if($CloseWhileLoading){
        $tracePath=Join-Path $lab 'dcm-startup-trace.json'
        if(-not(Test-Path $tracePath)){throw 'Pending-close trace missing'}
        $receipt['startup_trace']=Get-Content $tracePath -Raw|ConvertFrom-Json
        $receipt['startup_trace_sha256']=Hash $tracePath
        AddPass 'close-during-loading-drains-preparation-and-private-resources'
    }else{AddPass 'close-with-active-PDF-drains-viewers-profiles-and-file-handles'}
    $receipt['source_unchanged']=$true;$receipt['fixtures_preserved']=$true;$receipt['journal_empty']=$true;$receipt['private_viewers_remaining']=0;$receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage
    Write-Output ('FAIL='+$_.Exception.Message)
    if($app -and -not $app.HasExited){try{Shot 'failure';$receipt['failure_status']=Status}catch{$receipt['capture_error']=$_.Exception.Message}}
}finally{
    if($app){try{Own;[DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000)){throw 'Owned cleanup timeout.'};$receipt['cleanup_exit']=$app.ExitCode;$app.Dispose()}catch{$receipt['cleanup_error']=$_.Exception.Message};$receipt.passed=$false}
    try{if($fixture){CheckFiles;$receipt['fixtures_preserved']=$true};if($head){CheckSource;$receipt['source_unchanged']=$true}}catch{$receipt.passed=$false;$receipt['integrity_error']=$_.Exception.Message}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
