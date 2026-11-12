[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,[string]$Runtime=$env:DCM_SUMATRA_SOURCE_EXE)
# Source-bound readable 100-percent/Fit-Width scrolling acceptance.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt -or -not $Runtime){throw 'Workspace, exact build receipt and explicit runtime required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-sumatra-scroll-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]';$viewers=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-SUMATRA-SCROLL-01';passed=$false;report=$lab;cases=$cases;captures=$shots;viewers=$viewers;visual_reviewed=$false}
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
function Shot([string]$name,$region){
    Own;[DcmSumatraProbe]::CheckGeometry($hwnd,$app.Id,$viewer.pid)
    $area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
    [DcmNative]::Restore($hwnd,$app.Id,$area.Left+8,$area.Top+8);Start-Sleep -Milliseconds 400
    $state=[DcmScrollProbe]::State($hwnd,$app.Id,$viewer.pid)
    if($state.PageNumber -ne $region.page){throw ('Wrong document page before '+$name)}
    $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false)
    $ink=[DcmScrollProbe]::Inspect($hwnd,$app.Id,$viewer.pid,$s.Path,$s.Left,$s.Top,$region.rgb[0],$region.rgb[1],$region.rgb[2])
    $shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);expected_text=$region.text;scroll=$state;ink=$ink;visual_reviewed=$false})
    if(-not $ink.MarkerFullyVisible -or $ink.MarkerPixels -lt 100 -or $ink.TextInkPixels -lt 250 -or $ink.TextInkRows -lt 12){throw ('Readable fixture text block absent or clipped: '+$name)}
}
function ScrollTo([double]$fraction){
    $before=[DcmScrollProbe]::State($hwnd,$app.Id,$viewer.pid)
    if($before.Limit-$before.Minimum -lt 200){throw 'Fixture is not vertically scrollable at the selected zoom'}
    $target=[int]($before.Minimum+($before.Limit-$before.Minimum)*$fraction)
    if($fraction -eq 0 -or $fraction -eq 1){
        $command=if($fraction -eq 0){6}else{7}
        [DcmScrollProbe]::Scroll($hwnd,$app.Id,$viewer.pid,$command)
        Wait 'scroll boundary' { $p=[DcmScrollProbe]::State($hwnd,$app.Id,$viewer.pid);$p.Position -eq $target }
    }else{
        for($i=0;$i -lt 100;$i++){
            $p=[DcmScrollProbe]::State($hwnd,$app.Id,$viewer.pid)
            if($p.Position -ge $target){break}
            [DcmScrollProbe]::Scroll($hwnd,$app.Id,$viewer.pid,1);Start-Sleep -Milliseconds 60
        }
        if($p.Position -lt $target){throw 'Line scrolling did not reach the middle'}
    }
    Start-Sleep -Milliseconds 300
    $after=[DcmScrollProbe]::State($hwnd,$app.Id,$viewer.pid)
    $scrolls.Add([ordered]@{fraction=$fraction;before=$before;after=$after})
    if($after.PageNumber -ne $before.PageNumber){throw 'In-page scrolling changed PDF page unexpectedly'}
    if($before.Position -lt $target -and $after.Position -le $before.Position){throw 'No downward scroll movement'}
    if($before.Position -gt $target -and $after.Position -ge $before.Position){throw 'No upward scroll movement'}
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
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-scroll-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
    $spec=Get-Content -LiteralPath (Join-Path $repo 'tests/fixtures/sumatra-scroll.json') -Raw|ConvertFrom-Json
    $source=Join-Path $repo 'tests/fixtures/sumatra-scroll.pdf'
    if((Hash $source) -ne $spec.sha256){throw 'Readable-scroll fixture identity changed'}
    $document=Join-Path $fixture.inbox '09-scroll.pdf';Copy-Item -LiteralPath $source -Destination $document
    $expected=@{};Get-ChildItem -LiteralPath $fixture.inbox -File|ForEach-Object {$expected[$_.Name]=Hash $_.FullName}
    $receipt['fixture_spec']=$spec;$receipt['initial_hashes']=$expected
    $exe=Join-Path $lab ('DCM-Sumatra-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
    $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming
    $psi.EnvironmentVariables['TEMP']=$lab;$psi.EnvironmentVariables['TMP']=$lab
    $psi.EnvironmentVariables['DCM_PDF_PROVIDER']='sumatra';$psi.EnvironmentVariables['DCM_SUMATRA_EXE']=$runtimeExe
    $psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'App start failed'}
    $receipt['host_pid']=$app.Id;$receipt['host_start']=$app.StartTime.ToUniversalTime().ToString('o')
    $until=[DateTime]::UtcNow.AddSeconds(15);do{$app.Refresh();$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    Wait 'fixture list' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 5}
    [DcmSumatraProbe]::SetEdit($hwnd,$app.Id,1003,'Accepted')

    SelectFile '09-scroll.pdf';$viewer=Viewer $document
    $scrolls=New-Object 'Collections.Generic.List[object]';$receipt['scroll_events']=$scrolls
    foreach($mode in @(@{name='100-percent';command=275},@{name='fit-width';command=276})){
        [DcmSumatraProbe]::SetPage($hwnd,$app.Id,$viewer.pid,1)
        Wait 'page one' {([DcmScrollProbe]::State($hwnd,$app.Id,$viewer.pid)).PageNumber -eq 1}
        [DcmSumatraProbe]::Command($hwnd,$app.Id,$viewer.pid,$mode.command);Start-Sleep -Milliseconds 400
        ScrollTo 0;Shot ($mode.name+'-01-top') $spec.regions[0];AddPass ($mode.name+'-page-one-top-readable')
        ScrollTo 0.5;Shot ($mode.name+'-02-middle') $spec.regions[1];AddPass ($mode.name+'-page-one-middle-readable')
        ScrollTo 1;Shot ($mode.name+'-03-bottom') $spec.regions[2];AddPass ($mode.name+'-page-one-bottom-readable')
        ScrollTo 0;Shot ($mode.name+'-04-back-to-top') $spec.regions[0];AddPass ($mode.name+'-reverse-scroll-restores-top')
        [DcmSumatraProbe]::SetPage($hwnd,$app.Id,$viewer.pid,2)
        Wait 'page two' {([DcmScrollProbe]::State($hwnd,$app.Id,$viewer.pid)).PageNumber -eq 2}
        ScrollTo 0;Shot ($mode.name+'-05-page-two-top') $spec.regions[3];AddPass ($mode.name+'-page-two-top-distinct-content')
        ScrollTo 1;Shot ($mode.name+'-06-page-two-bottom') $spec.regions[5];AddPass ($mode.name+'-page-two-bottom-distinct-content')
    }
    SelectFile '01-alpha.txt';Gone $viewer
    Wait 'text restored' {(Status) -like 'Plain-text preview: TXT*'}
    if($null -ne [DcmSumatraProbe]::Find($hwnd,$app.Id)){throw 'Stale PDF viewer after text selection'}
    $h=[IO.File]::Open($document,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()
    AddPass 'switch-to-text-releases-scroll-PDF'
    [DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000) -or $app.ExitCode -ne 0){throw 'Host did not close normally'}
    $receipt['exit_code']=$app.ExitCode;$app.Dispose();$app=$null
    foreach($v in $viewers){$p=Get-Process -Id $v.pid -ErrorAction SilentlyContinue;if($p -and $p.StartTime.ToUniversalTime().ToString('o') -eq $v.start){throw 'Owned viewer survived normal close'}}
    if(@(Get-ChildItem -LiteralPath $lab -Directory -Filter 'dcm-sumatra-*').Count){throw 'Viewer profile remains'}
    $files=@(Get-ChildItem -LiteralPath $fixture.inbox -File)
    if($files.Count -ne $expected.Count){throw 'Fixture file set changed'}
    foreach($f in $files){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Fixture bytes changed'}}
    if(@(Get-ChildItem -LiteralPath $fixture.destination -Force).Count){throw 'Preview test moved files'}
    $journal=Join-Path $fixture.config 'recovery/move-journal.jsonl'
    if((Test-Path $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Preview test created operation history'}
    if((Hash $runtimeExe) -ne $receipt.runtime_sha256 -or (Hash (Join-Path $runtimeDir 'sumatrapdfrestrict.ini')) -ne $receipt.policy_sha256){throw 'Runtime/policy changed'}
    $h=[IO.File]::Open($document,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()
    CheckSource;$receipt['source_unchanged']=$true;$receipt['fixtures_preserved']=$true;$receipt['journal_empty']=$true
    AddPass 'normal-close-cleanup-and-original-bytes';$receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message;Write-Output ('FAIL='+$_.Exception.Message)
    if($app -and -not $app.HasExited){try{
        $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab 'failure.png'),$false)
        $shots.Add([ordered]@{name='failure';path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false})
    }catch{$receipt['capture_error']=$_.Exception.Message}}
}finally{
    if($app){try{Own;[DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000)){throw 'Owned host cleanup timeout'};$receipt.passed=$false;$receipt['cleanup_exit']=$app.ExitCode}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
