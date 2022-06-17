@echo off

{{range .denote}}
REM {{.}}
{{end}}

"{{.stateExec}}"
"{{.stateExec}}" "{{.stateSock}}" "{{.target}}" %*
