[CmdletBinding()]
param(
    [ValidateSet('basics','move','batch','pdf','previews')][string]$Suite='move',
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
function Assert-Owned {
    if($null -eq $script:app){throw 'Owned app has not started.'}
    $script:app.Refresh()
    if($script:app.HasExited){throw 'Owned app exited before its acceptance step completed.'}
    if($script:app.Path -ne $script:owned.exe -or $script:app.StartTime.ToUniversalTime().ToString('o') -ne $script:owned.start){throw 'App process identity changed.'}
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
function Start-OwnedApp {
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe
    $psi.Arguments=Quote-Argument $fixture.inbox
    $psi.WorkingDirectory=$lab
    $psi.UseShellExecute=$false
    $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local
    $psi.EnvironmentVariables['APPDATA']=$fixture.roaming
    $psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'configuration-backups'
    $psi.EnvironmentVariables['TEMP']=Join-Path $lab 'temp'
    $psi.EnvironmentVariables['TMP']=Join-Path $lab 'temp'
    $script:app=New-Object Diagnostics.Process
    $script:app.StartInfo=$psi
    if(-not $script:app.Start()){throw 'Could not launch exact candidate copy.'}
    $until=[DateTime]::UtcNow.AddSeconds(20)
    do {$script:app.Refresh();if($script:app.HasExited){throw 'Candidate exited during launch.'};$script:hwnd=$script:app.MainWindowHandle;if($script:hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100} while([DateTime]::UtcNow -lt $until)
    if($script:hwnd -eq [IntPtr]::Zero){throw 'Candidate main window deadline.'}
    $script:owned=[ordered]@{pid=$script:app.Id;exe=$exe;start=$script:app.StartTime.ToUniversalTime().ToString('o');hwnd=$script:hwnd.ToInt64();sha256=(Hash $exe)}
    $processes.Add($script:owned)
    Write-Json (Join-Path $lab ('owned-'+$processes.Count+'.json')) $script:owned
    Assert-Owned
    if([DcmNative]::Text($script:hwnd,$script:app.Id) -notlike ('*'+$version+'*')){throw 'Candidate title does not identify the source version.'}
    $null=Save-Shot ('launch-'+$processes.Count)
}
function Stop-OwnedApp {
    if($null -eq $script:app){return}
    $script:app.Refresh()
    if(-not $script:app.HasExited){
        Assert-Owned
        # Retain identity of direct renderer children before parent shutdown.
        $children=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$script:app.Id) | ForEach-Object {[ordered]@{pid=[int]$_.ProcessId;path=$_.ExecutablePath;created=$_.CreationDate.ToUniversalTime().ToString('o')}})
        $script:owned['children_before_close']=$children
        [DcmNative]::CloseAsync($script:hwnd,$script:app.Id)
        if(-not $script:app.WaitForExit(15000)){
            Assert-Owned
            & "$env:SystemRoot\System32\taskkill.exe" /PID $script:app.Id /T /F | Out-Null
            $null=$script:app.WaitForExit(10000)
            $script:owned['forced_stop']=$true
            throw 'Owned app required forced shutdown; lifecycle acceptance failed.'
        }
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
    }
    $script:owned['closed_utc']=[DateTime]::UtcNow.ToString('o')
    $script:owned['exit_code']=$script:app.ExitCode
    $script:app.Dispose();$script:app=$null;$script:hwnd=[IntPtr]::Zero
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
function Confirm-Dialog([string]$Title,[string[]]$TextFragments,[int]$Button=6){
    Wait-For 'expected confirmation' { @([DcmNative]::Dialogs($script:hwnd,$script:app.Id)).Count -gt 0 } 10 -AllowDialog
    $dialogs=@([DcmNative]::Dialogs($script:hwnd,$script:app.Id))
    if($dialogs.Count -ne 1 -or -not $dialogs[0].Text.StartsWith($Title)){Write-Json (Join-Path $lab 'unexpected-confirmation.json') $dialogs;throw 'Unexpected confirmation dialog; no response was sent.'}
    $dialog=[IntPtr]$dialogs[0].Handle
    $controls=[DcmNative]::Snapshot($dialog,$script:app.Id)
    $text=($controls | ForEach-Object {$_.Text}) -join "`n"
    foreach($fragment in $TextFragments){if(-not $text.Contains($fragment)){throw "Confirmation does not identify expected fixture scope: $fragment"}}
    Write-Json (Join-Path $lab ('confirmation-'+$cases.Count+'.json')) ([ordered]@{title=$dialogs[0].Text;controls=$controls;button=$Button})
    [DcmNative]::ClickAsync($dialog,$script:app.Id,$Button)
    Wait-For 'confirmation closed' { @([DcmNative]::Dialogs($script:hwnd,$script:app.Id)).Count -eq 0 } 10 -AllowDialog
}
function Status-Contains([string]$Text){return @([DcmNative]::Snapshot($script:hwnd,$script:app.Id) | Where-Object {$_.Visible -and $_.Text.Contains($Text)}).Count -gt 0}
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
    . (Join-Path $repo 'tests\native\new-fixtures.ps1')
    $fixture=New-DcmFixtures $lab
    $receipt['configuration_backup_root']=Join-Path $lab 'configuration-backups'
    New-Item -ItemType Directory -Path (Join-Path $lab 'temp') | Out-Null
    if($Suite -eq 'previews'){
        $prior=@()
        foreach($name in @('04-image.png','05-letter.docx','06-orders.xlsx','07-slides.pptx')){
            $from=Join-Path $PreviousFixtureInbox $name
            if(-not(Test-Path -LiteralPath $from -PathType Leaf)){throw "Required historical preview fixture missing: $from"}
            $digest=Hash $from
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
    if($Suite -eq 'basics'){
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
        Invoke-Fvs 'name-clean-spacing' "click clear`ntype name `"03-charlie`"`nwait_ready type=textbox target=newname text=`"03-charlie`" timeout=5s`ntype newname `"03   cleanup   test`"`nselect tool `"Clean spacing`"`nclick apply`ninspect main`nexpect newname text=`"03 cleanup test`""
        Invoke-Fvs 'rename-readiness' "click clear`ntype name `"01-alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ntype newname `"01-renamed-alpha`"`ninspect main`nexpect rename enabled"
        $old=Join-Path $fixture.inbox '01-alpha.txt';$renamed=Join-Path $fixture.inbox '01-renamed-alpha.txt'
        $originalHash=Hash $old
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1006)
        Wait-For 'rename completion' { -not(Test-Path -LiteralPath $old) -and (Test-Path -LiteralPath $renamed) -and [DcmNative]::IsWindowEnabled([DcmNative]::Control($script:hwnd,$script:app.Id,1008)) } 20
        if((Hash $renamed) -ne $originalHash){throw 'Rename changed fixture content.'}
        $cases.Add([ordered]@{name='rename independent byte oracle';passed=$true;from=$old;to=$renamed;sha256=$originalHash})
        Save-State 'rename-completed';$null=Save-Shot 'rename-completed'
        [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1008)
        Wait-For 'Undo Rename completion' { (Test-Path -LiteralPath $old) -and -not(Test-Path -LiteralPath $renamed) } 20
        Wait-For 'Undo Rename status' {Status-Contains 'Undid processed transaction'}
        Assert-Fixtures
        $cases.Add([ordered]@{name='Undo Rename independent byte oracle';passed=$true;restored=$old;sha256=(Hash $old)})
        Save-State 'undo-rename-completed';$null=Save-Shot 'undo-rename-completed'
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
    if($Suite -in @('pdf','previews')){
        Invoke-Fvs 'pdf-open' "click clear`ntype name `"08-preview`"`nwait_ready type=textbox target=newname text=`"08-preview`" timeout=5s`nwait_ready type=control target=pdfstatus text~=`"Page 1/2`" timeout=20s`ninspect main`nexpect pdfstatus text~=`"Fit Width`""
        foreach($case in @(@('pdf-fit-width','width','Fit Width'),@('pdf-100','zoomreset','100%'),@('pdf-110','zoomin','110%'),@('pdf-return-100','zoomout','100%'))){
            Invoke-Fvs $case[0] ("click "+$case[1]+"`nwait_ready type=control target=pdfstatus text~=`""+$case[2]+"`" timeout=20s`ndelay 400ms`ninspect main`nexpect pdfstatus text~=`""+$case[2]+"`"")
            $null=Save-Shot ($case[0]+'-window')
            $pixels=Save-Shot ($case[0]+'-document') -Preview
            if($pixels.teal_samples -lt 100){throw "$($case[0]) has no adequate visible teal document content; footer state is insufficient."}
            $cases.Add([ordered]@{name=($case[0]+' visible document pixels');passed=$true;teal_samples=$pixels.teal_samples;visual_reviewed=$false})
        }
        Invoke-Fvs 'pdf-page-two' "click nextpage`nwait_ready type=control target=pdfstatus text~=`"Page 2/2`" timeout=20s`ndelay 400ms`ninspect main`nexpect pdfstatus text~=`"Page 2/2`""
        $pixels=Save-Shot 'pdf-page-two-document' -Preview
        if($pixels.orange_samples -lt 100){throw 'Second PDF page did not display orange document content.'}
        Invoke-Fvs 'pdf-page-one-return' "click prevpage`nwait_ready type=control target=pdfstatus text~=`"Page 1/2`" timeout=20s`nclick width`nwait_ready type=control target=pdfstatus text~=`"Fit Width`" timeout=20s`ndelay 400ms`ninspect main"
        $pixels=Save-Shot 'pdf-page-one-return-document' -Preview
        if($pixels.teal_samples -lt 100){throw 'Returning to PDF page one did not restore teal content.'}
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
