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
        [string]
        $t
    ,[Parameter(Mandatory=$False)][switch]$n
    ,[Parameter(Mandatory=$False)][switch]$h
    ,[Parameter(Mandatory=$False)]
        [ValidateScript({[IO.Path]::GetExtension($_) -eq '.exe'})]
        [string]
        $f = "state.exe"
)

$script:NOPROMPT = $n
$script:TARGET = $t
# C:\Users\cgcho\AppData\Roaming\ActiveState\bin
$script:DEFAULTDIR = (Join-Path $Env:APPDATA (Join-Path "ActiveState" "bin")) 
$script:STATEEXE = $f
$script:BRANCH = $b

# Helpers
function isInRegistry(){
    $regpaths = (Get-ItemProperty -Path 'Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment' -Name PATH).Path.Split(';')
    $inReg = $False
    for ($i = 0; $i -lt $regpaths.Count; $i++) {
        if ($regpaths[$i] -eq $script:INSTALLDIR) {
            $inReg = $True
        }
    }
    $inReg
}
function isOnPath(){
    $envpaths = $env:Path.Split(';')
    $inEnv = $False
    for ($i = 0; $i -lt $envpaths.Count; $i++) {
        if ($envpaths[$i] -eq $script:INSTALLDIR) {
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
function setTargetDir([string] $path)
{   
    if ($path -eq "")
    {
        $path = $script:TARGET
    }

    if ((get-command $script:STATEEXE -ErrorAction 'silentlycontinue'))
    {
        $script:PREVIOUSINSTALL = (Resolve-Path (split-path -Path (get-command $script:STATEEXE -ErrorAction 'silentlycontinue').Source -Parent)).Path
    } else {
        # Wasn't found on path but confirm there isn't an existing install in TARGET
        $targetFile =  Join-Path $path $script:STATEEXE
        if (Test-Path $targetFile -PathType Leaf)
        {
            Write-host "Previous installation detected at $targetFile"
            if( -Not (promptYN "Do you want to continue installation?"))
            {
                Write-Warning "Choose new installation location.  Aborting Installation"
                exit 0
            } else 
            {
                Write-Warn "Overwriting previous installation"
                $script:INSTALLDIR = $path
                return
            }
        }
    }
    # Exists on PATH not TARGET
    Write-Host "Previous install detected at '$script:PREVIOUSINSTALL'" -ForegroundColor Yellow
    if  ($path -eq "")
    {
        $script:INSTALLDIR = $script:PREVIOUSINSTALL
    }
    elseif(-Not ($script:PREVIOUSINSTALL -eq $path))
    {
        Write-Host "Do you want to use the previous install location instead?  This will overwrite the '$script:STATEEXE' file there."
        if (promptYN "Overwrite?")
        {
            $script:INSTALLDIR = $script:PREVIOUSINSTALL
        } else 
        {
            Write-Warning "Installing elsewhere from previous installation"
            $script:INSTALLDIR = $path
        }
    } 
    elseif( -Not (promptYN "Do you wish to overwrite this install?"))
    # Exists on Path AND is in target
    {
        Write-Warning "Abort Installation"
        exit 0
    } 
    else 
    {
        Write-Warning $("Overwriting previous installation")
        $script:INSTALLDIR = $path
    }
}

function hasWritePermission([string] $path)
{
    $user = "$env:userdomain\$env:username"
    $acl = Get-Acl $path
    return (($acl.Access | Select-Object -ExpandProperty IdentityReference) -contains $user)
}

function isValidFolder([string] $path)
{   
    if(Test-Path $path){
        #it's a folder
        if (-Not (Test-Path $path -PathType 'Container')){
            Write-Warning "'$path' exist but isn't a folder"
            return $false
        # and it's writable
        } elseif ( -Not (hasWritePermission $path)) {
            Write-Warning "You do not have permission to write to '$path'"
            return $false
        }
    # check parent permissions if path doesn't exist
    } elseif ( -Not (hasWritePermission (split-path $path))) {
        Write-Warning $("You do not have permission to write to '"+(split-path $path)+"'")
        return $false
    }
    return $true
}

function promptInstallDir()
{
    $validPath = $false
    while ( -Not $validPath )
    {   
        if (($dir = Read-Host "Please enter the installation directory [$script:INSTALLDIR]") -eq ""){   
            $validPath = $true
        } else  {
            if (isValidFolder $dir) {
                setTargetDir $dir
                $validPath = $true
            } 
        }
    }
}

function setInstallDir()
{   
    setTargetDir
    promptInstallDir
}

function install()
{
    $USAGE="install.ps1 [flags]`n`r`n`rFlags:`n`r -b <branch>   Default 'unstable'.  Specify an alternative branch to install from (eg. master)`n`r -n               Don't prompt for anything, just install and override any existing executables`n`r -t               Install target dir`n`r -f               Binary filename to use`n`r -h               Shows usage information (what you're currently reading)`n`rEOF`n`r"
    if ($h) {
        Write-Host $USAGE
        exit 0
    }
    
    # State tool binary base dir
    $STATEURL="https://s3.ca-central-1.amazonaws.com/cli-update/update/state"
    
    Write-Host "Preparing for installation...`n"
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
    $jsonurl = "$STATEURL/$script:BRANCH/$statejson"
    Write-Host "Determining latest version...`n"
    try{
        $branchJson = ConvertFrom-Json -InputObject $downloader.DownloadString($jsonurl)
        $latestVersion = $branchJson.Version
        $versionedJson = ConvertFrom-Json -InputObject $downloader.DownloadString("$STATEURL/$script:BRANCH/$latestVersion/$statejson")
    } catch [System.Exception] {
        Write-Warning "Unable to retrieve the latest version number"
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
    $zipURL = "$STATEURL/$script:BRANCH/$latestVersion/$statepkg"
    Write-Host "Fetching the latest version: $latestVersion...`n"
    try{
        $downloader.DownloadFile($zipURL, $zipPath)
    } catch [System.Exception] {
        Write-Warning "Could not install state tool."
        Write-Warning "Could not access $zipURL"
        Write-Error $_.Exception.Message
        exit 1
    }

    #Check the sums
    Write-Host "Verifying checksums...`n"
    $hash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash
    if ($hash -ne $latestChecksum){
        Write-Warning "SHA256 sum did not match:"
        Write-Warning "Expected: $latestChecksum"
        Write-Warning "Received: $hash"
        Write-Warning "Aborting installation."
        exit 1
    }

    # Extract binary from pkg and confirm checksum
    Write-Host "Extracting $statepkg...`n"
    Expand-Archive $zipPath $tmpParentPath

    # Confirm the user wants to use the default install location by prompting for new dir
    if ( -Not $script:NOPROMPT) {
        if($script:TARGET -ne ""){
            setTargetDir $script:TARGET
        } else {
            setInstallDir
        }
    } else {
        $script:INSTALLDIR = $script:DEFAULTDIR
    }
    # Install binary
    Write-Host "Installing to '$script:INSTALLDIR'..." -ForegroundColor Yellow
    #  If the install dir doesn't exist
    if( -Not (Test-Path $script:INSTALLDIR)) {
        Write-host "NOTE: $script:INSTALLDIR will be created"
        New-Item -Path $script:INSTALLDIR -ItemType Directory
    } else {
        Remove-Item (Join-Path $script:INSTALLDIR $script:STATEEXE) -Erroraction 'silentlycontinue'
    }
    Move-Item (Join-Path $tmpParentPath $stateexe) (Join-Path $script:INSTALLDIR $script:STATEEXE)

    # Path setup
    $newPath = "$script:INSTALLDIR;$env:Path"
    $manualPathTxt = "manually add '$script:INSTALLDIR' to your PATH system preferences to add '$script:STATEEXE' to your PATH permanently"
    if( -Not (isInRegistry) ){
        if ( -Not (isAdmin)) {
            Write-Host "Please run this installer in a terminal with admin privileges or $manualPathTxt" -ForegroundColor Yellow
        } elseif ( -Not $script:NOPROMPT -And (promptYN $("Allow '"+(Join-Path $script:INSTALLDIR $script:STATEEXE)+"' to be appended to your PATH?"))) {
            Write-Host "Updating environment..."
            Write-Host "Adding $script:INSTALLDIR to registry"
            # This only sets it in the regsitry and it will NOT be accessible in the current session
            Set-ItemProperty -Path "Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment" -Name PATH -Value $newPath
            if( -Not (isOnPath)) {
                # This only sets it in the current session
                $Env:Path = $newPath
                Write-Host "You may now start using the '$script:STATEEXE' program."
            }
        }
    }

    if( -Not (isInRegistry) -And -Not (isOnPath)){
        Write-Host "Please $manualPathTxt then start a new shell" -ForegroundColor Yellow
    }
}

install
Write-Host "Installation complete."
