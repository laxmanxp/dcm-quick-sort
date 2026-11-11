[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT)
# DCM-25: Refresh, folder switching and exact operation targets; isolated fixtures only.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt){throw 'Workspace and exact build receipt required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-refresh-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Lab must be outside checkout.'}
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-25';app='dcm-quicksort';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;cases=$cases;captures=$captures}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$head='';$manifest=@();$expected=@{};$texts=@{};$roots=@();$mutationLog=New-Object 'Collections.Generic.List[object]';$operationsStarted=$false
Write-Output ('REPORT_ROOT='+$lab)
function Write-Json([string]$Path,$Value){[IO.File]::WriteAllText($Path,($Value | ConvertTo-Json -Depth 30),$utf8)}
function Hash([string]$Path){return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()}
function Git-Value([string[]]$Arguments){$s=(& git.exe -C $repo @Arguments | Out-String).Trim();if($LASTEXITCODE -ne 0){throw 'Git query failed.'};return $s}
function Check-Source {
    if((Git-Value @('rev-parse','HEAD')) -ne $head -or (Git-Value @('status','--porcelain=v1','--untracked-files=all'))){throw 'Checkout identity/clean status changed.'}
    foreach($f in $manifest){$p=[IO.Path]::GetFullPath((Join-Path $repo $f.path));if(-not $p.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $p) -ne $f.sha256){throw 'Source manifest mismatch.'}}
}
function Require-App {
    if($null -eq $app){throw 'Owned app absent.'};$app.Refresh()
    if($app.HasExited -or $app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $owned.start){throw 'Owned process identity/liveness changed.'}
    [DcmNative]::RequireOwned($hwnd,$app.Id)
}
function Wait-For([string]$Name,[scriptblock]$Check){$until=[DateTime]::UtcNow.AddSeconds(15);do{Require-App;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count -gt 1){throw 'Unexpected multiple dialogs.'};if(& $Check){return};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until);throw ('Deadline: '+$Name)}
function Text([int]$Id){return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,$Id),$app.Id)}
function Check-Files {
    foreach($dir in $roots){
        $want=@($expected.Keys | Where-Object {[IO.Path]::GetDirectoryName($_) -eq $dir})
        $files=@();if(Test-Path -LiteralPath $dir -PathType Container){$files=@(Get-ChildItem -LiteralPath $dir -Force)}
        if($files.Count -ne $want.Count){throw ('Fixture file set changed: '+$dir)}
        foreach($f in $files){if($f.PSIsContainer -or -not $expected.ContainsKey($f.FullName) -or (Hash $f.FullName) -ne $expected[$f.FullName]){throw ('Fixture names/bytes changed: '+$f.FullName)}}
    }
    $journal=Join-Path $config 'recovery\move-journal.jsonl'
    if(-not $script:operationsStarted -and (Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Read-only workflow appended a file-operation journal.'}
}
function Visible-Text([string]$Value){return @([DcmNative]::Snapshot($hwnd,$app.Id) | Where-Object {$_.Visible -and $_.Text.Contains($Value)}).Count -gt 0}
function Assert-View([string]$Case,[string]$Folder,[string[]]$Names,[string]$Selected,[string]$EmptyHeading='No files remaining',[switch]$Capture){
    Wait-For $Case { (Text 1001) -eq $Folder -and (@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($Names -join '|') }
    $indexes=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016))
    $snapshot=@([DcmNative]::Snapshot($hwnd,$app.Id))
    if($Selected){
        $index=[Array]::IndexOf($Names,$Selected)
        if($index -lt 0 -or $indexes.Count -ne 1 -or $indexes[0] -ne $index){throw ('DCM25_SELECTION_MISMATCH: '+$Case)}
        if((Text 1005) -cne [IO.Path]::GetFileNameWithoutExtension($Selected)){throw ('DCM25_STALE_DRAFT: '+$Case+' draft='+(Text 1005))}
        $path=Join-Path $Folder $Selected
        if(-not @($snapshot | Where-Object {$_.Visible -and $_.Class -eq 'Edit' -and $_.Text -ceq $texts[$path]}).Count){throw ('DCM25_STALE_PREVIEW: '+$Case)}
        if(-not @($snapshot | Where-Object {$_.Visible -and $_.Text.StartsWith('PREVIEW') -and $_.Text.EndsWith($Selected)}).Count){throw 'Preview heading disagrees with selection.'}
    }else{
        if($indexes.Count -ne 0 -or (Text 1005) -ne ''){throw ('DCM25_STALE_EMPTY_TARGET: '+$Case)}
        if(-not (Visible-Text $EmptyHeading)){throw ('Unexpected empty/error state: '+$Case)}
        if(@($snapshot | Where-Object {$_.Visible -and $_.Class -eq 'Edit' -and $_.Text.StartsWith('DCM-25 fixture:')}).Count){throw 'Old preview remains visible after clearing current file.'}
    }
    Check-Files
    $cases.Add([ordered]@{name=$Case;source=$Folder;files=$Names;selected=$Selected;draft=(Text 1005);passed=$true})
    if($Capture){Shot $Case $hwnd}
}
function Select-File([string]$Name){$null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@($Name));Start-Sleep -Milliseconds 150}
function Set-Draft([string]$Value){Require-App;[DcmFolderEdit]::Set([DcmNative]::Control($hwnd,$app.Id,1005),$Value);if((Text 1005) -cne $Value){throw 'Test draft was not entered.'}}
function Reload {
    Require-App;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected dialog before Reload.'}
    [DcmNative]::ClickAsync($hwnd,$app.Id,1011);Start-Sleep -Milliseconds 350
    if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected dialog after Reload.'}
}
function Map-Fixture([string]$Old,[string]$New){
    $expected[$New]=$expected[$Old];$texts[$New]=$texts[$Old];$expected.Remove($Old);$texts.Remove($Old)
}
function Shot([string]$Name,[IntPtr]$Window){
    Require-App
    $screen=[Windows.Forms.Screen]::FromHandle($Window).WorkingArea
    [DcmNative]::Restore($Window,$app.Id,$screen.Left+12,$screen.Top+12)
    Start-Sleep -Milliseconds 350
    Write-Json (Join-Path $lab ($Name+'.controls.json')) ([DcmNative]::Snapshot($Window,$app.Id))
    $s=[DcmNative]::Capture($Window,$app.Id,(Join-Path $lab ($Name+'.png')),$false)
    $captures.Add([ordered]@{feature='DCM-25';name=$Name;path=$s.Path;sha256=(Hash $s.Path);width=$s.Width;height=$s.Height;visual_reviewed=$false})
}
function Open-Picker {
    Require-App
    if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Refusing input with an unexpected dialog.'}
    [DcmNative]::ClickAsync($hwnd,$app.Id,1002)
    Wait-For 'owned folder picker' { @([DcmNative]::Dialogs($hwnd,$app.Id)).Count -gt 0 }
    $d=@([DcmNative]::Dialogs($hwnd,$app.Id))
    if($d.Count -ne 1 -or $d[0].Text -ne 'Choose folder to sort'){throw 'Unexpected Browse dialog; no response sent.'}
    return [IntPtr]$d[0].Handle
}
function Cancel-Picker([IntPtr]$Dialog){
    Require-App;[DcmNative]::RequireOwned($Dialog,$app.Id)
    if([DcmNative]::Text($Dialog,$app.Id) -ne 'Choose folder to sort'){throw 'Refusing to cancel unknown dialog.'}
    [DcmNative]::ClickAsync($Dialog,$app.Id,2)
    Wait-For 'picker cancellation' { @([DcmNative]::Dialogs($hwnd,$app.Id)).Count -eq 0 }
}
function Check-SourceView([string]$Path,[string[]]$Names){
    Wait-For 'source folder and list' { (Text 1001) -eq $Path -and (@([DcmNative]::ListItems($hwnd,$app.Id,1016) | Sort-Object) -join '|') -ceq (@($Names | Sort-Object) -join '|') }
    Check-Files
    $cases.Add([ordered]@{name='source/list/read-only integrity';source=$Path;files=$Names;passed=$true})
}
function Select-PickerPath([string]$Folder,[string]$Name){
    if($Folder -notin @($inbox,$target,$empty) -or -not(Test-Path -LiteralPath $Folder -PathType Container)){throw 'Refusing picker path outside declared fixtures.'}
    $dialog=Open-Picker
    $edit=[DcmNative]::Control($dialog,$app.Id,1152)
    if([DcmNative]::ClassName($edit) -ne 'Edit' -or -not [DcmNative]::IsWindowVisible($edit) -or -not [DcmNative]::IsWindowEnabled($edit)){throw 'Observed folder field is unavailable.'}
    if([DcmNative]::Text([DcmNative]::Control($dialog,$app.Id,1090),$app.Id) -ne 'Folder:'){throw 'Picker field label changed.'}
    [DcmFolderEdit]::Set($edit,$Folder)
    if([DcmNative]::Text($edit,$app.Id) -ne $Folder){throw 'Picker folder field readback differs.'}
    if([DcmNative]::Text([DcmNative]::Control($dialog,$app.Id,1),$app.Id) -ne 'Select Folder'){throw 'Unexpected picker confirmation.'}
    Write-Json (Join-Path $lab ($Name+'-picker.controls.json')) ([DcmNative]::Snapshot($dialog,$app.Id))
    [DcmNative]::ClickAsync($dialog,$app.Id,1)
    Wait-For 'picker accepted' { @([DcmNative]::Dialogs($hwnd,$app.Id)).Count -eq 0 }
    $cases.Add([ordered]@{name=$Name;kind='actual folder picker selection';passed=$true;folder=$Folder;field_id=1152;confirm_id=1})
}
function Verify-Vision([string]$Folder,[string]$Name){
    Require-App
    $vision=$env:DCM_FLOWUIVISION_EXE
    if(-not $vision){$vision='C:\FlowverseReports\dcm-flowuivision-20261103\FlowUIVision.exe'}
    $receipt['flowuivision']=[ordered]@{path=$vision;sha256=(Hash $vision)}
    $fvs=Join-Path $lab ($Name+'.fvs')
    $body=@"
target sut type=desktop app=dcm-quicksort
surface main target=sut process="$([IO.Path]::GetFileName($exe))" title~="1.0.71"
control source surface=main id=1001 role=textbox
control sort surface=main id=1045 role=combobox
control destination surface=main id=1003 role=textbox
wait main timeout=5s
select sort "Name"
type destination "Accepted"
inspect main
expect source text="$Folder"
expect destination text="Accepted"
"@
    [IO.File]::WriteAllText($fvs,$body,$utf8)
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$vision;$psi.Arguments='run-script "'+$fvs+'"';$psi.WorkingDirectory=$lab;$psi.UseShellExecute=$false;$psi.CreateNoWindow=$true;$psi.RedirectStandardOutput=$true;$psi.RedirectStandardError=$true;$psi.EnvironmentVariables['FLOWVERSE_REPO']=$lab
    $job=New-Object Diagnostics.Process;$job.StartInfo=$psi
    try{
        if(-not $job.Start()){throw 'FlowUIVision start failed.'}
        $out=$job.StandardOutput.ReadToEndAsync();$err=$job.StandardError.ReadToEndAsync()
        if(-not $job.WaitForExit(60000)){& "$env:SystemRoot\System32\taskkill.exe" /PID $job.Id /T /F | Out-Null;$null=$job.WaitForExit(10000);throw 'Owned FlowUIVision deadline.'}
        $raw=$out.GetAwaiter().GetResult();$errors=$err.GetAwaiter().GetResult()
        [IO.File]::WriteAllText((Join-Path $lab ($Name+'.vision.json')),$raw,$utf8);[IO.File]::WriteAllText((Join-Path $lab ($Name+'.stderr.log')),$errors,$utf8)
        $r=$raw | ConvertFrom-Json
        if($job.ExitCode -ne 0 -or -not $r.Passed -or @($r.Steps).Count -eq 0){throw 'Folder source expectation failed in FlowUIVision.'}
        Require-App;$cases.Add([ordered]@{name=$Name;kind='FlowUIVision source-path expectation';passed=$true;steps=@($r.Steps).Count})
    }finally{$job.Dispose()}
}
try{
    $b=Get-Content -LiteralPath $BuildReceipt -Raw -Encoding UTF8 | ConvertFrom-Json
    $head=Git-Value @('rev-parse','HEAD')
    if(-not $b.passed -or -not $b.source_unchanged -or $b.mode -notin @('build','package') -or $head -ne $b.commit){throw 'Build identity mismatch.'}
    if((Git-Value @('symbolic-ref','--short','HEAD')) -ne 'release/v1.0.71' -or (Git-Value @('remote','get-url','origin')) -ne 'https://github.com/laxmanxp/dcm-quick-sort.git'){throw 'Unexpected branch/origin.'}
    if((Hash $b.source_hash_manifest.path) -ne $b.source_hash_manifest.sha256 -or (Hash $b.app_artifact.path) -ne $b.app_artifact.sha256){throw 'Build hashes changed.'}
    $manifest=Get-Content -LiteralPath $b.source_hash_manifest.path -Raw -Encoding UTF8 | ConvertFrom-Json
    Check-Source
    $receipt['commit']=$head;$receipt['build_receipt']=[ordered]@{path=$BuildReceipt;sha256=(Hash $BuildReceipt)};$receipt['app_sha256']=$b.app_artifact.sha256
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    $inbox=Join-Path $lab 'Folder A';$target=Join-Path $lab 'Folder B With Spaces';$empty=Join-Path $lab 'Empty Folder';$unavailable=Join-Path $lab 'Temporarily Moved A';$local=Join-Path $lab 'local';$roaming=Join-Path $lab 'roaming';$temp=Join-Path $lab 'temp';$backup=Join-Path $lab 'backups';$config=Join-Path $local 'DCM Quick Sort';$vault=Join-Path $lab 'Vault';$destination=Join-Path $vault 'Accepted'
    $null=New-Item -ItemType Directory -Path $inbox,$target,$empty,$config,$roaming,$temp,$backup,$destination
    $roots=@($inbox,$target,$empty,$unavailable,$destination)
    foreach($pair in @(@($inbox,'01-alpha.txt'),@($inbox,'02-shared.txt'),@($inbox,'03-to-remove.txt'),@($target,'02-shared.txt'),@($target,'04-bravo.txt'))){
        $p=Join-Path $pair[0] $pair[1];$text='DCM-25 fixture: '+[IO.Path]::GetFileName($pair[0])+' / '+$pair[1]+' | Folder identity must match the preview and action target.'
        [IO.File]::WriteAllText($p,$text,$utf8);$expected[$p]=Hash $p;$texts[$p]=$text
    }
    Write-Json (Join-Path $config 'settings.json') ([ordered]@{default_source=$inbox;vault_root=$vault;pdf_default_view='fit_page'})
    $exe=Join-Path $lab ('DCM-QS-Folder-'+[Guid]::NewGuid().ToString('N')+'.exe')
    Copy-Item -LiteralPath $b.app_artifact.path -Destination $exe
    if((Hash $exe) -ne $b.app_artifact.sha256){throw 'App copy changed.'}
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe;$psi.Arguments='"'+$inbox+'"';$psi.WorkingDirectory=$lab;$psi.UseShellExecute=$false
    $envs=@{LOCALAPPDATA=$local;APPDATA=$roaming;TEMP=$temp;TMP=$temp;DCM_QUICKSORT_BACKUP_DIR=$backup}
    foreach($k in $envs.Keys){$psi.EnvironmentVariables[$k]=$envs[$k]}
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi
    if(-not $app.Start()){throw 'App start failed.'}
    $owned=[ordered]@{pid=$app.Id;exe=$exe;start=$app.StartTime.ToUniversalTime().ToString('o');sha256=(Hash $exe)};$receipt['owned_process']=$owned
    $until=[DateTime]::UtcNow.AddSeconds(20)
    do{$app.Refresh();if($app.HasExited){throw 'App exited during startup.'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    Require-App
    Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class DcmFolderEdit {
    [DllImport("user32.dll", EntryPoint="SendMessageTimeoutW", CharSet=CharSet.Unicode)]
    private static extern IntPtr Send(IntPtr h,uint m,IntPtr w,string text,uint flags,uint ms,out IntPtr result);
    public static void Set(IntPtr h,string text){IntPtr r;if(Send(h,0x000C,IntPtr.Zero,text,2,1500,out r)==IntPtr.Zero || r==IntPtr.Zero)throw new Exception("Owned text input failed");}
}
'@
    Verify-Vision $inbox 'DCM-25-setup'
    $aNames=@('01-alpha.txt','02-shared.txt','03-to-remove.txt')
    Select-File '02-shared.txt'
    Assert-View 'DCM-25-start-A-shared' $inbox $aNames '02-shared.txt'
    Set-Draft 'stale-draft-must-not-survive'
    Reload
    Assert-View 'DCM-25-reload-resets-draft' $inbox $aNames '02-shared.txt' -Capture
    $null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@('01-alpha.txt','02-shared.txt'))
    Reload
    Assert-View 'DCM-25-reload-multiselection-safe' $inbox $aNames '01-alpha.txt'
    Select-File '02-shared.txt'
    # Controlled external changes, strictly inside this disposable lab.
    $added=Join-Path $inbox '00-added.txt';$text='DCM-25 fixture: Folder A / newly added file'
    [IO.File]::WriteAllText($added,$text,$utf8);$expected[$added]=Hash $added;$texts[$added]=$text
    $mutationLog.Add([ordered]@{actor='test fixture setup';operation='create';path=$added})
    $aNames=@('00-added.txt','01-alpha.txt','02-shared.txt','03-to-remove.txt')
    Reload;Assert-View 'DCM-25-reload-sees-added-file' $inbox $aNames '02-shared.txt'
    Select-File '03-to-remove.txt';Set-Draft 'removed-target-draft'
    $removed=Join-Path $inbox '03-to-remove.txt';Remove-Item -LiteralPath $removed
    $expected.Remove($removed);$texts.Remove($removed)
    $mutationLog.Add([ordered]@{actor='test fixture setup';operation='remove';path=$removed})
    $aNames=@('00-added.txt','01-alpha.txt','02-shared.txt')
    Reload;Assert-View 'DCM-25-reload-drops-removed-target' $inbox $aNames '00-added.txt' -Capture
    Select-File '02-shared.txt';Set-Draft 'A-draft-must-not-leak-to-B'
    $d=Open-Picker;Cancel-Picker $d
    if((Text 1005) -cne 'A-draft-must-not-leak-to-B'){throw 'Cancelling Browse unexpectedly discarded the draft.'}
    Check-SourceView $inbox $aNames
    Select-PickerPath $target 'DCM-25-switch-to-B'
    $bNames=@('02-shared.txt','04-bravo.txt')
    Assert-View 'DCM-25-same-name-different-folder' $target $bNames '02-shared.txt' -Capture
    Set-Draft 'B-draft-must-not-leak-to-empty'
    Select-PickerPath $empty 'DCM-25-switch-to-empty'
    Assert-View 'DCM-25-empty-folder-clears-targets' $empty @() '' -Capture
    foreach($button in @(1006,1041,1007)){
        [DcmNative]::ClickAsync($hwnd,$app.Id,$button);Start-Sleep -Milliseconds 200
        Assert-View ('DCM-25-empty-action-'+$button) $empty @() ''
    }
    Select-PickerPath $target 'DCM-25-return-B'
    Assert-View 'DCM-25-return-B-restores-preview' $target $bNames '02-shared.txt'
    # Representative operations verify full paths, not merely matching basenames.
    Set-Draft 'B-renamed';$old=Join-Path $target '02-shared.txt';$renamed=Join-Path $target 'B-renamed.txt'
    $script:operationsStarted=$true
    [DcmNative]::ClickAsync($hwnd,$app.Id,1006)
    Wait-For 'rename B only' { (Test-Path -LiteralPath $renamed) -and -not(Test-Path -LiteralPath $old) }
    Map-Fixture $old $renamed
    Wait-For 'renamed row ready' { @([DcmNative]::ListItems($hwnd,$app.Id,1016)) -contains 'B-renamed.txt' }
    Reload
    $bNames=@('04-bravo.txt','B-renamed.txt')
    Assert-View 'DCM-25-rename-target-is-B-not-A' $target $bNames 'B-renamed.txt' -Capture
    $moved=Join-Path $destination 'B-renamed.txt'
    [DcmNative]::ClickAsync($hwnd,$app.Id,1041)
    Wait-For 'move B only' { (Test-Path -LiteralPath $moved) -and -not(Test-Path -LiteralPath $renamed) -and @([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq 1 }
    Map-Fixture $renamed $moved
    Assert-View 'DCM-25-move-target-is-B-not-A' $target @('04-bravo.txt') '04-bravo.txt' -Capture
    Select-PickerPath $inbox 'DCM-25-return-A'
    Select-File '02-shared.txt'
    Assert-View 'DCM-25-original-A-shared-unchanged' $inbox $aNames '02-shared.txt' -Capture
    # Simulate an unavailable source without deleting any of its remaining bytes.
    Move-Item -LiteralPath $inbox -Destination $unavailable
    foreach($old in @($expected.Keys | Where-Object {[IO.Path]::GetDirectoryName($_) -eq $inbox})){Map-Fixture $old (Join-Path $unavailable ([IO.Path]::GetFileName($old)))}
    $mutationLog.Add([ordered]@{actor='test fixture setup';operation='temporarily rename source directory';from=$inbox;to=$unavailable})
    Reload
    if(@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -ne 0 -or (Text 1005) -ne ''){throw 'DCM25_UNAVAILABLE_SOURCE_STALE_TARGET: failed Refresh retained old list or filename draft.'}
    Assert-View 'DCM-25-unavailable-source-clears-targets' $inbox @() '' 'Source unavailable' -Capture
    if(-not (Visible-Text 'Cannot read source folder:')){throw 'Missing source error was not shown.'}
    foreach($button in @(1006,1041)){
        [DcmNative]::ClickAsync($hwnd,$app.Id,$button);Start-Sleep -Milliseconds 200
        if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unavailable source retained an actionable target.'}
        Check-Files
    }
    Move-Item -LiteralPath $unavailable -Destination $inbox
    foreach($old in @($expected.Keys | Where-Object {[IO.Path]::GetDirectoryName($_) -eq $unavailable})){Map-Fixture $old (Join-Path $inbox ([IO.Path]::GetFileName($old)))}
    $mutationLog.Add([ordered]@{actor='test fixture setup';operation='restore source directory';from=$unavailable;to=$inbox})
    Reload;Assert-View 'DCM-25-source-recovery' $inbox $aNames '00-added.txt' -Capture
    $receipt['controlled_fixture_mutations']=$mutationLog
    $receipt['representative_operations']=[ordered]@{renamed_source=(Join-Path $target '02-shared.txt');renamed_in_folder_B='B-renamed.txt';moved_to=$moved;original_A_shared_unchanged=$true}
    $receipt['fixture_sha256']=$expected
    $receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage
    Write-Output ('FAIL='+$_.Exception.Message)
    if($null -ne $app){try{Shot 'DCM-25-failure' $hwnd;Check-Files;$receipt['fixtures_match_expected_after_failure']=$true}catch{$receipt['failure_capture_error']=$_.Exception.Message}}
}finally{
    if($null -ne $app){
        try{
            Require-App
            foreach($d in @([DcmNative]::Dialogs($hwnd,$app.Id))){Cancel-Picker ([IntPtr]$d.Handle)}
            $children=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$app.Id))
            $owned['close_requested_utc']=[DateTime]::UtcNow.ToString('o');[DcmNative]::CloseAsync($hwnd,$app.Id)
            if(-not $app.WaitForExit(15000)){Require-App;& "$env:SystemRoot\System32\taskkill.exe" /PID $app.Id /T /F | Out-Null;$owned['forced_stop']=$true;throw 'Owned app required forced close.'}
            $owned['exit_code']=$app.ExitCode;if($app.ExitCode -ne 0){throw 'Nonzero app close.'}
            $late=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$app.Id) | Where-Object {$_.CreationDate.ToUniversalTime() -ge [DateTime]::Parse($owned.start).ToUniversalTime() -and $_.CreationDate.ToUniversalTime() -le $app.ExitTime.ToUniversalTime()})
            foreach($child in @(@($children)+@($late))){$live=Get-CimInstance Win32_Process -Filter ('ProcessId='+$child.ProcessId);if($live -and $live.ExecutablePath -eq $child.ExecutablePath -and $live.CreationDate -eq $child.CreationDate){$p=Get-Process -Id $child.ProcessId -ErrorAction SilentlyContinue;if($p -and -not $p.WaitForExit(10000)){throw 'Owned child did not drain.'}}}
            $owned['children_drained']=$true;$app.Dispose()
        }catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}
    }
    try{if($head){Check-Source;$receipt['source_unchanged']=$true}}catch{$receipt.passed=$false;$receipt['integrity_error']=$_.Exception.Message}
    $receipt['controlled_fixture_mutations']=$mutationLog
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Write-Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
