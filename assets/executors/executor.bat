@echo off

{{range .denote}}
REM {{.}}
{{end}}

{{.state}} shim --path {{.projectPath}} -- {{.exe}} %*
