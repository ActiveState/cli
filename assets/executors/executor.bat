@echo off

{{range .denote}}
REM {{.}}
{{end}}

{{.state}} exec --path {{.projectPath}} -- {{.exe}} %*
