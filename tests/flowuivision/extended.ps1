. (Join-Path $PSScriptRoot 'common.ps1')
function Observation([string]$Name){return (Get-Content (Join-Path $report ($Name+'.json')) -Raw -Encoding UTF8|ConvertFrom-Json).Observations.main}
[void](Run-Case 'pptx-open' @'
click clear
type name "07-slides"
wait_ready type=textbox target=newname text="07-slides" timeout=5s
delay 1200ms
inspect main
expect slides visible
expect slides count=2
'@ $null)
$slideItems=@((Observation 'pptx-open'|Where-Object ID -eq '1025').Items)
if($slideItems.Count -eq 2){
 [void](Run-Case 'pptx-slide2' ("select slides `""+$slideItems[1]+"`"`ndelay 900ms`ninspect main`nexpect slides text=`""+$slideItems[1]+"`"") $null)
 [void](Run-Case 'pptx-slide1-return' ("select slides `""+$slideItems[0]+"`"`ndelay 900ms`ninspect main`nexpect slides text=`""+$slideItems[0]+"`"") $null)
}
[void](Run-Case 'xlsx-sheet2' @'
click clear
type name "06-orders"
wait_ready type=textbox target=newname text="06-orders" timeout=5s
delay 1000ms
inspect main
expect sheets visible
select sheets "Read me"
delay 700ms
inspect main
expect sheets text="Read me"
'@ $null)
[void](Run-Case 'pdf-reset' @'
click clear
type name "08-print"
wait_ready type=textbox target=newname text="08-print" timeout=5s
delay 1500ms
click zoomreset
delay 2000ms
inspect main
expect pdfstatus text~="100%"
'@ $null)
[void](Run-Case 'pdf-zoom-plus' @'
click zoomin
delay 2000ms
inspect main
expect pdfstatus not_empty
'@ $null)
[void](Run-Case 'pdf-zoom-minus' @'
click zoomout
delay 2000ms
inspect main
expect pdfstatus text~="100%"
'@ $null)
[void](Run-Case 'pdf-fit-width' @'
click width
delay 2000ms
inspect main
expect pdfstatus text~="Fit Width"
'@ $null)
# Tool coverage probe: do not confuse a missing FlowUIVision ListBox adapter
# with a product failure. No mutation is performed by this probe.
[void](Run-Case 'multiselect-tool-coverage' @'
click clear
select files "0,1"
inspect main
'@ $null)
$source=Join-Path $Lab 'inbox\01-alpha.txt'
$target=Join-Path $Lab 'vault\01-alpha.txt'
if(-not(Test-Path $source)-or(Test-Path $target)){throw 'Unsafe/unexpected move fixture state'}
$hash=(Get-FileHash $source).Hash
$destination=Join-Path $Lab 'vault'
$moveBody="click clear`ntype name `"01-alpha`"`nwait_ready type=textbox target=newname text=`"01-alpha`" timeout=5s`ntype destination `"$destination`"`ninspect main`nexpect destination text=`"$destination`"`nclick move`ndelay 1000ms`ninspect main`nexpect undo enabled"
$moved=Run-Case 'move-to-vault' $moveBody {if((Test-Path $source)-or(-not(Test-Path $target))){throw 'Move did not relocate fixture to the specified test vault'};if((Get-FileHash $target).Hash -ne $hash){throw 'Move changed content'}}
if($moved){[void](Run-Case 'undo-move' @'
click undo
delay 900ms
inspect main
'@ {if((-not(Test-Path $source))-or(Test-Path $target)){throw 'Undo did not restore moved fixture'};if((Get-FileHash $source).Hash -ne $hash){throw 'Undo move changed content'}})}
$statuses=@();foreach($name in @('pdf-reset','pdf-zoom-plus','pdf-zoom-minus','pdf-fit-width')){$statuses+=@{case=$name;status=@(Observation $name|Where-Object Text -like 'DCM PDF Preview*'|Select-Object -ExpandProperty Text)}}
$statuses|ConvertTo-Json -Depth 5|Set-Content (Join-Path $report 'pdf-statuses.json') -Encoding UTF8
Write-Output ('REPORT='+$report)
