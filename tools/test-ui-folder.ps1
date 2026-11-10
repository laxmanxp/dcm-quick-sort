[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT,[switch]$Probe)
# DCM-18: actual owned folder picker, isolated paths, no installed application.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt){throw 'Workspace and exact build receipt required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-folder-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Lab must be outside checkout.'}
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-18';app='dcm-quicksort';probe_only=[bool]$Probe;report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;cases=$cases;captures=$captures}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$head='';$manifest=@();$expected=@{}
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
function Wait-For([string]$Name,[scriptblock]$Check){$until=[DateTime]::UtcNow.AddSeconds(15);do{Require-App;if(& $Check){return};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until);throw ('Deadline: '+$Name)}
function Text([int]$Id){return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,$Id),$app.Id)}
function Check-Files {
    foreach($dir in @($inbox,$target)){
        $files=@(Get-ChildItem -LiteralPath $dir -File)
        $want=@($expected.Keys | Where-Object {[IO.Path]::GetDirectoryName($_) -eq $dir})
        if($files.Count -ne $want.Count){throw 'Fixture file set changed.'}
        foreach($f in $files){if(-not $expected.ContainsKey($f.FullName) -or (Hash $f.FullName) -ne $expected[$f.FullName]){throw 'Fixture name or bytes changed.'}}
    }
    if(@(Get-ChildItem -LiteralPath $destination -Force).Count){throw 'Browse changed destination contents.'}
    $journal=Join-Path $config 'recovery\move-journal.jsonl'
    if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Browse appended a file-operation journal.'}
}
function Shot([string]$Name,[IntPtr]$Window){
    Require-App
    $screen=[Windows.Forms.Screen]::FromHandle($Window).WorkingArea
    [DcmNative]::Restore($Window,$app.Id,$screen.Left+12,$screen.Top+12)
    Start-Sleep -Milliseconds 350
    Write-Json (Join-Path $lab ($Name+'.controls.json')) ([DcmNative]::Snapshot($Window,$app.Id))
    $s=[DcmNative]::Capture($Window,$app.Id,(Join-Path $lab ($Name+'.png')),$false)
    $captures.Add([ordered]@{feature='DCM-18';name=$Name;path=$s.Path;sha256=(Hash $s.Path);width=$s.Width;height=$s.Height;visual_reviewed=$false})
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
    $inbox=Join-Path $lab 'Initial Inbox';$target=Join-Path $lab 'Folder With Spaces';$local=Join-Path $lab 'local';$roaming=Join-Path $lab 'roaming';$temp=Join-Path $lab 'temp';$backup=Join-Path $lab 'backups';$config=Join-Path $local 'DCM Quick Sort';$vault=Join-Path $lab 'Vault';$destination=Join-Path $vault 'Accepted'
    $null=New-Item -ItemType Directory -Path $inbox,$target,$config,$roaming,$temp,$backup,$destination
    foreach($pair in @(@($inbox,'00-initial.txt'),@($target,'01-target alpha.txt'),@($target,'02-target bravo.txt'))){$p=Join-Path $pair[0] $pair[1];[IO.File]::WriteAllText($p,('DCM-18 disposable fixture: '+$pair[1]),$utf8);$expected[$p]=Hash $p}
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
    Check-SourceView $inbox @('00-initial.txt')
    $dialog=Open-Picker
    Shot 'DCM-18-picker-open' $dialog
    Cancel-Picker $dialog
    Check-SourceView $inbox @('00-initial.txt')
    Shot 'DCM-18-cancel-preserves-source' $hwnd
    if(-not $Probe){throw 'Folder-selection acceptance awaits observed picker controls; probe only.'}
    $receipt['fixture_sha256']=$expected
    $receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage
    Write-Output ('FAIL='+$_.Exception.Message)
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
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Write-Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
