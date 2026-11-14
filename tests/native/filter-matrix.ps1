# DCM-21 extension of the existing owned ui-filters runner. No host files or
# configuration are used. Expectations are explicit, not derived from app output.
$allNames=@('01-alpha.txt','02-bravo.txt','03-charlie.txt','08-preview.pdf')
$recentNames=@('01-alpha.txt','02-bravo.txt','08-preview.pdf')
function Check-DcmFilter([string]$Name,[string]$Commands,[string[]]$Names){
    Invoke-Fvs $Name ($Commands+"`ninspect main")
    Wait-For $Name { (@([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016) | Sort-Object) -join '|') -ceq (@($Names | Sort-Object) -join '|') }
    Assert-List $Name $Names
    Assert-Fixtures
    if($Names.Count -eq 0){
        if(@([DcmNative]::SelectedIndexes($script:hwnd,$script:app.Id,1016)).Count -ne 0){throw 'Filtered-empty retained a selected row.'}
        if([DcmNative]::Text([DcmNative]::Control($script:hwnd,$script:app.Id,1005),$script:app.Id) -ne ''){throw 'Filtered-empty retained a filename draft.'}
        if(-not (Status-Contains 'No files match the current filters')){throw 'DCM-UI-EMPTY-COPY-01: filtered view lacks truthful no-match header.'}
    }
}
$categoryCases=@('All files','Images','Documents','Video','Audio','Archives','Programs','Code & text','Design & CAD','Other / unknown')
for($i=0;$i -lt $categoryCases.Count;$i++){
    $category=$categoryCases[$i];$names=@();if($category -in @('All files','Documents')){$names=$allNames}
    Check-DcmFilter ('DCM-21-category-'+$i) ("click clear`nselect category `""+$category+"`"") $names
}
Check-DcmFilter 'DCM-21-extension-txt' "click clear`nselect extension `".txt`"" @('01-alpha.txt','02-bravo.txt','03-charlie.txt')
Check-DcmFilter 'DCM-21-extension-pdf' "click clear`nselect extension `".pdf`"" @('08-preview.pdf')
Check-DcmFilter 'DCM-21-extension-absent' "click clear`nselect extension `".jpg`"" @()
$ageCases=@('All ages','Last 24 hours','Last 7 days','Last 30 days','Older than 30 days')
for($i=0;$i -lt $ageCases.Count;$i++){
    $age=$ageCases[$i];$names=$recentNames
    if($age -eq 'All ages'){$names=$allNames}elseif($age -eq 'Older than 30 days'){$names=@('03-charlie.txt')}
    Check-DcmFilter ('DCM-21-age-'+$i) ("click clear`nselect age `""+$age+"`"") $names
}
$sizeCases=@('All sizes','Under 1 MiB','1-10 MiB','10-100 MiB','100 MiB+')
for($i=0;$i -lt $sizeCases.Count;$i++){
    $size=$sizeCases[$i];$names=@();if($i -le 1){$names=$allNames}
    Check-DcmFilter ('DCM-21-size-'+$i) ("click clear`nselect size `""+$size+"`"") $names
}
Check-DcmFilter 'DCM-21-name-case-insensitive' "click clear`ntype name `"ALPHA.TXT`"" @('01-alpha.txt')
Check-DcmFilter 'DCM-21-combined-intersection' "click clear`nselect category `"Documents`"`nselect extension `".txt`"`nselect age `"Older than 30 days`"`nselect size `"Under 1 MiB`"`ntype name `"CHARLIE`"" @('03-charlie.txt')
Save-State 'DCM-21-combined-match';$null=Save-Shot 'DCM-21-combined-match'
Check-DcmFilter 'DCM-21-combined-no-match' "type name `"no-dcm21-fixture-matches`"" @()
Save-State 'DCM-21-filtered-empty';$null=Save-Shot 'DCM-21-filtered-empty'
# Opt-in evidence probe reproduces the old missing-destination setup, then fails
# deliberately. Normal qualification never acknowledges unexpected dialogs.
if($env:DCM21_PROBE_EMPTY_DESTINATION -eq '1'){
    . (Join-Path $repo 'tests\native\filter-empty-destination-probe.ps1')
}
# Use the known valid disposable destination to isolate no-selection safety
# from the separate invalid-destination validation dialog (DCM-38).
Invoke-Fvs 'DCM-21-valid-disposable-destination' "type destination `"Accepted`"`ninspect main`nexpect destination text=`"Accepted`""
# The hidden charlie fixture must not become a stale Rename or Move target.
foreach($button in @(1006,1041,1007)){
    [DcmNative]::ClickAsync($script:hwnd,$script:app.Id,$button)
    Start-Sleep -Milliseconds 300
    Wait-For ('no-selection action settled '+$button) { @([DcmNative]::ListItems($script:hwnd,$script:app.Id,1016)).Count -eq 0 }
    Assert-Owned;Assert-Fixtures
    Assert-List ('DCM-21-no-selection-action-'+$button) @()
    if([DcmNative]::Text([DcmNative]::Control($script:hwnd,$script:app.Id,1005),$script:app.Id) -ne ''){throw 'No-selection action restored an actionable draft.'}
}
$journal=Join-Path $fixture.config 'recovery\move-journal.jsonl'
if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Filter-only workflow appended a file operation.'}
Check-DcmFilter 'DCM-21-clear-restores-all' "click clear" $allNames
foreach($pair in @(@(1030,''),@(1012,'All files'),@(1013,'All extensions'),@(1031,'All ages'),@(1032,'All sizes'))){
    if([DcmNative]::Text([DcmNative]::Control($script:hwnd,$script:app.Id,[int]$pair[0]),$script:app.Id) -ne [string]$pair[1]){throw ('Clear Filters did not reset control '+$pair[0])}
}
$null=[DcmNative]::SelectFiles($script:hwnd,$script:app.Id,1016,[string[]]@('01-alpha.txt'))
Wait-For 'restored filename draft' { [DcmNative]::Text([DcmNative]::Control($script:hwnd,$script:app.Id,1005),$script:app.Id) -eq '01-alpha' }
Save-State 'DCM-21-clear-restored';$null=Save-Shot 'DCM-21-clear-restored'
$cases.Add([ordered]@{name='DCM-21 filter safety and clear-control reset';passed=$true;source_files=4;operation_journal_empty=$true;scope='all category/age/size options; representative extensions; name/intersection; unit tests cover numeric boundaries'})
