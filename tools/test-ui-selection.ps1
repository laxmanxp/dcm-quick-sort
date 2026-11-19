[CmdletBinding()]
param(
    [string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,
    [string]$VisionExe=$env:DCM_FLOWUIVISION_EXE
)
# DCM-24: use the existing owned native adapter and FlowUIVision, not host files.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt){throw 'Workspace and exact DCM_BUILD_RECEIPT are required.'}
if(-not $VisionExe){$VisionExe='C:\FlowverseReports\dcm-flowuivision-20261103\FlowUIVision.exe'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$workspace=[IO.Path]::GetFullPath($env:FLOWDEV_WORKSPACE_ROOT)
$id=[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8)
$lab=Join-Path $workspace ('dcm-quicksort-ui-selection-'+$id)
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Evidence must be outside checkout.'}
New-Item -ItemType Directory -Path $lab | Out-Null
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$processes=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;app='dcm-quicksort';feature='DCM-24';suite='selection';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;cases=$cases;captures=$captures;processes=$processes}
$script:app=$null;$script:hwnd=[IntPtr]::Zero;$script:owned=$null
$sourceManifest=@();$expected=@{};$head='';$script:moved=@();$inputs=New-Object 'Collections.Generic.List[object]';$receipt['input_receipts']=$inputs
Write-Output ('REPORT_ROOT='+$lab)
function Write-Json([string]$Path,$Value){[IO.File]::WriteAllText($Path,($Value | ConvertTo-Json -Depth 30),$utf8)}
function Hash([string]$Path){return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()}
function Quote-Argument([string]$Value){return '"'+(($Value -replace '(\\*)"','$1$1\"') -replace '(\\+)$','$1$1')+'"'}
function Git-Value([string[]]$Arguments){$v=(& git.exe -C $repo @Arguments | Out-String).Trim();if($LASTEXITCODE -ne 0){throw 'Git identity query failed.'};return $v}
function Check-Source {
    if((Git-Value @('rev-parse','HEAD')) -ne $head -or (Git-Value @('status','--porcelain=v1','--untracked-files=all'))){throw 'Source HEAD/clean status changed.'}
    foreach($f in $sourceManifest){
        $path=[IO.Path]::GetFullPath((Join-Path $repo $f.path))
        if(-not $path.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Source manifest escaped checkout.'}
        if((Hash $path) -ne $f.sha256){throw ('Source changed: '+$f.path)}
    }
}
function Assert-Owned {
    if($null -eq $script:app){throw 'Owned app is not running.'}
    $script:app.Refresh()
    if($script:app.HasExited){throw 'App exited unexpectedly.'}
    if($script:app.Path -ne $exe -or $script:app.StartTime.ToUniversalTime().ToString('o') -ne $script:owned.start){throw 'Process identity changed.'}
    [DcmNative]::RequireOwned($script:hwnd,$script:app.Id)
    if(@([DcmNative]::Dialogs($script:hwnd,$script:app.Id)).Count){throw 'Unexpected owned dialog; no further input permitted.'}
}
function Wait-For([string]$Name,[scriptblock]$Check){
    $until=[DateTime]::UtcNow.AddSeconds(15)
    do{Assert-Owned;if(& $Check){return};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    throw ('Deadline waiting for '+$Name)
}
function Control-Text([int]$ID){return [DcmNative]::Text([DcmNative]::Control($script:hwnd,$script:app.Id,$ID),$script:app.Id)}
function Assert-Files {
    $wantSource=@($expected.Keys | Where-Object {$_ -notin $script:moved} | Sort-Object)
    $actualSource=@(Get-ChildItem -LiteralPath $inbox -File | Select-Object -ExpandProperty Name | Sort-Object)
    $actualDest=@(Get-ChildItem -LiteralPath $destination -File | Select-Object -ExpandProperty Name | Sort-Object)
    if(($wantSource -join '|') -cne ($actualSource -join '|') -or (($script:moved | Sort-Object) -join '|') -cne ($actualDest -join '|')){throw 'DCM24_OPERATION_SET_MISMATCH: unexpected source/destination file set.'}
    foreach($name in $expected.Keys){$dir=if($name -in $script:moved){$destination}else{$inbox};if((Hash (Join-Path $dir $name)) -ne $expected[$name]){throw ('Fixture bytes changed: '+$name)}}
    if($script:moved.Count -eq 0){
        $journal=Join-Path $config 'recovery\move-journal.jsonl'
        if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Selection-only workflow appended a file operation.'}
    }
}
function Selected-Names {
    $items=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016))
    return @([DcmNative]::SelectedIndexes($script:hwnd,$script:app.Id,1016) | ForEach-Object {$items[$_]})
}
function Click-Row([string]$Name,[int]$Modifier=0){
    Assert-Owned
    $area=[Windows.Forms.Screen]::FromHandle($script:hwnd).WorkingArea
    [DcmNative]::Restore($script:hwnd,$script:app.Id,$area.Left+8,$area.Top+8)
    Start-Sleep -Milliseconds 250
    $items=[string[]]@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016));$index=[Array]::IndexOf($items,$Name)
    if($index -lt 0){throw 'Click target is not in the visible list.'}
    $list=[DcmNative]::Control($script:hwnd,$script:app.Id,1016)
    $inputs.Add([DcmSelectionInput]::ClickRow($script:hwnd,$list,$script:app.Id,$index,$Name,$Modifier))
}
function Assert-Selection([string]$Case,[string[]]$Names,[switch]$Capture){
    Start-Sleep -Milliseconds 200;Assert-Owned
    $selected=@(Selected-Names)
    if(($selected -join '|') -cne ($Names -join '|')){throw ('DCM24_SELECTION_MISMATCH: '+$Case+' expected '+($Names -join ',')+'; actual '+($selected -join ','))}
    $draft=Control-Text 1005
    $snapshot=@([DcmNative]::Snapshot($script:hwnd,$script:app.Id))
    if($Names.Count -eq 1){
        $stem=[IO.Path]::GetFileNameWithoutExtension($Names[0]);$text='DCM-24 fixture: '+$Names[0]+' | Only selected files may move.'
        if($draft -cne $stem){throw ('DCM24_ACTIVE_SELECTION_MISMATCH: '+$Case+' selected '+$Names[0]+' but draft='+$draft)}
        if(-not @($snapshot | Where-Object {$_.Visible -and $_.Class -eq 'Edit' -and $_.Text -ceq $text}).Count){throw 'Single selected file has a mismatched visible preview.'}
    }elseif($Names.Count -eq 0){
        if($draft -ne ''){throw 'Zero selection retained an actionable filename draft.'}
    }else{
        if(-not @($snapshot | Where-Object {$_.Visible -and $_.Text.StartsWith('FILES') -and $_.Text.Contains(([string]$Names.Count+' selected'))}).Count){throw 'Multi-selection count missing from UI.'}
    }
    Assert-Files
    $cases.Add([ordered]@{name=$Case;selected=$selected;selected_count=$selected.Count;draft=$draft;passed=$true})
    if($Capture){Save-State $Case}
}
function Confirm-Move([bool]$Proceed){
    Assert-Owned;[DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
    $until=[DateTime]::UtcNow.AddSeconds(8);$dialog=$null
    do{
        $dialogs=@([DcmNative]::Dialogs($script:hwnd,$script:app.Id))
        if($dialogs.Count -gt 1){throw 'Multiple unexpected dialogs.'}
        if($dialogs.Count -eq 1){$dialog=$dialogs[0];break};Start-Sleep -Milliseconds 100
    }while([DateTime]::UtcNow -lt $until)
    if($null -eq $dialog -or $dialog.Text -notlike 'Move Selected Files to Vault*'){throw 'Expected batch-selection confirmation not found.'}
    $controls=@([DcmNative]::Snapshot([IntPtr]$dialog.Handle,$script:app.Id))
    if(-not @($controls | Where-Object {$_.Text.Contains('Move 2 selected files into the Vault?') -and $_.Text.Contains($destination)}).Count){throw 'Confirmation count or destination differs from the selected fixture set.'}
    Write-Json (Join-Path $lab ('confirmation-'+$Proceed+'.json')) $controls
    $button=if($Proceed){6}else{7}
    [DcmNative]::ClickAsync([IntPtr]$dialog.Handle,$script:app.Id,$button)
    Start-Sleep -Milliseconds 250
}
function Save-State([string]$Name){
    Assert-Owned
    $snapshot=[DcmNative]::Snapshot($script:hwnd,$script:app.Id)
    Write-Json (Join-Path $lab ($Name+'.controls.json')) $snapshot
    $area=[Windows.Forms.Screen]::FromHandle($script:hwnd).WorkingArea
    [DcmNative]::Restore($script:hwnd,$script:app.Id,$area.Left+8,$area.Top+8)
    Start-Sleep -Milliseconds 350
    $shot=[DcmNative]::Capture($script:hwnd,$script:app.Id,(Join-Path $lab ($Name+'.png')),$false)
    $captures.Add([ordered]@{feature='DCM-24';name=$Name;path=$shot.Path;sha256=(Hash $shot.Path);width=$shot.Width;height=$shot.Height;method=$shot.Method;visual_reviewed=$false})
}
function Invoke-Vision([string]$Name,[string]$Body){
    Assert-Owned
    # FlowUIVision expectations read the most recent explicit inspection.
    $Body=$Body -replace '(?m)^expect ', "inspect main`nexpect "
    $header=@"
target sut type=desktop app=dcm-quicksort
surface main target=sut process="$([IO.Path]::GetFileName($exe))" title~="$version"
control source surface=main id=1001 role=textbox
control name surface=main id=1030 role=textbox
control newname surface=main id=1005 role=textbox
control destination surface=main id=1003 role=textbox
control clear surface=main id=1033 role=button
control sort surface=main id=1045 role=combobox
control extension surface=main id=1013 role=combobox
wait main timeout=5s
inspect main
expect source text="$inbox"
"@
    $path=Join-Path $lab ($Name+'.fvs')
    [IO.File]::WriteAllText($path,($header+"`n"+$Body+"`n"),$utf8)
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$VisionExe;$psi.Arguments='run-script '+(Quote-Argument $path);$psi.WorkingDirectory=$lab
    $psi.UseShellExecute=$false;$psi.CreateNoWindow=$true;$psi.RedirectStandardOutput=$true;$psi.RedirectStandardError=$true
    $psi.EnvironmentVariables['FLOWVERSE_REPO']=$lab
    $job=New-Object Diagnostics.Process;$job.StartInfo=$psi
    try{
        if(-not $job.Start()){throw 'FlowUIVision did not start.'}
        $out=$job.StandardOutput.ReadToEndAsync();$err=$job.StandardError.ReadToEndAsync()
        if(-not $job.WaitForExit(60000)){& "$env:SystemRoot\System32\taskkill.exe" /PID $job.Id /T /F | Out-Null;$null=$job.WaitForExit(10000);throw 'Owned FlowUIVision deadline.'}
        $raw=$out.GetAwaiter().GetResult();$stderr=$err.GetAwaiter().GetResult()
        [IO.File]::WriteAllText((Join-Path $lab ($Name+'.vision.json')),$raw,$utf8)
        [IO.File]::WriteAllText((Join-Path $lab ($Name+'.stderr.log')),$stderr,$utf8)
        $r=$raw | ConvertFrom-Json
        if($job.ExitCode -ne 0 -or $r.Passed -ne $true -or @($r.Steps).Count -eq 0){throw ('FlowUIVision failed: '+$Name)}
        Assert-Owned
        $cases.Add([ordered]@{name=$Name;kind='FlowUIVision';passed=$true;exit_code=$job.ExitCode;steps=@($r.Steps).Count})
    }finally{$job.Dispose()}
}
function Start-App {
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe;$psi.Arguments=Quote-Argument $inbox;$psi.WorkingDirectory=$lab;$psi.UseShellExecute=$false
    foreach($pair in $isolated.GetEnumerator()){$psi.EnvironmentVariables[$pair.Key]=$pair.Value}
    $script:app=New-Object Diagnostics.Process;$script:app.StartInfo=$psi
    if(-not $script:app.Start()){throw 'Candidate did not start.'}
    $script:owned=[ordered]@{pid=$script:app.Id;exe=$exe;start=$script:app.StartTime.ToUniversalTime().ToString('o');sha256=(Hash $exe);startup_argument=$inbox}
    $processes.Add($script:owned)
    $until=[DateTime]::UtcNow.AddSeconds(20)
    do{$script:app.Refresh();if($script:app.HasExited){throw 'Candidate exited during startup.'};$script:hwnd=$script:app.MainWindowHandle;if($script:hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    if($script:hwnd -eq [IntPtr]::Zero){throw 'Main window did not appear.'}
    $script:owned['hwnd']=$script:hwnd.ToInt64();Assert-Owned
    if([DcmNative]::Text($script:hwnd,$script:app.Id) -notlike ('*'+$version+'*')){throw 'Source version not in window title.'}
}
function Stop-App {
    if($null -eq $script:app){return}
    $script:app.Refresh()
    if($script:app.HasExited){throw 'Unexpected app exit before close request.'}
    $childRows=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$script:app.Id))
    $script:owned['children_before_close']=@($childRows | ForEach-Object {$_.ProcessId})
    if($script:app.Path -ne $exe -or $script:app.StartTime.ToUniversalTime().ToString('o') -ne $script:owned.start){throw 'Refusing cleanup of changed process identity.'}
    if($script:hwnd -eq [IntPtr]::Zero){
        & "$env:SystemRoot\System32\taskkill.exe" /PID $script:app.Id /T /F | Out-Null
        $null=$script:app.WaitForExit(10000);$script:owned['forced_stop']=$true
        throw 'Owned startup failed without a closable window.'
    }
    [DcmNative]::RequireOwned($script:hwnd,$script:app.Id)
    $script:owned['close_requested_utc']=[DateTime]::UtcNow.ToString('o')
    [DcmNative]::CloseAsync($script:hwnd,$script:app.Id)
    if(-not $script:app.WaitForExit(15000)){
        & "$env:SystemRoot\System32\taskkill.exe" /PID $script:app.Id /T /F | Out-Null
        $null=$script:app.WaitForExit(10000);$script:owned['forced_stop']=$true
        throw 'App required forced shutdown.'
    }
    $script:owned['exit_code']=$script:app.ExitCode
    if($script:app.ExitCode -ne 0){throw 'App returned a nonzero exit on close.'}
    $parentStart=[DateTime]::Parse($script:owned.start).ToUniversalTime();$parentExit=$script:app.ExitTime.ToUniversalTime()
    $late=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$script:app.Id) | Where-Object {$_.CreationDate.ToUniversalTime() -ge $parentStart -and $_.CreationDate.ToUniversalTime() -le $parentExit})
    foreach($child in @(@($childRows)+@($late))){
        $live=Get-CimInstance Win32_Process -Filter ('ProcessId='+$child.ProcessId)
        if($live -and $live.ExecutablePath -eq $child.ExecutablePath -and $live.CreationDate -eq $child.CreationDate){
            $p=Get-Process -Id $child.ProcessId -ErrorAction SilentlyContinue
            if($p -and -not $p.WaitForExit(10000)){
                $again=Get-CimInstance Win32_Process -Filter ('ProcessId='+$child.ProcessId)
                if($again -and $again.ExecutablePath -eq $child.ExecutablePath -and $again.CreationDate -eq $child.CreationDate){Stop-Process -Id $child.ProcessId -Force}
                throw 'Owned child failed to drain.'
            }
        }
    }
    $script:owned['closed_utc']=[DateTime]::UtcNow.ToString('o')
    $script:owned['children_drained']=$true
    $script:app.Dispose();$script:app=$null;$script:hwnd=[IntPtr]::Zero
}
try{
    $build=Get-Content -LiteralPath $BuildReceipt -Raw -Encoding UTF8 | ConvertFrom-Json
    if($build.passed -ne $true -or $build.mode -notin @('build','package') -or -not $build.source_unchanged){throw 'Receipt is not a successful source-bound build.'}
    $head=Git-Value @('rev-parse','HEAD')
    if($head -ne $build.commit -or (Git-Value @('symbolic-ref','--short','HEAD')) -ne 'release/v1.0.71' -or (Git-Value @('remote','get-url','origin')) -ne 'https://github.com/laxmanxp/dcm-quick-sort.git'){throw 'Build/source identity mismatch.'}
    if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256){throw 'Manifest hash mismatch.'}
    $sourceManifest=Get-Content -LiteralPath $build.source_hash_manifest.path -Raw -Encoding UTF8 | ConvertFrom-Json
    Check-Source
    if((Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'App hash mismatch.'}
    $receipt['commit']=$head;$receipt['build_receipt']=[ordered]@{path=$BuildReceipt;sha256=(Hash $BuildReceipt)};$receipt['app_sha256']=$build.app_artifact.sha256
    $receipt['flowuivision']=[ordered]@{path=$VisionExe;sha256=(Hash $VisionExe)}
    $settings=Get-Content -LiteralPath (Join-Path $repo 'app-src\settings.go') -Raw
    if($settings -notmatch 'const currentAppVersion = "([^"]+)"'){throw 'Source version missing.'}
    $version=$Matches[1];$receipt['version']=$version
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-selection-input.cs')
    $inbox=Join-Path $lab 'Selection Inbox';$vault=Join-Path $lab 'Vault';$destination=Join-Path $vault 'Accepted'
    $local=Join-Path $lab 'local';$config=Join-Path $local 'DCM Quick Sort'
    $isolated=@{LOCALAPPDATA=$local;APPDATA=(Join-Path $lab 'roaming');TEMP=(Join-Path $lab 'temp');TMP=(Join-Path $lab 'temp');DCM_QUICKSORT_BACKUP_DIR=(Join-Path $lab 'backups')}
    foreach($dir in @($inbox,$vault,$destination,$config)+@($isolated.Values | Select-Object -Unique)){New-Item -ItemType Directory -Path $dir -Force | Out-Null}
    Write-Json (Join-Path $config 'settings.json') ([ordered]@{default_source=$inbox;vault_root=$vault;pdf_default_view='fit_page'})
    [IO.File]::WriteAllText((Join-Path $config 'destinations.conf'),"Acceptance=Accepted`n",$utf8)
    $exe=Join-Path $lab ('DCM-QS-Selection-'+$id+'.exe');Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    if((Hash $exe) -ne $build.app_artifact.sha256){throw 'Isolated copy hash mismatch.'}
    $names=@('01-alpha.txt','02-bravo.csv','03-charlie.txt','04-delta.txt','05-echo.txt','06-foxtrot.csv')
    foreach($name in $names){$p=Join-Path $inbox $name;[IO.File]::WriteAllText($p,('DCM-24 fixture: '+$name+' | Only selected files may move.'),$utf8);$expected[$name]=Hash $p}
    Start-App
    Invoke-Vision 'DCM-24-setup' "select sort `"Name`"`ntype destination `"Accepted`"`nexpect destination text=`"Accepted`""
    Wait-For 'six fixture list' {@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016)).Count -eq 6}
    Click-Row '01-alpha.txt';Assert-Selection 'DCM-24-single' @('01-alpha.txt')
    Click-Row '03-charlie.txt' 17;Assert-Selection 'DCM-24-ctrl-nonadjacent' @('01-alpha.txt','03-charlie.txt') -Capture
    Click-Row '01-alpha.txt' 17;Assert-Selection 'DCM-24-ctrl-remove-to-single' @('03-charlie.txt') -Capture
    Click-Row '03-charlie.txt' 17;Assert-Selection 'DCM-24-ctrl-remove-last' @() -Capture
    Click-Row '02-bravo.csv';Click-Row '05-echo.txt' 16
    Assert-Selection 'DCM-24-shift-range' @('02-bravo.csv','03-charlie.txt','04-delta.txt','05-echo.txt') -Capture
    Click-Row '03-charlie.txt' 17;Assert-Selection 'DCM-24-ctrl-remove-middle' @('02-bravo.csv','04-delta.txt','05-echo.txt')
    Click-Row '05-echo.txt';Click-Row '02-bravo.csv' 16
    Assert-Selection 'DCM-24-shift-reverse' @('02-bravo.csv','03-charlie.txt','04-delta.txt','05-echo.txt')
    Click-Row '03-charlie.txt';Assert-Selection 'DCM-24-plain-click-collapses' @('03-charlie.txt')
    Click-Row '01-alpha.txt';Click-Row '03-charlie.txt' 17
    [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1007)
    Assert-Selection 'DCM-24-next-is-single-navigation' @('04-delta.txt')
    Invoke-Vision 'DCM-24-filter' "select extension `".txt`""
    Click-Row '01-alpha.txt';Click-Row '05-echo.txt' 16
    Assert-Selection 'DCM-24-filtered-shift-visible-only' @('01-alpha.txt','03-charlie.txt','04-delta.txt','05-echo.txt') -Capture
    Invoke-Vision 'DCM-24-clear' 'click clear'
    Click-Row '01-alpha.txt';Click-Row '04-delta.txt' 17
    Assert-Selection 'DCM-24-operation-set-before-confirm' @('01-alpha.txt','04-delta.txt') -Capture
    Confirm-Move $false
    Wait-For 'cancelled confirmation' {@([DcmNative]::Dialogs($script:hwnd,$script:app.Id)).Count -eq 0}
    Assert-Selection 'DCM-24-cancel-preserves-selection-and-bytes' @('01-alpha.txt','04-delta.txt')
    Confirm-Move $true
    Wait-For 'exact chosen files moved' {
        @([DcmNative]::Dialogs($script:hwnd,$script:app.Id)).Count -eq 0 -and
        (@(Get-ChildItem -LiteralPath $destination -File | Select-Object -ExpandProperty Name | Sort-Object) -join '|') -ceq '01-alpha.txt|04-delta.txt' -and
        @([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016)).Count -eq 4
    }
    $script:moved=@('01-alpha.txt','04-delta.txt');Assert-Files
    $cases.Add([ordered]@{name='DCM-24-exact-operation-set';passed=$true;moved=$script:moved;untouched=@('02-bravo.csv','03-charlie.txt','05-echo.txt','06-foxtrot.csv');scope='representative same-volume two-file selection; not full Batch Move/Undo acceptance'})
    Save-State 'DCM-24-exact-operation-set';Stop-App
    $receipt['fixture_sha256']=$expected;$receipt['moved']=$script:moved;$receipt['input_method']='Physical SendInput Ctrl/Shift mouse clicks, owned foreground and row hit-tests; no LB_SETSEL';$receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)
    if($null -ne $script:app){try{Save-State 'DCM-24-failure'}catch{$receipt['failure_capture_error']=$_.Exception.Message}}
}finally{
    try{Stop-App}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}
    if($head -and $sourceManifest.Count){try{Check-Source;$receipt['source_unchanged']=$true}catch{$receipt.passed=$false;$receipt['integrity_error']=$_.Exception.Message}}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    Write-Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
