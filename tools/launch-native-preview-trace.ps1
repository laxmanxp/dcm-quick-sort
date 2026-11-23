param(
 [string]$Build = "C:\DCMQuickSortCollisionAcceptance\dcm-viewport-resize-dev.exe",
 [string]$Fixture = "C:\DCMQuickSortCollisionAcceptance\tests\fixtures\dcm33-legacy.source.docx",
 [string]$TraceRoot = "C:\DCMQuickSortCollisionAcceptance\TraceDOCX",
 [ValidateSet("off","aware")][string]$WordRectMode="off"
)
$ErrorActionPreference="Stop"
New-Item -ItemType Directory -Force $TraceRoot | Out-Null
$copy = Join-Path $TraceRoot "native-word-fixture.docx"
Copy-Item -LiteralPath $Fixture -Destination $copy -Force
if (-not (Test-Path $Build)) { throw "Missing development build: $Build" }
$env:DCM_WORD_COM_RECT_MODE=$WordRectMode
$env:DCM_LAYOUT_TRACE="1"
$env:TEMP=$TraceRoot
$env:TMP=$TraceRoot
$process = Start-Process -FilePath $Build -ArgumentList @($copy) -WorkingDirectory (Split-Path $Build) -PassThru
Write-Output ("Diagnostic QuickSort PID: {0}" -f $process.Id)
Write-Output ("Fixture: {0}; Word mode: {1}" -f $copy,$WordRectMode)
Write-Output ("Trace on graceful exit: {0}" -f (Join-Path $TraceRoot "dcm-layout-trace.json"))
