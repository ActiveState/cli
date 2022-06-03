@echo off

{{range .denote}}
REM {{.}}
{{end}}

"{{.stateExec}}" "{{.stateSock}}" "{{.targetPath}}\{{.exe}}" %*
