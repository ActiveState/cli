#!/bin/sh
# Copyright 2019 ActiveState Software Inc. All rights reserved.
<#
.DESCRIPTION
Install the ActiveState state.exe tool.  Must be run as admin OR install state tool to
User profile folder.

.EXAMPLE
install.ps1 -b branchToInstall -t C:\dir\on\path
#>

param (
    [Parameter(Mandatory=$False)][string]$b = "unstable"
    ,[Parameter(Mandatory=$False)]
        [ValidateScript({Test-Path $_ -PathType 'Container'})]
        [string]
        $t = (Join-Path $Env:ALLUSERSPROFILE "ActiveState") # C:\ProgramData\ActiveState
    ,[Parameter(Mandatory=$False)][switch]$n
    ,[Parameter(Mandatory=$False)][switch]$h
    ,[Parameter(Mandatory=$False)]
        [ValidateScript({[IO.Path]::GetExtension($_) -eq '.exe'})]
        [string]
        $f = "state.exe"
)

# Helpers
function isInRegistry(){
    $regpaths = (Get-ItemProperty -Path 'Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment' -Name PATH).Path.Split(';')
    $inReg = $False
    for ($i = 0; $i -lt $regpaths.Count; $i++) {
        if ($regpaths[$i] -eq $INSTALLDIR) {
            $inReg = $True
        }
    }
    $inReg
}
function isOnPath(){
    $envpaths = $env:Path.Split(';')
    $inEnv = $False
    for ($i = 0; $i -lt $envpaths.Count; $i++) {
        if ($envpaths[$i] -eq $INSTALLDIR) {
            $inEnv = $True
        }
    }
    $inEnv
}

function isAdmin
{
    $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function promptYN([string]$msg)
{
    $response = Read-Host -Prompt $msg" [y|N]"

    if ( -Not ($response.ToLower() -eq "y") )
    {
        return $false
    }
    return $true
}

# Check for an existing install
# If found and it matches TARGET, ask to overwrite, bail if N
# If found and NOT matches TARGET, ask to change TARGET and ask to overwrite, continue if N
# Finally checks to make sure TARGET doesn't already have a binary in it.
function checkForExisting
{
    try {
        $existing = (Resolve-Path (split-path -Path (get-command $f -ErrorAction 'silentlycontinue').Source -Parent)).Path
    }
    catch {
        # Wasn't found on path but confirm there isn't an existing install in TARGET
        $target =  Join-Path $t $f
        if (Test-Path $target -PathType Leaf)
        {
            Write-host "Target '$target' contains existing state tool."
            if( -Not (promptYN "Do you want to continue installation?  Existing binary will be overwritten."))
            {
                Write-Warning "Choose new installation location.  Abort Installation"
                exit 0
            }
        }
        return $t
    }
    
    # Exists on PATH not TARGET
    Write-Host "Previous install detected: '$existing'" -ForegroundColor Yellow
    if ( -Not ($existing -eq $t))
    {
        Write-Host "Do you want to use the previous install location instead?  This will overwrite the '$f' file there."
        if (-Not (promptYN "Overwrite?"))
        {
            return $t
        } 
    } elseif( -Not (promptYN "Do you wish to overwrite this install?"))
    # Exists on Path AND is in target
    {
        Write-Warning "Abort Installation"
        exit 0
    }
    return  $existing
}

function install()
{
    $USAGE="install.ps1 [flags]`n`r`n`rFlags:`n`r -b <branch>   Default 'unstable'.  Specify an alternative branch to install from (eg. master)`n`r -n               Don't prompt for anything, just install and override any existing executables`n`r -t               Install target dir`n`r -f               Binary filename to use`n`r -h               Shows usage information (what you're currently reading)`n`rEOF`n`r"
    if ($h) {
        Write-Host $USAGE
        exit 0
    }
    $NOPROMPT = $n
    if ($NOPROMPT)
    {
        $INSTALLDIR = $t
    } else 
    {
        $INSTALLDIR = checkForExisting
    }
    $STATEFILE = $f
    $BRANCH = $b
    # State tool binary base dir
    $STATEURL="https://s3.ca-central-1.amazonaws.com/cli-update/update/state"
    
    # Check for various existing installs, this might bail out of the install process
    # So i figured I should comment here so it's not such an innocuous line
    
    # $ENV:PROCESSOR_ARCHITECTURE == AMD64 | x86
    if ($ENV:PROCESSOR_ARCHITECTURE -eq "AMD64") {
        $statejson="windows-amd64.json"
        $statepkg="windows-amd64.zip"
        $stateexe="windows-amd64.exe"

    } else {
        Write-Warning "x86 processors are not supported at this time."
        Write-Warning "Contact ActiveState Support for assistance."
        Write-Warning "Aborting install"
        exit 1
    }

    $downloader = new-object System.Net.WebClient

    # Get version and checksum
    $jsonurl = "$STATEURL/$BRANCH/$statejson"
    Write-Host "Finding latest version...`n"
    try{
        $branchJson = ConvertFrom-Json -InputObject $downloader.DownloadString($jsonurl)
        $latestVersion = $branchJson.Version
        $versionedJson = ConvertFrom-Json -InputObject $downloader.DownloadString("$STATEURL/$BRANCH/$latestVersion/$statejson")
    } catch [System.Exception] {
        Write-Warning "Could not install state tool."
        Write-Warning "Missing branch json or versioned json file."
        Write-Error $_.Exception.Message
        exit 1
    }
    $latestChecksum = $versionedJson.Sha256v2

    # Download pkg file
    $tmpParentPath = Join-Path $env:TEMP "ActiveState"
    $zipPath = Join-Path $tmpParentPath $statepkg
    # Clean it up to start but leave it behind when done 
    if(Test-Path $tmpParentPath){
        Remove-Item $tmpParentPath -Recurse
    }
    New-Item -Path $tmpParentPath -ItemType Directory | Out-Null # There is output from this command, don't show the user.
    $zipURL = "$STATEURL/$BRANCH/$latestVersion/$statepkg"
    Write-Host "Downloading compressed binary...`n"
    try{
        $downloader.DownloadFile($zipURL, $zipPath)
    } catch [System.Exception] {
        Write-Warning "Could not install state tool."
        Write-Warning "Could not access $zipURL"
        Write-Error $_.Exception.Message
        exit 1
    }

    # Extract binary from pkg and confirm checksum
    Write-Host "Extracting binary...`n"
    Expand-Archive $zipPath $tmpParentPath
    $hash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash
    Write-Host "Confirming checksums...`n"
    if ($hash -ne $latestChecksum){
        Write-Warning "SHA256 sum did not match:"
        Write-Warning "Expected: $latestChecksum"
        Write-Warning "Received: $hash"
        Write-Warning "~Aborting installation.~"
        exit 1
    }

    # Install binary
    Write-Host $("Installing state tool in '"+(Join-Path $INSTALLDIR $STATEFILE)+"'") -ForegroundColor Yellow
    #  If the install dir doesn't exist
    if( -Not (Test-Path $INSTALLDIR)) {
        New-Item -Path $INSTALLDIR -ItemType Directory
    } else {
        Remove-Item (Join-Path $INSTALLDIR $STATEFILE) -Erroraction 'silentlycontinue'
    }
    Move-Item (Join-Path $tmpParentPath $stateexe) (Join-Path $INSTALLDIR $STATEFILE)

    # Path setup
    $newPath = "$env:Path;$INSTALLDIR"
    if( -Not (isInRegistry) ){
        if ( -Not (isAdmin)) {
            Write-Host "We tried to add the install directory to your Registry PATH but this session does not have Administrator privileges.  Please run this script in a terminal with Administrator permissions to permanently add the state tool to your path." -ForegroundColor Yellow
        } elseif ( -Not $NOPROMPT -And (promptYN $("We want to add '"+(Join-Path $INSTALLDIR $STATEFILE)+"' to your registry.  This means '$STATEFILE' will be on your PATH in shells you open from now on.  May me?"))) {
            Write-Host "Adding $INSTALLDIR to registry"
            # This only sets it in the regsitry and it will NOT be accessible in the current session
            Set-ItemProperty -Path "Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment" -Name PATH -Value $newPath
        } else {
            Write-Host "To ensure '$STATEFILE' can be used in shells opened later, please add '$INSTALLDIR' to your system path in your system settings." -ForegroundColor Yellow
        }
    } else {
        Write-Host "'$INSTALLDIR' is already in registry" -ForegroundColor Yellow
    }
    if( -Not (isOnPath)){
        if(-Not $NOPROMPT -And (promptYN $("We want to add '"+(Join-Path $INSTALLDIR $STATEFILE)+"' to your session PATH.  May me?")))
        {
            Write-Host "Adding $INSTALLDIR to terminal PATH"
            # This only sets it in the current session
            $Env:Path = $newPath
        } else 
        {
            Write-Host "Update your session PATH to include '$INSTALLDIR' to start using the state tool." -ForegroundColor Yellow
        }
    } else {
        Write-Host "'$INSTALLDIR' is already on your PATH" -ForegroundColor Yellow
    }
}

install
