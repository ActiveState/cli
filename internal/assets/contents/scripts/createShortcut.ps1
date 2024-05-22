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
    Write-Host "Unauthorized Access: Please ensure you have the necessary permissions to create a shortcut at: $dir."
    exit 1
}
catch {
    Write-Host $_.Exception.Message
    exit 1
}
