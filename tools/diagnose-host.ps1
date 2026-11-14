# Read-only host observation. No process command lines, termination, product
# execution, registry changes, or source changes are performed by this selector.
[CmdletBinding()]
param([ValidateRange(5,120)][int]$BudgetSeconds=30)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT'){throw 'This diagnostic is for the native Windows host.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT){throw 'FLOWDEV_WORKSPACE_ROOT is required.'}
$report=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-host-diagnostics-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
New-Item -ItemType Directory -Path $report | Out-Null
$utf8=New-Object Text.UTF8Encoding($false)
$clock=[Diagnostics.Stopwatch]::StartNew()
$issues=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{
    schema=1;app='dcm-quicksort';scope='read-only host/resource observation; no product acceptance claim'
    report=$report;runner_pid=$PID;started_utc=[DateTime]::UtcNow.ToString('o');budget_seconds=$BudgetSeconds
    logical_processors=[Environment]::ProcessorCount;process_filter=@('go','git','flowdev-test','powershell','DCM-QS*')
    ownership_note='Process names identify diagnostic candidates; ownership is not inferred from names.'
    passed=$false;issues=$issues
}
Write-Output ('REPORT_ROOT='+$report)
function Write-Json([string]$Path,$Value){[IO.File]::WriteAllText($Path,($Value | ConvertTo-Json -Depth 20),$utf8)}
function Checkpoint([string]$Phase){
    Write-Json (Join-Path $report 'active-step.tmp') ([ordered]@{schema=1;runner_pid=$PID;phase=$Phase;updated_utc=[DateTime]::UtcNow.ToString('o');elapsed_seconds=[Math]::Round($clock.Elapsed.TotalSeconds,3)})
    Move-Item -LiteralPath (Join-Path $report 'active-step.tmp') -Destination (Join-Path $report 'active-step.json') -Force
}
function Within-Budget([string]$Stage){
    if($clock.Elapsed.TotalSeconds -lt $BudgetSeconds){return $true}
    $issues.Add([ordered]@{stage=$Stage;reason='observation budget exhausted; step was not started'})
    return $false
}
function Process-Snapshot {
    $items=@()
    # Get-Process reads only these requested names. Do not query Path, Modules,
    # Responding, WMI CommandLine, or any other process's arguments/environment.
    $found=@(Get-Process -Name 'go','git','flowdev-test','powershell','DCM-QS*' -ErrorAction SilentlyContinue | Sort-Object Id -Unique)
    foreach($process in $found){
        $item=[ordered]@{name=$process.ProcessName;pid=$process.Id;observed_utc=[DateTime]::UtcNow.ToString('o')}
        try{
            $item['cpu_seconds']=$process.TotalProcessorTime.TotalSeconds
            $item['working_set_bytes']=$process.WorkingSet64
            $item['private_bytes']=$process.PrivateMemorySize64
            $item['started_utc']=$process.StartTime.ToUniversalTime().ToString('o')
        }catch{$item['observation_error']=$_.Exception.GetType().FullName}
        finally{$process.Dispose()}
        $items+=$item
    }
    return $items
}
try{
    Checkpoint 'drive-capacity'
    if(Within-Budget 'drive-capacity'){
        $step=[Diagnostics.Stopwatch]::StartNew()
        try{
            $drive=New-Object IO.DriveInfo('C:\')
            $receipt['c_drive']=[ordered]@{name=$drive.Name;ready=$drive.IsReady;total_bytes=$drive.TotalSize;available_free_bytes=$drive.AvailableFreeSpace;total_free_bytes=$drive.TotalFreeSpace}
        }catch{$issues.Add([ordered]@{stage='drive-capacity';reason=$_.Exception.GetType().FullName})}
        finally{$receipt['drive_query_seconds']=[Math]::Round($step.Elapsed.TotalSeconds,3)}
    }
    Checkpoint 'physical-memory'
    if(Within-Budget 'physical-memory'){
        $step=[Diagnostics.Stopwatch]::StartNew()
        try{
            # Framework ComputerInfo wraps GlobalMemoryStatusEx; it does not run
            # WMI, performance-counter sampling, a compiler, or a child process.
            Add-Type -AssemblyName Microsoft.VisualBasic
            $memory=New-Object Microsoft.VisualBasic.Devices.ComputerInfo
            $receipt['memory']=[ordered]@{total_physical_bytes=$memory.TotalPhysicalMemory;available_physical_bytes=$memory.AvailablePhysicalMemory;total_virtual_bytes=$memory.TotalVirtualMemory;available_virtual_bytes=$memory.AvailableVirtualMemory;method='Microsoft.VisualBasic.Devices.ComputerInfo / GlobalMemoryStatusEx'}
        }catch{$issues.Add([ordered]@{stage='physical-memory';reason=$_.Exception.GetType().FullName})}
        finally{$receipt['memory_query_seconds']=[Math]::Round($step.Elapsed.TotalSeconds,3)}
    }
    Checkpoint 'requested-process-snapshot'
    if(Within-Budget 'requested-process-snapshot'){
        $firstTime=[DateTime]::UtcNow
        $first=@(Process-Snapshot)
        $receipt['processes']=$first
        Write-Json (Join-Path $report 'processes-first.json') $first
        if(Within-Budget 'requested-process-cpu-sample'){
            Start-Sleep -Milliseconds 1000
            Checkpoint 'requested-process-cpu-sample'
            $secondTime=[DateTime]::UtcNow
            $second=@(Process-Snapshot)
            $elapsed=($secondTime-$firstTime).TotalSeconds
            $sample=@()
            foreach($item in $second){
                $entry=[ordered]@{name=$item.name;pid=$item.pid;observed_utc=$item.observed_utc}
                if($item.Contains('cpu_seconds') -and $item.Contains('started_utc')){
                    $entry['cpu_seconds']=$item.cpu_seconds;$entry['working_set_bytes']=$item.working_set_bytes;$entry['private_bytes']=$item.private_bytes;$entry['started_utc']=$item.started_utc
                    $before=@($first | Where-Object {$_['pid'] -eq $item.pid -and $_.Contains('started_utc') -and $_['started_utc'] -eq $item.started_utc})
                    if($before.Count -eq 1 -and $before[0].Contains('cpu_seconds') -and $elapsed -gt 0){
                        $delta=$item.cpu_seconds-$before[0].cpu_seconds
                        $entry['cpu_delta_seconds']=[Math]::Round($delta,4)
                        $entry['sample_elapsed_seconds']=[Math]::Round($elapsed,3)
                        $entry['approx_percent_of_logical_cpu_capacity']=[Math]::Round(100*$delta/$elapsed/$receipt.logical_processors,2)
                    }
                }else{$entry['observation_error']='process changed/exited or attributes unavailable'}
                $sample+=$entry
            }
            $receipt['process_sample']=$sample
        }
    }
    $receipt['data_status']=if($issues.Count -eq 0){'complete'}else{'partial; inspect issues'}
    $receipt.passed=$true
}catch{$receipt['error']=$_.Exception.Message;Write-Output ('FAIL='+$_.Exception.Message)}
finally{
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    $receipt['elapsed_seconds']=[Math]::Round($clock.Elapsed.TotalSeconds,3)
    $receipt['budget_note']='Budget prevents starting additional steps; direct local OS queries cannot be cancelled in-process. No diagnostic child process is spawned.'
    Write-Json (Join-Path $report 'receipt.json') $receipt
    Checkpoint 'finished'
    Write-Output ('RECEIPT='+(Join-Path $report 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
