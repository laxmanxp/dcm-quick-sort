# Build the actual QuickSort setup/uninstaller candidate without installing it.
# Requires the current source checkout, Go, and a staged Explorer package.
# Does NOT write Windows registry keys or replace the installed QuickSort.
[CmdletBinding()]
param(
 [Parameter(Mandatory=$true)][string]$PackageDir,
 [Parameter(Mandatory=$true)][string]$OutputDir
)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
$repo=Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$appSource=Join-Path $repo 'app-src'
$installerSource=Join-Path $repo 'installer-src'
$verify=Join-Path $PSScriptRoot 'verify-installer-candidate.ps1'
if(-not (Test-Path -LiteralPath (Join-Path $appSource 'go.mod'))){throw 'QuickSort application source not found'}
if(-not (Test-Path -LiteralPath (Join-Path $installerSource 'go.mod'))){throw 'QuickSort installer source not found'}
$null=& $verify -PackageDir $PackageDir
$output=[IO.Path]::GetFullPath($OutputDir)
if(Test-Path -LiteralPath $output){throw 'Use a new, empty output directory to avoid overwriting an existing build'}
New-Item -Path $output -ItemType Directory | Out-Null
$source=Join-Path $output 'installer-source'
New-Item -Path $source -ItemType Directory | Out-Null
Copy-Item -Path (Join-Path $installerSource '*') -Destination $source -Recurse -Force
$payload=Join-Path $source 'payload'
New-Item -Path $payload -ItemType Directory -Force | Out-Null
$app=Join-Path $payload 'DCM-QuickSort.exe'
Push-Location $appSource
try {
 & go build -o $app .
 if($LASTEXITCODE -ne 0){throw 'QuickSort app build failed'}
} finally { Pop-Location }
foreach($name in @('DCMQuickSortExplorerSend.dll','DCMQuickSortExplorerSend.exe')){
 Copy-Item -LiteralPath (Join-Path $PackageDir ('payload\'+$name)) -Destination (Join-Path $payload $name)
}
Push-Location $source
try {
 & go test ./... -count=1
 if($LASTEXITCODE -ne 0){throw 'Installer tests failed'}
 $setup=Join-Path $output 'DCM-QuickSort-Setup-Candidate.exe'
 & go build -o $setup .
 if($LASTEXITCODE -ne 0){throw 'Installer build failed'}
 & $setup '--verify-release-bundle'
 if($LASTEXITCODE -ne 0){throw 'Installer embedded bundle self-check failed'}
 $uninstall=Join-Path $output 'Uninstall DCM Quick Sort.exe'
 Copy-Item -LiteralPath $setup -Destination $uninstall
 & $uninstall '--verify-release-bundle'
 if($LASTEXITCODE -ne 0){throw 'Uninstaller embedded bundle self-check failed'}
 Write-Output ('Setup candidate: '+$setup)
 Write-Output ('Uninstaller candidate: '+$uninstall)
 Write-Output 'Candidate built and verified; no install, uninstall or registry changes performed.'
} finally {Pop-Location}
