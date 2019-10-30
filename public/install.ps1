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
    ,[Parameter(Mandatory=$False)][string]$activate = ""
)

$script:NOPROMPT = $n
$script:TARGET = ($t).Trim()
$script:STATEEXE = ($f).Trim()
$script:STATE = $f.Substring(0, $f.IndexOf("."))
$script:BRANCH = ($b).Trim()
$script:ACTIVATE =($activate).Trim()

# Some cmd-lets throw exceptions that don't stop the script.  Force them to stop.
$ErrorActionPreference = "Stop"

# Helpers
function notifySettingChange(){
    $HWND_BROADCAST = [IntPtr] 0xffff;
    $WM_SETTINGCHANGE = 0x1a;
    $result = [UIntPtr]::Zero

    if (-not ("Win32.NativeMethods" -as [Type]))
    {
        # import sendmessagetimeout from win32
        Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition @"
        [DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
        public static extern IntPtr SendMessageTimeout(
        IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
        uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
"@
    }
    # notify all windows of environment block change
    [Win32.Nativemethods]::SendMessageTimeout($HWND_BROADCAST, $WM_SETTINGCHANGE, [UIntPtr]::Zero, "Environment", 2, 5000, [ref] $result);

}

function isAdmin
{
    $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function promptYN([string]$msg)
{
    $response = Read-Host -Prompt $msg" [y/N]`n"

    if ( -Not ($response.ToLower() -eq "y") )
    {
        return $False
    }
    return $True
}

function promptYNQ([string]$msg)
{
    $response = Read-Host -Prompt $msg" [y/N/q]`n"

    if ($response.ToLower() -eq "q")
    {
        Write-Host "Aborting Installation" -ForegroundColor Yellow
        exit 0
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
        return $True, $errMsg
    }
    return $False, ""
}

function hasWritePermission([string] $path)
{
    # $user = "$env:userdomain\$env:username"
    # $acl = Get-Acl $path -ErrorAction 'silentlycontinue'
    # return (($acl.Access | Select-Object -ExpandProperty IdentityReference) -contains $user)
    $thefile = "activestate-perms"
    New-Item -Path (Join-Path $path $thefile) -ItemType File -ErrorAction 'silentlycontinue'
    $occurance = errorOccured $True
    #  If an error occurred and it's NOT and IOExpction error where the file already exists
    if( $occurance[0] -And -Not ($occurance[1].exception.GetType().fullname -eq "System.IO.IOException" -And (Test-Path $path))){
        return $False
    }
    Remove-Item -Path (Join-Path $path $thefile) -Force  -ErrorAction 'silentlycontinue'
    if((errorOccured $True)[0]){
        return $False
    }
    return $True
}

function checkPermsRecur([string] $path)
{
    $orig = $path
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
    if($path[1] -ne ":"){
        Write-Warning "Must provide an absolute path."
        return $False
    }
    if(Test-Path $path){
        #it's a folder
        if (-Not (Test-Path $path -PathType 'Container')){
            Write-Warning "'$path' exists and is not a directory"
            return $False
        }
    }
    return checkPermsRecur $path
}

# isStateToolInstallationOnPath returns true if the state tool's installation directory is in the current PATH
function isStateToolInstallationOnPath($installDirectory) {
    $existing = getExistingOnPath
    $existing -eq $installDirectory
}

function getExistingOnPath(){
    $path = (get-command $script:STATEEXE -ErrorAction 'silentlycontinue').Source
    if ($path -eq $null) {
        ""
    } else {
       (Resolve-Path (split-path -Path $path -Parent)).Path
    }
}

function activateIfRequested() {
    if ( $script:ACTIVATE -ne "" ) {
        # This creates an interactive sub-shell.
        Write-Host "`nActivating project $script:ACTIVATE`n" -ForegroundColor Yellow
        &$script:STATEEXE activate $script:ACTIVATE
    }
}

function warningIfadmin() {
    if (IsAdmin) {
        Write-Warning "It's recommended that you close this command prompt and start a new one without admin privileges.`n"
    }
}

function install()
{
    $USAGE="install.ps1 [flags]
    
    Flags:
    -b <branch>          Default 'unstable'.  Specify an alternative branch to install from (eg. master)
    -n                   Don't prompt for anything, just install and override any existing executables
    -t <dir>             Install target dir
    -f <file>            Default 'state.exe'.  Binary filename to use
    -activate <project>  Activate a project when state tools is correctly installed
    -h                   Show usage information (what you're currently reading)"

    # Ensure errors from previously run commands are reported during install
    $Error.Clear()

    if ($h) {
        Write-Host $USAGE
        exit 0
    }

    if ($script:NOPROMPT -and $script:ACTIVATE -ne "" ) {
        Write-Error "Flags -n and -activate cannot be set at the same time."
        exit 1
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
        Write-Warning "Aborting installation"
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

    # Get the install directory and ensure we have permissions on it.
    # If the user provided an install dir we do no verification.
    if ($script:TARGET) {
        $installDir = $script:TARGET
   } else {
        $installDir = (Join-Path $Env:APPDATA (Join-Path "ActiveState" "bin"))
        if (-Not (hasWritePermission $Env:APPDATA)){
            Write-Error "Do not have write permissions to: '$Env:APPDATA'"
            Write-Error "Aborting installation"
            exit 1
        }
   }

    # Install binary
    Write-Host "`nInstalling to '$installDir'...`n" -ForegroundColor Yellow
    if ( -Not $script:NOPROMPT ) {
        if( -Not (promptYNQ "Continue?") ) {
            exit 0
        }
    }

    # Check if previous installation exists and let user know if it does
   $installPath = Join-Path $installDir $script:STATEEXE
   if (Test-Path $installPath -PathType Leaf) {
        Write-Host $("Previous install detected at '"+($installPath)+"'") -ForegroundColor Yellow
        if( -Not (promptYNQ "Do you want to continue installation with this directory?")) {
            Write-Host "Aborting installation"
            exit 0
        } else {
            Write-Warning "Overwriting previous installation"
        }
    }

    #  If the install dir doesn't exist
    if( -Not (Test-Path $installDir)) {
        Write-host "NOTE: $installDir will be created`n"
        New-Item -Path $installDir -ItemType Directory | Out-Null
    } else {
        if(Test-Path $installPath -PathType Leaf) {
            # TODO: There is a bug here that if you run the `.\public\install.ps1 -t C:\temp\state\bin` twice it will error the second time
            Remove-Item $installPath -Erroraction 'silentlycontinue'
            $occurance = errorOccured $False
            if($occurance[0]){
                Write-Host "Aborting Installation" -ForegroundColor Yellow
                exit 1
            }
        }
    }
    Move-Item (Join-Path $tmpParentPath $stateexe) $installPath

    # Check if installation is in $PATH, if not, update SYSTEM or USER settings
    if (isStateToolInstallationOnPath $installDir) {
        Write-Host "`nInstallation complete." -ForegroundColor Yellow
        Write-Host "You may now start using the '$script:STATEEXE' program."
        warningIfAdmin
        activateIfRequested
        exit 0
    }

    # Update PATH for state tool installation directory
    $envTarget = [EnvironmentVariableTarget]::User
    $envTargetName = "user"
    if (isAdmin) {
        $envTarget = [EnvironmentVariableTarget]::Machine
	    $envTargetName = "system"
    }

    Write-Host "Updating environment...`n"
    Write-Host "Adding $installDir to $envTargetName PATH`n"
    # This only sets it in the registry and it will NOT be accessible in the current session
    [Environment]::SetEnvironmentVariable(
        'Path',
        $installDir + ";" + [Environment]::GetEnvironmentVariable(
            'Path', [EnvironmentVariableTarget]::Machine),
        $envTarget)

    notifySettingChange

    $env:Path = $installDir + ";" + $env:Path
    activateIfRequested

    warningIfAdmin
    Write-Host "State tool successfully installed to: $installDir." -ForegroundColor Yellow
    Write-Host "Please restart your command prompt in order to start using the 'state.exe' program." -ForegroundColor Yellow
}

install
