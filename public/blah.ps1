
try{
    [System.IO.File]::ReadAllText( '\\test\no\filefound.log')
}
catch{
    Write-Warning("$PSItem") -WarningAction stop
}
write-host("did it keep going?")