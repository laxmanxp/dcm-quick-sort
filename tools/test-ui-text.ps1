[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT)
# DCM-26: source-bound native text-preview acceptance; no installed app or user files.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt){throw 'Workspace and exact build receipt required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-text-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Lab must be outside checkout.'}
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-26';app='dcm-quicksort';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;cases=$cases;captures=$captures}
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
    if($files.Count -ne $expected.Count){throw 'Text fixture file set changed.'}
    foreach($f in $files){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Text preview changed source bytes.'}}
    if(@(Get-ChildItem -LiteralPath $destination -Force).Count){throw 'Text preview changed destination contents.'}
    $journal=Join-Path $config 'recovery\move-journal.jsonl'
    if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Text preview appended a file operation.'}
}
function String-Hash([string]$Text){$h=[Security.Cryptography.SHA256]::Create();try{return ([BitConverter]::ToString($h.ComputeHash($utf8.GetBytes($Text)))).Replace('-','').ToLowerInvariant()}finally{$h.Dispose()}}
function Shot([string]$Name){
    Require-App
    $screen=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
    [DcmNative]::Restore($hwnd,$app.Id,$screen.Left+12,$screen.Top+12)
    Start-Sleep -Milliseconds 300
    Write-Json (Join-Path $lab ($Name+'.controls.json')) @([DcmTextProbe]::Controls($hwnd,$app.Id) | ForEach-Object {[ordered]@{handle=$_.Handle;class=$_.Class;visible=$_.Visible;length=$_.Text.Length;text=$_.Text.Substring(0,[Math]::Min(512,$_.Text.Length));sha256=(String-Hash $_.Text)}})
    $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($Name+'.png')),$false)
    $captures.Add([ordered]@{feature='DCM-26';name=$Name;path=$s.Path;sha256=(Hash $s.Path);width=$s.Width;height=$s.Height;visual_reviewed=$false})
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
function Select-Text([string]$Name){
    Require-App
    $null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@($Name))
    $selectedStem=[IO.Path]::GetFileNameWithoutExtension($Name)
    Wait-For ('text selection '+$Name) { (Text 1005) -ceq $selectedStem }
}
function Assert-Text([string]$Name,[string]$Want,[switch]$Tail,[switch]$Capture){
    Require-App
    $p=[DcmTextProbe]::Preview($hwnd,$app.Id)
    if($null -eq $p -or $p.Text -cne $Want){
        $actual=if($null -eq $p){'<no visible text preview>'}else{$p.Text.Substring(0,[Math]::Min(200,$p.Text.Length))}
        throw ('DCM26_TEXT_MISMATCH: '+$Name+' actual='+$actual)
    }
    $cases.Add([ordered]@{name=$Name;passed=$true;displayed_characters=$p.Text.Length;displayed_sha256=(String-Hash $p.Text);expected_sha256=(String-Hash $Want)})
    if($Tail){$cases[$cases.Count-1]['tail_first_visible_line']=[DcmTextProbe]::ScrollEnd($hwnd,$app.Id,[IntPtr]$p.Handle)}
    if($Capture){Shot $Name}
}
function Assert-TextError([string]$Name,[string]$Reason){
    Require-App
    $controls=@([DcmTextProbe]::Controls($hwnd,$app.Id))
    $errorText=@($controls | Where-Object {$_.Visible -and $_.Class -eq 'Static' -and $_.Text.Contains('Text preview unavailable:') -and $_.Text.Contains($Reason)})
    if($null -ne [DcmTextProbe]::Preview($hwnd,$app.Id) -or $errorText.Count -ne 1){throw ('DCM26_TRUTHFUL_ERROR_MISSING: '+$Name)}
    $cases.Add([ordered]@{name=$Name;passed=$true;visible_text_preview=$false;status=$errorText[0].Text})
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
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-text-probe.cs')
    $vision=$env:DCM_FLOWUIVISION_EXE
    if(-not $vision){$vision='C:\FlowverseReports\dcm-flowuivision-20261103\FlowUIVision.exe'}
    $receipt['flowuivision']=[ordered]@{path=$vision;sha256=(Hash $vision)}
    $inbox=Join-Path $lab 'Text Inbox'
    $vault=Join-Path $lab 'Vault';$destination=Join-Path $vault 'Accepted'
    $local=Join-Path $lab 'profile\local';$roaming=Join-Path $lab 'profile\roaming'
    $config=Join-Path $local 'DCM Quick Sort';$temp=Join-Path $lab 'temp'
    $null=New-Item -ItemType Directory -Path $inbox,$destination,$config,$roaming,$temp
    Write-Json (Join-Path $config 'settings.json') ([ordered]@{default_source=$inbox;pdf_default_view='fit_width';vault_root=$vault})
    [IO.File]::WriteAllText((Join-Path $config 'destinations.conf'),"Acceptance=Accepted`r`n",$utf8)
    $unicode='Unicode: caf'+[char]0xE9+' '+[char]0x03A9+' '+[char]0x0B13+[char]0x0B21+[char]0x0B3C+[char]0x0B3F+[char]0x0B06+' '+[char]::ConvertFromUtf32(0x1F600)
    $le=New-Object Text.UnicodeEncoding($false,$true,$true);$be=New-Object Text.UnicodeEncoding($true,$true,$true)
    $line="DCM-26 bounded text line; actual content must be read.`n"
    $prefix=($line*6000).Substring(0,262142)
    $normalPrefix=$prefix.Replace("`n","`r`n")
    $notice="`r`n`r`n[Preview limited to the first 256.0 KiB.]"
    $spec=@(
        @{name='01-plain.txt';data=$utf8.GetBytes("First line`nSecond line`rThird line`r`nEND");want="First line`r`nSecond line`r`nThird line`r`nEND"},
        @{name='02-markdown.md';data=$utf8.GetBytes("# Heading`n**literal markup**`n- item");want="# Heading`r`n**literal markup**`r`n- item"},
        @{name='03-records.csv';data=$utf8.GetBytes("name,value`n`"two,words`",42");want="name,value`r`n`"two,words`",42"},
        @{name='04-utf8.TXT';data=([byte[]]@(239,187,191)+$utf8.GetBytes($unicode));want=$unicode},
        @{name='05-utf16-le.txt';data=($le.GetPreamble()+$le.GetBytes($unicode));want=$unicode},
        @{name='06-utf16-be.txt';data=($be.GetPreamble()+$be.GetBytes($unicode));want=$unicode},
        @{name='07-empty.txt';data=[byte[]]@();want='[Empty file]'},
        @{name='08-locked.txt';data=$utf8.GetBytes('Text after the lock is released.');want='Text after the lock is released.'},
        @{name='09-invalid-utf8.txt';data=[byte[]]@(65,255,66);reason='encoding'},
        @{name='10-odd-utf16.txt';data=[byte[]]@(255,254,65,0,66);reason='UTF-16'},
        @{name='11-utf32.txt';data=[byte[]]@(255,254,0,0,65,0,0,0);reason='UTF-32'},
        @{name='12-binary.txt';data=[byte[]]@(65,0,1,2,66);want=('[Binary/non-text bytes detected. Showing a safe text interpretation.]' + "`r`n`r`nA" + [char]0xB7+[char]0xB7+[char]0xB7+'B')},
        @{name='13-limit-utf8.txt';data=$utf8.GetBytes($prefix+[char]0x20AC+'TAIL_NOT_DISPLAYED');want=($normalPrefix+$notice);tail=$true},
        @{name='14-limit-ascii.txt';data=$utf8.GetBytes($prefix+'XYTAIL_NOT_DISPLAYED');want=($normalPrefix+'XY'+$notice);tail=$true},
        @{name='15-manual.dcm26text';data=$utf8.GetBytes('Unknown extension: explicit Read as text.');want='Unknown extension: explicit Read as text.'}
    )
    foreach($f in $spec){$path=Join-Path $inbox $f.name;[IO.File]::WriteAllBytes($path,[byte[]]$f.data);$expected[$f.name]=Hash $path}
    Write-Json (Join-Path $lab 'fixture-hashes.json') $expected
    $exe=Join-Path $lab ('DCM-QS-Text-'+[Guid]::NewGuid().ToString('N')+'.exe')
    Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
    if((Hash $exe) -ne $build.app_artifact.sha256){throw 'Owned app copy hash mismatch.'}
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $inbox '01-plain.txt')+'"';$psi.WorkingDirectory=$lab;$psi.UseShellExecute=$false
    $psi.EnvironmentVariables['LOCALAPPDATA']=$local;$psi.EnvironmentVariables['APPDATA']=$roaming
    $psi.EnvironmentVariables['TEMP']=$temp;$psi.EnvironmentVariables['TMP']=$temp;$psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'configuration-backups'
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi
    if(-not $app.Start()){throw 'Candidate failed to start.'}
    $owned=[ordered]@{pid=$app.Id;exe=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')}
    $receipt['owned_process']=$owned
    $until=[DateTime]::UtcNow.AddSeconds(20)
    do{$app.Refresh();if($app.HasExited){throw 'Candidate exited at startup.'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    if($hwnd -eq [IntPtr]::Zero){throw 'Main window deadline.'}
    Wait-For 'text fixtures ready' { @([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $spec.Count }
    Invoke-Sort 'DCM-26-setup' 'Name'
    foreach($f in $spec[0..6]){Select-Text $f.name;Assert-Text ('DCM-26-'+$f.name) $f.want -Capture;Check-Files}
    # Exclusive read lock changes no bytes or ACLs and deterministically tests read failure.
    $lock=[IO.File]::Open((Join-Path $inbox '08-locked.txt'),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None)
    try{Select-Text '08-locked.txt';Assert-TextError 'DCM-26-sharing-violation' '08-locked.txt'}finally{$lock.Dispose();$lock=$null}
    Select-Text '08-locked.txt';Assert-Text 'DCM-26-read-recovery' $spec[7].want -Capture;Check-Files
    foreach($f in $spec[8..10]){Select-Text $f.name;Assert-TextError ('DCM-26-'+$f.name) $f.reason;Check-Files}
    foreach($f in $spec[11..13]){Select-Text $f.name;Assert-Text ('DCM-26-'+$f.name) $f.want -Tail:($f.ContainsKey('tail')) -Capture;Check-Files}
    Select-Text '15-manual.dcm26text'
    [DcmNative]::ClickAsync($hwnd,$app.Id,1014)
    Wait-For 'explicit text preview' { $null -ne [DcmTextProbe]::Preview($hwnd,$app.Id) }
    Assert-Text 'DCM-26-explicit-read-as-text' $spec[14].want -Capture;Check-Files
    if(@([DcmTextProbe]::Controls($hwnd,$app.Id) | Where-Object {$_.Visible -and $_.Class -eq 'Static' -and $_.Text -ceq 'Read as text: DCM26TEXT'}).Count -ne 1){throw 'DCM26_MANUAL_STATUS_STALE: explicit text retained an unrelated file-type status.'}
    Select-Text '01-plain.txt';Assert-Text 'DCM-26-final-recovery' $spec[0].want -Capture;Check-Files
    $receipt['fixture_hashes']=$expected;$receipt['preview_limit_bytes']=262144
    Check-Files;Stop-App;Check-Source
    if((Hash $exe) -ne $build.app_artifact.sha256 -or (Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'App bytes changed during qualification.'}
    $receipt['fixtures_unchanged']=$true;$receipt['source_unchanged']=$true;$receipt.passed=$true
}catch{
    $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage
    Write-Output ('FAIL='+$_.Exception.Message)
    if($null -ne $app){try{Shot 'DCM-26-failure'}catch{$receipt['capture_error']=$_.Exception.Message}}
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
