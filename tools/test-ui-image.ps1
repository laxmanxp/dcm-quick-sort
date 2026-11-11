[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT)
# DCM-27: source-bound native image-preview acceptance; no installed app or user files.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt){throw 'Workspace and exact build receipt required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-image-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Lab must be outside checkout.'}
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-27';app='dcm-quicksort';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;cases=$cases;captures=$captures}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$head='';$manifest=@();$expected=@{};$lock=$null
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
    $files=@(Get-ChildItem -LiteralPath $inbox -File)
    if($files.Count -ne $expected.Count){throw 'Image fixture file set changed.'}
    foreach($f in $files){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Image preview changed source bytes.'}}
    if(@(Get-ChildItem -LiteralPath $destination -Force).Count){throw 'Image preview changed destination contents.'}
    $journal=Join-Path $config 'recovery\move-journal.jsonl'
    if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Image preview appended a file operation.'}
}
function Shot([string]$Name){
    Require-App
    $screen=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
    [DcmNative]::Restore($hwnd,$app.Id,$screen.Left+12,$screen.Top+12)
    Start-Sleep -Milliseconds 300
    Write-Json (Join-Path $lab ($Name+'.controls.json')) ([DcmNative]::Snapshot($hwnd,$app.Id))
    $shot=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($Name+'.png')),$false)
    $captures.Add([ordered]@{feature='DCM-27';name=$Name;path=$shot.Path;sha256=(Hash $shot.Path);width=$shot.Width;height=$shot.Height;visual_reviewed=$false})
    return $shot
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
function Select-Image([string]$File){
    Require-App
    $null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@($File))
    $stem=[IO.Path]::GetFileNameWithoutExtension($File)
    Wait-For ('image selection '+$File) { (Text 1005) -ceq $stem }
}
function Assert-Image([string]$Case,$Spec,[string]$Mode='fit',[int]$Percent=100,[int]$PanX=0,[int]$PanY=0){
    Require-App
    $items=@([DcmNative]::ListItems($hwnd,$app.Id,1016));$selected=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016))
    if($selected.Count -ne 1 -or $items[$selected[0]] -cne $Spec.name -or (Text 1005) -cne [IO.Path]::GetFileNameWithoutExtension($Spec.name)){throw 'Image target/draft mismatch.'}
    $snapshot=@([DcmNative]::Snapshot($hwnd,$app.Id))
    if(-not @($snapshot | Where-Object {$_.Visible -and $_.Text.StartsWith('PREVIEW') -and $_.Text.EndsWith($Spec.name)}).Count){throw 'Image heading mismatch.'}
    $statusPrefix=if($Mode -eq 'manual'){'Image preview zoom: '+$Percent+'%'}elseif($Mode -eq 'width'){'Image preview: Fit Width'}else{'Image preview: Fit'}
    if(-not @($snapshot | Where-Object {$_.Visible -and $_.Text.StartsWith($statusPrefix)}).Count){throw 'Expected owned image renderer status.'}
    foreach($id in @(1020,1021,1022,1023,1047)){if(-not @($snapshot | Where-Object {$_.ID -eq $id -and $_.Enabled}).Count){throw 'Image sizing control disabled.'}}
    $shot=Shot $Case
    $preview=[DcmNative]::ChildByClass($hwnd,$app.Id,'QuickSortPreview')
    $view=[DcmImageProbe]::Geometry($preview,$app.Id)
    $pixels=[DcmImageProbe]::Verify($shot.Path,$shot.Left,$shot.Top,$view,$Spec.w,$Spec.h,$Spec.variant,$Spec.alpha,$Mode,$Percent,$PanX,$PanY)
    Check-Files
    $cases.Add([ordered]@{name=$Case;passed=$true;file=$Spec.name;mode=$Mode;zoom=$Percent;pixel_evidence=$pixels})
}
function Size-Action([int]$ID){Require-App;[DcmNative]::ClickAsync($hwnd,$app.Id,$ID);Start-Sleep -Milliseconds 200}
function Assert-ImageError([string]$Case,[string]$File){
    Require-App
    $snapshot=@([DcmNative]::Snapshot($hwnd,$app.Id))
    $errors=@($snapshot | Where-Object {$_.Visible -and $_.Class -eq 'Static' -and $_.Text.StartsWith('Image preview unavailable:')})
    if($errors.Count -ne 1){throw ('DCM27_TRUTHFUL_ERROR_MISSING: '+$Case)}
    foreach($id in @(1020,1021,1022,1023,1047)){if(@($snapshot | Where-Object {$_.ID -eq $id -and $_.Enabled}).Count){throw 'Failed image retained enabled sizing controls.'}}
    $null=Shot $Case
    $cases.Add([ordered]@{name=$Case;file=$File;passed=$true;status=$errors[0].Text})
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
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-image-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    $vision=$env:DCM_FLOWUIVISION_EXE
    if(-not $vision){$vision='C:\FlowverseReports\dcm-flowuivision-20261103\FlowUIVision.exe'}
    $receipt['flowuivision']=[ordered]@{path=$vision;sha256=(Hash $vision)}
    $inbox=Join-Path $lab 'Image Inbox'
    $vault=Join-Path $lab 'Vault';$destination=Join-Path $vault 'Accepted'
    $local=Join-Path $lab 'profile\local';$roaming=Join-Path $lab 'profile\roaming'
    $config=Join-Path $local 'DCM Quick Sort';$temp=Join-Path $lab 'temp'
    $null=New-Item -ItemType Directory -Path $inbox,$destination,$config,$roaming,$temp
    Write-Json (Join-Path $config 'settings.json') ([ordered]@{default_source=$inbox;pdf_default_view='fit_width';vault_root=$vault})
    [IO.File]::WriteAllText((Join-Path $config 'destinations.conf'),"Acceptance=Accepted`r`n",$utf8)
    $spec=@(
        @{name='01-landscape.png';w=360;h=240;variant=0;alpha=$false},
        @{name='02-portrait.jpg';w=240;h=360;variant=1;alpha=$false},
        @{name='03-square.jpeg';w=300;h=300;variant=2;alpha=$false},
        @{name='04-landscape.bmp';w=400;h=200;variant=3;alpha=$false},
        @{name='05-static.gif';w=320;h=240;variant=1;alpha=$false},
        @{name='06-portrait.tif';w=240;h=320;variant=2;alpha=$false},
        @{name='07-square.tiff';w=300;h=300;variant=3;alpha=$false},
        @{name='08-alpha.PNG';w=320;h=240;variant=0;alpha=$true},
        @{name='09-large.png';w=1200;h=900;variant=2;alpha=$false},
        @{name='10-next.png';w=240;h=320;variant=3;alpha=$false}
    )
    foreach($f in $spec){[DcmImageProbe]::Create((Join-Path $inbox $f.name),$f.w,$f.h,$f.variant,$f.alpha)}
    [IO.File]::WriteAllBytes((Join-Path $inbox '11-empty.png'),[byte[]]@())
    [IO.File]::WriteAllBytes((Join-Path $inbox '12-corrupt.jpg'),[byte[]]@(255,216,255,0,1,2,3,4))
    foreach($f in @(Get-ChildItem -LiteralPath $inbox -File)){$expected[$f.Name]=Hash $f.FullName}
    Write-Json (Join-Path $lab 'fixture-manifest.json') ([ordered]@{images=$spec;hashes=$expected})
    $exe=Join-Path $lab ('DCM-QS-Image-'+[Guid]::NewGuid().ToString('N')+'.exe')
    Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    if((Hash $exe) -ne $build.app_artifact.sha256){throw 'Owned app copy hash mismatch.'}
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $inbox '01-landscape.png')+'"';$psi.WorkingDirectory=$lab;$psi.UseShellExecute=$false
    $psi.EnvironmentVariables['LOCALAPPDATA']=$local;$psi.EnvironmentVariables['APPDATA']=$roaming
    $psi.EnvironmentVariables['TEMP']=$temp;$psi.EnvironmentVariables['TMP']=$temp;$psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'configuration-backups'
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi
    if(-not $app.Start()){throw 'Candidate failed to start.'}
    $owned=[ordered]@{pid=$app.Id;exe=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')}
    $receipt['owned_process']=$owned
    $until=[DateTime]::UtcNow.AddSeconds(20)
    do{$app.Refresh();if($app.HasExited){throw 'Candidate exited at startup.'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    if($hwnd -eq [IntPtr]::Zero){throw 'Main window deadline.'}
    Wait-For 'image fixtures ready' { @([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $expected.Count }
    Invoke-Sort 'DCM-27-setup' 'Name'
    foreach($f in $spec){Select-Image $f.name;Assert-Image ('DCM-27-fit-'+$f.name) $f}
    Select-Image $spec[1].name;Size-Action 1047;Assert-Image 'DCM-27-portrait-fit-width-top' $spec[1] 'width'
    Size-Action 1021;Assert-Image 'DCM-27-100-percent' $spec[1] 'manual' 100
    Size-Action 1022;Assert-Image 'DCM-27-110-percent' $spec[1] 'manual' 110
    Size-Action 1020;Assert-Image 'DCM-27-minus-back-to-100' $spec[1] 'manual' 100
    Size-Action 1023;Assert-Image 'DCM-27-fit-restored' $spec[1]
    Select-Image $spec[8].name;Size-Action 1021;Assert-Image 'DCM-27-large-100-centered' $spec[8] 'manual' 100
    $preview=[DcmNative]::ChildByClass($hwnd,$app.Id,'QuickSortPreview')
    [DcmImageProbe]::Drag($preview,$app.Id,80,60);Assert-Image 'DCM-27-large-pan' $spec[8] 'manual' 100 80 60
    [DcmNative]::ClickAsync($hwnd,$app.Id,1007)
    Wait-For 'next image selected' { (Text 1005) -ceq '10-next' }
    Assert-Image 'DCM-27-next-resets-zoom-pan' $spec[9]
    # Stream-backed preview must not keep a source-file handle open.
    $exclusive=[IO.File]::Open((Join-Path $inbox $spec[9].name),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None)
    $exclusive.Dispose();$cases.Add([ordered]@{name='DCM-27-preview-releases-source-handle';passed=$true})
    foreach($f in @('11-empty.png','12-corrupt.jpg')){Select-Image $f;Assert-ImageError ('DCM-27-'+$f) $f;Check-Files}
    $lock=[IO.File]::Open((Join-Path $inbox $spec[0].name),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None)
    try{Select-Image $spec[0].name;Assert-ImageError 'DCM-27-sharing-violation' $spec[0].name}finally{$lock.Dispose();$lock=$null}
    Select-Image $spec[0].name;Assert-Image 'DCM-27-read-recovery' $spec[0]
    $receipt['fixture_hashes']=$expected;$receipt['fixture_specs']=$spec
    Check-Files;Stop-App;Check-Source
    if((Hash $exe) -ne $build.app_artifact.sha256 -or (Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'App bytes changed during qualification.'}
    $receipt['fixtures_unchanged']=$true;$receipt['source_unchanged']=$true;$receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage
    Write-Output ('FAIL='+$_.Exception.Message)
    if($null -ne $app){try{$null=Shot 'DCM-27-failure'}catch{$receipt['capture_error']=$_.Exception.Message}}
}finally{
    if($null -ne $lock){$lock.Dispose();$lock=$null}
    try{Check-Files;$receipt['fixtures_unchanged']=$true}catch{$receipt.passed=$false;$receipt['fixture_error']=$_.Exception.Message}
    try{Stop-App}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    Write-Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
