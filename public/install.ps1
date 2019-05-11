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

$global:NOPROMPT = $n
$global:TARGET = $t
# C:\Users\cgcho\AppData\Roaming\ActiveState\bin
$global:DEFAULTDIR = (Join-Path $Env:APPDATA (Join-Path "ActiveState" "bin")) 
$global:STATEEXE = $f
$global:BRANCH = $b

# Helpers
function isInRegistry(){
    $regpaths = (Get-ItemProperty -Path 'Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment' -Name PATH).Path.Split(';')
    $inReg = $False
    for ($i = 0; $i -lt $regpaths.Count; $i++) {
        if ($regpaths[$i] -eq $global:INSTALLDIR) {
            $inReg = $True
        }
    }
    $inReg
}
function isOnPath(){
    $envpaths = $env:Path.Split(';')
    $inEnv = $False
    for ($i = 0; $i -lt $envpaths.Count; $i++) {
        if ($envpaths[$i] -eq $global:INSTALLDIR) {
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
function checkForExisting()
{   
    try {
        $global:PREVIOUSINSTALL = (Resolve-Path (split-path -Path (get-command $global:STATEEXE -ErrorAction 'silentlycontinue').Source -Parent)).Path
    }
    catch {
        # Wasn't found on path but confirm there isn't an existing install in TARGET
        $targetFile =  Join-Path $global:TARGET $global:STATEEXE
        if (Test-Path $targetFile -PathType Leaf)
        {
            Write-host "Target '$targetFile' contains existing state tool."
            if( -Not (promptYN "Do you want to continue installation?"))
            {
                Write-Warning "Choose new installation location.  Aborting Installation"
                exit 0
            } else 
            {
                Write-Warn "Overwriting previous installation"
                $global:INSTALLDIR = $global:TARGET
                return
            }
        }
    }
    # Exists on PATH not TARGET
    Write-Host "Previous install detected at '$global:PREVIOUSINSTALL'" -ForegroundColor Yellow
    if  ($global:TARGET -eq "")
    {
        $global:INSTALLDIR = $global:PREVIOUSINSTALL
    }
    elseif(-Not ($global:PREVIOUSINSTALL -eq $global:TARGET))
    {
        Write-Host "Do you want to use the previous install location instead?  This will overwrite the '$global:STATEEXE' file there."
        if (promptYN "Overwrite?")
        {
            $global:INSTALLDIR = $global:PREVIOUSINSTALL
        } else 
        {
            Write-Warning "Installing elsewhere from previous installation"
            $global:INSTALLDIR = $global:TARGET
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
        Write-Warn $("Overwriting previous installation")
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
    if(Test-Path $path)
    {
        if (-Not (Test-Path $path -PathType 'Container')) 
        {
            Write-Warning "'$path' exist but isn't a folder"
            return $false
        } elseif ( -Not (hasWritePermission $path))
        {
            Write-Warning "You do not have permission to write to '$path'"
            return $false
        }
    #check parent if doesn't exist
    } elseif ( -Not (hasWritePermission (split-path $path)))
    {
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
        if (($dir = Read-Host "Please enter the installation directory [$global:INSTALLDIR]") -eq "")
        {   
            $validPath = $true
        } else 
        {
            if (isValidFolder $dir)
            {
                $global:INSTALLDIR = $dir
                $validPath = $true
            } 
        }
    }
}

function setInstallDir()
{   
    checkForExisting
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
    $jsonurl = "$STATEURL/$global:BRANCH/$statejson"
    Write-Host "Determining latest version...`n"
    try{
        $branchJson = ConvertFrom-Json -InputObject $downloader.DownloadString($jsonurl)
        $latestVersion = $branchJson.Version
        $versionedJson = ConvertFrom-Json -InputObject $downloader.DownloadString("$STATEURL/$global:BRANCH/$latestVersion/$statejson")
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
    $zipURL = "$STATEURL/$global:BRANCH/$latestVersion/$statepkg"
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
    if ( -Not $global:NOPROMPT -And ($global:TARGET -eq ""))
    {
        setInstallDir
    } else {
        if( -Not ($global:TARGET -eq ""))
        {
            $global:INSTALLDIR = $global:TARGET
        }
        else 
        {
            $global:INSTALLDIR = $global:DEFAULTDIR
        }
    }
    # Install binary
    Write-Host "Installing to '$global:INSTALLDIR'..." -ForegroundColor Yellow
    #  If the install dir doesn't exist
    if( -Not (Test-Path $global:INSTALLDIR)) {
        Write-host "NOTE: $global:INSTALLDIR will be created"
        New-Item -Path $global:INSTALLDIR -ItemType Directory
    } else {
        Remove-Item (Join-Path $global:INSTALLDIR $global:STATEEXE) -Erroraction 'silentlycontinue'
    }
    Move-Item (Join-Path $tmpParentPath $stateexe) (Join-Path $global:INSTALLDIR $global:STATEEXE)

    # Path setup
    $newPath = "$global:INSTALLDIR;$env:Path"
    $manualPathTxt = "manually add '$global:INSTALLDIR' to your `$PATH"
    $andSystem = "in your system preferences to add $global:STATEEXE to your `$PATH permanently"
    if( -Not (isInRegistry) ){
        if ( -Not (isAdmin)) {
            Write-Host "Please run this installer in a terminal with admin privileges or $manualPathTxt $andSystem" -ForegroundColor Yellow
        } elseif ( -Not $global:NOPROMPT -And (promptYN $("Allow '"+(Join-Path $global:INSTALLDIR $global:STATEEXE)+"' to be appended to the registry `$PATH?"))) {
            Write-Host "Adding $global:INSTALLDIR to registry"
            # This only sets it in the regsitry and it will NOT be accessible in the current session
            Set-ItemProperty -Path "Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment" -Name PATH -Value $newPath
        } else {
            Write-Host "Please $manualPathTxt $andSystem" -ForegroundColor Yellow
        }
    } else {
        Write-Host "'$global:INSTALLDIR' is already in registry" -ForegroundColor Yellow
    }
    if( -Not (isOnPath)){
        if(-Not $global:NOPROMPT -And (promptYN $("Allow '"+(Join-Path $global:INSTALLDIR $global:STATEEXE)+"' to be append to your session `$PATH?")))
        {
            Write-Host "Updating environment..."
            # This only sets it in the current session
            $Env:Path = $newPath
            Write-Host "You may now start using the '$global:STATEEXE' program."
        } else 
        {   
            Write-Host "Please $manualPathTxt to start using start using the $global:STATEEXE" -ForegroundColor Yellow
        }
    } else {
        Write-Host "'$global:INSTALLDIR' is already on your PATH" -ForegroundColor Yellow
        
    }

}

install
Write-Host "Installation complete."