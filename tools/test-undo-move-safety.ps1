[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if(-not $env:FLOWDEV_WORKSPACE_ROOT){throw 'FLOWDEV_WORKSPACE_ROOT required'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$head=(& git.exe -C $repo rev-parse HEAD | Out-String).Trim()
if($LASTEXITCODE -ne 0){throw 'Cannot read committed HEAD'}
$dirty=(& git.exe -C $repo status --porcelain --untracked-files=all | Out-String).Trim()
if($LASTEXITCODE -ne 0 -or $dirty){throw 'Committed clean candidate required'}
if($env:FLOWDEV_SOURCE_COMMIT -and $env:FLOWDEV_SOURCE_COMMIT -ne $head){throw 'FlowDev source pin mismatch'}
$id=[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8)
$root=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-undo-move-unit-'+$id)
New-Item -ItemType Directory -Path $root -ErrorAction Stop|Out-Null
$log=Join-Path $root 'go-tests.log'
$code=1
try{
    Push-Location (Join-Path $repo 'app-src')
    try{
        $env:GOWORK='off'
        & go.exe test -count=1 -run '^TestUndoMoveWithRecordedFingerprintRejectsChangedDestination$' ./... 2>&1 | Tee-Object -FilePath $log
        $code=$LASTEXITCODE
    }finally{Pop-Location}
}finally{
    $bytes=[Text.Encoding]::UTF8.GetBytes(($head+'|'+$code))
    $r=[ordered]@{app='dcm-quicksort';selector='undo-move-safety-unit';commit=$head;passed=($code -eq 0);exit_code=$code;log=$log;log_sha256=if(Test-Path $log){(Get-FileHash -LiteralPath $log -Algorithm SHA256).Hash.ToLowerInvariant()}else{''};source_unchanged=$false}
    $after=(& git.exe -C $repo rev-parse HEAD | Out-String).Trim()
    $dirtyAfter=(& git.exe -C $repo status --porcelain --untracked-files=all | Out-String).Trim()
    $r.source_unchanged=($after -eq $head -and -not $dirtyAfter)
    if(-not $r.source_unchanged){$r.passed=$false}
    $receipt=Join-Path $root 'receipt.json'
    [IO.File]::WriteAllText($receipt,($r|ConvertTo-Json -Depth 5),(New-Object Text.UTF8Encoding($false)))
    Write-Output ('RECEIPT='+$receipt)
}
if($code -ne 0){exit $code}
if(-not $r.source_unchanged){exit 1}
