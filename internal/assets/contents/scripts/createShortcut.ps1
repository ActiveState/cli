param (
    [string]$dir,
    [string]$name,
    [string]$target,
    [string]$shortcutArgs,
    [string]$iconLocation,
    [int]$windowStyle
)

$ShortcutPath = "$dir\$name.lnk"
$WScriptShell = New-Object -ComObject WScript.Shell
$Shortcut = $WScriptShell.CreateShortcut($ShortcutPath)
$Shortcut.TargetPath = $target
$Shortcut.Arguments = $shortcutArgs

if ($iconLocation -ne "") {
    $Shortcut.IconLocation = $iconLocation
}

if ($windowStyle -ne 0) {
    $Shortcut.WindowStyle = $windowStyle
}

try {
    $Shortcut.Save()
}
catch  [System.UnauthorizedAccessException] {
    Write-Host "Access denied."
    exit 1
}
catch {
    Write-Host $_.Exception.Message
    Write-Host $_.Exception.GetType().FullName
    exit 1
}
