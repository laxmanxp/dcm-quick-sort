[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT)
# DCM-22: source-bound native sorting acceptance; no installed app or user files.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt){throw 'Workspace and exact build receipt required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-sort-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Lab must be outside checkout.'}
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-22';app='dcm-quicksort';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;cases=$cases;captures=$captures}
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
function Wait-For([string]$Name,[scriptblock]$Check){
    $until=[DateTime]::UtcNow.AddSeconds(15)
    do{
        Require-App
        $dialogs=@([DcmNative]::Dialogs($hwnd,$app.Id))
        if($dialogs.Count){
            Write-Json (Join-Path $lab 'unexpected-dialogs.json') @($dialogs | ForEach-Object {[ordered]@{window=$_;controls=@([DcmNative]::Snapshot([IntPtr]$_.Handle,$app.Id))}})
            throw ('Unexpected dialog during '+$Name)
        }
        if(& $Check){return};Start-Sleep -Milliseconds 100
    }while([DateTime]::UtcNow -lt $until)
    throw ('Deadline: '+$Name)
}
function Text([int]$Id){return [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,$Id),$app.Id)}
function Check-Files {
    $files=@(Get-ChildItem -LiteralPath $inbox -Force)
    if($files.Count -ne $expected.Count){throw 'Fixture file set changed.'}
    foreach($f in $files){
        if($f.PSIsContainer -or -not $expected.ContainsKey($f.Name)){throw 'Unexpected fixture path.'}
        $want=$expected[$f.Name]
        if((Hash $f.FullName) -ne $want.sha256 -or $f.CreationTimeUtc.Ticks -ne $want.creation_ticks -or $f.LastWriteTimeUtc.Ticks -ne $want.modified_ticks){throw ('Fixture contents/timestamps changed: '+$f.Name)}
    }
    if(@(Get-ChildItem -LiteralPath $destination -Force).Count){throw 'Sorting changed destination contents.'}
    $journal=Join-Path $config 'recovery\move-journal.jsonl'
    if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Sorting appended a file-operation journal.'}
}
function Shot([string]$Name){
    Require-App
    $screen=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
    [DcmNative]::Restore($hwnd,$app.Id,$screen.Left+12,$screen.Top+12)
    Start-Sleep -Milliseconds 300
    Write-Json (Join-Path $lab ($Name+'.controls.json')) ([DcmNative]::Snapshot($hwnd,$app.Id))
    $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($Name+'.png')),$false)
    $captures.Add([ordered]@{feature='DCM-22';name=$Name;path=$s.Path;sha256=(Hash $s.Path);width=$s.Width;height=$s.Height;visual_reviewed=$false})
}
function Invoke-Sort([string]$Name,[string]$Mode){
    Require-App
    $fvs=Join-Path $lab ($Name+'.fvs')
    $body=@"
target sut type=desktop app=dcm-quicksort
surface main target=sut process="$([IO.Path]::GetFileName($exe))" title~="1.0.71"
control sort surface=main id=1045 role=combobox
wait main timeout=5s
inspect main
select sort "$Mode"
inspect main
expect sort text="$Mode"
"@
    [IO.File]::WriteAllText($fvs,$body,$utf8)
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$vision;$psi.Arguments='run-script "'+$fvs+'"';$psi.WorkingDirectory=$lab
    $psi.UseShellExecute=$false;$psi.CreateNoWindow=$true;$psi.RedirectStandardOutput=$true;$psi.RedirectStandardError=$true
    $psi.EnvironmentVariables['FLOWVERSE_REPO']=$lab
    $job=New-Object Diagnostics.Process;$job.StartInfo=$psi
    try{
        if(-not $job.Start()){throw 'Could not launch FlowUIVision.'}
        $outTask=$job.StandardOutput.ReadToEndAsync();$errTask=$job.StandardError.ReadToEndAsync()
        if(-not $job.WaitForExit(30000)){
            & "$env:SystemRoot\System32\taskkill.exe" /PID $job.Id /T /F | Out-Null
            $null=$job.WaitForExit(10000);throw ('FlowUIVision deadline: '+$Name)
        }
        $raw=$outTask.GetAwaiter().GetResult();$stderr=$errTask.GetAwaiter().GetResult()
        [IO.File]::WriteAllText((Join-Path $lab ($Name+'.vision.json')),$raw,$utf8)
        [IO.File]::WriteAllText((Join-Path $lab ($Name+'.stderr.log')),$stderr,$utf8)
        $result=$raw | ConvertFrom-Json
        if($job.ExitCode -ne 0 -or $result.Passed -ne $true -or @($result.Steps).Count -eq 0){throw ('FlowUIVision assertions failed: '+$Name)}
        Require-App
    }finally{$job.Dispose()}
}
function Assert-View([string]$Name,[string]$Mode,[string[]]$Names,[string]$Selected){
    Wait-For $Name { (Text 1045) -eq $Mode -and (@([DcmNative]::ListItems($hwnd,$app.Id,1016)) -join '|') -ceq ($Names -join '|') }
    $indices=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016))
    $position=[Array]::IndexOf($Names,$Selected)
    if($position -lt 0 -or $indices.Count -ne 1 -or $indices[0] -ne $position){throw ('Sort changed the selected file: '+$Name)}
    if((Text 1005) -cne [IO.Path]::GetFileNameWithoutExtension($Selected)){throw ('Filename draft mismatch: '+$Name)}
    $snapshot=@([DcmNative]::Snapshot($hwnd,$app.Id))
    if(-not @($snapshot | Where-Object {$_.Visible -and $_.Text.StartsWith('PREVIEW') -and $_.Text.EndsWith($Selected)}).Count){throw 'Preview heading mismatch.'}
    if(-not @($snapshot | Where-Object {$_.Visible -and $_.Class -eq 'Edit' -and $_.Text -ceq $expected[$Selected].text}).Count){throw 'Visible preview text does not match selected fixture.'}
    $counter=('{0} of {1}' -f ($position+1),$Names.Count)
    if(-not @($snapshot | Where-Object {$_.Visible -and $_.Text.StartsWith('FILES') -and $_.Text.EndsWith($counter)}).Count){throw 'Position counter mismatch.'}
    Check-Files
    $cases.Add([ordered]@{name=$Name;mode=$Mode;expected_order=$Names;actual_order=@([DcmNative]::ListItems($hwnd,$app.Id,1016));selected=$Selected;selected_index=$position;draft=(Text 1005);preview_text=$expected[$Selected].text;passed=$true})
    Shot $Name
}
function Stop-App {
    if($null -eq $app){return}
    Require-App
    $children=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$app.Id) | ForEach-Object {[ordered]@{pid=[int]$_.ProcessId;path=$_.ExecutablePath;created=$_.CreationDate.ToUniversalTime().ToString('o')}})
    [DcmNative]::CloseAsync($hwnd,$app.Id)
    if(-not $app.WaitForExit(15000)){
        Require-App
        & "$env:SystemRoot\System32\taskkill.exe" /PID $app.Id /T /F | Out-Null
        $null=$app.WaitForExit(10000);$owned['forced_stop']=$true
        throw 'Owned app did not close normally.'
    }
    $owned['exit_code']=$app.ExitCode;$owned['closed_utc']=[DateTime]::UtcNow.ToString('o')
    $late=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$app.Id) | Where-Object {$_.CreationDate.ToUniversalTime() -ge [DateTime]::Parse($owned.start).ToUniversalTime() -and $_.CreationDate.ToUniversalTime() -le $app.ExitTime.ToUniversalTime()} | ForEach-Object {[ordered]@{pid=[int]$_.ProcessId;path=$_.ExecutablePath;created=$_.CreationDate.ToUniversalTime().ToString('o')}})
    foreach($child in @(@($children)+@($late))){
        $live=Get-CimInstance Win32_Process -Filter ('ProcessId='+$child.pid)
        if($live -and $live.ExecutablePath -eq $child.path -and $live.CreationDate.ToUniversalTime().ToString('o') -eq $child.created){
            $drain=Get-Process -Id $child.pid -ErrorAction SilentlyContinue
            if($drain -and -not $drain.WaitForExit(10000)){throw 'Owned child did not drain.'}
        }
    }
    $owned['children_drained']=$true
    $code=$app.ExitCode;$app.Dispose();$script:app=$null
    if($code -ne 0){throw 'Owned app returned nonzero exit.'}
}
try{
    $build=Get-Content -LiteralPath $BuildReceipt -Raw -Encoding UTF8 | ConvertFrom-Json
    if(-not $build.passed -or $build.mode -ne 'build' -or -not $build.source_unchanged){throw 'Successful dedicated build receipt required.'}
    $head=Git-Value @('rev-parse','HEAD')
    if($head -ne $build.commit){throw 'Build/current source identity mismatch.'}
    if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256){throw 'Build manifest hash changed.'}
    $manifest=Get-Content -LiteralPath $build.source_hash_manifest.path -Raw -Encoding UTF8 | ConvertFrom-Json
    if(-not $manifest.Count){throw 'Empty source manifest.'}
    Check-Source
    if((Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'App hash changed.'}
    $receipt['commit']=$head;$receipt['build_receipt']=[ordered]@{path=$BuildReceipt;sha256=(Hash $BuildReceipt)}
    $receipt['app_sha256']=$build.app_artifact.sha256
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    $vision=$env:DCM_FLOWUIVISION_EXE
    if(-not $vision){$vision='C:\FlowverseReports\dcm-flowuivision-20261103\FlowUIVision.exe'}
    $receipt['flowuivision']=[ordered]@{path=$vision;sha256=(Hash $vision)}
    $inbox=Join-Path $lab 'Sort Inbox'
    $vault=Join-Path $lab 'Vault';$destination=Join-Path $vault 'Accepted'
    $local=Join-Path $lab 'profile\local';$roaming=Join-Path $lab 'profile\roaming'
    $config=Join-Path $local 'DCM Quick Sort';$temp=Join-Path $lab 'temp'
    $null=New-Item -ItemType Directory -Path $inbox,$destination,$config,$roaming,$temp
    Write-Json (Join-Path $config 'settings.json') ([ordered]@{default_source=$inbox;pdf_default_view='fit_width';vault_root=$vault})
    [IO.File]::WriteAllText((Join-Path $config 'destinations.conf'),"Acceptance=Accepted`r`n",$utf8)
    # Same-day ties and inverted modification times distinguish all four sort rules.
    $spec=@(
        @('alpha.txt','2026-10-20T10:00:00'),
        @('beta.log','2026-10-21T10:00:00'),
        @('gamma.log','2026-10-21T07:00:00'),
        @('middle.txt','2026-10-21T08:00:00'),
        @('omega.csv','2026-10-21T08:00:00'),
        @('zeta.txt','2026-10-20T11:00:00')
    )
    for($i=0;$i -lt $spec.Count;$i++){
        $name=$spec[$i][0];$path=Join-Path $inbox $name
        $text='DCM-22 sorting fixture: '+$name+' | Selection and preview must stay together.'
        [IO.File]::WriteAllText($path,$text,$utf8)
        $created=[DateTime]::SpecifyKind([DateTime]::ParseExact($spec[$i][1],'yyyy-MM-ddTHH:mm:ss',[Globalization.CultureInfo]::InvariantCulture),[DateTimeKind]::Local)
        $modified=[DateTime]::SpecifyKind([DateTime]'2026-09-01T12:00:00',[DateTimeKind]::Utc).AddHours($i)
        [IO.File]::SetCreationTimeUtc($path,$created.ToUniversalTime())
        [IO.File]::SetLastWriteTimeUtc($path,$modified)
        if([IO.File]::GetCreationTimeUtc($path).Ticks -ne $created.ToUniversalTime().Ticks -or [IO.File]::GetLastWriteTimeUtc($path).Ticks -ne $modified.Ticks){throw 'Controlled fixture timestamps were not applied.'}
        $expected[$name]=[ordered]@{text=$text;sha256=(Hash $path);creation_local=$created.ToString('o');creation_ticks=$created.ToUniversalTime().Ticks;modified_utc=$modified.ToString('o');modified_ticks=$modified.Ticks}
    }
    Write-Json (Join-Path $lab 'fixture-manifest.json') $expected
    $receipt['fixtures']=$expected
    $orders=@(
        [pscustomobject]@{mode='Name';names=[string[]]@('alpha.txt','beta.log','gamma.log','middle.txt','omega.csv','zeta.txt')},
        [pscustomobject]@{mode='File Type';names=[string[]]@('omega.csv','beta.log','gamma.log','alpha.txt','middle.txt','zeta.txt')},
        [pscustomobject]@{mode='Date Created';names=[string[]]@('beta.log','middle.txt','omega.csv','gamma.log','zeta.txt','alpha.txt')},
        [pscustomobject]@{mode='Date Created > Type > Name';names=[string[]]@('omega.csv','beta.log','gamma.log','middle.txt','alpha.txt','zeta.txt')}
    )
    $exe=Join-Path $lab ('DCM-QS-Sort-'+[Guid]::NewGuid().ToString('N')+'.exe')
    Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    if((Hash $exe) -ne $build.app_artifact.sha256){throw 'Owned app copy hash mismatch.'}
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $inbox 'middle.txt')+'"';$psi.WorkingDirectory=$lab;$psi.UseShellExecute=$false
    $psi.EnvironmentVariables['LOCALAPPDATA']=$local;$psi.EnvironmentVariables['APPDATA']=$roaming
    $psi.EnvironmentVariables['TEMP']=$temp;$psi.EnvironmentVariables['TMP']=$temp;$psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'configuration-backups'
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi
    if(-not $app.Start()){throw 'Candidate failed to start.'}
    $owned=[ordered]@{pid=$app.Id;exe=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')}
    $receipt['owned_process']=$owned
    $until=[DateTime]::UtcNow.AddSeconds(20)
    do{$app.Refresh();if($app.HasExited){throw 'Candidate exited at startup.'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    if($hwnd -eq [IntPtr]::Zero){throw 'Main window deadline.'}
    Assert-View 'DCM-22-default' 'Date Created' $orders[2].names 'middle.txt'
    for($i=0;$i -lt $orders.Count;$i++){
        $o=$orders[$i];$name='DCM-22-middle-sort-'+$i
        Invoke-Sort $name $o.mode
        Assert-View $name $o.mode $o.names 'middle.txt'
    }
    $null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@('alpha.txt'))
    Wait-For 'alpha selection settled' { (Text 1005) -eq 'alpha' }
    for($i=$orders.Count-1;$i -ge 0;$i--){
        $o=$orders[$i];$name='DCM-22-alpha-sort-'+$i
        Invoke-Sort $name $o.mode
        Assert-View $name $o.mode $o.names 'alpha.txt'
    }
    Check-Files;Stop-App;Check-Source
    if((Hash $exe) -ne $build.app_artifact.sha256 -or (Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'App bytes changed during qualification.'}
    $receipt['fixtures_unchanged']=$true;$receipt['source_unchanged']=$true;$receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage
    Write-Output ('FAIL='+$_.Exception.Message)
    if($null -ne $app){try{Shot 'DCM-22-failure'}catch{$receipt['capture_error']=$_.Exception.Message}}
}finally{
    try{Stop-App}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    Write-Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
