# Infrastructure gate: parse committed scripts and compile the owned adapter.
# This does not launch the product, run an installer, or claim UI acceptance.
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 tooling gate required.'}
if(-not $env:FLOWDEV_WORKSPACE_ROOT){throw 'FLOWDEV_WORKSPACE_ROOT is required.'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$report=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-tooling-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
New-Item -ItemType Directory -Path $report | Out-Null
$receipt=[ordered]@{schema=1;app='dcm-quicksort';scope='tooling syntax and fixture construction only';report=$report;passed=$false;started_utc=[DateTime]::UtcNow.ToString('o')}
Write-Output ('REPORT_ROOT='+$report)
try{
    $head=(& git.exe -C $repo rev-parse HEAD | Out-String).Trim()
    if($LASTEXITCODE -ne 0){throw 'Cannot read candidate HEAD.'}
    $receipt['commit']=$head
    $parsed=@()
    foreach($path in @(Get-ChildItem -LiteralPath (Join-Path $repo 'tools'),(Join-Path $repo 'tests') -Filter '*.ps1' -File -Recurse)){
        $tokens=$null;$errors=$null
        $null=[Management.Automation.Language.Parser]::ParseFile($path.FullName,[ref]$tokens,[ref]$errors)
        $parsed+=[ordered]@{path=$path.FullName;sha256=(Get-FileHash -LiteralPath $path.FullName -Algorithm SHA256).Hash.ToLowerInvariant();errors=@($errors | ForEach-Object {$_.ToString()})}
        if($errors.Count){$receipt['scripts']=$parsed;throw "PowerShell syntax error: $($path.FullName)"}
    }
    $receipt['scripts']=$parsed
    $null=Get-Content -LiteralPath (Join-Path $repo 'tools\flowdev-dcm\config\checks.json') -Raw | ConvertFrom-Json
    Add-Type -AssemblyName System.Drawing
    Add-Type -Path (Join-Path $repo 'tests\native\dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-scroll-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    $receipt['scroll_probe_compiled']=$true
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-label-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    $receipt['label_probe_compiled']=$true
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-xlsx-input.cs')
    $receipt['xlsx_input_compiled']=$true
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-pptx-input.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
    $receipt['pptx_input_compiled']=$true
    # Exercise the exact-name input boundary without targeting a real window.
    # Invalid names must fail before any HWND lookup; both allowed fixtures must
    # still fail the ownership guard when given the zero handle.
    $nativeBindingGuards=@()
    foreach($name in @('', '07-slides.PPT', '07-slides.ppt.exe', '08-slides.ppt', '..\07-slides.ppt', 'C:\07-slides.ppt', '07-slides.pptx.tmp')){
        $caught=''
        try{$null=[DcmPptxInput]::BindNativeSlide([IntPtr]::Zero,0,$name)}catch{$caught=$_.Exception.ToString()}
        if(-not $caught.Contains('Only the pinned native PPTX or PPT fixture is allowed')){throw ('Native binding accepted an unpinned name or missed the name guard: '+$name)}
        $nativeBindingGuards+=[ordered]@{fixture_name=$name;expected_guard='exact_fixture_name';passed=$true}
    }
    foreach($name in @('07-slides.pptx','07-slides.ppt')){
        $caught=''
        try{$null=[DcmPptxInput]::BindNativeSlide([IntPtr]::Zero,0,$name)}catch{$caught=$_.Exception.ToString()}
        if(-not $caught.Contains('PPTX control ownership mismatch')){throw ('Native binding failed to retain the HWND ownership guard: '+$name)}
        $nativeBindingGuards+=[ordered]@{fixture_name=$name;expected_guard='owned_host';passed=$true}
    }
    $receipt['native_presentation_binding_guards']=$nativeBindingGuards
    Add-Type -Path (Join-Path $repo 'tests/native/dcm-preview-geometry.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
    $receipt['preview_geometry_compiled']=$true
    $receipt['native_adapter_compiled']=$true
    . (Join-Path $repo 'tests\native\new-fixtures.ps1')
    $fixture=New-DcmFixtures $report
    $settings=Get-Content -LiteralPath (Join-Path $fixture.config 'settings.json') -Raw | ConvertFrom-Json
    if($settings.vault_root -ne $fixture.vault -or $settings.default_source -ne $fixture.inbox -or -not(Test-Path -LiteralPath $fixture.destination -PathType Container)){throw 'Fixture does not configure its own logical Vault destination.'}
    if(([IO.File]::ReadAllText((Join-Path $fixture.config 'destinations.conf'))).Trim() -ne 'Acceptance=Accepted'){throw 'Destination favorite fixture does not use the product Name=LogicalLocation format.'}
    $receipt['fixtures']=@(Get-ChildItem -LiteralPath $fixture.inbox -File | ForEach-Object {[ordered]@{name=$_.Name;size=$_.Length;sha256=(Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()}})
    if($receipt.fixtures.Count -ne 4){throw 'Unexpected fixture count.'}
    $receipt.passed=$true
}catch{$receipt['error']=$_.Exception.Message;Write-Output ('FAIL='+$_.Exception.Message)}
finally{
    $receipt['finished_utc']=[DateTime]::UtcNow.ToString('o')
    [IO.File]::WriteAllText((Join-Path $report 'receipt.json'),($receipt | ConvertTo-Json -Depth 20),(New-Object Text.UTF8Encoding($false)))
    Write-Output ('RECEIPT='+(Join-Path $report 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
exit 0
