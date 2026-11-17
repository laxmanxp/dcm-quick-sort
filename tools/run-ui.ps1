[CmdletBinding()]
param(
    [ValidateSet('basics','filters','rename','rename-edge','undo-rename','undo-move-safety','destination','move','cross-volume','conflicts','batch','batch-partial','batch-undo-safety','history','pdf','previews','preview-release-operations','office-preview-release-operations')][string]$Suite='move',
    [string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,
    [string]$VisionExe=$env:DCM_FLOWUIVISION_EXE,
    [string]$PreviousFixtureInbox='C:\FlowverseReports\dcm-flowuivision-20261103\inbox'
)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Use native Windows x64 FlowDev.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT){throw 'FLOWDEV_WORKSPACE_ROOT is required.'}
if(-not $BuildReceipt){throw 'Set DCM_BUILD_RECEIPT to the exact successful build/package receipt before invoking this FlowDev selector.'}
if(-not $VisionExe){$VisionExe='C:\FlowverseReports\dcm-flowuivision-20261103\FlowUIVision.exe'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$workspace=[IO.Path]::GetFullPath($env:FLOWDEV_WORKSPACE_ROOT)
$id=[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8)
$lab=Join-Path $workspace ('dcm-quicksort-ui-'+$Suite+'-'+$id)
New-Item -ItemType Directory -Path $lab | Out-Null
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$processes=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;app='dcm-quicksort';suite=$Suite;report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;cases=$cases;captures=$captures;processes=$processes}
$script:app=$null
$script:hwnd=[IntPtr]::Zero
$script:owned=$null
$fixture=$null
$initial=@()
Write-Output ('REPORT_ROOT='+$lab)
function Write-Json([string]$Path,$Value){[IO.File]::WriteAllText($Path,($Value | ConvertTo-Json -Depth 30),$utf8)}
function Quote-Argument([string]$Value){return '"'+(($Value -replace '(\\*)"','$1$1\"') -replace '(\\+)$','$1$1')+'"'}
function Hash([string]$Path){return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()}
function Assert-OwnedProcess {
    if($null -eq $script:app){throw 'Owned app has not started.'}
    $script:app.Refresh()
    if($script:app.HasExited){throw 'Owned app exited before its acceptance step completed.'}
    if($script:app.Path -ne $script:owned.exe -or $script:app.StartTime.ToUniversalTime().ToString('o') -ne $script:owned.start){throw 'App process identity changed.'}
}
function Assert-Owned {
    Assert-OwnedProcess
    [DcmNative]::RequireOwned($script:hwnd,$script:app.Id)
}
function Wait-For([string]$Name,[scriptblock]$Predicate,[int]$Seconds=15,[switch]$AllowDialog){
    $until=[DateTime]::UtcNow.AddSeconds($Seconds)
    do {
        Assert-Owned
        if(-not $AllowDialog){
            $dialogs=@([DcmNative]::Dialogs($script:hwnd,$script:app.Id))
            if($dialogs.Count){Write-Json (Join-Path $lab 'unexpected-dialogs.json') $dialogs;throw "Unexpected owned dialog while waiting for $Name"}
        }
        if(& $Predicate){return}
        Start-Sleep -Milliseconds 100
    } while([DateTime]::UtcNow -lt $until)
    throw "Deadline waiting for $Name"
}
function Save-Shot([string]$Name,[switch]$Preview){
    Assert-Owned
    $area=[Windows.Forms.Screen]::FromHandle($script:hwnd).WorkingArea
    [DcmNative]::Restore($script:hwnd,$script:app.Id,$area.Left+8,$area.Top+8)
    Wait-For ('restored capture '+$Name) { -not [DcmNative]::IsIconic($script:hwnd) -and [DcmNative]::IsWindowVisible($script:hwnd) } 5
    Start-Sleep -Milliseconds 250
    $shot=[DcmNative]::Capture($script:hwnd,$script:app.Id,(Join-Path $lab ($Name+'.png')),[bool]$Preview)
    $entry=[ordered]@{name=$Name;path=$shot.Path;sha256=(Hash $shot.Path);left=$shot.Left;top=$shot.Top;width=$shot.Width;height=$shot.Height;method=$shot.Method;sampled_pixels=$shot.SampledPixels;teal_samples=$shot.TealSamples;orange_samples=$shot.OrangeSamples;visual_reviewed=$false}
    $captures.Add($entry)
    Write-Json (Join-Path $lab ($Name+'.capture.json')) $entry
    return $entry
}
function Save-State([string]$Name){
    Assert-Owned
    Write-Json (Join-Path $lab ($Name+'.controls.json')) ([DcmNative]::Snapshot($script:hwnd,$script:app.Id))
    Write-Json (Join-Path $lab ($Name+'.files.json')) ([ordered]@{inbox=@(Get-ChildItem -LiteralPath $fixture.inbox -File | ForEach-Object {[ordered]@{name=$_.Name;sha256=(Hash $_.FullName)}});destination=@(Get-ChildItem -LiteralPath $fixture.destination -File | ForEach-Object {[ordered]@{name=$_.Name;sha256=(Hash $_.FullName)}})})
}
function Assert-PdfPage([string]$Name,[int]$Page){
    Assert-Owned
    if($Page -lt 1 -or $Page -gt 2){throw 'Only the known two-page fixture is qualified'}
    $controls=@([DcmNative]::Snapshot($script:hwnd,$script:app.Id))
    if(@($controls|Where-Object {$_.Visible -and $_.Text -ceq ('Page '+$Page+' of 2')}).Count -ne 1){throw 'Wrong visible PDF page label'}
    if(-not (Status-Contains ('DCM PDF Preview')) -or -not (Status-Contains ('Page '+$Page+'/2'))){throw 'Wrong provider or page status'}
    $previous=[DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1060))
    $next=[DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1061))
    if($previous -ne ($Page -eq 2) -or $next -ne ($Page -eq 1)){throw 'First/last PDF boundary button state is wrong'}
    $null=Save-Shot ($Name+'-window')
    $pixels=Save-Shot ($Name+'-document') -Preview
    if($Page -eq 1 -and ($pixels.teal_samples -lt 100 -or $pixels.orange_samples -gt 100)){throw 'Wrong or blank first-page pixels'}
    if($Page -eq 2 -and ($pixels.orange_samples -lt 100 -or $pixels.teal_samples -gt 100)){throw 'Wrong or blank second-page pixels'}
    $cases.Add([ordered]@{name=$Name;kind='default PDF page pixels and boundary readback';passed=$true;page=$Page;previous_enabled=$previous;next_enabled=$next;teal_samples=$pixels.teal_samples;orange_samples=$pixels.orange_samples})
}
function Pan-PdfToTop([string]$Name){
    # Exercise the existing drag handler only in this owned default-provider test.
    # Native messages are not physical mouse/touchpad acceptance.
    Assert-Owned
    $preview=[DcmNative]::ChildByClass($script:hwnd,$script:app.Id,'QuickSortPreview')
    $v=[DcmImageProbe]::Geometry($preview,$script:app.Id)
    $dx=[Math]::Min(80,[int]($v.Width/4));$dy=[Math]::Min(120,[int]($v.Height/4))
    for($i=0;$i -lt 6;$i++){[DcmImageProbe]::Drag($preview,$script:app.Id,$dx,$dy);Start-Sleep -Milliseconds 80}
    Assert-PdfPage $Name 1
}
function Start-OwnedApp {
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe
    $startupFile=Get-ChildItem -LiteralPath $fixture.inbox -Filter '*.txt' -File | Sort-Object Name | Select-Object -First 1
    $startupArgument=$fixture.inbox
    if($startupFile){$startupArgument=$startupFile.FullName}
    $psi.Arguments=Quote-Argument $startupArgument
    $psi.WorkingDirectory=$lab
    $psi.UseShellExecute=$false
    $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local
    $psi.EnvironmentVariables['APPDATA']=$fixture.roaming
    $psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'configuration-backups'
    $psi.EnvironmentVariables['TEMP']=Join-Path $lab 'temp'
    $psi.EnvironmentVariables['TMP']=Join-Path $lab 'temp'
    if($Suite -eq 'pdf'){
        $psi.EnvironmentVariables.Remove('DCM_PDF_PROVIDER')
        $psi.EnvironmentVariables.Remove('DCM_SUMATRA_EXE')
        $receipt['pdf_provider']='default Windows.Data.Pdf; opt-in environment removed'
    }
    $script:app=New-Object Diagnostics.Process
    $script:app.StartInfo=$psi
    if(-not $script:app.Start()){throw 'Could not launch exact candidate copy.'}
    $script:owned=[ordered]@{pid=$script:app.Id;exe=$exe;start=$script:app.StartTime.ToUniversalTime().ToString('o');hwnd=0;sha256=(Hash $exe);startup_argument=$startupArgument;backup_root=(Join-Path $lab 'configuration-backups')}
    $processes.Add($script:owned)
    Write-Json (Join-Path $lab ('owned-'+$processes.Count+'.json')) $script:owned
    $until=[DateTime]::UtcNow.AddSeconds(20)
    do {$script:app.Refresh();if($script:app.HasExited){throw 'Candidate exited during launch.'};$script:hwnd=$script:app.MainWindowHandle;if($script:hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100} while([DateTime]::UtcNow -lt $until)
    if($script:hwnd -eq [IntPtr]::Zero){throw 'Candidate main window deadline.'}
    $script:owned.hwnd=$script:hwnd.ToInt64()
    Write-Json (Join-Path $lab ('owned-'+$processes.Count+'.json')) $script:owned
    Assert-Owned
    if([DcmNative]::Text($script:hwnd,$script:app.Id) -notlike ('*'+$version+'*')){throw 'Candidate title does not identify the source version.'}
    $null=Save-Shot ('launch-'+$processes.Count)
}
function Stop-OwnedApp {
    if($null -eq $script:app){return}
    $script:app.Refresh()
    if($script:app.HasExited -and -not $script:owned.Contains('close_requested_utc')){
        $script:owned['unexpected_exit']=$true;$script:owned['exit_code']=$script:app.ExitCode
        $script:app.Dispose();$script:app=$null;$script:hwnd=[IntPtr]::Zero
        throw 'Owned app exited before a close request; lifecycle acceptance failed.'
    }
    if(-not $script:app.HasExited){
        Assert-OwnedProcess
        # Retain identity of direct renderer children before parent shutdown.
        $children=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$script:app.Id) | ForEach-Object {[ordered]@{pid=[int]$_.ProcessId;path=$_.ExecutablePath;created=$_.CreationDate.ToUniversalTime().ToString('o')}})
        $script:owned['children_before_close']=$children
        $script:owned['close_attempted_utc']=[DateTime]::UtcNow.ToString('o')
        $closeError=''
        try{
            [DcmNative]::CloseAsync($script:hwnd,$script:app.Id)
            $script:owned['close_requested_utc']=[DateTime]::UtcNow.ToString('o')
        }catch{$closeError=$_.Exception.Message;$script:owned['close_window_error']=$closeError}
        if(-not $script:app.WaitForExit(15000)){
            Assert-OwnedProcess
            & "$env:SystemRoot\System32\taskkill.exe" /PID $script:app.Id /T /F | Out-Null
            $null=$script:app.WaitForExit(10000)
            $script:owned['forced_stop']=$true
            throw 'Owned app required forced shutdown; lifecycle acceptance failed.'
        }
        $parentStart=[DateTime]::Parse($script:owned.start).ToUniversalTime()
        $parentExit=$script:app.ExitTime.ToUniversalTime()
        $late=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$script:app.Id) | Where-Object {$_.CreationDate.ToUniversalTime() -ge $parentStart -and $_.CreationDate.ToUniversalTime() -le $parentExit} | ForEach-Object {[ordered]@{pid=[int]$_.ProcessId;path=$_.ExecutablePath;created=$_.CreationDate.ToUniversalTime().ToString('o')}})
        $script:owned['children_after_close']=$late
        $children=@(@($children)+@($late) | Sort-Object -Property {$_['pid']} -Unique)
        foreach($child in $children){
            $live=Get-CimInstance Win32_Process -Filter ('ProcessId='+$child.pid)
            if($live -and $live.ExecutablePath -eq $child.path -and $live.CreationDate.ToUniversalTime().ToString('o') -eq $child.created){
                $drain=Get-Process -Id $child.pid -ErrorAction SilentlyContinue
                if($drain -and -not $drain.WaitForExit(10000)){
                    $again=Get-CimInstance Win32_Process -Filter ('ProcessId='+$child.pid)
                    if($again -and $again.ExecutablePath -eq $child.path -and $again.CreationDate.ToUniversalTime().ToString('o') -eq $child.created){Stop-Process -Id $child.pid -Force;throw 'Owned renderer did not drain after app close.'}
                }
            }
        }
        if($closeError){throw ('Owned app lost its main window before close: '+$closeError)}
    }
    $script:owned['closed_utc']=[DateTime]::UtcNow.ToString('o')
    $script:owned['exit_code']=$script:app.ExitCode
    $cleanExit=($script:app.ExitCode -eq 0)
    $script:app.Dispose();$script:app=$null;$script:hwnd=[IntPtr]::Zero
    if(-not $cleanExit){throw 'Owned app returned a nonzero exit code on close.'}
}
function Invoke-Fvs([string]$Name,[string]$Body){
    Assert-Owned
    $path=Join-Path $lab ($Name+'.fvs')
    $header=@"
target sut type=desktop app=dcm-quicksort
surface main target=sut process="$([IO.Path]::GetFileName($exe))" title~="$version"
control path surface=main id=1001 role=textbox
control name surface=main id=1030 role=textbox
control clear surface=main id=1033 role=button
control category surface=main id=1012 role=combobox
control extension surface=main id=1013 role=combobox
control age surface=main id=1031 role=combobox
control size surface=main id=1032 role=combobox
control sort surface=main id=1045 role=combobox
control processedsearch surface=main id=1046 role=textbox
control next surface=main id=1007 role=button
control newname surface=main id=1005 role=textbox
control destination surface=main id=1003 role=textbox
control move surface=main id=1041 role=button
control undo surface=main id=1008 role=button
control undobatch surface=main id=1048 role=button
control rename surface=main id=1006 role=button
control tool surface=main id=1027 role=combobox
control apply surface=main id=1028 role=button
control sheets surface=main id=1024 role=combobox
control slides surface=main id=1025 role=combobox
control fit surface=main id=1023 role=button
control width surface=main id=1047 role=button
control zoomreset surface=main id=1021 role=button
control zoomin surface=main id=1022 role=button
control zoomout surface=main id=1020 role=button
control nextpage surface=main id=1061 role=button
control prevpage surface=main id=1060 role=button
control pdfstatus surface=main role=text text~="DCM PDF Preview"
control count surface=main role=text text~="FILES"
wait main timeout=5s
inspect main
expect path text~="dcm-quicksort-ui-"
"@
    [IO.File]::WriteAllText($path,($header+"`n"+$Body+"`n"),$utf8)
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$VisionExe;$psi.Arguments='run-script '+(Quote-Argument $path);$psi.WorkingDirectory=$lab
    $psi.UseShellExecute=$false;$psi.CreateNoWindow=$true;$psi.RedirectStandardOutput=$true;$psi.RedirectStandardError=$true
    $psi.EnvironmentVariables['FLOWVERSE_REPO']=$lab
    $job=New-Object Diagnostics.Process;$job.StartInfo=$psi
    $entry=[ordered]@{name=$Name;kind='FlowUIVision';script=$path;passed=$false;started_utc=[DateTime]::UtcNow.ToString('o')}
    $cases.Add($entry)
    try{
        if(-not $job.Start()){throw 'Could not launch FlowUIVision.'}
        $entry['pid']=$job.Id;$outTask=$job.StandardOutput.ReadToEndAsync();$errTask=$job.StandardError.ReadToEndAsync()
        if(-not $job.WaitForExit(60000)){& "$env:SystemRoot\System32\taskkill.exe" /PID $job.Id /T /F | Out-Null;$null=$job.WaitForExit(10000);throw "FlowUIVision deadline at $Name; no further UI input is permitted."}
        $raw=$outTask.GetAwaiter().GetResult();$stderr=$errTask.GetAwaiter().GetResult()
        [IO.File]::WriteAllText((Join-Path $lab ($Name+'.vision.json')),$raw,$utf8)
        [IO.File]::WriteAllText((Join-Path $lab ($Name+'.stderr.log')),$stderr,$utf8)
        $result=$raw | ConvertFrom-Json
        $entry['exit_code']=$job.ExitCode;$entry['steps']=@($result.Steps).Count
        if($job.ExitCode -ne 0 -or $result.Passed -ne $true -or @($result.Steps).Count -eq 0){throw "FlowUIVision assertions failed in $Name"}
        Assert-Owned;$entry.passed=$true
    } catch {$entry['error']=$_.Exception.Message;throw}
    finally {$entry['finished_utc']=[DateTime]::UtcNow.ToString('o');$job.Dispose();Write-Json (Join-Path $lab 'cases.json') $cases}
}
function Assert-Fixtures([string[]]$Moved=@()){
    foreach($item in $initial){
        $original=Join-Path $fixture.inbox $item.name;$target=Join-Path $fixture.destination $item.name
        if($Moved -contains $item.name){
            if((Test-Path -LiteralPath $original) -or -not(Test-Path -LiteralPath $target -PathType Leaf)){throw "Expected moved fixture state: $($item.name)"}
            if((Hash $target) -ne $item.sha256){throw "Moved bytes differ: $($item.name)"}
        } else {
            if(-not(Test-Path -LiteralPath $original -PathType Leaf) -or (Test-Path -LiteralPath $target)){throw "Unexpected unselected/restored fixture state: $($item.name)"}
            if((Hash $original) -ne $item.sha256){throw "Original/restored bytes differ: $($item.name)"}
        }
    }
    $actualNames=@(Get-ChildItem -LiteralPath $fixture.destination -File | ForEach-Object {$_.Name} | Sort-Object)
    if(($actualNames -join '|') -cne (($Moved | Sort-Object) -join '|')){throw 'Destination contains an unexpected file.'}
    $actualInbox=@(Get-ChildItem -LiteralPath $fixture.inbox -File | ForEach-Object {$_.Name} | Sort-Object)
    $expectedInbox=@($initial | Where-Object {$Moved -notcontains $_.name} | ForEach-Object {$_.name} | Sort-Object)
    if(($actualInbox -join '|') -cne ($expectedInbox -join '|')){throw 'Inbox contains an unexpected file or lost an unselected fixture.'}
}
function Assert-Journal([string]$Name,[string[]]$Names,[string]$State,[switch]$Batch){
    $path=Join-Path $fixture.config 'recovery\move-journal.jsonl'
    $events=@(Get-Content -LiteralPath $path -Encoding UTF8 | Where-Object {$_.Trim()} | ForEach-Object {$_ | ConvertFrom-Json})
    $terminal=@();$batchID=''
    foreach($file in $Names){
        $entries=@($events | Where-Object {$_.record.from -eq (Join-Path $fixture.inbox $file) -and $_.record.to -eq (Join-Path $fixture.destination $file)})
        if(-not $entries.Count -or $entries[-1].state -ne $State){throw "Journal does not confirm $State for $file"}
        $terminal+=$entries[-1]
        if($Batch){foreach($e in $entries){if(-not $e.record.PSObject.Properties['batch_id'] -or -not $e.record.batch_id){throw 'Batch membership absent from durable journal event.'};if(-not $batchID){$batchID=$e.record.batch_id};if($e.record.batch_id -ne $batchID){throw 'Batch journal identity changed.'}}}
    }
    Copy-Item -LiteralPath $path -Destination (Join-Path $lab ($Name+'.journal.jsonl'))
    $cases.Add([ordered]@{name=$Name;kind='independent filesystem and journal oracle';passed=$true;state=$State;files=$Names;batch_id=$batchID;journal_sha256=(Hash $path);terminal_entries=$terminal})
}
function Assert-RenameJournal([string]$Name,[string]$From,[string]$To,[string]$State,[string]$ExpectedID=''){
    $path=Join-Path $fixture.config 'recovery\move-journal.jsonl'
    $events=@(Get-Content -LiteralPath $path -Encoding UTF8 | Where-Object {$_.Trim()} | ForEach-Object {$_ | ConvertFrom-Json})
    $matched=@($events | Where-Object {$_.record.from -eq $From -and $_.record.to -eq $To})
    if(-not $matched.Count -or $matched[0].state -ne 'pending' -or $matched[-1].state -ne $State){throw "Rename durable journal does not prove pending-to-$State"}
    $ids=@($matched | ForEach-Object {$_.id} | Sort-Object -Unique)
    if($ids.Count -ne 1 -or ($ExpectedID -and $ids[0] -ne $ExpectedID)){throw 'Rename journal operation identity changed.'}
    Copy-Item -LiteralPath $path -Destination (Join-Path $lab ($Name+'.journal.jsonl'))
    $cases.Add([ordered]@{name=$Name;kind='independent rename journal oracle';passed=$true;from=$From;to=$To;state=$State;id=$ids[0];journal_sha256=(Hash $path);terminal_entry=$matched[-1]})
    return $ids[0]
}
function Confirm-Dialog([string]$Title,[string[]]$TextFragments,[int]$Button=6,[switch]$ExpectFollowupDialog){
    Wait-For 'expected confirmation' { @([DcmNative]::Dialogs($script:hwnd,$script:app.Id)).Count -gt 0 } 10 -AllowDialog
    $dialogs=@([DcmNative]::Dialogs($script:hwnd,$script:app.Id))
    if($dialogs.Count -ne 1 -or -not $dialogs[0].Text.StartsWith($Title)){Write-Json (Join-Path $lab 'unexpected-confirmation.json') $dialogs;throw 'Unexpected confirmation dialog; no response was sent.'}
    $dialog=[IntPtr]$dialogs[0].Handle
    $controls=[DcmNative]::Snapshot($dialog,$script:app.Id)
    $text=($controls | ForEach-Object {$_.Text}) -join "`n"
    foreach($fragment in $TextFragments){if(-not $text.Contains($fragment)){throw "Confirmation does not identify expected fixture scope: $fragment"}}
    Write-Json (Join-Path $lab ('confirmation-'+$cases.Count+'.json')) ([ordered]@{title=$dialogs[0].Text;controls=$controls;button=$Button})
    [DcmNative]::ClickAsync($dialog,$script:app.Id,$Button)
    # After BM_CLICK the native message box is being destroyed. Give it a
    # bounded settle interval before querying its owned descendants; a
    # synchronous window-text query against the dying HWND can time out.
    Start-Sleep -Milliseconds 250
    if(-not $ExpectFollowupDialog){
    Wait-For 'confirmation closed' { @([DcmNative]::Dialogs($script:hwnd,$script:app.Id)).Count -eq 0 } 10 -AllowDialog
    }
}
function Status-Contains([string]$Text){return @([DcmNative]::Snapshot($script:hwnd,$script:app.Id) | Where-Object {$_.Visible -and $_.Text.Contains($Text)}).Count -gt 0}
function OfficeStatusContains([string]$Text){
    # Host-owned top-level status controls only. The installed Office preview
    # subtree may legitimately contain windows from another provider PID.
    $captions=@([DcmDocxProbe]::HostStatics($script:hwnd,$script:app.Id) | ForEach-Object {[DcmNative]::Text([IntPtr]$_,$script:app.Id)})
    return @($captions | Where-Object {$_.Contains($Text)}).Count -gt 0
}
function Assert-List([string]$Name,[string[]]$Expected,[switch]$Ordered){
    $actual=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016))
    $left=$actual;$right=$Expected
    if(-not $Ordered){$left=@($left | Sort-Object);$right=@($right | Sort-Object)}
    if(($left -join '|') -cne ($right -join '|')){throw "$Name list readback mismatch: got $($actual -join ', ')"}
    $cases.Add([ordered]@{name=$Name;kind='native list readback';passed=$true;items=$actual;ordered=[bool]$Ordered})
}
function Wait-Moved([string[]]$Names){
    Wait-For 'physical fixture relocation and idle controls' {
        foreach($name in $Names){if((Test-Path -LiteralPath (Join-Path $fixture.inbox $name)) -or -not(Test-Path -LiteralPath (Join-Path $fixture.destination $name))){return $false}}
        return [DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1008))
    } 30
    Assert-Fixtures $Names
}
function Wait-Restored([string[]]$Names){
    Wait-For 'physical fixture restoration' {
        foreach($name in $Names){if(-not(Test-Path -LiteralPath (Join-Path $fixture.inbox $name)) -or (Test-Path -LiteralPath (Join-Path $fixture.destination $name))){return $false}}
        return $true
    } 30
    Assert-Fixtures
}

try {
    $build=Get-Content -LiteralPath $BuildReceipt -Raw -Encoding UTF8 | ConvertFrom-Json
    if($build.passed -ne $true -or $build.mode -notin @('build','package')){throw 'Build receipt is not a successful app-producing build/package run.'}
    $head=(& git.exe -C $repo rev-parse HEAD | Out-String).Trim();if($LASTEXITCODE -ne 0){throw 'Cannot read candidate HEAD.'}
    $dirty=(& git.exe -C $repo status --porcelain=v1 --untracked-files=all | Out-String).Trim();if($LASTEXITCODE -ne 0 -or $dirty){throw 'UI qualification requires a clean committed checkout.'}
    if($head -ne $build.commit){throw 'Build receipt does not match current committed tooling/source HEAD.'}
    if(-not $build.source_unchanged){throw 'Build receipt did not verify source integrity.'}
    if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256){throw 'Build source manifest changed.'}
    $sourceManifest=Get-Content -LiteralPath $build.source_hash_manifest.path -Raw -Encoding UTF8 | ConvertFrom-Json
    foreach($sourceFile in $sourceManifest){
        $current=[IO.Path]::GetFullPath((Join-Path $repo $sourceFile.path))
        if(-not $current.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Build source manifest escapes the candidate.'}
        if((Hash $current) -ne $sourceFile.sha256){throw "UI source bytes differ from build: $($sourceFile.path)"}
    }
    if((Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'Qualified app artifact hash changed.'}
    if(-not(Test-Path -LiteralPath $VisionExe -PathType Leaf)){throw 'Existing FlowUIVision helper is unavailable; set DCM_FLOWUIVISION_EXE explicitly.'}
    $receipt['commit']=$head;$receipt['build_receipt']=[ordered]@{path=$BuildReceipt;sha256=(Hash $BuildReceipt)}
    $receipt['app_sha256']=$build.app_artifact.sha256
    $receipt['flowuivision']=[ordered]@{path=$VisionExe;sha256=(Hash $VisionExe);source_revision='existing native helper; source identity not inferred'}
    $sourceSettings=Get-Content -LiteralPath (Join-Path $repo 'app-src\settings.go') -Raw
    if($sourceSettings -notmatch 'const currentAppVersion = "([^"]+)"'){throw 'Cannot identify source app version.'}
    $version=$Matches[1];$receipt['version']=$version
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    if($Suite -eq 'office-preview-release-operations'){
        Add-Type -AssemblyName UIAutomationClient;Add-Type -AssemblyName UIAutomationTypes;Add-Type -AssemblyName WindowsBase
        Add-Type -Path (Join-Path $repo 'tests/native/dcm-docx-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll',([Windows.Automation.AutomationElement]).Assembly.Location,([Windows.Automation.AutomationPattern]).Assembly.Location,([Windows.Point]).Assembly.Location)
    }
    if($Suite -eq 'pdf'){Add-Type -Path (Join-Path $repo 'tests/native/dcm-image-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')}
    . (Join-Path $repo 'tests\native\new-fixtures.ps1')
    $fixture=New-DcmFixtures $lab
    if($Suite -eq 'cross-volume'){
        # The only off-workspace write location is a fresh, test-owned directory
        # on the distinct D: NTFS volume. Never use or overwrite user files.
        $srcVolume=Get-Volume -DriveLetter E -ErrorAction Stop
        $dstVolume=Get-Volume -DriveLetter D -ErrorAction Stop
        if($srcVolume.UniqueId -eq $dstVolume.UniqueId){throw 'DCM-40 requires physically distinct source/destination volumes'}
        if($dstVolume.FileSystem -ne 'NTFS'){throw 'DCM-40 target must be NTFS'}
        $base='D:\FlowverseTesting\DCM-QuickSort'
        $secondary=Join-Path $base ('DCM40-'+$id)
        if(Test-Path -LiteralPath $secondary){throw 'DCM-40 destination already exists; no overwrite allowed'}
        if(-not(Test-Path -LiteralPath 'D:\' -PathType Container)){throw 'D: volume not mounted'}
        New-Item -ItemType Directory -Path $secondary -Force | Out-Null
        $marker=Join-Path $secondary '.dcm40-disposable-test-owned'
        [IO.File]::WriteAllText($marker,('Source report: '+$lab),$utf8)
        $newDestination=Join-Path $secondary 'Accepted'
        New-Item -ItemType Directory -Path $newDestination | Out-Null
        $settingsPath=Join-Path $fixture.config 'settings.json'
        $settings=Get-Content -LiteralPath $settingsPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $settings.vault_root=$secondary
        [IO.File]::WriteAllText($settingsPath,($settings|ConvertTo-Json),$utf8)
        $fixture.vault=$secondary
        $fixture.destination=$newDestination
        $receipt['cross_volume']=[ordered]@{source_volume=$srcVolume.UniqueId;target_volume=$dstVolume.UniqueId;source_drive='E:';target_drive='D:';isolated_root=$secondary;ownership_marker=$marker;test_files_only=$true}
        if((Get-ChildItem -LiteralPath $newDestination -Force | Measure-Object).Count -ne 0){throw 'Cross-volume destination is not empty'}
    }
    $receipt['configuration_backup_root']=Join-Path $lab 'configuration-backups'
    New-Item -ItemType Directory -Path (Join-Path $lab 'temp') | Out-Null
    if($Suite -in @('previews','office-preview-release-operations')){
        $prior=@()
        $previewNames=if($Suite -eq 'office-preview-release-operations'){@('05-letter.docx','06-orders.xlsx','07-slides.pptx')}else{@('04-image.png','05-letter.docx','06-orders.xlsx','07-slides.pptx')}
        foreach($name in $previewNames){
            $from=if($Suite -eq 'office-preview-release-operations' -and $name -eq '07-slides.pptx'){
                Join-Path $repo 'tests/fixtures/dcm32-slides.pptx'
            }else{Join-Path $PreviousFixtureInbox $name}
            if(-not(Test-Path -LiteralPath $from -PathType Leaf)){throw "Required preview fixture missing: $from"}
            $digest=Hash $from
            if($Suite -eq 'office-preview-release-operations'){
                $pinned=@{'05-letter.docx'='11ca20995a5da66033c8dd75cc55a145bc6179bf189ac7d8bfaf9d90562a72a7';'06-orders.xlsx'='5059729f4390ecd2e169bb2d1eb850c768532ebbb24b9733e87b82b012d6d24f';'07-slides.pptx'='af42fff55297109195b95039d2ac39c5d53b1fae23b59e7b40a7184d679b93f1'}
                if($pinned[$name] -cne $digest){throw "Pinned Office fixture identity mismatch: $name"}
            }
            Copy-Item -LiteralPath $from -Destination (Join-Path $fixture.inbox $name)
            if((Hash (Join-Path $fixture.inbox $name)) -ne $digest){throw 'Preview fixture copy changed bytes.'}
            $prior+=[ordered]@{source=$from;name=$name;sha256=$digest}
        }
        $receipt['reused_preview_fixtures']=$prior
    }
    $initial=@(Get-ChildItem -LiteralPath $fixture.inbox -File | Sort-Object Name | ForEach-Object {[pscustomobject]@{name=$_.Name;sha256=(Hash $_.FullName);size=$_.Length}})
    Write-Json (Join-Path $lab 'fixtures-before.json') $initial
    $exe=Join-Path $lab ('DCM-QS-'+$id+'.exe')
    Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    if((Hash $exe) -ne $build.app_artifact.sha256){throw 'Owned app copy changed bytes.'}
    Start-OwnedApp
    if($Suite -in @('basics','filters')){
        Invoke-Fvs 'filters-text' "click clear`nselect category `"Documents`"`nselect extension `".txt`"`ninspect main`nexpect count text~=`"of 3`""
        Assert-List 'text extension filter' @('01-alpha.txt','02-bravo.txt','03-charlie.txt')
        Invoke-Fvs 'filters-name' "type name `"alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ninspect main`nexpect count text~=`"of 1`""
        Assert-List 'name filter' @('01-alpha.txt')
        Invoke-Fvs 'filters-no-match' "type name `"no-qualification-file-matches`"`ninspect main`nexpect count text~=`"0`""
        Assert-List 'no-match state' @()
        if([DcmNative]::Text([DcmNative]::Control($script:hwnd,$script:app.Id,1005),$script:app.Id) -ne ''){throw 'No-match state retained an actionable filename draft.'}
        $null=Save-Shot 'no-match-state'
        Invoke-Fvs 'filters-age' "click clear`nselect age `"Older than 30 days`"`ninspect main`nexpect count text~=`"of 1`""
        Assert-List 'age filter' @('03-charlie.txt')
        Invoke-Fvs 'filters-size' "click clear`nselect size `"1-10 MiB`"`ninspect main`nexpect count text~=`"0`""
        Assert-List 'size filter' @()
        Invoke-Fvs 'sort-name' "click clear`nselect sort `"Name`"`ninspect main"
        Assert-List 'name sort' @('01-alpha.txt','02-bravo.txt','03-charlie.txt','08-preview.pdf') -Ordered
        $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
        Invoke-Fvs 'next-file' "click next`nwait_ready type=textbox target=newname text=`"02-bravo`" timeout=5s`ninspect main`nexpect newname text=`"02-bravo`""
        Invoke-Fvs 'sort-type' "select sort `"File Type`"`ninspect main"
        Assert-List 'file type sort' @('08-preview.pdf','01-alpha.txt','02-bravo.txt','03-charlie.txt') -Ordered
        Assert-Fixtures
        if($Suite -eq 'filters'){. (Join-Path $repo 'tests\native\filter-matrix.ps1')}
    }
    if($Suite -in @('basics','rename')){
        Invoke-Fvs 'name-clean-spacing' "click clear`ntype name `"03-charlie`"`nwait_ready type=textbox target=newname text=`"03-charlie`" timeout=5s`ntype newname `"03   cleanup   test`"`nselect tool `"Clean spacing`"`nclick apply`ninspect main`nexpect newname text=`"03 cleanup test`""
        Invoke-Fvs 'rename-readiness' "click clear`ntype name `"01-alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ntype newname `"01-renamed-alpha`"`ninspect main`nexpect rename enabled"
        $old=Join-Path $fixture.inbox '01-alpha.txt';$renamed=Join-Path $fixture.inbox '01-renamed-alpha.txt'
        $originalHash=Hash $old
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1006)
        Wait-For 'rename completion' { -not(Test-Path -LiteralPath $old) -and (Test-Path -LiteralPath $renamed) -and [DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1008)) } 20
        if((Hash $renamed) -ne $originalHash){throw 'Rename changed fixture content.'}
        $renameID=Assert-RenameJournal 'rename-journal-completed' $old $renamed 'completed'
        $cases.Add([ordered]@{name='rename independent byte oracle';passed=$true;from=$old;to=$renamed;sha256=$originalHash})
        Save-State 'rename-completed';$null=Save-Shot 'rename-completed'
        Stop-OwnedApp;Start-OwnedApp
        Invoke-Fvs 'rename-recovery-search' "type processedsearch `"01-renamed-alpha`"`ninspect main"
        $processed=[DcmNative]::ListItems($script:hwnd,$script:app.Id,1017)
        $processedMatches=@();for($i=0;$i -lt $processed.Length;$i++){if($processed[$i].Contains('01-renamed-alpha')){$processedMatches+=$i}}
        if($processedMatches.Count -ne 1){throw 'Restarted Processed list did not expose exactly one renamed fixture.'}
        $selected=[DcmNative]::SelectSingleItem($script:hwnd,$script:app.Id,1017,$processedMatches[0])
        $cases.Add([ordered]@{name='durable Rename visible after restart';passed=$true;processed_selection=$selected;journal_id=$renameID})
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Wait-For 'Undo Rename completion' { (Test-Path -LiteralPath $old) -and -not(Test-Path -LiteralPath $renamed) } 20
        Wait-For 'Undo Rename status' {Status-Contains 'Undid processed transaction'}
        Assert-Fixtures
        $null=Assert-RenameJournal 'undo-rename-journal-completed' $old $renamed 'undone' $renameID
        $cases.Add([ordered]@{name='Undo Rename independent byte oracle';passed=$true;restored=$old;sha256=(Hash $old)})
        Save-State 'undo-rename-completed';$null=Save-Shot 'undo-rename-completed'
    }
    if($Suite -eq 'undo-rename'){
        # DCM-37: real UI Undo following restart, including an occupied old
        # pathname. A colliding file is created only as disposable test setup.
        Invoke-Fvs 'dcm37-rename-ready' "click clear`ntype name `"01-alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ntype newname `"01-renamed-alpha`"`ninspect main`nexpect rename enabled"
        $original=Join-Path $fixture.inbox '01-alpha.txt'
        $renamed=Join-Path $fixture.inbox '01-renamed-alpha.txt'
        $originalHash=Hash $original
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1006)
        Wait-For 'DCM-37 rename finished' {
            -not (Test-Path -LiteralPath $original) -and
            (Test-Path -LiteralPath $renamed) -and
            [DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1008))
        } 20
        if((Hash $renamed) -ne $originalHash){throw 'DCM-37 Rename changed source bytes'}
        $id=Assert-RenameJournal 'dcm37-rename-completed-journal' $original $renamed 'completed'
        $cases.Add([ordered]@{name='dcm37-rename-created-transaction';passed=$true;sha256=$originalHash;id=$id})
        $null=Save-Shot 'dcm37-renamed'
        Stop-OwnedApp;Start-OwnedApp
        Invoke-Fvs 'dcm37-processed-search' "type processedsearch `"01-renamed-alpha`"`ninspect main"
        $processed=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1017))
        $indexes=@()
        for($i=0;$i -lt $processed.Count;$i++){
            if($processed[$i].Contains('01-renamed-alpha')){$indexes+=$i}
        }
        if($indexes.Count -ne 1){throw 'DCM-37 restart did not retain exactly one renamed processed transaction'}
        $null=[DcmNative]::SelectSingleItem($script:hwnd,$script:app.Id,1017,$indexes[0])
        $cases.Add([ordered]@{name='dcm37-rename-durable-and-selected-after-restart';passed=$true;id=$id})
        $collisionBytes=[Text.Encoding]::UTF8.GetBytes('DCM37-OCCUPIED-ORIGINAL-V1')
        [IO.File]::WriteAllBytes($original,$collisionBytes)
        $collisionHash=Hash $original
        if($collisionHash -eq $originalHash){throw 'DCM-37 collision fixture is not distinct'}
        $journal=Join-Path $fixture.config 'recovery/move-journal.jsonl'
        $journalBefore=Hash $journal
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Confirm-Dialog 'DCM - Quick Sort' @('cannot undo: original path is already occupied') 2
        if((Hash $original) -ne $collisionHash -or (Hash $renamed) -ne $originalHash){
            throw 'DCM-37 refused Undo damaged colliding or renamed file'
        }
        if((Hash $journal) -ne $journalBefore){throw 'DCM-37 refused Undo changed durable journal'}
        $null=Assert-RenameJournal 'dcm37-refused-undo-journal-remains-completed' $original $renamed 'completed' $id
        $cases.Add([ordered]@{name='dcm37-occupied-original-undo-refused-without-overwrite';passed=$true;collision_sha256=$collisionHash;renamed_sha256=$originalHash;journal_sha256=$journalBefore})
        $null=Save-Shot 'dcm37-refused-occupied-original'
        [IO.File]::Delete($original)
        if(Test-Path -LiteralPath $original){throw 'Cannot clear disposable collision fixture'}
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Wait-For 'DCM-37 Undo after clearing occupied path' {
            (Test-Path -LiteralPath $original) -and -not(Test-Path -LiteralPath $renamed)
        } 20
        Wait-For 'DCM-37 successful Undo status' {Status-Contains 'Undid processed transaction'}
        if((Hash $original) -ne $originalHash){throw 'DCM-37 Undo did not restore original bytes'}
        $null=Assert-RenameJournal 'dcm37-successful-undo-journal' $original $renamed 'undone' $id
        Assert-Fixtures
        $cases.Add([ordered]@{name='dcm37-cleared-original-undo-restores-name-bytes-and-journal';passed=$true;sha256=$originalHash;id=$id})
        $null=Save-Shot 'dcm37-undo-restored'
    }
    if($Suite -eq 'rename-edge'){
        # DCM-36: exact native GUI rename and Undo; no filesystem rename shortcuts.
        # Windows case-only rename must be checked by enumerated exact casing,
        # not Test-Path, because Windows path existence is case-insensitive.
        $original='01-alpha.txt'
        $old=Join-Path $fixture.inbox $original
        $digest=Hash $old
        foreach($scenario in @(
            @{label='spaces';stem='Odisha Project Notes'},
            @{label='unicode';stem=('Caf'+[char]0x00E9+' Odisha 2026')},
            @{label='case-only';stem='01-ALPHA'}
        )){
            $targetName=$scenario.stem+'.txt'
            $target=Join-Path $fixture.inbox $targetName
            $key=[string]$scenario.label
            $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($original))
            Invoke-Fvs ('rename-edge-input-'+$key) ("type newname `""+$scenario.stem+"`"`ninspect main`nexpect rename enabled")
            [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1006)
            Wait-For ('rename edge completed '+$key) {
                $match=@(Get-ChildItem -LiteralPath $fixture.inbox -File | Where-Object {$_.Name -ieq $targetName})
                ($match.Count -eq 1) -and ($match[0].Name -ceq $targetName) -and
                [DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1008))
            } 30
            if((Hash $target) -ne $digest){throw 'Rename edge changed source bytes'}
            $id=Assert-RenameJournal ('rename-edge-'+$key) $old $target 'completed'
            $cases.Add([ordered]@{name=('rename-edge-'+$key+'-exact-name-and-bytes');passed=$true;target=$targetName;sha256=$digest})
            $null=Save-Shot ('rename-edge-'+$key)
            [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
            Wait-For ('rename edge Undo '+$key) {
                $match=@(Get-ChildItem -LiteralPath $fixture.inbox -File | Where-Object {$_.Name -ieq $original})
                ($match.Count -eq 1) -and ($match[0].Name -ceq $original)
            } 30
            $null=Assert-RenameJournal ('rename-edge-undo-'+$key) $old $target 'undone' $id
            if((Hash $old) -ne $digest){throw 'Rename edge Undo altered bytes'}
            Assert-Fixtures
            $cases.Add([ordered]@{name=('rename-edge-'+$key+'-undo-preserved');passed=$true;sha256=$digest})
            $null=Save-Shot ('rename-edge-'+$key+'-undone')
        }
        # Existing-source conflicts and illegal Windows characters must never
        # alter either original fixture or its durable operation history.
        $journal=Join-Path $fixture.config 'recovery/move-journal.jsonl'
        $beforeJournal=Hash $journal
        $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($original))
        Invoke-Fvs 'rename-edge-invalid-input' "type newname `"bad:name`"`ninspect main"
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1006)
        Confirm-Dialog 'DCM - Quick Sort' @('invalid character') 2
        Assert-Fixtures
        if((Hash $journal) -ne $beforeJournal){throw 'Rejected invalid name changed durable journal'}
        $cases.Add([ordered]@{name='rename-invalid-character-refused-without-data-loss';passed=$true;original_sha256=$digest})
        $null=Save-Shot 'rename-edge-invalid-refused'

        $other=Join-Path $fixture.inbox '02-bravo.txt'
        $otherHash=Hash $other
        $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($original))
        Invoke-Fvs 'rename-edge-conflict-input' "type newname `"02-bravo`"`ninspect main"
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1006)
        Confirm-Dialog 'Filename Conflict' @('02-bravo.txt') 2
        Wait-For 'conflicting rename cancelled' {Status-Contains 'Filename conflict action cancelled'} 20
        Assert-Fixtures
        if((Hash $other) -ne $otherHash -or (Hash $old) -ne $digest){throw 'Conflicting name changed fixture bytes'}
        if((Hash $journal) -ne $beforeJournal){throw 'Cancelled conflict changed durable journal'}
        $cases.Add([ordered]@{name='rename-existing-name-conflict-cancelled-without-data-loss';passed=$true;original_sha256=$digest;collision_sha256=$otherHash})
        $null=Save-Shot 'rename-edge-conflict-cancelled'
    }
    if($Suite -eq 'undo-move-safety'){
        # DCM-44: verify the exact journaled move and both unsafe Undo
        # refusals through native QuickSort's owned UI controls.
        $name='01-alpha.txt'
        $src=Join-Path $fixture.inbox $name
        $dst=Join-Path $fixture.destination $name
        $originalHash=Hash $src
        $reference=Join-Path $lab 'dcm44-original-bytes.bin'
        Copy-Item -LiteralPath $src -Destination $reference -ErrorAction Stop
        if((Hash $reference) -ne $originalHash){throw 'DCM-44 baseline copy mismatch'}
        Invoke-Fvs 'dcm44-move-ready' "click clear`ntype name `"01-alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ntype destination `"Accepted`"`ninspect main`nexpect move enabled"
        $selected=@([DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($name)))
        if($selected.Count -ne 1){throw 'DCM-44 did not select exactly one source'}
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Wait-Moved @($name)
        $journal=Join-Path $fixture.config 'recovery/move-journal.jsonl'
        $events=@(Get-Content -LiteralPath $journal -Encoding UTF8 | Where-Object {$_.Trim()} | ForEach-Object {$_|ConvertFrom-Json})
        $record=@($events|Where-Object {$_.record.from -eq $src -and $_.record.to -eq $dst})
        if($record.Count -lt 2 -or $record[-1].state -ne 'completed'){throw 'DCM-44 move journal missing completed transaction'}
        $opID=[string]$record[-1].id
        if(-not $opID -or [string]$record[-1].record.original_sha256 -cne $originalHash){throw 'DCM-44 completed move fingerprint missing or different'}
        if([int64]$record[-1].record.original_size -ne (Get-Item -LiteralPath $dst).Length){throw 'DCM-44 completed move size missing or different'}
        $cases.Add([ordered]@{name='dcm44-single-move-completed-with-source-fingerprint';passed=$true;journal_id=$opID;sha256=$originalHash})
        $null=Save-Shot 'dcm44-move-completed'
        Stop-OwnedApp;Start-OwnedApp
        Invoke-Fvs 'dcm44-recovered-processed' "type processedsearch `"01-alpha`"`ninspect main"
        $items=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1017))
        $idx=@()
        for($i=0;$i -lt $items.Count;$i++){if($items[$i].Contains($name)){$idx+=$i}}
        if($idx.Count -ne 1){throw 'DCM-44 restart did not recover exactly one transaction'}
        $null=[DcmNative]::SelectSingleItem($script:hwnd,$script:app.Id,1017,$idx[0])
        $cases.Add([ordered]@{name='dcm44-single-undo-transaction-recovered-after-restart';passed=$true;journal_id=$opID})
        # Occupied original must refuse Undo and preserve BOTH distinct files.
        [IO.File]::WriteAllBytes($src,[Text.Encoding]::UTF8.GetBytes('DCM44-OCCUPIED-ORIGINAL-DISTINCT'))
        $occupiedHash=Hash $src
        $journalBefore=Hash $journal
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Confirm-Dialog 'DCM - Quick Sort' @('cannot undo: original path is already occupied') 2
        if((Hash $src) -ne $occupiedHash -or (Hash $dst) -ne $originalHash -or (Hash $journal) -ne $journalBefore){throw 'DCM-44 occupied source Undo refusal modified bytes or journal'}
        $cases.Add([ordered]@{name='dcm44-occupied-original-undo-refused-without-loss';passed=$true;source_sha256=$occupiedHash;destination_sha256=$originalHash;journal_sha256=$journalBefore})
        $null=Save-Shot 'dcm44-occupied-original-refused'
        [IO.File]::Delete($src)
        if(Test-Path -LiteralPath $src){throw 'DCM-44 could not remove its disposable occupancy fixture'}
        # Edited moved target must also refuse Undo; never replace the moved
        # original with unapproved bytes.
        [IO.File]::WriteAllBytes($dst,[Text.Encoding]::UTF8.GetBytes('DCM44-MOVED-DESTINATION-EDITED-DISTINCT'))
        $changedHash=Hash $dst
        if($changedHash -eq $originalHash){throw 'DCM-44 edited destination did not differ'}
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Confirm-Dialog 'DCM - Quick Sort' @('cannot undo: moved file contents changed after the move') 2
        if(Test-Path -LiteralPath $src){throw 'DCM-44 changed target Undo created source'}
        if((Hash $dst) -ne $changedHash -or (Hash $journal) -ne $journalBefore){throw 'DCM-44 changed target Undo refusal altered file/journal'}
        $cases.Add([ordered]@{name='dcm44-changed-destination-undo-refused-without-loss';passed=$true;destination_sha256=$changedHash;journal_sha256=$journalBefore})
        $null=Save-Shot 'dcm44-changed-destination-refused'
        # Restore ONLY the disposable test target from its pinned original
        # fixture bytes, not via QuickSort, before asking QuickSort to Undo.
        $baseline=@($initial|Where-Object {$_.name -eq $name})
        if($baseline.Count -ne 1 -or $baseline[0].sha256 -ne $originalHash){throw 'DCM-44 missing baseline fingerprint'}
        $reference=Join-Path $lab 'dcm44-original-bytes.bin'
        if(-not(Test-Path -LiteralPath $reference)){throw 'DCM-44 immutable original backup absent'}
        if((Hash $reference) -ne $originalHash){throw 'DCM-44 immutable original backup was changed'}
        Copy-Item -LiteralPath $reference -Destination $dst -Force
        if((Hash $dst) -ne $originalHash){throw 'DCM-44 original destination bytes could not be restored'}
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Wait-Restored @($name)
        Wait-For 'DCM-44 successful Undo status' {Status-Contains 'Undid processed transaction'} 20
        if((Hash $src) -ne $originalHash){throw 'DCM-44 restored file contents differ'}
        $events=@(Get-Content -LiteralPath $journal -Encoding UTF8 | Where-Object {$_.Trim()} | ForEach-Object {$_|ConvertFrom-Json})
        $record=@($events|Where-Object {$_.id -eq $opID})
        if($record.Count -lt 3 -or $record[-1].state -ne 'undone' -or [string]$record[-1].record.original_sha256 -cne $originalHash){throw 'DCM-44 Undo journal identity or fingerprint mismatch'}
        Assert-Fixtures
        $cases.Add([ordered]@{name='dcm44-normal-single-undo-restored-original-name-location-and-hash';passed=$true;journal_id=$opID;sha256=$originalHash})
        $null=Save-Shot 'dcm44-single-undo-restored'
    }
    if($Suite -eq 'destination'){
        # DCM-38: all inputs through owned UI; independently prove that invalid
        # or unavailable Vault destinations cannot mutate the source or journal.
        $source=Join-Path $fixture.inbox '01-alpha.txt'
        $dest=Join-Path $fixture.destination '01-alpha.txt'
        $initialHash=Hash $source
        $journal=Join-Path $fixture.config 'recovery/move-journal.jsonl'
        $journalBefore=if(Test-Path -LiteralPath $journal){Hash $journal}else{''}
        $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
        foreach($invalid in @(
            @{label='missing-logical-location';value='NotConfigured'},
            @{label='invalid-traversal-location';value='../OutsideVault'}
        )){
            $label=[string]$invalid.label
            $value=[string]$invalid.value
            Invoke-Fvs ('dcm38-'+$label+'-input') ("type destination `""+$value+"`"`ninspect main")
            $typed=[DcmNative]::Text([DcmNative]::Control($script:hwnd,$script:app.Id,1003),$script:app.Id)
            if($typed -cne $value){throw ('Destination UI did not retain rejected input '+$label)}
            [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
            Confirm-Dialog 'DCM - Quick Sort' @('Choose an existing logical location inside the configured Vault') 2
            Assert-Fixtures
            if((Hash $source) -ne $initialHash){throw ('Rejected destination changed source bytes '+$label)}
            $journalAfter=if(Test-Path -LiteralPath $journal){Hash $journal}else{''}
            if($journalAfter -ne $journalBefore){throw ('Rejected destination modified durable journal '+$label)}
            $cases.Add([ordered]@{name=('dcm38-'+$label+'-rejected-without-mutation');passed=$true;input=$value;source_sha256=$initialHash;journal_sha256=$journalAfter})
            $null=Save-Shot ('dcm38-'+$label+'-rejected')
        }
        Invoke-Fvs 'dcm38-valid-vault-location' "type destination `"Accepted`"`ninspect main`nexpect destination text=`"Accepted`"`nexpect move enabled"
        $selected=@([DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt')))
        if($selected.Count -ne 1){throw 'Valid destination does not have exactly one selected source'}
        $actualDestination=[IO.Path]::GetFullPath($fixture.destination)
        if(-not(Test-Path -LiteralPath $actualDestination -PathType Container)){throw 'Test Vault destination unexpectedly missing'}
        $cases.Add([ordered]@{name='dcm38-valid-destination-address-resolved-to-existing-vault-folder';passed=$true;logical='Accepted';physical=$actualDestination})
        $null=Save-Shot 'dcm38-valid-location-ready'
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Wait-Moved @('01-alpha.txt')
        if((Hash $dest) -ne $initialHash){throw 'Valid destination moved different file bytes'}
        Assert-Journal 'dcm38-valid-location-move-journal' @('01-alpha.txt') 'completed'
        $cases.Add([ordered]@{name='dcm38-valid-location-moved-only-selected-file';passed=$true;logical='Accepted';physical=$actualDestination;sha256=$initialHash})
        $null=Save-Shot 'dcm38-valid-location-moved'
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Wait-Restored @('01-alpha.txt')
        Assert-Journal 'dcm38-valid-location-undo-journal' @('01-alpha.txt') 'undone'
        Assert-Fixtures
        if((Hash $source) -ne $initialHash){throw 'Valid destination Undo altered source'}
        $cases.Add([ordered]@{name='dcm38-valid-location-undo-restored-fixtures';passed=$true;sha256=$initialHash})
        $null=Save-Shot 'dcm38-valid-location-undone'
    }
    if($Suite -eq 'cross-volume'){
        $source=Join-Path $fixture.inbox '01-alpha.txt'
        $target=Join-Path $fixture.destination '01-alpha.txt'
        $digest=Hash $source
        $other=@($initial|Where-Object {$_.name -ne '01-alpha.txt'})
        $marker=Join-Path $fixture.vault '.dcm40-disposable-test-owned'
        if(-not(Test-Path -LiteralPath $marker -PathType Leaf)){throw 'Cross-volume ownership marker absent'}
        if([IO.Path]::GetPathRoot($source) -eq [IO.Path]::GetPathRoot($target)){throw 'Cross-volume test accidentally targets same drive'}
        Invoke-Fvs 'dcm40-valid-cross-volume' "click clear`ntype name `"01-alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ntype destination `"Accepted`"`ninspect main`nexpect destination text=`"Accepted`"`nexpect move enabled"
        $selected=@([DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt')))
        if($selected.Count -ne 1){throw 'Cross-volume selector did not select exactly one file'}
        $cases.Add([ordered]@{name='dcm40-real-distinct-E-to-D-volumes-and-selection';passed=$true;from=$source;to=$target;source_volume=$receipt.cross_volume.source_volume;target_volume=$receipt.cross_volume.target_volume})
        $null=Save-Shot 'dcm40-before-copy'
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Wait-Moved @('01-alpha.txt')
        Assert-Journal 'dcm40-cross-volume-completed' @('01-alpha.txt') 'completed'
        if((Hash $target) -ne $digest){throw 'Cross-volume destination copy differs from source hash'}
        foreach($item in $other){
            if((Hash (Join-Path $fixture.inbox $item.name)) -ne $item.sha256){throw ('Unselected file changed: '+$item.name)}
        }
        $cases.Add([ordered]@{name='dcm40-copy-verified-and-source-safely-removed';passed=$true;sha256=$digest})
        $null=Save-Shot 'dcm40-cross-volume-moved'
        Stop-OwnedApp;Start-OwnedApp
        Invoke-Fvs 'dcm40-recovery-search' "type processedsearch `"01-alpha`"`ninspect main"
        $items=[DcmNative]::ListItems($script:hwnd,$script:app.Id,1017)
        $indices=@()
        for($i=0;$i -lt $items.Length;$i++){if($items[$i].Contains('01-alpha')){$indices+=$i}}
        if($indices.Count -ne 1){throw 'Cross-volume move was not recovered after restart'}
        $null=[DcmNative]::SelectSingleItem($script:hwnd,$script:app.Id,1017,$indices[0])
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Wait-Restored @('01-alpha.txt')
        Assert-Journal 'dcm40-cross-volume-undone' @('01-alpha.txt') 'undone'
        if((Hash $source) -ne $digest){throw 'Cross-volume Undo source hash mismatch'}
        Assert-Fixtures
        $cases.Add([ordered]@{name='dcm40-cross-volume-restart-and-undo-preserves-bytes';passed=$true;sha256=$digest})
        $null=Save-Shot 'dcm40-cross-volume-undone'

        # Hold a share-read-only handle: copy is permitted, but delete/rename
        # of the original is denied on NTFS. The result MUST be attention,
        # not success and not source loss.
        $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
        Invoke-Fvs 'dcm40-partial-ready' "type destination `"Accepted`"`ninspect main`nexpect move enabled"
        $hold=[IO.File]::Open($source,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::Read)
        try {
            [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
            Confirm-Dialog 'DCM - Quick Sort' @('Review these files before retrying','Source:') 2
            if(-not(Test-Path -LiteralPath $source -PathType Leaf)){throw 'Guarded source retirement lost original file'}
            if(-not(Test-Path -LiteralPath $target -PathType Leaf)){throw 'Expected retained verified copy absent on partial failure'}
            if((Hash $source) -ne $digest -or (Hash $target) -ne $digest){throw 'Partial move copied file or original bytes mismatch'}
            $path=Join-Path $fixture.config 'recovery/move-journal.jsonl'
            $events=@(Get-Content -LiteralPath $path -Encoding UTF8 | Where-Object {$_.Trim()} | ForEach-Object {$_|ConvertFrom-Json})
            $matching=@($events|Where-Object {$_.record.from -eq $source -and $_.record.to -eq $target})
            if($matching.Count -lt 5 -or $matching[-1].state -ne 'attention'){throw 'Partial move did not record attention state'}
            if(-not(Status-Contains 'Source retained; the move needs attention')){throw 'Partial move UI did not report attention-required state'}
            foreach($item in $other){
                if((Hash (Join-Path $fixture.inbox $item.name)) -ne $item.sha256){throw ('Partial failure modified unselected '+$item.name)}
            }
            $cases.Add([ordered]@{name='dcm40-retirement-refused-source-and-copy-both-retained-attention';passed=$true;source_sha256=(Hash $source);destination_sha256=(Hash $target);journal_terminal_state=$matching[-1].state})
            $null=Save-Shot 'dcm40-source-retained-attention'
        }finally{
            $hold.Dispose()
        }
        # Preserve the partial copy and attention journal as immutable evidence
        # within uniquely marked disposable test locations. Never auto-delete.
        $receipt['partial_failure_preserved']=[ordered]@{source=$source;destination=$target;sha256=$digest;journal=(Join-Path $fixture.config 'recovery/move-journal.jsonl');ownership_marker=$marker}
    }
    if($Suite -eq 'conflicts'){
        # DCM-41: Every conflicting file is created by this disposable test.
        # The app owns every move, conflict decision and Undo. The harness
        # exclusively owns setup and independent file/journal verification.
        $name='01-alpha.txt'
        $src=Join-Path $fixture.inbox $name
        $dst=Join-Path $fixture.destination $name
        $originalHash=Hash $src
        $journal=Join-Path $fixture.config 'recovery/move-journal.jsonl'
        $seed=[Text.Encoding]::UTF8.GetBytes('DCM41-EXISTING-DESTINATION-DIFFERENT-BYTES')
        $scenarioList=@(
            @{label='skip';kind='different';button=2},
            @{label='keep-both';kind='different';button=6},
            @{label='replace';kind='different';button=7},
            @{label='duplicate-no';kind='duplicate';button=7},
            @{label='duplicate-yes';kind='duplicate';button=6}
        )
        foreach($scenario in $scenarioList){
            $label=[string]$scenario.label
            if(-not(Test-Path -LiteralPath $src -PathType Leaf) -or (Test-Path -LiteralPath $dst)){throw ('DCM-41 precondition failed for '+$label)}
            $expectedDestinationHash=''
            if($scenario.kind -eq 'different'){
                [IO.File]::WriteAllBytes($dst,$seed)
                $expectedDestinationHash=Hash $dst
                if($expectedDestinationHash -eq $originalHash){throw 'Different-file seed unexpectedly identical'}
            }else{
                Copy-Item -LiteralPath $src -Destination $dst -ErrorAction Stop
                $expectedDestinationHash=Hash $dst
                if($expectedDestinationHash -ne $originalHash){throw 'Duplicate seed differs'}
            }
            $before=if(Test-Path -LiteralPath $journal){Hash $journal}else{''}
            Invoke-Fvs ('dcm41-'+$label+'-ready') "click clear`ntype name `"01-alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ntype destination `"Accepted`"`ninspect main`nexpect move enabled"
            $selected=@([DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($name)))
            if($selected.Count -ne 1){throw ('DCM-41 not exactly one selected source for '+$label)}
            [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
            if($scenario.kind -eq 'different'){
                Confirm-Dialog 'Filename Conflict' @('Keep Both','Replace','Cancel') ([int]$scenario.button)
            }else{
                Confirm-Dialog 'Exact Duplicate' @('SHA-256','Remove the source duplicate','Keep the source unchanged') ([int]$scenario.button)
            }
            if($label -in @('skip','duplicate-no')){
                Wait-For ('DCM-41 '+$label+' refusal status') {
                    if($label -eq 'skip'){Status-Contains 'Filename conflict action cancelled'}
                    else{Status-Contains 'Exact duplicate action skipped'}
                } 20
                if((Hash $src) -ne $originalHash -or (Hash $dst) -ne $expectedDestinationHash){throw ('DCM-41 '+$label+' changed file bytes')}
                $after=if(Test-Path -LiteralPath $journal){Hash $journal}else{''}
                if($after -ne $before){throw ('DCM-41 '+$label+' changed durable journal')}
                $cases.Add([ordered]@{name=('dcm41-'+$label+'-preserves-both-without-journal');passed=$true;source_sha256=$originalHash;destination_sha256=$expectedDestinationHash})
                $null=Save-Shot ('dcm41-'+$label)
            }else{
                Wait-For ('DCM-41 '+$label+' completed') {
                    (-not(Test-Path -LiteralPath $src)) -and [DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1008))
                } 35
                $finalTarget=$dst
                $kind='move'
                if($label -eq 'keep-both'){
                    $extras=@(Get-ChildItem -LiteralPath $fixture.destination -File | Where-Object {$_.Name -ne $name})
                    if($extras.Count -ne 1){throw 'Keep Both did not create exactly one numbered destination'}
                    $finalTarget=$extras[0].FullName
                    $kind='move'
                    if((Hash $dst) -ne $expectedDestinationHash){throw 'Keep Both modified original destination'}
                }
                if($label -eq 'replace'){$kind='replace'}
                if($label -eq 'duplicate-yes'){$kind='duplicate-removed'}
                if((Hash $finalTarget) -ne $originalHash){throw ('DCM-41 '+$label+' moved bytes differ')}
                $events=@(Get-Content -LiteralPath $journal -Encoding UTF8 | Where-Object {$_.Trim()} | ForEach-Object {$_|ConvertFrom-Json})
                $records=@($events|Where-Object {$_.record.from -eq $src -and $_.record.to -eq $finalTarget})
                $operationID=[string]$records[-1].id
                $transaction=@($records | Where-Object {$_.id -eq $operationID})
                if($transaction.Count -lt 2 -or $transaction[0].state -ne 'pending' -or $transaction[-1].state -ne 'completed'){throw ('DCM-41 '+$label+' journal not completed')}
                if([string]$transaction[-1].record.kind -cne $kind){throw ('DCM-41 '+$label+' unexpected journal operation kind')}
                if($label -eq 'replace'){
                    if(-not $records[-1].record.replaced_backup){throw 'Replace journal did not retain displaced-file backup'}
                    $backup=[string]$records[-1].record.replaced_backup
                    if(-not(Test-Path -LiteralPath $backup -PathType Leaf) -or (Hash $backup) -ne $expectedDestinationHash){throw 'Replacement backup not preserved'}
                }
                $cases.Add([ordered]@{name=('dcm41-'+$label+'-completed-with-byte-oracle');passed=$true;target=$finalTarget;sha256=$originalHash;kind=$kind;id=$records[-1].id})
                $null=Save-Shot ('dcm41-'+$label+'-completed')
                [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
                Wait-For ('DCM-41 '+$label+' Undo') {
                    (Test-Path -LiteralPath $src -PathType Leaf) -and
                    ((Get-ChildItem -LiteralPath $fixture.destination -File|Measure-Object).Count -eq 1)
                } 30
                Wait-For ('DCM-41 '+$label+' journal Undo finalization') {
                    $seen=@(Get-Content -LiteralPath $journal -Encoding UTF8 | Where-Object {$_.Trim()} | ForEach-Object {$_|ConvertFrom-Json} | Where-Object {$_.id -eq $operationID})
                    return $seen.Count -ge 3 -and $seen[-1].state -eq 'undone'
                } 20
                if((Hash $src) -ne $originalHash -or (Hash $dst) -ne $expectedDestinationHash){throw ('DCM-41 '+$label+' Undo corrupted source/destination')}
                $remaining=@(Get-ChildItem -LiteralPath $fixture.destination -File)
                if($remaining.Count -ne 1 -or $remaining[0].Name -ne $name){throw ('DCM-41 '+$label+' Undo left unexpected destination')}
                $events=@(Get-Content -LiteralPath $journal -Encoding UTF8 | Where-Object {$_.Trim()} | ForEach-Object {$_|ConvertFrom-Json})
                $records=@($events|Where-Object {$_.record.from -eq $src -and $_.record.to -eq $finalTarget})
                $transaction=@($records | Where-Object {$_.id -eq $operationID})
                if($transaction.Count -lt 3 -or $transaction[0].state -ne 'pending' -or $transaction[-1].state -ne 'undone' -or $transaction[-1].id -ne $operationID){throw ('DCM-41 '+$label+' Undo journal wrong state/identity')}
                $cases.Add([ordered]@{name=('dcm41-'+$label+'-undo-restored-both-hashes');passed=$true;source_sha256=$originalHash;destination_sha256=$expectedDestinationHash})
                $null=Save-Shot ('dcm41-'+$label+'-undone')
            }
            # Remove only this test-created destination after verifying outcome;
            # never remove the source or its original baseline fixtures.
            if((Hash $dst) -ne $expectedDestinationHash){throw 'DCM-41 cleanup guard: seeded destination changed'}
            Remove-Item -LiteralPath $dst -Force
            Assert-Fixtures
            $cases.Add([ordered]@{name=('dcm41-'+$label+'-baseline-restored');passed=$true})
        }
    }
    if($Suite -eq 'move'){
        Invoke-Fvs 'move-readiness' "click clear`ntype name `"01-alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ntype destination `"Accepted`"`ninspect main`nexpect destination text=`"Accepted`"`nexpect move enabled"
        $indexes=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
        $cases.Add([ordered]@{name='single native selection readback';passed=$true;indexes=$indexes;names=@('01-alpha.txt')})
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Wait-Moved @('01-alpha.txt')
        Assert-Journal 'move-completed' @('01-alpha.txt') 'completed'
        Save-State 'move-completed';$null=Save-Shot 'move-completed'
        Stop-OwnedApp;Start-OwnedApp
        Invoke-Fvs 'move-recovery-search' "type processedsearch `"01-alpha`"`ninspect main"
        $items=[DcmNative]::ListItems($script:hwnd,$script:app.Id,1017)
        $index=-1;for($i=0;$i -lt $items.Length;$i++){if($items[$i].Contains('01-alpha')){$index=$i}}
        if($index -lt 0){throw 'Restarted Processed list did not recover moved fixture.'}
        $selected=[DcmNative]::SelectSingleItem($script:hwnd,$script:app.Id,1017,$index)
        $cases.Add([ordered]@{name='durable move visible after restart';passed=$true;processed_selection=$selected})
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Wait-Restored @('01-alpha.txt')
        Wait-For 'Undo status' {Status-Contains 'Undid processed transaction'}
        Assert-Journal 'undo-move-completed' @('01-alpha.txt') 'undone'
        Save-State 'undo-move-completed';$null=Save-Shot 'undo-move-completed'
    }
    if($Suite -eq 'preview-release-operations'){
        # DCM-35: same source PDF through preview/release/rename/Undo/move/Undo.
        $pdf='08-preview.pdf'
        $old=Join-Path $fixture.inbox $pdf
        $renamed=Join-Path $fixture.inbox '08-preview-released.pdf'
        $digest=Hash $old
        foreach($operation in @('rename','move')){
            $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($pdf))
            Wait-For ('PDF ready before '+$operation) {
                (Status-Contains 'DCM PDF Preview') -and (Status-Contains 'Page 1/2')
            } 30
            $cases.Add([ordered]@{name=('real-PDF-preview-before-'+$operation);passed=$true;sha256=(Hash $old)})
            $null=Save-Shot ('pdf-before-'+$operation)
            $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
            Wait-For ('TXT after PDF '+$operation) {Status-Contains 'Plain-text preview: TXT'} 20
            $handle=[IO.File]::Open($old,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None)
            $handle.Dispose()
            if((Hash $old) -ne $digest){throw 'PDF changed during release'}
            $cases.Add([ordered]@{name=('same-PDF-exclusively-released-before-'+$operation);passed=$true;sha256=$digest})
            $null=Save-Shot ('text-after-pdf-'+$operation)
            # Make the target explicit again. Operation succeeds even when the
            # file is selected and its preview is active/settled.
            $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($pdf))
            Wait-For ('PDF reselected for '+$operation) {
                (Status-Contains 'DCM PDF Preview') -and (Status-Contains 'Page 1/2')
            } 30
            if($operation -eq 'rename'){
                Invoke-Fvs 'released-pdf-rename-ready' "type newname `"08-preview-released`"`ninspect main`nexpect rename enabled"
                [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1006)
                Wait-For 'same PDF renamed' {
                    -not(Test-Path -LiteralPath $old) -and
                    (Test-Path -LiteralPath $renamed) -and
                    [DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1008))
                } 30
                if((Hash $renamed) -ne $digest){throw 'Renamed PDF bytes changed'}
                $renameID=Assert-RenameJournal 'released-pdf-rename-journal' $old $renamed 'completed'
                $null=Save-Shot 'released-pdf-renamed'
                [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
                Wait-For 'PDF rename Undo' {
                    (Test-Path -LiteralPath $old) -and -not(Test-Path -LiteralPath $renamed)
                } 30
                $null=Assert-RenameJournal 'released-pdf-undo-rename-journal' $old $renamed 'undone' $renameID
            } else {
                Invoke-Fvs 'released-pdf-move-ready' "type destination `"Accepted`"`ninspect main`nexpect destination text=`"Accepted`"`nexpect move enabled"
                [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
                Wait-Moved @($pdf)
                Assert-Journal 'released-pdf-move-journal' @($pdf) 'completed'
                $null=Save-Shot 'released-pdf-moved'
                [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
                Wait-Restored @($pdf)
                Assert-Journal 'released-pdf-undo-move-journal' @($pdf) 'undone'
            }
            if((Hash $old) -ne $digest){throw 'PDF Undo bytes changed'}
            Assert-Fixtures
            $cases.Add([ordered]@{name=('same-PDF-'+$operation+'-undo-bytes-preserved');passed=$true;sha256=$digest})
            $null=Save-Shot ('released-pdf-'+$operation+'-undone')
        }
        $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
        Wait-For 'final TXT after PDF operations' {Status-Contains 'Plain-text preview: TXT'} 20
        $handle=[IO.File]::Open($old,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None)
        $handle.Dispose()
        Assert-Fixtures
        $cases.Add([ordered]@{name='final-PDF-exclusively-released-and-fixtures-preserved';passed=$true;sha256=(Hash $old)})
    }
    if($Suite -eq 'office-preview-release-operations'){
        # DCM-35: real installed provider or built-in fallback, no registry edits.
        # Each operation is performed by native QuickSort buttons on the SAME
        # previously previewed file; filesystem operations here are oracles only.
        $items=@(
            @{file='05-letter.docx';status=@('Windows preview: DOCX','DCM DOCX Preview')},
            @{file='06-orders.xlsx';status=@('Windows preview: XLSX','DCM XLSX Preview')},
            @{file='07-slides.pptx';status=@('Windows preview: PPTX','DCM PPTX Preview')}
        )
        foreach($item in $items){
            $file=[string]$item.file
            $from=Join-Path $fixture.inbox $file
            $digest=Hash $from
            $stem=[IO.Path]::GetFileNameWithoutExtension($file)
            $ext=[IO.Path]::GetExtension($file)
            $renameName=$stem+'-released'
            $renamed=Join-Path $fixture.inbox ($renameName+$ext)
            foreach($operation in @('rename','move')){
                $key=$stem+'-'+$operation
                $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($file))
                Wait-For ('Office provider for '+$key) {
                    $found=$false
                    foreach($prefix in $item.status){if(OfficeStatusContains $prefix){$found=$true}}
                    return $found
                } 30
                $cases.Add([ordered]@{name=('office-preview-before-'+$key);passed=$true;sha256=$digest;status=@([DcmDocxProbe]::HostStatics($script:hwnd,$script:app.Id)|ForEach-Object {[DcmNative]::Text([IntPtr]$_,$script:app.Id)}|Where-Object {$_ -match 'preview'})})
                $null=Save-Shot ('office-before-'+$key)
                $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
                Wait-For ('TXT after Office '+$key) {OfficeStatusContains 'Plain-text preview: TXT'} 20
                # Retry only the lock oracle while a genuine provider releases.
                $until=[DateTime]::UtcNow.AddSeconds(12)
                $released=$false
                do {
                    try {
                        $h=[IO.File]::Open($from,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None)
                        $h.Dispose()
                        $released=$true
                    }catch [IO.IOException] {
                        Start-Sleep -Milliseconds 150
                    }
                } while (-not $released -and [DateTime]::UtcNow -lt $until)
                if(-not $released){throw ('Office source retained a file lock after switch: '+$file)}
                if((Hash $from) -ne $digest){throw ('Office source changed after release: '+$file)}
                $cases.Add([ordered]@{name=('office-exclusively-released-before-'+$key);passed=$true;sha256=$digest})
                $null=Save-Shot ('office-text-after-'+$key)
                $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($file))
                Wait-For ('Office reselected for '+$key) {
                    $found=$false
                    foreach($prefix in $item.status){if(OfficeStatusContains $prefix){$found=$true}}
                    return $found
                } 30
                if($operation -eq 'rename'){
                    Invoke-Fvs ('office-rename-ready-'+$stem) ("type newname `""+$renameName+"`"`ninspect main`nexpect rename enabled")
                    [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1006)
                    Wait-For ('Office rename completed '+$file) {
                        -not(Test-Path -LiteralPath $from) -and
                        (Test-Path -LiteralPath $renamed) -and
                        [DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1008))
                    } 30
                    if((Hash $renamed) -ne $digest){throw ('Office rename changed bytes: '+$file)}
                    $id=Assert-RenameJournal ('office-rename-'+$stem) $from $renamed 'completed'
                    $null=Save-Shot ('office-renamed-'+$stem)
                    [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
                    Wait-For ('Office Undo Rename '+$file) {
                        (Test-Path -LiteralPath $from) -and -not(Test-Path -LiteralPath $renamed)
                    } 30
                    $null=Assert-RenameJournal ('office-undo-rename-'+$stem) $from $renamed 'undone' $id
                }else{
                    Invoke-Fvs ('office-move-ready-'+$stem) "type destination `"Accepted`"`ninspect main`nexpect destination text=`"Accepted`"`nexpect move enabled"
                    [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
                    Wait-Moved @($file)
                    Assert-Journal ('office-move-'+$stem) @($file) 'completed'
                    $null=Save-Shot ('office-moved-'+$stem)
                    [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
                    Wait-Restored @($file)
                    Assert-Journal ('office-undo-move-'+$stem) @($file) 'undone'
                }
                if((Hash $from) -ne $digest){throw ('Office Undo changed bytes: '+$file)}
                Assert-Fixtures
                $cases.Add([ordered]@{name=('office-'+$key+'-undo-preserved');passed=$true;sha256=$digest})
                $null=Save-Shot ('office-'+$key+'-undone')
            }
        }
        $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
        Wait-For 'final TXT after Office operations' {OfficeStatusContains 'Plain-text preview: TXT'} 20
        foreach($item in $items){
            $p=Join-Path $fixture.inbox $item.file
            $h=[IO.File]::Open($p,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None)
            $h.Dispose()
        }
        Assert-Fixtures
        $cases.Add([ordered]@{name='all-Office-fixtures-exclusively-released-and-preserved';passed=$true})
    }
    if($Suite -eq 'batch'){
        $names=[string[]]@('02-bravo.txt','03-charlie.txt')
        Invoke-Fvs 'batch-readiness' "click clear`ntype destination `"Accepted`"`ninspect main`nexpect destination text=`"Accepted`""
        $indexes=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,$names)
        $cases.Add([ordered]@{name='batch native selection readback';passed=$true;indexes=$indexes;names=$names})
        Invoke-Fvs 'batch-selection-visible' ("inspect main`n"+'expect move text~="Move 2"'+"`n"+'expect count text~="2 selected"')
        $null=Save-Shot 'batch-selection'
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Confirm-Dialog 'Move Selected Files to Vault' @('Move 2 selected files',$fixture.destination) 7
        Wait-For 'batch cancellation' {Status-Contains 'Batch move cancelled'}
        Assert-Fixtures
        $cases.Add([ordered]@{name='cancel batch leaves all fixture bytes unchanged';passed=$true})
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Confirm-Dialog 'Move Selected Files to Vault' @('Move 2 selected files',$fixture.destination)
        Wait-Moved $names
        Wait-For 'batch completed' {Status-Contains 'Batch move complete: 2 of 2'}
        Assert-Journal 'batch-completed' $names 'completed' -Batch
        Save-State 'batch-completed';$null=Save-Shot 'batch-completed'
        Stop-OwnedApp;Start-OwnedApp
        Invoke-Fvs 'batch-recovered-after-restart' "inspect main`nexpect undobatch enabled"
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1048)
        Confirm-Dialog 'Undo Latest Batch' @('Undo the latest batch of 2 processed file(s)')
        Wait-Restored $names
        Wait-For 'batch restored status' {Status-Contains 'Undid latest batch: 2 file(s) restored'}
        Assert-Journal 'undo-batch-completed' $names 'undone' -Batch
        Save-State 'undo-batch-completed';$null=Save-Shot 'undo-batch-completed'
    }
    if($Suite -eq 'batch-undo-safety'){
        # DCM-45: prove that the native Undo Batch preflights all members
        # after restart, and refuses occupied or altered members atomically.
        $names=[string[]]@('02-bravo.txt','03-charlie.txt')
        $bravo=Join-Path $fixture.inbox $names[0]
        $charlie=Join-Path $fixture.inbox $names[1]
        $bravoDest=Join-Path $fixture.destination $names[0]
        $charlieDest=Join-Path $fixture.destination $names[1]
        $bravoHash=Hash $bravo
        $charlieHash=Hash $charlie
        $bravoBackup=Join-Path $lab 'dcm45-bravo-original.bin'
        $charlieBackup=Join-Path $lab 'dcm45-charlie-original.bin'
        Copy-Item -LiteralPath $bravo -Destination $bravoBackup -ErrorAction Stop
        Copy-Item -LiteralPath $charlie -Destination $charlieBackup -ErrorAction Stop
        if((Hash $bravoBackup) -ne $bravoHash -or (Hash $charlieBackup) -ne $charlieHash){throw 'DCM-45 immutable original backups mismatch'}
        Invoke-Fvs 'dcm45-selected-batch-ready' "click clear`ntype destination `"Accepted`"`ninspect main`nexpect destination text=`"Accepted`""
        $indexes=@([DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,$names))
        if($indexes.Count -ne 2){throw 'DCM-45 did not select exactly two files'}
        $null=Save-Shot 'dcm45-batch-selected'
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Confirm-Dialog 'Move Selected Files to Vault' @('Move 2 selected files',$fixture.destination)
        Wait-Moved $names
        Wait-For 'DCM45 batch completed' {Status-Contains 'Batch move complete: 2 of 2'} 30
        Assert-Journal 'dcm45-batch-completed' $names 'completed' -Batch
        $journal=Join-Path $fixture.config 'recovery/move-journal.jsonl'
        $events=@(Get-Content -LiteralPath $journal -Encoding UTF8 | Where-Object {$_.Trim()} | ForEach-Object {$_|ConvertFrom-Json})
        $last=@($events|Where-Object {$_.record.from -in @($bravo,$charlie) -and $_.state -eq 'completed'})
        if($last.Count -ne 2){throw 'DCM-45 expected exactly two completed batch members'}
        $batchID=[string]$last[0].record.batch_id
        if(-not $batchID -or $last[1].record.batch_id -ne $batchID){throw 'DCM-45 members missing shared batch ID'}
        foreach($entry in $last){
            $expected=if($entry.record.from -eq $bravo){$bravoHash}else{$charlieHash}
            if([string]$entry.record.original_sha256 -cne $expected){throw 'DCM-45 member missing durable destination fingerprint'}
        }
        $cases.Add([ordered]@{name='dcm45-two-journaled-members-and-original-hashes';passed=$true;batch_id=$batchID;files=$names})
        $null=Save-Shot 'dcm45-batch-completed'
        Stop-OwnedApp;Start-OwnedApp
        Invoke-Fvs 'dcm45-after-restart' "inspect main`nexpect undobatch enabled"
        $items=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1017))
        if(@($items|Where-Object{$_.Contains('02-bravo.txt') -or $_.Contains('03-charlie.txt')}).Count -ne 2){throw 'DCM-45 restarted Processed did not contain both members'}
        $cases.Add([ordered]@{name='dcm45-restarted-batch-recovered-exactly-two-members';passed=$true;batch_id=$batchID})
        $null=Save-Shot 'dcm45-batch-recovered'
        # Source occupancy: create only a disposable original-path collision.
        [IO.File]::WriteAllBytes($charlie,[Text.Encoding]::UTF8.GetBytes('DCM45-OCCUPIED-CHARLIE-DISTINCT'))
        $occupiedHash=Hash $charlie
        $journalBefore=Hash $journal
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1048)
        Confirm-Dialog 'Undo Latest Batch' @('Undo the latest batch of 2 processed file(s)') -ExpectFollowupDialog
        Confirm-Dialog 'DCM - Quick Sort' @('batch undo preflight failed','original path is already occupied') 2
        if((Hash $charlie) -ne $occupiedHash -or (Hash $bravoDest) -ne $bravoHash -or (Hash $charlieDest) -ne $charlieHash -or (Hash $journal) -ne $journalBefore){throw 'DCM-45 occupied original refusal modified the batch'}
        if(Test-Path -LiteralPath $bravo){throw 'DCM-45 occupied-path preflight partially restored other member'}
        $cases.Add([ordered]@{name='dcm45-occupied-member-preflight-refuses-entire-batch';passed=$true;occupant_sha256=$occupiedHash;journal_sha256=$journalBefore})
        $null=Save-Shot 'dcm45-occupied-refused'
        [IO.File]::Delete($charlie)
        if(Test-Path -LiteralPath $charlie){throw 'DCM-45 failed to clear disposable occupancy'}
        # Modified destination: this member's recorded fingerprint must fail
        # before *either* member can be moved back.
        [IO.File]::WriteAllBytes($charlieDest,[Text.Encoding]::UTF8.GetBytes('DCM45-MODIFIED-DESTINATION-DISTINCT'))
        $changedHash=Hash $charlieDest
        if($changedHash -eq $charlieHash){throw 'DCM-45 edited file unexpectedly matches baseline'}
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1048)
        Confirm-Dialog 'Undo Latest Batch' @('Undo the latest batch of 2 processed file(s)') -ExpectFollowupDialog
        Confirm-Dialog 'DCM - Quick Sort' @('batch undo preflight failed','moved file contents changed') 2
        if(Test-Path -LiteralPath $bravo -or (Test-Path -LiteralPath $charlie)){throw 'DCM-45 changed-member preflight partially restored the batch'}
        if((Hash $bravoDest) -ne $bravoHash -or (Hash $charlieDest) -ne $changedHash -or (Hash $journal) -ne $journalBefore){throw 'DCM-45 changed-member refusal modified batch bytes or journal'}
        $cases.Add([ordered]@{name='dcm45-modified-member-preflight-refuses-entire-batch';passed=$true;changed_sha256=$changedHash;journal_sha256=$journalBefore})
        $null=Save-Shot 'dcm45-modified-refused'
        # Restore known disposable target from its pinned immutable backup,
        # then let QuickSort perform the *only* actual batch Undo.
        if((Hash $charlieBackup) -ne $charlieHash){throw 'DCM-45 immutable charlie backup changed'}
        Copy-Item -LiteralPath $charlieBackup -Destination $charlieDest -Force
        if((Hash $charlieDest) -ne $charlieHash){throw 'DCM-45 target recovery fixture hash mismatch'}
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1048)
        Confirm-Dialog 'Undo Latest Batch' @('Undo the latest batch of 2 processed file(s)')
        Wait-Restored $names
        Wait-For 'DCM-45 batch Undo success status' {Status-Contains 'Undid latest batch: 2 file(s) restored'} 20
        Assert-Journal 'dcm45-restarted-batch-undone' $names 'undone' -Batch
        if((Hash $bravo) -ne $bravoHash -or (Hash $charlie) -ne $charlieHash){throw 'DCM-45 restored files differ from original hashes'}
        $cases.Add([ordered]@{name='dcm45-restarted-batch-safe-retry-restored-both';passed=$true;batch_id=$batchID;files=$names})
        $null=Save-Shot 'dcm45-batch-undone'
        Assert-Fixtures
    }
    if($Suite -eq 'batch-partial'){
        # DCM-42 supplemental: one selected file is moved successfully,
        # one has a pre-existing disposable conflicting target and is skipped.
        # Both remain members of the confirmed two-file plan, but only the
        # successful move has a completed transaction in the journal.
        $names=[string[]]@('02-bravo.txt','03-charlie.txt')
        $bravo=Join-Path $fixture.inbox '02-bravo.txt'
        $charlie=Join-Path $fixture.inbox '03-charlie.txt'
        $bravoTarget=Join-Path $fixture.destination '02-bravo.txt'
        $charlieTarget=Join-Path $fixture.destination '03-charlie.txt'
        $bravoSHA=Hash $bravo
        $charlieSHA=Hash $charlie
        $collision=[Text.Encoding]::UTF8.GetBytes('DCM42-SYNTHETIC-CONFLICT-DISTINCT-CONTENTS')
        [IO.File]::WriteAllBytes($charlieTarget,$collision)
        $collisionSHA=Hash $charlieTarget
        if($collisionSHA -eq $charlieSHA){throw 'DCM-42 collision accidentally identical'}
        Invoke-Fvs 'batch-partial-readiness' "click clear`ntype destination `"Accepted`"`ninspect main`nexpect destination text=`"Accepted`""
        $idx=@([DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,$names))
        if($idx.Count -ne 2){throw 'DCM-42 partial batch selected wrong number of files'}
        $cases.Add([ordered]@{name='batch-partial-select-exact-two-files';passed=$true;files=$names;indexes=$idx})
        $null=Save-Shot 'batch-partial-selection'
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Confirm-Dialog 'Move Selected Files to Vault' @('Move 2 selected files',$fixture.destination) -ExpectFollowupDialog
        Confirm-Dialog 'Filename Conflict' @('03-charlie.txt','Keep Both','Replace') 2
        Wait-For 'DCM-42 mixed result status' {Status-Contains 'Batch move finished: 1 moved; 1 skipped or failed out of 2'} 30
        if(Test-Path -LiteralPath $bravo){throw 'Successful batch source was not removed'}
        if(-not(Test-Path -LiteralPath $bravoTarget)){throw 'Successful batch destination absent'}
        if((Hash $bravoTarget) -ne $bravoSHA){throw 'Successful batch destination hash mismatch'}
        if((Hash $charlie) -ne $charlieSHA -or (Hash $charlieTarget) -ne $collisionSHA){throw 'Skipped file or conflicting destination changed'}
        foreach($name in @('01-alpha.txt','08-preview.pdf')){
            $item=@($initial|Where-Object{$_.name -eq $name})
            if($item.Count -ne 1 -or (Hash (Join-Path $fixture.inbox $name)) -ne $item[0].sha256){throw ('Unselected fixture changed: '+$name)}
        }
        $journal=Join-Path $fixture.config 'recovery/move-journal.jsonl'
        $events=@(Get-Content -LiteralPath $journal -Encoding UTF8 | Where-Object{$_.Trim()} | ForEach-Object{$_|ConvertFrom-Json})
        $success=@($events|Where-Object{$_.record.from -eq $bravo -and $_.record.to -eq $bravoTarget})
        $skipped=@($events|Where-Object{$_.record.from -eq $charlie})
        if($success.Count -lt 2 -or $success[0].state -ne 'pending' -or $success[-1].state -ne 'completed'){throw 'Successful batch member journal not completed'}
        if($skipped.Count -ne 0){throw 'Skipped batch member received a false success or journal transaction'}
        $batchID=[string]$success[-1].record.batch_id
        if(-not $batchID -or $batchID -notlike 'batch-*'){throw 'Successful member missing batch identity'}
        if(@($success|Where-Object{$_.record.batch_id -ne $batchID}).Count -gt 0){throw 'Batch identity inconsistent across journal events'}
        $cases.Add([ordered]@{name='batch-partial-per-file-results-and-journal-membership';passed=$true;moved='02-bravo.txt';skipped='03-charlie.txt';batch_id=$batchID;source_sha256=$bravoSHA;skipped_sha256=$charlieSHA;collision_sha256=$collisionSHA})
        $null=Save-Shot 'batch-partial-completed'
        Stop-OwnedApp;Start-OwnedApp
        Invoke-Fvs 'batch-partial-recovery' "inspect main`nexpect undobatch enabled"
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1048)
        Confirm-Dialog 'Undo Latest Batch' @('Undo the latest batch of 1 processed file(s)')
        Wait-For 'partial batch restored' {
            (Test-Path -LiteralPath $bravo) -and (-not(Test-Path -LiteralPath $bravoTarget))
        } 30
        Wait-For 'partial batch Undo status' {Status-Contains 'Undid latest batch: 1 file(s) restored'} 20
        if((Hash $bravo) -ne $bravoSHA -or (Hash $charlie) -ne $charlieSHA -or (Hash $charlieTarget) -ne $collisionSHA){throw 'Partial batch Undo corrupted file bytes'}
        $events=@(Get-Content -LiteralPath $journal -Encoding UTF8 | Where-Object{$_.Trim()} | ForEach-Object{$_|ConvertFrom-Json})
        $success=@($events|Where-Object{$_.record.from -eq $bravo -and $_.record.to -eq $bravoTarget})
        if($success[-1].state -ne 'undone' -or $success[-1].record.batch_id -ne $batchID){throw 'Partial batch Undo journal mismatch'}
        if(@($events|Where-Object{$_.record.from -eq $charlie}).Count -gt 0){throw 'Partial batch Undo inserted skipped member journal'}
        $cases.Add([ordered]@{name='batch-partial-restart-and-only-successful-member-undo';passed=$true;batch_id=$batchID;original_hash=$bravoSHA})
        $null=Save-Shot 'batch-partial-undone'
        if((Hash $charlieTarget) -ne $collisionSHA){throw 'Test-owned collision file altered before cleanup'}
        Remove-Item -LiteralPath $charlieTarget -Force
        Assert-Fixtures
        $cases.Add([ordered]@{name='batch-partial-fully-restored-fixtures';passed=$true})
    }
    if($Suite -eq 'history'){
        # DCM-43: Completed and refused file operations must not be conflated.
        # Search, selection and inspection of Processed cannot modify any
        # source, Vault destination or durable journal event.
        $movedName='02-bravo.txt'
        $refusedName='03-charlie.txt'
        $src=Join-Path $fixture.inbox $movedName
        $dst=Join-Path $fixture.destination $movedName
        $refusedSrc=Join-Path $fixture.inbox $refusedName
        $refusedDst=Join-Path $fixture.destination $refusedName
        $movedHash=Hash $src
        $refusedHash=Hash $refusedSrc
        $journal=Join-Path $fixture.config 'recovery/move-journal.jsonl'
        $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($movedName))
        Invoke-Fvs 'dcm43-success-ready' "type destination `"Accepted`"`ninspect main`nexpect move enabled"
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Wait-Moved @($movedName)
        Assert-Journal 'dcm43-success-journal' @($movedName) 'completed'
        $completed=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1017))
        if(@($completed|Where-Object{$_.Contains($movedName)}).Count -ne 1){throw 'Completed move not represented in Processed'}
        if((Hash $dst) -ne $movedHash){throw 'Completed move content mismatch'}
        $cases.Add([ordered]@{name='dcm43-success-visible-in-processed';passed=$true;file=$movedName;sha256=$movedHash})
        $null=Save-Shot 'dcm43-successful-processed'
        # A distinct preexisting destination must cause a per-file refusal.
        $seed=[Text.Encoding]::UTF8.GetBytes('DCM43-REFUSED-DESTINATION-PRESERVE')
        [IO.File]::WriteAllBytes($refusedDst,$seed)
        $collisionHash=Hash $refusedDst
        $journalBefore=Hash $journal
        $null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@($refusedName))
        Invoke-Fvs 'dcm43-refusal-ready' "type destination `"Accepted`"`ninspect main`nexpect move enabled"
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
        Confirm-Dialog 'Filename Conflict' @($refusedName,'Keep Both','Replace') 2
        Wait-For 'DCM43 refusal status' {Status-Contains 'Filename conflict action cancelled'} 20
        if((Hash $refusedSrc) -ne $refusedHash -or (Hash $refusedDst) -ne $collisionHash){throw 'Refused move changed file bytes'}
        if((Hash $journal) -ne $journalBefore){throw 'Refused move changed journal'}
        $processed=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1017))
        if(@($processed|Where-Object{$_.Contains($refusedName)}).Count -ne 0){throw 'Refused move falsely appears as completed Processed'}
        $cases.Add([ordered]@{name='dcm43-refused-not-misrepresented-as-success';passed=$true;refused=$refusedName;preserved_hash=$refusedHash;collision_hash=$collisionHash})
        $null=Save-Shot 'dcm43-refused-not-processed'
        # Snapshot all relevant bytes and journal. From this point, Processed
        # views, searches and selected-history readback must be read-only.
        $stateBefore=[ordered]@{journal=(Hash $journal);source=(Hash $refusedSrc);collision=(Hash $refusedDst);destination=(Hash $dst)}
        Invoke-Fvs 'dcm43-history-filter-refused' "type processedsearch `"03-charlie`"`ninspect main"
        $filtered=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1017))
        if(@($filtered|Where-Object{$_.Contains($refusedName)}).Count -ne 0){throw 'Refused item appeared under Processed search'}
        $cases.Add([ordered]@{name='dcm43-refused-search-has-no-success-row';passed=$true})
        $null=Save-Shot 'dcm43-refused-history-search'
        Invoke-Fvs 'dcm43-history-filter-success' "type processedsearch `"02-bravo`"`ninspect main"
        $filtered=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1017))
        $idx=@()
        for($i=0;$i -lt $filtered.Count;$i++){if($filtered[$i].Contains($movedName)){$idx+=$i}}
        if($idx.Count -ne 1){throw 'Successful item not found once by Processed search'}
        $null=[DcmNative]::SelectSingleItem($script:hwnd,$script:app.Id,1017,$idx[0])
        $cases.Add([ordered]@{name='dcm43-success-history-search-and-selection';passed=$true;selected=$movedName})
        $null=Save-Shot 'dcm43-success-history-search'
        Invoke-Fvs 'dcm43-history-clear-search' "type processedsearch `"`"`ninspect main"
        Save-State 'dcm43-history-view'
        if((Hash $journal) -ne $stateBefore.journal -or (Hash $refusedSrc) -ne $stateBefore.source -or (Hash $refusedDst) -ne $stateBefore.collision -or (Hash $dst) -ne $stateBefore.destination){throw 'Viewing history modified files or durable journal'}
        $cases.Add([ordered]@{name='dcm43-history-navigation-readonly-byte-and-journal-proof';passed=$true;journal_sha256=$stateBefore.journal})
        $null=Save-Shot 'dcm43-history-readonly'
        Stop-OwnedApp;Start-OwnedApp
        Invoke-Fvs 'dcm43-restart-history-search' "type processedsearch `"02-bravo`"`ninspect main"
        $again=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1017))
        if(@($again|Where-Object{$_.Contains($movedName)}).Count -ne 1){throw 'Restart failed to recover successful Processed entry'}
        if((Hash $journal) -ne $stateBefore.journal -or (Hash $refusedSrc) -ne $stateBefore.source -or (Hash $refusedDst) -ne $stateBefore.collision -or (Hash $dst) -ne $stateBefore.destination){throw 'Restart viewing history changed bytes or journal'}
        $cases.Add([ordered]@{name='dcm43-restarted-history-display-remains-readonly';passed=$true})
        $null=Save-Shot 'dcm43-restarted-history'
        # The synthetic conflicting destination is test-owned. All history
        # read-only checks above proved it unchanged; remove it before the
        # standard Wait-Restored oracle, which requires an empty Vault.
        if((Hash $refusedDst) -ne $collisionHash){throw 'History read-only collision changed before cleanup'}
        Remove-Item -LiteralPath $refusedDst -Force
        # Undo is the only deliberately mutating history action.
        $list=@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1017))
        $idx=@()
        for($i=0;$i -lt $list.Count;$i++){if($list[$i].Contains($movedName)){$idx+=$i}}
        if($idx.Count -ne 1){throw 'Cannot find history record for Undo'}
        $null=[DcmNative]::SelectSingleItem($script:hwnd,$script:app.Id,1017,$idx[0])
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Wait-Restored @($movedName)
        Assert-Journal 'dcm43-undo-history' @($movedName) 'undone'
        if((Hash $src) -ne $movedHash -or (Hash $refusedSrc) -ne $refusedHash){throw 'Undo or refusal altered source bytes'}
        $cases.Add([ordered]@{name='dcm43-explicit-undo-restores-and-journals';passed=$true})
        $null=Save-Shot 'dcm43-undo-completed'
        Assert-Fixtures
        $cases.Add([ordered]@{name='dcm43-original-fixtures-restored';passed=$true})
    }
    if($Suite -in @('pdf','previews')){
        Invoke-Fvs 'pdf-open' "click clear`ntype name `"08-preview`"`nwait_ready type=textbox target=newname text=`"08-preview`" timeout=5s`nwait_ready type=control target=pdfstatus text~=`"Page 1/2`" timeout=20s`ninspect main`nexpect pdfstatus text~=`"Fit Width`""
        if($Suite -eq 'pdf'){Assert-PdfPage 'pdf-first-boundary' 1}
        foreach($case in @(@('pdf-fit-width','width','Fit Width'),@('pdf-100','zoomreset','100%'),@('pdf-110','zoomin','110%'),@('pdf-return-100','zoomout','100%'))){
            Invoke-Fvs $case[0] ("click "+$case[1]+"`nwait_ready type=control target=pdfstatus text~=`""+$case[2]+"`" timeout=20s`ndelay 400ms`ninspect main`nexpect pdfstatus text~=`""+$case[2]+"`"")
            $null=Save-Shot ($case[0]+'-window')
            $pixels=Save-Shot ($case[0]+'-document') -Preview
            if($pixels.teal_samples -lt 100){throw "$($case[0]) has no adequate visible teal document content; footer state is insufficient."}
            $cases.Add([ordered]@{name=($case[0]+' visible document pixels');passed=$true;teal_samples=$pixels.teal_samples;visual_reviewed=$false})
            if($Suite -eq 'pdf' -and $case[0] -in @('pdf-100','pdf-110')){Pan-PdfToTop ($case[0]+'-top-readable')}
        }
        if($Suite -eq 'pdf'){
            Invoke-Fvs 'pdf-fit-page' "click fit`nwait_ready type=control target=pdfstatus text~=`"Fit`" timeout=20s`ndelay 400ms`ninspect main"
            if(Status-Contains 'Fit Width'){throw 'Fit Page did not replace Fit Width'}
            Assert-PdfPage 'pdf-fit-page-visible' 1
        }
        Invoke-Fvs 'pdf-page-two' "click nextpage`nwait_ready type=control target=pdfstatus text~=`"Page 2/2`" timeout=20s`ndelay 400ms`ninspect main`nexpect pdfstatus text~=`"Page 2/2`""
        $pixels=Save-Shot 'pdf-page-two-document' -Preview
        if($pixels.orange_samples -lt 100){throw 'Second PDF page did not display orange document content.'}
        if($Suite -eq 'pdf'){Assert-PdfPage 'pdf-last-boundary' 2}
        Invoke-Fvs 'pdf-page-one-return' "click prevpage`nwait_ready type=control target=pdfstatus text~=`"Page 1/2`" timeout=20s`nclick width`nwait_ready type=control target=pdfstatus text~=`"Fit Width`" timeout=20s`ndelay 400ms`ninspect main"
        $pixels=Save-Shot 'pdf-page-one-return-document' -Preview
        if($pixels.teal_samples -lt 100){throw 'Returning to PDF page one did not restore teal content.'}
        if($Suite -eq 'pdf'){
            Assert-PdfPage 'pdf-first-boundary-restored' 1
            $journal=Join-Path $fixture.config 'recovery\move-journal.jsonl'
            if((Test-Path $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Preview-only workflow changed the operation journal'}
            $h=[IO.File]::Open((Join-Path $fixture.inbox '08-preview.pdf'),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()
            $cases.Add([ordered]@{name='default-PDF-source-released-and-journal-empty';passed=$true})
        }
        Assert-Fixtures
    }
    if($Suite -eq 'previews'){
        foreach($f in @(@('text','01-alpha'),@('png','04-image'),@('docx','05-letter'),@('xlsx','06-orders'),@('pptx','07-slides'))){
            Invoke-Fvs ('preview-'+$f[0]) ("click clear`ntype name `""+$f[1]+"`"`nwait_ready type=textbox target=newname text=`""+$f[1]+"`" timeout=5s`ndelay 1800ms`ninspect main`nexpect count text~=`"of 1`"")
            Save-State ('preview-'+$f[0]);$null=Save-Shot ('preview-'+$f[0])
        }
        Invoke-Fvs 'pptx-slide-two' "inspect main`nexpect slides visible`nexpect slides count=2"
        $observation=Get-Content -LiteralPath (Join-Path $lab 'pptx-slide-two.vision.json') -Raw -Encoding UTF8 | ConvertFrom-Json
        $slideItems=@(($observation.Observations.main | Where-Object {$_.ID -eq '1025'}).Items)
        if($slideItems.Count -ne 2){throw 'PPTX fixture did not expose two slides.'}
        for($i=1;$i -ge 0;$i--){Invoke-Fvs ('pptx-select-'+$i) ("select slides `""+$slideItems[$i]+"`"`ndelay 1000ms`ninspect main`nexpect slides text=`""+$slideItems[$i]+"`"");$null=Save-Shot ('pptx-select-'+$i)}
        Invoke-Fvs 'xlsx-sheet-two' "click clear`ntype name `"06-orders`"`nwait_ready type=textbox target=newname text=`"06-orders`" timeout=5s`ndelay 1000ms`ninspect main`nexpect sheets visible`nselect sheets `"Read me`"`ndelay 700ms`ninspect main`nexpect sheets text=`"Read me`""
        $null=Save-Shot 'xlsx-sheet-two'
        Assert-Fixtures
    }
    Save-State 'final';$null=Save-Shot 'final'
    Stop-OwnedApp
    $afterHead=(& git.exe -C $repo rev-parse HEAD | Out-String).Trim()
    $afterDirty=(& git.exe -C $repo status --porcelain=v1 --untracked-files=all | Out-String).Trim()
    if($LASTEXITCODE -ne 0 -or $afterDirty -or $afterHead -ne $head){throw 'Source identity changed during UI qualification.'}
    if((Hash $build.app_artifact.path) -ne $build.app_artifact.sha256 -or (Hash $exe) -ne $build.app_artifact.sha256){throw 'App bytes changed during UI qualification.'}
    $receipt['source_unchanged']=$true;$receipt['all_owned_processes_closed']=$true;$receipt.passed=$true
} catch {
    $receipt['error']=$_.Exception.Message
    Write-Output ('FAIL='+$_.Exception.Message)
    try{if($script:app -and -not $script:app.HasExited){Save-State 'failure';$null=Save-Shot 'failure'}}catch{$receipt['failure_capture_error']=$_.Exception.Message}
} finally {
    try{Stop-OwnedApp}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    Write-Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
