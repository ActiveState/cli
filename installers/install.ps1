﻿# Copyright 2019 ActiveState Software Inc. All rights reserved.
<#
.DESCRIPTION
Install the ActiveState state.exe tool.  Must be run as admin OR install State Tool to
User profile folder.

.EXAMPLE
install.ps1 -b branchToInstall -t C:\dir\on\path
#>

param (
    [Parameter(Mandatory=$False)][string]$b = 'release'
    ,[Parameter(Mandatory=$False)]
        [string]
        $t
    ,[Parameter(Mandatory=$False)][switch]$n
    ,[Parameter(Mandatory=$False)][switch]$f
    ,[Parameter(Mandatory=$False)][string]$c
    ,[Parameter(Mandatory=$False)][switch]$h
    ,[Parameter(Mandatory=$False)][string]$v
    ,[Parameter(Mandatory=$False)][string]$activate = ""
    ,[Parameter(Mandatory=$False)][string]${activate-default} = ""
)

Set-StrictMode -Off

# ignore project file if we are already in an activated environment
$Env:ACTIVESTATE_PROJECT=""

$script:NOPROMPT = $n
$script:FORCEOVERWRITE = $f
$script:TARGET = ($t).Trim()
$script:STATEEXE = "state.exe"
$script:BRANCH = ($b).Trim()
$script:VERSION = ($v).Trim()
$script:POST_INSTALL_COMMAND = ($c).Trim()
$script:ACTIVATE = ($activate).Trim()
$script:ACTIVATE_DEFAULT = (${activate-default}).Trim()

$script:SESSION_TOKEN_VERIFY = -join("{","TOKEN","}")
$script:SESSION_TOKEN = "{TOKEN}"
$script:SESSION_TOKEN_VALUE = ""

if ("$SESSION_TOKEN" -ne "$SESSION_TOKEN_VERIFY") {
  $script:SESSION_TOKEN_VALUE = $script:SESSION_TOKEN
}

# For recipe installation without prompts we need to be able to disable
# prompts through an environment variable.
if ($Env:NOPROMPT_INSTALL -eq "true") {
    $script:NOPROMPT = $true
    $script:FORCEOVERWRITE = $true
}

# Some cmd-lets throw exceptions that don't stop the script.  Force them to stop.
$ErrorActionPreference = "Stop"

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

function errorOccured($suppress, $errMsg) {
    if($errMsg) {
        if (-Not $suppress){
            Write-Warning $errMsg
        }
        return $True, $errMsg
    }
    return $False, ""
}

function download([string] $url, [string] $out) {
    [int]$Retrycount = "0"

    do {
        try {
            $downloader = new-object System.Net.WebClient
            if ($out -eq "") {
                return $downloader.DownloadString($url)
            }
            else {
                return $downloader.DownloadFile($url, $out)
            }
        }
        catch {
            if ($Retrycount -gt 5){
                Write-Error "Could not Download after 5 retries."
                throw $_
            }
            else {
                Write-Host "Could not Download, retrying..."
                Write-Host $_
                $Retrycount = $Retrycount + 1
            }
        }
    }
    While ($true)
}

function hasWritePermission([string] $path)
{
    # $user = "$env:userdomain\$env:username"
    # $acl = Get-Acl $path -ErrorAction 'silentlycontinue'
    # return (($acl.Access | Select-Object -ExpandProperty IdentityReference) -contains $user)
    $thefile = "activestate-perms"
    New-Item -Path (Join-Path $path $thefile) -ItemType File -ErrorAction 'SilentlyContinue' -ErrorVariable err
    $occurance = errorOccured $True "$err"
    #  If an error occurred and it's NOT and IOExpction error where the file already exists
    if( $occurance[0] -And -Not ($occurance[1].exception.GetType().fullname -eq "System.IO.IOException" -And (Test-Path -LiteralPath $path))){
        return $False
    }
    Remove-Item -Path (Join-Path $path $thefile) -Force  -ErrorAction 'silentlycontinue' -ErrorVariable err
    if((errorOccured $True "$err")[0]){
        return $False
    }
    return $True
}

# isStateToolInstallationOnPath returns true if the State Tool's installation directory is in the current PATH
function isStateToolInstallationOnPath($installDirectory) {
    $existing = getExistingOnPath
    $existing -eq $installDirectory
}

function getExistingOnPath(){
    $path = (get-command $script:STATEEXE -ErrorAction 'silentlycontinue').Source
    if ($null -eq $path) {
        ""
    } else {
       (Resolve-Path (split-path -Path $path -Parent)).Path
    }
}

function activateIfRequested() {
    if ( $script:ACTIVATE -ne "" ) {
        # This creates an interactive sub-shell.
        Write-Host "`nActivating project $script:ACTIVATE`n" -ForegroundColor Yellow
        & $script:STATEEXE activate $script:ACTIVATE
    } elseif ( $script:ACTIVATE_DEFAULT -ne "" ) {
        # This creates an interactive sub-shell.
        Write-Host "`nActivating project $script:ACTIVATE_DEFAULT as default`n" -ForegroundColor Yellow
        & $script:STATEEXE activate $script:ACTIVATE_DEFAULT --default
    }
}

function warningIfadmin() {
    if ( (IsAdmin) -and -not (Test-Path env:CI) ) {
        Write-Warning "It is recommended to use the State Tool in a new terminal session without admin privileges.`n"
    }
}

function runPreparationStep($installDirectory) {
    &$installDirectory\$script:STATEEXE _prepare | Write-Host
    return $LASTEXITCODE
}

function displayConsent() {
    $consentText="
ActiveState collects usage statistics and diagnostic data about failures. The collected data complies with ActiveState Privacy Policy (https://www.activestate.com/company/privacy-policy/) and will be used to identify product enhancements, help fix defects, and prevent abuse.

By running the State Tool installer you consent to the Privacy Policy. This is required for the State Tool to operate while we are still in beta.

Please note that the installer may have to modify your system environment to add the installation PATH.
"
    Write-Host $consentText
}

function fetchArtifacts($downloadDir, $statejson, $statepkg) {
    # State Tool binary base dir
    $STATEURL= "https://state-tool.s3.amazonaws.com/update/state"
    
    Write-Host "Preparing for installation...`n"

    if ($script:VERSION -ne "") {
        Write-Host "Attempting to fetch version: $script:VERSION...`n"
        $jsonURL = "$STATEURL/$script:BRANCH/$script:VERSION/$statejson"

        try {
            $jsonString = download $jsonURL
        } catch [System.Exception] {
            Write-Error "Could not fetch version: $script:VERSION, please verify the version number and try again."
            return 1
        }

        $infoJson = ConvertFrom-Json -InputObject $jsonString

        $version = $script:VERSION

        Write-Host "Fetching version: $version...`n"
    } else {
        Write-Host "Determining latest version...`n"
        $jsonurl = "$STATEURL/$script:BRANCH/$statejson"
        $infoJson = ConvertFrom-Json -InputObject (download $jsonurl)
        $version = $infoJson.Version

        Write-Host "Fetching the latest version: $version...`n"
    }

    $checksum = $infoJson.Sha256
    $relUrl = $infoJson.Path
    $zipUrl = "$STATEURL/$relurl"

    # Download pkg file
    $zipPath = Join-Path $downloadDir $statepkg
    # Clean it up to start but leave it behind when done 
    if(Test-Path -LiteralPath $downloadDir){
        Remove-Item $downloadDir -Recurse
    }
    New-Item -Path $downloadDir -ItemType Directory | Out-Null # There is output from this command, don't show the user.
    try{
        download $zipURL $zipPath
    } catch [System.Exception] {
        Write-Warning "Could not install State Tool"
        Write-Warning "Could not access $zipURL"
        Write-Error $_.Exception.Message
        return 1
    }

    # Check the sums
    Write-Host "Verifying checksums...`n"
    $hash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash
    if ($hash -ne $checksum){
        Write-Warning "SHA256 sum did not match:"
        Write-Warning "Expected: $checksum"
        Write-Warning "Received: $hash"
        Write-Warning "Aborting installation"
        return 1
    }

    # Extract binary from pkg and confirm checksum
    Write-Host "Extracting $statepkg...`n"
    # using LiteralPath argument prevents interpretation of wildcards in zipPath
    Expand-Archive -LiteralPath $zipPath -DestinationPath $downloadDir
}

function test-64Bit() {
    return (test-is64bitOS -or test-wow64 -or test-64bitWMI -or test-64bitPtr)
}

function test-is64bitOS() {
    return [System.Environment]::Is64BitOperatingSystem
}

function test-wow64() {
    # Only 64 bit Operating Systems should have this directory
    return test-path (join-path $env:WinDir "SysWow64") 
}

function test-64bitWMI() {
    if ((Get-WmiObject Win32_OperatingSystem).OSArchitecture -eq "64-bit") {
        return $True
    } else {
        return $False
    }
}

function test-64bitPtr() {
    # The int pointer size is 8 for 64 bit Operating Systems and
    # 4 for 32 bit
    return [IntPtr]::size -eq 8
}

function install() {
    $USAGE="install.ps1 [flags]
    
    Flags:
    -b <branch>          Default 'release'.  Specify an alternative branch to install from (eg. beta)
    -n                   Don't prompt for anything, just install and override any existing executables
    -t <dir>             Install target dir
    -c <command>         Run any command after the install script has completed
    -activate <project>  Activate a project when State Tools is correctly installed
    -h                   Show usage information (what you're currently reading)"

    # Ensure errors from previously run commands are not reported during install
    $Error.Clear()

    displayConsent

    if ($h) {
        Write-Host $USAGE
        return
    }

    if ($script:NOPROMPT -and $script:ACTIVATE -ne "" ) {
        Write-Warning "Flags -n and -activate cannot be set at the same time."
        return 1
    }

    if ($script:ACTIVATE -and $script:ACTIVATE_DEFAULT -ne "") {
        Write-Warning "Flags -activate and -activate-default cannot be set at the same time."
        return 1
    }

    Write-Host '╔═══════════════════════╗ ' -ForegroundColor DarkGray
    Write-Host '║ ' -ForegroundColor DarkGray -NoNewline
    Write-Host "Installing State Tool" -ForegroundColor White -NoNewline;
    Write-Host " ║" -ForegroundColor DarkGray;
    Write-Host "╚═══════════════════════╝" -ForegroundColor DarkGray;

    # $ENV:PROCESSOR_ARCHITECTURE == AMD64 | x86
    $installerexe = (Join-Path "state-install" "state-installer.exe")
    if (test-64Bit) {
        $statepkg="windows-amd64.zip"
        $statejson="windows-amd64/info.json"

    } else {
        $statepkg="windows-386.zip"
        $statejson="windows-386/info.json"
    }

    $existing = getExistingOnPath

    # Check if previous installation is detected
    if (get-command $script:STATEEXE -ErrorAction 'silentlycontinue') {
        # Check if a different target directory matches existing installation
        if (-not $script:TARGET -or ( $script:TARGET -eq $existing )) {
            if ($script:FORCEOVERWRITE -and ( Test-Path (Join-Path -Path $existing -ChildPath $script:STATEEXE) -PathType leaf )) {
                Write-Warning "Overwriting previous installation."
                $script:NOPROMPT = $true
            }
            else {
                Write-Host $("State Tool is already installed at '" + ($existing) + "' to reinstall run this command again with -f") -ForegroundColor Yellow
                Write-Host "To update the State Tool to the latest version, please run 'state update'."
                Write-Host "To install in a different location, please specify the installation directory with '-t TARGET_DIR'."
                return
            }
        }
    }

    Write-Host "`nInstalling ActiveState State Tool...`n" -ForegroundColor Yellow
    if ( -Not $script:NOPROMPT ) {
        if( -Not (promptYN "Continue?") ) {
            return 2
        }
    }

    $tmpParentPath = Join-Path $env:TEMP "ActiveState"
    $err = fetchArtifacts $tmpParentPath $statejson $statepkg
    if ($err -eq 1){
        return 1
    }

    $InstallerPath = Join-Path -Path $tmpParentPath -ChildPath $installerexe
    $env:ACTIVESTATE_SESSION_TOKEN = $script:SESSION_TOKEN_VALUE
    if ($script:TARGET) {
        & "$InstallerPath" "$script:TARGET" 2>&1 | Tee-Object -Variable output | Write-Host
    } else {
        & "$InstallerPath" 2>&1 | Tee-Object -Variable output | Write-Host
    }
    Remove-Item Env:ACTIVESTATE_SESSION_TOKEN -ErrorAction 'SilentlyContinue'

    $outputString = $output | Out-String
    $match = $outputString -match "Install Location: (?<content>[^\s]+)"
    if ($match) {
        $installDir = $Matches['content']
    } else {
        Write-Error "Unexpected installer output"
        return 1
    }

    # Write install file
    $StatePath = Join-Path -Path $installDir -ChildPath $script:STATEEXE
    $Command = "`"$StatePath`" export config --filter=dir"
    $ConfigDir = Invoke-Expression "& $Command" | Out-String
    $InstallFilePath = Join-Path -Path $ConfigDir.Trim() -ChildPath "installsource.txt"
    "install.ps1" | Out-File -Encoding ascii -FilePath $InstallFilePath

    $prepExitCode = runPreparationStep $installDir
    if ($prepExitCode -ne 0) {
        return $prepExitCode
    }

    # Check if installation is in $PATH
    if (isStateToolInstallationOnPath $installDir) {
        Write-Host "`nState Tool installation complete." -ForegroundColor Yellow
        Write-Host "You may now start using the '$script:STATEEXE' program."


        Write-Host '╔══════════════════════╗ ' -ForegroundColor DarkGreen
        Write-Host '║ ' -ForegroundColor DarkGreen -NoNewline
        Write-Host "State Tool Installed" -ForegroundColor White -NoNewline;
        Write-Host " ║" -ForegroundColor DarkGreen;
        Write-Host "╚══════════════════════╝" -ForegroundColor DarkGreen;

        warningIfAdmin
        return 0
    }

    warningIfAdmin
    Write-Host "State Tool successfully installed to: $installDir." -ForegroundColor Yellow

    # Ensure that the new install dir is on the PATH variable for follow-up commands
    $Env:PATH = "$installDir;$Env:PATH"
    return 0
}

$code = install | Select-Object -Last 1
if (($null -eq $code) -or ($code -eq 0)) {
    if ($script:POST_INSTALL_COMMAND) {
        # Extract executable from post install command string
        $executable, $arguments = $script:POST_INSTALL_COMMAND.Split(" ")
        & $executable $arguments
    }
    else {
        # Keep --activate and --activate-default flags for backwards compatibility
        activateIfRequested
    }
}
exit $code
