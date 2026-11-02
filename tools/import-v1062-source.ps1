param(
  [Parameter(Mandatory=$true)][string]$Archive,
  [string]$Checkout='C:\FlowverseReports\dcm-quicksort-v1062-certification'
)
$ErrorActionPreference='Stop'
if(-not (Test-Path -LiteralPath $Archive)){throw "Source archive missing: $Archive"}
if(-not (Test-Path -LiteralPath (Join-Path $Checkout '.git'))){throw "Missing isolated clone: $Checkout"}
$branch=(& git -C $Checkout branch --show-current).Trim()
if($branch -ne 'recovery/v1.0.62-source'){throw "Refusing to modify branch $branch"}
if(& git -C $Checkout status --porcelain){throw 'Clone is dirty: abort rather than overwrite files'}
$stage=Join-Path $env:TEMP ('dcm-v1062-import-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $stage|Out-Null
try {
  Expand-Archive -LiteralPath $Archive -DestinationPath $stage
  $app=Join-Path $stage 'app-src'
  if(-not (Test-Path $app)) {throw 'Unexpected archive layout: app-src missing'}
  $fileCount=@(Get-ChildItem -LiteralPath $stage -Recurse -File).Count
  if($fileCount -lt 50){throw "Only $fileCount files: invalid source package"}
  foreach($part in @('app-src','installer-src')) {
    $src=Join-Path $stage $part
    $dst=Join-Path $Checkout ('source-v1.0.62\'+$part)
    New-Item -ItemType Directory -Force -Path $dst|Out-Null
    Get-ChildItem -LiteralPath $src -Force | Copy-Item -Destination $dst -Recurse -Force
    foreach($f in Get-ChildItem -LiteralPath $src -Recurse -File) {
      $rel=$f.FullName.Substring($src.Length+1)
      $copy=Join-Path $dst $rel
      if((Get-FileHash -LiteralPath $f.FullName -Algorithm SHA256).Hash -ne (Get-FileHash -LiteralPath $copy -Algorithm SHA256).Hash){throw "Hash mismatch $part/$rel"}
    }
  }
  Write-Output "COPY_VERIFIED source_files=$fileCount"
  Write-Output "Next: run go test in source-v1.0.62/app-src and installer-src, then commit/push only after PASS"
} finally {Remove-Item -LiteralPath $stage -Recurse -Force -ErrorAction SilentlyContinue}
