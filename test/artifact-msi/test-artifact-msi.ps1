$logDir = "Z:\bin"
$installAction = "install"
$uninstallAction = "uninstall"

class MSI {
    [string]$Path
    [string]$Version
}

[MSI[]]$perlMSIs = @(
    [MSI]@{
        Path="Z:\bin\ActivePerl-5.26.msi"
        Version="5.26"
    },
    [MSI]@{
        Path="Z:\bin\ActivePerl-5.28.msi"
        Version="5.28"
    }
)

function ExecMsiexec([string] $path, [string] $action, [string] $logFile) {
    $actionArg = "/package"
    if ($action -eq $uninstallAction) {
        $actionArg = "/uninstall"
    }

    $argList = "{0} {1} /quiet /qn /norestart /log {2}" -f $actionArg, $path, $logFile
    $proc = Start-Process msiexec.exe -Wait -ArgumentList "$($argList)" -PassThru
    $handle = $proc.Handle # cache proc.Handle (necessary for exit code to return properly)
    $proc.WaitForExit();
    return $proc.ExitCode
}

function LogFileName([string] $path, [string] $action) {
    return "$($logDir)\$([io.path]::GetFileNameWithoutExtension($path))_$($action).log"
}

function CommandExists([string] $command) {
    try {
        if(Get-Command $command -ErrorAction Stop){
            return $true
        }
    }
    catch {
        return $false
    }
}

$perlMSIs | ForEach-Object -Process {
    $installLogFile = LogFileName $_.Path $installAction
    $installExitCode = ExecMsiexec $_.Path $installAction $installLogFile
    if ($installExitCode -ne 0) {
        Write-Error "failed to install correctly"
        exit $installExitCode
    }
    # should the log file be dumped? or can it be inspected after-the-fact?

    $companyMatches=(perl -v | Select-String -Pattern "ActiveState").length
    $versionMatches=(perl -v | Select-String -Pattern $_.Version).length
    if (($companyMatches -eq 0) -or ($versionMatches -eq 0)) { 
        Write-Error "perl $($_.Version) does not appear to be provided by ActiveState from $($_.Path)"
        exit 1 
    }

    $uninstallLogFile = LogFileName $_.Path $uninstallAction
    $uninstallExitCode = ExecMsiexec $_.Path $uninstallAction $uninstallLogFile
    if ($uninstallExitCode -ne 0) {
        Write-Error "failed to uninstall correctly"
        exit $uninstallExitCode
    }

    if (CommandExists "perl") {
        $companyMatches=(perl -v | Select-String -Pattern "ActiveState").length
        if ($companyMatches -gt 0) { 
            Write-Error "ActiveState perl is still detected"
            exit 1 
        }
    }
}
