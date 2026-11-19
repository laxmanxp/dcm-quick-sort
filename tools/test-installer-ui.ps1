# Actual packaged setup: open, inspect, capture, then Cancel. Never activates Install.
[CmdletBinding()]
param([string]$PackageReceipt=$env:DCM_PACKAGE_RECEIPT)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT){throw 'FLOWDEV_WORKSPACE_ROOT is required.'}
if(-not $PackageReceipt){throw 'Set DCM_PACKAGE_RECEIPT to the exact successful package receipt.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$id=[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8)
$report=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-installer-ui-'+$id)
New-Item -ItemType Directory -Path $report | Out-Null
$utf8=New-Object Text.UTF8Encoding($false)
$receipt=[ordered]@{schema=1;app='dcm-quicksort';scope='actual packaged setup open/inspect/cancel only';report=$report;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;visual_reviewed=$false;clicked_control_ids=@()}
$proc=$null;$owned=$null;$window=[IntPtr]::Zero;$beforeRegistry=$null;$beforeHostFiles=$null
Write-Output ('REPORT_ROOT='+$report)
function Hash([string]$Path){return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()}
function Write-Json([string]$Path,$Value){[IO.File]::WriteAllText($Path,($Value | ConvertTo-Json -Depth 40),$utf8)}
function Read-RegistryNode($Key){
    $node=[ordered]@{exists=$true;values=[ordered]@{};children=[ordered]@{}}
    foreach($name in @($Key.GetValueNames() | Sort-Object)){$node.values[$name]=[ordered]@{kind=$Key.GetValueKind($name).ToString();value=$Key.GetValue($name,$null,[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)}}
    foreach($name in @($Key.GetSubKeyNames() | Sort-Object)){$child=$Key.OpenSubKey($name,$false);try{$node.children[$name]=Read-RegistryNode $child}finally{$child.Dispose()}}
    return $node
}
function Registry-Snapshot {
    $snapshot=[ordered]@{}
    foreach($path in @('Software\Microsoft\Windows\CurrentVersion\Uninstall\DCMQuickSort','Software\Microsoft\Windows\CurrentVersion\App Paths\DCM-QuickSort.exe','Software\Classes\*\shell\DCMQuickSort','Software\Classes\Directory\shell\DCMQuickSort','Software\Classes\Directory\Background\shell\DCMQuickSort')){
        $key=[Microsoft.Win32.Registry]::CurrentUser.OpenSubKey($path,$false)
        if($key){try{$snapshot[$path]=Read-RegistryNode $key}finally{$key.Dispose()}}else{$snapshot[$path]=[ordered]@{exists=$false}}
    }
    return $snapshot
}
function Host-Files {
    return @(@((Join-Path $env:LOCALAPPDATA 'Programs\DCM Quick Sort\DCM-QuickSort.exe'),(Join-Path $env:LOCALAPPDATA 'Programs\DCM Quick Sort\Uninstall DCM Quick Sort.exe'),(Join-Path $env:LOCALAPPDATA 'DCM Quick Sort\settings.json'),(Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\DCM Quick Sort.lnk')) | ForEach-Object {if(Test-Path -LiteralPath $_ -PathType Leaf){[ordered]@{path=$_;exists=$true;sha256=(Hash $_)}}else{[ordered]@{path=$_;exists=$false}}})
}
function Require-OwnedProcess {
    $proc.Refresh()
    if($proc.HasExited -or $proc.Path -ne $owned.exe -or $proc.StartTime.ToUniversalTime().ToString('o') -ne $owned.start){throw 'Packaged setup process identity/lifetime changed.'}
}
function Require-Owned {
    Require-OwnedProcess
    [DcmNative]::RequireOwned($window,$proc.Id)
}
try{
    $package=Get-Content -LiteralPath $PackageReceipt -Raw -Encoding UTF8 | ConvertFrom-Json
    if($package.passed -ne $true -or $package.mode -ne 'package'){throw 'A successful exact package receipt is required.'}
    $head=(& git.exe -C $repo rev-parse HEAD | Out-String).Trim();if($LASTEXITCODE -ne 0){throw 'Cannot read source HEAD.'}
    $dirty=(& git.exe -C $repo status --porcelain=v1 --untracked-files=all | Out-String).Trim()
    if($LASTEXITCODE -ne 0 -or $dirty -or $head -ne $package.commit){throw 'Package and clean current source/tooling commit must match.'}
    if((Hash $package.installer_artifact.path) -ne $package.installer_artifact.sha256){throw 'Packaged installer hash changed.'}
    if((Hash $package.source_hash_manifest.path) -ne $package.source_hash_manifest.sha256){throw 'Package source manifest changed.'}
    foreach($file in (Get-Content -LiteralPath $package.source_hash_manifest.path -Raw | ConvertFrom-Json)){
        $full=[IO.Path]::GetFullPath((Join-Path $repo $file.path))
        if(-not $full.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $full) -ne $file.sha256){throw 'Current source bytes differ from packaged snapshot.'}
    }
    $receipt['commit']=$head;$receipt['package_receipt']=[ordered]@{path=$PackageReceipt;sha256=(Hash $PackageReceipt)};$receipt['installer_sha256']=$package.installer_artifact.sha256
    $beforeRegistry=Registry-Snapshot;$beforeHostFiles=Host-Files
    Write-Json (Join-Path $report 'host-registry-before.json') $beforeRegistry
    Write-Json (Join-Path $report 'host-files-before.json') $beforeHostFiles
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    . (Join-Path $repo 'tests\native\new-fixtures.ps1')
    $fixture=New-DcmFixtures $report
    $configHash=Hash (Join-Path $fixture.config 'settings.json')
    $exe=Join-Path $report ('DCM-QuickSort-Setup-'+$id+'.exe')
    Copy-Item -LiteralPath $package.installer_artifact.path -Destination $exe
    if((Hash $exe) -ne $package.installer_artifact.sha256){throw 'Copied setup bytes changed.'}
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe;$psi.WorkingDirectory=$report;$psi.UseShellExecute=$false
    $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming
    $psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $report 'configuration-backups'
    $psi.EnvironmentVariables['TEMP']=Join-Path $report 'temp';$psi.EnvironmentVariables['TMP']=Join-Path $report 'temp'
    New-Item -ItemType Directory -Path (Join-Path $report 'temp') | Out-Null
    $proc=New-Object Diagnostics.Process;$proc.StartInfo=$psi
    if(-not $proc.Start()){throw 'Could not launch packaged setup.'}
    $owned=[ordered]@{pid=$proc.Id;exe=$exe;start=$proc.StartTime.ToUniversalTime().ToString('o');hwnd=0};$receipt['owned']=$owned
    $until=[DateTime]::UtcNow.AddSeconds(15)
    do{$proc.Refresh();if($proc.HasExited){throw 'Packaged setup exited before observation.'};$window=$proc.MainWindowHandle;if($window -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    if($window -eq [IntPtr]::Zero){throw 'Packaged setup window deadline.'}
    $owned.hwnd=$window.ToInt64()
    Require-Owned
    $source=Get-Content -LiteralPath (Join-Path $repo 'installer-src\core.go') -Raw
    if($source -notmatch 'productVersion\s*=\s*"([^"]+)"'){throw 'Cannot read packaged source version.'}
    $version=$Matches[1]
    if([DcmNative]::Text($window,$proc.Id) -notlike ('*'+$version+'*')){throw 'Setup title version mismatch.'}
    if([DcmNative]::Text([DcmNative]::Control($window,$proc.Id,1001),$proc.Id) -ne $fixture.inbox){throw 'Setup did not read its isolated source configuration.'}
    if(-not [DcmNative]::IsWindowEnabled([DcmNative]::Control($window,$proc.Id,1003))){throw 'Setup Install control unavailable.'}
    $area=[Windows.Forms.Screen]::FromHandle($window).WorkingArea
    [DcmNative]::Restore($window,$proc.Id,$area.Left+8,$area.Top+8)
    Start-Sleep -Milliseconds 300
    Write-Json (Join-Path $report 'setup-controls.json') ([DcmNative]::Snapshot($window,$proc.Id))
    $capture=[DcmNative]::Capture($window,$proc.Id,(Join-Path $report 'setup-open.png'),$false)
    $receipt['capture']=[ordered]@{path=$capture.Path;sha256=(Hash $capture.Path);width=$capture.Width;height=$capture.Height;method=$capture.Method;visual_reviewed=$false}
    Require-Owned
    $receipt.clicked_control_ids=@(1004)
    [DcmNative]::ClickAsync($window,$proc.Id,1004)
    if(-not $proc.WaitForExit(10000)){throw 'Setup Cancel did not close the owned process.'}
    $receipt['exit_code']=$proc.ExitCode
    if($proc.ExitCode -ne 0){throw 'Setup returned nonzero after Cancel.'}
    if(Test-Path -LiteralPath (Join-Path $fixture.local 'Programs\DCM Quick Sort')){throw 'Open/Cancel unexpectedly created an install directory.'}
    if(Test-Path -LiteralPath (Join-Path $fixture.roaming 'Microsoft\Windows\Start Menu\Programs\DCM Quick Sort.lnk')){throw 'Open/Cancel unexpectedly created a shortcut.'}
    if((Hash (Join-Path $fixture.config 'settings.json')) -ne $configHash){throw 'Open/Cancel changed configuration.'}
    $afterRegistry=Registry-Snapshot;$afterHostFiles=Host-Files
    Write-Json (Join-Path $report 'host-registry-after.json') $afterRegistry
    Write-Json (Join-Path $report 'host-files-after.json') $afterHostFiles
    $receipt['host_registry_unchanged']=(($beforeRegistry | ConvertTo-Json -Depth 40 -Compress) -ceq ($afterRegistry | ConvertTo-Json -Depth 40 -Compress))
    $receipt['host_files_unchanged']=(($beforeHostFiles | ConvertTo-Json -Depth 10 -Compress) -ceq ($afterHostFiles | ConvertTo-Json -Depth 10 -Compress))
    if(-not $receipt.host_registry_unchanged -or -not $receipt.host_files_unchanged){throw 'Host installed state changed during open/cancel smoke.'}
    if((Hash $exe) -ne $package.installer_artifact.sha256){throw 'Setup bytes changed.'}
    $receipt.passed=$true
}catch{$receipt['error']=$_.Exception.Message;Write-Output ('FAIL='+$_.Exception.Message)}
finally{
    if($proc){
        try{
            $proc.Refresh()
            if(-not $proc.HasExited){
                Require-OwnedProcess
                try{[DcmNative]::CloseAsync($window,$proc.Id)}catch{$receipt['close_window_error']=$_.Exception.Message}
                if(-not $proc.WaitForExit(5000)){
                    Require-OwnedProcess
                    & "$env:SystemRoot\System32\taskkill.exe" /PID $proc.Id /T /F | Out-Null
                    if(-not $proc.WaitForExit(5000)){throw 'Owned setup could not be drained after forced cleanup.'}
                }
                $receipt.passed=$false;$receipt['cleanup_after_failure']=$true
            }
        }catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}
        finally{$proc.Dispose()}
    }
    if($null -ne $beforeRegistry){
        try{
            $finalRegistry=Registry-Snapshot;$finalHostFiles=Host-Files
            Write-Json (Join-Path $report 'host-registry-final.json') $finalRegistry
            Write-Json (Join-Path $report 'host-files-final.json') $finalHostFiles
            $receipt['host_registry_unchanged']=(($beforeRegistry | ConvertTo-Json -Depth 40 -Compress) -ceq ($finalRegistry | ConvertTo-Json -Depth 40 -Compress))
            $receipt['host_files_unchanged']=(($beforeHostFiles | ConvertTo-Json -Depth 10 -Compress) -ceq ($finalHostFiles | ConvertTo-Json -Depth 10 -Compress))
            if(-not $receipt.host_registry_unchanged -or -not $receipt.host_files_unchanged){$receipt.passed=$false}
        }catch{$receipt.passed=$false;$receipt['final_state_error']=$_.Exception.Message}
    }
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    Write-Json (Join-Path $report 'receipt.json') $receipt
    Write-Output ('RECEIPT='+(Join-Path $report 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
