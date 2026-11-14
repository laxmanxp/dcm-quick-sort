[CmdletBinding()]
param([string]$BuildReceipt=$env:DCM_BUILD_RECEIPT)
# DCM-32: natural installed-handler/fallback routing, exact committed fixture.
# Native input is scoped to the observed unique PowerPoint childClass directly
# inside the owned preview pane. Posted Page Down/Up never uses global input.
# Screen pixels corroborate navigation; exact slide wording requires review.
$ErrorActionPreference='Stop';Set-StrictMode -Version Latest
if($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitProcess){throw 'Native Windows x64 required'}
if(-not $BuildReceipt -or -not $env:FLOWDEV_WORKSPACE_ROOT){throw 'Exact build and workspace required'}
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$presentation=Join-Path $repo 'tests/fixtures/dcm32-slides.pptx'
$fixtureManifest=Join-Path $repo 'tests/fixtures/dcm32-slides.manifest.json'
$lab=Join-Path $env:FLOWDEV_WORKSPACE_ROOT ('dcm-quicksort-ui-pptx-'+[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')+'-'+[Guid]::NewGuid().ToString('N').Substring(0,8))
if($lab.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)){throw 'Test lab must be outside source'}
$null=New-Item -ItemType Directory $lab;$utf8=New-Object Text.UTF8Encoding($false)
$cases=New-Object 'Collections.Generic.List[object]';$shots=New-Object 'Collections.Generic.List[object]'
$receipt=[ordered]@{schema=1;feature='DCM-32';report=$lab;started_utc=[DateTime]::UtcNow.ToString('o');passed=$false;feature_accepted=$false;cases=$cases;captures=$shots;visual_review_required=$true;user_test='not_recorded';provider='not_observed';native_slide_navigation='not_exercised';fallback_slide_navigation='not_exercised';all_fixture_slides_captured=$false;native_input_actions=(New-Object 'Collections.Generic.List[object]');native_capture_attempts=(New-Object 'Collections.Generic.List[object]')}
$app=$null;$hwnd=[IntPtr]::Zero;$owned=$null;$head='';$manifest=@();$fixture=$null;$expected=@{};$exe='';$presentationHash='';$nativeTarget=$null
function Hash([string]$p){return (Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Json([string]$p,$v){[IO.File]::WriteAllText($p,($v|ConvertTo-Json -Depth 25),$utf8)}
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
function RequireStatus([string]$text){if(-not ((HostCaptions) -ccontains $text)){throw ('PPTX provider/status mismatch: '+$text)}}
function StatusPrefix([string]$prefix){$matched=@((HostCaptions)|Where-Object {$_.StartsWith($prefix,[StringComparison]::Ordinal)});if($matched.Count -ne 1){throw ('Expected one provider status: '+$prefix)};return $matched[0]}
function NativeNodes {
 # Require a nonempty, visible subtree within the exact owned preview pane as
 # well as the native footer. The provider can live in the host or a surrogate.
 return @([DcmDocxProbe]::Snapshot($hwnd,$app.Id)|Where-Object {$_.Visible -and $_.Right -gt $_.Left -and $_.Bottom -gt $_.Top -and ($_.PID -ne $app.Id -or $_.Class -notin @('Edit','ComboBox'))})
}
function PreviewText {
 if((HostCaptions) -ccontains 'Windows preview: PPTX'){return $null}
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
function CaptureFrame([string]$name,[bool]$slide,[bool]$requireInk=$true){
 Own;if($slide -and $null -ne $nativeTarget){[DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget)};$s=[DcmNative]::Capture($hwnd,$app.Id,(Join-Path $lab ($name+'.png')),$false)
 $shots.Add([ordered]@{name=$name;path=$s.Path;sha256=(Hash $s.Path);visual_reviewed=$false;kind='full_window';left=$s.Left;top=$s.Top;width=$s.Width;height=$s.Height})
 if(-not $slide){return $null}
 # Crop the exact full-window pixels with physical HWND coordinates. This
 # preserves native slide titles and avoids a second capture with mismatched DPI.
 $crop=[DcmPptxInput]::CropFromFull($hwnd,$app.Id,$nativeTarget,$s.Path,$s.Left,$s.Top,(Join-Path $lab ($name+'.preview.png')))
 $ink=[DcmPptxInput]::InspectPixels($crop.Path)
 $shots.Add([ordered]@{name=($name+'.preview');path=$crop.Path;sha256=(Hash $crop.Path);visual_reviewed=$false;kind='same_frame_preview_pixels';left=$crop.Left;top=$crop.Top;width=$crop.Width;height=$crop.Height;pixels=$ink})
 # These conservative counters reject an empty/single-colour pane. They cannot
 # prove words, correct slide order or Office fidelity; every frame is reviewed.
 if($requireInk -and ($ink.DarkPixels -lt 20 -or $ink.LightPixels -lt 100)){throw ('DCM32_BLANK_PREVIEW: insufficient visible slide ink/light background in '+$name)}
 return [pscustomobject]@{path=$crop.Path;pixels=$ink}
}
function Shot([string]$name,[bool]$slide=$false){Own;$screen=[Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea;[DcmNative]::Restore($hwnd,$app.Id,$screen.Left+8,$screen.Top+8);Start-Sleep -Milliseconds 300;return (CaptureFrame $name $slide)}
function NativePageKey([string]$name,[uint32]$key){
 Own;RequireStatus 'Windows preview: PPTX';[DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget)
 $action=[ordered]@{name=$name;virtual_key=$key;method='WM_KEYDOWN/WM_KEYUP to bound native childClass only';target=$nativeTarget;queued=$false;started_utc=[DateTime]::UtcNow.ToString('o')}
 $receipt.native_input_actions.Add($action)
 [DcmPptxInput]::QueueNativePageKey($hwnd,$app.Id,$nativeTarget,$key)
 $action.queued=$true;$action['queued_utc']=[DateTime]::UtcNow.ToString('o')
}
function NativeShot([string]$name,[int]$slideNumber=1,[string]$comparison='initial',[string]$baseline=''){
 if($slideNumber -notin @(1,2) -or $comparison -notin @('initial','different','restore')){throw 'Invalid native slide expectation'}
 if($comparison -ne 'initial' -and -not $baseline){throw 'Native slide comparison requires its first-frame baseline'}
 Wait 'native PPTX provider and visible subtree' {((HostCaptions) -ccontains 'Windows preview: PPTX') -and @(NativeNodes).Count -gt 0}
 Own;[DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget)
 Json (Join-Path $lab ($name+'.native-windows.json')) @([DcmDocxProbe]::Snapshot($hwnd,$app.Id))
 # No Restore, resize, repaint or synthetic refresh follows provider attachment
 # or key input. Save every bounded readiness attempt; a later frame never
 # overwrites the first observation. Numeric slide index is not exposed here.
 $frame=$null;$ready=$false;$previous='';$until=[DateTime]::UtcNow.AddSeconds(8)
 for($attempt=1;$attempt -le 8 -and [DateTime]::UtcNow -lt $until;$attempt++){
  Own;[DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget);Start-Sleep -Milliseconds 200
  $frame=CaptureFrame ($name+'-attempt-'+$attempt.ToString('00')) $true $false
  RequireStatus 'Windows preview: PPTX';$changed=$null;$settling=$null
  $pixelCondition=$frame.pixels.DarkPixels -ge 20 -and $frame.pixels.LightPixels -ge 100
  $expectedColours=if($slideNumber -eq 1){$frame.pixels.TealPixels -ge 30 -and $frame.pixels.OrangePixels -ge 30}else{$frame.pixels.OrangePixels -lt 30}
  $pixelCondition=$pixelCondition -and $expectedColours
  if($baseline){$changed=[DcmPptxInput]::PixelDifference($baseline,$frame.path);if($comparison -eq 'different'){$pixelCondition=$pixelCondition -and $changed -ge [Math]::Max(100,[Math]::Ceiling($frame.pixels.SampledPixels*0.005))}elseif($comparison -eq 'restore'){$pixelCondition=$pixelCondition -and $changed -le 30}}
  if($previous){$settling=[DcmPptxInput]::PixelDifference($previous,$frame.path)}
  $ready=$pixelCondition -and $null -ne $settling -and $settling -le 30
  $receipt.native_capture_attempts.Add([ordered]@{stage=$name;attempt=$attempt;expected_slide=$slideNumber;capture=$frame.path;target=$nativeTarget;pixels=$frame.pixels;comparison=$comparison;changed_samples=$changed;changed_since_previous=$settling;pixel_condition=[bool]$pixelCondition;stable=[bool]$ready;utc=[DateTime]::UtcNow.ToString('o')})
  if($ready){break};$previous=if($pixelCondition){$frame.path}else{''}
 }
 if(-not $ready){throw ('DCM32_NATIVE_SLIDE_PIXELS: expected stable '+$comparison+' slide '+$slideNumber+' content for '+$name)}
 $text=[DcmDocxProbe]::ReadAccessibleText($hwnd,$app.Id);[IO.File]::WriteAllText((Join-Path $lab ($name+'.accessible.txt')),$text,$utf8)
 [DcmPptxInput]::ValidateNativeSlide($hwnd,$app.Id,$nativeTarget)
 Record $name ([ordered]@{native_target=$nativeTarget;expected_slide=$slideNumber;programmatic_slide_index_available=$false;accessible_text_sha256=(Hash (Join-Path $lab ($name+'.accessible.txt')));rendered_text_machine_asserted=$false;rendered_content_review='required_separately';status='Windows preview: PPTX';accepted_capture=$frame.path;pixels=$frame.pixels;changed_samples=$changed;changed_since_previous=$settling})
 return $frame
}
function FallbackSlide([string]$name,[int]$index,[string]$label){
 Wait ('fallback slide '+($index+1)) {([DcmPptxInput]::SelectedSlide($hwnd,$app.Id) -ceq $label) -and @((HostCaptions)|Where-Object {$_.StartsWith(('DCM PPTX Preview: Slide '+($index+1)+' of 2'),[StringComparison]::Ordinal)}).Count -eq 1}
 $status=StatusPrefix ('DCM PPTX Preview: Slide '+($index+1)+' of 2')
 if($status.Contains('unavailable') -or $status.Contains('hidden slide')){throw 'Unexpected fixture slide error/hidden state'}
 $frame=Shot $name $true
 if($index -eq 0 -and ($frame.pixels.TealPixels -lt 30 -or $frame.pixels.OrangePixels -lt 30)){throw ('DCM32_PICTURE_NOT_VISIBLE: first-slide image colours absent in '+$name)}
 Record $name ([ordered]@{selected_slide=$label;status=$status;pixels=$frame.pixels;rendered_text_machine_asserted=$false;rendered_content_review='required_separately'})
 return $frame
}
function Exclusive {$h=[IO.File]::Open((Join-Path $fixture.inbox '07-slides.pptx'),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None);$h.Dispose()}
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
 foreach($field in @('schema_version','feature','file','slide_count','sha256','bytes')){if($null -eq $definition -or -not $definition.PSObject.Properties[$field]){throw ('PPTX fixture manifest field missing: '+$field)}}
 $presentationHash='af42fff55297109195b95039d2ac39c5d53b1fae23b59e7b40a7184d679b93f1'
 if($definition.schema_version -ne 1 -or $definition.feature -cne 'DCM-32' -or $definition.file -cne 'dcm32-slides.pptx' -or $definition.slide_count -ne 2 -or $definition.sha256 -cne $presentationHash -or $definition.bytes -ne (Get-Item -LiteralPath $presentation).Length -or (Hash $presentation) -cne $presentationHash){throw 'Committed PPTX fixture identity differs from pinned two-slide manifest'}
 $fixtureEntries=@($manifest|Where-Object {$_.path -ceq 'tests/fixtures/dcm32-slides.pptx'})
 $fixtureManifestEntries=@($manifest|Where-Object {$_.path -ceq 'tests/fixtures/dcm32-slides.manifest.json'})
 if($fixtureEntries.Count -ne 1 -or $fixtureEntries[0].sha256 -cne $presentationHash -or $fixtureManifestEntries.Count -ne 1 -or $fixtureManifestEntries[0].sha256 -cne (Hash $fixtureManifest)){throw 'PPTX fixture and manifest must belong to the exact native build snapshot'}
 Add-Type -AssemblyName System.Drawing;Add-Type -AssemblyName System.Windows.Forms
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-native.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-undo-input.cs')
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-pptx-input.cs') -ReferencedAssemblies @('System.dll','System.Core.dll','System.Drawing.dll')
 Add-Type -AssemblyName UIAutomationClient;Add-Type -AssemblyName UIAutomationTypes;Add-Type -AssemblyName WindowsBase
 Add-Type -Path (Join-Path $repo 'tests/native/dcm-docx-probe.cs') -ReferencedAssemblies @('System.dll','System.Core.dll',([Windows.Automation.AutomationElement]).Assembly.Location,([Windows.Automation.AutomationPattern]).Assembly.Location,([Windows.Point]).Assembly.Location)
 . (Join-Path $repo 'tests/native/new-fixtures.ps1');$fixture=New-DcmFixtures $lab
 Copy-Item -LiteralPath $presentation -Destination (Join-Path $fixture.inbox '07-slides.pptx')
 Get-ChildItem $fixture.inbox -File|ForEach-Object {$expected[$_.Name]=Hash $_.FullName};$receipt['source_fixture_hashes']=$expected
 $receipt['presentation_source']=$presentation;$receipt['presentation_sha256']=$presentationHash;$receipt['fixture_manifest_sha256']=Hash $fixtureManifest
 $receipt['fixture_scope']='Two-slide qualification deck: Bridge Inspection / September 2026 plus an embedded image, then Payments table Scheme/Amount, Bridge/125000, Road/86000. Exact visible text and picture identity require separate review.'
 $receipt['fallback_limits']='Built-in renderer uses simplified text, table and image geometry; no animation, theme, font, charts, grouped-shape or arbitrary PowerPoint fidelity claim.'
 $key='{8895b1c6-b41f-4c1c-a562-0d564250836f}';$reg=[Microsoft.Win32.Registry]::ClassesRoot;$r=$reg.OpenSubKey('.pptx');$prog='';if($r){try{$prog=[string]$r.GetValue('')}finally{$r.Dispose()}}
 $registrations=@();foreach($k in @(($prog+'\shellex\'+$key),('.pptx\shellex\'+$key),('SystemFileAssociations\.pptx\shellex\'+$key))){$r=$reg.OpenSubKey($k);$value=$null;if($r){try{$value=$r.GetValue('')}finally{$r.Dispose()}};$registrations+=[ordered]@{key=$k;value=$value}};$receipt['handler_registrations_read_only']=$registrations
 $exe=Join-Path $lab ('DCM-PPTX-'+[Guid]::NewGuid().ToString('N')+'.exe');Copy-Item $build.app_artifact.path $exe
 $temp=Join-Path $lab 'temp';$null=New-Item -ItemType Directory $temp
 $psi=New-Object Diagnostics.ProcessStartInfo;$psi.FileName=$exe;$psi.Arguments='"'+(Join-Path $fixture.inbox '01-alpha.txt')+'"';$psi.UseShellExecute=$false;$psi.WorkingDirectory=$lab
 $psi.EnvironmentVariables['LOCALAPPDATA']=$fixture.local;$psi.EnvironmentVariables['APPDATA']=$fixture.roaming;$psi.EnvironmentVariables['TEMP']=$temp;$psi.EnvironmentVariables['TMP']=$temp;$psi.EnvironmentVariables['DCM_QUICKSORT_BACKUP_DIR']=Join-Path $lab 'backups'
 foreach($var in @('DCM_PDF_PROVIDER','DCM_SUMATRA_EXE','DCM_STARTUP_TRACE','DCM_LAYOUT_TRACE')){$psi.EnvironmentVariables.Remove($var)}
 $app=New-Object Diagnostics.Process;$app.StartInfo=$psi;if(-not $app.Start()){throw 'Host launch failed'};$owned=[ordered]@{pid=$app.Id;path=$exe;start=$app.StartTime.ToUniversalTime().ToString('o')};$receipt['owned_process']=$owned
 $until=[DateTime]::UtcNow.AddSeconds(20);do{$app.Refresh();if($app.HasExited){throw 'Host exited before window'};$hwnd=$app.MainWindowHandle;if($hwnd -ne [IntPtr]::Zero){break};Start-Sleep -Milliseconds 100}while([DateTime]::UtcNow -lt $until)
 Wait 'fixture list' {@([DcmNative]::ListItems($hwnd,$app.Id,1016)).Count -eq $expected.Count}
 SelectFile '01-alpha.txt';$plain=[IO.File]::ReadAllText((Join-Path $fixture.inbox '01-alpha.txt'));TextBody '01-initial-text' $plain;$null=Shot '01-initial-text'
 SelectFile '07-slides.pptx'
 Wait 'native PPTX or completed fallback slide' {
  if((HostCaptions) -ccontains 'Windows preview: PPTX'){return (@(NativeNodes).Count -gt 0)}
  return @((HostCaptions)|Where-Object {$_.StartsWith('DCM PPTX Preview: Slide 1 of 2',[StringComparison]::Ordinal)}).Count -eq 1
 }
 $native=(HostCaptions) -ccontains 'Windows preview: PPTX'
 if($native){
  $receipt.provider='Installed Windows PPTX preview handler'
  $nativeTarget=[DcmPptxInput]::BindNativeSlide($hwnd,$app.Id);$receipt['initial_native_target']=$nativeTarget
  $first=NativeShot '02-native-slide1'
  NativePageKey 'page-down' 0x22;$second=NativeShot '03-native-slide2' 2 'different' $first.path
  NativePageKey 'page-up' 0x21;$returned=NativeShot '04-native-slide1-return' 1 'restore' $first.path
  $receipt.native_slide_navigation='Bound PowerPoint childClass PageDown/PageUp; 1 -> 2 -> 1 pixel change/restoration and stable captures pass; semantic slide identity requires review'
  $receipt.all_fixture_slides_captured=$true
 }else{
  $receipt.provider='Built-in PPTX Open XML slide fallback'
  $labels=@([DcmPptxInput]::SlideNames($hwnd,$app.Id));if($labels.Count -ne 2 -or -not $labels[0].StartsWith('Slide 1',[StringComparison]::Ordinal) -or -not $labels[0].Contains('Bridge Inspection') -or -not $labels[1].StartsWith('Slide 2',[StringComparison]::Ordinal) -or -not $labels[1].Contains('Payments')){throw 'Fallback slide labels/order differ'}
  $receipt['fallback_slide_labels']=$labels
  $first=FallbackSlide '02-fallback-slide1' 0 $labels[0]
  [DcmPptxInput]::SelectSlide($hwnd,$app.Id,$labels[1]);$second=FallbackSlide '03-fallback-slide2' 1 $labels[1]
  $difference=[DcmPptxInput]::PixelDifference($first.path,$second.path);if($difference -lt 30){throw 'DCM32_STALE_SLIDE: slide2 preview did not visibly change'}
  Record '03-distinct-slide2-pixels' ([ordered]@{changed_samples=$difference;semantic_slide_identity='requires_review'})
  [DcmPptxInput]::SelectSlide($hwnd,$app.Id,$labels[0]);$returned=FallbackSlide '04-fallback-slide1-return' 0 $labels[0]
  $returnDifference=[DcmPptxInput]::PixelDifference($first.path,$returned.path);if($returnDifference -gt 30){throw 'DCM32_SLIDE_RETURN: slide1 preview did not restore'}
  Record '04-slide1-pixels-restored' ([ordered]@{changed_samples=$returnDifference;semantic_slide_identity='requires_review'})
  $receipt.fallback_slide_navigation='owned selector 1 -> 2 -> 1 readback and visible change/return pass; semantic images still require review'
  $receipt.all_fixture_slides_captured=$true
 }
 $nativeTarget=$null
 SelectFile '01-alpha.txt';TextBody '05-text-recovery' $plain;$null=Shot '05-text-recovery'
 if(@(NativeNodes).Count){throw 'Native PPTX subtree retained after text selection'}
 Exclusive;Record '06-presentation-released' $null
 SelectFile '07-slides.pptx'
 if($native){
  Wait 'reopened native PPTX subtree' {((HostCaptions) -ccontains 'Windows preview: PPTX') -and @(NativeNodes).Count -gt 0}
  $nativeTarget=[DcmPptxInput]::BindNativeSlide($hwnd,$app.Id);$receipt['reopened_native_target']=$nativeTarget
  $reopened=NativeShot '07-native-slide1-reopened'
 }else{$reopened=FallbackSlide '07-fallback-slide1-reopened' 0 $labels[0]}
 Files;StopApp;Source;Exclusive
 if((Hash $presentation) -cne $presentationHash -or (Hash $exe) -ne $build.app_artifact.sha256){throw 'Original presentation/app changed'}
 if(-not $receipt.all_fixture_slides_captured -or $cases.Count -lt 7 -or $shots.Count -lt 10){throw 'Incomplete two-slide PPTX qualification records/captures'}
 Record '08-normal-close-and-original-bytes' $null;$receipt['source_unchanged']=$true;$receipt['fixtures_preserved']=$true;$receipt['journal_empty']=$true;$receipt.passed=$true
}catch{
 $receipt['error']=$_.Exception.Message;$receipt['error_position']=$_.InvocationInfo.PositionMessage;Write-Output ('FAIL='+$_.Exception.Message)
 if($app){try{Own;Json (Join-Path $lab 'failure-preview-windows.json') @([DcmDocxProbe]::Snapshot($hwnd,$app.Id));$receipt['failure_host_captions']=HostCaptions}catch{$receipt['diagnostic_error']=$_.Exception.Message};try{$null=Shot 'failure'}catch{$receipt['capture_error']=$_.Exception.Message}}
}finally{try{StopApp}catch{$receipt.passed=$false;$receipt['cleanup_error']=$_.Exception.Message};$receipt['finished_utc']=[DateTime]::UtcNow.ToString('o');Json (Join-Path $lab 'receipt.json') $receipt;Write-Output ('RECEIPT='+(Join-Path $lab 'receipt.json'))}
if(-not $receipt.passed){exit 1}
