# Copyright 2019-2022 ActiveState Software Inc. All rights reserved.
<#
.EXAMPLE
install.ps1 -b branchToInstall
#>

Set-StrictMode -Off

# URL to fetch update infos from.
$script:BASEINFOURL = "https://platform.activestate.com/sv/state-update/api/v1/info"
# URL to fetch installer archive from
$script:BASEFILEURL = "https://state-tool.s3.amazonaws.com/update/state"
# The name of the remove archive to download
$script:ARCHIVENAME = "state-installer.zip"
# Name of the installer executable to ultimately use
$script:INSTALLERNAME = "state-install\\state-installer.exe"
# Channel the installer will target
$script:CHANNEL = "release"
# The version to install (autodetermined to be the latest if left unspecified)
$script:VERSION = ""

$script:SESSION_TOKEN_VERIFY = -join ("{", "TOKEN", "}")
$script:SESSION_TOKEN = "{TOKEN}"
$script:SESSION_TOKEN_VALUE = ""

if ("$SESSION_TOKEN" -ne "$SESSION_TOKEN_VERIFY")
{
    $script:SESSION_TOKEN_VALUE = $script:SESSION_TOKEN
}

function getopt([string] $opt, [string] $default, [string[]] $arr)
{
    for ($i = 0; $i -le $arr.Length; $i++)
    {
        $arg = $arr[$i]
        if ($arg -eq $opt -and $arr.Length -ge ($i + 2))
        {
            return $arr[$i + 1]
        }
    }
    return $default
}

$script:CHANNEL = getopt "-b" $script:CHANNEL $args
$script:VERSION = getopt "-v" $script:VERSION $args

function download([string] $url, [string] $out)
{
    [int]$Retrycount = "0"

    do
    {
        try
        {
            $downloader = New-Object System.Net.WebClient
            if ($out -eq "")
            {
                return $downloader.DownloadString($url)
            }
            else
            {
                return $downloader.DownloadFile($url, $out)
            }
        }
        catch
        {
            if ($Retrycount -gt 5)
            {
                $exception = $_.Exception
                $errorMessage = ""

                if ($exception -is [System.Management.Automation.MethodInvocationException] -and $exception.InnerException -is [System.Net.WebException])
                {
                    $webException = [System.Net.WebException]$exception.InnerException
                    $response = $webException.Response

                    if ($response -ne $null)
                    {
                        $responseStream = $response.GetResponseStream()
                        $reader = New-Object System.IO.StreamReader($responseStream)
                        $responseBody = $reader.ReadToEnd()
                        $reader.Close()
                        $responseStream.Close()

                        try
                        {
                            $errorMessage = (ConvertFrom-Json $responseBody).message
                        }
                        catch
                        {
                            $errorMessage = $responseBody
                        }
                    }
                    else
                    {
                        $errorMessage = $webException.Message
                    }
                }
                else
                {
                    $errorMessage = $exception.Message
                }

                Write-Error "Could not download after 5 retries. Received error: $errorMessage"
                throw $exception
            }
            else
            {
                Write-Host "Could not download, retrying..."
                $Retrycount = $Retrycount + 1
            }
        }
    }
    While ($true)
}

function tempDir()
{
    $parent = [System.IO.Path]::GetTempPath()
    [string]$name = [System.Guid]::NewGuid()
    New-Item -ItemType Directory -Path (Join-Path $parent $name)
}

function progress([string] $msg)
{
    Write-Host "• $msg..." -NoNewline
}

function progress_done()
{
    $greenCheck = @{
        Object = [Char]8730
        ForegroundColor = 'Green'
        NoNewLine = $true
    }
    Write-Host @greenCheck
    Write-Host ' Done' -ForegroundColor Green
}

function progress_fail()
{
    Write-Host 'x Failed' -ForegroundColor Red
}

function error([string] $msg)
{
    Write-Host $msg -ForegroundColor Red
}

function setShellOverride
{
    # Walk up the process tree to find cmd.exe
    # If we encounter it we set the shell override
    $currentPid = $PID
    while ($currentPid -ne 0)
    {
        $process = Get-CimInstance Win32_Process | Where-Object { $_.ProcessId -eq $currentPid }
        if (!$process) { break }

        if ($process.Name -eq "cmd" -or $process.Name -eq "cmd.exe")
        {
            [System.Environment]::SetEnvironmentVariable("ACTIVESTATE_CLI_SHELL_OVERRIDE", $process.Name, "Process")
            break
        }

        $currentPid = $process.ParentProcessId
    }
}

$version = $script:VERSION
if (!$version) {
    # If the user did not specify a version, formulate a query to fetch the JSON info of the latest
    # version, including where it is.
    $jsonURL = "$script:BASEINFOURL/?channel=$script:CHANNEL&platform=windows&source=install"
} elseif (!($version | Select-String -Pattern "-SHA" -SimpleMatch)) {
    # If the user specified a partial version (i.e. no SHA), formulate a query to fetch the JSON
    # info of that version's latest SHA, including where it is.
    $versionNoSHA = $version
    $version = ""
    $jsonURL = "$script:BASEINFOURL/?channel=$script:CHANNEL&platform=windows&source=install&target-version=$versionNoSHA"
} else {
    # If the user specified a full version with SHA, formulate a query to fetch the JSON info of
    # that version.
    $versionNoSHA = $version -replace "-SHA.*", ""
    $jsonURL = "$script:BASEINFOURL/?channel=$script:CHANNEL&platform=windows&source=install&target-version=$versionNoSHA"
}

# Fetch version info.
try {
    $infoJson = ConvertFrom-Json -InputObject (download $jsonURL)
} catch [System.Exception] {
}
if (!$infoJson) {
    if (!$version) {
        Write-Error "Unable to retrieve the latest version number"
    } else {
        Write-Error "Could not download a State Tool Installer for the given command line arguments"
    }
    Write-Error $_.Exception.Message
    exit 1
}

# Extract checksum.
$checksum = $infoJson.Sha256

if (!$version) {
    # If the user specified no version or a partial version we need to use the json URL to get the
    # actual installer URL.
    $version = $infoJson.Version
    $relUrl = $infoJson.Path
} else {
    # If the user specified a full version, construct the installer URL.
    if ($version -ne $infoJson.Version) {
        Write-Error "Unknown version: $version"
        exit 1
    }
    $relUrl = "$script:CHANNEL/$versionNoSHA/windows-amd64/state-windows-amd64-$version.zip"
}

# Fetch the requested or latest version.
progress "Preparing Installer for State Tool Package Manager version $version"
$zipURL = "$script:BASEFILEURL/$relUrl"
$tmpParentPath = tempDir
$zipPath = Join-Path $tmpParentPath $script:ARCHIVENAME
$exePath = Join-Path $tmpParentPath $script:INSTALLERNAME
try
{
    download $zipURL $zipPath
}
catch [System.Exception]
{
    progress_fail
    Write-Error "Could not download $zipURL to $zipPath."
    Write-Error $_.Exception.Message
    exit 1
}

# Verify checksum.
$hash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash
if ($hash -ne $checksum)
{
    Write-Warning "SHA256 sum did not match:"
    Write-Warning "Expected: $checksum"
    Write-Warning "Received: $hash"
    Write-Warning "Aborting installation"
    exit 1
}

# Extract it.
try
{
    Expand-Archive -ErrorAction Stop -LiteralPath $zipPath -DestinationPath $tmpParentPath
}
catch
{
    progress_fail
    Write-Error $_.Exception.Message
    exit 1
}
progress_done

Write-Host ""

$OutputEncoding = [System.Console]::OutputEncoding = [System.Console]::InputEncoding = [System.Text.Encoding]::UTF8
$PSDefaultParameterValues['*:Encoding'] = 'utf8'
[System.Console]::OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Run the installer.
$env:ACTIVESTATE_SESSION_TOKEN = $script:SESSION_TOKEN_VALUE
setShellOverride
& $exePath $args --source-installer="install.ps1"
$success = $?
if (Test-Path env:ACTIVESTATE_SESSION_TOKEN)
{
    Remove-Item Env:\ACTIVESTATE_SESSION_TOKEN
}
if ( !$success ) {
    exit 1
}
