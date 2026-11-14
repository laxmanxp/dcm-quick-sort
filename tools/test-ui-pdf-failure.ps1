[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT)
# DCM-34: one header-valid, unloadable PDF through the default Windows.Data.Pdf
# renderer and its natural Windows fallback. No registry changes, fallback
# suppression, foreign-window input, synthetic repaint or cancellation claim.
# Machine checks and original image review are deliberately separate results.
$ErrorActionPreference='Stop';Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required'}
if(-not $BuildReceipt -or -not $env:FLOWDEV_WORKSPACE_ROOT){throw 'Exact build and workspace required'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$damagedSource=Join-Path $repo 'tests/fixtures/dcm34-damaged.pdf'
$fixtureManifest=Join-Path $repo 'tests/fixtures/dcm34-damaged.manifest.json'
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-pdf-failure-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Test lab must be outside source'}
$null=New-Item -ItemType Directory $lab;$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-34';format_scope='default_PDF_damage_and_recovery_only';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;machine_checks_passed=$false;feature_accepted=$false;visual_accepted=$false;visual_review_required=$true;user_test='not_recorded';cases=$cases;captures=$shots;observations=(New-Object 'Collections.Generic.List[object]');native_error_control_readiness_accepted=$false;interruption_or_cancellation_tested=$false;txt_pixel_contract='initial_baseline_difference_le_30'}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$head='';$manifest=@();$fixture=$null;$expected=@{};$exe='';$build=$null;$selectedName='';$controlRefs=@();$basePaneHandles=@{};$registrations=$null
$fallbackFooter='DCM PDF renderer unavailable; using Windows PDF preview fallback'
$unavailablePrefix='PDF preview unavailable: '
$validFooterPrefix='DCM PDF Preview '+[char]0xB7+' Page 1/2 '+[char]0xB7+' Fit Width '+[char]0xB7+' '
$validFooterPattern='^'+[regex]::Escape($validFooterPrefix)+'[0-9]+%$'
$sizingKeys=@('zoom_out','zoom_actual','zoom_in','fit','width')
# Complete explicit error phrases only, read from the owned preview subtree.
# A generic word such as "error" or the host's fallback footer is insufficient.
$nativeErrorPhrases=@("This file can't be previewed","We can't open this file",'This file cannot be previewed','This file could not be previewed','This file could not be opened','This document could not be opened','Failed to load PDF document','Unable to load PDF','Unable to open this document','There was an error opening this document','This file is damaged','The file is damaged','This document is damaged','The document is damaged','Invalid PDF file','Invalid PDF document','PDF preview unavailable')
function Hash([string]$p){return (Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Json([string]$p,$value){[IO.File]::WriteAllText($p,(ConvertTo-Json -InputObject $value -Depth 30),$utf8)}
function Git([string[]]$arguments){$s=(& git.exe -C $repo @arguments|Out-String).Trim();if($LASTEXITCODE){throw 'Git query failed'};return $s}
function Source {
 if((Git @('rev-parse','HEAD')) -ne $head -or (Git @('status','--porcelain=v1','--untracked-files=all'))){throw 'Source identity changed'}
 foreach($f in $manifest){$p=[IO.Path]::GetFullPath((Join-Path $repo $f.path));if(-not $p.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $p) -ne $f.sha256){throw 'Source manifest changed'}}
}
function OwnProcess {$app.Refresh();if($app.HasExited -or $app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $owned.start){throw 'Owned host identity changed'}}
function Own {OwnProcess;[DcmNative]::RequireOwned($hwnd,$app.Id)}
function NoDialog {Own;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected owned dialog'}}
function Wait([string]$name,[scriptblock]$predicate){$until=[DateTime]::UtcNow.AddSeconds(25);do{NoDialog;if(& $predicate){return};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until);throw ('Deadline: '+$name)}
function Files {
 if(-not $fixture){return};$all=@(Get-ChildItem -LiteralPath $fixture.inbox -File)
 if($all.Count -ne $expected.Count){throw 'Fixture names changed'}
 foreach($f in $all){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Fixture content changed'}}
 if(@(Get-ChildItem -LiteralPath $fixture.destination -Force).Count){throw 'Destination changed'}
 $journal=Join-Path $fixture.config 'recovery\move-journal.jsonl';if((Test-Path -LiteralPath $journal) -and ([IO.File]::ReadAllText($journal)).Trim()){throw 'Unexpected operation journal'}
}
function Record([string]$name,$detail){Files;$cases.Add([ordered]@{name=$name;passed=$true;detail=$detail})}
function SelectFile([string]$name){
 NoDialog
 if($name -cnotin @('01-alpha.txt','08-preview.pdf','10-damaged.pdf')){throw 'Only the three pinned qualification selections are allowed'}
 $items=@([DcmNative]::ListItems($hwnd,$app.Id,1016));$index=[Array]::IndexOf($items,$name);if($index -lt 0){throw 'Missing fixture in list'}
 [DcmUndoInput]::Select($hwnd,$app.Id,$index);$stem=[IO.Path]::GetFileNameWithoutExtension($name)
 Wait ('selection '+$name) {([DcmUndoInput]::TryText($hwnd,$app.Id,1005)) -ceq $stem}
 $selected=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016));if($selected.Count -ne 1 -or $selected[0] -ne $index){throw 'Selected fixture index mismatch'}
 $script:selectedName=$name
}
function HostCaptions {return @([DcmDocxProbe]::HostStatics($hwnd,$app.Id)|ForEach-Object {[DcmNative]::Text([IntPtr]$_,$app.Id)})}
function BindDefaultControls {
 # Bind once while the initial TXT is visible, before any foreign handler can
 # attach. Hidden control text/Enabled values are retained by the application.
 $nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id))
 foreach($node in $nodes){if($node.PID -ne $app.Id){throw 'Foreign preview child before the first PDF'};$basePaneHandles[[string]$node.Handle]=$node}
 $refs=@();$names=@('previous','next','zoom_out','zoom_actual','zoom_in','fit','width');$ids=@(1060,1061,1020,1021,1022,1023,1047)
 for($i=0;$i -lt $ids.Count;$i++){$h=[DcmNative]::Control($hwnd,$app.Id,$ids[$i]);if([DcmNative]::ClassName($h) -ine 'Button'){throw 'Default PDF/sizing control is not a Button'};$refs+=[pscustomobject]@{name=$names[$i];id=$ids[$i];handle=$h.ToInt64();class='Button'}}
 $labels=@($nodes|Where-Object {$_.Class -ieq 'Static' -and $_.Name -ceq 'Page'})
 if($labels.Count -ne 1){throw 'Cannot uniquely bind the initial hidden PDF page label'}
 $refs+=[pscustomobject]@{name='page';id=0;handle=$labels[0].Handle;class='Static'};$script:controlRefs=$refs
 $receipt['default_control_bindings']=$refs;$receipt['initial_owned_preview_children']=$nodes
}
function Controls {
 $result=[ordered]@{}
 foreach($r in $controlRefs){$h=[IntPtr]$r.handle;[DcmNative]::RequireOwned($h,$app.Id);if([DcmNative]::ClassName($h) -ine $r.class){throw 'Bound control class changed'};$result[$r.name]=[ordered]@{handle=$r.handle;pid=$app.Id;id=$r.id;class=$r.class;visible=[DcmNative]::IsWindowVisible($h);enabled=[DcmNative]::IsWindowEnabled($h);text=[DcmNative]::Text($h,$app.Id)}}
 return [pscustomobject]$result
}
function ProviderIdentities($nodes){
 $rows=@();$pids=@($nodes|ForEach-Object {$_.PID}|Sort-Object -Unique)
 if($pids.Count -gt 8){throw 'Native provider process observation bound exceeded'}
 foreach($id in $pids){$p=$null;try{$p=Get-Process -Id $id -ErrorAction Stop;$p.Refresh();$path=$p.Path;$start=$p.StartTime.ToUniversalTime().ToString('o');if($p.HasExited -or -not $path){throw 'Provider path/start identity unavailable'};$rows+=[ordered]@{pid=$id;path=$path;start=$start;available=$true;relation='visible descendant of owned QuickSortPreview'}}catch{$rows+=[ordered]@{pid=$id;available=$false;error=$_.Exception.Message;relation='visible descendant of owned QuickSortPreview'}}finally{if($p){$p.Dispose()}}}
 return $rows
}
function ObservePreview([bool]$accessibility=$false){
 NoDialog;$captions=HostCaptions;$controls=Controls;$nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$native=@();$edits=@()
 foreach($node in $nodes){
  $key=[string]$node.Handle
  if($basePaneHandles.ContainsKey($key)){$old=$basePaneHandles[$key];if($node.PID -ne $app.Id -or $node.Class -cne $old.Class -or $node.Parent -ne $old.Parent){throw 'Initial owned preview child identity changed'}}
  if(-not $node.Visible -or $node.Right -le $node.Left -or $node.Bottom -le $node.Top){continue}
  if(-not $basePaneHandles.ContainsKey($key)){$native+=$node}
  if($basePaneHandles.ContainsKey($key) -and $node.Class -ieq 'Edit'){$edits+=$node}
 }
 $state='pending_or_other';$status=''
 $errors=@($captions|Where-Object {$_.StartsWith($unavailablePrefix,[StringComparison]::Ordinal) -and $_.Length -gt $unavailablePrefix.Length})
 $success=@($captions|Where-Object {$_ -cmatch $validFooterPattern})
 if($captions -ccontains $fallbackFooter){$state='windows_fallback';$status=$fallbackFooter}
 elseif($errors.Count -eq 1){$state='dcm_unavailable';$status=$errors[0]}
 elseif($success.Count -eq 1){$state='default_page';$status=$success[0]}
 elseif($captions -ccontains 'Plain-text preview: TXT'){$state='text';$status='Plain-text preview: TXT'}
 $accessible='';$matched=@();$identities=@()
 if($state -ceq 'windows_fallback'){
  $identities=@(ProviderIdentities $native)
  if($accessibility){$accessible=[DcmDocxProbe]::ReadAccessibleText($hwnd,$app.Id);$visibleNames=(@($native|ForEach-Object {$_.Name}) -join "`n");$errorReadback=$accessible+"`n"+$visibleNames;foreach($phrase in $nativeErrorPhrases){if($errorReadback.IndexOf($phrase,[StringComparison]::OrdinalIgnoreCase) -ge 0){$matched+=$phrase}}}
 }
 return [pscustomobject]@{utc=[DateTime]::UtcNow.ToString('o');selected_file=$selectedName;selected_stem=[DcmUndoInput]::TryText($hwnd,$app.Id,1005);state=$state;status=$status;host_captions=$captions;controls=$controls;preview_windows=$nodes;native_windows=$native;native_processes=$identities;visible_owned_text_edits=$edits;accessible_text=$accessible;matched_explicit_error_phrases=$matched}
}
function ObservationSignature($o){return (ConvertTo-Json -Depth 15 -Compress -InputObject ([ordered]@{selected=$o.selected_stem;state=$o.state;status=$o.status;controls=$o.controls;native_windows=$o.native_windows;native_processes=$o.native_processes;visible_owned_text_edits=$o.visible_owned_text_edits}))}
function CaptureFrame([string]$name){
 Own;$s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false)
 $shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false;kind='full_window';left=$s.Left;top=$s.Top;width=$s.Width;height=$s.Height})
 $geometry=[DcmPreviewGeometry]::Observe($hwnd,$app.Id);$geometryPath=Join-Path $lab ($name+'.geometry.json');Json $geometryPath $geometry
 $contexts=@($geometry.Contexts|Where-Object {$_.CallerContextLabel -ceq 'per_monitor_aware_v2_requested_minus4'})
 if($contexts.Count -ne 1){throw 'Physical preview observation missing'}
 $roots=@($contexts[0].Windows|Where-Object {$_.Role -ceq 'host' -and $_.Available -and $_.Visible -and $_.WindowRectAvailable});$panes=@($contexts[0].Windows|Where-Object {$_.Role -ceq 'owned_preview_pane' -and $_.Available -and $_.Visible -and $_.WindowRectAvailable})
 if($roots.Count -ne 1 -or $panes.Count -ne 1){throw 'Owned PDF capture geometry unavailable'}
 $root=$roots[0].WindowRect;$region=$panes[0].WindowRect
 if($root.Left -ne $s.Left -or $root.Top -ne $s.Top -or $root.Right-$root.Left -ne $s.Width -or $root.Bottom-$root.Top -ne $s.Height){throw 'Host moved or resized after source capture'}
 if($region.Left -lt $root.Left -or $region.Top -lt $root.Top -or $region.Right -gt $root.Right -or $region.Bottom -gt $root.Bottom -or $region.Right-$region.Left -lt 100 -or $region.Bottom-$region.Top -lt 80){throw 'Preview pane escapes full captured frame'}
 $path=Join-Path $lab ($name+'.preview.png');$bitmap=New-Object Drawing.Bitmap($s.Path)
 try{if($bitmap.Width -ne $s.Width -or $bitmap.Height -ne $s.Height){throw 'Saved screenshot dimensions changed'};$rectangle=New-Object Drawing.Rectangle(($region.Left-$s.Left),($region.Top-$s.Top),($region.Right-$region.Left),($region.Bottom-$region.Top));$crop=$bitmap.Clone($rectangle,[Drawing.Imaging.PixelFormat]::Format32bppArgb);try{$crop.Save($path,[Drawing.Imaging.ImageFormat]::Png)}finally{$crop.Dispose()}}finally{$bitmap.Dispose()}
 Own;$ink=[DcmPptxInput]::InspectPixels($path)
 $shots.Add([ordered]@{name=($name+'.preview');path=$path;sha256=(Hash $path);visual_reviewed=$false;kind='same_frame_preview_pixels';source_frame=$s.Path;geometry=$geometryPath;geometry_sha256=(Hash $geometryPath);left=$region.Left;top=$region.Top;pixels=$ink})
 return [pscustomobject]@{path=$path;pixels=$ink}
}
function ContractIssues($o,[string]$kind){
 $issues=@();$c=$o.controls
 if($o.selected_stem -cne [IO.Path]::GetFileNameWithoutExtension($selectedName)){$issues+='Selected filename readback changed'}
 foreach($key in $sizingKeys){if(-not $c.$key.visible){$issues+=('Default sizing control hidden: '+$key)}}
 if($o.visible_owned_text_edits.Count){$issues+='Previous TXT body remains visible'}
 if($kind -ceq 'valid'){
  if($o.state -cne 'default_page'){$issues+='Expected exact default PDF page-one footer'}
  if($o.native_windows.Count){$issues+='Valid default PDF unexpectedly has a native fallback subtree'}
  if(-not $c.previous.visible -or $c.previous.enabled -or -not $c.next.visible -or -not $c.next.enabled -or -not $c.page.visible -or $c.page.text -cne 'Page 1 of 2'){$issues+='Expected visible Page 1 of 2 with Previous disabled and Next enabled'}
  foreach($key in $sizingKeys){if(-not $c.$key.enabled){$issues+=('Valid PDF sizing control disabled: '+$key)}}
 }else{
  if($c.previous.visible -or $c.next.visible -or $c.page.visible){$issues+='Default PDF page controls remain visible after error'}
  if($o.state -ceq 'dcm_unavailable'){
   if($o.native_windows.Count){$issues+='Direct DCM error retains a native subtree'}
   foreach($key in $sizingKeys){if($c.$key.enabled){$issues+=('Direct error retains enabled sizing control: '+$key)}}
  }elseif($o.state -ceq 'windows_fallback'){
   if(-not $o.native_windows.Count){$issues+='Fallback footer has no visible owned-pane native subtree'}
   if(-not $o.matched_explicit_error_phrases.Count){$issues+='Fallback has no explicit error in its scoped accessible/visible child text'}
   if(@($o.native_processes|Where-Object {-not $_.available}).Count){$issues+='Native provider process identity unavailable'}
  }else{$issues+='Expected terminal DCM unavailable or explicit native error'}
  if($o.host_captions -ccontains 'Plain-text preview: TXT' -or @($o.host_captions|Where-Object {$_ -cmatch $validFooterPattern}).Count){$issues+='Previous successful provider status remains visible'}
 }
 return $issues
}
function ReadyPreview([string]$name,[string]$kind,$baseline=$null){
 if($kind -cnotin @('valid','error') -or ($kind -ceq 'error' -and $null -eq $baseline)){throw 'Invalid PDF readiness expectation'}
 $previous='';$previousEligible=$false;$ready=$false;$accepted=$null;$until=[DateTime]::UtcNow.AddSeconds(45)
 for($attempt=1;$attempt -le 30 -and [DateTime]::UtcNow -lt $until;$attempt++){
  NoDialog;$before=ObservePreview $true;$tag=$name+'-attempt-'+$attempt.ToString('00');$observationPath=Join-Path $lab ($tag+'.state.json')
  # Persist the pre-capture observation even if capture or post-readback fails.
  Json $observationPath ([ordered]@{stage=$name;attempt=$attempt;expected_kind=$kind;before=$before;completed=$false})
  $attemptEntry=[ordered]@{stage=$name;attempt=$attempt;path=$observationPath;sha256=(Hash $observationPath);capture=$null;state=$before.state;stable=$false;completed=$false};$receipt.observations.Add($attemptEntry)
  $frame=CaptureFrame $tag;$after=ObservePreview $false
  $issues=@(ContractIssues $before $kind);if((ObservationSignature $before) -cne (ObservationSignature $after)){$issues+='Readback/window identity changed across captured frame'}
  $pixels=$frame.pixels;$pixelCondition=$pixels.DarkPixels -ge 20 -and $pixels.LightPixels -ge 100;$changedFromValid=$null;$tealLimit=$null
  if($kind -ceq 'valid'){$pixelCondition=$pixelCondition -and $pixels.TealPixels -ge 1000 -and $pixels.OrangePixels -lt 100}
  if($baseline){$changedFromValid=[DcmPptxInput]::PixelDifference($baseline.path,$frame.path)}
  if($kind -ceq 'error'){
   $tealLimit=[Math]::Max(100,[Math]::Ceiling($baseline.pixels.TealPixels*0.05))
   $pixelCondition=$pixelCondition -and $pixels.TealPixels -lt $tealLimit -and $changedFromValid -ge [Math]::Max(100,[Math]::Ceiling($pixels.SampledPixels*0.005))
  }
  $difference=$null;if($previous){$difference=[DcmPptxInput]::PixelDifference($previous,$frame.path)}
  $eligible=$issues.Count -eq 0 -and $pixelCondition;$ready=$eligible -and $previousEligible -and $null -ne $difference -and $difference -le 30
  $enabledSizing=@($sizingKeys|Where-Object {$key=$_;$before.controls.$key.enabled})
  $row=[ordered]@{stage=$name;attempt=$attempt;expected_kind=$kind;before=$before;after=$after;capture=$frame.path;pixels=$pixels;contract_issues=$issues;pixel_condition=[bool]$pixelCondition;changed_from_valid_page=$changedFromValid;error_teal_sample_limit=$tealLimit;changed_since_previous=$difference;eligible=[bool]$eligible;stable=[bool]$ready;native_error_with_enabled_sizing_controls=($before.state -ceq 'windows_fallback' -and $enabledSizing.Count -gt 0);enabled_sizing_controls=$enabledSizing;native_error_control_readiness_acceptance='not_accepted_by_this_gate';semantic_image_review_required=$true}
  $row['completed']=$true;Json $observationPath $row
  $attemptEntry.sha256=Hash $observationPath;$attemptEntry.capture=$frame.path;$attemptEntry.stable=[bool]$ready;$attemptEntry.completed=$true
  if($ready){$accepted=[pscustomobject]@{path=$frame.path;pixels=$pixels;state=$before.state;observation=$observationPath;readback=$before;native_error_with_enabled_sizing_controls=$row.native_error_with_enabled_sizing_controls};break}
  $previous=$frame.path;$previousEligible=$eligible;Start-Sleep -Milliseconds 650
 }
 if(-not $ready){throw ('DCM34_PDF_READINESS: no stable honest '+$kind+' state for '+$name+'; inspect every retained state/image attempt')}
 $detail=[ordered]@{selected_file=$selectedName;state=$accepted.state;status=$accepted.readback.status;accepted_capture=$accepted.path;accepted_observation=$accepted.observation;pixels=$accepted.pixels;default_controls=$accepted.readback.controls;native_processes=$accepted.readback.native_processes;matched_explicit_error_phrases=$accepted.readback.matched_explicit_error_phrases;native_error_with_enabled_sizing_controls=$accepted.native_error_with_enabled_sizing_controls;visible_error_semantics_machine_accepted=$false;semantic_image_review_required=$true}
 if($accepted.state -ceq 'dcm_unavailable'){$detail['expected_gdi_body']='PDF preview unavailable'+"`n`n"+$accepted.readback.status.Substring($unavailablePrefix.Length);$detail['gdi_body_text_readback_available']=$false}
 if($accepted.state -ceq 'windows_fallback'){$detail['native_error_control_readiness']='review_required: attachment may enable generic sizing without a loaded page';$receipt['native_fallback_observed']=$true;if($accepted.native_error_with_enabled_sizing_controls){$receipt['native_error_with_enabled_sizing_controls']=$true}}
 Record $name $detail;return $accepted
}
function TextBody([string]$name,[string]$plain,$baseline=$null){
 $initial=$name -ceq '01-initial-text'
 if(($initial -and $null -ne $baseline) -or (-not $initial -and ($name -cnotin @('04-first-text-recovery','07-final-text-recovery') -or $null -eq $baseline))){throw 'TXT recovery requires its initial captured baseline'}
 Wait ('exact text '+$name) {$nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$edits=@($nodes|Where-Object {$_.Visible -and $_.PID -eq $app.Id -and $_.Class -ieq 'Edit' -and $basePaneHandles.ContainsKey([string]$_.Handle)});$edits.Count -eq 1 -and ([DcmNative]::Text([IntPtr]$edits[0].Handle,$app.Id)) -ceq $plain}
 $before=ObservePreview;$c=$before.controls
 if($before.state -cne 'text' -or $before.native_windows.Count -or $before.visible_owned_text_edits.Count -ne 1){throw 'TXT recovery retains wrong provider or native subtree'}
 if($c.previous.visible -or $c.next.visible -or $c.page.visible){throw 'TXT recovery retains visible PDF page controls'}
 foreach($key in $sizingKeys){if(-not $c.$key.visible -or $c.$key.enabled -ne ($key -cne 'width')){throw ('TXT supported sizing controls not restored: '+$key)}}
 $actual=[DcmNative]::Text([IntPtr]$before.visible_owned_text_edits[0].Handle,$app.Id);if($actual -cne $plain){throw 'TXT exact body changed'}
 $textPath=Join-Path $lab ($name+'.displayed.txt');[IO.File]::WriteAllText($textPath,$actual,$utf8)
 $frame=CaptureFrame $name;$after=ObservePreview
 # ClearType text can contain teal/orange fringes. Bind the original exact TXT
 # image once, then compare recoveries with that saved frame instead of using
 # an absolute colour limit. Preserve readbacks before any pixel failure.
 $reference=if($initial){[pscustomobject]@{path=$frame.path;sha256=(Hash $frame.path);pixels=$frame.pixels}}else{$baseline}
 $geometryMatch=$frame.pixels.Width -eq $reference.pixels.Width -and $frame.pixels.Height -eq $reference.pixels.Height
 $baselineHashVerified=$false;$difference=$null;$pixelError=''
 try{
  if($reference.path -cne (Join-Path $lab '01-initial-text.preview.png')){throw 'TXT baseline path differs from the initial owned capture'}
  $baselineHashVerified=(Hash $reference.path) -ceq $reference.sha256;if(-not $baselineHashVerified){throw 'Initial TXT baseline bytes changed'}
  if(-not $geometryMatch){throw 'TXT recovery geometry differs from its initial baseline'}
  $difference=if($initial){0}else{[DcmPptxInput]::PixelDifference($reference.path,$frame.path)}
  if($difference -gt 30){throw 'TXT recovery differs from its initial captured pixels'}
 }catch{$pixelError=$_.Exception.Message}
 $readbackStable=(ObservationSignature $before) -ceq (ObservationSignature $after);$pixelCondition=$pixelError -ceq ''
 $statePath=Join-Path $lab ($name+'.state.json');Json $statePath ([ordered]@{before=$before;after=$after;displayed_text_sha256=(Hash $textPath);capture=$frame.path;pixels=$frame.pixels;pixel_contract=$receipt.txt_pixel_contract;comparison=$(if($initial){'bind_initial'}else{'compare_recovery'});baseline_path=$reference.path;baseline_sha256=$reference.sha256;baseline_hash_verified=$baselineHashVerified;baseline_geometry_match=$geometryMatch;changed_from_initial_txt=$difference;maximum_changed_samples=30;readback_stable=$readbackStable;pixel_condition=$pixelCondition;pixel_error=$pixelError})
 if(-not $readbackStable){throw 'TXT readback changed across capture'};if(-not $pixelCondition){throw $pixelError}
 Record $name ([ordered]@{displayed_sha256=(Hash $textPath);exact_txt_body=$true;native_subtree_absent=$true;pdf_controls_hidden=$true;state=$statePath;accepted_capture=$frame.path;pixel_contract=$receipt.txt_pixel_contract;baseline_path=$reference.path;baseline_sha256=$reference.sha256;changed_from_initial_txt=$difference;semantic_image_review_required=$true})
 return $reference
}
function Registration {
 $extension='.pdf';$key='{8895b1c6-b41f-4c1c-a562-0d564250836f}';$reg=[Microsoft.Win32.Registry]::ClassesRoot;$r=$reg.OpenSubKey($extension);$prog=''
 if($r){try{if($r.GetValueNames() -contains '' -and $r.GetValueKind('') -eq [Microsoft.Win32.RegistryValueKind]::String){$prog=[string]$r.GetValue('')}}finally{$r.Dispose()}}
 $keys=@();if($prog){$keys+=($prog+'\shellex\'+$key)};$keys+=($extension+'\shellex\'+$key);$keys+=('SystemFileAssociations\'+$extension+'\shellex\'+$key)
 $rows=@();$selected='';$selectedKey=''
 foreach($k in $keys){$r=$reg.OpenSubKey($k);$value=$null;if($r){try{if($r.GetValueNames() -contains '' -and $r.GetValueKind('') -eq [Microsoft.Win32.RegistryValueKind]::String){$value=[string]$r.GetValue('')}}finally{$r.Dispose()}};$rows+=[ordered]@{key=$k;value=$value};if(-not $selected -and $null -ne $value -and $value.Trim().StartsWith('{',[StringComparison]::Ordinal)){$selected=$value.Trim();$selectedKey=$k}}
 return [pscustomobject]@{extension=$extension;prog_id=$prog;candidates=$rows;selected_clsid=$selected;selected_key=$selectedKey;read_only=$true}
}
function ExclusivePDFs {
 foreach($name in @('08-preview.pdf','10-damaged.pdf')){$h=[IO.File]::Open((Join-Path $fixture.inbox $name),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()}
}
function StopApp {
 if(-not $app){return};$app.Refresh();$receipt['normal_exit']=$false;$issues=New-Object 'Collections.Generic.List[string]'
 $lifecycle=[ordered]@{started_utc=[DateTime]::UtcNow.ToString('o');host=$owned;forced_actions=(New-Object 'Collections.Generic.List[object]');natural_child_drain_seconds=10;normal_exit=$false}
 if(-not $receipt.Contains('lifecycle_attempts')){$receipt['lifecycle_attempts']=New-Object 'Collections.Generic.List[object]'};$receipt.lifecycle_attempts.Add($lifecycle)
 $parentStart=[DateTime]::Parse($owned.start).ToUniversalTime();$upper=if($app.HasExited){$app.ExitTime.ToUniversalTime()}else{[DateTime]::UtcNow}
 if(-not $app.HasExited){OwnProcess}
 $children=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$owned.pid)|Where-Object {$_.CreationDate.ToUniversalTime() -ge $parentStart -and $_.CreationDate.ToUniversalTime() -le $upper}|ForEach-Object {[ordered]@{pid=[int]$_.ProcessId;path=$_.ExecutablePath;created=$_.CreationDate.ToUniversalTime().ToString('o')}})
 $lifecycle['children_before_close']=$children
 if($app.HasExited){if(-not $owned.Contains('close_requested_utc')){$owned['unexpected_exit']=$true;$issues.Add('Owned host exited before a close request')}}else{
  try{Own;[DcmNative]::CloseAsync($hwnd,$app.Id);$owned['close_requested_utc']=[DateTime]::UtcNow.ToString('o')}catch{$lifecycle['close_window_error']=$_.Exception.Message;$issues.Add('Owned close request failed')}
  if(-not $app.WaitForExit(15000)){
   # Failure-only teardown: retain evidence, then terminate this exact host.
   # Never terminate a process tree or a shared Windows preview provider.
   $receipt.passed=$false;$owned['forced_stop']=$true;$issues.Add('Owned host required forced shutdown')
   try{$null=CaptureFrame ('cleanup-host-timeout-'+$receipt.lifecycle_attempts.Count)}catch{$lifecycle['timeout_capture_error']=$_.Exception.Message}
   OwnProcess;$action=[ordered]@{role='owned_host';pid=$owned.pid;path=$exe;start=$owned.start;reason='normal close exceeded 15 seconds';requested_utc=[DateTime]::UtcNow.ToString('o');exited=$false};$lifecycle.forced_actions.Add($action)
   $app.Kill();$action.exited=$app.WaitForExit(10000);if(-not $action.exited){throw 'Exact owned host did not exit after failure-only cleanup'}
  }
 }
 $owned['exit_code']=$app.ExitCode;$parentExit=$app.ExitTime.ToUniversalTime()
 $late=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$owned.pid)|Where-Object {$_.CreationDate.ToUniversalTime() -ge $parentStart -and $_.CreationDate.ToUniversalTime() -le $parentExit}|ForEach-Object {[ordered]@{pid=[int]$_.ProcessId;path=$_.ExecutablePath;created=$_.CreationDate.ToUniversalTime().ToString('o')}})
 $lifecycle['children_after_host_exit']=$late;$unique=@{}
 foreach($child in @($children)+@($late)){$key=[string]$child.pid+'|'+$child.created+'|'+$child.path;$unique[$key]=$child}
 $children=@($unique.Values);if($children.Count -gt 8){throw 'Owned direct-child cleanup bound exceeded'}
 $liveChildren=@();$drainUntil=[DateTime]::UtcNow.AddSeconds(10)
 do{
  $liveChildren=@()
  foreach($child in $children){$live=Get-CimInstance Win32_Process -Filter ('ProcessId='+$child.pid);if($live -and $live.CreationDate.ToUniversalTime().ToString('o') -eq $child.created -and $live.ExecutablePath -eq $child.path){$liveChildren+=$child}}
  if(-not $liveChildren.Count){break};Start-Sleep -Milliseconds 150
 }while([DateTime]::UtcNow -lt $drainUntil)
 $lifecycle['children_after_natural_drain']=$liveChildren
 foreach($child in $liveChildren){
  $receipt.passed=$false;$issues.Add('Owned renderer did not drain naturally');$drain=$null
  try{
   if(-not $child.path){throw 'Cannot force a child with unreadable executable identity'}
   $again=Get-CimInstance Win32_Process -Filter ('ProcessId='+$child.pid)
   if(-not $again -or $again.ExecutablePath -ne $child.path -or $again.CreationDate.ToUniversalTime().ToString('o') -ne $child.created){continue}
   $drain=Get-Process -Id $child.pid -ErrorAction Stop;$drain.Refresh()
   if($drain.HasExited){continue};if($drain.Path -ne $child.path){throw 'Refusing cleanup after direct-child executable changed'}
   # Compare CIM creation timestamps with CIM timestamps; Process.StartTime can
   # expose a finer precision. Revalidate the recorded identity after opening it.
   $confirm=Get-CimInstance Win32_Process -Filter ('ProcessId='+$child.pid)
   if(-not $confirm -or $confirm.ExecutablePath -ne $child.path -or $confirm.CreationDate.ToUniversalTime().ToString('o') -ne $child.created){throw 'Refusing cleanup after direct-child creation identity changed'}
   $action=[ordered]@{role='recorded_direct_child';pid=$child.pid;path=$child.path;cim_created=$child.created;process_start=$drain.StartTime.ToUniversalTime().ToString('o');reason='natural child drain exceeded 10 seconds';requested_utc=[DateTime]::UtcNow.ToString('o');exited=$false};$lifecycle.forced_actions.Add($action)
   $drain.Kill();$action.exited=$drain.WaitForExit(5000);if(-not $action.exited){throw 'Exact owned child did not exit after failure-only cleanup'}
  }catch{$issues.Add($_.Exception.Message)}finally{if($drain){$drain.Dispose()}}
 }
 if($owned.exit_code -ne 0){$issues.Add('Host nonzero exit')};if($owned.Contains('forced_stop')){$receipt.passed=$false;$issues.Add('Forced host cleanup cannot pass normal-close acceptance')};if($owned.Contains('unexpected_exit')){$receipt.passed=$false;$issues.Add('Unexpected host exit cannot pass normal-close acceptance')}
 $lifecycle['finished_utc']=[DateTime]::UtcNow.ToString('o');$lifecycle['issues']=@($issues);$app.Dispose();$script:app=$null;$script:hwnd=[IntPtr]::Zero
 if($issues.Count){throw ($issues -join '; ')};$lifecycle.normal_exit=$true;$receipt['normal_exit']=$true
}
Write-Output ('REPORT_ROOT='+$lab)
try {
 $build=Get-Content -LiteralPath $BuildReceipt -Raw -Encoding UTF8|ConvertFrom-Json;$head=Git @('rev-parse','HEAD')
 if(-not $build.passed -or $build.mode -ne 'build' -or -not $build.source_unchanged -or $build.commit -ne $head){throw 'Exact clean build required'}
 if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256 -or (Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'Build identity changed'}
 $manifest=Get-Content -LiteralPath $build.source_hash_manifest.path -Raw -Encoding UTF8|ConvertFrom-Json;Source
 $receipt['commit']=$head;$receipt['build_receipt']=$BuildReceipt;$receipt['build_receipt_sha256']=Hash $BuildReceipt;$receipt['app_sha256']=$build.app_artifact.sha256
 $definition=Get-Content -LiteralPath $fixtureManifest -Raw -Encoding UTF8|ConvertFrom-Json
 foreach($field in @('schema_version','file','bytes','sha256')){if($null -eq $definition -or -not $definition.PSObject.Properties[$field]){throw ('Damaged PDF manifest field missing: '+$field)}}
 $payload="%PDF-1.7`n1 0 obj`n<< /Type /Catalog /Pages 99 0 R >>`nendobj`n";$bytes=[IO.File]::ReadAllBytes($damagedSource)
 if($definition.schema_version -ne 1 -or $definition.file -cne 'dcm34-damaged.pdf' -or $definition.bytes -ne $bytes.Length -or $definition.sha256 -cnotmatch '^[0-9a-f]{64}$' -or (Hash $damagedSource) -cne $definition.sha256 -or [Text.Encoding]::ASCII.GetString($bytes) -cne $payload){throw 'Damaged PDF differs from pinned header-valid incomplete catalog'}
 foreach($path in @('tests/fixtures/dcm34-damaged.pdf','tests/fixtures/dcm34-damaged.manifest.json')){$entries=@($manifest|Where-Object {$_.path -ceq $path});if($entries.Count -ne 1 -or $entries[0].sha256 -cne (Hash (Join-Path $repo $path))){throw 'Damaged PDF fixture/manifest missing from exact build snapshot'}}
 $receipt['damaged_fixture']=$definition;$receipt['damaged_manifest_sha256']=Hash $fixtureManifest
 Add-Type -AssemblyName System.Drawing;Add-Type -AssemblyName System.Windows.Forms
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-pptx-input.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-preview-geometry.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
 Add-Type -AssemblyName UIAutomationClient;Add-Type -AssemblyName UIAutomationTypes;Add-Type -AssemblyName WindowsBase
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-docx-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll',([Windows.Automation.AutomationElement]).Assembly.Location,([Windows.Automation.AutomationPattern]).Assembly.Location,([Windows.Point]).Assembly.Location)
 . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
 Copy-Item -LiteralPath $damagedSource -Destination (Join-Path $fixture.inbox '10-damaged.pdf')
 Get-ChildItem -LiteralPath $fixture.inbox -File|ForEach-Object {$expected[$_.Name]=Hash $_.FullName};if($expected.Count -ne 5){throw 'Unexpected qualification fixture count'};$receipt['source_fixture_hashes']=$expected
 $registrations=Registration;$receipt['handler_registrations_before']=$registrations
 $receipt['scope_limit']='One deterministic damaged PDF using default Windows.Data.Pdf and natural installed-handler fallback. Completed-state recovery only; no protected/unsupported formats, cancellation, interruption or full DCM-34 acceptance. Direct GDI error wording and native error/control readiness require separate original-image review.'
 $receipt['sequence']=@('TXT','valid default PDF page 1','damaged PDF','TXT recovery','valid default PDF page 1 recovery','damaged PDF again','TXT recovery','exclusive release','normal close and source preservation')
 $receipt['native_error_phrase_allowlist']=$nativeErrorPhrases;$receipt['native_fallback_observed']=$false;$receipt['native_error_with_enabled_sizing_controls']=$false
 $exe=Join-Path $lab ('DCM-PDF-FAILURE-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item -LiteralPath $build.app_artifact.path -Destination $exe
 if((Hash $exe) -cne $build.app_artifact.sha256){throw 'Owned executable copy differs from exact build'}
 $temp=Join-Path $lab 'temp';$null=New-Item -ItemType Directory $temp
 $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
 $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming;$psi.EnvironmentVariables['TEMP']=$temp;$psi.EnvironmentVariables['TMP']=$temp;$psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
 foreach($var in @('DCM_PDF_PROVIDER','DCM_SUMATRA_EXE')){$psi.EnvironmentVariables.Remove($var)}
 $receipt['child_process_environment']=[ordered]@{pdf_provider_override_removed=$true;sumatra_executable_override_removed=$true;default_pdf_view='fit_width';registry_mutated=$false;fallback_suppressed=$false}
 $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'Host launch failed'};$owned=[ordered]@{pid=$app.Id;path=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')};$receipt['owned_process']=$owned
 $until=[DateTime]::UtcNow.AddSeconds(20);do{$app.Refresh();if($app.HasExited){throw 'Host exited before window'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
 Wait 'fixture list' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $expected.Count}
 SelectFile '01-alpha.txt';BindDefaultControls;$plain=[IO.File]::ReadAllText((Join-Path $fixture.inbox '01-alpha.txt'))
 NoDialog;$screen=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea;[DcmNative]::Restore($hwnd,$app.Id,$screen.Left+8,$screen.Top+8);Start-Sleep -Milliseconds 300
 $initialText=TextBody '01-initial-text' $plain
 SelectFile '08-preview.pdf';$firstValid=ReadyPreview '02-valid-before-damage' 'valid'
 SelectFile '10-damaged.pdf';$firstError=ReadyPreview '03-first-damaged' 'error' $firstValid
 SelectFile '01-alpha.txt';$null=TextBody '04-first-text-recovery' $plain $initialText;ExclusivePDFs
 SelectFile '08-preview.pdf';$recovered=ReadyPreview '05-valid-pdf-recovery' 'valid' $firstValid
 SelectFile '10-damaged.pdf';$secondError=ReadyPreview '06-second-damaged' 'error' $recovered
 if($firstError.state -cne $secondError.state){throw 'Natural damaged-PDF terminal state differs on repeat opening'}
 SelectFile '01-alpha.txt';$null=TextBody '07-final-text-recovery' $plain $initialText;ExclusivePDFs
 Record '08-both-pdfs-exclusively-released' ([ordered]@{files=@('08-preview.pdf','10-damaged.pdf');exclusive_read_open=$true})
 StopApp;Source;Files;ExclusivePDFs
 if((Hash $damagedSource) -cne $definition.sha256 -or (Hash $exe) -cne $build.app_artifact.sha256){throw 'Original fixture/application bytes changed'}
 Record '09-normal-close-and-original-bytes' ([ordered]@{exit_code=$owned.exit_code;original_fixture_bytes_preserved=$true;source_unchanged=$true;journal_empty=$true})
 if($cases.Count -ne 9 -or $shots.Count -lt 22){throw 'Incomplete PDF damage/recovery cases or original captures'}
 $receipt['machine_checks_passed']=$true;$receipt.passed=$true
}catch{
 $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)
 if($app){try{Own;Json (Join-Path $lab 'failure-preview-windows.json') @([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$receipt['failure_host_captions']=HostCaptions}catch{$receipt['diagnostic_error']=$_.Exception.Message};try{$null=CaptureFrame 'failure'}catch{$receipt['capture_error']=$_.Exception.Message}}
}finally{
 try{StopApp}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message}
 try{if($head -and $manifest.Count){Source;$receipt['source_unchanged']=$true};if($fixture -and $expected.Count){Files;$receipt['fixtures_preserved']=$true;$receipt['journal_empty']=$true};if($registrations){$after=Registration;$receipt['handler_registrations_after']=$after;if(($registrations|ConvertTo-Json -Depth 8 -Compress) -cne ($after|ConvertTo-Json -Depth 8 -Compress)){throw 'PDF handler registrations changed during qualification'};$receipt['handler_registrations_unchanged']=$true}}catch{$receipt.passed=$false;$receipt['preservation_error']=$_.Exception.Message}
 $receipt.machine_checks_passed=$receipt.passed;$receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt;Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))
}
if(-not $receipt.passed){exit 1}
