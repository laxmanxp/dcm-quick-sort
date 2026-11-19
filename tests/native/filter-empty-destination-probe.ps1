# Diagnostic-only reproduction of DCM-21's historical invalid test setup.
# Dot-sourced ONLY when DCM21_PROBE_EMPTY_DESTINATION=1. Never an acceptance PASS.
# No modal is dismissed or answered; the normal runner owns cleanup and keeps RED.
Assert-Owned
if(@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016)).Count -ne 0 -or
   @([DcmNative]::SelectedIndexes($script:hwnd,$script:app.Id,1016)).Count -ne 0 -or
   [DcmNative]::Text([DcmNative]::Control($script:hwnd,$script:app.Id,1005),$script:app.Id) -ne ''){
    throw 'Diagnostic requires the verified no-match state without a filename draft.'
}
if([DcmNative]::Text([DcmNative]::Control($script:hwnd,$script:app.Id,1003),$script:app.Id) -ne ''){
    throw 'Diagnostic refuses to replace a nonempty destination.'
}
$receipt['diagnostic_only']=$true
$receipt['feature_accepted']=$false
[DcmNative]::ClickAsync($script:hwnd,$script:app.Id,1041)
$until=[DateTime]::UtcNow.AddSeconds(5)
$dialogs=@()
do {
    Assert-Owned
    $dialogs=@([DcmNative]::Dialogs($script:hwnd,$script:app.Id))
    if($dialogs.Count){break}
    Start-Sleep -Milliseconds 100
} while([DateTime]::UtcNow -lt $until)
if($dialogs.Count -ne 1 -or $dialogs[0].Text -ne 'DCM - Quick Sort'){
    throw 'Diagnostic expected one owned validation dialog; no response sent.'
}
$controls=@([DcmNative]::Snapshot([IntPtr]$dialogs[0].Handle,$script:app.Id))
$messages=@($controls | Where-Object {$_.Visible -and $_.Class -eq 'Static'} | ForEach-Object {$_.Text})
$expected='Choose an existing logical location inside the configured Vault.'
$matched=@($messages | Where-Object {$_.StartsWith($expected,[StringComparison]::Ordinal)}).Count -eq 1
Assert-Fixtures
$evidence=[ordered]@{diagnostic_only=$true;feature_accepted=$false;trigger='Move with no selected file and empty destination';dialog_title=$dialogs[0].Text;controls=$controls;expected_prefix=$expected;expected_message_observed=$matched;fixtures_unchanged=$true;dialog_answered=$false}
Write-Json (Join-Path $lab 'dcm21-empty-destination-probe.json') $evidence
$receipt['empty_destination_probe']=$evidence
if(-not $matched){throw 'Owned dialog differs from expected destination validation; inspect captured controls.'}
throw 'DCM21_DIAGNOSTIC_RED: blank destination validation reproduced; not a filter acceptance run.'
