[CmdletBinding()]
param(
 [Parameter(Mandatory=$true)][string]$WorkerExe,
 [Parameter(Mandatory=$true)][string]$ValidPdf,
 [string]$Root='D:\FlowverseReportsRepo',
 [int]$Iterations=20
)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($Iterations -lt 5 -or $Iterations -gt 60){throw 'Iterations outside 5..60'}
$exe=[IO.Path]::GetFullPath($WorkerExe);$pdf=[IO.Path]::GetFullPath($ValidPdf)
if(-not (Test-Path $exe -PathType Leaf) -or -not(Test-Path $pdf -PathType Leaf)){throw 'Worker or fixture missing'}
$lab=Join-Path $Root ('dcm-pdfium-worker-memory-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,6))
$null=New-Item -ItemType Directory -Path $lab
$receipt=[ordered]@{schema=1;feature='PDFIUM-WORKER-MEMORY';started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;worker_sha256=(Get-FileHash $exe -Algorithm SHA256).Hash;fixture_sha256=(Get-FileHash $pdf -Algorithm SHA256).Hash;iterations=$Iterations;samples=@();lab=$lab;error=$null}
try{
 for($i=0;$i -lt $Iterations;$i++){
  $out=Join-Path $lab ('render-'+$i.ToString('000')+'.png')
  $psi=New-Object Diagnostics.ProcessStartInfo
  $psi.FileName=$exe
  $psi.UseShellExecute=$false
  $psi.RedirectStandardOutput=$true
  $psi.RedirectStandardError=$true
  $psi.CreateNoWindow=$true
  $psi.Arguments='-InputPath "'+$pdf+'" -OutputPath "'+$out+'" -PageIndex '+($i%2)+' -Mode zoom -Zoom '+(100+($i%5)*20)+' -ViewportWidth 800 -ViewportHeight 600'
  $p=New-Object Diagnostics.Process;$p.StartInfo=$psi
  if(-not $p.Start()){throw 'Could not start worker'}
  $pidOwned=$p.Id;$started=$p.StartTime.ToUniversalTime().ToString('o')
  $clock=[Diagnostics.Stopwatch]::StartNew()
  $maxPrivate=0L;$maxWorking=0L
  while(-not $p.WaitForExit(20)){
   $p.Refresh()
   $maxPrivate=[Math]::Max($maxPrivate,[long]$p.PrivateMemorySize64)
   $maxWorking=[Math]::Max($maxWorking,[long]$p.WorkingSet64)
   if($clock.Elapsed.TotalSeconds -gt 25){
    # Failure-only teardown; terminate only the exact worker this test started.
    $p.Refresh()
    if(-not $p.HasExited -and $p.Id -eq $pidOwned -and $p.StartTime.ToUniversalTime().ToString('o') -eq $started){$p.Kill();$null=$p.WaitForExit(5000)}
    throw ('Owned worker exceeded 25-second deadline at iteration '+$i)
   }
  }
  $clock.Stop();$p.Refresh()
  $maxPrivate=[Math]::Max($maxPrivate,[long]$p.PeakPagedMemorySize64)
  $maxWorking=[Math]::Max($maxWorking,[long]$p.PeakWorkingSet64)
  $stdout=$p.StandardOutput.ReadToEnd().Trim();$stderr=$p.StandardError.ReadToEnd().Trim()
  if($p.ExitCode -ne 0 -or $stdout -notmatch '^OK\|'){throw ('Failed render '+$i+' exit='+$p.ExitCode+' output='+$stderr+' '+$stdout)}
  if(-not (Test-Path $out -PathType Leaf)){throw ('Missing PNG on iteration '+$i)}
  $sample=[ordered]@{index=$i;pid=$pidOwned;duration_ms=$clock.ElapsedMilliseconds;peak_private_bytes=$maxPrivate;peak_working_bytes=$maxWorking;result=$stdout;png_bytes=(Get-Item $out).Length;normal_exit=$true}
  $receipt.samples+=@($sample)
  Remove-Item -LiteralPath $out -Force
  $p.Dispose()
 }
 if(@(Get-ChildItem -LiteralPath $lab -Filter '*.png').Count){throw 'Leftover rendered PNGs'}
 $receipt.passed=$true
}catch{
 $receipt.error=$_.Exception.Message
 Write-Output ('FAIL='+$receipt.error)
}finally{
 $receipt['completed_utc']=[DateTime]::UtcNow.ToString('o')
 [IO.File]::WriteAllText((Join-Path $lab 'receipt.json'),($receipt|ConvertTo-Json -Depth 10))
 $durations=@($receipt.samples|ForEach-Object {$_.duration_ms}|Sort-Object)
 if($durations.Count){Write-Output ('SAMPLES='+$durations.Count+' MEDIAN_MS='+$durations[[int][Math]::Floor(($durations.Count-1)/2)]+' MAX_MS='+$durations[-1])}
 $privateMax=0L;foreach($sample in $receipt.samples){$privateMax=[Math]::Max($privateMax,[long]$sample.peak_private_bytes)}
 Write-Output ('PEAK_PRIVATE_MB='+[Math]::Round($privateMax/1MB,1))
 Write-Output ('REPORT='+$lab);Write-Output ('PASSED='+$receipt.passed)
}
if(-not $receipt.passed){exit 1}
