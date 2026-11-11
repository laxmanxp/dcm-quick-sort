[CmdletBinding()]
param(
    [string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,
    [string]$VisionExe=$env:DCM_FLOWUIVISION_EXE
)
# DCM-23: navigation through the actual Next control and native list key messages.
# Reuses the source-bound launch harness; no user documents or host settings.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt){throw 'Workspace and exact DCM_BUILD_RECEIPT are required.'}
if(-not $VisionExe){$VisionExe='C:\FlowverseReports\dcm-flowuivision-20261103\FlowUIVision.exe'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$workspace=[IO.Path]::GetFullPath($env:FLOWDEV_WORKSPACE_ROOT)
$id=[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8)
$lab=Join-Path $workspace ('dcm-quicksort-ui-navigation-'+$id)
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Evidence must be outside checkout.'}
New-Item -ItemType Directory -Path $lab | Out-Null
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$processes=New-Object 'Collections.Generic.List[object]'
$visionRuns=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;app='dcm-quicksort';feature='DCM-23';suite='navigation';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;cases=$cases;captures=$captures;processes=$processes;vision_runs=$visionRuns;keyboard_input='owned queued WM_KEYDOWN/WM_KEYUP; not physical keyboard acceptance'}
$script:app=$null;$script:hwnd=[IntPtr]::Zero;$script:owned=$null
$sourceManifest=@();$expected=@{};$head=''
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
    $files=@(Get-ChildItem -LiteralPath $inbox -Force)
    if($files.Count -ne $expected.Count){throw 'Inbox fixture set changed.'}
    foreach($f in $files){if($f.PSIsContainer -or -not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Fixture name/bytes changed.'}}
    if(@(Get-ChildItem -LiteralPath $destination -Force).Count){throw 'No-selection action created a destination entry.'}
    $journal=Join-Path $config 'recovery\move-journal.jsonl'
    if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'No-selection action appended an operation journal.'}
}
function Save-State([string]$Name){
    Assert-Owned
    $snapshot=[DcmNative]::Snapshot($script:hwnd,$script:app.Id)
    Write-Json (Join-Path $lab ($Name+'.controls.json')) $snapshot
    $area=[Windows.Forms.Screen]::FromHandle($script:hwnd).WorkingArea
    [DcmNative]::Restore($script:hwnd,$script:app.Id,$area.Left+8,$area.Top+8)
    Start-Sleep -Milliseconds 350
    $shot=[DcmNative]::Capture($script:hwnd,$script:app.Id,(Join-Path $lab ($Name+'.png')),$false)
    $captures.Add([ordered]@{feature='DCM-23';name=$Name;path=$shot.Path;sha256=(Hash $shot.Path);width=$shot.Width;height=$shot.Height;method=$shot.Method;visual_reviewed=$false})
}
function Assert-View([string]$Name,[string[]]$Names,[string]$Selected,[switch]$Capture){
    Wait-For $Name { (@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016)) -join '|') -ceq ($Names -join '|') -and (Control-Text 1005) -ceq [IO.Path]::GetFileNameWithoutExtension($Selected) }
    $indices=@([DcmNative]::SelectedIndexes($script:hwnd,$script:app.Id,1016))
    $pos=[Array]::IndexOf($Names,$Selected)
    Write-Json (Join-Path $lab ($Name+'.selection.json')) ([ordered]@{expected=$Selected;expected_index=$pos;actual_indexes=$indices;draft=(Control-Text 1005)})
    if($pos -lt 0 -or $indices.Count -ne 1 -or $indices[0] -ne $pos){throw ('DCM23_SELECTION_MISMATCH: '+$Name+' expected '+$Selected+' at '+$pos+'; actual indexes '+($indices -join ','))}
    $snapshot=@([DcmNative]::Snapshot($script:hwnd,$script:app.Id))
    if(-not @($snapshot | Where-Object {$_.Visible -and $_.Text.StartsWith('PREVIEW') -and $_.Text.EndsWith($Selected)}).Count){throw 'Preview header does not identify the selected file.'}
    if(-not @($snapshot | Where-Object {$_.Visible -and $_.Class -eq 'Edit' -and $_.Text -ceq $fixtureText[$Selected]}).Count){throw 'Visible text preview does not match the selected file.'}
    $counter=('{0} of {1}' -f ($pos+1),$Names.Count)
    if(-not @($snapshot | Where-Object {$_.Visible -and $_.Text.StartsWith('FILES') -and $_.Text.EndsWith($counter)}).Count){throw 'File counter does not match selected row.'}
    Assert-Files
    $cases.Add([ordered]@{name=$Name;expected_order=$Names;selected=$Selected;selected_index=$pos;draft=(Control-Text 1005);preview_text=$fixtureText[$Selected];passed=$true})
    if($Capture){Save-State $Name}
}
function Assert-NoCurrent([string]$Name,[string[]]$Names,[string]$Heading,[switch]$Capture){
    Wait-For $Name { (@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016)) -join '|') -ceq ($Names -join '|') -and (Control-Text 1005) -eq '' }
    if(@([DcmNative]::SelectedIndexes($script:hwnd,$script:app.Id,1016)).Count){throw 'No-current state retained a selected row.'}
    $snapshot=@([DcmNative]::Snapshot($script:hwnd,$script:app.Id))
    if(-not @($snapshot | Where-Object {$_.Visible -and $_.Text.StartsWith('PREVIEW') -and $_.Text.EndsWith($Heading)}).Count){throw ('No-current heading mismatch: '+$Heading)}
    foreach($text in $fixtureText.Values){if(@($snapshot | Where-Object {$_.Visible -and $_.Class -eq 'Edit' -and $_.Text -ceq $text}).Count){throw 'Stale visible text preview in no-current state.'}}
    Assert-Files
    $cases.Add([ordered]@{name=$Name;listed_files=$Names;selected_count=0;draft='';heading=$Heading;passed=$true})
    if($Capture){Save-State $Name}
}
function Key([int]$VirtualKey){
    Assert-Owned
    [DcmNavigationKeys]::FocusList($script:hwnd,$script:app.Id)
    Wait-For 'owned file-list keyboard focus' { [DcmNavigationKeys]::ListFocused($script:hwnd,$script:app.Id) }
    [DcmNavigationKeys]::Press($script:hwnd,$script:app.Id,$VirtualKey)
    Start-Sleep -Milliseconds 200
}
function Next {
    Assert-Owned
    [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1007)
    Start-Sleep -Milliseconds 200
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
control next surface=main id=1007 role=button
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
        $visionRuns.Add([ordered]@{name=$Name;passed=$true;exit_code=$job.ExitCode;steps=@($r.Steps).Count})
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
    Add-Type -Path @((Join-Path $repo 'tests\native\dcm-native.cs'),(Join-Path $repo 'tests\native\dcm-navigation-keys.cs')) -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    $inbox=Join-Path $lab 'Navigation Inbox';$vault=Join-Path $lab 'Vault';$destination=Join-Path $vault 'Accepted'
    $local=Join-Path $lab 'local';$config=Join-Path $local 'DCM Quick Sort'
    $isolated=@{LOCALAPPDATA=$local;APPDATA=(Join-Path $lab 'roaming');TEMP=(Join-Path $lab 'temp');TMP=(Join-Path $lab 'temp');DCM_QUICKSORT_BACKUP_DIR=(Join-Path $lab 'backups')}
    foreach($dir in @($inbox,$vault,$destination,$config)+@($isolated.Values | Select-Object -Unique)){New-Item -ItemType Directory -Path $dir -Force | Out-Null}
    Write-Json (Join-Path $config 'settings.json') ([ordered]@{default_source=$inbox;vault_root=$vault;pdf_default_view='fit_page'})
    [IO.File]::WriteAllText((Join-Path $config 'destinations.conf'),"Acceptance=Accepted`n",$utf8)
    $exe=Join-Path $lab ('DCM-QS-Navigation-'+$id+'.exe');Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    if((Hash $exe) -ne $build.app_artifact.sha256){throw 'Isolated copy hash mismatch.'}
    $fixtureText=@{}
    Start-App
    Assert-NoCurrent 'DCM-23-empty-start' @() 'No files remaining'
    Next;Next
    Assert-NoCurrent 'DCM-23-empty-next' @() 'No files remaining' -Capture
    Stop-App
    foreach($name in @('01-alpha.txt','02-hidden.log','03-charlie.txt','04-delta.txt')){
        $p=Join-Path $inbox $name
        $fixtureText[$name]='DCM-23 navigation fixture: '+$name+' | The selected row and preview must agree.'
        [IO.File]::WriteAllText($p,$fixtureText[$name],$utf8);$expected[$name]=Hash $p
    }
    $all=[string[]]@('01-alpha.txt','02-hidden.log','03-charlie.txt','04-delta.txt')
    $filtered=[string[]]@('01-alpha.txt','03-charlie.txt','04-delta.txt')
    Start-App
    Invoke-Vision 'setup-sort' 'select sort "Name"'
    $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
    Assert-View 'DCM-23-first' $all '01-alpha.txt' -Capture
    Next
    Assert-View 'DCM-23-next-adjacent' $all '02-hidden.log' -Capture
    Key 0x26
    Assert-View 'DCM-23-up-previous' $all '01-alpha.txt'
    Key 0x26
    Assert-View 'DCM-23-up-at-first' $all '01-alpha.txt'
    Key 0x28
    Assert-View 'DCM-23-down-adjacent' $all '02-hidden.log'
    Key 0x23
    Assert-View 'DCM-23-end-key' $all '04-delta.txt'
    Key 0x28
    Assert-View 'DCM-23-down-at-last' $all '04-delta.txt' -Capture
    Next
    Assert-NoCurrent 'DCM-23-next-beyond-last' $all 'No file selected' -Capture
    Next
    Assert-NoCurrent 'DCM-23-repeated-next-at-end' $all 'No file selected'
    Key 0x24
    Assert-View 'DCM-23-home-recovers' $all '01-alpha.txt'
    Invoke-Vision 'filter-text' 'select extension ".txt"'
    Assert-View 'DCM-23-filter-first' $filtered '01-alpha.txt'
    Next
    Assert-View 'DCM-23-filter-next-skips-hidden' $filtered '03-charlie.txt' -Capture
    Key 0x26
    Assert-View 'DCM-23-filter-up-skips-hidden' $filtered '01-alpha.txt'
    Key 0x28
    Assert-View 'DCM-23-filter-down-skips-hidden' $filtered '03-charlie.txt'
    Invoke-Vision 'one-file-filter' 'type name "04-delta"'
    Assert-View 'DCM-23-single-file' @('04-delta.txt') '04-delta.txt'
    Key 0x26;Key 0x28
    Assert-View 'DCM-23-single-arrow-boundaries' @('04-delta.txt') '04-delta.txt'
    Next
    Assert-NoCurrent 'DCM-23-single-next-end' @('04-delta.txt') 'No file selected'
    Key 0x24
    Assert-View 'DCM-23-single-home-recovers' @('04-delta.txt') '04-delta.txt'
    Invoke-Vision 'zero-match-filter' 'type name "nothing-matches-dcm23"'
    Next;Next
    Assert-NoCurrent 'DCM-23-filtered-empty-next' @() 'No files match the current filters' -Capture
    Invoke-Vision 'clear-filter' 'click clear'
    Wait-For 'clear restores list' { @([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016)).Count -eq 4 }
    Key 0x24
    Assert-View 'DCM-23-clear-and-home' $all '01-alpha.txt' -Capture
    Stop-App
    $receipt['fixture_sha256']=$expected;$receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)
    if($null -ne $script:app){try{Save-State 'DCM-23-failure'}catch{$receipt['failure_capture_error']=$_.Exception.Message}}
}finally{
    try{Stop-App}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}
    if($head -and $sourceManifest.Count){try{Check-Source;$receipt['source_unchanged']=$true}catch{$receipt.passed=$false;$receipt['integrity_error']=$_.Exception.Message}}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    Write-Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
