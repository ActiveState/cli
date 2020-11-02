@echo off

REM {{.denote}}

{{.exe}} shim --path {{.projectPath}} -- {{.command}} %*
