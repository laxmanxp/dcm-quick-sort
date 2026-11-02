param(
    [Parameter(Mandatory=$true)][string]$InputPath,
    [Parameter(Mandatory=$true)][string]$OutputPath,
    [int]$PageIndex = 0,
    [ValidateSet('fit','zoom')][string]$Mode = 'fit',
    [int]$Zoom = 100,
    [int]$ViewportWidth = 1000,
    [int]$ViewportHeight = 1000
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

Add-Type -AssemblyName System.Runtime.WindowsRuntime
[Windows.Storage.StorageFile, Windows.Storage, ContentType=WindowsRuntime] | Out-Null
[Windows.Storage.FileAccessMode, Windows.Storage, ContentType=WindowsRuntime] | Out-Null
[Windows.Storage.Streams.IRandomAccessStream, Windows.Storage.Streams, ContentType=WindowsRuntime] | Out-Null
[Windows.Data.Pdf.PdfDocument, Windows.Data.Pdf, ContentType=WindowsRuntime] | Out-Null
[Windows.Data.Pdf.PdfPageRenderOptions, Windows.Data.Pdf, ContentType=WindowsRuntime] | Out-Null

$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and $_.IsGenericMethod -and $_.GetParameters().Count -eq 1
})[0]
$asTaskAction = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and -not $_.IsGenericMethod -and $_.GetParameters().Count -eq 1
})[0]

function Await-Operation($operation, [Type]$resultType) {
    $method = $script:asTaskGeneric.MakeGenericMethod($resultType)
    $task = $method.Invoke($null, @($operation))
    $task.Wait()
    return $task.Result
}

function Await-Action($action) {
    $task = $script:asTaskAction.Invoke($null, @($action))
    $task.Wait()
}

$inputFull = [System.IO.Path]::GetFullPath($InputPath)
$outputFull = [System.IO.Path]::GetFullPath($OutputPath)
$outputDir = [System.IO.Path]::GetDirectoryName($outputFull)
if (-not [string]::IsNullOrWhiteSpace($outputDir)) {
    [System.IO.Directory]::CreateDirectory($outputDir) | Out-Null
}

$storageFile = Await-Operation ([Windows.Storage.StorageFile]::GetFileFromPathAsync($inputFull)) ([Windows.Storage.StorageFile])
$pdf = Await-Operation ([Windows.Data.Pdf.PdfDocument]::LoadFromFileAsync($storageFile)) ([Windows.Data.Pdf.PdfDocument])
$pageCount = [int]$pdf.PageCount
if ($PageIndex -lt 0 -or $PageIndex -ge $pageCount) {
    throw "Page index $PageIndex is outside document page count $pageCount"
}

$page = $pdf.GetPage([uint32]$PageIndex)
try {
    $pageWidth = [double]$page.Size.Width
    $pageHeight = [double]$page.Size.Height
    if ($pageWidth -le 0 -or $pageHeight -le 0) {
        throw 'PDF page reported an invalid size'
    }

    if ($Mode -eq 'fit') {
        $vw = [Math]::Max(64, $ViewportWidth)
        $vh = [Math]::Max(64, $ViewportHeight)
        $scale = [Math]::Min($vw / $pageWidth, $vh / $pageHeight)
        if ($scale -le 0) { $scale = 1.0 }
    } else {
        $scale = [Math]::Max(0.10, [Math]::Min(8.0, $Zoom / 100.0))
    }

    $destWidth = [Math]::Max(1, [int][Math]::Round($pageWidth * $scale))
    $destHeight = [Math]::Max(1, [int][Math]::Round($pageHeight * $scale))

    [System.IO.File]::WriteAllBytes($outputFull, [byte[]]@())
    $outputFile = Await-Operation ([Windows.Storage.StorageFile]::GetFileFromPathAsync($outputFull)) ([Windows.Storage.StorageFile])
    $stream = Await-Operation ($outputFile.OpenAsync([Windows.Storage.FileAccessMode]::ReadWrite)) ([Windows.Storage.Streams.IRandomAccessStream])
    try {
        $options = New-Object Windows.Data.Pdf.PdfPageRenderOptions
        $options.DestinationWidth = [uint32]$destWidth
        $options.DestinationHeight = [uint32]$destHeight
        Await-Action ($page.RenderToStreamAsync($stream, $options))
        $flush = $stream.FlushAsync()
        $null = Await-Operation $flush ([bool])
    } finally {
        if ($stream -is [System.IDisposable]) { $stream.Dispose() }
    }

    Write-Output ("OK|{0}|{1}|{2}|{3}" -f $pageCount, $PageIndex, $destWidth, $destHeight)
} finally {
    if ($page -is [System.IDisposable]) { $page.Dispose() }
}
