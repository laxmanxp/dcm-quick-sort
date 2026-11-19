# Creates only disposable, clearly marked qualification content under Root.
function New-DcmFixtures([string]$Root) {
    $utf8 = New-Object Text.UTF8Encoding($false)
    $inbox=Join-Path $Root 'inbox'
    $vault=Join-Path $Root 'Vault'
    $destination=Join-Path $vault 'Accepted'
    $local=Join-Path $Root 'profile\local'
    $roaming=Join-Path $Root 'profile\roaming'
    $config=Join-Path $local 'DCM Quick Sort'
    New-Item -ItemType Directory -Path $inbox,$destination,$config,$roaming | Out-Null
    foreach($name in @('01-alpha.txt','02-bravo.txt','03-charlie.txt')) {
        [IO.File]::WriteAllText((Join-Path $inbox $name),("DCM disposable native qualification fixture`r`nName: " + $name + "`r`nExact bytes must survive Move and Undo.`r`n"),$utf8)
    }
    [IO.File]::SetLastWriteTimeUtc((Join-Path $inbox '03-charlie.txt'),[DateTime]::UtcNow.AddDays(-60))
    $settings=[ordered]@{default_source=$inbox;pdf_default_view='fit_width';vault_root=$vault}
    [IO.File]::WriteAllText((Join-Path $config 'settings.json'),($settings | ConvertTo-Json),$utf8)
    [IO.File]::WriteAllText((Join-Path $config 'destinations.conf'),("Acceptance=Accepted`r`n"),$utf8)
    # Deterministic, standards-simple two-page PDF. Large center colour panels are
    # visible at Fit Width, 100%, and 110%; the screen oracle never uses footer text
    # as a substitute for visible document pixels.
    $streams=@(
        "0.02 0.55 0.65 rg 60 95 492 590 re f`n1 1 1 rg BT /F1 26 Tf 88 620 Td (DCM PDF PREVIEW) Tj 0 -45 Td (PAGE ONE - TEAL) Tj 0 -55 Td (VISIBLE DOCUMENT CONTENT) Tj ET`n0 0 0 rg BT /F1 14 Tf 75 45 Td (Disposable qualification fixture - page 1 of 2) Tj ET",
        "0.90 0.35 0.08 rg 60 95 492 590 re f`n1 1 1 rg BT /F1 26 Tf 88 620 Td (DCM PDF PREVIEW) Tj 0 -45 Td (PAGE TWO - ORANGE) Tj 0 -55 Td (PAGE NAVIGATION VERIFIED) Tj ET`n0 0 0 rg BT /F1 14 Tf 75 45 Td (Disposable qualification fixture - page 2 of 2) Tj ET"
    )
    $objects=@(
        '<< /Type /Catalog /Pages 2 0 R >>',
        '<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>',
        '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 6 0 R >>',
        '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 7 0 R >>',
        '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>',
        ('<< /Length '+[Text.Encoding]::ASCII.GetByteCount($streams[0])+" >>`nstream`n"+$streams[0]+"`nendstream"),
        ('<< /Length '+[Text.Encoding]::ASCII.GetByteCount($streams[1])+" >>`nstream`n"+$streams[1]+"`nendstream")
    )
    $pdf=New-Object Text.StringBuilder
    [void]$pdf.Append("%PDF-1.4`n")
    $offsets=New-Object 'Collections.Generic.List[int]'
    for($i=0;$i -lt $objects.Count;$i++){
        $offsets.Add([Text.Encoding]::ASCII.GetByteCount($pdf.ToString()))
        [void]$pdf.Append(($i+1).ToString()+" 0 obj`n"+$objects[$i]+"`nendobj`n")
    }
    $xref=[Text.Encoding]::ASCII.GetByteCount($pdf.ToString())
    [void]$pdf.Append("xref`n0 8`n0000000000 65535 f `n")
    foreach($offset in $offsets){[void]$pdf.Append($offset.ToString('0000000000')+" 00000 n `n")}
    [void]$pdf.Append("trailer`n<< /Size 8 /Root 1 0 R >>`nstartxref`n"+$xref+"`n%%EOF`n")
    [IO.File]::WriteAllBytes((Join-Path $inbox '08-preview.pdf'),[Text.Encoding]::ASCII.GetBytes($pdf.ToString()))
    return [pscustomobject]@{inbox=$inbox;vault=$vault;destination=$destination;local=$local;roaming=$roaming;config=$config;logical_destination='Accepted'}
}
