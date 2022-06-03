@echo off

{{range .denote}}
REM {{.}}
{{end}}

"{{.state-exec}}" "{{.state-sock}}" "{{.targetPath}}\{{.exe}}" %*
