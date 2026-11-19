[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT)
# DCM-20: bounded, owned native right-click regression. No host registration.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT -or -not $BuildReceipt){throw 'Workspace and exact build receipt required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-context-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Evidence must be outside checkout.'}
$null=New-Item -ItemType Directory -Path $lab
$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]'
$captures=New-Object 'Collections.Generic.List[object]'
$r=[ordered]@{schema=1;app='dcm-quicksort';feature='DCM-20';scope='native right-click targeting, cancel, blank area and controlled default-handler Open; no user-document application claim';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;feature_accepted=$false;external_open_tested=$false;visual_reviewed=$false;cases=$cases;captures=$captures}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$manifest=@();$head='';$expected=@{}
Write-Output ('REPORT_ROOT='+$lab)
function Json([string]$Path,$Value){[IO.File]::WriteAllText($Path,($Value | ConvertTo-Json -Depth 25),$utf8)}
function Hash([string]$Path){return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()}
function Git-Value([string[]]$Arguments){$v=(& git.exe -C $repo @Arguments | Out-String).Trim();if($LASTEXITCODE -ne 0){throw 'Git query failed.'};return $v}
function Check-Source {
    if((Git-Value @('rev-parse','HEAD')) -ne $head -or (Git-Value @('status','--porcelain=v1','--untracked-files=all'))){throw 'Checkout identity or clean status changed.'}
    foreach($f in $manifest){$p=[IO.Path]::GetFullPath((Join-Path $repo $f.path));if(-not $p.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $p) -ne $f.sha256){throw 'Source manifest mismatch.'}}
}
function Owned {
    if($null -eq $app){throw 'Owned app absent.'};$app.Refresh()
    if($app.HasExited -or $app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $owned.start){throw 'Owned process identity/liveness changed.'}
    [DcmNative]::RequireOwned($hwnd,$app.Id)
}
function Files-Unchanged {
    $files=@(Get-ChildItem -LiteralPath $inbox -File)
    if($files.Count -ne $expected.Count){throw 'Fixture set changed.'}
    foreach($f in $files){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Fixture names or bytes changed.'}}
    if(@(Get-ChildItem -LiteralPath $dest -Force).Count){throw 'Unexpected destination entry.'}
    $journal=Join-Path $config 'recovery\move-journal.jsonl'
    if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Right-click or Cancel appended a file-operation journal.'}
}
function Menu-Shot([string]$Name,[IntPtr]$Menu){
    Owned
    $shot=[DcmMenuActions]::Capture($Menu,$app.Id,(Join-Path $lab ($Name+'.png')))
    $captures.Add([ordered]@{name=$Name;path=$shot.Path;sha256=(Hash $shot.Path);width=$shot.Width;height=$shot.Height;method='owned popup CopyFromScreen with occlusion checks';visual_reviewed=$false})
    Json (Join-Path $lab ($Name+'.menu.json')) ([DcmMenuActions]::Items($Menu,$app.Id))
}
function Wait-Menu {
    $until=[DateTime]::UtcNow.AddSeconds(5)
    do{Owned;$found=@([DcmContextProbe]::Menus($app.Id));if($found.Count -eq 1){return [IntPtr]$found[0]};if($found.Count -gt 1){throw 'Unexpected multiple owned menu windows.'};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    throw 'Owned menu did not appear.'
}
function Shot([string]$Name){
    Owned
    $area=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
    [DcmNative]::Restore($hwnd,$app.Id,$area.Left+8,$area.Top+8)
    Start-Sleep -Milliseconds 350
    $s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($Name+'.png')),$false)
    $captures.Add([ordered]@{name=$Name;path=$s.Path;sha256=(Hash $s.Path);width=$s.Width;height=$s.Height;visual_reviewed=$false})
}
try{
    $b=Get-Content -LiteralPath $BuildReceipt -Raw -Encoding UTF8 | ConvertFrom-Json
    if($b.passed -ne $true -or $b.mode -notin @('build','package') -or -not $b.source_unchanged){throw 'Successful source-bound build receipt required.'}
    $head=Git-Value @('rev-parse','HEAD')
    if($head -ne $b.commit -or (Git-Value @('symbolic-ref','--short','HEAD')) -ne 'release/v1.0.71' -or (Git-Value @('remote','get-url','origin')) -ne 'https://github.com/laxmanxp/dcm-quick-sort.git'){throw 'Build/source identity mismatch.'}
    if((Hash $b.source_hash_manifest.path) -ne $b.source_hash_manifest.sha256){throw 'Manifest hash mismatch.'}
    $manifest=Get-Content -LiteralPath $b.source_hash_manifest.path -Raw -Encoding UTF8 | ConvertFrom-Json
    Check-Source
    if((Hash $b.app_artifact.path) -ne $b.app_artifact.sha256){throw 'App hash mismatch.'}
    $r['commit']=$head;$r['build_receipt']=[ordered]@{path=$BuildReceipt;sha256=(Hash $BuildReceipt)};$r['app_sha256']=$b.app_artifact.sha256
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
public static class DcmContextProbe {
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left,Top,Right,Bottom; }
    private delegate bool EnumProc(IntPtr h,IntPtr p);
    [DllImport("user32.dll")] private static extern bool EnumWindows(EnumProc cb,IntPtr p);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr h,StringBuilder text,int max);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll",EntryPoint="SendMessageTimeoutW")] private static extern IntPtr RectMessage(IntPtr h,uint m,IntPtr w,ref RECT rect,uint flags,uint timeout,out IntPtr result);
    [DllImport("user32.dll")] private static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    [DllImport("user32.dll")] private static extern bool GetClientRect(IntPtr h,out RECT r);
    public static int[] RightClick(IntPtr list,int pid,int index) {
        uint actual;GetWindowThreadProcessId(list,out actual);if(actual!=(uint)pid)throw new Exception("List PID mismatch");
        var c=new StringBuilder(128);GetClassName(list,c,c.Capacity);if(c.ToString()!="ListBox")throw new Exception("Not a ListBox");
        RECT rect=new RECT();IntPtr result;
        if(RectMessage(list,0x0198,(IntPtr)index,ref rect,2,1500,out result)==IntPtr.Zero || result.ToInt64()<0)throw new Exception("Cannot read target row geometry");
        int x=rect.Left+8,y=rect.Top+(rect.Bottom-rect.Top)/2;
        if(x<0||y<0||x>32767||y>32767)throw new Exception("Unsafe row coordinates");
        IntPtr point=(IntPtr)((y<<16)|(x&65535));
        if(!PostMessage(list,0x0204,(IntPtr)2,point)||!PostMessage(list,0x0205,IntPtr.Zero,point))throw new Exception("Right-click message failed");
        return new int[]{x,y};
    }
    public static void BlankArea(IntPtr list,int pid,int lastIndex) {
        uint actual;GetWindowThreadProcessId(list,out actual);if(actual!=(uint)pid)throw new Exception("List PID mismatch");
        RECT item=new RECT(),client;IntPtr result;
        if(RectMessage(list,0x0198,(IntPtr)lastIndex,ref item,2,1500,out result)==IntPtr.Zero || result.ToInt64()<0 || !GetClientRect(list,out client))throw new Exception("Cannot read blank list geometry");
        int x=8,y=item.Bottom+20;if(y>=client.Bottom-4)throw new Exception("No blank area for this fixture");
        IntPtr point=(IntPtr)((y<<16)|x);
        if(!PostMessage(list,0x0204,(IntPtr)2,point)||!PostMessage(list,0x0205,IntPtr.Zero,point))throw new Exception("Blank-area click failed");
    }
    public static long[] Menus(int pid){
        var found=new List<long>();
        EnumWindows(delegate(IntPtr h,IntPtr p){uint actual;GetWindowThreadProcessId(h,out actual);if(actual==(uint)pid && IsWindowVisible(h)){var b=new StringBuilder(128);GetClassName(h,b,b.Capacity);if(b.ToString()=="#32768")found.Add(h.ToInt64());}return true;},IntPtr.Zero);
        return found.ToArray();
    }
    public static void Cancel(IntPtr root,int pid){uint actual;GetWindowThreadProcessId(root,out actual);if(actual!=(uint)pid)throw new Exception("Cancel PID mismatch");if(!PostMessage(root,0x001F,IntPtr.Zero,IntPtr.Zero))throw new Exception("Menu cancellation failed");}
}
'@
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-menu-actions.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    $inbox=Join-Path $lab 'Context Inbox';$dest=Join-Path $lab 'Vault\Accepted';$local=Join-Path $lab 'Local';$roaming=Join-Path $lab 'Roaming';$config=Join-Path $roaming 'DCM-QuickSort';$temp=Join-Path $lab 'Temp'
    foreach($p in @($inbox,$dest,$local,$config,$temp)){ $null=New-Item -ItemType Directory -Path $p -Force }
    foreach($name in @('01-selected.txt','02-right-click target.txt')){ $p=Join-Path $inbox $name;[IO.File]::WriteAllText($p,('DCM-20 disposable fixture: '+$name),$utf8);$expected[$name]=Hash $p }
    # The only executable test files are locally generated, audited marker writers.
    # The Windows default .cmd Open handler is exercised; no association is changed.
    if($lab -match '[%&!^]'){throw 'Lab path contains unsupported command-file metacharacters.'}
    $openMarker=Join-Path $lab 'external-open.txt';$wrongMarker=Join-Path $lab 'wrong-open.txt'
    $token='DCM20-'+[Guid]::NewGuid().ToString('N')
    foreach($entry in @(@('03-wrong-open.cmd',$wrongMarker),@('04-external open.cmd',$openMarker))){
        $path=Join-Path $inbox $entry[0]
        $body="@echo off`r`n> `"$($entry[1])`" echo $token`r`nexit /b 0`r`n"
        [IO.File]::WriteAllText($path,$body,[Text.Encoding]::ASCII);$expected[$entry[0]]=Hash $path
    }
    $r['external_open_contract']='Actual native Open on a disposable .cmd file writes a unique owned marker and exits; verifies default command-file handling, not all document associations.'
    Json (Join-Path $config 'settings.json') ([ordered]@{default_source=$inbox;vault_root=(Join-Path $lab 'Vault');pdf_default_view='fit-width'})
    [IO.File]::WriteAllText((Join-Path $config 'destinations.conf'),"Acceptance=Accepted`r`n",$utf8)
    $exe=Join-Path $lab ('DCM-QS-Context-'+[Guid]::NewGuid().ToString('N')+'.exe')
    Copy-Item -LiteralPath $b.app_artifact.path -Destination $exe
    if((Hash $exe) -ne $b.app_artifact.sha256){throw 'Copied app hash mismatch.'}
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$exe;$psi.Arguments='"'+$inbox+'"';$psi.WorkingDirectory=$lab;$psi.UseShellExecute=$false
    foreach($pair in (@{LOCALAPPDATA=$local;APPDATA=$roaming;TEMP=$temp;TMP=$temp;DCM_QUICKSORT_BACKUP_DIR=(Join-Path $lab 'Backups')}).GetEnumerator()){$psi.EnvironmentVariables[$pair.Key]=$pair.Value}
    $app=New-Object Diagnostics.Process;$app.StartInfo=$psi
    if(-not $app.Start()){throw 'App failed to start.'}
    $owned=[ordered]@{pid=$app.Id;exe=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')};$r['owned_process']=$owned
    $until=[DateTime]::UtcNow.AddSeconds(20)
    do{$app.Refresh();if($app.HasExited){throw 'App exited during launch.'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    Owned
    if([DcmNative]::Text($hwnd,$app.Id) -notlike '*1.0.71*'){throw 'Wrong candidate version.'}
    $items=@([DcmNative]::ListItems($hwnd,$app.Id,1016))
    if(($items | Sort-Object) -join '|' -cne (($expected.Keys | Sort-Object) -join '|')){throw 'Unexpected initial file list.'}
    $null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@('01-selected.txt'))
    Shot 'DCM-20-before-right-click'
    $targetIndex=[Array]::IndexOf($items,'02-right-click target.txt')
    if($targetIndex -lt 0){throw 'Right-click target absent.'}
    $list=[DcmNative]::Control($hwnd,$app.Id,1016)
    if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count -or @([DcmContextProbe]::Menus($app.Id)).Count){throw 'Unexpected popup before right-click.'}
    $point=[DcmContextProbe]::RightClick($list,$app.Id,$targetIndex)
    $r['right_click']=[ordered]@{target='02-right-click target.txt';index=$targetIndex;list_client_point=$point;input='owned ListBox WM_RBUTTONDOWN / WM_RBUTTONUP'}
    $until=[DateTime]::UtcNow.AddSeconds(5);$menus=@()
    do{Owned;$menus=@([DcmContextProbe]::Menus($app.Id));if($menus.Count){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    $r['observed_menu_handles']=$menus
    Json (Join-Path $lab 'DCM-20-after-right-click.controls.json') ([DcmNative]::Snapshot($hwnd,$app.Id))
    $r['selected_indexes_after_click']=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016))
    if($menus.Count -eq 0){Shot 'DCM-20-no-context-menu';Files-Unchanged;throw 'DCM-20-MENU-01: no owned native context menu appeared after right-clicking the second file.'}
    if($menus.Count -ne 1){throw 'Unexpected multiple native menu windows.'}
    Menu-Shot 'DCM-20-text-file-menu' ([IntPtr]$menus[0])
    $cases.Add([ordered]@{name='owned context menu appears';passed=$true})
    [DcmContextProbe]::Cancel($hwnd,$app.Id)
    $until=[DateTime]::UtcNow.AddSeconds(5)
    do{Start-Sleep -Milliseconds 100;$menus=@([DcmContextProbe]::Menus($app.Id))}while($menus.Count -and [DateTime]::UtcNow -lt $until)
    if($menus.Count){throw 'Menu did not cancel.'}
    if(([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016) -join ',') -ne [string]$targetIndex){throw 'Right-click menu did not target only the row under the pointer.'}
    Files-Unchanged;Shot 'DCM-20-after-menu-cancel'
    $cases.Add([ordered]@{name='right-click target and cancel preserve fixture bytes';passed=$true})
    # Blank space must not act on the previously selected file.
    $selectedBefore=[DcmNative]::SelectedIndexes($hwnd,$app.Id,1016) -join ','
    [DcmContextProbe]::BlankArea($list,$app.Id,($items.Count-1))
    Start-Sleep -Milliseconds 700
    if(@([DcmContextProbe]::Menus($app.Id)).Count){throw 'Blank list area opened a stale-file context menu.'}
    if(([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016) -join ',') -ne $selectedBefore){throw 'Blank-area click changed the file selection.'}
    Files-Unchanged
    $cases.Add([ordered]@{name='blank area does not open a stale-file menu';passed=$true})

    # Invoke exactly Open on another row; never invoke Run as administrator.
    $null=[DcmNative]::SelectFiles($hwnd,$app.Id,1016,[string[]]@('03-wrong-open.cmd'))
    $openIndex=[Array]::IndexOf($items,'04-external open.cmd')
    if($openIndex -lt 0){throw 'External Open fixture absent.'}
    $null=[DcmContextProbe]::RightClick($list,$app.Id,$openIndex)
    $menu=Wait-Menu
    if(([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016) -join ',') -ne [string]$openIndex){throw 'External Open menu targets the old selection.'}
    Menu-Shot 'DCM-20-external-open-menu' $menu
    if((Test-Path -LiteralPath $openMarker) -or (Test-Path -LiteralPath $wrongMarker)){throw 'Marker appeared before Open was invoked.'}
    $clicked=[DcmMenuActions]::ClickOpen($menu,$app.Id)
    $r['invoked_menu_item']=$clicked
    $until=[DateTime]::UtcNow.AddSeconds(15)
    do{Owned;if(Test-Path -LiteralPath $openMarker){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
    if(-not(Test-Path -LiteralPath $openMarker) -or ([IO.File]::ReadAllText($openMarker)).Trim() -cne $token){throw 'External Open did not execute the intended fixture marker.'}
    if(Test-Path -LiteralPath $wrongMarker){throw 'External Open executed the previously selected wrong file.'}
    if(@([DcmContextProbe]::Menus($app.Id)).Count -or @([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected menu/dialog after external Open.'}
    Files-Unchanged
    $r.external_open_tested=$true
    $r['external_open_evidence']=[ordered]@{target='04-external open.cmd';previous_selection='03-wrong-open.cmd';marker=$openMarker;token=$token;marker_sha256=(Hash $openMarker);wrong_marker_absent=$true;association_changed=$false}
    $cases.Add([ordered]@{name='native Open launches only the right-clicked default-handler fixture';passed=$true})
    Shot 'DCM-20-after-external-open'
    $r.passed=$true
}catch{$r['error']=$_.Exception.Message;$r['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)}
finally{
    try{
        if($null -ne $app){
            $app.Refresh()
            if($app.HasExited){throw 'Unexpected app exit before cleanup.'}
            if($app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $owned.start){throw 'Refusing changed process cleanup.'}
            if($hwnd -ne [IntPtr]::Zero){[DcmContextProbe]::Cancel($hwnd,$app.Id);[DcmNative]::CloseAsync($hwnd,$app.Id)}
            if(-not $app.WaitForExit(15000)){& "$env:SystemRoot\System32\taskkill.exe" /PID $app.Id /T /F | Out-Null;$null=$app.WaitForExit(5000);$owned['forced_stop']=$true;throw 'Owned app required forced shutdown.'}
            $owned['exit_code']=$app.ExitCode;$owned['closed_utc']=[DateTime]::UtcNow.ToString('o')
            if($app.ExitCode -ne 0){throw 'App close returned nonzero.'}
            Files-Unchanged;$r['fixtures_unchanged']=$true;$r['fixture_sha256']=$expected
            $start=[DateTime]::Parse($owned.start).ToUniversalTime();$end=$app.ExitTime.ToUniversalTime()
            $children=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$app.Id) | Where-Object {$_.CreationDate.ToUniversalTime() -ge $start -and $_.CreationDate.ToUniversalTime() -le $end})
            if($children.Count){throw 'Owned children still present; no unknown child terminated.'}
            $owned['children_drained']=$true;$app.Dispose()
        }
    }catch{$r.passed=$false;$r['cleanup_error']=$_.Exception.Message}
    try{if($head){Check-Source;$r['source_unchanged']=$true}}catch{$r.passed=$false;$r['integrity_error']=$_.Exception.Message}
    $r['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $r
    Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $r.passed){exit 1}
exit 0
