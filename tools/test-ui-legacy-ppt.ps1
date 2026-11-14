[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT)
# DCM-33, PPT only: natural installed-handler or honest unavailable behavior.
# No registry mutation, Office conversion, forced provider absence or global
# input. Native Page Down/Up goes only to the pinned owned PowerPoint surface.
# Saved pixels corroborate navigation; exact slide content needs separate review.
$ErrorActionPreference='Stop';Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required'}
if(-not $BuildReceipt -or -not $env:FLOWDEV_WORKSPACE_ROOT){throw 'Exact build and workspace required'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$presentation=Join-Path $repo 'tests/fixtures/dcm33-legacy-ppt.ppt'
$fixtureManifest=Join-Path $repo 'tests/fixtures/dcm33-legacy-ppt.manifest.json'
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-legacy-ppt-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Test lab must be outside source'}
$null=New-Item -ItemType Directory $lab;$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-33';format_scope='PPT_only';other_formats_not_tested=@('DOC','XLS');report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;feature_accepted=$false;format_accepted=$false;cases=$cases;captures=$shots;visual_review_required=$true;user_test='not_recorded';provider='not_observed';native_slide_navigation='not_exercised';all_fixture_slides_captured=$false;native_input_actions=(New-Object 'Collections.Generic.List[object]');native_capture_attempts=(New-Object 'Collections.Generic.List[object]');observations=(New-Object 'Collections.Generic.List[object]')}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$head='';$manifest=@();$fixture=$null;$expected=@{};$exe='';$presentationHash='';$nativeTarget=$null;$requiredText=@();$slideOracles=@();$handlerRegistered=$false;$registrations=$null;$unavailableStatus=''
function Hash([string]$p){return (Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Json([string]$p,$v){[IO.File]::WriteAllText($p,(ConvertTo-Json -InputObject $v -Depth 25),$utf8)}
function Git([string[]]$arguments){$s=(& git.exe -C $repo @arguments|Out-String).Trim();if($LASTEXITCODE){throw 'Git query failed'};return $s}
function Source {
 if((Git @('rev-parse','HEAD')) -ne $head -or (Git @('status','--porcelain=v1','--untracked-files=all'))){throw 'Source identity changed'}
 foreach($f in $manifest){$p=[IO.Path]::GetFullPath((Join-Path $repo $f.path));if(-not $p.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $p) -ne $f.sha256){throw 'Source manifest changed'}}
}
function Own {$app.Refresh();if($app.HasExited -or $app.Path -ne $exe -or $app.StartTime.ToUniversalTime().ToString('o') -ne $owned.start){throw 'Owned host identity changed'};[DcmNative]::RequireOwned($hwnd,$app.Id)}
function Wait([string]$name,[scriptblock]$predicate){$until=[DateTime]::UtcNow.AddSeconds(20);do{Own;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw ('Unexpected owned dialog: '+$name)};if(& $predicate){return};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until);throw ('Deadline: '+$name)}
function Files {
 if(-not $fixture){return};$all=@(Get-ChildItem $fixture.inbox -File)
 if($all.Count -ne $expected.Count){throw 'Fixture names changed'}
 foreach($f in $all){if(-not $expected.ContainsKey($f.Name) -or (Hash $f.FullName) -ne $expected[$f.Name]){throw 'Fixture content changed'}}
 if(@(Get-ChildItem $fixture.destination -Force).Count){throw 'Destination changed'}
 $p=Join-Path $fixture.config 'recovery\move-journal.jsonl';if((Test-Path $p) -and ([IO.File]::ReadAllText($p)).Trim()){throw 'Unexpected operation journal'}
}
function Record([string]$name,$detail){$cases.Add([ordered]@{name=$name;passed=$true;detail=$detail});Files}
function SelectFile([string]$name){$items=@([DcmNative]::ListItems($hwnd,$app.Id,1016));$index=[Array]::IndexOf($items,$name);if($index -lt 0){throw 'Missing fixture in list'};[DcmUndoInput]::Select($hwnd,$app.Id,$index);$stem=[IO.Path]::GetFileNameWithoutExtension($name);Wait ('selection '+$name) {([DcmUndoInput]::TryText($hwnd,$app.Id,1005)) -ceq $stem};$selected=@([DcmNative]::SelectedIndexes($hwnd,$app.Id,1016));if($selected.Count -ne 1 -or $selected[0] -ne $index){throw 'Selected fixture index mismatch'}}
function HostCaptions {return @([DcmDocxProbe]::HostStatics($hwnd,$app.Id)|ForEach-Object {[DcmNative]::Text([IntPtr]$_,$app.Id)})}
function RequireStatus([string]$text){if(-not ((HostCaptions) -ccontains $text)){throw ('PPT provider/status mismatch: '+$text)}}
function NativeNodes {
 # Native handlers can run in the host or a surrogate. Require an actual
 # visible nonempty subtree and exact PPT status, not a foreign PID alone.
 return @([DcmDocxProbe]::Snapshot($hwnd,$app.Id)|Where-Object {$_.Visible -and $_.Right -gt $_.Left -and $_.Bottom -gt $_.Top -and ($_.PID -ne $app.Id -or $_.Class -notin @('Edit','ComboBox'))})
}
function PreviewText {
 if((HostCaptions) -ccontains 'Windows preview: PPT'){return $null}
 $nodes=@([DcmDocxProbe]::Snapshot($hwnd,$app.Id))
 if(@($nodes|Where-Object {$_.Visible -and $_.PID -ne $app.Id}).Count){return $null}
 $edits=@($nodes|Where-Object {$_.Visible -and $_.PID -eq $app.Id -and $_.Class -eq 'Edit'})
 if($edits.Count -gt 1){throw 'Multiple owned preview edits'}
 if($edits.Count -eq 1){return [pscustomobject]@{Handle=$edits[0].Handle;Text=[DcmNative]::Text([IntPtr]$edits[0].Handle,$app.Id)}}
 return $null
}
function TextBody([string]$name,[string]$plain){
 Wait ('text '+$name) {$p=PreviewText;($null -ne $p) -and $p.Text -ceq $plain}
 $p=PreviewText;if($p.Text -cne $plain){throw 'Expected exact recovered text fixture'}
 RequireStatus 'Plain-text preview: TXT'
 [IO.File]::WriteAllText((Join-Path $lab ($name+'.displayed.txt')),$p.Text,$utf8)
 Record $name ([ordered]@{displayed_sha256=(Hash (Join-Path $lab ($name+'.displayed.txt')))})
}
function CaptureFrame([string]$name,[bool]$preview=$false){
 Own;if($preview -and $null -ne $nativeTarget){[DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget)};$s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false)
 $shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false;kind='full_window';left=$s.Left;top=$s.Top;width=$s.Width;height=$s.Height})
 if(-not $preview){return $null}
 # Read physical PMv2 coordinates after the exact saved frame. Require the
 # current root rectangle to agree with its capture metadata before cropping.
 $geometry=[DcmPreviewGeometry]::Observe($hwnd,$app.Id)
 $geometryPath=Join-Path $lab ($name+'.geometry.json');Json $geometryPath $geometry
 $contexts=@($geometry.Contexts|Where-Object {$_.CallerContextLabel -ceq 'per_monitor_aware_v2_requested_minus4'})
 if($contexts.Count -ne 1){throw 'Physical preview observation missing'}
 $roots=@($contexts[0].Windows|Where-Object {$_.Role -ceq 'host' -and $_.Available -and $_.Visible -and $_.WindowRectAvailable})
 $panes=@($contexts[0].Windows|Where-Object {$_.Role -ceq 'owned_preview_pane' -and $_.Available -and $_.Visible -and $_.WindowRectAvailable})
 if($roots.Count -ne 1 -or $panes.Count -ne 1){throw 'Owned PPT capture geometry unavailable'}
 $root=$roots[0].WindowRect;$region=$panes[0].WindowRect
 if($root.Left -ne $s.Left -or $root.Top -ne $s.Top -or $root.Right-$root.Left -ne $s.Width -or $root.Bottom-$root.Top -ne $s.Height){throw 'Host moved or resized after source capture'}
 if($region.Left -lt $root.Left -or $region.Top -lt $root.Top -or $region.Right -gt $root.Right -or $region.Bottom -gt $root.Bottom -or $region.Right-$region.Left -lt 100 -or $region.Bottom-$region.Top -lt 80){throw 'Preview pane escapes full captured frame'}
 $path=Join-Path $lab ($name+'.preview.png');$bitmap=New-Object Drawing.Bitmap($s.Path)
 try{
  if($bitmap.Width -ne $s.Width -or $bitmap.Height -ne $s.Height){throw 'Saved screenshot dimensions changed'}
  $rectangle=New-Object Drawing.Rectangle(($region.Left-$s.Left),($region.Top-$s.Top),($region.Right-$region.Left),($region.Bottom-$region.Top))
  $crop=$bitmap.Clone($rectangle,[Drawing.Imaging.PixelFormat]::Format32bppArgb)
  try{$crop.Save($path,[Drawing.Imaging.ImageFormat]::Png)}finally{$crop.Dispose()}
 }finally{$bitmap.Dispose()}
 Own;if($preview -and $null -ne $nativeTarget){[DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget)};$ink=[DcmPptxInput]::InspectPixels($path)
 $shots.Add([ordered]@{name=($name+'.preview');path=$path;sha256=(Hash $path);visual_reviewed=$false;kind='same_frame_preview_pixels';source_frame=$s.Path;geometry=$geometryPath;geometry_sha256=(Hash $geometryPath);left=$region.Left;top=$region.Top;pixels=$ink})
 return [pscustomobject]@{path=$path;pixels=$ink}
}
function PositionForInitialText {
 Own;$screen=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea;[DcmNative]::Restore($hwnd,$app.Id,$screen.Left+8,$screen.Top+8);Start-Sleep -Milliseconds 300
}
function Registration([string]$extension='.ppt') {
 $key='{8895b1c6-b41f-4c1c-a562-0d564250836f}';$reg=[Microsoft.Win32.Registry]::ClassesRoot;$r=$reg.OpenSubKey($extension);$prog=''
 if($r){try{if($r.GetValueNames() -contains '' -and $r.GetValueKind('') -eq [Microsoft.Win32.RegistryValueKind]::String){$prog=[string]$r.GetValue('')}}finally{$r.Dispose()}}
 $keys=@();if($prog){$keys+=($prog+'\shellex\'+$key)};$keys+=($extension+'\shellex\'+$key);$keys+=('SystemFileAssociations\'+$extension+'\shellex\'+$key)
 $rows=@();$selected='';$selectedKey=''
 foreach($k in $keys){$r=$reg.OpenSubKey($k);$value=$null;if($r){try{if($r.GetValueNames() -contains '' -and $r.GetValueKind('') -eq [Microsoft.Win32.RegistryValueKind]::String){$value=[string]$r.GetValue('')}}finally{$r.Dispose()}};$rows+=[ordered]@{key=$k;value=$value};if(-not $selected -and $null -ne $value -and $value.Trim().StartsWith('{',[StringComparison]::Ordinal)){$selected=$value.Trim();$selectedKey=$k}}
 return [pscustomobject]@{extension=$extension;prog_id=$prog;candidates=$rows;selected_clsid=$selected;selected_key=$selectedKey;read_only=$true}
}
function StableFrame([string]$name,[string]$state,[scriptblock]$revalidate) {
 $frame=$null;$previous='';$stable=$false;$until=[DateTime]::UtcNow.AddSeconds(8)
 for($attempt=1;$attempt -le 8 -and [DateTime]::UtcNow -lt $until;$attempt++){
  Own;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected owned dialog during preview capture'};& $revalidate;Start-Sleep -Milliseconds 200
  $frame=CaptureFrame ($name+'-attempt-'+$attempt.ToString('00')) $true
  $ink=$frame.pixels.DarkPixels -ge 20 -and $frame.pixels.LightPixels -ge 100;$difference=$null
  if($previous){$difference=[DcmPptxInput]::PixelDifference($previous,$frame.path)}
  $stable=$ink -and $null -ne $difference -and $difference -le 30
  $receipt.observations.Add([ordered]@{stage=$name;attempt=$attempt;state=$state;path=$frame.path;pixels=$frame.pixels;changed_since_previous=$difference;nonblank=[bool]$ink;stable=[bool]$stable;utc=[DateTime]::UtcNow.ToString('o')})
  if($stable){break};$previous=if($ink){$frame.path}else{''}
 }
 if(-not $stable){throw ('DCM33_PREVIEW_PIXELS: no stable nonblank '+$state+' preview in '+$name)}
 & $revalidate;return $frame
}
function PresentationState {
 Own;$captions=HostCaptions;$native=@(NativeNodes);$text=PreviewText
 if($captions -ccontains 'Windows preview: PPT' -and $native.Count -gt 0){return 'native'}
 if($captions -ccontains $unavailableStatus){
  if($native.Count -or $null -ne $text){throw 'DCM33_STALE_PREVIEW: unavailable PPT retains an old renderer or text body'}
  if($captions -ccontains 'Plain-text preview: TXT' -or $captions -ccontains 'Windows preview: PPT'){throw 'DCM33_STALE_PROVIDER_STATUS: unavailable PPT retains a previous success footer'}
  return 'unavailable'
 }
 return 'pending'
}
function WaitPresentationState {
 $until=[DateTime]::UtcNow.AddSeconds(20)
 do{Own;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected owned PPT dialog'};$state=PresentationState;if($state -ne 'pending'){return $state};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
 $captions=HostCaptions
 if(@(NativeNodes).Count -eq 0 -and ($captions -ccontains 'Plain-text preview: TXT' -or $captions -ccontains 'Windows preview: PPT')){throw 'DCM33_STALE_PROVIDER_STATUS: PPT has no native content and retains the previous TXT/PPT success footer'}
 throw 'DCM33_PPT_STATE: neither native PPT preview nor the exact honest unavailable status was observed'
}
function UnavailableObservation([string]$name) {
 if((WaitPresentationState) -cne 'unavailable'){throw 'Expected the naturally unavailable PPT state'}
 $frame=StableFrame $name 'unavailable' {if((PresentationState) -cne 'unavailable'){throw 'PPT unavailable state changed during capture'}}
 $captions=HostCaptions;Json (Join-Path $lab ($name+'.host-captions.json')) $captions
 Json (Join-Path $lab ($name+'.native-windows.json')) @([DcmDocxProbe]::Snapshot($hwnd,$app.Id))
 RequireStatus $unavailableStatus
 Record $name ([ordered]@{state='unavailable';actual_host_captions=$captions;accepted_capture=$frame.path;pixels=$frame.pixels;native_subtree_absent=$true;previous_text_edit_absent=$true;unavailable_drawing_text_machine_asserted=$false;semantic_image_review_required=$true;slide_navigation_exercised=$false})
}
function NativePageKey([string]$name,[uint32]$key){
 Own;RequireStatus 'Windows preview: PPT';[DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget)
 $action=[ordered]@{name=$name;virtual_key=$key;method='WM_KEYDOWN/WM_KEYUP to bound native childClass only';target=$nativeTarget;queued=$false;started_utc=[DateTime]::UtcNow.ToString('o')}
 $receipt.native_input_actions.Add($action)
 [DcmPptxInput]::QueueNativePageKey($hwnd,$app.Id,$nativeTarget,$key)
 $action.queued=$true;$action['queued_utc']=[DateTime]::UtcNow.ToString('o')
}
function NativeShot([string]$name,[int]$slideNumber=1,[string]$comparison='initial',[string]$baseline=''){
 if($slideNumber -notin @(1,2) -or $comparison -notin @('initial','different','restore')){throw 'Invalid native slide expectation'}
 if($comparison -ne 'initial' -and -not $baseline){throw 'Native slide comparison requires its first-frame baseline'}
 Wait 'native PPT provider and visible subtree' {((HostCaptions) -ccontains 'Windows preview: PPT') -and @(NativeNodes).Count -gt 0}
 Own;[DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget)
 Json (Join-Path $lab ($name+'.native-windows.json')) @([DcmDocxProbe]::Snapshot($hwnd,$app.Id))
 # No Restore, resize, repaint or synthetic refresh follows provider attachment
 # or key input. Save every bounded readiness attempt; a later frame never
 # overwrites the first observation. Numeric slide index is not exposed here.
 $frame=$null;$ready=$false;$previous='';$until=[DateTime]::UtcNow.AddSeconds(8)
 for($attempt=1;$attempt -le 8 -and [DateTime]::UtcNow -lt $until;$attempt++){
  Own;if(@([DcmNative]::Dialogs($hwnd,$app.Id)).Count){throw 'Unexpected owned PPT dialog during capture'};[DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget);Start-Sleep -Milliseconds 200
  $frame=CaptureFrame ($name+'-attempt-'+$attempt.ToString('00')) $true
  RequireStatus 'Windows preview: PPT';$changed=$null;$settling=$null
  $pixelCondition=$frame.pixels.DarkPixels -ge 20 -and $frame.pixels.LightPixels -ge 100
  $expectedColours=if($slideNumber -eq 1){$frame.pixels.TealPixels -ge 30 -and $frame.pixels.OrangePixels -ge 30}else{$frame.pixels.OrangePixels -lt 30}
  $pixelCondition=$pixelCondition -and $expectedColours
  if($baseline){$changed=[DcmPptxInput]::PixelDifference($baseline,$frame.path);if($comparison -eq 'different'){$pixelCondition=$pixelCondition -and $changed -ge [Math]::Max(100,[Math]::Ceiling($frame.pixels.SampledPixels*0.005))}elseif($comparison -eq 'restore'){$pixelCondition=$pixelCondition -and $changed -le 30}}
  if($previous){$settling=[DcmPptxInput]::PixelDifference($previous,$frame.path)}
  $ready=$pixelCondition -and $null -ne $settling -and $settling -le 30
  $receipt.native_capture_attempts.Add([ordered]@{stage=$name;attempt=$attempt;expected_slide=$slideNumber;capture=$frame.path;target=$nativeTarget;pixels=$frame.pixels;comparison=$comparison;changed_samples=$changed;changed_since_previous=$settling;pixel_condition=[bool]$pixelCondition;stable=[bool]$ready;utc=[DateTime]::UtcNow.ToString('o')})
  if($ready){break};$previous=if($pixelCondition){$frame.path}else{''}
 }
 if(-not $ready){throw ('DCM33_PPT_NATIVE_SLIDE_PIXELS: expected stable '+$comparison+' slide '+$slideNumber+' content for '+$name)}
 Json (Join-Path $lab ($name+'.host-captions.json')) (HostCaptions)
 $text=[DcmDocxProbe]::ReadAccessibleText($hwnd,$app.Id);[IO.File]::WriteAllText((Join-Path $lab ($name+'.accessible.txt')),$text,$utf8)
 foreach($errorText in @("This file can't be previewed",'This file cannot be previewed','This file could not be previewed','Preview unavailable')){if($text.IndexOf($errorText,[StringComparison]::OrdinalIgnoreCase) -ge 0){throw ('DCM33_NATIVE_ERROR: rendered provider reports '+$errorText)}}
 [DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget)
 RequireStatus 'Windows preview: PPT'
 Record $name ([ordered]@{native_target=$nativeTarget;expected_slide=$slideNumber;slide_oracle=$slideOracles[$slideNumber-1];programmatic_slide_index_available=$false;accessible_text_sha256=(Hash (Join-Path $lab ($name+'.accessible.txt')));rendered_text_machine_asserted=$false;rendered_content_review='required_separately';status='Windows preview: PPT';accepted_capture=$frame.path;pixels=$frame.pixels;changed_samples=$changed;changed_since_previous=$settling})
 return $frame
}
function Exclusive {$h=[IO.File]::Open((Join-Path $fixture.inbox '07-slides.ppt'),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()}
function StopApp {
 if(-not $app){return};Own
 $children=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$app.Id)|ForEach-Object {[ordered]@{pid=[int]$_.ProcessId;path=$_.ExecutablePath;created=$_.CreationDate.ToUniversalTime().ToString('o')}})
 [DcmNative]::CloseAsync($hwnd,$app.Id);if(-not $app.WaitForExit(15000)){throw 'Owned host did not close normally'}
 $owned['exit_code']=$app.ExitCode
 foreach($c in $children){$live=Get-CimInstance Win32_Process -Filter ('ProcessId='+$c.pid);if($live -and $live.ExecutablePath -eq $c.path -and $live.CreationDate.ToUniversalTime().ToString('o') -eq $c.created){throw 'Owned direct child survived close'}}
 $late=@(Get-CimInstance Win32_Process -Filter ('ParentProcessId='+$app.Id)|Where-Object {$_.CreationDate.ToUniversalTime() -ge [DateTime]::Parse($owned.start).ToUniversalTime() -and $_.CreationDate.ToUniversalTime() -le $app.ExitTime.ToUniversalTime()})
 if($late.Count){throw 'Owned renderer remains after close'};$code=$app.ExitCode;$app.Dispose();$script:app=$null;if($code -ne 0){throw 'Host nonzero exit'}
}
Write-Output ('REPORT_ROOT='+$lab)
try {
 $build=Get-Content $BuildReceipt -Raw -Encoding UTF8|ConvertFrom-Json;$head=Git @('rev-parse','HEAD')
 if(-not $build.passed -or $build.mode -ne 'build' -or -not $build.source_unchanged -or $build.commit -ne $head){throw 'Exact clean build required'}
 if((Hash $build.source_hash_manifest.path) -ne $build.source_hash_manifest.sha256 -or (Hash $build.app_artifact.path) -ne $build.app_artifact.sha256){throw 'Build identity changed'}
 $manifest=Get-Content $build.source_hash_manifest.path -Raw -Encoding UTF8|ConvertFrom-Json;Source
 $receipt['commit']=$head;$receipt['build_receipt']=$BuildReceipt;$receipt['app_sha256']=$build.app_artifact.sha256
 $definition=Get-Content $fixtureManifest -Raw -Encoding UTF8|ConvertFrom-Json
 foreach($field in @('schema_version','file','sha256','bytes','slide_count','required_text','slide_oracles')){if($null -eq $definition -or -not $definition.PSObject.Properties[$field]){throw ('PPT fixture manifest field missing: '+$field)}}
 $presentationHash=[string]$definition.sha256;$requiredText=@($definition.required_text);$slideOracles=@($definition.slide_oracles)
 if($definition.schema_version -ne 1 -or $definition.file -cne 'dcm33-legacy-ppt.ppt' -or $definition.slide_count -ne 2 -or $presentationHash -cnotmatch '^[0-9a-f]{64}$' -or $definition.bytes -ne (Get-Item -LiteralPath $presentation).Length -or (Hash $presentation) -cne $presentationHash){throw 'Committed PPT fixture identity differs from its pinned two-slide manifest'}
 if($slideOracles.Count -ne 2 -or $requiredText.Count -lt 1 -or $requiredText.Count -gt 20 -or @($requiredText|Where-Object {$_ -isnot [string] -or [string]::IsNullOrWhiteSpace($_) -or $_.Length -gt 1024}).Count){throw 'PPT text and slide oracle bounds invalid'}
 $pinnedSlideText=@(@('Bridge Inspection','September 2026'),@('Payments','Scheme','Amount','Bridge','125000','Road','86000'))
 $allSlideText=@()
 for($i=0;$i -lt 2;$i++){
  $oracle=$slideOracles[$i]
  foreach($field in @('number','required_text')){if($null -eq $oracle -or -not $oracle.PSObject.Properties[$field]){throw ('PPT slide oracle field missing: '+$field)}}
  $tokens=@($oracle.required_text)
  if($oracle.number -ne $i+1 -or $tokens.Count -ne $pinnedSlideText[$i].Count -or @($tokens|Where-Object {$_ -isnot [string]}).Count -or ($tokens -join "`n") -cne ($pinnedSlideText[$i] -join "`n")){throw 'PPT slide expectations differ from the representative fixture'}
  $allSlideText+=$tokens
 }
 $uniqueText=@($allSlideText|Select-Object -Unique)
 if(($requiredText -join "`n") -cne ($uniqueText -join "`n")){throw 'PPT required_text does not match the two ordered slide oracles'}
 $bytes=[IO.File]::ReadAllBytes($presentation);if($bytes.Length -lt 512 -or [BitConverter]::ToString($bytes,0,8) -cne 'D0-CF-11-E0-A1-B1-1A-E1'){throw 'Legacy PPT must have CFBF magic, not renamed Open XML or plain text'}
 $fixtureEntries=@($manifest|Where-Object {$_.path -ceq 'tests/fixtures/dcm33-legacy-ppt.ppt'});$manifestEntries=@($manifest|Where-Object {$_.path -ceq 'tests/fixtures/dcm33-legacy-ppt.manifest.json'})
 if($fixtureEntries.Count -ne 1 -or $fixtureEntries[0].sha256 -cne $presentationHash -or $manifestEntries.Count -ne 1 -or $manifestEntries[0].sha256 -cne (Hash $fixtureManifest)){throw 'PPT fixture and manifest must belong to this exact native build snapshot'}
 $receipt['presentation_source']=$presentation;$receipt['presentation_sha256']=$presentationHash;$receipt['fixture_manifest_sha256']=Hash $fixtureManifest;$receipt['required_presentation_text']=$requiredText;$receipt['slide_oracles']=$slideOracles;$receipt['cfbf_header_verified']=$true
 Add-Type -AssemblyName System.Drawing;Add-Type -AssemblyName System.Windows.Forms
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-pptx-input.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-preview-geometry.cs') -ReferencedAssemblies @('System.dll','System.Core.dll')
 Add-Type -AssemblyName UIAutomationClient;Add-Type -AssemblyName UIAutomationTypes;Add-Type -AssemblyName WindowsBase
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-docx-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll',([Windows.Automation.AutomationElement]).Assembly.Location,([Windows.Automation.AutomationPattern]).Assembly.Location,([Windows.Point]).Assembly.Location)
 . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
 Copy-Item -LiteralPath $presentation -Destination (Join-Path $fixture.inbox '07-slides.ppt')
 Get-ChildItem $fixture.inbox -File|ForEach-Object {$expected[$_.Name]=Hash $_.FullName};$receipt['source_fixture_hashes']=$expected
 $registrations=Registration;$handlerRegistered=[bool]$registrations.selected_clsid;$receipt['handler_registrations_read_only']=$registrations
 $unavailableStatus=if($handlerRegistered){'Preview unavailable: PPT (Windows handler failed)'}else{'Preview unavailable: PPT (no Windows preview handler installed)'}
 $receipt['expected_unavailable_status']=$unavailableStatus
 $receipt['expected_unavailable_drawing_text']=if($handlerRegistered){@('Preview unavailable','Windows has a preview handler for PPT files, but it could not preview this file.','Name: 07-slides.ppt','You can still rename, move, open, or open its folder.')}else{@('No preview available','No Windows preview handler is installed for PPT files.','Name: 07-slides.ppt','You can still rename, move, open, or open its folder.')}
 $receipt['scope_limit']='PPT only: naturally selected installed-handler or honest unavailable path. Bound PageDown/PageUp input and stable pixel comparisons corroborate native 1 -> 2 -> 1; exact slide meanings require separate image review. No registry mutation, forced absent provider, arbitrary deck fidelity, animations, damaged/protected files, DOC/XLS rerun or complete DCM-33 acceptance.'
 $exe=Join-Path $lab ('DCM-LEGACY-PPT-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item $build.app_artifact.path $exe
 $temp=Join-Path $lab 'temp';$null=New-Item -ItemType Directory $temp
 $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
 $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming;$psi.EnvironmentVariables['TEMP']=$temp;$psi.EnvironmentVariables['TMP']=$temp;$psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
 foreach($var in @('DCM_PDF_PROVIDER','DCM_SUMATRA_EXE','DCM_STARTUP_TRACE','DCM_LAYOUT_TRACE')){$psi.EnvironmentVariables.Remove($var)}
 $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'Host launch failed'};$owned=[ordered]@{pid=$app.Id;path=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')};$receipt['owned_process']=$owned
 $until=[DateTime]::UtcNow.AddSeconds(20);do{$app.Refresh();if($app.HasExited){throw 'Host exited before window'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
 Wait 'fixture list' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $expected.Count}
 SelectFile '01-alpha.txt';$plain=[IO.File]::ReadAllText((Join-Path $fixture.inbox '01-alpha.txt'));TextBody '01-initial-text' $plain;PositionForInitialText;$null=CaptureFrame '01-initial-text'
 SelectFile '07-slides.ppt';$state=WaitPresentationState;$native=$state -eq 'native'
 $receipt['registered_handler_present']=$handlerRegistered;$receipt['native_presentation_preview_observed']=$native;$receipt['missing_handler_case_exercised']=(-not $handlerRegistered -and -not $native);$receipt['registered_handler_failure_case_exercised']=($handlerRegistered -and -not $native)
 if($native){
  $receipt.provider='Installed Windows PPT preview handler'
  $nativeTarget=[DcmPptxInput]::BindNativeSlide($hwnd,$app.Id,'07-slides.ppt');$receipt['initial_native_target']=$nativeTarget
  $first=NativeShot '02-native-slide1'
  NativePageKey 'page-down' 0x22;$second=NativeShot '03-native-slide2' 2 'different' $first.path
  NativePageKey 'page-up' 0x21;$returned=NativeShot '04-native-slide1-return' 1 'restore' $first.path
  $receipt.native_slide_navigation='Bound PowerPoint childClass PageDown/PageUp; 1 -> 2 -> 1 pixel change/restoration and stable captures pass; semantic slide identity requires review'
  $receipt.all_fixture_slides_captured=$true
 }else{
  $receipt.provider='PPT preview unavailable; exact footer checked, GDI guidance needs separate image review'
  UnavailableObservation '02-ppt-unavailable'
 }
 $nativeTarget=$null
 SelectFile '01-alpha.txt';TextBody '05-text-recovery' $plain;$null=CaptureFrame '05-text-recovery'
 if(@(NativeNodes).Count){throw 'Native PPT subtree retained after text selection'}
 Exclusive;Record '06-presentation-released' $null
 SelectFile '07-slides.ppt';$reopenedState=WaitPresentationState
 if($reopenedState -cne $state){throw 'PPT provider state changed between first open and reopening'}
 if($native){
  $nativeTarget=[DcmPptxInput]::BindNativeSlide($hwnd,$app.Id,'07-slides.ppt');$receipt['reopened_native_target']=$nativeTarget
  $reopened=NativeShot '07-native-slide1-reopened'
 }else{UnavailableObservation '07-ppt-unavailable-reopened'}
 Files;StopApp;Source;Exclusive
 if((Hash $presentation) -cne $presentationHash -or (Hash $exe) -ne $build.app_artifact.sha256){throw 'Original presentation/app bytes changed'}
 $after=Registration;$receipt['handler_registrations_after']=$after
 if(($registrations|ConvertTo-Json -Depth 8 -Compress) -cne ($after|ConvertTo-Json -Depth 8 -Compress)){throw 'PPT handler registrations changed during qualification'}
 if($native){
  if(-not $receipt.all_fixture_slides_captured -or $receipt.native_input_actions.Count -ne 2 -or $cases.Count -lt 7 -or $shots.Count -lt 18){throw 'Incomplete native two-slide PPT qualification records/captures'}
 }elseif($cases.Count -lt 5 -or $shots.Count -lt 10){throw 'Incomplete unavailable PPT qualification records/captures'}
 Record '08-normal-close-and-original-bytes' $null;$receipt['source_unchanged']=$true;$receipt['fixtures_preserved']=$true;$receipt['journal_empty']=$true;$receipt.passed=$true
}catch{
 $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)
 if($app){try{Own;Json (Join-Path $lab 'failure-preview-windows.json') @([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$receipt['failure_host_captions']=HostCaptions}catch{$receipt['diagnostic_error']=$_.Exception.Message};try{$null=CaptureFrame 'failure'}catch{$receipt['capture_error']=$_.Exception.Message}}
}finally{try{StopApp}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message};$receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt;Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))}
if(-not $receipt.passed){exit 1}
