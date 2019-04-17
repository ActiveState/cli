#!/bin/sh
# Copyright 2018 ActiveState Software Inc. All rights reserved.
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
    ][string]$t = (Join-Path -Path $Env:ALLUSERSPROFILE -ChildPath "ActiveState")
    ,[Parameter(
        Mandatory=$False,
        HelpMessage="Binary name def")
    ][string]$f = "state"
    ,[switch]$h
)
$USAGE="install.ps1 [flags]`n`r`n`rFlags:`n`r -b <branch>      Specify an alternative branch to install from (eg. master)`n`r -n               Don't prompt for anything, just install and override any existing executables`n`r -t               Target directory`n`r -f               Filename to use`n`r -h               Shows usage information (what you're currently reading)`n`rEOF`n`r"
$installDir=$t
# State tool binary base dir
$STATEURL="https://s3.ca-central-1.amazonaws.com/cli-update/update/state"

if ($h.IsPresent) {
    Write-Output $USAGE
}

# $ENV:PROCESSOR_ARCHITECTURE == AMD64 | x86
if ($ENV:PROCESSOR_ARCHITECTURE -eq "AMD64") {
    $statejson="windows-amd64.json"
    $statepkg="windows-amd64.zip"
    $stateexe="windows-amd64.exe"

} else {
    $statejson="windows-386.json"
    $statepkg="windows-386.zip"
    $stateexe="windows-386.exe"
}

$downloader = new-object System.Net.WebClient

# Get version and checksum

# $jsonurl = "$STATEURL/$b/$statejson"
$jsonurl = "$STATEURL/$b/linux-amd64.json"
Write-Host $jsonurl
$result = ConvertFrom-Json -InputObject $downloader.DownloadString($jsonurl)
$latestVersion = $result.Version
$latestChecksum = $result.Sha256

# Download pkg file
$tmpParentPath = Join-Path -Path $env:TEMP -ChildPath (Join-Path -Path "ActiveState" -ChildPath $latestVersion)
$tmpFilePath = Join-Path -Path $tmpParentPath -ChildPath $statepkg
Remove-Item -Path $tmpParentPath -recurse
New-Item -Path $tmpParentPath -ItemType Directory
# $pkgUrl = "$STATEURL/$b/$latestVersion/$statepkg"
$pkgUrl = "$STATEURL/$b/$latestVersion/linux-amd64.gz"
$downloader.DownloadFile($pkgUrl, $tmpFilePath)

# Extract binary from pkg and confirm checksum
[System.IO.Compression.ZipFile]::ExtractToDirectory($tmpFilePath, $tmpParentPath)
$tmpExepath = Join-Path -Path $tmpParentPath -ChildPath $stateexe
$hash = Get-FileHash -Path $tmpExepath -Algorithm SHA256
if ($hash -ne $latestChecksum){
    Write-Host "SHA256 sum did not match:"
    Write-Host "Expected: $latestChecksum"
    Write-Host "Received: hash"
    Write-Host "Aborting installation."
    exit 1
}

### Check for existing Binary
### Prompt user where to install, with default location

# Install binary
Move-Item -Path tmpExepath -Destination $installDir
### Prompt user to Add to path

# Add to path
$newPath = $env:Path+=$installDir
Set-ItemProperty -Path ‘Registry::HKEY_LOCAL_MACHINESystemCurrentControlSetControlSession ManagerEnvironment’ -Name PATH –Value $newPath