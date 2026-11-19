[CmdletBinding()]
param()
# DCM-19 discovery only. Never install, enable features, change registry, create
# users, launch a VM, or reinterpret this receipt as Explorer UI acceptance.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 is required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT){throw 'FLOWDEV_WORKSPACE_ROOT is required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$workspace=[IO.Path]::GetFullPath($env:FLOWDEV_WORKSPACE_ROOT)
$report=Join-Path $workspace ('dcm-quicksort-explorer-isolation-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($report.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Reports must be outside the candidate.'}
New-Item -ItemType Directory -Path $report | Out-Null
$issues=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;app='dcm-quicksort';feature='DCM-19';scope='read-only isolation discovery, not Explorer acceptance';report=$report;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;feature_accepted=$false;explorer_clicked=$false;host_mutations_performed=$false;issues=$issues}
$head='';$before=@()
Write-Output ('REPORT_ROOT='+$report)
function Git-Value([string[]]$Arguments){
    $value=(& git.exe -C $repo @Arguments | Out-String).Trim()
    if($LASTEXITCODE -ne 0){throw 'Git identity query failed.'}
    return $value
}
function Read-Identity {
    return @('tools/check-explorer-isolation.ps1','tools/flowdev-dcm/config/checks.json','installer-src/core.go' | ForEach-Object {
        [ordered]@{path=$_;sha256=(Get-FileHash -LiteralPath (Join-Path $repo $_) -Algorithm SHA256).Hash.ToLowerInvariant()}
    })
}
function Add-Issue([string]$Stage,$ErrorRecord){
    $issues.Add([ordered]@{stage=$Stage;type=$ErrorRecord.Exception.GetType().FullName;reason=$ErrorRecord.Exception.Message})
}
try{
    $head=Git-Value @('rev-parse','HEAD')
    if((Git-Value @('symbolic-ref','--short','HEAD')) -ne 'release/v1.0.71' -or (Git-Value @('remote','get-url','origin')) -ne 'https://github.com/laxmanxp/dcm-quick-sort.git'){throw 'Unexpected candidate branch or origin.'}
    if(Git-Value @('status','--porcelain=v1','--untracked-files=all')){throw 'Candidate must be clean.'}
    $receipt['commit']=$head
    $before=Read-Identity
    $receipt['observed_source_files']=$before
    try{
        $os=Get-CimInstance -ClassName Win32_OperatingSystem -Property Caption,Version,BuildNumber,FreePhysicalMemory -OperationTimeoutSec 10
        $receipt['os']=[ordered]@{caption=$os.Caption;version=$os.Version;build=$os.BuildNumber;free_physical_bytes=([uint64]$os.FreePhysicalMemory*1024)}
    }catch{Add-Issue 'os' $_}
    try{
        $cs=Get-CimInstance -ClassName Win32_ComputerSystem -Property HypervisorPresent,TotalPhysicalMemory -OperationTimeoutSec 10
        $receipt['virtualization']=[ordered]@{hypervisor_present=$cs.HypervisorPresent;total_physical_bytes=[uint64]$cs.TotalPhysicalMemory}
    }catch{Add-Issue 'virtualization' $_}
    $featureNames=@('Containers-DisposableClientVM','Microsoft-Hyper-V-All','Microsoft-Hyper-V','VirtualMachinePlatform')
    $states=@()
    foreach($name in $featureNames){
        try{
            $rows=@(Get-CimInstance -ClassName Win32_OptionalFeature -Filter ("Name='"+$name+"'") -Property Name,InstallState -OperationTimeoutSec 10)
            if($rows.Count -gt 1){throw 'Optional-feature identity is ambiguous.'}
            if($rows.Count -eq 1){
                $state=[int]$rows[0].InstallState
                $label=switch($state){1{'enabled'} 2{'disabled'} 3{'absent'} default{'unknown'}}
                $states+=[ordered]@{name=$name;install_state=$state;state=$label;query_succeeded=$true}
            }else{$states+=[ordered]@{name=$name;state='not_returned';query_succeeded=$true}}
        }catch{Add-Issue ('feature:'+ $name) $_;$states+=[ordered]@{name=$name;state='query_failed';query_succeeded=$false}}
    }
    $receipt['optional_features']=$states
    $tools=@()
    foreach($name in @('WindowsSandbox.exe','wsb.exe','vmconnect.exe','VBoxManage.exe','vmrun.exe')){
        $commands=@(Get-Command $name -CommandType Application -ErrorAction SilentlyContinue)
        $tools+=[ordered]@{name=$name;found_on_path=($commands.Count -gt 0)}
    }
    $receipt['launchers']=$tools
    $receipt['system_sandbox_executable_exists']=Test-Path -LiteralPath (Join-Path $env:SystemRoot 'System32\WindowsSandbox.exe') -PathType Leaf
    $receipt['hyperv_module_available']=(@(Get-Module -ListAvailable -Name Hyper-V).Count -gt 0)
    $receipt['vmms_service_present']=(@(Get-Service -Name vmms -ErrorAction SilentlyContinue).Count -gt 0)
    $receipt['approved_disposable_account_or_vm']='not configured by this DCM profile; no host account or existing VM assumed disposable'
    $sandbox=@($states | Where-Object {$_['name'] -eq 'Containers-DisposableClientVM' -and $_['state'] -eq 'enabled'})
    $receipt['sandbox_feature_enabled']=($sandbox.Count -eq 1)
    $receipt['isolation_status']=if($sandbox.Count -eq 1){'sandbox_feature_enabled_runtime_not_verified'}else{'no_ready_isolation_verified'}
    $receipt['next_action']=if($sandbox.Count -eq 1){'Prepare an explicitly owned isolated launch; verify runtime and resource readiness before any registration or Explorer UI test.'}else{'Keep DCM-19 pending. Provision or identify an approved disposable Windows environment before actual Explorer clicks; do not enable features, reboot, or use host registration as a substitute.'}
    $receipt['discovery_status']=if($issues.Count -eq 0){'complete_for_listed_probes'}else{'partial; unknown probes must not be treated as absence'}
    $receipt.passed=$true
}catch{$receipt['error']=$_.Exception.Message;Write-Output ('FAIL='+$_.Exception.Message)}
finally{
    try{
        if($head){
            $after=Read-Identity
            $receipt['source_unchanged']=((Git-Value @('rev-parse','HEAD')) -eq $head -and -not (Git-Value @('status','--porcelain=v1','--untracked-files=all')) -and (($before | ConvertTo-Json -Depth 5 -Compress) -ceq ($after | ConvertTo-Json -Depth 5 -Compress)))
            if(-not $receipt.source_unchanged){throw 'Source changed during discovery.'}
        }
    }catch{$receipt.passed=$false;$receipt['integrity_error']=$_.Exception.Message}
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    [IO.File]::WriteAllText((Join-Path $report 'receipt.json'),($receipt | ConvertTo-Json -Depth 15),(New-Object Text.UTF8Encoding($false)))
    Write-Output ('RECEIPT='+(Join-Path $report 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
