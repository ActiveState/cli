# Copyright 2022-2022 ActiveState Software Inc. All rights reserved.

Set-StrictMode -Off

$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (!$currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator))
{
    Write-Error "Please run this script as administrator. This is required to ensure any existing State Tool processes are terminated."
    Exit 1
}

$continue = Read-Host "You are about to kill all State Tool processes, continue? (y/n)"
if ($continue -ne 'y')
{
    Write-Host "Cancelled"
    Exit 1
}

Write-Host "Stopping running State Tool processes"

$pss = Get-Process | Where {$_.Name -eq "state-svc" -or $_.Name -eq "state" -or $_.Name -eq "state-installer"} 

foreach ($ps in $pss) {
    if ($ps.Path.ToLower().Contains('activestate') -or $ps.Name -eq "state-installer") {
        Write-Host "Stopping $($ps.Name) at $($ps.Path)"
        Stop-Process -Id $ps.Id
    }
}

Write-Host "Done"
