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

# stateToolPathIsSet returns true if the state tool's installation directory is in the current PATH
function isStateToolInstallationOnPath($installDirectory) {
    $existing = getExistingOnPath
    Write-Host "existing $existing"
    $existing -eq $installDirectory
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

function promptInstallDir()
{   
    $installDir = ""
    $defaultDir = getDefaultInstallDir
    while($True){
        $installDir = (Read-Host "Please enter the installation directory [$defaultDir]").Trim()
        if ($installDir -eq ""){
            $installDir = $defaultDir
        }
        if( -Not (isValidFolder $installDir)) {
            continue
        }
        $targetFile = Join-Path $installDir $script:STATEEXE
        if (Test-Path $targetFile -PathType Leaf) {
            Write-host "Previous installation detected at '$targetFile'"
            if( -Not (promptYNQ "Do you want to continue installation with this directory?"))
            {
                continue
            } else  {
                Write-Warning "Overwriting previous installation"
            }
        }
        break
    }
    $installDir
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

    if ($h) {
        Write-Host $USAGE
        exit 0
    }

    if ($script:NOPROMPT && $script:ACTIVATE != "" ) {
        Write-Error "Flags -n and -activate cannot be set at the same time."
        Write-Host $USAGE
        exit(1)
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
        $installDir = promptInstallDir
    } else {
        $installDir = getDefaultInstallDir
    }
    # Install binary
    Write-Host "`nInstalling to '$installDir'...`n" -ForegroundColor Yellow
    #  If the install dir doesn't exist
    $installPath = Join-Path $installDir $script:STATEEXE
    if( -Not (Test-Path $installDir)) {
        Write-host "NOTE: $installDir will be created`n"
        New-Item -Path $installDir -ItemType Directory | Out-Null
    } else {
        if(Test-Path $installPath -PathType Leaf) {
            Remove-Item $installPath -Erroraction 'silentlycontinue'
            $occurance = errorOccured $False
            if($occurance[0]){
                Write-Host "Aborting Installation" -ForegroundColor Yellow
                exit(1)
            }
        }
    }
    Move-Item (Join-Path $tmpParentPath $stateexe) $installPath

    # Path setup
    $newPath = "$installDir;$env:Path"
    # Check if installation is in $PATH, if not, update SYSTEM or USER settings
    if (isStateToolInstallationOnPath $installDir) {
        Write-Host "`nInstallation complete." -ForegroundColor Yellow
        Write-Host "You may now start using the '$script:STATEEXE' program."
        if ( $script:ACTIVATE -ne "" ) {
            # This creates an interactive sub-shell.
            Write-Host "`nActivating project $script:ACTIVATE`n" -ForegroundColor Yellow
            &$script:STATEEXE activate $script:ACTIVATE
        }
        exit(0)
    }

    # Beyond this point, the state tool is not in the PATH and therefor unsafe to execute.

    # If we have administrative rights, attempt to set PATH system wide...
    if( -Not (isAdmin)){
        if ( -Not $script:NOPROMPT -And (promptYN $("Allow '"+$installPath+"' to be appended to your PATH?"))) {
            Write-Host "Updating environment...`n"
            Write-Host "Adding $installDir to system PATH`n"
            # This only sets it in the registry and it will NOT be accessible in the current session
            Set-ItemProperty -Path "Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment" -Name PATH -Value $newPath
            notifySettingChange
            $msg="To start using the State tool please open a new command prompt with no admin rights.  Please close the current command shell unless you need to perform further task as an administrator.  It is not recommended to run commands as an administrator that do not require it.`n"
            Write-Host $msg
        } else {
            Write-Host "Manually add '$installDir' to your PATH system preferences`n" -ForegroundColor Yellow
        }

    # ... without administrative rights,  we tell the user to update the PATH variable manually
    } else {
        Write-Warning "It's recommended that you close this command prompt and start a new one without admin privileges.`n"
    }
    Write-Host "To start using the State tool right away update your current PATH by running 'set PATH=%PATH%;$installDir'`n" -ForegroundColor Yellow

    # Print a warning that we cannot automatically activate a requested project.
    if ( "$script:ACTIVATE" -eq "" ) {
        Write-Host "`nCannot activate project $script:ACTIVATE yet." -ForegroundColor Yellow
        Write-Host "In order to activate a project, the state tool needs to be installed in your PATH first."
        Write-Host "To manually activate the project run 'state activate $script:ACTIVATE'"
    }
}

install
Write-Host "Installation complete"
