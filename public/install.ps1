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
$script:STATEEXE = $f
$script:BRANCH = $b

# Helpers
function isInRegistry($path){
    $regpaths = (Get-ItemProperty -Path 'Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment' -Name PATH).Path.Split(';')
    $inReg = $False
    for ($i = 0; $i -lt $regpaths.Count; $i++) {
        if ($regpaths[$i] -eq $path) {
            $inReg = $True
        }
    }
    $inReg
}
function isOnPath($path){
    $envpaths = $env:Path.Split(';')
    $inEnv = $False
    for ($i = 0; $i -lt $envpaths.Count; $i++) {
        if ($envpaths[$i] -eq $path) {
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
    $response = Read-Host -Prompt $msg" [y/N]"

    if ( -Not ($response.ToLower() -eq "y") )
    {
        return $False
    }
    return $True
}

function promptYNQ([string]$msg)
{
    $response = Read-Host -Prompt $msg" [y/N/q]"

    if ($response.ToLower() -eq "q")
    {
        Write-Host "Aborting Installation" -ForegroundColor Yellow
        exit(0)
    }
    if ( -Not ($response.ToLower() -eq "y") )
    {
        return $False
    }
    return $True
}

function errorOccured($suppress) {
    $errMsg = $Error[0]
    $Error.Clear()
    if($errMsg) {
        if (-Not $suppress){
            Write-Warning $errMsg
        }
        return $True
    }
    return $False
}

function hasWritePermission([string] $path)
{
    # $user = "$env:userdomain\$env:username"
    # $acl = Get-Acl $path -ErrorAction 'silentlycontinue'
    # return (($acl.Access | Select-Object -ExpandProperty IdentityReference) -contains $user)
    New-Item -Path (Join-Path $path "perms") -ItemType File -ErrorAction 'silentlycontinue'
    if(errorOccured $True){
        return $False
    }
    Remove-Item -Path (Join-Path $path "perms") -Force  -ErrorAction 'silentlycontinue'
    if(errorOccured $True){
        return $False
    }
    return $True
}

function checkPermsRecur([string] $path){
    # recurse up to the drive root if we have to
    while ($path -ne "") {
        if (Test-Path $path){
            if (-Not (hasWritePermission $path)){
                Write-Warning "You do not have permission to write to '$path'.  Are you running as admin?"
                return $False
            } else {
                return $True
            }
        }
        $path = split-path $path
    }
    Write-Warning "'$orig' is not a valid path"
    return $False
}

function isValidFolder([string] $path)
{   
    if(Test-Path $path){
        #it's a folder
        if (-Not (Test-Path $path -PathType 'Container')){
            Write-Warning "'$path' exists and is not a directory"
            return $False
        }
    }
    return checkPermsRecur $path
}

function getExistingOnPath(){
    (Resolve-Path (split-path -Path (get-command $script:STATEEXE -ErrorAction 'silentlycontinue').Source -Parent)).Path
}

function getDefaultInstallDir() {
    if ($script:TARGET) {
         $script:TARGET
    } elseif (get-command $script:STATEEXE -ErrorAction 'silentlycontinue') {
        $existing = getExistingOnPath
        Write-Host $("Previous install detected at '"+($existing)+"'") -ForegroundColor Yellow
        $existing
    } else {
        (Join-Path $Env:APPDATA (Join-Path "ActiveState" "bin"))
    }
}

function getInstallDir()
{   
    $installDir = ""
    $defaultDir = getDefaultInstallDir
    $validPath = $False
    while( -Not $validPath){
        $installDir = Read-Host "Please enter the installation directory [$defaultDir]"
        if ($installDir -eq ""){
            $installDir = $defaultDir
        }
        if( -Not (isValidFolder $installDir) ) {
            continue
        }
        $targetFile = Join-Path $installDir $script:STATEEXE
        if (Test-Path $targetFile -PathType Leaf) {
            Write-host "Previous installation detected at '$targetFile'"
            if( -Not (promptYNQ "Do you want to continue installation with this directory?"))
            {
                Write-Warning "Choose new installation location"
                continue
            } else  {
                Write-Warning "Overwriting previous installation"
            }
        } 
        $validPath = $True
    }
    $installDir
}

function install()
{
    $USAGE="install.ps1 [flags]
    
    Flags:
    -b <branch>   Default 'unstable'.  Specify an alternative branch to install from (eg. master)
    -n               Don't prompt for anything, just install and override any existing executables
    -t               Install target dir
    -f               Binary filename to use
    -h               Shows usage information (what you're currently reading)"
    if ($h) {
        Write-Host $USAGE
        exit 0
    }
    
    # State tool binary base dir
    $STATEURL="https://s3.ca-central-1.amazonaws.com/cli-update/update/state"
    
    Write-Host "Preparing for installation...`n"
    
    # $ENV:PROCESSOR_ARCHITECTURE == AMD64 | x86
    if ($ENV:PROCESSOR_ARCHITECTURE -eq "AMD64") {
        $statejson="windows-amd64.json"
        $statepkg="windows-amd64.zip"
        $stateexe="windows-amd64.exe"

    } else {
        Write-Warning "x86 processors are not supported at this time"
        Write-Warning "Contact ActiveState Support for assistance"
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
    $zipURL = "$STATEURL/$script:BRANCH/$latestVersion/$statepkg"
    Write-Host "Fetching the latest version: $latestVersion...`n"
    try{
        $downloader.DownloadFile($zipURL, $zipPath)
    } catch [System.Exception] {
        Write-Warning "Could not install state tool"
        Write-Warning "Could not access $zipURL"
        Write-Error $_.Exception.Message
        exit 1
    }

    # Check the sums
    Write-Host "Verifying checksums...`n"
    $hash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash
    if ($hash -ne $latestChecksum){
        Write-Warning "SHA256 sum did not match:"
        Write-Warning "Expected: $latestChecksum"
        Write-Warning "Received: $hash"
        Write-Warning "Aborting installation"
        exit 1
    }

    # Extract binary from pkg and confirm checksum
    Write-Host "Extracting $statepkg...`n"
    Expand-Archive $zipPath $tmpParentPath

    # Confirm the user wants to use the default install location by prompting for new dir
    if ( -Not $script:NOPROMPT) {
        $installDir = getInstallDir
    } else {
        $installDir = getDefaultInstallDir
    }
    # Install binary
    Write-Host "Installing to '$installDir'..." -ForegroundColor Yellow
    #  If the install dir doesn't exist
    if( -Not (Test-Path $installDir)) {
        Write-host "NOTE: $installDir will be created"
        New-Item -Path $installDir -ItemType Directory
    } else {
        Remove-Item (Join-Path $installDir $script:STATEEXE) -Erroraction 'silentlycontinue'
    }
    Move-Item (Join-Path $tmpParentPath $stateexe) (Join-Path $installDir $script:STATEEXE)

    # Path setup
    $newPath = "$installDir;$env:Path"
    if( -Not (isInRegistry $installDir) ){
        if ( -Not (isAdmin)) {
            Write-Host "Please run this installer in a terminal with admin privileges or manually add '$installDir' to your PATH system preferences`n" -ForegroundColor Yellow
        } elseif ( -Not $script:NOPROMPT -And (promptYN $("Allow '"+(Join-Path $installDir $script:STATEEXE)+"' to be appended to your PATH?"))) {
            Write-Host "Updating environment..."
            Write-Host "Adding $installDir to system and current session PATH"
            # This only sets it in the registry and it will NOT be accessible in the current session
            Set-ItemProperty -Path "Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment" -Name PATH -Value $newPath
            $Env:Path = $newPath
            Write-Host "You may now start using the '$script:STATEEXE' program"
        }
    }
    if( -Not (isOnPath $installDir)) {
        # This only sets it in the current session
        # $Env:Path = $newPath
        Write-Host "'$installDir' appended to PATH for current session`n" -ForegroundColor Yellow
    }
}

install
Write-Host "Installation complete"
Write-Host "You may now start using the '$script:STATEEXE' program"
