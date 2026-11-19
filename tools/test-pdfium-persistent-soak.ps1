[CmdletBinding()]
param(
 [Parameter(Mandatory=$true)][string]$WorkerExe,
 [Parameter(Mandatory=$true)][string]$SmallPdf,
 [Parameter(Mandatory=$true)][string]$LongPdf,
 [Parameter(Mandatory=$true)][string]$RasterPdf,
 [Parameter(Mandatory=$true)][string]$VectorPdf,
 [string]$Root='D:\FlowverseReportsRepo',
 [ValidateRange(40,500)][int]$Iterations=160,
 [ValidateRange(10,100)][int]$Warmup=24,
 [ValidateRange(5,30)][int]$RequestDeadlineSec=15
)
# Exercise one real persistent FlowPDF process, not repeated one-shot workers.
# All generated PNGs and receipts live in a new disposable test directory.
# No user PDFs, production binaries, settings, or other process PIDs are changed.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
Add-Type -AssemblyName System.Drawing
$worker=[IO.Path]::GetFullPath($WorkerExe)
$pdfs=@(
 [ordered]@{name='small';path=[IO.Path]::GetFullPath($SmallPdf);pages=2},
 [ordered]@{name='long';path=[IO.Path]::GetFullPath($LongPdf);pages=300},
 [ordered]@{name='raster';path=[IO.Path]::GetFullPath($RasterPdf);pages=1},
 [ordered]@{name='vector';path=[IO.Path]::GetFullPath($VectorPdf);pages=1}
)
if($Iterations -lt ($Warmup+24)){throw 'At least 24 post-warmup renders required'}
foreach($entry in @(@{name='worker';path=$worker})+$pdfs){
 if(-not(Test-Path -LiteralPath $entry.path -PathType Leaf)){throw ('Missing '+$entry.name+': '+$entry.path)}
}
$runtime='C:\Flowverse\runtime'
if(-not(Test-Path -LiteralPath (Join-Path $runtime 'pdfium\bin\pdfium.dll') -PathType Leaf)){throw 'Managed PDFium runtime DLL missing'}
if(-not(Test-Path -LiteralPath $Root -PathType Container)){throw 'Test output root missing'}
$lab=Join-Path $Root ('dcm-pdfium-persistent-soak-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,6))
$null=New-Item -ItemType Directory -Path $lab
$receipt=[ordered]@{
 schema=1;feature='PERSISTENT-PDFIUM-PROCESS-SOAK';started_utc=[DateTime]::UtcNow.ToString('o');
 worker=$worker;worker_sha256=(Get-FileHash $worker -Algorithm SHA256).Hash;
 pdfs=@($pdfs|ForEach-Object {@{name=$_.name;sha256=(Get-FileHash $_.path -Algorithm SHA256).Hash;bytes=(Get-Item $_.path).Length}});
 iterations=$Iterations;warmup=$Warmup;deadline_seconds=$RequestDeadlineSec;cache_limit_bytes=33554432;
 samples=@();passed=$false;error=$null;lab=$lab
}
$child=$null
$ownedId=0
$ownedStart=[DateTime]::MinValue
$started=[Diagnostics.Stopwatch]::StartNew()
function Measure-Child([string]$stage){
 if($null -eq $child){return $null}
 $child.Refresh()
 if($child.HasExited){return $null}
 return [ordered]@{stage=$stage;private_bytes=[long]$child.PrivateMemorySize64;
  working_set_bytes=[long]$child.WorkingSet64;handles=[int]$child.HandleCount;
  cpu_ms=[Math]::Round($child.TotalProcessorTime.TotalMilliseconds)}
}
function Service([hashtable]$request,[int]$deadline=$RequestDeadlineSec){
 $json=ConvertTo-Json -InputObject $request -Compress -Depth 6
 $child.StandardInput.WriteLine($json)
 $child.StandardInput.Flush()
 $clock=[Diagnostics.Stopwatch]::StartNew()
 $reply=$child.StandardOutput.ReadLineAsync()
 $peakPrivate=0L;$peakWorking=0L;$peakHandles=0
 while(-not $reply.IsCompleted){
  $child.Refresh()
  if($child.HasExited){throw ('Worker exited while waiting for '+$request.id+' exit='+$child.ExitCode)}
  $peakPrivate=[Math]::Max($peakPrivate,[long]$child.PrivateMemorySize64)
  $peakWorking=[Math]::Max($peakWorking,[long]$child.WorkingSet64)
  $peakHandles=[Math]::Max($peakHandles,[int]$child.HandleCount)
  if($clock.Elapsed.TotalSeconds -gt $deadline){throw ('Owned worker response timed out for '+$request.id+' after '+$deadline+'s')}
  Start-Sleep -Milliseconds 25
 }
 $line=$reply.GetAwaiter().GetResult()
 $clock.Stop()
 if([string]::IsNullOrWhiteSpace($line)){throw ('Empty worker response for '+$request.id)}
 $response=$line|ConvertFrom-Json
 if($response.version -ne 1 -or $response.id -cne $request.id){throw ('Protocol ID/version mismatch at '+$request.id+': '+$line)}
 if(-not $response.ok){throw ('PDFium rejected '+$request.id+': '+$response.error)}
 if([long]$response.cacheBytes -gt 33554432){throw ('Worker exceeded 32 MiB cache limit on '+$request.id)}
 $after=Measure-Child $request.id
 if($null -ne $after){
  $peakPrivate=[Math]::Max($peakPrivate,[long]$after.private_bytes)
  $peakWorking=[Math]::Max($peakWorking,[long]$after.working_set_bytes)
  $peakHandles=[Math]::Max($peakHandles,[int]$after.handles)
 }
 return [ordered]@{response=$response;duration_ms=$clock.ElapsedMilliseconds;after=$after;
  sampled_peak_private_bytes=$peakPrivate;sampled_peak_working_bytes=$peakWorking;sampled_peak_handles=$peakHandles}
}
function Median([long[]]$values){
 if($values.Count -lt 1){return 0}
 $sorted=@($values|Sort-Object)
 return [long]$sorted[[int][Math]::Floor(($sorted.Count-1)/2)]
}
try{
 $psi=New-Object Diagnostics.ProcessStartInfo
 $psi.FileName=$worker
 $psi.Arguments='-ServeJSONL'
 $psi.UseShellExecute=$false
 $psi.CreateNoWindow=$true
 $psi.RedirectStandardInput=$true
 $psi.RedirectStandardOutput=$true
 # Avoid a blocked stderr pipe in long tests: diagnostics do not carry protocol.
 $psi.RedirectStandardError=$false
 $psi.EnvironmentVariables['FILEWORKSPACE_RUNTIME_DIR']=$runtime
 $child=New-Object Diagnostics.Process
 $child.StartInfo=$psi
 if(-not $child.Start()){throw 'Unable to launch persistent PDFium worker'}
 $ownedId=$child.Id
 $ownedStart=$child.StartTime.ToUniversalTime()
 $receipt['pid']=$ownedId
 $pong=Service @{version=1;id='soak-ping';command='ping'} 8
 if($pong.response.result -cne 'pong'){throw 'Worker ping failed'}
 $receipt['initial']=Measure-Child 'initial'
 $angles=@(0,90,180,270)
 for($i=0;$i -lt $Iterations;$i++){
  $entry=$pdfs[$i%$pdfs.Count]
  $out=Join-Path $lab ('render-'+$i.ToString('0000')+'.png')
  $cycle=[int][Math]::Floor($i/4)
  $page=if($entry.pages -eq 300){@(0,149,299)[$cycle%3]}elseif($entry.pages -eq 2){$cycle%2}else{0}
  $angle=$angles[$cycle%4]
  $zoom=@(75,100,125,150)[$cycle%4]
  $command=@{version=1;id=('soak-'+$i.ToString('0000'));command='render';inputPath=$entry.path;outputPath=$out;
   mode='zoom';pageIndex=[int]$page;rotation=[int]$angle;zoom=[int]$zoom;viewportWidth=900;viewportHeight=700}
  $round=Service $command
  $meta=([string]$round.response.result).Split('|')
  if($meta.Count -ne 6 -or $meta[0] -cne 'OK'){throw ('Unexpected render metadata at '+$i)}
  if([int]$meta[1] -ne $entry.pages -or [int]$meta[2] -ne $page){throw ('Wrong document/page at '+$i)}
  if(-not(Test-Path -LiteralPath $out -PathType Leaf)){throw ('Missing PNG at '+$i)}
  $img=[Drawing.Image]::FromFile($out)
  try{
   if($img.Width -ne [int]$meta[3] -or $img.Height -ne [int]$meta[4]){throw ('PNG dimensions mismatch at '+$i)}
   $size=(Get-Item -LiteralPath $out).Length
  }finally{$img.Dispose()}
  Remove-Item -LiteralPath $out -Force
  $after=$round.after
  if($null -eq $after){throw ('Worker vanished after render '+$i)}
  $receipt.samples+=@([ordered]@{index=$i;document=$entry.name;page=$page;rotation=$angle;zoom=$zoom;
   duration_ms=$round.duration_ms;png_bytes=$size;cache_bytes=[long]$round.response.cacheBytes;
   private_bytes=[long]$after.private_bytes;working_set_bytes=[long]$after.working_set_bytes;
   handles=[int]$after.handles;sampled_peak_private_bytes=[long]$round.sampled_peak_private_bytes;
   sampled_peak_working_bytes=[long]$round.sampled_peak_working_bytes;
   sampled_peak_handles=[int]$round.sampled_peak_handles})
  if(($i+1)%40 -eq 0){Write-Output ('RENDERS_COMPLETED='+($i+1)+' / '+$Iterations+' PRIVATE_MB='+[Math]::Round($after.private_bytes/1MB,1)+' HANDLES='+$after.handles)}
 }
 foreach($entry in $pdfs){
  $now=(Get-FileHash -LiteralPath $entry.path -Algorithm SHA256).Hash
  $expected=@($receipt.pdfs|Where-Object {$_.name -eq $entry.name})[0].sha256
  if($now -cne $expected){throw ('Input PDF changed: '+$entry.name)}
  $handle=[IO.File]::Open($entry.path,[IO.FileMode]::Open,[IO.FileAccess]::ReadWrite,[IO.FileShare]::None)
  $handle.Dispose()
 }
 if(@(Get-ChildItem -LiteralPath $lab -Filter '*.png' -File).Count){throw 'Rendered PNGs remain after cleanup'}
 $first=@($receipt.samples|Where-Object {$_.index -ge $Warmup -and $_.index -lt ($Warmup+32)})
 $last=@($receipt.samples|Select-Object -Last 32)
 $firstPrivate=Median ([long[]]@($first|ForEach-Object {$_.private_bytes}))
 $lastPrivate=Median ([long[]]@($last|ForEach-Object {$_.private_bytes}))
 $firstHandles=Median ([long[]]@($first|ForEach-Object {$_.handles}))
 $lastHandles=Median ([long[]]@($last|ForEach-Object {$_.handles}))
 $maxPrivate=0L;$maxHandles=0;$maxCache=0L
 foreach($sample in $receipt.samples){
  # Samples are ordered dictionaries. Measure-Object -Property does not
  # reliably expose dictionary keys as PowerShell object properties.
  $maxPrivate=[Math]::Max($maxPrivate,[long]$sample.sampled_peak_private_bytes)
  $maxHandles=[Math]::Max($maxHandles,[int]$sample.sampled_peak_handles)
  $maxCache=[Math]::Max($maxCache,[long]$sample.cache_bytes)
 }
 $deltaPrivate=$lastPrivate-$firstPrivate
 $deltaHandles=$lastHandles-$firstHandles
 $receipt['summary']=[ordered]@{
  first_post_warmup_private_median_bytes=$firstPrivate;last_private_median_bytes=$lastPrivate;
  delta_private_bytes=$deltaPrivate;first_post_warmup_handles_median=$firstHandles;
  last_handles_median=$lastHandles;delta_handles=$deltaHandles;
  sampled_peak_private_bytes=[long]$maxPrivate;sampled_peak_handles=[int]$maxHandles;
  maximum_cache_bytes=[long]$maxCache;duration_seconds=[Math]::Round($started.Elapsed.TotalSeconds,2);
  unique_worker_pids=1;peak_memory_is_sampled=$true
 }
 # Conservative diagnostic bounds: after 24 warmup renders there must
 # not be sustained +48 MiB private memory or +32 handles across 32
 # samples; 256 MiB observed private and 1000 handles are hard safety stops.
 if($deltaPrivate -gt 48MB){throw ('Potential private-byte growth after warmup: '+[Math]::Round($deltaPrivate/1MB,1)+' MiB')}
 if($deltaHandles -gt 32){throw ('Potential handle leak after warmup: '+$deltaHandles)}
 if($maxPrivate -gt 256MB -or $maxHandles -gt 1000){throw 'Excessive worker footprint; inspect receipt'}
 $shutdown=Service @{version=1;id='soak-shutdown';command='shutdown'} 8
 if($shutdown.response.result -cne 'bye'){throw 'Worker rejected normal shutdown'}
 $child.StandardInput.Close()
 if(-not $child.WaitForExit(8000)){throw 'Worker did not exit after shutdown'}
 if($child.ExitCode -ne 0){throw ('Worker exit code '+$child.ExitCode)}
 $receipt['normal_shutdown']=$true
 $receipt.passed=$true
}catch{
 $receipt.error=$_.Exception.Message
 Write-Output ('FAIL='+$receipt.error)
}finally{
 $receipt['completed_utc']=[DateTime]::UtcNow.ToString('o')
 if($null -ne $child){
  try{
   $child.Refresh()
   if(-not $child.HasExited){
    if($child.Id -eq $ownedId -and $child.StartTime.ToUniversalTime() -eq $ownedStart){
     # Failure-only cleanup of the exact worker process launched here.
     $child.Kill()
     $null=$child.WaitForExit(5000)
    }
   }
   $receipt['worker_exited']=$child.HasExited
  }catch{$receipt['cleanup_error']=$_.Exception.Message;$receipt.passed=$false}
  $child.Dispose()
 }
 [IO.File]::WriteAllText((Join-Path $lab 'receipt.json'),($receipt|ConvertTo-Json -Depth 12))
 Write-Output ('REPORT='+$lab)
 Write-Output ('PASSED='+$receipt.passed)
 if($receipt.Contains('summary')){
  Write-Output ('PRIVATE_MEDIAN_DELTA_MIB='+[Math]::Round($receipt.summary.delta_private_bytes/1MB,2))
  Write-Output ('HANDLE_MEDIAN_DELTA='+$receipt.summary.delta_handles)
  Write-Output ('SAMPLED_PEAK_PRIVATE_MIB='+[Math]::Round($receipt.summary.sampled_peak_private_bytes/1MB,2))
  Write-Output ('SAMPLED_PEAK_HANDLES='+$receipt.summary.sampled_peak_handles)
 }
}
if(-not $receipt.passed){exit 1}
