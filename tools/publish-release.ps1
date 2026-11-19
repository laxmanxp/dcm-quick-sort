# prepare is read-only. draft uploads a reviewable GitHub draft; it never publishes.
[CmdletBinding()]
param(
    [ValidateSet('prepare','draft')][string]$Mode='prepare',
    [string]$PackageReceipt=$env:DCM_PACKAGE_RECEIPT,
    [string]$NotesFile=$env:DCM_RELEASE_NOTES,
    [string]$ExpectedCommit=$env:DCM_EXPECTED_COMMIT
)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
$repo=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$repository='laxmanxp/dcm-quick-sort'
$tag='v1.0.71'
$plan=[ordered]@{schema=1;mode=$Mode;repository=$repository;tag=$tag;passed=$false;gh_available=$false;gh_authenticated=$false;automatic_publish=$false;draft_create_attempted=$false}

function Hash([string]$Path){return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()}
function Quote-Argument([string]$Value){return '"'+(($Value -replace '(\\*)"','$1$1\"') -replace '(\\+)$','$1$1')+'"'}
function Invoke-CLI([string]$Name,[string]$File,[string[]]$Arguments,[int]$Seconds=60){
    $psi=New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$File;$psi.Arguments=(($Arguments | ForEach-Object {Quote-Argument $_}) -join ' ');$psi.WorkingDirectory=$repo
    $psi.UseShellExecute=$false;$psi.CreateNoWindow=$true;$psi.RedirectStandardOutput=$true;$psi.RedirectStandardError=$true
    $psi.EnvironmentVariables['GH_HOST']='github.com';$psi.EnvironmentVariables['GH_PROMPT_DISABLED']='1'
    $psi.EnvironmentVariables['GH_NO_UPDATE_NOTIFIER']='1';$psi.EnvironmentVariables['GH_NO_EXTENSION_UPDATE_NOTIFIER']='1'
    $psi.EnvironmentVariables.Remove('GH_DEBUG')
    $proc=New-Object Diagnostics.Process;$proc.StartInfo=$psi
    try{
        if(-not $proc.Start()){throw "$Name could not start."}
        $output=$proc.StandardOutput.ReadToEndAsync();$errors=$proc.StandardError.ReadToEndAsync()
        if(-not $proc.WaitForExit($Seconds*1000)){
            if(-not $proc.HasExited){$proc.Kill()}
            $null=$proc.WaitForExit(10000)
            throw "$Name exceeded its deadline; inspect GitHub before retrying a draft attempt."
        }
        $raw=$output.GetAwaiter().GetResult();$null=$errors.GetAwaiter().GetResult()
        # Never echo native stderr or auth-status output: they can contain credentials.
        if($proc.ExitCode -ne 0){throw "$Name failed (exit $($proc.ExitCode)); native output was suppressed."}
        if($Name -eq 'gh-auth'){return ''}
        return $raw.Trim()
    }finally{$proc.Dispose()}
}
function Git([string[]]$Arguments){return Invoke-CLI 'git-read' $git.Source (@('--no-optional-locks','-C',$repo)+$Arguments)}
function Gh([string]$Name,[string[]]$Arguments){return Invoke-CLI $Name $gh.Source $Arguments}
function Assert-Artifact($Artifact,[string]$Name){
    $path=[IO.Path]::GetFullPath($Artifact.path)
    if([IO.Path]::GetFileName($path) -cne $Name -or $Artifact.sha256 -notmatch '^[a-fA-F0-9]{64}$'){throw "Invalid package artifact identity: $Name"}
    if((Get-Item -LiteralPath $path).Length -ne $Artifact.size -or (Hash $path) -ne $Artifact.sha256){throw "Package artifact bytes changed: $Name"}
    return [ordered]@{name=$Name;path=$path;size=$Artifact.size;sha256=$Artifact.sha256.ToLowerInvariant()}
}
$releaseQuery='.[] | select(.tag_name == "v1.0.71") | {id,tag_name,draft,html_url,target_commitish,assets:[.assets[]|{name,size,digest}]}'
function Release-ForTag {return Gh 'release-read' @('api',"repos/$repository/releases?per_page=100",'--paginate','--jq',$releaseQuery)}

try{
    if($env:OS -ne 'Windows_NT'){throw 'Run this publication helper on Windows with the qualified package files.'}
    $gh=Get-Command gh.exe -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    $git=Get-Command git.exe -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if(-not $gh){throw 'gh.exe availability is unverified: the executable was not found. No authentication or publication was attempted.'}
    if(-not $git){throw 'git.exe is required to bind the package to the committed candidate.'}
    $plan.gh_available=$true;$plan['gh_path']=$gh.Source
    $plan['gh_version']=Gh 'gh-version' @('--version')
    $null=Gh 'gh-auth' @('auth','status','--hostname','github.com')
    $plan.gh_authenticated=$true
    if(-not $PackageReceipt -or -not $NotesFile){throw 'Supply the exact successful package receipt and a reviewed release-notes file; prepare creates neither.'}
    $PackageReceipt=[IO.Path]::GetFullPath($PackageReceipt);$NotesFile=[IO.Path]::GetFullPath($NotesFile)
    $packageHash=Hash $PackageReceipt
    $package=Get-Content -LiteralPath $PackageReceipt -Raw -Encoding UTF8 | ConvertFrom-Json
    if($package.app -ne 'dcm-quicksort' -or $package.mode -ne 'package' -or $package.passed -ne $true -or $package.source_unchanged -ne $true){throw 'Receipt does not prove a successful unchanged-source DCM package run.'}
    if($package.commit -notmatch '^[a-fA-F0-9]{40}$'){throw 'Package receipt has no exact commit.'}
    $commit=$package.commit.ToLowerInvariant()
    if((Git @('rev-parse','HEAD')) -ne $commit -or (Git @('status','--porcelain=v1','--untracked-files=all'))){throw 'Current checkout must be clean and match the package commit exactly.'}
    if((Git @('remote','get-url','origin')) -ne "https://github.com/$repository.git" -or $package.origin -ne "https://github.com/$repository.git"){throw 'Package or checkout origin differs from the intended DCM repository.'}
    if($package.branch -ne 'release/v1.0.71'){throw 'Package was not qualified from release/v1.0.71.'}
    if((Hash $package.source_hash_manifest.path) -ne $package.source_hash_manifest.sha256){throw 'Package source hash manifest changed.'}
    $manifest=Get-Content -LiteralPath $package.source_hash_manifest.path -Raw -Encoding UTF8 | ConvertFrom-Json
    $manifest=@($manifest)
    if($manifest.Count -eq 0 -or $manifest.Count -ne $package.source_hash_manifest.files){throw 'Package source manifest is incomplete.'}
    foreach($file in $manifest){
        $full=[IO.Path]::GetFullPath((Join-Path $repo $file.path))
        if(-not $full.StartsWith($repo+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or (Hash $full) -ne $file.sha256){throw "Current source differs from package: $($file.path)"}
    }
    $appVersion=[regex]::Match((Get-Content -LiteralPath (Join-Path $repo 'app-src\settings.go') -Raw),'const currentAppVersion = "([^"]+)"').Groups[1].Value
    $installerVersion=[regex]::Match((Get-Content -LiteralPath (Join-Path $repo 'installer-src\core.go') -Raw),'productVersion\s*=\s*"([^"]+)"').Groups[1].Value
    if($appVersion -ne '1.0.71' -or $installerVersion -ne '1.0.71'){throw 'Both qualified binaries must identify version 1.0.71.'}
    $assets=@((Assert-Artifact $package.app_artifact 'DCM-QuickSort.exe'),(Assert-Artifact $package.installer_artifact 'DCM-QuickSort-Setup.exe'))
    $notes=Get-Content -LiteralPath $NotesFile -Raw -Encoding UTF8
    if([string]::IsNullOrWhiteSpace($notes)){throw 'Release notes are empty.'}
    $notesHash=Hash $NotesFile
    $plan['commit']=$commit;$plan['package_receipt']=[ordered]@{path=$PackageReceipt;sha256=$packageHash}
    $plan['assets']=$assets;$plan['notes']=[ordered]@{path=$NotesFile;sha256=$notesHash;content=$notes}
    if((Gh 'repository-permissions' @('api',"repos/$repository",'--jq','.permissions.push')) -ne 'true'){throw 'Authenticated account has no verified write permission for the DCM repository.'}
    if((Gh 'remote-commit' @('api',"repos/$repository/git/commits/$commit",'--jq','.sha')) -ne $commit){throw 'The exact packaged commit is not present in the remote repository.'}
    $existing=Release-ForTag
    if($existing){$plan['existing_release']=$existing | ConvertFrom-Json;throw 'A release or draft already uses v1.0.71; inspect it explicitly. This helper never edits or replaces existing releases.'}
    $tagObject=Gh 'remote-tag' @('api',"repos/$repository/git/matching-refs/tags/$tag",'--jq','.[] | select(.ref == "refs/tags/v1.0.71") | .object')
    $plan['tag_exists']=[bool]$tagObject
    if($tagObject){
        $object=$tagObject | ConvertFrom-Json
        for($depth=0;$object.type -eq 'tag' -and $depth -lt 8;$depth++){
            if($object.sha -notmatch '^[a-fA-F0-9]{40}$'){throw 'Remote annotated tag has an invalid object identity.'}
            $object=(Gh 'annotated-tag' @('api',"repos/$repository/git/tags/$($object.sha)",'--jq','.object')) | ConvertFrom-Json
        }
        if($object.type -ne 'commit' -or $object.sha -ne $commit){throw 'Existing v1.0.71 tag does not resolve to the qualified package commit; no tag will be changed.'}
    }
    $plan['draft_arguments']=@('-Mode','draft','-PackageReceipt',$PackageReceipt,'-NotesFile',$NotesFile,'-ExpectedCommit',$commit)
    if($Mode -eq 'draft'){
        if($ExpectedCommit -ne $commit){throw 'Draft requires -ExpectedCommit with the exact commit shown by prepare.'}
        if((Hash $PackageReceipt) -ne $packageHash -or (Hash $NotesFile) -ne $notesHash){throw 'Receipt or notes changed during preparation.'}
        foreach($asset in $assets){if((Hash $asset.path) -ne $asset.sha256){throw 'An upload artifact changed during preparation.'}}
        if((Git @('rev-parse','HEAD')) -ne $commit -or (Git @('status','--porcelain=v1','--untracked-files=all'))){throw 'Candidate changed before draft creation.'}
        $arguments=@('release','create',$tag,$assets[0].path,$assets[1].path,'--repo',$repository,'--draft','--title','DCM QuickSort v1.0.71','--target',$commit,'--notes-file',$NotesFile)
        if($plan.tag_exists){$arguments+='--verify-tag'}
        $plan.draft_create_attempted=$true
        $null=Invoke-CLI 'draft-create' $gh.Source $arguments 180
        $created=(Release-ForTag) | ConvertFrom-Json
        if(@($created).Count -ne 1 -or $created.draft -ne $true -or $created.tag_name -ne $tag -or $created.target_commitish -ne $commit){throw 'Created release did not verify as the requested draft. Inspect GitHub; no automatic retry or publication will occur.'}
        if(@($created.assets).Count -ne 2){throw 'Draft asset count differs from the two qualified executables; inspect the draft.'}
        foreach($asset in $assets){
            $remote=@($created.assets | Where-Object {$_.name -ceq $asset.name})
            if($remote.Count -ne 1 -or $remote[0].size -ne $asset.size){throw "Draft upload metadata differs: $($asset.name)"}
            if($remote[0].digest -and $remote[0].digest -ne ('sha256:'+$asset.sha256)){throw "GitHub upload digest differs: $($asset.name)"}
            if((Hash $asset.path) -ne $asset.sha256){throw 'Local artifact changed during upload; inspect the draft.'}
        }
        $plan['draft_url']=$created.html_url;$plan['draft_id']=$created.id
        $plan['remote_digest_verified']=@($created.assets | Where-Object {-not $_.digest}).Count -eq 0
    }
    $plan.passed=$true
}catch{
    $plan['error']=$_.Exception.Message
    if($plan.draft_create_attempted){$plan['next_step']='Inspect the existing GitHub draft and its assets before any retry; this helper never deletes, replaces, or publishes a release.'}
}finally{
    $plan['finished_utc']=[DateTime]::UtcNow.ToString('o')
    $plan | ConvertTo-Json -Depth 20
}
if(-not $plan.passed){exit 1}
exit 0
