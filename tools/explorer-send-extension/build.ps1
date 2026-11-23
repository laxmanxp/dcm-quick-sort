<#
Build the QuickSort classic Explorer extension and Go batch launcher.
No registry writes, Explorer restarts, system installation or file moves.
Requires Go and llvm-mingw clang++ available on PATH.
#>
[CmdletBinding()]
param(
 [string]$OutDir=(Join-Path $PSScriptRoot 'build')
)
$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$source=Join-Path $PSScriptRoot 'quicksort_send.cpp'
$helper=Join-Path $root 'tools\explorer-send-helper'
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$dll=Join-Path $OutDir 'DCMQuickSortExplorerSend.dll'
$exe=Join-Path $OutDir 'DCMQuickSortExplorerSend.exe'
& clang++ -std=c++17 -O2 -static -shared -o $dll $source -lole32 -lshell32 -luuid -ladvapi32
if($LASTEXITCODE -ne 0){throw 'Native Explorer extension compilation failed'}
Push-Location $helper
try{
 & go test ./... -count=1
 if($LASTEXITCODE -ne 0){throw 'Explorer launcher tests failed'}
 & go build -o $exe .
 if($LASTEXITCODE -ne 0){throw 'Explorer launcher build failed'}
}finally{Pop-Location}
$imports=@(& llvm-objdump -p $dll | Select-String 'DLL Name:' | ForEach-Object { $_.Line.Trim() })
if($LASTEXITCODE -ne 0){throw 'Could not inspect DLL dependencies'}
if(($imports -join [Environment]::NewLine) -match 'libc\+\+\.dll|libunwind\.dll'){throw 'Non-portable LLVM runtime dependency detected'}
Write-Output "Built DLL: $dll"
Write-Output "Built helper: $exe"
Write-Output "DLL imported libraries:"
$imports
Write-Output 'No Explorer registry keys were changed.'
