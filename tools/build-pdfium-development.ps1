[CmdletBinding()]
param(
  [Parameter(Mandatory=$true)][string]$FileWorkspaceCheckout,
  [string]$OutputDirectory = '',
  [switch]$RunPolicyTests
)
# Development-only build: source stays in GitHub; test machines compile
# from checked-out commits and never edit the source directly.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest

$quicksort=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$workspace=[IO.Path]::GetFullPath($FileWorkspaceCheckout)
if(-not (Test-Path (Join-Path $quicksort 'app-src\go.mod') -PathType Leaf)){throw 'Quick Sort module not found'}
if(-not (Test-Path (Join-Path $workspace 'go.mod') -PathType Leaf)){throw 'File Workspace module not found'}
$qBranch=(& git -C $quicksort branch --show-current).Trim()
$fBranch=(& git -C $workspace branch --show-current).Trim()
if($LASTEXITCODE -ne 0){throw 'Cannot identify File Workspace git branch'}
if($qBranch -ne 'feature/shared-flowpdf-pdfium-provider'){throw ('Refusing development build from unexpected Quick Sort branch: '+$qBranch)}
if($fBranch -ne 'feature/flowpdf-shared-provider-contract'){throw ('Refusing development build from unexpected File Workspace branch: '+$fBranch)}

if([string]::IsNullOrWhiteSpace($OutputDirectory)){
  $OutputDirectory=Join-Path $quicksort 'out\pdfium-dev'
}
$out=[IO.Path]::GetFullPath($OutputDirectory)
if($out -eq $quicksort -or $out -eq $workspace){throw 'Output must be a separate build directory'}
$null=New-Item -ItemType Directory -Force -Path $out
$worker=Join-Path $out 'flowpdf-preview-worker-persistent.exe'
$app=Join-Path $out 'DCMQuickSort-pdfium-dev.exe'

$previousCGO=$env:CGO_ENABLED
try{
  $env:CGO_ENABLED='0'
  Push-Location $workspace
  try{
    & go build -trimpath -o $worker ./cmd/flowpdf-preview-worker
    if($LASTEXITCODE -ne 0){throw 'FlowPDF worker build failed'}
  }finally{Pop-Location}

  Push-Location (Join-Path $quicksort 'app-src')
  try{
    if($RunPolicyTests){
      & go test -tags pdfium_dev -run '^TestPDFProvider|^TestPersistentPDFiumOptInIsStrict' -count=1
      if($LASTEXITCODE -ne 0){throw 'Development default policy tests failed'}
    }
    & go build -trimpath -tags pdfium_dev -o $app .
    if($LASTEXITCODE -ne 0){throw 'Quick Sort PDFium development build failed'}
  }finally{Pop-Location}
}finally{
  $env:CGO_ENABLED=$previousCGO
}

$runtimeRoot=$env:FILEWORKSPACE_RUNTIME_DIR
if([string]::IsNullOrWhiteSpace($runtimeRoot)){$runtimeRoot='C:\Flowverse\runtime'}
$dll=Join-Path $runtimeRoot 'pdfium\bin\pdfium.dll'
$manifest=[ordered]@{
  schema=1
  mode='development-persistent-pdfium-default'
  build_tag='pdfium_dev'
  built_utc=[DateTime]::UtcNow.ToString('o')
  quicksort_commit=(& git -C $quicksort rev-parse HEAD).Trim()
  fileworkspace_commit=(& git -C $workspace rev-parse HEAD).Trim()
  application=$app
  application_sha256=(Get-FileHash $app -Algorithm SHA256).Hash
  worker=$worker
  worker_sha256=(Get-FileHash $worker -Algorithm SHA256).Hash
  runtime_dll_available=(Test-Path $dll -PathType Leaf)
  runtime_root=$runtimeRoot
  release_default_changed=$false
}
$manifestPath=Join-Path $out 'pdfium-development-manifest.json'
[IO.File]::WriteAllText($manifestPath,($manifest|ConvertTo-Json -Depth 5))
Write-Output ('DEV_APP='+$app)
Write-Output ('DEV_WORKER='+$worker)
Write-Output ('RUNTIME_DLL_AVAILABLE='+$manifest.runtime_dll_available)
Write-Output ('MANIFEST='+$manifestPath)
if(-not $manifest.runtime_dll_available){
  Write-Warning 'Managed PDFium runtime missing: this development build will default to Windows.Data.Pdf until the runtime becomes available.'
}
