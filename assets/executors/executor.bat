@echo off

{{range .denote}}
REM {{.}}
{{end}}

{{.state}} exec --path {{.targetPath}} -- {{.exe}} %*
