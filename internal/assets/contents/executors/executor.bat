@echo off

{{range .denote}}
REM {{.}}
{{end}}

"{{.stateExec}}" "{{.stateSock}}"
"{{.stateExec}}" "{{.stateSock}}" "{{.target}}" %*
