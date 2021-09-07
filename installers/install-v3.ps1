# Copyright 2019-2021 ActiveState Software Inc. All rights reserved.
<#
.EXAMPLE
install.ps1 -b branchToInstall
#>

# URL to fetch installer archive from
$script:BASEFILEURL = "https://state-tool.s3.amazonaws.com/update/state"
# The name of the remove archive to download
$script:ARCHIVENAME = "state-installer.zip"
# Name of the installer executable to ultimately use
$script:INSTALLERNAME="state-installer.exe"
# Channel the installer will target
$script:CHANNEL = "release"

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
            if ($Retrycount -gt 5) {
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

Write-Host "Preparing ActiveState Installer"

$zipURL = $script:BASEFILEURL/$script:ARCHIVENAME
$tmpParentPath = Join-Path $env:TEMP "ActiveState"
$zipPath = Join-Path $tmpParentPath $script:ARCHIVENAME
$exePath = Join-Path $tmpParentPath $script:INSTALLERNAME
try{
    download $zipURL $zipPath
} catch [System.Exception] {
    Write-Warning "Could not install State Tool"
    Write-Warning "Could not access $zipURL"
    Write-Error $_.Exception.Message
    return 1
}

# Extract binary from pkg and confirm checksum
Write-Host "Extracting $script:ARCHIVENAME...`n"
# using LiteralPath argument prevents interpretation of wildcards in zipPath
Expand-Archive -LiteralPath $zipPath -DestinationPath $tmpParentPath

& $exePath $args