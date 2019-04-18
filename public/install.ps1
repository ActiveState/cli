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
    [Parameter(
        Mandatory=$False,
        HelpMessage="Branch build to install.")
    ][string]$b = "unstable"
    ,[Parameter(
        Mandatory=$False,
        HelpMessage="No prompt")
    ][boolean]$n = $false
    ,[Parameter(
        Mandatory=$False,
        HelpMessage="Install target dir")
    ][string]$t = (Join-Path $Env:ALLUSERSPROFILE "ActiveState") # C:\ProgramData\ActiveState
    ,[Parameter(
        Mandatory=$False,
        HelpMessage="Binary name def")
    ][string]$f = "state"
    ,[switch]$h
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


$USAGE="install.ps1 [flags]`n`r`n`rFlags:`n`r -b <branch>      Specify an alternative branch to install from (eg. master)`n`r -n               Don't prompt for anything, just install and override any existing executables`n`r -t               Target directory`n`r -f               Filename to use`n`r -h               Shows usage information (what you're currently reading)`n`rEOF`n`r"
$STATE="state.exe"
$INSTALLDIR=$t
# State tool binary base dir
$STATEURL="https://s3.ca-central-1.amazonaws.com/cli-update/update/state"

if ($h.IsPresent) {
    Write-Output $USAGE
}

if ($f.IsPresent) {
    # This should attempt to clean up the filename eg.
    #  appending .exe if it's missing, fail if it's some other ext.
    $STATE = $f
}

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
$jsonurl = "$STATEURL/$b/$statejson"
try{
    $branchJson = ConvertFrom-Json -InputObject $downloader.DownloadString($jsonurl)
    $latestVersion = $branchJson.Version
    $versionedJson = ConvertFrom-Json -InputObject $downloader.DownloadString("$STATEURL/$b/$latestVersion/$statejson")
} catch [System.Exception] {
    Write-Warning "Could not install state tool."
    Write-Warning "Missing branch json or versioned json file."
    Write-Error $_.Exception.Message
    exit 1
}
$latestChecksum = $versionedJson.Sha256

# Download pkg file
$tmpParentPath = Join-Path $env:TEMP "ActiveState"
$zipPath = Join-Path $tmpParentPath $statepkg
# Clean it up to start but leave it behind when done 
if(Test-Path $tmpParentPath){
    Remove-Item $tmpParentPath -Recurse
}
New-Item -Path $tmpParentPath -ItemType Directory | Out-Null # There is output from this command, don't show the user.
$zipURL = "$STATEURL/$b/$latestVersion/$statepkg"
try{
    $downloader.DownloadFile($zipURL, $zipPath)
} catch [System.Exception] {
    Write-Warning "Could not install state tool."
    Write-Warning "Could not access $zipURL"
    Write-Error $_.Exception.Message
    exit 1
}

# Extract binary from pkg and confirm checksum
Write-Host "Extracting binary..."
Expand-Archive $zipPath $tmpParentPath
$hash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash
if ($hash -ne $latestChecksum){
    Write-Warning "SHA256 sum did not match:"
    Write-Warning "Expected: $latestChecksum"
    Write-Warning "Received: $hash"
    Write-Warning "~Aborting installation.~"
    exit 1
}

### Check for existing Binary
### Prompt user where to install, with default location

# Install binary
#  If the install dir doesn't exist
$installPathExe = Join-Path $INSTALLDIR $STATE
Write-Host "Install state tool in $installPathExe"
if( -Not (Test-Path $INSTALLDIR)) {
    New-Item -Path $INSTALLDIR -ItemType Directory
} else {
    Remove-Item $installPathExe
}
Move-Item (Join-Path $tmpParentPath $stateexe) $installPathExe
### Prompt user to Add to path

# Add to path
$newPath = "$env:Path;$INSTALLDIR"
if( -Not (isInRegistry) ){
    
    if ( -Not (isAdmin)) {
        Write-Warning "We tried to add the install directory to your Registry PATH but this session does not have Administrator privileges.  Please run this script in a terminal with Administrator permissions to permanently add the state tool to your path."
    } else {
        Write-Host "Adding $INSTALLDIR to registry"
        # This only sets it in the regsitry and it will NOT be accessible in the current session
        Set-ItemProperty -Path "Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment" -Name PATH -Value $newPath
    }
} else {
    Write-Host "Install dir is already in registry"
}
if( -Not (isOnPath) ){
    Write-Host "Adding $INSTALLDIR to terminal PATH"
    # This only sets it in the current session
    $Env:Path = $newPath
} else {
    Write-Host "Install dir is already on your PATH"
}
