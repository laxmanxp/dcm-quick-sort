[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT)
# Source-bound native rotation acceptance. No installed app/user files are touched.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt){throw 'Exact FlowDev workspace and build receipt required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-rotation-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Lab must be outside checkout.'}
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-ROTATE-01';app='dcm-quicksort';suite='ui-rotation';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;cases=$cases;captures=$captures}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$head='';$manifest=@();$expected=@{};$fixture=$null;$script:capturePositioned=$false
Write-Output ('REPORT_ROOT='+$lab)
function Write-Json([string]$Path,$Value){[IO.File]::WriteAllText($Path,($Value|ConvertTo-Json -Depth 30),$utf8)}
function Hash([string]$Path){return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()}
function Git-Value([string[]]$Arguments){$s=(& git.exe -C $repo @Arguments|Out-String).Trim();if($LASTEXITCODE -ne 0){throw 'Git query failed.'};return $s}
function Check-Source {
    if((Git-Value @('rev-parse','HEAD')) -ne $head -or (Git-Value @('status','--porcelain=v1','--untracked-files=all'))){throw 'Checkout/source identity changed.'}
    foreach($f in $manifest){$p=[IO.Path]::GetFullPath((Join-Path $repo $f.path));if(-not $p.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $p) -ne $f.sha256){throw ('Source manifest changed: '+$f.path)}}
}
function Check-Files {
    if($null -eq $fixture){return}
    $files=@(Get-ChildItem -LiteralPath $fixture.inbox -File)
    if($files.Count -ne $expected.Count){throw 'Fixture file set changed.'}
    foreach($f in $files){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw ('Fixture source bytes changed: '+$f.Name)}}
    if(@(Get-ChildItem -LiteralPath $fixture.destination -Force).Count){throw 'Destination changed during preview.'}
    $journal=Join-Path $fixture.config 'recovery\move-journal.jsonl'
    if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Preview appended file transaction.'}
}
function Require-App {
    if($null -eq $app){throw 'Owned app absent.'}
    $app.Refresh()
    if($app.HasExited -or $app.Path -ne $owned.exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $owned.start){throw 'Owned app PID/path/start identity changed.'}
    [DcmNative]::RequireOwned($hwnd,$app.Id)
}
function Wait-For([string]$Name,[scriptblock]$Predicate,[int]$Seconds=18){
    $end=[DateTime]::UtcNow.AddSeconds($Seconds)
    do{
        Require-App
        if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw ('Unexpected owned dialog while waiting for '+$Name)}
        if(& $Predicate){return}
        Start-Sleep -Milliseconds 120
    }while([DateTime]::UtcNow -lt $end)
    throw ('Timeout: '+$Name)
}
function Snap {return @([DcmNative]::Snapshot($hwnd,$app.Id))}
function Status-Is([string]$Needle){return @(Snap|Where-Object {$_.Visible -and $_.Text.Contains($Needle)}).Count -gt 0}
function Assert-Toolbar([bool]$Enabled,[string]$Filename){
    Require-App
    foreach($id in @(1049,1050)){
        $ctrl=[DcmNative]::Control($hwnd,$app.Id,$id)
        if(-not [DcmNative]::IsWindowVisible($ctrl)){throw ('Rotation control disappeared: '+$id)}
        if([DcmNative]::IsWindowEnabled($ctrl) -ne $Enabled){throw ('Wrong rotation enable state: '+$id)}
    }
    $state=Snap
    if(-not @($state|Where-Object {$_.Visible -and $_.Class -eq 'Edit' -and $_.Text -ceq $Filename}).Count){throw ('Full filename field missing: '+$Filename)}
    $extension=[IO.Path]::GetExtension($Filename).ToUpperInvariant()
    if(-not @($state|Where-Object {$_.Visible -and $_.Text -ceq ('EXT: '+$extension)}).Count){throw ('File extension indicator missing: '+$Filename)}
}
function Select-File([string]$Name){
    Require-App
    $null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@($Name))
    Wait-For ('file selection '+$Name) { ( [DcmNative]::Text([DcmNative]::Control($hwnd,$app.Id,1005),$app.Id) ) -ceq [IO.Path]::GetFileNameWithoutExtension($Name) }
}
function Capture([string]$Name){
    Require-App
    $screen=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
    if([DcmNative]::IsIconic($hwnd) -or -not [DcmNative]::IsWindowVisible($hwnd) -or -not $script:capturePositioned){
        [DcmNative]::Restore($hwnd,$app.Id,$screen.Left+8,$screen.Top+8)
        Wait-For 'restore for screenshot' { -not [DcmNative]::IsIconic($hwnd) -and [DcmNative]::IsWindowVisible($hwnd) }
        $script:capturePositioned=$true
    }
    Start-Sleep -Milliseconds 275
    $path=Join-Path $lab ($Name+'.png')
    Write-Json (Join-Path $lab ($Name+'.controls.json')) (Snap)
    # Occlusion/minimize guards are intentionally not bypassed.
    $shot=[DcmNative]::Capture($hwnd,$app.Id,$path,$false)
    $captures.Add([ordered]@{name=$Name;path=$shot.Path;sha256=(Hash $path);width=$shot.Width;height=$shot.Height;left=$shot.Left;top=$shot.Top;visual_reviewed=$false})
    return $shot
}
function Click([int]$Id){Require-App;[DcmNative]::ClickAsync($hwnd,$app.Id,$Id)}
function Rotate-PDF([int]$Id,[string]$Name){
    Click $Id
    # Require the actual asynchronous PDF renderer transition; a stale
    # completed footer from before the click is not a successful rotation.
    Wait-For ('PDF rendering started '+$Name) { (Status-Is 'Rendering PDF page') -or (Status-Is 'PDF zoom/page update queued') } 12
    Wait-For ('PDF rendering completed '+$Name) { Status-Is 'DCM PDF Preview' } 35
}
function Image-State([string]$Name,[int]$Turns){
    Assert-Toolbar $true '04-rotation.png'
    $shot=Capture $Name
    $preview=[DcmNative]::ChildByClass($hwnd,$app.Id,'QuickSortPreview')
    $geo=[DcmImageProbe]::Geometry($preview,$app.Id)
    $pixels=[DcmRotationProbe]::CheckImage($shot.Path,$shot.Left,$shot.Top,$geo,360,240,$Turns)
    $cases.Add([ordered]@{name=$Name;family='image';turns=$Turns;passed=$true;pixels=$pixels})
    Check-Files
}
function Pdf-State([string]$Name,[int]$Page,[int]$Turns){
    Wait-For ('PDF page rendered '+$Name) { (Status-Is 'DCM PDF Preview') -and (Status-Is ('Page '+$Page+'/2')) -and [DcmNative]::IsWindowEnabled([DcmNative]::Control($hwnd,$app.Id,1049)) } 35
    Assert-Toolbar $true '08-preview.pdf'
    $state=Snap
    if(-not @($state|Where-Object {$_.Visible -and $_.Text -ceq ('Page '+$Page+' of 2')}).Count){throw 'Wrong PDF page label.'}
    $shot=Capture $Name
    $preview=[DcmNative]::ChildByClass($hwnd,$app.Id,'QuickSortPreview')
    $geo=[DcmImageProbe]::Geometry($preview,$app.Id)
    $color=if($Page -eq 1){'teal'}else{'orange'}
    $box=[DcmRotationProbe]::PdfPanel($shot.Path,$shot.Left,$shot.Top,$geo,$color)
    $cases.Add([ordered]@{name=$Name;family='pdf';page=$Page;turns=$Turns;passed=$true;panel=$box})
    Check-Files
    return $box
}
function Stop-App {
    if($null -eq $app){return}
    Require-App
    [DcmNative]::CloseAsync($hwnd,$app.Id)
    if(-not $app.WaitForExit(18000)){throw 'Owned app did not close normally; refusal to claim acceptance.'}
    $owned['exit_code']=$app.ExitCode;$owned['closed_utc']=[DateTime]::UtcNow.ToString('o')
    $code=$app.ExitCode;$app.Dispose();$script:app=$null
    if($code -ne 0){throw ('Owned app exit code '+$code)}
}
try{
    $build=Get-Content -LiteralPath $BuildReceipt -Raw -Encoding UTF8|ConvertFrom-Json
    if(-not $build.passed -or $build.mode -ne 'build' -or -not $build.source_unchanged){throw 'Successful source-bound build receipt required.'}
    $head=Git-Value @('rev-parse','HEAD')
    if($head -ne $build.commit){throw 'Build/current source commit mismatch.'}
    if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256){throw 'Build manifest hash mismatch.'}
    $manifest=Get-Content -LiteralPath $build.source_hash_manifest.path -Raw -Encoding UTF8|ConvertFrom-Json
    if(-not $manifest.Count){throw 'Build manifest empty.'}
    Check-Source
    if((Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'Build app hash mismatch.'}
    $receipt['commit']=$head
    $receipt['build_receipt']=[ordered]@{path=$BuildReceipt;sha256=(Hash $BuildReceipt)}
    $receipt['app_sha256']=$build.app_artifact.sha256
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-image-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-rotation-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    . (Join-Path $repo 'tests\native\new-fixtures.ps1')
    $fixture=New-DcmFixtures $lab
    $temp=Join-Path $lab 'temp'
    $null=New-Item -ItemType Directory -Path $temp
    $settings=Join-Path $fixture.config 'settings.json'
    $preferences=Get-Content -LiteralPath $settings -Raw|ConvertFrom-Json
    $preferences.pdf_default_view='fit_page'
    Write-Json $settings $preferences
    [DcmImageProbe]::Create((Join-Path $fixture.inbox '04-rotation.png'),360,240,0,$false)
    foreach($f in @(Get-ChildItem -LiteralPath $fixture.inbox -File)){$expected[$f.Name]=Hash $f.FullName}
    $receipt['fixture_sha256']=$expected
    $exe=Join-Path $lab ('DCM-QS-Rotation-'+[Guid]::NewGuid().ToString('N')+'.exe')
    Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    if((Hash $exe) -ne $build.app_artifact.sha256){throw 'Owned app copy mismatch.'}
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '04-rotation.png')+'"'
    $psi.WorkingDirectory=$lab;$psi.UseShellExecute=$false
    $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local
    $psi.EnvironmentVariables['APPDATA']=$fixture.roaming
    $psi.EnvironmentVariables['TEMP']=$temp;$psi.EnvironmentVariables['TMP']=$temp
    $psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'configuration-backups'
    $psi.EnvironmentVariables.Remove('DCM_PDF_PROVIDER')
    $psi.EnvironmentVariables.Remove('DCM_SUMATRA_EXE')
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi
    if(-not $app.Start()){throw 'Candidate did not start.'}
    $owned=[ordered]@{pid=$app.Id;exe=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')}
    $receipt['owned_process']=$owned
    $until=[DateTime]::UtcNow.AddSeconds(25)
    do{$app.Refresh();if($app.HasExited){throw 'Candidate exited at startup.'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    if($hwnd -eq [IntPtr]::Zero){throw 'Main window deadline.'}
    Wait-For 'fixture list visible' { @([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $expected.Count }
    Select-File '04-rotation.png'
    Wait-For 'image preview ready' { (Status-Is 'Image preview: Fit') -and [DcmNative]::IsWindowEnabled([DcmNative]::Control($hwnd,$app.Id,1049)) }
    Image-State 'image-000-original' 0
    Click 1049;Wait-For 'image left 270' { Status-Is 'Image preview rotated 270' };Image-State 'image-270-left' 3
    Click 1050;Wait-For 'image right returns 0' { Status-Is 'Image preview rotated 0' };Image-State 'image-000-return' 0
    Click 1050;Wait-For 'image right 90' { Status-Is 'Image preview rotated 90' };Image-State 'image-090-right' 1
    Click 1050;Wait-For 'image right 180' { Status-Is 'Image preview rotated 180' };Image-State 'image-180-double' 2
    Click 1050;Wait-For 'image right 270' { Status-Is 'Image preview rotated 270' };Image-State 'image-270-three' 3
    Click 1050;Wait-For 'image right back 0' { Status-Is 'Image preview rotated 0' };Image-State 'image-000-four' 0
    Select-File '01-alpha.txt'
    Wait-For 'text preview ready' { Status-Is 'Plain-text preview' }
    Assert-Toolbar $false '01-alpha.txt'
    $null=Capture 'text-buttons-visible-disabled'
    $cases.Add([ordered]@{name='text-rotation-disabled-not-hidden';passed=$true})
    Select-File '08-preview.pdf'
    $normal=Pdf-State 'pdf-p1-000-original' 1 0
    Rotate-PDF 1050 'right'
    $right=Pdf-State 'pdf-p1-090-right' 1 1
    if($right.Ratio -lt ($normal.Ratio*1.20)){throw ('PDF right-rotation orientation did not change: '+$normal.Ratio+' -> '+$right.Ratio)}
    Rotate-PDF 1049 'right-then-left'
    $restored=Pdf-State 'pdf-p1-000-restored' 1 0
    if([Math]::Abs($restored.Ratio-$normal.Ratio) -gt .16){throw 'PDF orientation did not restore after right+left.'}
    Rotate-PDF 1049 'left'
    $left=Pdf-State 'pdf-p1-270-left' 1 3
    if($left.Ratio -lt ($normal.Ratio*1.20)){throw 'PDF left rotation did not change orientation.'}
    Click 1061
    $page2=Pdf-State 'pdf-p2-270-left' 2 3
    if($page2.Samples -lt 100){throw 'Second PDF page lacks orange pixels.'}
    Check-Files
    Stop-App
    Check-Source
    if((Hash $exe) -ne $build.app_artifact.sha256){throw 'Candidate executable changed.'}
    $receipt['fixtures_unchanged']=$true
    $receipt['source_unchanged']=$true
    $receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message
    $receipt['error_position']=$_.InvocationInfo.PositionMessage
    Write-Output ('FAIL='+$_.Exception.Message)
    if($null -ne $app){try{$null=Capture 'failure-state'}catch{$receipt['failure_capture_error']=$_.Exception.Message}}
}finally{
    try{Check-Files;$receipt['fixtures_unchanged']=$true}catch{$receipt.passed=$false;$receipt['fixture_error']=$_.Exception.Message}
    if($null -ne $app){try{Stop-App}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}}
    if($head -and $manifest.Count){try{Check-Source;$receipt['source_unchanged']=$true}catch{$receipt.passed=$false;$receipt['source_error']=$_.Exception.Message}}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    Write-Json (Join-Path $lab 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
