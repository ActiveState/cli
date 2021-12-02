# Copyright 2019-2021 ActiveState Software Inc. All rights reserved.
<#
.EXAMPLE
install.ps1 -b branchToInstall
#>

Set-StrictMode -Off

# URL to fetch installer archive from
$script:BASEFILEURL = "https://state-tool.s3.amazonaws.com/update/state"
# The name of the remove archive to download
$script:ARCHIVENAME = "state-installer.zip"
# Name of the installer executable to ultimately use
$script:INSTALLERNAME = "state-installer.exe"
# Channel the installer will target
$script:CHANNEL = "release"

$script:SESSION_TOKEN_VERIFY = -join ("{", "TOKEN", "}")
$script:SESSION_TOKEN = "{TOKEN}"
$script:SESSION_TOKEN_VALUE = ""

if ("$SESSION_TOKEN" -ne "$SESSION_TOKEN_VERIFY")
{
    $script:SESSION_TOKEN_VALUE = $script:SESSION_TOKEN
}

function parseChannel([string[]]$arr)
{
    for ($i = 0; $i -le $arr.Length; $i++)
    {
        $arg = $arr[$i]
        if ($arg -eq "-b" -And $arr.Length -ge ($i + 2))
        {
            return $arr[$i + 1]
        }
    }
    return $script:CHANNEL
}

$script:CHANNEL = parseChannel $args

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

progress "Preparing Installer for State Tool Package Manager"

$zipURL = "$script:BASEFILEURL/$script:CHANNEL/windows-amd64/$script:ARCHIVENAME"
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

$env:ACTIVESTATE_SESSION_TOKEN = $script:SESSION_TOKEN_VALUE
& $exePath @args --source-installer="install.ps1"
if (Test-Path env:ACTIVESTATE_SESSION_TOKEN)
{
    Remove-Item Env:\ACTIVESTATE_SESSION_TOKEN
}
