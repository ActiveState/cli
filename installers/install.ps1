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
            $downloader = new-object System.Net.WebClient
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
                Write-Error "Could not Download after 5 retries."
                throw $_
            }
            else
            {
                Write-Host "Could not Download, retrying..."
                Write-Host $_
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

if (!$script:VERSION) {
  # Determine the latest version to fetch and parse info.
  $jsonURL = "$script:BASEINFOURL/?channel=$script:CHANNEL&platform=windows&source=install"
  $infoJson = ConvertFrom-Json -InputObject (download $jsonURL)
  $version = $infoJson.Version
  $checksum = $infoJson.Sha256
  $relUrl = $infoJson.Path
} else {
  $relUrl = "$script:CHANNEL/$script:VERSION/windows-amd64/state-windows-amd64-$script:VERSION.zip"
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
    return 1
}

# Verify checksum if possible.
$hash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash
if ($checksum -and $hash -ne $checksum)
{
    Write-Warning "SHA256 sum did not match:"
    Write-Warning "Expected: $checksum"
    Write-Warning "Received: $hash"
    Write-Warning "Aborting installation"
    return 1
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
    return 1
}
progress_done

Write-Host ""

$OutputEncoding = [System.Console]::OutputEncoding = [System.Console]::InputEncoding = [System.Text.Encoding]::UTF8
$PSDefaultParameterValues['*:Encoding'] = 'utf8'
[System.Console]::OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Run the installer.
$env:ACTIVESTATE_SESSION_TOKEN = $script:SESSION_TOKEN_VALUE
& $exePath $args --source-installer="install.ps1"
$success = $?
if (Test-Path env:ACTIVESTATE_SESSION_TOKEN)
{
    Remove-Item Env:\ACTIVESTATE_SESSION_TOKEN
}
if ( !$success ) {
  exit 1
}
